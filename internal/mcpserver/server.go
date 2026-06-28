package mcpserver

import (
	"context"
	"log/slog"
	"strings"

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
		Description: "Conclude an investigation and generate the final report: timeline, investigation graph, root-cause candidates, contradictions, blast radius, missing evidence, confidence, postmortem and recommendations.",
	}, s.handleFinish)

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name: "explain_reasoning",
		Description: "Return the full reasoning snapshot for debugging: hypotheses, reasoning trace, " +
			"supporting evidence, contradictions, coverage, confidence breakdown, blocking questions, " +
			"missing evidence, strategy, journal, and metrics.",
	}, s.handleExplain)

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name: "explain_investigation",
		Description: "Primary debugging endpoint for the investigation protocol: plan, questions, " +
			"question graph, open/resolved questions, evidence requests, stage, reasoning trace, and metrics.",
	}, s.handleExplainInvestigation)

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "get_investigation_plan",
		Description: "Return the complete investigation plan including questions, evidence requests, and stage.",
	}, s.handleGetPlan)

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "resolve_question",
		Description: "Explicitly resolve a protocol question when evidence is conclusive.",
	}, s.handleResolveQuestion)

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "list_open_questions",
		Description: "Return unresolved investigation questions sorted by priority.",
	}, s.handleListOpenQuestions)

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "get_graph",
		Description: "Return the full investigation graph: all nodes and typed relationships.",
	}, s.handleGetGraph)

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name: "query_graph",
		Description: "Run a graph query: upstream, downstream, supporting_evidence, contradictions, " +
			"unanswered_questions, service_evidence, blast_radius, shortest_causal_path (target from->to), strongest_path.",
	}, s.handleQueryGraph)

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "get_subgraph",
		Description: "Extract a filtered subgraph by node_type or explicit node_ids.",
	}, s.handleGetSubgraph)

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "explain_path",
		Description: "Explain a causal path between two nodes with supporting evidence per hop.",
	}, s.handleExplainPath)

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name: "get_reasoning_cycles",
		Description: "Return reasoning cycle history for replay: reasoners run, actions proposed, " +
			"applied vs rejected, execution time, and confidence per cycle.",
	}, s.handleGetReasoningCycles)

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name: "find_similar_investigations",
		Description: "Find archived investigations similar to the current session (historical intelligence).",
	}, s.handleFindSimilarInvestigations)

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name:        "suggest_patterns",
		Description: "Suggest recurring investigation patterns from historical data.",
	}, s.handleSuggestPatterns)

	mcp.AddTool(s.mcp, &mcp.Tool{
		Name: "calibrate_confidence",
		Description: "Calibrate hypothesis confidence using historical investigations. Returns original " +
			"and adjusted confidence with supporting history.",
	}, s.handleCalibrateConfidence)
}

func (s *Server) handleStart(ctx context.Context, _ *mcp.CallToolRequest, in StartInput) (*mcp.CallToolResult, StartOutput, error) {
	if strings.TrimSpace(in.Question) == "" {
		return nil, StartOutput{}, toolErr("question is required")
	}
	start, err := parseFlexibleTime(in.TimeWindow.Start)
	if err != nil {
		return nil, StartOutput{}, toolErr("%s", err.Error())
	}
	end, err := parseFlexibleTime(in.TimeWindow.End)
	if err != nil {
		return nil, StartOutput{}, toolErr("%s", err.Error())
	}
	tw := model.TimeWindow{Start: start, End: end}
	if !start.IsZero() && !end.IsZero() && end.Before(start) {
		return nil, StartOutput{}, toolErr("time_window end must not be before start")
	}

	sess, err := s.rt.Start(runtime.StartInput{
		Question:   in.Question,
		Service:    in.Service,
		TimeWindow: tw,
		Goal:       model.InvestigationGoal(in.Goal),
	})
	if err != nil {
		return nil, StartOutput{}, mapRuntimeError(err)
	}

	s.log.Info("investigation started", "session_id", sess.ID, "question", in.Question)

	return nil, StartOutput{
		SessionID:        sess.ID,
		Status:           string(sess.Status),
		State:            string(sess.State),
		Goal:             string(sess.Goal),
		Stage:            planStage(sess.Plan),
		Plan:             sess.Plan,
		Questions:        planQuestions(sess.Plan),
		EvidenceRequests: planEvidenceRequests(sess.Plan),
		Progress:         sess.Progress,
		Confidence:       sess.Confidence,
		RequiredEvidence: sess.RequiredEvidence,
		Strategy:         sess.Strategy,
		Hypotheses:       sess.Hypotheses,
	}, nil
}

