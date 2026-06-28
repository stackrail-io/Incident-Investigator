package runtime

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/stackrail/incident-investigator/internal/engine"
	"github.com/stackrail/incident-investigator/internal/engine/protocol"
	"github.com/stackrail/incident-investigator/internal/intelligence"
	"github.com/stackrail/incident-investigator/internal/model"
	"github.com/stackrail/incident-investigator/internal/reasoners"
)

// Engines bundles the pluggable reasoning components. Every field is an
// interface so an alternative (e.g. LLM-backed) implementation can be swapped in
// without changing the runtime.
type Engines struct {
	Planner       engine.Planner
	Hypothesis    engine.HypothesisEngine
	Confidence    engine.ConfidenceScorer
	Contradiction engine.ContradictionDetector
	Missing       engine.MissingEvidenceDetector
	Graph         engine.GraphBuilder
	Timeline      engine.TimelineBuilder
	Blast         engine.BlastRadiusEstimator
	Report        engine.ReportGenerator
	Coverage      engine.CoverageEngine
	Strategy      engine.StrategyEngine
	Sufficiency   engine.SufficiencyEngine
	Importance    engine.ImportanceEngine
}

// DefaultEngines returns the built-in heuristic implementations.
func DefaultEngines() Engines {
	return Engines{
		Planner:       engine.NewHeuristicPlanner(),
		Hypothesis:    engine.NewHeuristicHypothesisEngine(),
		Confidence:    engine.NewHeuristicConfidenceScorer(),
		Contradiction: engine.NewHeuristicContradictionDetector(),
		Missing:       engine.NewHeuristicMissingEvidenceDetector(),
		Graph:         engine.NewHeuristicGraphBuilder(),
		Timeline:      engine.NewHeuristicTimelineBuilder(),
		Blast:         engine.NewHeuristicBlastRadiusEstimator(),
		Report:        engine.NewHeuristicReportGenerator(),
		Coverage:      engine.NewHeuristicCoverageEngine(),
		Strategy:      engine.NewHeuristicStrategyEngine(),
		Sufficiency:   engine.NewHeuristicSufficiencyEngine(),
		Importance:    engine.NewHeuristicImportanceEngine(),
	}
}

// Thresholds control when an investigation is considered ready to conclude.
type Thresholds struct {
	Confidence float64
	Progress   float64
}

// DefaultThresholds are conservative defaults.
func DefaultThresholds() Thresholds {
	return Thresholds{Confidence: 70, Progress: 70}
}

// Runtime is the stateful investigation engine. It owns the session store and
// orchestrates the engines, recomputing derived state incrementally on each
// mutation.
type Runtime struct {
	store        Store
	engines      Engines
	orchestrator *engine.Orchestrator
	stateMachine *engine.StateMachine
	intelligence intelligence.Intelligence
	thresholds   Thresholds
	now          func() time.Time
}

// Option customizes a Runtime.
type Option func(*Runtime)

// WithEngines overrides the reasoning engines.
func WithEngines(e Engines) Option { return func(r *Runtime) { r.engines = e } }

// WithThresholds overrides the conclusion thresholds.
func WithThresholds(t Thresholds) Option { return func(r *Runtime) { r.thresholds = t } }

// WithClock overrides the time source (useful in tests).
func WithClock(now func() time.Time) Option { return func(r *Runtime) { r.now = now } }

// WithOrchestrator overrides the reasoning orchestrator.
func WithOrchestrator(o *engine.Orchestrator) Option {
	return func(r *Runtime) { r.orchestrator = o }
}

// New constructs a Runtime with sensible defaults.
func New(opts ...Option) *Runtime {
	r := &Runtime{
		store:        NewMemoryStore(),
		engines:      DefaultEngines(),
		stateMachine: engine.NewStateMachine(),
		intelligence: intelligence.NewMemoryService(),
		thresholds:   DefaultThresholds(),
		now:          time.Now,
	}
	for _, opt := range opts {
		opt(r)
	}
	r.orchestrator = engine.NewOrchestratorWithRegistry(reasoners.DefaultRegistry(r.reasoningEngines()))
	return r
}

