package mcpserver_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stackrail/incident-investigator/internal/mcpserver"
	"github.com/stackrail/incident-investigator/internal/runtime"
)

// TestEndToEndOverMCP drives the full MCP protocol over an in-memory transport:
// start -> submit -> status -> finish, exactly as a real AI assistant would.
func TestEndToEndOverMCP(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rt := runtime.New()
	srv := mcpserver.New(rt, nil)

	clientT, serverT := mcp.NewInMemoryTransports()

	serverSession, err := srv.MCP().Connect(ctx, serverT, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.1"}, nil)
	cs, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer cs.Close()

	// Tools must be advertised.
	tools, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	wantTools := map[string]bool{
		"start_investigation":      false,
		"submit_evidence":          false,
		"get_investigation_status": false,
		"finish_investigation":     false,
	}
	for _, tl := range tools.Tools {
		if _, ok := wantTools[tl.Name]; ok {
			wantTools[tl.Name] = true
		}
	}
	for name, found := range wantTools {
		if !found {
			t.Errorf("tool %q not advertised", name)
		}
	}

	// start_investigation
	var startOut mcpserver.StartOutput
	callInto(t, ctx, cs, "start_investigation", map[string]any{
		"question": "Why did checkout fail yesterday?",
		"service":  "checkout-api",
	}, &startOut)
	if startOut.SessionID == "" {
		t.Fatalf("expected session id")
	}
	if len(startOut.RequiredEvidence) == 0 {
		t.Fatalf("expected required evidence in start response")
	}

	// submit_evidence
	var submitOut mcpserver.SubmitOutput
	callInto(t, ctx, cs, "submit_evidence", map[string]any{
		"session_id": startOut.SessionID,
		"evidence": []map[string]any{
			{
				"timestamp": "2026-06-27T09:01:00Z",
				"category":  "deployment_events",
				"entity":    "checkout-api",
				"summary":   "Deployed checkout-api v2.4.0",
			},
			{
				"timestamp": "2026-06-27T09:05:00Z",
				"category":  "application_logs",
				"entity":    "checkout-api",
				"summary":   "HTTP 500 errors spiking after deploy",
			},
		},
	}, &submitOut)
	if submitOut.Confidence <= 0 {
		t.Errorf("expected positive confidence after submit, got %.2f", submitOut.Confidence)
	}
	if len(submitOut.UpdatedHypotheses) == 0 {
		t.Errorf("expected hypotheses after submit")
	}

	// get_investigation_status
	var statusOut mcpserver.StatusOutput
	callInto(t, ctx, cs, "get_investigation_status", map[string]any{
		"session_id": startOut.SessionID,
	}, &statusOut)
	if statusOut.EvidenceCount != 2 {
		t.Errorf("expected 2 evidence items, got %d", statusOut.EvidenceCount)
	}
	if statusOut.Graph == nil || len(statusOut.Graph.Nodes) == 0 {
		t.Errorf("expected a populated graph in status")
	}

	// finish_investigation
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "finish_investigation",
		Arguments: map[string]any{"session_id": startOut.SessionID},
	})
	if err != nil {
		t.Fatalf("finish call: %v", err)
	}
	if res.IsError {
		t.Fatalf("finish returned tool error: %v", res.Content)
	}

	// A bad session id should surface as a tool error, not a transport error.
	bad, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_investigation_status",
		Arguments: map[string]any{"session_id": "nope"},
	})
	if err != nil {
		t.Fatalf("unexpected protocol error: %v", err)
	}
	if !bad.IsError {
		t.Errorf("expected tool error for unknown session id")
	}
}

// callInto invokes a tool and decodes its structured output into out.
func callInto(t *testing.T, ctx context.Context, cs *mcp.ClientSession, name string, args map[string]any, out any) {
	t.Helper()
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	if res.IsError {
		t.Fatalf("call %s returned tool error: %v", name, res.Content)
	}
	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshal %s output: %v", name, err)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		t.Fatalf("decode %s output: %v", name, err)
	}
}