func (s *Server) handleSubmit(ctx context.Context, _ *mcp.CallToolRequest, in SubmitInput) (*mcp.CallToolResult, SubmitOutput, error) {
	if in.SessionID == "" {
		return nil, SubmitOutput{}, toolErr("session_id is required")
	}
	if len(in.Evidence) == 0 {
		return nil, SubmitOutput{}, toolErr("at least one evidence item is required")
	}

	evidence := make([]*model.Evidence, 0, len(in.Evidence))
	for _, e := range in.Evidence {
		me, err := e.toModelEvidence()
		if err != nil {
			return nil, SubmitOutput{}, toolErr("%s", err.Error())
		}
		evidence = append(evidence, me)
	}

	sess, err := s.rt.Submit(in.SessionID, evidence)
	if err != nil {
		return nil, SubmitOutput{}, mapRuntimeError(err)
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
		ConfidenceDelta:      sess.LastTurn.ConfidenceDelta,
		Status:               string(sess.Status),
		State:                string(sess.State),
		Stage:                planStage(sess.Plan),
		Plan:                 sess.Plan,
		ResolvedQuestions:    sess.LastTurn.ResolvedQuestions,
		NewQuestions:         sess.LastTurn.NewQuestions,
		EvidenceRequests:     planEvidenceRequests(sess.Plan),
		MissingEvidence:      sess.MissingEvidence,
		NextRequiredEvidence: sess.MissingEvidence,
		Strategy:             sess.Strategy,
		UpdatedHypotheses:    sess.Hypotheses,
		Contradictions:       sess.Contradictions,
		Sufficiency:          sess.Sufficiency,
	}, nil
}

func (s *Server) handleStatus(ctx context.Context, _ *mcp.CallToolRequest, in SessionIDInput) (*mcp.CallToolResult, StatusOutput, error) {
	if in.SessionID == "" {
		return nil, StatusOutput{}, toolErr("session_id is required")
	}
	sess, err := s.rt.Get(in.SessionID)
	if err != nil {
		return nil, StatusOutput{}, mapRuntimeError(err)
	}

	return nil, StatusOutput{
		SessionID:             sess.ID,
		Question:              sess.Question,
		Status:                string(sess.Status),
		State:                 string(sess.State),
		Goal:                  string(sess.Goal),
		Stage:                 planStage(sess.Plan),
		Plan:                  sess.Plan,
		QuestionGraph:         sess.QuestionGraph,
		Questions:             planQuestions(sess.Plan),
		EvidenceRequests:      planEvidenceRequests(sess.Plan),
		ResolutionHistory:     resolutionHistory(sess.Plan),
		ProtocolMetrics:       sess.ProtocolMetrics,
		Progress:              sess.Progress,
		InvestigationProgress: sess.InvestigationProgress,
		Confidence:            sess.Confidence,
		ConfidenceBreakdown:   sess.ConfidenceBreakdown,
		Hypotheses:            sess.Hypotheses,
		Timeline:              sess.Timeline,
		Graph:                 sess.Graph,
		Contradictions:        sess.Contradictions,
		MissingEvidence:       sess.MissingEvidence,
		BlockingQuestions:     sess.Sufficiency.BlockingQuestions,
		Strategy:              sess.Strategy,
		Coverage:              sess.Coverage,
		ReasoningTrace:        sess.ReasoningTrace,
		Journal:               sess.Journal,
		Metrics:               sess.Metrics,
		BlastRadius:           sess.BlastRadius,
		EvidenceCount:         len(sess.Evidence),
		Sufficiency:           sess.Sufficiency,
	}, nil
}

func (s *Server) handleExplainInvestigation(ctx context.Context, _ *mcp.CallToolRequest, in SessionIDInput) (*mcp.CallToolResult, runtime.ExplainInvestigationOutput, error) {
	if in.SessionID == "" {
		return nil, runtime.ExplainInvestigationOutput{}, toolErr("session_id is required")
	}
	out, err := s.rt.ExplainInvestigation(in.SessionID)
	if err != nil {
		return nil, runtime.ExplainInvestigationOutput{}, mapRuntimeError(err)
	}
	return nil, *out, nil
}

func (s *Server) handleGetPlan(ctx context.Context, _ *mcp.CallToolRequest, in SessionIDInput) (*mcp.CallToolResult, PlanOutput, error) {
	if in.SessionID == "" {
		return nil, PlanOutput{}, toolErr("session_id is required")
	}
	plan, err := s.rt.GetPlan(in.SessionID)
	if err != nil {
		return nil, PlanOutput{}, mapRuntimeError(err)
	}
	return nil, PlanOutput{Plan: plan}, nil
}