// StartInput is the request to begin an investigation.
type StartInput struct {
	Question   string
	Service    string
	TimeWindow model.TimeWindow
	Goal       model.InvestigationGoal
}

// ExplainInvestigationOutput is the full protocol snapshot for debugging.
type ExplainInvestigationOutput struct {
	SessionID         string                      `json:"session_id"`
	Plan              *model.InvestigationPlan    `json:"plan"`
	QuestionGraph     model.QuestionGraph         `json:"question_graph"`
	OpenQuestions     []model.Question            `json:"open_questions"`
	ResolvedQuestions []model.QuestionResolution  `json:"resolved_questions"`
	EvidenceRequests  []model.ProtocolEvidenceRequest `json:"evidence_requests"`
	CurrentStage      model.InvestigationStage    `json:"current_stage"`
	ReasoningTrace    []model.ReasoningTrace      `json:"reasoning_trace"`
	Confidence        float64                     `json:"confidence"`
	Hypotheses        []model.Hypothesis          `json:"hypotheses"`
	Metrics           model.ProtocolMetrics       `json:"metrics"`
}

// ResolveQuestionInput is the input to resolve_question.
type ResolveQuestionInput struct {
	SessionID  string
	QuestionID string
	Confirmed  bool
	Reason     string
}
// ExplainOutput is the full reasoning snapshot for debugging.
type ExplainOutput struct {
	SessionID           string                      `json:"session_id"`
	State               model.InvestigationState    `json:"state"`
	Goal                model.InvestigationGoal     `json:"goal"`
	Plan                *model.InvestigationPlan    `json:"plan,omitempty"`
	Hypotheses          []model.Hypothesis          `json:"hypotheses"`
	ReasoningTrace      []model.ReasoningTrace      `json:"reasoning_trace"`
	Contradictions      []model.Contradiction       `json:"contradictions"`
	Coverage            model.CoverageReport        `json:"coverage"`
	Confidence            float64                   `json:"confidence"`
	ConfidenceBreakdown model.ConfidenceBreakdown   `json:"confidence_breakdown"`
	BlockingQuestions   []model.BlockingQuestion    `json:"blocking_questions"`
	MissingEvidence     []model.EvidenceRequirement `json:"missing_evidence"`
	Sufficiency         model.SufficiencyReport     `json:"sufficiency"`
	Strategy            []model.NextStep            `json:"strategy"`
	Journal             []model.JournalEntry        `json:"journal"`
	Metrics             model.ReasoningMetrics      `json:"metrics"`
}

