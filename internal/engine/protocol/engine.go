package protocol

import (
	"fmt"
	"sort"
	"time"

	"github.com/stackrail/incident-investigator/internal/engine"
	"github.com/stackrail/incident-investigator/internal/engine/playbook"
	"github.com/stackrail/incident-investigator/internal/model"
)

// Engine executes the investigation protocol: questions → evidence requests →
// resolution → hypothesis effects.
type Engine struct {
	playbook *playbook.Playbook
}

// NewEngine loads the default playbook for the session goal.
func NewEngine(goal model.InvestigationGoal) (*Engine, error) {
	pb, err := playbook.ForGoal(goal)
	if err != nil {
		return nil, err
	}
	return &Engine{playbook: pb}, nil
}

// Run advances the investigation protocol for one reasoning iteration.
func (e *Engine) Run(s *model.Session, sig engine.Signals, now time.Time, prevConfidence float64) model.ProtocolTurn {
	turn := model.ProtocolTurn{}
	if s.Plan == nil {
		s.Plan = e.initPlan(s)
		s.Plan.CurrentStage = model.StagePlanning
	}

	e.ensureQuestions(s, sig)
	e.generateDynamicQuestions(s, sig, &turn)
	e.syncEvidenceRequests(s)
	e.resolveQuestions(s, sig, now, &turn)
	e.ensureQuestions(s, sig)
	e.generateDynamicQuestions(s, sig, &turn)
	e.syncEvidenceRequests(s)
	e.applyPlaybookEffects(s, &turn)
	e.updatePlanLists(s)
	s.QuestionGraph = buildQuestionGraph(s.Plan)
	s.ProtocolMetrics = computeProtocolMetrics(s, prevConfidence)
	s.LastTurn = turn
	s.Plan.Confidence = s.Confidence
	s.Plan.CurrentStage = deriveStage(s)
	s.Plan.ActiveHypotheses = activeHypothesisIDs(s.Hypotheses)
	return turn
}

func (e *Engine) initPlan(s *model.Session) *model.InvestigationPlan {
	return &model.InvestigationPlan{
		ID:           fmt.Sprintf("plan-%s", s.ID),
		Goal:         s.Goal,
		CurrentStage: model.StagePlanning,
		PlaybookID:   e.playbook.ID,
	}
}

func (e *Engine) ensureQuestions(s *model.Session, sig engine.Signals) {
	existing := map[string]bool{}
	for _, q := range s.Plan.Questions {
		existing[q.ID] = true
	}

	for _, pq := range e.playbook.Questions {
		if pq.TriggerSignal != "" {
			continue // dynamic questions added separately
		}
		if !dependenciesMet(s.Plan, pq.DependsOn) {
			continue
		}
		if existing[pq.ID] {
			continue
		}
		s.Plan.Questions = append(s.Plan.Questions, playbookQuestionToModel(pq))
		existing[pq.ID] = true
	}
	s.Plan.CurrentStage = model.StageQuestionGeneration
}

func (e *Engine) generateDynamicQuestions(s *model.Session, sig engine.Signals, turn *model.ProtocolTurn) {
	existing := map[string]bool{}
	for _, q := range s.Plan.Questions {
		existing[q.ID] = true
	}

	for _, pq := range e.playbook.Questions {
		if pq.TriggerSignal == "" {
			continue
		}
		if existing[pq.ID] {
			continue
		}
		if !signalTriggered(sig, pq.TriggerSignal) {
			continue
		}
		if !dependenciesMet(s.Plan, pq.DependsOn) {
			continue
		}
		q := playbookQuestionToModel(pq)
		q.Status = model.QuestionWaitingForEvidence
		s.Plan.Questions = append(s.Plan.Questions, q)
		turn.NewQuestions = append(turn.NewQuestions, q)
		existing[pq.ID] = true
	}

	// Unlock GENERATES children when parent is confirmed.
	for _, q := range s.Plan.Questions {
		if q.Resolution == nil || q.Resolution.Status != model.ResolutionConfirmed {
			continue
		}
		for _, pq := range e.playbook.Questions {
			for _, gen := range parentGenerates(e.playbook, q.ID) {
				if pq.ID != gen || existing[pq.ID] {
					continue
				}
				child := playbookQuestionToModel(pq)
				child.Status = model.QuestionWaitingForEvidence
				s.Plan.Questions = append(s.Plan.Questions, child)
				turn.NewQuestions = append(turn.NewQuestions, child)
				existing[pq.ID] = true
			}
		}
	}
}

