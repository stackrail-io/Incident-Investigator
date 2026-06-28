package engine

import (
	"strings"
	"time"

	"github.com/stackrail/incident-investigator/internal/model"
)

// ImportanceEngine scores how much each evidence item contributed.
type ImportanceEngine interface {
	Score(s *model.Session, sig Signals, hyps []model.Hypothesis) []model.EvidenceImportance
}

// HeuristicImportanceEngine assigns deterministic importance scores.
type HeuristicImportanceEngine struct{}

// NewHeuristicImportanceEngine returns the default importance engine.
func NewHeuristicImportanceEngine() *HeuristicImportanceEngine { return &HeuristicImportanceEngine{} }

// Score implements ImportanceEngine.
func (i *HeuristicImportanceEngine) Score(s *model.Session, sig Signals, hyps []model.Hypothesis) []model.EvidenceImportance {
	summaryCounts := map[string]int{}
	supporting := supportingSet(hyps)

	out := make([]model.EvidenceImportance, 0, len(s.Evidence))
	for _, e := range s.Evidence {
		score := baseImportance(e, sig, supporting)
		norm := strings.ToLower(strings.TrimSpace(e.Summary))
		summaryCounts[norm]++
		if summaryCounts[norm] > 1 {
			score = round1(score / float64(summaryCounts[norm]))
		}
		out = append(out, model.EvidenceImportance{
			EvidenceID: e.ID,
			Category:   e.Category,
			Summary:    e.Summary,
			Score:      clamp(score, 1, 100),
		})
	}
	return out
}

func supportingSet(hyps []model.Hypothesis) map[string]bool {
	set := map[string]bool{}
	for _, h := range hyps {
		for _, id := range h.SupportingEvidence {
			set[id] = true
		}
	}
	return set
}

func baseImportance(e *model.Evidence, sig Signals, supporting map[string]bool) float64 {
	text := haystack(e)
	score := 20.0

	switch e.Category {
	case model.CategoryDeploymentEvents:
		score = 75
		if sig.FirstDeployment != nil && sig.FirstDeployment.ID == e.ID {
			score = 95
		}
		if matchesAny(text, signalKeywords["rollback"]) {
			score = 88
		}
	case model.CategoryAlertEvents:
		score = 70
		if sig.IncidentOnset != nil && sig.IncidentOnset.ID == e.ID {
			score = 90
		}
	case model.CategoryMetrics:
		score = 55
		if matchesAny(text, signalKeywords["latency"]) || matchesAny(text, signalKeywords["error"]) {
			score = 81
		}
	case model.CategoryApplicationLogs:
		score = 45
		if matchesAny(text, signalKeywords["error"]) {
			score = 65
		}
	case model.CategoryDatabaseEvents:
		score = 72
	case model.CategoryHumanContext:
		score = 23
	case model.CategoryTraceEvents:
		score = 60
	default:
		score = 30
	}

	if supporting[e.ID] {
		score += 15
	}
	if !e.Timestamp.IsZero() && sig.IncidentOnset != nil &&
		absDuration(e.Timestamp.Sub(sig.IncidentOnset.Timestamp)) < 2*time.Minute {
		score += 10
	}
	return score
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