// Start creates a new investigation session and runs the planner once.
func (r *Runtime) Start(in StartInput) (*model.Session, error) {
	if strings.TrimSpace(in.Question) == "" {
		return nil, fmt.Errorf("question is required")
	}
	if err := validateTimeWindow(in.TimeWindow); err != nil {
		return nil, err
	}

	goal := in.Goal
	if goal == "" {
		goal = model.DefaultGoal
	}
	if !goal.Valid() {
		return nil, fmt.Errorf("unknown investigation goal %q", goal)
	}

	now := r.now()
	s := &model.Session{
		ID:         newID("inv"),
		Question:   in.Question,
		Service:    in.Service,
		TimeWindow: in.TimeWindow,
		Goal:       goal,
		Evidence:   []*model.Evidence{},
		Graph:      model.NewEmptyGraphView(),
		Hypotheses: []model.Hypothesis{},
		Timeline:   model.Timeline{},
		Status:     model.StatusCollecting,
		State:      model.StateStarted,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	s.AddHistory("started", in.Question, now)
	s.AddJournal("investigation_started", in.Question, 0, now)

	r.recompute(s)

	if err := r.store.Create(s); err != nil {
		return nil, err
	}
	return s, nil
}

// Submit ingests new evidence into a session and recomputes derived state.
func (r *Runtime) Submit(sessionID string, evidence []*model.Evidence) (*model.Session, error) {
	if len(evidence) == 0 {
		return nil, ErrEmptyEvidence
	}

	now := r.now()
	return r.store.WithSession(sessionID, func(s *model.Session) error {
		if s.Status == model.StatusCompleted {
			return ErrSessionCompleted
		}

		seen := existingEvidenceIDs(s)
		for _, e := range evidence {
			if e == nil {
				return ErrInvalidEvidence
			}
			r.normalizeEvidence(e, now)
			if strings.TrimSpace(e.Summary) == "" {
				return ErrInvalidEvidence
			}
			if seen[e.ID] {
				return fmt.Errorf("%w: %s", ErrDuplicateEvidenceID, e.ID)
			}
			seen[e.ID] = true
			s.Evidence = append(s.Evidence, e)
		}

		s.AddHistory("evidence_submitted", pluralizeCount(len(evidence), "evidence item"), now)
		s.AddJournal("evidence_submitted", pluralizeCount(len(evidence), "item"), s.Confidence, now)
		r.recompute(s)
		s.UpdatedAt = now
		return nil
	})
}

// Get returns the current session state, recomputing derived fields first.
func (r *Runtime) Get(sessionID string) (*model.Session, error) {
	return r.store.WithSession(sessionID, func(s *model.Session) error {
		r.recompute(s)
		return nil
	})
}

// Explain returns the full reasoning snapshot for an investigation.
func (r *Runtime) Explain(sessionID string) (*ExplainOutput, error) {
	sess, err := r.Get(sessionID)
	if err != nil {
		return nil, err
	}
	return &ExplainOutput{
		SessionID:           sess.ID,
		State:               sess.State,
		Goal:                sess.Goal,
		Plan:                sess.Plan,
		Hypotheses:          sess.Hypotheses,
		ReasoningTrace:      sess.ReasoningTrace,
		Contradictions:      sess.Contradictions,
		Coverage:            sess.Coverage,
		Confidence:          sess.Confidence,
		ConfidenceBreakdown: sess.ConfidenceBreakdown,
		BlockingQuestions:   sess.Sufficiency.BlockingQuestions,
		MissingEvidence:     sess.Sufficiency.MissingEvidence,
		Sufficiency:         sess.Sufficiency,
		Strategy:            sess.Strategy,
		Journal:             sess.Journal,
		Metrics:             sess.Metrics,
	}, nil
}

// ExplainInvestigation returns the protocol-centric investigation snapshot.
func (r *Runtime) ExplainInvestigation(sessionID string) (*ExplainInvestigationOutput, error) {
	sess, err := r.Get(sessionID)
	if err != nil {
		return nil, err
	}
	stage := model.StagePlanning
	if sess.Plan != nil {
		stage = sess.Plan.CurrentStage
	}
	return &ExplainInvestigationOutput{
		SessionID:         sess.ID,
		Plan:              sess.Plan,
		QuestionGraph:     sess.QuestionGraph,
		OpenQuestions:     protocol.ListOpenQuestions(sess.Plan),
		ResolvedQuestions: resolutionHistory(sess.Plan),
		EvidenceRequests:  evidenceRequests(sess.Plan),
		CurrentStage:      stage,
		ReasoningTrace:    sess.ReasoningTrace,
		Confidence:        sess.Confidence,
		Hypotheses:        sess.Hypotheses,
		Metrics:           sess.ProtocolMetrics,
	}, nil
}

// GetPlan returns the investigation plan for a session.
func (r *Runtime) GetPlan(sessionID string) (*model.InvestigationPlan, error) {
	sess, err := r.Get(sessionID)
	if err != nil {
		return nil, err
	}
	return sess.Plan, nil
}

// ListOpenQuestions returns unresolved questions sorted by priority.
func (r *Runtime) ListOpenQuestions(sessionID string) ([]model.Question, error) {
	sess, err := r.Get(sessionID)
	if err != nil {
		return nil, err
	}
	return protocol.ListOpenQuestions(sess.Plan), nil
}

// ResolveQuestion explicitly resolves a protocol question.
func (r *Runtime) ResolveQuestion(in ResolveQuestionInput) (*model.QuestionResolution, *model.Session, error) {
	var res *model.QuestionResolution
	sess, err := r.store.WithSession(in.SessionID, func(s *model.Session) error {
		if s.Status == model.StatusCompleted {
			return ErrSessionCompleted
		}
		pe, err := protocol.NewEngine(s.Goal)
		if err != nil {
			return err
		}
		if s.Plan == nil {
			r.recompute(s)
		}
		now := r.now()
		res, err = pe.ResolveQuestion(s, in.QuestionID, in.Confirmed, in.Reason, now)
		if err != nil {
			return err
		}
		sig := engine.Analyze(s)
		s.Confidence = r.engines.Confidence.Score(s, sig, s.Hypotheses, s.Contradictions, s.Progress)
		s.AddJournal("question_resolved", in.QuestionID, s.Confidence, now)
		s.UpdatedAt = now
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return res, sess, nil
}

func resolutionHistory(plan *model.InvestigationPlan) []model.QuestionResolution {
	if plan == nil {
		return nil
	}
	return plan.ResolutionHistory
}

func evidenceRequests(plan *model.InvestigationPlan) []model.ProtocolEvidenceRequest {
	if plan == nil {
		return nil
	}
	return plan.EvidenceRequests
}

// Finish generates the final report and marks the session completed.
func (r *Runtime) Finish(sessionID string) (model.Report, *model.Session, error) {
	var report model.Report
	sess, err := r.store.WithSession(sessionID, func(s *model.Session) error {
		r.recompute(s)
		sig := engine.Analyze(s)
		report = r.engines.Report.Generate(s, sig)

		if s.Status != model.StatusCompleted {
			now := r.now()
			s.Status = model.StatusCompleted
			s.State = model.StateCompleted
			s.UpdatedAt = now
			if cal, err := r.intelligence.CalibrateConfidence(context.Background(), calibrationRequestFromSession(s, true)); err == nil && cal != nil {
				s.Confidence = cal.CalibratedConfidence
				s.AddJournal("intelligence_archived", cal.Reason, s.Confidence, now)
			}
			s.AddHistory("finished", "final report generated", now)
			s.AddJournal("investigation_completed", "final report generated", s.Confidence, now)
		}
		return nil
	})
	if err != nil {
		return model.Report{}, nil, err
	}
	return report, sess, nil
}

func validateTimeWindow(w model.TimeWindow) error {
	if w.Start.IsZero() || w.End.IsZero() {
		return nil
	}
	if w.End.Before(w.Start) {
		return ErrInvalidTimeWindow
	}
	return nil
}

func existingEvidenceIDs(s *model.Session) map[string]bool {
	seen := make(map[string]bool, len(s.Evidence))
	for _, e := range s.Evidence {
		if e != nil {
			seen[e.ID] = true
		}
	}
	return seen
}

// normalizeEvidence fills in defaults the client may have omitted.
func (r *Runtime) normalizeEvidence(e *model.Evidence, now time.Time) {
	if e.ID == "" {
		e.ID = newID("ev")
	}
	if e.Timestamp.IsZero() {
		e.Timestamp = now
	}
	if e.Source == "" {
		e.Source = "provided_by_client"
	}
	if e.Category == "" || !e.Category.Valid() {
		e.Category = model.CategoryCustom
	}
}

// recompute rebuilds all derived state from the current evidence through the
// reasoning orchestrator and state machine.
func (r *Runtime) recompute(s *model.Session) {
	now := r.now()
	prevMetrics := s.Metrics
	prevConfidence := s.Confidence

	s.SnapshotHypotheses()
	if s.State != model.StateCompleted && s.State != model.StateFailed {
		s.State = r.stateMachine.BeginReasoning(s.State)
	}

	sig := engine.Analyze(s)
	ctx := &engine.ReasoningContext{
		Session: s,
		Signals: sig,
		Engines: r.reasoningEngines(),
	}
	if cycle, err := r.orchestrator.Run(context.Background(), ctx); err == nil && cycle != nil {
		s.ReasoningCycles = append(s.ReasoningCycles, *cycle)
		s.AddJournal("reasoning_cycle", fmt.Sprintf(
			"cycle %s: %d reasoners, %d applied, %d rejected",
			cycle.CycleID, len(cycle.Reasoners), len(cycle.AppliedActions), len(cycle.RejectedActions),
		), s.Confidence, now)
	}

	// Investigation protocol: questions → evidence requests → resolution → hypotheses.
	if pe, err := protocol.NewEngine(s.Goal); err == nil {
		turn := pe.Run(s, sig, now, prevConfidence)
		s.Confidence = r.engines.Confidence.Score(s, sig, s.Hypotheses, s.Contradictions, s.Progress)
		turn.ConfidenceDelta = roundConfidenceDelta(s.Confidence - prevConfidence)
		s.LastTurn = turn
		s.Strategy = protocolRequestsToStrategy(s.Plan)
		s.Sufficiency = r.engines.Sufficiency.Evaluate(s, sig, s.Coverage)
		s.InvestigationProgress = engine.ComputeInvestigationProgress(s, s.Coverage, s.Sufficiency)
	}

	r.applyIntelligenceCalibration(context.Background(), s)

	engine.UpdateReasoningTrace(s, now)
	engine.ApplyHypothesisTraces(s)
	s.Metrics = engine.ComputeMetrics(s, s.Coverage, prevMetrics)

	if s.State != model.StateCompleted {
		s.State = r.stateMachine.Transition(s.State, s, s.Sufficiency)
	}
	s.Status = r.deriveStatus(s)
}

func protocolRequestsToStrategy(plan *model.InvestigationPlan) []model.NextStep {
	if plan == nil {
		return nil
	}
	steps := make([]model.NextStep, 0, len(plan.EvidenceRequests))
	for i, req := range plan.EvidenceRequests {
		if req.Status == model.RequestFulfilled {
			continue
		}
		cat := model.CategoryApplicationLogs
		if len(req.Categories) > 0 {
			cat = req.Categories[0]
		}
		steps = append(steps, model.NextStep{
			Priority:               i + 1,
			Category:               cat,
			Reason:                 req.Reason,
			ExpectedConfidenceGain: req.ExpectedConfidenceGain,
		})
		if len(steps) >= 2 {
			break
		}
	}
	return steps
}

func (r *Runtime) reasoningEngines() engine.RuntimeEngines {
	return engine.RuntimeEngines{
		Planner:       r.engines.Planner,
		Hypothesis:    r.engines.Hypothesis,
		Confidence:    r.engines.Confidence,
		Contradiction: r.engines.Contradiction,
		Missing:       r.engines.Missing,
		Graph:         r.engines.Graph,
		Timeline:      r.engines.Timeline,
		Blast:         r.engines.Blast,
		Coverage:      r.engines.Coverage,
		Strategy:      r.engines.Strategy,
		Sufficiency:   r.engines.Sufficiency,
		Importance:    r.engines.Importance,
		StateMachine:  r.stateMachine,
	}
}

func (r *Runtime) deriveStatus(s *model.Session) model.Status {
	if s.Status == model.StatusCompleted || s.State == model.StateCompleted {
		return model.StatusCompleted
	}
	if s.Sufficiency.CanAnswer ||
		(s.Confidence >= r.thresholds.Confidence && s.InvestigationProgress.PercentComplete >= r.thresholds.Progress) {
		return model.StatusReady
	}
	return s.State.LegacyStatus()
}

func pluralizeCount(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return strconv.Itoa(n) + " " + noun + "s"
}

func roundConfidenceDelta(v float64) float64 {
	return float64(int(v*10+0.5)) / 10
}
