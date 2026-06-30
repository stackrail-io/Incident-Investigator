package explain

import (
	"testing"

	"github.com/stackrail/incident-investigator/internal/model"
)

func TestWhyHypothesisAndConfidence(t *testing.T) {
	s := &model.Session{
		Hypotheses: []model.Hypothesis{
			{ID: "hypothesis-deployment-caused", Statement: "Bad deploy", Confidence: 60, Rationale: "Deploy preceded errors"},
			{ID: "hypothesis-unknown", Statement: "Unknown", Confidence: 20},
		},
		Confidence: 55,
		ConfidenceBreakdown: model.ConfidenceBreakdown{LeadingHypothesis: 60, Overall: 55},
		Sufficiency: model.SufficiencyReport{CanAnswer: false, Reason: "Blocking questions remain"},
	}
	why := WhyHypothesis(s, "hypothesis-deployment-caused")
	if why.Summary == "" {
		t.Fatal("empty summary")
	}
	not := WhyNotHypothesis(s, "", "hypothesis-unknown")
	if not.Summary == "" {
		t.Fatal("empty not summary")
	}
	if WhyConfidence(s).Summary == "" {
		t.Fatal("confidence explanation")
	}
	if WhyIncomplete(s).Summary == "" {
		t.Fatal("incomplete explanation")
	}
}
