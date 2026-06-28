package intelligence

import (
	"fmt"
	"math"

	"github.com/stackrail/incident-investigator/internal/model"
)

// ConfidenceCalibrator adjusts hypothesis confidence using historical snapshots.
type ConfidenceCalibrator interface {
	Calibrate(hypothesis *model.Hypothesis, history []*model.InvestigationSnapshot) (float64, *model.CalibrationExplanation)
}

// HeuristicCalibrator uses historical accuracy for similar root causes.
type HeuristicCalibrator struct{}

// NewHeuristicCalibrator returns the default calibrator.
func NewHeuristicCalibrator() *HeuristicCalibrator { return &HeuristicCalibrator{} }

// Calibrate implements ConfidenceCalibrator.
func (c *HeuristicCalibrator) Calibrate(hypothesis *model.Hypothesis, history []*model.InvestigationSnapshot) (float64, *model.CalibrationExplanation) {
	if hypothesis == nil {
		return 0, &model.CalibrationExplanation{}
	}
	raw := hypothesis.Confidence
	if len(history) == 0 {
		return raw, &model.CalibrationExplanation{
			SupportingHistory: []string{"No historical investigations available."},
		}
	}

	var similar, correct int
	var supporting []string
	for _, snap := range history {
		if snap.RootCause == "" {
			continue
		}
		similar++
		if snap.RootCause == hypothesis.ID {
			correct++
			supporting = append(supporting, fmt.Sprintf("%s confirmed %s at %.0f%% confidence",
				snap.InvestigationID, snap.RootCause, snap.Confidence))
		}
	}
	if similar == 0 {
		// Fall back to service/goal mean confidence blend.
		var sum float64
		for _, snap := range history {
			sum += snap.Confidence
		}
		mean := sum / float64(len(history))
		calibrated := raw*0.8 + mean*0.2
		return round1(math.Max(0, math.Min(100, calibrated))), &model.CalibrationExplanation{
			SimilarInvestigations: len(history),
			TotalComparisons:      len(history),
			SupportingHistory:     []string{fmt.Sprintf("Blended with historical mean confidence %.1f%%.", mean)},
		}
	}

	historicalRate := float64(correct) / float64(similar) * 100
	// Blend raw confidence with historical accuracy for this hypothesis.
	calibrated := raw*0.55 + historicalRate*0.45
	calibrated = math.Max(0, math.Min(100, calibrated))

	return round1(calibrated), &model.CalibrationExplanation{
		SimilarInvestigations: similar,
		CorrectCount:          correct,
		TotalComparisons:      similar,
		SupportingHistory:     supporting,
	}
}

// CalibrateFromRequest calibrates session-level confidence for the API layer.
func CalibrateFromRequest(req model.CalibrationRequest, history []*model.InvestigationSnapshot, cal ConfidenceCalibrator) *model.CalibrationResponse {
	raw := req.RawConfidence
	hypID := req.HypothesisID
	var hyp *model.Hypothesis
	if req.Session != nil {
		for i := range req.Session.Hypotheses {
			if hypID != "" && req.Session.Hypotheses[i].ID == hypID {
				hyp = &req.Session.Hypotheses[i]
				break
			}
		}
		if hyp == nil && len(req.Session.Hypotheses) > 0 {
			hyp = &req.Session.Hypotheses[0]
			hypID = hyp.ID
		}
	}
	if hyp == nil {
		hyp = &model.Hypothesis{ID: hypID, Confidence: raw}
		if hypID == "" {
			hyp.ID = "leading"
		}
	} else {
		hyp = &model.Hypothesis{ID: hyp.ID, Confidence: raw}
	}

	calibrated, expl := cal.Calibrate(hyp, filterHistory(history, req))
	delta := calibrated - raw

	reason := fmt.Sprintf("Calibrated using %d historical investigations.", len(history))
	if expl != nil && expl.CorrectCount > 0 {
		reason = fmt.Sprintf("%d similar investigations; hypothesis %s confirmed %d/%d times.",
			expl.SimilarInvestigations, hyp.ID, expl.CorrectCount, expl.TotalComparisons)
	}

	return &model.CalibrationResponse{
		OriginalConfidence:   round1(raw),
		CalibratedConfidence: calibrated,
		Delta:                round1(delta),
		Reason:               reason,
		HistoricalSampleSize: len(history),
		HypothesisID:         hyp.ID,
		Explanation:          expl,
	}
}

func filterHistory(all []*model.InvestigationSnapshot, req model.CalibrationRequest) []*model.InvestigationSnapshot {
	var out []*model.InvestigationSnapshot
	for _, snap := range all {
		if req.SessionID != "" && snap.InvestigationID == req.SessionID {
			continue
		}
		if req.Service != "" {
			if svc := serviceFromSnapshot(snap); svc != "" && svc != req.Service {
				continue
			}
		}
		if req.Goal != "" && snap.Goal != req.Goal {
			continue
		}
		out = append(out, snap)
	}
	return out
}