func (s *Server) handleResolveQuestion(ctx context.Context, _ *mcp.CallToolRequest, in ResolveQuestionInput) (*mcp.CallToolResult, ResolveQuestionOutput, error) {
	if in.SessionID == "" {
		return nil, ResolveQuestionOutput{}, toolErr("session_id is required")
	}
	if in.QuestionID == "" {
		return nil, ResolveQuestionOutput{}, toolErr("question_id is required")
	}
	res, sess, err := s.rt.ResolveQuestion(runtime.ResolveQuestionInput{
		SessionID:  in.SessionID,
		QuestionID: in.QuestionID,
		Confirmed:  in.Confirmed,
		Reason:     in.Reason,
	})
	if err != nil {
		return nil, ResolveQuestionOutput{}, mapRuntimeError(err)
	}
	return nil, ResolveQuestionOutput{
		Resolution: res,
		Plan:       sess.Plan,
		Confidence: sess.Confidence,
	}, nil
}

func (s *Server) handleListOpenQuestions(ctx context.Context, _ *mcp.CallToolRequest, in SessionIDInput) (*mcp.CallToolResult, OpenQuestionsOutput, error) {
	if in.SessionID == "" {
		return nil, OpenQuestionsOutput{}, toolErr("session_id is required")
	}
	questions, err := s.rt.ListOpenQuestions(in.SessionID)
	if err != nil {
		return nil, OpenQuestionsOutput{}, mapRuntimeError(err)
	}
	return nil, OpenQuestionsOutput{Questions: questions}, nil
}

func (s *Server) handleGetGraph(ctx context.Context, _ *mcp.CallToolRequest, in SessionIDInput) (*mcp.CallToolResult, GraphOutput, error) {
	if in.SessionID == "" {
		return nil, GraphOutput{}, toolErr("session_id is required")
	}
	g, err := s.rt.GetGraph(in.SessionID)
	if err != nil {
		return nil, GraphOutput{}, mapRuntimeError(err)
	}
	return nil, GraphOutput{Graph: g}, nil
}

func (s *Server) handleQueryGraph(ctx context.Context, _ *mcp.CallToolRequest, in QueryGraphInput) (*mcp.CallToolResult, QueryGraphOutput, error) {
	if in.SessionID == "" {
		return nil, QueryGraphOutput{}, toolErr("session_id is required")
	}
	if in.Kind == "" {
		return nil, QueryGraphOutput{}, toolErr("kind is required")
	}
	sg, err := s.rt.QueryGraph(in.SessionID, runtime.GraphQueryInput{
		Kind:   model.GraphQueryKind(in.Kind),
		Target: in.Target,
		Limit:  in.Limit,
	})
	if err != nil {
		return nil, QueryGraphOutput{}, mapRuntimeError(err)
	}
	return nil, QueryGraphOutput{Subgraph: sg}, nil
}

func (s *Server) handleGetSubgraph(ctx context.Context, _ *mcp.CallToolRequest, in GetSubgraphInput) (*mcp.CallToolResult, GetSubgraphOutput, error) {
	if in.SessionID == "" {
		return nil, GetSubgraphOutput{}, toolErr("session_id is required")
	}
	sg, err := s.rt.GetSubgraph(in.SessionID, runtime.SubgraphInput{
		Name:     in.Name,
		NodeType: in.NodeType,
		NodeIDs:  in.NodeIDs,
	})
	if err != nil {
		return nil, GetSubgraphOutput{}, mapRuntimeError(err)
	}
	return nil, GetSubgraphOutput{Subgraph: sg}, nil
}

func (s *Server) handleExplainPath(ctx context.Context, _ *mcp.CallToolRequest, in ExplainPathInput) (*mcp.CallToolResult, ExplainPathOutput, error) {
	if in.SessionID == "" {
		return nil, ExplainPathOutput{}, toolErr("session_id is required")
	}
	if in.From == "" || in.To == "" {
		return nil, ExplainPathOutput{}, toolErr("from and to are required")
	}
	ex, err := s.rt.ExplainPath(in.SessionID, in.From, in.To)
	if err != nil {
		return nil, ExplainPathOutput{}, mapRuntimeError(err)
	}
	return nil, ExplainPathOutput{Explanation: ex}, nil
}

