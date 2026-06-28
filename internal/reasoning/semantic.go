package reasoning

import (
	"context"
	"encoding/json"

	"github.com/stackrail/incident-investigator/internal/model"
)

// LLMResponse is structured output from a semantic reasoner backend.
type LLMResponse struct {
	Findings        []model.ReasoningAction `json:"findings"`
	NewQuestions    []model.Question        `json:"new_questions,omitempty"`
	Recommendations []string                `json:"recommendations,omitempty"`
}

// LLM is the provider-agnostic semantic reasoning backend.
type LLM interface {
	Analyze(prompt string) (*LLMResponse, error)
}

// MockLLM returns deterministic structured actions for tests.
type MockLLM struct {
	Response *LLMResponse
	Err      error
	LastPrompt string
}

// NewMockLLM returns a mock with no default response.
func NewMockLLM() *MockLLM { return &MockLLM{} }

// Analyze implements LLM.
func (m *MockLLM) Analyze(prompt string) (*LLMResponse, error) {
	m.LastPrompt = prompt
	if m.Err != nil {
		return nil, m.Err
	}
	if m.Response != nil {
		return m.Response, nil
	}
	return &LLMResponse{}, nil
}

// BuildSemanticPrompt serializes investigation context for an LLM backend.
func BuildSemanticPrompt(inv *Investigation) string {
	type summary struct {
		Question   string              `json:"question"`
		Goal       string              `json:"goal"`
		Evidence   int                 `json:"evidence_count"`
		Hypotheses []model.Hypothesis  `json:"hypotheses"`
		OpenQuestions []model.Question `json:"open_questions,omitempty"`
		GraphNodes int                 `json:"graph_nodes"`
		TraceLen   int                 `json:"reasoning_trace_len"`
	}
	s := summary{
		Question:   inv.Session.Question,
		Goal:       string(inv.Session.Goal),
		Evidence:   len(inv.Session.Evidence),
		Hypotheses: inv.Session.Hypotheses,
		TraceLen:   len(inv.Session.ReasoningTrace),
	}
	if inv.Session.Plan != nil {
		for _, q := range inv.Session.Plan.Questions {
			if q.Status != model.QuestionAnswered && q.Status != model.QuestionRejected {
				s.OpenQuestions = append(s.OpenQuestions, q)
			}
		}
	}
	if inv.Session.Graph != nil {
		s.GraphNodes = len(inv.Session.Graph.Nodes)
	}
	b, _ := json.Marshal(s)
	return string(b)
}

// ParseLLMResponse converts structured LLM output into reasoning actions.
func ParseLLMResponse(reasoner string, resp *LLMResponse) []model.ReasoningAction {
	if resp == nil {
		return nil
	}
	var actions []model.ReasoningAction
	for _, a := range resp.Findings {
		a.Reasoner = reasoner
		actions = append(actions, a)
	}
	for _, q := range resp.NewQuestions {
		qcopy := q
		actions = append(actions, model.ReasoningAction{
			Type: model.ActionCreateQuestion, Reasoner: reasoner, Question: &qcopy,
		})
	}
	for _, rec := range resp.Recommendations {
		actions = append(actions, model.ReasoningAction{
			Type: model.ActionCreateRecommendation, Reasoner: reasoner, Recommendation: rec,
		})
	}
	return actions
}

// SemanticReasoner uses an LLM backend for ambiguous evidence (interface only).
type SemanticReasoner struct {
	llm LLM
}

// NewSemanticReasoner returns a semantic reasoner backed by the given LLM.
func NewSemanticReasoner(llm LLM) *SemanticReasoner {
	return &SemanticReasoner{llm: llm}
}

func (s *SemanticReasoner) Name() string    { return "semantic" }
func (s *SemanticReasoner) Priority() int   { return 50 }
func (s *SemanticReasoner) Supports(_ *model.Session) bool { return s.llm != nil }

func (s *SemanticReasoner) Analyze(ctx context.Context, inv *Investigation) (*model.ReasoningResult, error) {
	_ = ctx
	prompt := BuildSemanticPrompt(inv)
	resp, err := s.llm.Analyze(prompt)
	if err != nil {
		return nil, err
	}
	actions := ParseLLMResponse(s.Name(), resp)
	return &model.ReasoningResult{
		Reasoner:   s.Name(),
		Confidence: inv.Session.Confidence,
		Actions:    actions,
		Findings: []model.Finding{{
			Type: "semantic_analysis", Summary: "Structured semantic reasoning completed.",
		}},
	}, nil
}
