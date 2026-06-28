package engine

import (
	"github.com/stackrail/incident-investigator/internal/model"
)

// StrategyEngine prioritizes the highest-value next evidence to collect.
type StrategyEngine interface {
	NextSteps(s *model.Session, sig Signals, hyps []model.Hypothesis, required []model.EvidenceRequest) []model.NextStep
}

// HeuristicStrategyEngine wraps the planner output into prioritized next steps.
type HeuristicStrategyEngine struct {
	planner *HeuristicPlanner
}

// NewHeuristicStrategyEngine returns the default strategy engine.
func NewHeuristicStrategyEngine() *HeuristicStrategyEngine {
	return &HeuristicStrategyEngine{planner: NewHeuristicPlanner()}
}

// NextSteps implements StrategyEngine.
func (st *HeuristicStrategyEngine) NextSteps(s *model.Session, sig Signals, hyps []model.Hypothesis, required []model.EvidenceRequest) []model.NextStep {
	if required == nil {
		required = st.planner.Plan(s, sig)
	}

	missing := NewHeuristicMissingEvidenceDetector().Detect(s, required)
	if len(missing) == 0 {
		return []model.NextStep{}
	}

	leadID := ""
	if len(hyps) > 0 {
		leadID = hyps[0].ID
	}

	steps := make([]model.NextStep, 0, len(missing))
	for _, m := range missing {
		gain := expectedGain(m, sig, hyps, s.Goal)
		blocks := hypothesesBlockedBy(m.Category, hyps, leadID)
		steps = append(steps, model.NextStep{
			Category:               m.Category,
			Reason:                 strategyReason(m, hyps, leadID),
			ExpectedConfidenceGain: gain,
			BlocksHypothesis:       blocks,
		})
	}

	sortNextSteps(steps)

	// Return at most two highest-value steps.
	if len(steps) > 2 {
		steps = steps[:2]
	}
	for i := range steps {
		steps[i].Priority = i + 1
	}
	return steps
}

func sortNextSteps(steps []model.NextStep) {
	for i := 1; i < len(steps); i++ {
		for j := i; j > 0 && steps[j].ExpectedConfidenceGain > steps[j-1].ExpectedConfidenceGain; j-- {
			steps[j], steps[j-1] = steps[j-1], steps[j]
		}
	}
}

func expectedGain(m model.EvidenceRequest, sig Signals, hyps []model.Hypothesis, goal model.InvestigationGoal) float64 {
	base := m.Priority.Weight() * 8
	if goalRelevantCategory(goal, m.Category) {
		base += 10
	}
	if len(hyps) > 0 && hyps[0].Confidence < 60 {
		base += 5
	}
	if m.Category == model.CategoryDeploymentEvents && sig.FirstDeployment == nil {
		base += 15
	}
	if m.Category == model.CategoryMetrics && sig.Keywords["latency"] {
		base += 12
	}
	if m.Category == model.CategoryMetrics && goal == model.GoalBlastRadius {
		base += 18
	}
	if m.Category == model.CategoryTraceEvents && goal == model.GoalBlastRadius {
		base += 16
	}
	if m.Category == model.CategoryAlertEvents && goal == model.GoalBlastRadius {
		base += 14
	}
	if m.Category == model.CategoryDatabaseEvents && sig.Keywords["database"] {
		base += 14
	}
	return round1(clamp(base, 5, 35))
}

func goalRelevantCategory(goal model.InvestigationGoal, c model.Category) bool {
	for _, g := range goalCategories(goal) {
		if g == c {
			return true
		}
	}
	return false
}

func hypothesesBlockedBy(cat model.Category, hyps []model.Hypothesis, leadID string) []string {
	var out []string
	for _, h := range hyps {
		if h.Status == model.StatusRefuted {
			continue
		}
		if categorySupportsHypothesis(cat, h.ID) && h.ID != leadID {
			out = append(out, h.ID)
		}
	}
	if out == nil {
		return []string{}
	}
	return out
}

func categorySupportsHypothesis(cat model.Category, hypID string) bool {
	switch hypID {
	case "hypothesis-deployment-caused", "hypothesis-deployment-unrelated":
		return cat == model.CategoryDeploymentEvents || cat == model.CategoryConfigurationChanges
	case "hypothesis-database-saturation":
		return cat == model.CategoryDatabaseEvents || cat == model.CategoryMetrics
	case "hypothesis-network-dns":
		return cat == model.CategoryNetworkEvents
	case "hypothesis-certificate-expiry":
		return cat == model.CategorySecurityEvents
	case "hypothesis-resource-exhaustion":
		return cat == model.CategoryInfrastructureEvents || cat == model.CategoryMetrics
	case "hypothesis-retry-storm":
		return cat == model.CategoryMetrics || cat == model.CategoryTraceEvents
	default:
		return false
	}
}

func strategyReason(m model.EvidenceRequest, hyps []model.Hypothesis, leadID string) string {
	if m.Reason != "" {
		return m.Reason
	}
	if leadID != "" && len(hyps) > 0 {
		return "Current leading hypothesis (" + hyps[0].Statement + ") cannot be confirmed without " + string(m.Category) + "."
	}
	return "Collect " + string(m.Category) + " to advance the investigation."
}
