package reasoning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ErrNoHostLLM indicates the MCP host did not provide a sampling backend.
var ErrNoHostLLM = errors.New("host LLM sampling unavailable")

// HostLLMBackend requests completions from the MCP client (Claude, Codex, etc.).
type HostLLMBackend interface {
	Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

type hostLLMKey struct{}

// WithHostLLMBackend attaches a host LLM backend to the context for semantic reasoning.
func WithHostLLMBackend(ctx context.Context, backend HostLLMBackend) context.Context {
	if backend == nil {
		return ctx
	}
	return context.WithValue(ctx, hostLLMKey{}, backend)
}

// HostLLMBackendFromContext returns the host LLM backend injected by the MCP layer.
func HostLLMBackendFromContext(ctx context.Context) HostLLMBackend {
	if ctx == nil {
		return nil
	}
	b, _ := ctx.Value(hostLLMKey{}).(HostLLMBackend)
	return b
}

// HostLLM uses MCP sampling (client-side LLM) for semantic reasoning.
type HostLLM struct {
	MaxTokens int64
}

// NewHostLLM returns a semantic backend that delegates to the MCP host.
func NewHostLLM() *HostLLM {
	return &HostLLM{MaxTokens: 2048}
}

// Analyze implements LLM.
func (h *HostLLM) Analyze(ctx context.Context, prompt string) (*LLMResponse, error) {
	backend := HostLLMBackendFromContext(ctx)
	if backend == nil {
		return &LLMResponse{}, nil
	}
	raw, err := backend.Complete(ctx, semanticSystemPrompt(), prompt)
	if err != nil {
		return nil, err
	}
	return parseLLMResponseJSON(raw)
}

func semanticSystemPrompt() string {
	return `You are the semantic reasoner for an incident investigation engine.
Analyze the investigation context and return ONLY valid JSON matching this schema:
{
  "findings": [
    {
      "type": "IncreaseHypothesisConfidence|DecreaseHypothesisConfidence|CreateRecommendation",
      "hypothesis_id": "optional hypothesis id",
      "delta": 0,
      "reason": "why",
      "recommendation": "optional text for CreateRecommendation"
    }
  ],
  "new_questions": [
    {"id": "q-unique", "text": "question", "priority": "high|medium|low"}
  ],
  "recommendations": ["optional plain-text recommendations"]
}
Use hypothesis ids from the input when adjusting confidence.
Propose at most 3 findings, 2 questions, and 3 recommendations.
Do not wrap JSON in markdown.`
}

func parseLLMResponseJSON(raw string) (*LLMResponse, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return &LLMResponse{}, nil
	}
	if i := strings.Index(raw, "{"); i > 0 {
		raw = raw[i:]
	}
	if j := strings.LastIndex(raw, "}"); j >= 0 && j < len(raw)-1 {
		raw = raw[:j+1]
	}
	var resp LLMResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, fmt.Errorf("parse host LLM JSON: %w", err)
	}
	return &resp, nil
}
