package reasoning

import (
	"math"
	"strings"

	"github.com/stackrail/incident-investigator/internal/graph"
	"github.com/stackrail/incident-investigator/internal/model"
)

// Applier executes validated reasoning actions against session state.
type Applier struct{}

// NewApplier returns the default action applier.
func NewApplier() *Applier { return &Applier{} }

// ApplyResult holds applied and rejected actions.
type ApplyResult struct {
	Applied  []model.ReasoningAction
	Rejected []model.RejectedAction
}

// Apply runs each action through validation then mutates session state.
func (a *Applier) Apply(s *model.Session, g *graph.InvestigationGraph, actions []model.ReasoningAction, v *Validator) ApplyResult {
	var result ApplyResult
	for _, act := range actions {
		if err := v.Validate(s, g, act); err != nil {
			result.Rejected = append(result.Rejected, model.RejectedAction{Action: act, Reason: err.Error()})
			continue
		}
		if a.applyOne(s, g, act) {
			result.Applied = append(result.Applied, act)
		} else {
			result.Rejected = append(result.Rejected, model.RejectedAction{Action: act, Reason: "action had no effect"})
		}
	}
	return result
}

func (a *Applier) applyOne(s *model.Session, g *graph.InvestigationGraph, act model.ReasoningAction) bool {
	switch act.Type {
	case model.ActionIncreaseHypothesisConfidence:
		return adjustHypothesisConfidence(s, act.HypothesisID, act.Delta)
	case model.ActionDecreaseHypothesisConfidence:
		return adjustHypothesisConfidence(s, act.HypothesisID, -act.Delta)
	case model.ActionCreateHypothesis:
		return appendHypothesis(s, act.Hypothesis)
	case model.ActionRejectHypothesis:
		return setHypothesisStatus(s, act.HypothesisID, model.StatusRefuted)
	case model.ActionPromoteHypothesis:
		return setHypothesisStatus(s, act.HypothesisID, model.StatusLeading)
	case model.ActionReplaceHypotheses:
		s.Hypotheses = act.Hypotheses
		return true
	case model.ActionSetContradictions:
		s.Contradictions = act.Contradictions
		return true
	case model.ActionMarkContradiction:
		return appendContradiction(s, act.Contradiction)
	case model.ActionSetTimeline:
		s.Timeline = act.Timeline
		return true
	case model.ActionUpdateGraph:
		if act.Graph != nil {
			s.Graph = act.Graph
		}
		return act.Graph != nil
	case model.ActionSetBlastRadius:
		s.BlastRadius = act.BlastRadius
		return true
	case model.ActionSetEvidencePlan:
		s.RequiredEvidence = act.RequiredEvidence
		s.MissingEvidence = act.MissingEvidence
		return true
	case model.ActionSetCoverage:
		s.Coverage = act.Coverage
		return true
	case model.ActionSetStrategy:
		s.Strategy = act.Strategy
		return true
	case model.ActionSetSufficiency:
		if act.Sufficiency != nil {
			s.Sufficiency = *act.Sufficiency
		}
		return act.Sufficiency != nil
	case model.ActionSetConfidence:
		s.Confidence = clampConfidence(act.Confidence)
		return true
	case model.ActionSetProgress:
		s.Progress = clampPercent(act.Progress)
		return true
	case model.ActionSetInvestigationProgress:
		if act.InvestigationProgress != nil {
			s.InvestigationProgress = *act.InvestigationProgress
		}
		return act.InvestigationProgress != nil
	case model.ActionSetConfidenceBreakdown:
		if act.ConfidenceBreakdown != nil {
			s.ConfidenceBreakdown = *act.ConfidenceBreakdown
		}
		return act.ConfidenceBreakdown != nil
	case model.ActionSetEvidenceImportance:
		s.EvidenceImportance = act.EvidenceImportance
		return true
	case model.ActionLinkGraphNodes:
		if g == nil || act.GraphEdge == nil {
			return false
		}
		edge := *act.GraphEdge
		if edgeExists(g, edge) {
			return true // idempotent merge
		}
		_ = g.AddEdge(&edge)
		if s.Graph != nil {
			s.Graph = g.View()
		}
		return true
	case model.ActionCreateQuestion:
		return appendQuestion(s, act.Question)
	case model.ActionResolveQuestion:
		return resolveQuestion(s, act.QuestionID, model.QuestionAnswered)
	case model.ActionRejectQuestion:
		return resolveQuestion(s, act.QuestionID, model.QuestionRejected)
	case model.ActionCreateEvidenceRequest:
		return appendEvidenceRequest(s, act.EvidenceRequest)
	case model.ActionCreateRecommendation:
		if s.Plan == nil {
			s.Plan = &model.InvestigationPlan{}
		}
		// recommendations stored on plan metadata via journal for now
		return strings.TrimSpace(act.Recommendation) != ""
	case model.ActionMarkInvestigationComplete:
		s.State = model.StateCompleted
		s.Status = model.StatusCompleted
		return true
	default:
		return false
	}
}

