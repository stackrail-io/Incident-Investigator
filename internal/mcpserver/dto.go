package mcpserver

import (
	"fmt"
	"strings"
	"time"

	"github.com/stackrail/incident-investigator/internal/model"
)

// timeWindowInput is the MCP-facing representation of an investigation window.
type timeWindowInput struct {
	Start string `json:"start,omitempty" jsonschema:"Start of the time window (RFC3339, e.g. 2026-01-01T09:00:00Z)"`
	End   string `json:"end,omitempty" jsonschema:"End of the time window (RFC3339)"`
}

// StartInput is the input to start_investigation.
type StartInput struct {
	Question   string          `json:"question" jsonschema:"The incident question to investigate, e.g. 'Why did checkout fail yesterday?'"`
	Service    string          `json:"service,omitempty" jsonschema:"Primary service under investigation, e.g. 'checkout-api'"`
	TimeWindow timeWindowInput `json:"time_window,omitempty" jsonschema:"Optional time window to scope the investigation"`
	Goal       string          `json:"goal,omitempty" jsonschema:"Investigation goal: root_cause, timeline, blast_radius, deployment_verification, performance_regression, availability, custom"`
}

// StartOutput is the response from start_investigation.
type StartOutput struct {
	SessionID        string                         `json:"session_id"`
	Status           string                         `json:"status"`
	State            string                         `json:"state"`
	Goal             string                         `json:"goal"`
	Stage            string                         `json:"stage"`
	Plan             *model.InvestigationPlan       `json:"plan"`
	Questions        []model.Question               `json:"questions"`
	EvidenceRequests []model.ProtocolEvidenceRequest `json:"evidence_requests"`
	Progress         float64                        `json:"progress"`
	Confidence       float64                        `json:"confidence"`
	RequiredEvidence []model.EvidenceRequest        `json:"required_evidence"`
	Strategy         []model.NextStep               `json:"strategy"`
	Hypotheses       []model.Hypothesis             `json:"hypotheses"`
}

// EvidenceInput is a single piece of vendor-neutral evidence supplied by the
// assistant.
type EvidenceInput struct {
	ID        string         `json:"id,omitempty" jsonschema:"Optional client id; generated if omitted"`
	Timestamp string         `json:"timestamp,omitempty" jsonschema:"RFC3339 timestamp of the observation"`
	Category  string         `json:"category" jsonschema:"One of: application_logs, infrastructure_events, deployment_events, alert_events, metrics, trace_events, configuration_changes, network_events, database_events, security_events, human_context, custom"`
	Source    string         `json:"source,omitempty" jsonschema:"Where the assistant gathered this from"`
	Entity    string         `json:"entity,omitempty" jsonschema:"The service/host/resource this concerns"`
	Summary   string         `json:"summary" jsonschema:"One-line human-readable summary of the observation"`
	Payload   map[string]any `json:"payload,omitempty" jsonschema:"Arbitrary structured detail; the engine never depends on vendor schemas"`
}

// SubmitInput is the input to submit_evidence.
type SubmitInput struct {
	SessionID string          `json:"session_id" jsonschema:"The investigation session id"`
	Evidence  []EvidenceInput `json:"evidence" jsonschema:"One or more pieces of evidence to add"`
}

// SubmitOutput is the response from submit_evidence.
type SubmitOutput struct {
	Progress             float64                         `json:"progress"`
	Confidence           float64                         `json:"confidence"`
	ConfidenceDelta      float64                         `json:"confidence_delta"`
	Status               string                          `json:"status"`
	State                string                          `json:"state"`
	Stage                string                          `json:"stage"`
	Plan                 *model.InvestigationPlan        `json:"plan"`
	ResolvedQuestions    []model.QuestionResolution      `json:"resolved_questions"`
	NewQuestions         []model.Question                `json:"new_questions"`
	EvidenceRequests     []model.ProtocolEvidenceRequest `json:"evidence_requests"`
	MissingEvidence      []model.EvidenceRequest         `json:"missing_evidence"`
	NextRequiredEvidence []model.EvidenceRequest         `json:"next_required_evidence"`
	Strategy             []model.NextStep                `json:"strategy"`
	UpdatedHypotheses    []model.Hypothesis              `json:"updated_hypotheses"`
	Contradictions       []model.Contradiction           `json:"contradictions"`
	Sufficiency          model.SufficiencyReport         `json:"sufficiency"`
}

