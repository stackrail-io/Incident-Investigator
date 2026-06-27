package engine

import "github.com/stackrail/incident-investigator/internal/model"

// ConfidenceScorer computes the engine's overall confidence in the
// investigation so far (0..100).
type ConfidenceScorer interface {
	Score(s *model.Session, sig Signals, hyps []model.Hypothesis, contradictions []model.Contradiction, progress float64) float64
}

// HeuristicConfidenceScorer blends the leading hypothesis strength, evidence
// coverage, multi-category corroboration and contradiction penalties.
type HeuristicConfidenceScorer struct{}

// NewHeuristicConfidenceScorer returns the default scorer.
func NewHeuristicConfidenceScorer() *HeuristicConfidenceScorer { return &HeuristicConfidenceScorer{} }

// Score implements ConfidenceScorer.
//
// Confidence rises when independent evidence agrees, temporal ordering matches
// and multiple categories support the leading theory; it falls when evidence
// conflicts or critical evidence is missing.
func (c *HeuristicConfidenceScorer) Score(s *model.Session, sig Signals, hyps []model.Hypothesis, contradictions []model.Contradiction, progress float64) float64 {
	if len(hyps) == 0 {
		return 0
	}

	lead := hyps[0]
	second := 0.0
	if len(hyps) > 1 {
		second = hyps[1].Confidence
	}

	// Separation: how decisively the leading hypothesis beats the runner-up.
	separation := (lead.Confidence - second) / 100 // 0..1

	coverage := progress / 100 // 0..1

	// Corroboration: distinct supporting categories behind the leading theory.
	supportCats := map[model.Category]bool{}
	idToCat := map[string]model.Category{}
	for _, e := range s.Evidence {
		idToCat[e.ID] = e.Category
	}
	for _, id := range lead.SupportingEvidence {
		if cat, ok := idToCat[id]; ok {
			supportCats[cat] = true
		}
	}
	corroboration := float64(len(supportCats)) / 3
	if corroboration > 1 {
		corroboration = 1
	}

	// Temporal bonus: a clean before/after ordering is strong evidence.
	temporal := 0.0
	if sig.DeployBeforeIncident || (sig.IncidentOnset != nil && sig.Recovery != nil) {
		temporal = 1
	}

	base := 0.45*lead.Confidence +
		25*separation +
		15*corroboration +
		15*temporal

	// Scale down when we simply have not collected enough evidence yet.
	base *= 0.4 + 0.6*coverage

	// Contradictions erode confidence.
	penalty := 0.0
	for _, ct := range contradictions {
		switch ct.Severity {
		case "high":
			penalty += 12
		case "medium":
			penalty += 6
		default:
			penalty += 3
		}
	}
	base -= penalty

	return clamp(round1(base), 0, 99)
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