func parentGenerates(pb *playbook.Playbook, parentID string) []string {
	for _, pq := range pb.Questions {
		if pq.ID == parentID {
			return pq.Generates
		}
	}
	return nil
}

func (e *Engine) syncEvidenceRequests(s *model.Session) {
	open := openQuestions(s.Plan)
	requests := make([]model.ProtocolEvidenceRequest, 0)
	reqIndex := map[string]int{}

	for _, q := range open {
		if q.Status == model.QuestionAnswered || q.Status == model.QuestionRejected {
			continue
		}
		missing := missingCategories(s, q.RequiredEvidence)
		if len(missing) == 0 {
			continue
		}
		key := q.ID + ":" + string(missing[0])
		if _, ok := reqIndex[key]; ok {
			continue
		}
		gain := expectedGain(q, missing)
		req := model.ProtocolEvidenceRequest{
			ID:                     fmt.Sprintf("req-%s-%s", q.ID, missing[0]),
			QuestionID:             q.ID,
			Categories:             missing,
			Priority:               priorityForQuestion(q),
			Reason:                 fmt.Sprintf("Need %s to answer: %s", joinCategories(missing), q.Title),
			ExpectedConfidenceGain: gain,
			Status:                 model.RequestOpen,
		}
		requests = append(requests, req)
		reqIndex[key] = len(requests) - 1
	}

	sort.SliceStable(requests, func(i, j int) bool {
		return requests[i].ExpectedConfidenceGain > requests[j].ExpectedConfidenceGain
	})
	if len(requests) > 2 {
		requests = requests[:2]
	}

	// Mark fulfilled requests.
	for i := range s.Plan.EvidenceRequests {
		if s.Plan.EvidenceRequests[i].Status == model.RequestFulfilled {
			continue
		}
		if categoriesPresent(s, s.Plan.EvidenceRequests[i].Categories) {
			s.Plan.EvidenceRequests[i].Status = model.RequestFulfilled
		}
	}

	s.Plan.EvidenceRequests = requests
	s.Plan.CurrentStage = model.StageEvidenceCollection

	// Backward-compatible planner output from protocol requests.
	s.RequiredEvidence = protocolToLegacyRequests(requests, open)
	s.MissingEvidence = missingLegacyRequests(s, s.RequiredEvidence)
}

func (e *Engine) resolveQuestions(s *model.Session, sig engine.Signals, now time.Time, turn *model.ProtocolTurn) {
	resolver := NewResolutionEngine()
	for i := range s.Plan.Questions {
		q := &s.Plan.Questions[i]
		if q.Status == model.QuestionAnswered || q.Status == model.QuestionRejected {
			continue
		}
		res := resolver.Resolve(q, s, sig)
		if res == nil {
			updatePartialStatus(q, s)
			continue
		}
		res.ResolvedAt = now
		q.Resolution = res
		q.Confidence = res.Confidence
		switch res.Status {
		case model.ResolutionConfirmed:
			q.Status = model.QuestionAnswered
		case model.ResolutionRejected:
			q.Status = model.QuestionRejected
		default:
			q.Status = model.QuestionPartiallyAnswered
		}
		q.SupportingEvidence = append([]string(nil), res.SupportingEvidence...)
		q.ContradictingEvidence = append([]string(nil), res.ContradictingEvidence...)
		s.Plan.ResolutionHistory = append(s.Plan.ResolutionHistory, *res)
		turn.ResolvedQuestions = append(turn.ResolvedQuestions, *res)
	}
	if len(turn.ResolvedQuestions) > 0 {
		s.Plan.CurrentStage = model.StageQuestionResolution
	}
}