func (s *Server) handleGetReasoningCycles(ctx context.Context, _ *mcp.CallToolRequest, in SessionIDInput) (*mcp.CallToolResult, ReasoningCyclesOutput, error) {
	if in.SessionID == "" {
		return nil, ReasoningCyclesOutput{}, toolErr("session_id is required")
	}
	cycles, err := s.rt.GetReasoningCycles(in.SessionID)
	if err != nil {
		return nil, ReasoningCyclesOutput{}, mapRuntimeError(err)
	}
	return nil, ReasoningCyclesOutput{Cycles: cycles}, nil
}

func (s *Server) handleFindSimilarInvestigations(ctx context.Context, _ *mcp.CallToolRequest, in SimilarInvestigationsInput) (*mcp.CallToolResult, SimilarInvestigationsOutput, error) {
	if in.SessionID == "" {
		return nil, SimilarInvestigationsOutput{}, toolErr("session_id is required")
	}
	resp, err := s.rt.FindSimilarInvestigations(ctx, in.SessionID, in.Limit)
	if err != nil {
		return nil, SimilarInvestigationsOutput{}, mapRuntimeError(err)
	}
	return nil, SimilarInvestigationsOutput{
		Matches:         resp.Matches,
		Lessons:         resp.Lessons,
		Recommendations: resp.Recommendations,
	}, nil
}

func (s *Server) handleSuggestPatterns(ctx context.Context, _ *mcp.CallToolRequest, in SuggestPatternsInput) (*mcp.CallToolResult, SuggestPatternsOutput, error) {
	if in.SessionID == "" {
		return nil, SuggestPatternsOutput{}, toolErr("session_id is required")
	}
	resp, err := s.rt.SuggestPatterns(ctx, in.SessionID, in.Limit)
	if err != nil {
		return nil, SuggestPatternsOutput{}, mapRuntimeError(err)
	}
	return nil, SuggestPatternsOutput{Patterns: resp.Patterns}, nil
}

func (s *Server) handleCalibrateConfidence(ctx context.Context, _ *mcp.CallToolRequest, in CalibrateConfidenceInput) (*mcp.CallToolResult, CalibrateConfidenceOutput, error) {
	if in.SessionID == "" {
		return nil, CalibrateConfidenceOutput{}, toolErr("session_id is required")
	}
	resp, err := s.rt.CalibrateConfidenceForSession(ctx, in.SessionID, in.HypothesisID)
	if err != nil {
		return nil, CalibrateConfidenceOutput{}, mapRuntimeError(err)
	}
	return nil, CalibrateConfidenceOutput{
		OriginalConfidence:   resp.OriginalConfidence,
		CalibratedConfidence: resp.CalibratedConfidence,
		Delta:                resp.Delta,
		Reason:               resp.Reason,
		HistoricalSampleSize: resp.HistoricalSampleSize,
		HypothesisID:         resp.HypothesisID,
		Explanation:          resp.Explanation,
	}, nil
}

func planStage(plan *model.InvestigationPlan) string {
	if plan == nil {
		return string(model.StagePlanning)
	}
	return string(plan.CurrentStage)
}

func planQuestions(plan *model.InvestigationPlan) []model.Question {
	if plan == nil {
		return []model.Question{}
	}
	return plan.Questions
}

func planEvidenceRequests(plan *model.InvestigationPlan) []model.ProtocolEvidenceRequest {
	if plan == nil {
		return []model.ProtocolEvidenceRequest{}
	}
	return plan.EvidenceRequests
}

func resolutionHistory(plan *model.InvestigationPlan) []model.QuestionResolution {
	if plan == nil {
		return []model.QuestionResolution{}
	}
	return plan.ResolutionHistory
}

func (s *Server) handleExplain(ctx context.Context, _ *mcp.CallToolRequest, in SessionIDInput) (*mcp.CallToolResult, runtime.ExplainOutput, error) {
	if in.SessionID == "" {
		return nil, runtime.ExplainOutput{}, toolErr("session_id is required")
	}
	out, err := s.rt.Explain(in.SessionID)
	if err != nil {
		return nil, runtime.ExplainOutput{}, mapRuntimeError(err)
	}
	return nil, *out, nil
}

func (s *Server) handleFinish(ctx context.Context, _ *mcp.CallToolRequest, in SessionIDInput) (*mcp.CallToolResult, model.Report, error) {
	if in.SessionID == "" {
		return nil, model.Report{}, toolErr("session_id is required")
	}
	report, sess, err := s.rt.Finish(in.SessionID)
	if err != nil {
		return nil, model.Report{}, mapRuntimeError(err)
	}
	s.log.Info("investigation finished", "session_id", sess.ID, "confidence", report.Confidence)
	return nil, report, nil
}