// SessionIDInput is shared by status and finish tools.
type SessionIDInput struct {
	SessionID string `json:"session_id" jsonschema:"The investigation session id"`
}

// StatusOutput is the response from get_investigation_status.
type StatusOutput struct {
	SessionID             string                          `json:"session_id"`
	Question              string                          `json:"question"`
	Status                string                          `json:"status"`
	State                 string                          `json:"state"`
	Goal                  string                          `json:"goal"`
	Stage                 string                          `json:"stage"`
	Plan                  *model.InvestigationPlan        `json:"plan"`
	QuestionGraph         model.QuestionGraph             `json:"question_graph"`
	Questions             []model.Question                `json:"questions"`
	EvidenceRequests      []model.ProtocolEvidenceRequest `json:"evidence_requests"`
	ResolutionHistory     []model.QuestionResolution      `json:"resolution_history"`
	ProtocolMetrics       model.ProtocolMetrics           `json:"protocol_metrics"`
	Progress              float64                         `json:"progress"`
	InvestigationProgress model.InvestigationProgress     `json:"investigation_progress"`
	Confidence            float64                         `json:"confidence"`
	ConfidenceBreakdown   model.ConfidenceBreakdown       `json:"confidence_breakdown"`
	Hypotheses            []model.Hypothesis              `json:"hypotheses"`
	Timeline              model.Timeline                  `json:"timeline"`
	Graph                 *model.GraphView                `json:"graph"`
	Contradictions        []model.Contradiction           `json:"contradictions"`
	MissingEvidence       []model.EvidenceRequest         `json:"missing_evidence"`
	BlockingQuestions     []model.BlockingQuestion        `json:"blocking_questions"`
	Strategy              []model.NextStep                `json:"strategy"`
	Coverage              model.CoverageReport            `json:"coverage"`
	ReasoningTrace        []model.ReasoningTrace          `json:"reasoning_trace"`
	Journal               []model.JournalEntry            `json:"journal"`
	Metrics               model.ReasoningMetrics          `json:"metrics"`
	BlastRadius           model.BlastRadius               `json:"blast_radius"`
	EvidenceCount         int                             `json:"evidence_count"`
	Sufficiency           model.SufficiencyReport         `json:"sufficiency"`
}

// ResolveQuestionInput is the input to resolve_question.
type ResolveQuestionInput struct {
	SessionID  string `json:"session_id" jsonschema:"The investigation session id"`
	QuestionID string `json:"question_id" jsonschema:"The protocol question id to resolve"`
	Confirmed  bool   `json:"confirmed" jsonschema:"True if the question is confirmed, false if rejected"`
	Reason     string `json:"reason,omitempty" jsonschema:"Explanation for the resolution"`
}

// ResolveQuestionOutput is the response from resolve_question.
type ResolveQuestionOutput struct {
	Resolution *model.QuestionResolution `json:"resolution"`
	Plan       *model.InvestigationPlan  `json:"plan"`
	Confidence float64                   `json:"confidence"`
}

// PlanOutput is the response from get_investigation_plan.
type PlanOutput struct {
	Plan *model.InvestigationPlan `json:"plan"`
}

// OpenQuestionsOutput is the response from list_open_questions.
type OpenQuestionsOutput struct {
	Questions []model.Question `json:"questions"`
}

// GraphOutput is the response from get_graph.
type GraphOutput struct {
	Graph *model.GraphView `json:"graph"`
}

// QueryGraphInput is the input to query_graph.
type QueryGraphInput struct {
	SessionID string `json:"session_id" jsonschema:"The investigation session id"`
	Kind      string `json:"kind" jsonschema:"Query kind: upstream, downstream, supporting_evidence, contradictions, unanswered_questions, service_evidence, blast_radius, shortest_causal_path, strongest_path"`
	Target    string `json:"target,omitempty" jsonschema:"Query target (node id, service name, or from->to for shortest_causal_path)"`
	Limit     int    `json:"limit,omitempty" jsonschema:"Maximum nodes to return"`
}

// QueryGraphOutput is the response from query_graph.
type QueryGraphOutput struct {
	Subgraph *model.Subgraph `json:"subgraph"`
}

