package engine

import (
	"fmt"
	"time"

	"github.com/stackrail/incident-investigator/internal/model"
)

// UpdateReasoningTrace diffs hypotheses against the prior snapshot and appends traces.
func UpdateReasoningTrace(s *model.Session, now time.Time) {
	prev := map[string]model.Hypothesis{}
	for _, h := range s.PreviousHypotheses() {
		prev[h.ID] = h
	}

	for _, h := range s.Hypotheses {
		old, ok := prev[h.ID]
		delta := h.Confidence
		if ok {
			delta = round1(h.Confidence - old.Confidence)
		}
		if !ok || delta != 0 || len(h.SupportingEvidence) > len(old.SupportingEvidence) {
			trace := model.ReasoningTrace{
				ObservationID:         newObservationID(s, h.ID),
				HypothesisID:          h.ID,
				Reason:                reasoningReason(h, old, ok, delta),
				ConfidenceChange:      delta,
				SupportingEvidence:    append([]string(nil), h.SupportingEvidence...),
				ContradictingEvidence: append([]string(nil), h.ConflictingEvidence...),
				Timestamp:             now,
			}
			s.ReasoningTrace = append(s.ReasoningTrace, trace)
			appendHypothesisTrace(&h, trace)
		}
	}

	// Detect refuted/promoted hypotheses for journal-style events.
	for _, h := range s.Hypotheses {
		old, ok := prev[h.ID]
		if !ok {
			continue
		}
		if old.Status != model.StatusRefuted && h.Status == model.StatusRefuted {
			s.AddJournal("hypothesis_rejected", h.Statement, s.Confidence, now)
		}
		if old.Status != model.StatusLeading && h.Status == model.StatusLeading {
			s.AddJournal("hypothesis_promoted", h.Statement, h.Confidence, now)
		}
	}
}

func appendHypothesisTrace(h *model.Hypothesis, trace model.ReasoningTrace) {
	// Mutate session hypotheses in place via caller if needed; this helper is
	// used after hypotheses are assigned to the session.
	_ = h
}

// ApplyHypothesisTraces copies per-hypothesis traces onto session hypotheses.
func ApplyHypothesisTraces(s *model.Session) {
	byHyp := map[string][]model.ReasoningTrace{}
	for _, t := range s.ReasoningTrace {
		if t.HypothesisID != "" {
			byHyp[t.HypothesisID] = append(byHyp[t.HypothesisID], t)
		}
	}
	for i := range s.Hypotheses {
		if traces, ok := byHyp[s.Hypotheses[i].ID]; ok {
			s.Hypotheses[i].ReasoningTrace = traces
		}
	}
}

func reasoningReason(h model.Hypothesis, old model.Hypothesis, hadPrev bool, delta float64) string {
	if !hadPrev {
		return fmt.Sprintf("Hypothesis created: %s (%.0f%%).", h.Statement, h.Confidence)
	}
	if h.Status == model.StatusRefuted && old.Status != model.StatusRefuted {
		return fmt.Sprintf("Hypothesis rejected: %s.", h.Statement)
	}
	if delta > 0 {
		return fmt.Sprintf("Confidence increased by %.0f%% for: %s.", delta, h.Statement)
	}
	if delta < 0 {
		return fmt.Sprintf("Confidence decreased by %.0f%% for: %s.", -delta, h.Statement)
	}
	return fmt.Sprintf("Evidence updated for: %s.", h.Statement)
}

func newObservationID(s *model.Session, hypID string) string {
	return fmt.Sprintf("obs-%s-%d", hypID, len(s.ReasoningTrace)+1)
}

// UpdateJournal records investigation events based on session changes.
func UpdateJournal(s *model.Session, event string, detail string, now time.Time) {
	s.AddJournal(event, detail, s.Confidence, now)
}