func (e *Engine) applyPlaybookEffects(s *model.Session, turn *model.ProtocolTurn) {
	if len(s.Hypotheses) == 0 {
		return
	}
	for _, res := range turn.ResolvedQuestions {
		pq := findPlaybookQuestion(e.playbook, res.QuestionID)
		if pq == nil {
			continue
		}
		confirmed := res.Status == model.ResolutionConfirmed
		rejected := res.Status == model.ResolutionRejected
		for _, eff := range pq.Effects {
			apply := (eff.WhenTrue && confirmed) || (!eff.WhenTrue && rejected)
			if !apply {
				continue
			}
			for i := range s.Hypotheses {
				if s.Hypotheses[i].ID != eff.HypothesisID {
					continue
				}
				delta := eff.Amount
				if eff.Action == playbook.EffectDecrease {
					delta = -eff.Amount
				}
				s.Hypotheses[i].Confidence = clamp(s.Hypotheses[i].Confidence+delta, 0, 100)
			}
		}
	}
	normalizeHypothesisConfidences(s.Hypotheses)
	s.Plan.CurrentStage = model.StageHypothesisEvaluation
}

func (e *Engine) updatePlanLists(s *model.Session) {
	s.Plan.OpenQuestions = s.Plan.OpenQuestions[:0]
	s.Plan.CompletedQuestions = s.Plan.CompletedQuestions[:0]
	for _, q := range s.Plan.Questions {
		switch q.Status {
		case model.QuestionAnswered, model.QuestionRejected:
			s.Plan.CompletedQuestions = append(s.Plan.CompletedQuestions, q.ID)
		default:
			s.Plan.OpenQuestions = append(s.Plan.OpenQuestions, q.ID)
		}
	}
	if len(s.Plan.OpenQuestions) > 0 && len(s.Evidence) > 0 {
		s.Plan.CurrentStage = model.StageNeedMoreEvidence
	}
}

// ResolveQuestion explicitly resolves a question (MCP resolve_question tool).
func (e *Engine) ResolveQuestion(s *model.Session, questionID string, confirmed bool, reason string, now time.Time) (*model.QuestionResolution, error) {
	for i := range s.Plan.Questions {
		if s.Plan.Questions[i].ID != questionID {
			continue
		}
		q := &s.Plan.Questions[i]
		status := model.ResolutionConfirmed
		qStatus := model.QuestionAnswered
		if !confirmed {
			status = model.ResolutionRejected
			qStatus = model.QuestionRejected
		}
		res := &model.QuestionResolution{
			QuestionID:         questionID,
			Status:             status,
			Confidence:         91,
			Reason:             reason,
			SupportingEvidence: append([]string(nil), q.SupportingEvidence...),
			ResolvedAt:         now,
		}
		q.Resolution = res
		q.Status = qStatus
		q.Confidence = res.Confidence
		s.Plan.ResolutionHistory = append(s.Plan.ResolutionHistory, *res)
		turn := model.ProtocolTurn{ResolvedQuestions: []model.QuestionResolution{*res}}
		e.applyPlaybookEffects(s, &turn)
		e.updatePlanLists(s)
		return res, nil
	}
	return nil, fmt.Errorf("question %q not found", questionID)
}

// ListOpenQuestions returns unresolved questions sorted by priority.
func ListOpenQuestions(plan *model.InvestigationPlan) []model.Question {
	if plan == nil {
		return nil
	}
	var open []model.Question
	for _, q := range plan.Questions {
		if q.Status != model.QuestionAnswered && q.Status != model.QuestionRejected {
			open = append(open, q)
		}
	}
	sort.SliceStable(open, func(i, j int) bool {
		return open[i].Priority > open[j].Priority
	})
	return open
}

func playbookQuestionToModel(pq playbook.PlaybookQuestion) model.Question {
	title := pq.Title
	if title == "" {
		title = pq.ID
	}
	return model.Question{
		ID:               pq.ID,
		Title:            title,
		Description:      pq.Description,
		Priority:         pq.Priority,
		Status:           model.QuestionWaitingForEvidence,
		RequiredEvidence: append([]model.Category(nil), pq.Requires...),
		DependsOn:        append([]string(nil), pq.DependsOn...),
	}
}

