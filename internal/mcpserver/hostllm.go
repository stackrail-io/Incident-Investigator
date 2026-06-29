package mcpserver

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stackrail/incident-investigator/internal/reasoning"
)

type samplingBackend struct {
	session *mcp.ServerSession
}

func (b *samplingBackend) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if b.session == nil {
		return "", reasoning.ErrNoHostLLM
	}
	result, err := b.session.CreateMessage(ctx, &mcp.CreateMessageParams{
		SystemPrompt: systemPrompt,
		Messages: []*mcp.SamplingMessage{{
			Role:    "user",
			Content: &mcp.TextContent{Text: userPrompt},
		}},
		MaxTokens: 2048,
	})
	if err != nil {
		return "", fmt.Errorf("host LLM sampling: %w", err)
	}
	return contentText(result.Content), nil
}

func contentText(c mcp.Content) string {
	switch v := c.(type) {
	case *mcp.TextContent:
		return v.Text
	default:
		return fmt.Sprint(c)
	}
}

func (s *Server) ctxWithHostLLM(ctx context.Context, req *mcp.CallToolRequest) context.Context {
	if req == nil || req.Session == nil {
		return ctx
	}
	if !clientSupportsSampling(req.Session.InitializeParams()) {
		if s.log != nil {
			s.log.Debug("semantic reasoner skipped: client did not advertise sampling capability")
		}
		return ctx
	}
	return reasoning.WithHostLLMBackend(ctx, &samplingBackend{session: req.Session})
}

// clientSupportsSampling reports whether the client advertised the `sampling`
// capability during initialization.
//
// Per the MCP specification a server must not issue sampling/createMessage
// requests to a client that did not declare sampling support. Without this
// guard the semantic reasoner would call back into a client that cannot answer,
// blocking every reasoning tool call (e.g. submit_evidence) until the request
// errors or the connection drops. When sampling is unavailable the backend is
// left unattached and the semantic reasoner skips itself cleanly.
func clientSupportsSampling(params *mcp.InitializeParams) bool {
	return params != nil && params.Capabilities != nil && params.Capabilities.Sampling != nil
}
