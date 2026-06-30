// Package explain builds human-readable explanations for investigation decisions.
//
// Every conclusion should have an explanation path: why a hypothesis leads,
// why another does not, why confidence is at its level, and why more evidence
// is required.
package explain

import (
	"fmt"
	"strings"

	"github.com/stackrail/incident-investigator/internal/model"
)

// Explanation is a structured answer to a why-question.
type Explanation struct {
	Subject string   `json:"subject"`
	Summary string   `json:"summary"`
	Reasons []string `json:"reasons"`
}

// WhyHypothesis explains why a hypothesis is ranked where it is.
func WhyHypothesis(s *model.Session, hypothesisID string) Explanation {
	exp := Explanation{Subject: hypothesisID}
	if s == nil {
		exp.Summary = "No session available."
		return exp
	}
	var hyp *model.Hypothesis
	for i := range s.Hypotheses {
		if s.Hypotheses[i].ID == hypothesisID {
			hyp = &s.Hypotheses[i]
			break
		}
	}
	if hyp == nil {
		exp.Summary = fmt.Sprintf("Hypothesis %q is not in the current field.", hypothesisID)
		return exp
	}
	leading := leadingHypothesis(s)
	if leading != nil && leading.ID == hypothesisID {
		exp.Summary = fmt.Sprintf("%s leads the hypothesis field at %.1f%% confidence.", hypothesisID, hyp.Confidence)
	} else {
		exp.Summary = fmt.Sprintf("%s has %.1f%% confidence.", hypothesisID, hyp.Confidence)
	}
	if hyp.Rationale != "" {
		exp.Reasons = append(exp.Reasons, hyp.Rationale)
	}
	if len(hyp.SupportingEvidence) > 0 {
		exp.Reasons = append(exp.Reasons, fmt.Sprintf("Supported by evidence: %s.", strings.Join(hyp.SupportingEvidence, ", ")))
	}
	if len(hyp.ConflictingEvidence) > 0 {
		exp.Reasons = append(exp.Reasons, fmt.Sprintf("Conflicted by evidence: %s.", strings.Join(hyp.ConflictingEvidence, ", ")))
	}
	if leading != nil && leading.ID != hypothesisID {
		exp.Reasons = append(exp.Reasons, fmt.Sprintf("Trailing leader %s at %.1f%%.", leading.ID, leading.Confidence))
	}
	return exp
}

// WhyNotHypothesis explains why another hypothesis does not lead.
func WhyNotHypothesis(s *model.Session, hypothesisID, otherID string) Explanation {
	exp := Explanation{Subject: otherID}
	if s == nil {
		exp.Summary = "No session available."
		return exp
	}
	leading := leadingHypothesis(s)
	if leading == nil {
		exp.Summary = "No leading hypothesis yet."
		return exp
	}
	if leading.ID == otherID {
		exp.Summary = fmt.Sprintf("%s currently leads; use WhyHypothesis instead.", otherID)
		return exp
	}
	var other *model.Hypothesis
	for i := range s.Hypotheses {
		if s.Hypotheses[i].ID == otherID {
			other = &s.Hypotheses[i]
			break
		}
	}
	if other == nil {
		exp.Summary = fmt.Sprintf("Hypothesis %q is not in the field.", otherID)
		return exp
	}
	gap := leading.Confidence - other.Confidence
	exp.Summary = fmt.Sprintf("%s does not lead: %s is ahead by %.1f points (%.1f%% vs %.1f%%).",
		otherID, leading.ID, gap, leading.Confidence, other.Confidence)
	if other.Rationale != "" {
		exp.Reasons = append(exp.Reasons, other.Rationale)
	}
	if len(other.ConflictingEvidence) > 0 {
		exp.Reasons = append(exp.Reasons, "Conflicting evidence reduces this hypothesis.")
	}
	if hypothesisID != "" && hypothesisID != otherID {
		exp.Reasons = append(exp.Reasons, fmt.Sprintf("Compared against reference hypothesis %s.", hypothesisID))
	}
	return exp
}

// WhyConfidence explains the current confidence level.
func WhyConfidence(s *model.Session) Explanation {
	exp := Explanation{Subject: "confidence"}
	if s == nil {
		exp.Summary = "No session available."
		return exp
	}
	bd := s.ConfidenceBreakdown
	exp.Summary = fmt.Sprintf("Overall confidence is %.1f%%.", s.Confidence)
	if bd.Overall > 0 || bd.LeadingHypothesis > 0 {
		exp.Reasons = append(exp.Reasons,
			fmt.Sprintf("Leading hypothesis contribution: %.1f", bd.LeadingHypothesis),
			fmt.Sprintf("Separation from runner-up: %.1f", bd.Separation),
			fmt.Sprintf("Corroboration: %.1f", bd.Corroboration),
			fmt.Sprintf("Temporal signals: %.1f", bd.Temporal),
			fmt.Sprintf("Coverage factor: %.1f", bd.CoverageFactor),
		)
		if bd.ContradictionPenalty > 0 {
			exp.Reasons = append(exp.Reasons, fmt.Sprintf("Contradiction penalty: -%.1f", bd.ContradictionPenalty))
		}
	}
	if len(s.Contradictions) > 0 {
		exp.Reasons = append(exp.Reasons, fmt.Sprintf("%d active contradiction(s) affect confidence.", len(s.Contradictions)))
	}
	return exp
}

// WhyIncomplete explains why the investigation cannot conclude yet.
func WhyIncomplete(s *model.Session) Explanation {
	exp := Explanation{Subject: "sufficiency"}
	if s == nil {
		exp.Summary = "No session available."
		return exp
	}
	if s.Sufficiency.CanAnswer {
		exp.Summary = "Investigation is sufficient to answer."
		return exp
	}
	exp.Summary = "Investigation is incomplete."
	if s.Sufficiency.Reason != "" {
		exp.Reasons = append(exp.Reasons, s.Sufficiency.Reason)
	}
	for _, bq := range s.Sufficiency.BlockingQuestions {
		exp.Reasons = append(exp.Reasons, fmt.Sprintf("Blocking question: %s", bq.Question))
	}
	if s.Sufficiency.OverallConfidence > 0 {
		exp.Reasons = append(exp.Reasons, fmt.Sprintf("Overall confidence %.1f%% below threshold.", s.Sufficiency.OverallConfidence))
	}
	return exp
}

// WhyMoreEvidence explains what evidence is still needed.
func WhyMoreEvidence(s *model.Session) Explanation {
	exp := Explanation{Subject: "missing_evidence"}
	if s == nil {
		exp.Summary = "No session available."
		return exp
	}
	if len(s.Sufficiency.MissingEvidence) == 0 && len(s.MissingEvidence) == 0 {
		exp.Summary = "No missing evidence flagged by sufficiency engine."
		return exp
	}
	exp.Summary = "More evidence is required to increase confidence or resolve questions."
	for _, m := range s.Sufficiency.MissingEvidence {
		exp.Reasons = append(exp.Reasons, fmt.Sprintf("Need %s: %s", m.Category, m.Reason))
	}
	for _, m := range s.MissingEvidence {
		exp.Reasons = append(exp.Reasons, fmt.Sprintf("Planner requests %s (priority %s).", m.Category, m.Priority))
	}
	return exp
}

func leadingHypothesis(s *model.Session) *model.Hypothesis {
	if len(s.Hypotheses) == 0 {
		return nil
	}
	best := &s.Hypotheses[0]
	for i := 1; i < len(s.Hypotheses); i++ {
		if s.Hypotheses[i].Confidence > best.Confidence {
			best = &s.Hypotheses[i]
		}
	}
	return best
}