// GetSubgraphInput is the input to get_subgraph.
type GetSubgraphInput struct {
	SessionID string   `json:"session_id" jsonschema:"The investigation session id"`
	Name      string   `json:"name,omitempty" jsonschema:"Optional subgraph name"`
	NodeType  string   `json:"node_type,omitempty" jsonschema:"Filter by node type, e.g. evidence, hypothesis, service"`
	NodeIDs   []string `json:"node_ids,omitempty" jsonschema:"Explicit node ids to include"`
}

// GetSubgraphOutput is the response from get_subgraph.
type GetSubgraphOutput struct {
	Subgraph *model.Subgraph `json:"subgraph"`
}

// ExplainPathInput is the input to explain_path.
type ExplainPathInput struct {
	SessionID string `json:"session_id" jsonschema:"The investigation session id"`
	From      string `json:"from" jsonschema:"Start node id or label"`
	To        string `json:"to" jsonschema:"End node id or label"`
}

// ExplainPathOutput is the response from explain_path.
type ExplainPathOutput struct {
	Explanation *model.PathExplanation `json:"explanation"`
}

// ReasoningCyclesOutput is the response from get_reasoning_cycles.
type ReasoningCyclesOutput struct {
	Cycles []model.ReasoningCycle `json:"cycles"`
}

// SimilarInvestigationsInput is the input to find_similar_investigations.
type SimilarInvestigationsInput struct {
	SessionID string `json:"session_id" jsonschema:"The investigation session id"`
	Limit     int    `json:"limit,omitempty" jsonschema:"Maximum matches to return"`
}

// SimilarInvestigationsOutput is the response from find_similar_investigations.
type SimilarInvestigationsOutput struct {
	Matches         []model.SimilarInvestigation          `json:"matches"`
	Lessons         []model.LessonLearned                 `json:"lessons_learned,omitempty"`
	Recommendations *model.InvestigationRecommendations   `json:"recommendations,omitempty"`
}

// SuggestPatternsInput is the input to suggest_patterns.
type SuggestPatternsInput struct {
	SessionID string `json:"session_id" jsonschema:"The investigation session id"`
	Limit     int    `json:"limit,omitempty" jsonschema:"Maximum patterns to return"`
}

// SuggestPatternsOutput is the response from suggest_patterns.
type SuggestPatternsOutput struct {
	Patterns []model.SuggestedPattern `json:"patterns"`
}

// CalibrateConfidenceInput is the input to calibrate_confidence.
type CalibrateConfidenceInput struct {
	SessionID    string `json:"session_id" jsonschema:"The investigation session id"`
	HypothesisID string `json:"hypothesis_id,omitempty" jsonschema:"Hypothesis to calibrate; defaults to leading hypothesis"`
}

// CalibrateConfidenceOutput is the response from calibrate_confidence.
type CalibrateConfidenceOutput struct {
	OriginalConfidence   float64                       `json:"original_confidence"`
	CalibratedConfidence float64                       `json:"calibrated_confidence"`
	Delta                float64                       `json:"delta"`
	Reason               string                        `json:"reason"`
	HistoricalSampleSize int                           `json:"historical_sample_size"`
	HypothesisID         string                        `json:"hypothesis_id,omitempty"`
	Explanation          *model.CalibrationExplanation `json:"explanation,omitempty"`
}

// toModelEvidence converts an MCP evidence input into the internal model,
// parsing timestamps leniently. Validation of category happens in the runtime.
func (e EvidenceInput) toModelEvidence() (*model.Evidence, error) {
	if strings.TrimSpace(e.Summary) == "" {
		return nil, fmt.Errorf("evidence summary is required")
	}
	if strings.TrimSpace(e.Category) == "" {
		return nil, fmt.Errorf("evidence category is required")
	}
	cat := model.Category(e.Category)
	if !cat.Valid() {
		return nil, fmt.Errorf("unknown evidence category %q", e.Category)
	}
	ts, err := parseFlexibleTime(e.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("evidence %q: %w", e.Summary, err)
	}
	return &model.Evidence{
		ID:        e.ID,
		Timestamp: ts,
		Category:  cat,
		Source:    e.Source,
		Entity:    e.Entity,
		Summary:   strings.TrimSpace(e.Summary),
		Payload:   e.Payload,
	}, nil
}

// parseFlexibleTime accepts an empty string (zero time) or several common
// layouts so assistants do not have to be pedantic about formatting.
func parseFlexibleTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized timestamp %q", s)
}
