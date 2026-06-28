// Command incident-investigator runs the Incident Investigator MCP server.
//
// The server is a stateful investigation runtime. It owns NO infrastructure
// integrations: it does not know what CloudWatch, Datadog or Kubernetes are.
// The AI assistant gathers evidence; this engine reasons over it.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/stackrail/incident-investigator/internal/mcpserver"
	"github.com/stackrail/incident-investigator/internal/runtime"
	"github.com/stackrail/incident-investigator/internal/version"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "-version", "--version":
			fmt.Println(version.Full())
			return
		case "help", "-h", "--help":
			printHelp()
			return
		}
	}

	runServer()
}

func printHelp() {
	fmt.Fprintf(os.Stderr, `Incident Investigator — vendor-neutral MCP investigation engine

Usage:
  incident-investigator              Run the MCP server (stdio)
  incident-investigator version      Print version and exit
  incident-investigator help         Show this help

Install:
  https://github.com/stackrail-io/Incident-Investigator#install

`)
}

func runServer() {
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
