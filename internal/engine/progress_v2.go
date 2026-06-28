package engine

import "github.com/stackrail/incident-investigator/internal/model"

// ComputeInvestigationProgress derives the v0.2 progress model.
func ComputeInvestigationProgress(
	s *model.Session,
	coverage model.CoverageReport,
	sufficiency model.SufficiencyReport,
) model.InvestigationProgress {
	evidenceProgress := Progress(s, s.RequiredEvidence)

	openHyps := 0
	for _, h := range s.Hypotheses {
		if h.Status != model.StatusRefuted {
			openHyps++
		}
	}

	// Blend evidence collection, coverage, and open questions.
	questionFactor := 1.0
	if len(sufficiency.BlockingQuestions) > 0 {
		questionFactor = 1 - float64(len(sufficiency.BlockingQuestions))*0.12
		if questionFactor < 0.2 {
			questionFactor = 0.2
		}
	}

	percent := round1(0.45*evidenceProgress + 0.35*coverage.Overall + 0.20*(s.Confidence*questionFactor))
	if percent > 100 {
		percent = 100
	}

	return model.InvestigationProgress{
		PercentComplete:    percent,
		Confidence:         s.Confidence,
		Coverage:           coverage.Overall,
		RemainingQuestions: len(sufficiency.BlockingQuestions),
	}
}

// ComputeMetrics aggregates reasoning counters for the session.
func ComputeMetrics(s *model.Session, coverage model.CoverageReport, prevMetrics model.ReasoningMetrics) model.ReasoningMetrics {
	rejected := 0
	for _, h := range s.Hypotheses {
		if h.Status == model.StatusRefuted {
			rejected++
		}
	}

	gainSum := 0.0
	gainCount := 0
	for _, t := range s.ReasoningTrace {
		if t.ConfidenceChange != 0 {
			gainSum += t.ConfidenceChange
			gainCount++
		}
	}
	avgGain := 0.0
	if gainCount > 0 {
		avgGain = round1(gainSum / float64(gainCount))
	}

	return model.ReasoningMetrics{
		HypothesisCount:       len(s.Hypotheses),
		RejectedHypotheses:    rejected,
		ContradictionCount:    len(s.Contradictions),
		Coverage:              coverage.Overall,
		EvidenceCount:         len(s.Evidence),
		ReasoningIterations:   prevMetrics.ReasoningIterations + 1,
		PlannerIterations:     prevMetrics.PlannerIterations + 1,
		AverageConfidenceGain: avgGain,
	}
}

// ComputeConfidenceBreakdown explains the overall confidence score.
func ComputeConfidenceBreakdown(s *model.Session, sig Signals, hyps []model.Hypothesis, contradictions []model.Contradiction, progress float64) model.ConfidenceBreakdown {
	if len(hyps) == 0 {
		return model.ConfidenceBreakdown{}
	}

	lead := hyps[0]
	second := 0.0
	if len(hyps) > 1 {
		second = hyps[1].Confidence
	}
	separation := (lead.Confidence - second) / 100

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

	temporal := 0.0
	if sig.DeployBeforeIncident || (sig.IncidentOnset != nil && sig.Recovery != nil) {
		temporal = 1
	}

	coverageFactor := 0.4 + 0.6*(progress/100)

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

	base := 0.45*lead.Confidence + 25*separation + 15*corroboration + 15*temporal
	overall := clamp(round1(base*coverageFactor-penalty), 0, 99)

	return model.ConfidenceBreakdown{
		LeadingHypothesis:    round1(lead.Confidence),
		Separation:           round1(separation * 100),
		Corroboration:        round1(corroboration * 100),
		Temporal:             round1(temporal * 100),
		CoverageFactor:       round1(coverageFactor * 100),
		ContradictionPenalty: round1(penalty),
		Overall:              overall,
	}
}