func adjustHypothesisConfidence(s *model.Session, id string, delta float64) bool {
	for i := range s.Hypotheses {
		if s.Hypotheses[i].ID == id {
			s.Hypotheses[i].Confidence = clampConfidence(s.Hypotheses[i].Confidence + delta)
			normalizeHypothesisConfidences(s.Hypotheses)
			return true
		}
	}
	return false
}

func normalizeHypothesisConfidences(hs []model.Hypothesis) {
	var sum float64
	for _, h := range hs {
		if h.Status != model.StatusRefuted {
			sum += h.Confidence
		}
	}
	if sum <= 0 {
		return
	}
	for i := range hs {
		if hs[i].Status == model.StatusRefuted {
			continue
		}
		hs[i].Confidence = round1(hs[i].Confidence / sum * 100)
	}
}

func appendHypothesis(s *model.Session, h *model.Hypothesis) bool {
	if h == nil {
		return false
	}
	for _, existing := range s.Hypotheses {
		if existing.ID == h.ID {
			return false
		}
	}
	s.Hypotheses = append(s.Hypotheses, *h)
	return true
}

func setHypothesisStatus(s *model.Session, id string, status model.HypothesisStatus) bool {
	for i := range s.Hypotheses {
		if s.Hypotheses[i].ID == id {
			s.Hypotheses[i].Status = status
			return true
		}
	}
	return false
}

func appendContradiction(s *model.Session, c *model.Contradiction) bool {
	if c == nil {
		return false
	}
	for _, existing := range s.Contradictions {
		if existing.ID == c.ID {
			return true
		}
	}
	s.Contradictions = append(s.Contradictions, *c)
	return true
}

func appendQuestion(s *model.Session, q *model.Question) bool {
	if q == nil || s.Plan == nil {
		return false
	}
	for _, existing := range s.Plan.Questions {
		if existing.ID == q.ID {
			return false
		}
	}
	s.Plan.Questions = append(s.Plan.Questions, *q)
	return true
}

func resolveQuestion(s *model.Session, id string, status model.QuestionStatus) bool {
	if s.Plan == nil {
		return false
	}
	for i := range s.Plan.Questions {
		if s.Plan.Questions[i].ID == id {
			s.Plan.Questions[i].Status = status
			return true
		}
	}
	return false
}

func appendEvidenceRequest(s *model.Session, req *model.ProtocolEvidenceRequest) bool {
	if req == nil || s.Plan == nil {
		return false
	}
	for _, existing := range s.Plan.EvidenceRequests {
		if existing.ID == req.ID {
			return false
		}
	}
	s.Plan.EvidenceRequests = append(s.Plan.EvidenceRequests, *req)
	return true
}

func edgeExists(g *graph.InvestigationGraph, e model.GraphEdge) bool {
	for _, existing := range g.Edges() {
		if existing.From == e.From && existing.To == e.To && existing.Type == e.Type {
			return true
		}
	}
	return false
}

func clampConfidence(v float64) float64 {
	return math.Max(0, math.Min(100, v))
}

func clampPercent(v float64) float64 {
	return math.Max(0, math.Min(100, v))
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}
