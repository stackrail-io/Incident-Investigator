// Command incident-investigator runs the Incident Investigator MCP server.
//
// The server is a stateful investigation runtime. It owns NO infrastructure
// integrations: it does not know what CloudWatch, Datadog or Kubernetes are.
// The AI assistant gathers evidence; this engine reasons over it.
package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/stackrail/incident-investigator/internal/mcpserver"
	"github.com/stackrail/incident-investigator/internal/runtime"
)

func main() {
	// IMPORTANT: stdout is reserved for the MCP protocol over stdio, so all logs
	// go to stderr.
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	rt := runtime.New()
	srv := mcpserver.New(rt, logger)

	// A closed stdin (client disconnect) or a cancelled context are both normal
	// ways for the server to stop, not errors.
	switch err := srv.Run(ctx); {
	case err == nil, errors.Is(err, io.EOF), ctx.Err() != nil:
		logger.Info("incident-investigator MCP server stopped")
	default:
		logger.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}
