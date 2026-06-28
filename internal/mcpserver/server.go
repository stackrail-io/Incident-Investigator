package mcpserver

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stackrail/incident-investigator/internal/model"
	"github.com/stackrail/incident-investigator/internal/runtime"
	"github.com/stackrail/incident-investigator/internal/version"
)

// Server adapts the investigation runtime to the MCP protocol.
type Server struct {
	rt  *runtime.Runtime
	log *slog.Logger
	mcp *mcp.Server
}

// New builds an MCP server backed by the given runtime.
func New(rt *runtime.Runtime, log *slog.Logger) *Server {
	if log == nil {
		log = slog.Default()
	}
	s := &Server{rt: rt, log: log}
	s.mcp = mcp.NewServer(&mcp.Implementation{
		Name:    "incident-investigator",
		Version: version.Version,
	}, nil)
	s.registerTools()
	return s
}

// MCP exposes the underlying mcp.Server (useful for tests and custom transports).
func (s *Server) MCP() *mcp.Server { return s.mcp }

// Run serves the MCP protocol over stdio until the context is cancelled.
func (s *Server) Run(ctx context.Context) error {
	s.log.Info("incident-investigator MCP server starting", "version", version.Version)
	return s.mcp.Run(ctx, &mcp.StdioTransport{})
}

func (s *Server) registerTools() {
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name: "start_investigation",
		Description: "Begin a stateful incident investigation. Returns a session id and the " +
			"evidence the engine wants you to collect first. The engine never connects to any " +
			"external system; you gather evidence and submit it.",
	}, s.handleStart)

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name: "submit_evidence",
		Description: "Submit one or more pieces of vendor-neutral evidence to an investigation. " +
			"Returns updated progress, confidence, competing hypotheses and what evidence to collect next.",
	}, s.handleSubmit)

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "get_investigation_status",
		Description: "Get the current state of an investigation: hypotheses, confidence, graph, timeline, missing evidence and progress.",
	}, s.handleStatus)

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "finish_investigation",
		Description: "Conclude an investigation and generate the final report: timeline, evidence graph, root-cause candidates, contradictions, blast radius, missing evidence, confidence, postmortem and recommendations.",
	}, s.handleFinish)
}

func (s *Server) handleStart(ctx context.Context, _ *mcp.CallToolRequest, in StartInput) (*mcp.CallToolResult, StartOutput, error) {
	if in.Question == "" {
		return nil, StartOutput{}, fmt.Errorf("question is required")
	}
	start, err := parseFlexibleTime(in.TimeWindow.Start)
	if err != nil {
		return nil, StartOutput{}, err
	}
	end, err := parseFlexibleTime(in.TimeWindow.End)
	if err != nil {
		return nil, StartOutput{}, err
	}

	sess, err := s.rt.Start(runtime.StartInput{
		Question:   in.Question,
		Service:    in.Service,
		TimeWindow: model.TimeWindow{Start: start, End: end},
	})
	if err != nil {
		return nil, StartOutput{}, err
	}

	s.log.Info("investigation started", "session_id", sess.ID, "question", in.Question)

	return nil, StartOutput{
		SessionID:        sess.ID,
		Status:           string(sess.Status),
		Progress:         sess.Progress,
		Confidence:       sess.Confidence,
		RequiredEvidence: sess.RequiredEvidence,
		Hypotheses:       sess.Hypotheses,
	}, nil
}

func (s *Server) handleSubmit(ctx context.Context, _ *mcp.CallToolRequest, in SubmitInput) (*mcp.CallToolResult, SubmitOutput, error) {
	if in.SessionID == "" {
		return nil, SubmitOutput{}, fmt.Errorf("session_id is required")
	}
	if len(in.Evidence) == 0 {
		return nil, SubmitOutput{}, fmt.Errorf("at least one evidence item is required")
	}

	evidence := make([]*model.Evidence, 0, len(in.Evidence))
	for _, e := range in.Evidence {
		me, err := e.toModelEvidence()
		if err != nil {
			return nil, SubmitOutput{}, err
		}
		evidence = append(evidence, me)
	}

	sess, err := s.rt.Submit(in.SessionID, evidence)
	if err != nil {
		return nil, SubmitOutput{}, err
	}

	s.log.Info("evidence submitted",
		"session_id", sess.ID,
		"count", len(evidence),
		"progress", sess.Progress,
		"confidence", sess.Confidence,
	)

	return nil, SubmitOutput{
		Progress:             sess.Progress,
		Confidence:           sess.Confidence,
		Status:               string(sess.Status),
		MissingEvidence:      sess.MissingEvidence,
		NextRequiredEvidence: sess.MissingEvidence,
		UpdatedHypotheses:    sess.Hypotheses,
		Contradictions:       sess.Contradictions,
	}, nil
}

func (s *Server) handleStatus(ctx context.Context, _ *mcp.CallToolRequest, in SessionIDInput) (*mcp.CallToolResult, StatusOutput, error) {
	if in.SessionID == "" {
		return nil, StatusOutput{}, fmt.Errorf("session_id is required")
	}
	sess, err := s.rt.Get(in.SessionID)
	if err != nil {
		return nil, StatusOutput{}, err
	}

	return nil, StatusOutput{
		SessionID:       sess.ID,
		Question:        sess.Question,
		Status:          string(sess.Status),
		Progress:        sess.Progress,
		Confidence:      sess.Confidence,
		Hypotheses:      sess.Hypotheses,
		Timeline:        sess.Timeline,
		Graph:           sess.Graph.View(),
		Contradictions:  sess.Contradictions,
		MissingEvidence: sess.MissingEvidence,
		BlastRadius:     sess.BlastRadius,
		EvidenceCount:   len(sess.Evidence),
	}, nil
}

func (s *Server) handleFinish(ctx context.Context, _ *mcp.CallToolRequest, in SessionIDInput) (*mcp.CallToolResult, model.Report, error) {
	if in.SessionID == "" {
		return nil, model.Report{}, fmt.Errorf("session_id is required")
	}
	report, sess, err := s.rt.Finish(in.SessionID)
	if err != nil {
		return nil, model.Report{}, err
	}
	s.log.Info("investigation finished", "session_id", sess.ID, "confidence", report.Confidence)
	return nil, report, nil
}
