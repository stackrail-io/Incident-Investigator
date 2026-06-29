package mcpserver

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestClientSupportsSampling guards the capability check that prevents the
// server from issuing sampling/createMessage requests to clients that never
// advertised sampling support. Without the guard such requests block every
// reasoning tool call (e.g. submit_evidence) until they error or time out.
func TestClientSupportsSampling(t *testing.T) {
	tests := []struct {
		name   string
		params *mcp.InitializeParams
		want   bool
	}{
		{
			name:   "nil params",
			params: nil,
			want:   false,
		},
		{
			name:   "nil capabilities",
			params: &mcp.InitializeParams{},
			want:   false,
		},
		{
			name:   "capabilities without sampling",
			params: &mcp.InitializeParams{Capabilities: &mcp.ClientCapabilities{}},
			want:   false,
		},
		{
			name: "sampling advertised",
			params: &mcp.InitializeParams{
				Capabilities: &mcp.ClientCapabilities{Sampling: &mcp.SamplingCapabilities{}},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clientSupportsSampling(tt.params); got != tt.want {
				t.Errorf("clientSupportsSampling() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestCtxWithHostLLMNoSession verifies that a missing request or session never
// attaches the host-LLM backend, leaving the semantic reasoner to skip itself.
func TestCtxWithHostLLMNoSession(t *testing.T) {
	ctx := t.Context()
	if got := ctxWithHostLLM(ctx, nil); got != ctx {
		t.Errorf("ctxWithHostLLM(nil request) attached a backend; want unchanged context")
	}
	if got := ctxWithHostLLM(ctx, &mcp.CallToolRequest{}); got != ctx {
		t.Errorf("ctxWithHostLLM(nil session) attached a backend; want unchanged context")
	}
}
