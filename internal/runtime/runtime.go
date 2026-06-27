package runtime

import (
	"strconv"
	"time"

	"github.com/stackrail/incident-investigator/internal/engine"
	"github.com/stackrail/incident-investigator/internal/model"
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
	store      Store
	engines    Engines
	thresholds Thresholds
	now        func() time.Time
}

// Option customizes a Runtime.
type Option func(*Runtime)

// WithEngines overrides the reasoning engines.
func WithEngines(e Engines) Option { return func(r *Runtime) { r.engines = e } }

// WithThresholds overrides the conclusion thresholds.
func WithThresholds(t Thresholds) Option { return func(r *Runtime) { r.thresholds = t } }

// WithClock overrides the time source (useful in tests).
func WithClock(now func() time.Time) Option { return func(r *Runtime) { r.now = now } }

// New constructs a Runtime with sensible defaults.
func New(opts ...Option) *Runtime {
	r := &Runtime{
		store:      NewMemoryStore(),
		engines:    DefaultEngines(),
		thresholds: DefaultThresholds(),
		now:        time.Now,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// StartInput is the request to begin an investigation.
type StartInput struct {
	Question   string
	Service    string
	TimeWindow model.TimeWindow
}

// Start creates a new investigation session and runs the planner once.
func (r *Runtime) Start(in StartInput) (*model.Session, error) {
	now := r.now()
	s := &model.Session{
		ID:         newID("inv"),
		Question:   in.Question,
		Service:    in.Service,
		TimeWindow: in.TimeWindow,
		Evidence:   []*model.Evidence{},
		Graph:      model.NewGraph(),
		Hypotheses: []model.Hypothesis{},
		Timeline:   model.Timeline{},
		Status:     model.StatusCollecting,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	s.AddHistory("started", in.Question, now)

	r.recompute(s)

	if err := r.store.Create(s); err != nil {
		return nil, err
	}
	return s, nil
}

// Submit ingests new evidence into a session and recomputes derived state.
func (r *Runtime) Submit(sessionID string, evidence []*model.Evidence) (*model.Session, error) {
	s, err := r.store.Get(sessionID)
	if err != nil {
		return nil, err
	}

	now := r.now()
	for _, e := range evidence {
		r.normalizeEvidence(e, now)
		s.Evidence = append(s.Evidence, e)
	}
	s.AddHistory("evidence_submitted", pluralizeCount(len(evidence), "evidence item"), now)

	r.recompute(s)
	s.UpdatedAt = now

	if err := r.store.Save(s); err != nil {
		return nil, err
	}
	return s, nil
}

// Get returns the current session state.
func (r *Runtime) Get(sessionID string) (*model.Session, error) {
	return r.store.Get(sessionID)
}

// Finish generates the final report and marks the session completed.
func (r *Runtime) Finish(sessionID string) (model.Report, *model.Session, error) {
	s, err := r.store.Get(sessionID)
	if err != nil {
		return model.Report{}, nil, err
	}

	now := r.now()
	r.recompute(s)
	sig := engine.Analyze(s)
	report := r.engines.Report.Generate(s, sig)

	s.Status = model.StatusCompleted
	s.UpdatedAt = now
	s.AddHistory("finished", "final report generated", now)

	if err := r.store.Save(s); err != nil {
		return model.Report{}, nil, err
	}
	return report, s, nil
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

// recompute rebuilds all derived state from the current evidence. It is the
// single, ordered pipeline the whole engine flows through.
func (r *Runtime) recompute(s *model.Session) {
	sig := engine.Analyze(s)

	// Contradictions first: hypotheses and confidence both consume them.
	s.Contradictions = r.engines.Contradiction.Detect(s, sig)

	// Hypotheses must be set on the session before the graph is built, since the
	// graph links evidence to hypothesis nodes.
	s.Hypotheses = r.engines.Hypothesis.Generate(s, sig, s.Contradictions)

	required := r.engines.Planner.Plan(s, sig)
	s.RequiredEvidence = required
	s.MissingEvidence = r.engines.Missing.Detect(s, required)
	s.Progress = engine.Progress(s, required)
	s.Confidence = r.engines.Confidence.Score(s, sig, s.Hypotheses, s.Contradictions, s.Progress)

	s.Timeline = r.engines.Timeline.Build(s)
	s.Graph = r.engines.Graph.Build(s, sig)
	s.BlastRadius = r.engines.Blast.Estimate(s, sig)

	s.Status = r.deriveStatus(s)
}

func (r *Runtime) deriveStatus(s *model.Session) model.Status {
	if s.Status == model.StatusCompleted {
		return model.StatusCompleted
	}
	if s.Confidence >= r.thresholds.Confidence && s.Progress >= r.thresholds.Progress {
		return model.StatusReady
	}
	return model.StatusCollecting
}

func pluralizeCount(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return strconv.Itoa(n) + " " + noun + "s"
}