func findPlaybookQuestion(pb *playbook.Playbook, id string) *playbook.PlaybookQuestion {
	for i := range pb.Questions {
		if pb.Questions[i].ID == id {
			return &pb.Questions[i]
		}
	}
	return nil
}

func dependenciesMet(plan *model.InvestigationPlan, deps []string) bool {
	if len(deps) == 0 {
		return true
	}
	answered := map[string]bool{}
	for _, q := range plan.Questions {
		if q.Status == model.QuestionAnswered {
			answered[q.ID] = true
		}
	}
	for _, d := range deps {
		if !answered[d] {
			return false
		}
	}
	return true
}

func signalTriggered(sig engine.Signals, trigger string) bool {
	if sig.Keywords[trigger] {
		return true
	}
	switch trigger {
	case "config":
		return sig.Categories[model.CategoryConfigurationChanges] > 0
	case "restart":
		return sig.Keywords["restart"] || sig.Keywords["memory"]
	case "deploy_detected":
		return sig.FirstDeployment != nil
	case "database":
		return sig.Keywords["database"] || sig.Categories[model.CategoryDatabaseEvents] > 0
	case "cert":
		return sig.Keywords["cert"] || sig.Categories[model.CategorySecurityEvents] > 0
	case "dns":
		return sig.Keywords["dns"] || sig.Categories[model.CategoryNetworkEvents] > 0
	case "memory":
		return sig.Keywords["memory"] || sig.Keywords["restart"]
	case "dependency":
		return sig.Keywords["dependency"] || sig.Categories[model.CategoryTraceEvents] > 0
	case "external":
		return sig.Keywords["external"]
	case "auth":
		return sig.Keywords["auth"] || sig.Categories[model.CategorySecurityEvents] > 0
	case "human":
		return sig.Keywords["human"] || sig.Categories[model.CategoryHumanContext] > 0
	case "capacity":
		return sig.Keywords["capacity"] || sig.Categories[model.CategoryMetrics] > 0
	case "security":
		return sig.Keywords["security"]
	case "performance":
		return sig.Keywords["performance"] || sig.Keywords["latency"]
	default:
		return false
	}
}

func openQuestions(plan *model.InvestigationPlan) []model.Question {
	return ListOpenQuestions(plan)
}

func missingCategories(s *model.Session, required []model.Category) []model.Category {
	var out []model.Category
	for _, c := range required {
		if !s.HasCategory(c) {
			out = append(out, c)
		}
	}
	return out
}

func categoriesPresent(s *model.Session, cats []model.Category) bool {
	for _, c := range cats {
		if !s.HasCategory(c) {
			return false
		}
	}
	return true
}

func expectedGain(q model.Question, missing []model.Category) float64 {
	base := float64(q.Priority) / 3
	return clamp(base+float64(len(missing))*8, 10, 40)
}

func priorityForQuestion(q model.Question) model.Priority {
	if q.Priority >= 80 {
		return model.PriorityHigh
	}
	if q.Priority >= 50 {
		return model.PriorityMedium
	}
	return model.PriorityLow
}

func joinCategories(cats []model.Category) string {
	if len(cats) == 0 {
		return ""
	}
	out := string(cats[0])
	for _, c := range cats[1:] {
		out += ", " + string(c)
	}
	return out
}

func protocolToLegacyRequests(requests []model.ProtocolEvidenceRequest, open []model.Question) []model.EvidenceRequest {
	legacy := make([]model.EvidenceRequest, 0, len(requests))
	for _, r := range requests {
		if len(r.Categories) == 0 {
			continue
		}
		legacy = append(legacy, model.EvidenceRequest{
			Category: r.Categories[0],
			Priority: r.Priority,
			Reason:   r.Reason,
		})
	}
	if len(legacy) > 0 {
		return legacy
	}
	// Fallback: derive from open questions directly.
	for _, q := range open {
		for _, c := range q.RequiredEvidence {
			legacy = append(legacy, model.EvidenceRequest{
				Category: c,
				Priority: priorityForQuestion(q),
				Reason:   q.Title,
			})
		}
	}
	return legacy
}

