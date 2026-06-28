package reasoning

import (
	"fmt"
	"strings"

	"github.com/stackrail/incident-investigator/internal/graph"
	"github.com/stackrail/incident-investigator/internal/model"
)

// Validator checks whether an action may be applied to the current session.
type Validator struct{}

// NewValidator returns the default action validator.
func NewValidator() *Validator { return &Validator{} }

// Validate returns nil if the action is acceptable, or an error explaining rejection.
func (v *Validator) Validate(s *model.Session, g *graph.InvestigationGraph, a model.ReasoningAction) error {
	switch a.Type {
	case model.ActionResolveQuestion, model.ActionRejectQuestion:
		if a.QuestionID == "" {
			return fmt.Errorf("question id required")
		}
		if s.Plan == nil {
			return fmt.Errorf("no investigation plan")
		}
		q := findQuestion(s.Plan, a.QuestionID)
		if q == nil {
			return fmt.Errorf("unknown question %q", a.QuestionID)
		}
		if q.Status == model.QuestionAnswered || q.Status == model.QuestionRejected {
			return fmt.Errorf("question %q already resolved", a.QuestionID)
		}
	case model.ActionIncreaseHypothesisConfidence, model.ActionDecreaseHypothesisConfidence:
		if a.HypothesisID == "" {
			return fmt.Errorf("hypothesis id required")
		}
		if a.Delta <= 0 {
			return fmt.Errorf("delta must be positive")
		}
		if a.Delta > 50 {
			return fmt.Errorf("delta exceeds limit (50)")
		}
	case model.ActionCreateHypothesis:
		if a.Hypothesis == nil || a.Hypothesis.ID == "" {
			return fmt.Errorf("hypothesis payload required")
		}
	case model.ActionRejectHypothesis, model.ActionPromoteHypothesis:
		if a.HypothesisID == "" {
			return fmt.Errorf("hypothesis id required")
		}
	case model.ActionLinkGraphNodes:
		if a.GraphEdge == nil {
			return fmt.Errorf("graph edge required")
		}
		if a.GraphEdge.From == "" || a.GraphEdge.To == "" {
			return fmt.Errorf("edge endpoints required")
		}
		if g != nil {
			if !g.HasNode(a.GraphEdge.From) {
				return fmt.Errorf("invalid from node %q", a.GraphEdge.From)
			}
			if !g.HasNode(a.GraphEdge.To) {
				return fmt.Errorf("invalid to node %q", a.GraphEdge.To)
			}
		}
	case model.ActionMarkContradiction:
		if a.Contradiction == nil || a.Contradiction.ID == "" {
			return fmt.Errorf("contradiction payload required")
		}
	case model.ActionCreateQuestion:
		if a.Question == nil || a.Question.ID == "" {
			return fmt.Errorf("question payload required")
		}
	case model.ActionCreateEvidenceRequest:
		if a.EvidenceRequest == nil || a.EvidenceRequest.ID == "" {
			return fmt.Errorf("evidence request payload required")
		}
	case model.ActionCreateRecommendation:
		if strings.TrimSpace(a.Recommendation) == "" {
			return fmt.Errorf("recommendation text required")
		}
	}
	return nil
}

func findQuestion(plan *model.InvestigationPlan, id string) *model.Question {
	for i := range plan.Questions {
		if plan.Questions[i].ID == id {
			return &plan.Questions[i]
		}
	}
	return nil
}
