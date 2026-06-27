package mcpserver

import (
	"fmt"
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
}

// StartOutput is the response from start_investigation.
type StartOutput struct {
	SessionID        string                  `json:"session_id"`
	Status           string                  `json:"status"`
	Progress         float64                 `json:"progress"`
	Confidence       float64                 `json:"confidence"`
	RequiredEvidence []model.EvidenceRequest `json:"required_evidence"`
	Hypotheses       []model.Hypothesis      `json:"hypotheses"`
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
	Progress             float64                 `json:"progress"`
	Confidence           float64                 `json:"confidence"`
	Status               string                  `json:"status"`
	MissingEvidence      []model.EvidenceRequest `json:"missing_evidence"`
	NextRequiredEvidence []model.EvidenceRequest `json:"next_required_evidence"`
	UpdatedHypotheses    []model.Hypothesis      `json:"updated_hypotheses"`
	Contradictions       []model.Contradiction   `json:"contradictions"`
}

// SessionIDInput is shared by status and finish tools.
type SessionIDInput struct {
	SessionID string `json:"session_id" jsonschema:"The investigation session id"`
}

// StatusOutput is the response from get_investigation_status.
type StatusOutput struct {
	SessionID       string                  `json:"session_id"`
	Question        string                  `json:"question"`
	Status          string                  `json:"status"`
	Progress        float64                 `json:"progress"`
	Confidence      float64                 `json:"confidence"`
	Hypotheses      []model.Hypothesis      `json:"hypotheses"`
	Timeline        model.Timeline          `json:"timeline"`
	Graph           *model.GraphView        `json:"graph"`
	Contradictions  []model.Contradiction   `json:"contradictions"`
	MissingEvidence []model.EvidenceRequest `json:"missing_evidence"`
	BlastRadius     model.BlastRadius       `json:"blast_radius"`
	EvidenceCount   int                     `json:"evidence_count"`
}

// toModelEvidence converts an MCP evidence input into the internal model,
// parsing timestamps leniently. Validation of category happens in the runtime.
func (e EvidenceInput) toModelEvidence() (*model.Evidence, error) {
	ts, err := parseFlexibleTime(e.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("evidence %q: %w", e.Summary, err)
	}
	return &model.Evidence{
		ID:        e.ID,
		Timestamp: ts,
		Category:  model.Category(e.Category),
		Source:    e.Source,
		Entity:    e.Entity,
		Summary:   e.Summary,
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