func missingLegacyRequests(s *model.Session, required []model.EvidenceRequest) []model.EvidenceRequest {
	var out []model.EvidenceRequest
	for _, r := range required {
		if !s.HasCategory(r.Category) {
			out = append(out, r)
		}
	}
	if out == nil {
		return []model.EvidenceRequest{}
	}
	return out
}

func updatePartialStatus(q *model.Question, s *model.Session) {
	present := 0
	for _, c := range q.RequiredEvidence {
		if s.HasCategory(c) {
			present++
		}
	}
	if present == 0 {
		q.Status = model.QuestionWaitingForEvidence
	} else if present < len(q.RequiredEvidence) {
		q.Status = model.QuestionPartiallyAnswered
	}
}

func activeHypothesisIDs(hyps []model.Hypothesis) []string {
	var out []string
	for _, h := range hyps {
		if h.Status != model.StatusRefuted {
			out = append(out, h.ID)
		}
	}
	return out
}

func deriveStage(s *model.Session) model.InvestigationStage {
	if s.Status == model.StatusCompleted {
		return model.StageCompleted
	}
	if s.Plan == nil {
		return model.StagePlanning
	}
	if len(s.Plan.OpenQuestions) == 0 && len(s.Plan.CompletedQuestions) > 0 && s.Sufficiency.CanAnswer {
		return model.StageCompleted
	}
	return s.Plan.CurrentStage
}

func computeProtocolMetrics(s *model.Session, prevConfidence float64) model.ProtocolMetrics {
	m := model.ProtocolMetrics{}
	if s.Plan == nil {
		return m
	}
	m.TotalQuestions = len(s.Plan.Questions)
	for _, q := range s.Plan.Questions {
		switch q.Status {
		case model.QuestionAnswered:
			m.ResolvedQuestions++
		case model.QuestionRejected:
			m.RejectedQuestions++
		default:
			m.PendingQuestions++
		}
	}
	m.EvidenceRequestsCreated = len(s.Plan.EvidenceRequests)
	for _, r := range s.Plan.EvidenceRequests {
		if r.Status == model.RequestFulfilled {
			m.EvidenceRequestsCompleted++
		}
	}
	var confSum float64
	for _, r := range s.Plan.ResolutionHistory {
		confSum += r.Confidence
	}
	if len(s.Plan.ResolutionHistory) > 0 {
		m.AverageResolutionConfidence = round1(confSum / float64(len(s.Plan.ResolutionHistory)))
	}
	m.AverageConfidenceGain = round1(s.Confidence - prevConfidence)
	m.PlannerIterations = s.Metrics.PlannerIterations
	return m
}

func buildQuestionGraph(plan *model.InvestigationPlan) model.QuestionGraph {
	g := model.QuestionGraph{}
	if plan == nil {
		return g
	}
	for _, q := range plan.Questions {
		g.Nodes = append(g.Nodes, model.QuestionGraphNode{
			QuestionID: q.ID,
			Title:      q.Title,
			Status:     q.Status,
		})
		for _, dep := range q.DependsOn {
			g.Edges = append(g.Edges, model.QuestionGraphEdge{
				From:     dep,
				To:       q.ID,
				Relation: "depends_on",
			})
		}
	}
	return g
}

func normalizeHypothesisConfidences(hyps []model.Hypothesis) {
	var total float64
	for i := range hyps {
		if hyps[i].Confidence < 0 {
			hyps[i].Confidence = 0
		}
		total += hyps[i].Confidence
	}
	if total <= 0 {
		return
	}
	for i := range hyps {
		hyps[i].Confidence = round1(hyps[i].Confidence / total * 100)
	}
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func round1(v float64) float64 {
	return float64(int(v*10+0.5)) / 10
}
