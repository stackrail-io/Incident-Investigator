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

func ctxWithHostLLM(ctx context.Context, req *mcp.CallToolRequest) context.Context {
	if req == nil || req.Session == nil {
		return ctx
	}
	return reasoning.WithHostLLMBackend(ctx, &samplingBackend{session: req.Session})
}
