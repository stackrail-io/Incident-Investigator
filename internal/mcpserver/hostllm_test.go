package mcpserver

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stackrail/incident-investigator/internal/reasoning"
	"github.com/stackrail/incident-investigator/internal/runtime"
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
	srv := New(runtime.New(), nil)
	ctx := t.Context()
	if got := srv.ctxWithHostLLM(ctx, nil); got != ctx {
		t.Errorf("ctxWithHostLLM(nil request) attached a backend; want unchanged context")
	}
	if got := srv.ctxWithHostLLM(ctx, &mcp.CallToolRequest{}); got != ctx {
		t.Errorf("ctxWithHostLLM(nil session) attached a backend; want unchanged context")
	}
}

// TestCtxWithHostLLMSessionIntegration wires a real MCP session and verifies
// ctxWithHostLLM attaches the backend only when the client advertised sampling.
func TestCtxWithHostLLMSessionIntegration(t *testing.T) {
	t.Run("without_sampling", func(t *testing.T) {
		serverSess := connectServerSession(t, false)
		srv := New(runtime.New(), nil)

		req := &mcp.CallToolRequest{Session: serverSess}
		out := srv.ctxWithHostLLM(context.Background(), req)

		if reasoning.HostLLMBackendFromContext(out) != nil {
			t.Fatal("expected no host LLM backend when client lacks sampling")
		}
		if !clientSupportsSampling(serverSess.InitializeParams()) {
			// expected
		} else {
			t.Fatal("test setup: server session should not have sampling capability")
		}
	})

	t.Run("with_sampling", func(t *testing.T) {
		serverSess := connectServerSession(t, true)
		srv := New(runtime.New(), nil)

		req := &mcp.CallToolRequest{Session: serverSess}
		out := srv.ctxWithHostLLM(context.Background(), req)

		if reasoning.HostLLMBackendFromContext(out) == nil {
			t.Fatal("expected host LLM backend when client advertised sampling")
		}
		if !clientSupportsSampling(serverSess.InitializeParams()) {
			t.Fatal("test setup: server session should have sampling capability")
		}
	})
}

// TestCtxWithHostLLMSkipLogsDebug verifies a debug log is emitted when sampling
// is unavailable on an otherwise valid session.
func TestCtxWithHostLLMSkipLogsDebug(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	srv := New(runtime.New(), log)

	serverSess := connectServerSession(t, false)
	req := &mcp.CallToolRequest{Session: serverSess}
	_ = srv.ctxWithHostLLM(context.Background(), req)

	out := buf.String()
	if !strings.Contains(out, "semantic reasoner skipped") {
		t.Errorf("expected debug log on sampling skip, got: %q", out)
	}
	if !strings.Contains(out, "sampling") {
		t.Errorf("expected sampling mentioned in debug log, got: %q", out)
	}
}

// connectServerSession returns the server's session after a full MCP handshake.
// When withSampling is true the client advertises sampling/createMessage support.
func connectServerSession(t *testing.T, withSampling bool) *mcp.ServerSession {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	clientT, serverT := mcp.NewInMemoryTransports()
	srv := New(runtime.New(), nil)

	serverSess, err := srv.MCP().Connect(ctx, serverT, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	t.Cleanup(func() { _ = serverSess.Close() })

	var clientOpts *mcp.ClientOptions
	if withSampling {
		clientOpts = &mcp.ClientOptions{
			CreateMessageHandler: func(_ context.Context, _ *mcp.CreateMessageRequest) (*mcp.CreateMessageResult, error) {
				return &mcp.CreateMessageResult{
					Role:    "assistant",
					Content: &mcp.TextContent{Text: `{"findings":[],"recommendations":[]}`},
				}, nil
			},
		}
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.1"}, clientOpts)
	clientSess, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = clientSess.Close() })

	return serverSess
}
