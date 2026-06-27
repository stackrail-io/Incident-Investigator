package model

import "time"

// Status is the lifecycle state of an investigation.
type Status string

const (
	// StatusCollecting means the engine still wants more evidence.
	StatusCollecting Status = "collecting_evidence"
	// StatusReady means confidence is high enough to conclude, though more
	// evidence could still be submitted.
	StatusReady Status = "ready_to_conclude"
	// StatusCompleted means a final report has been generated.
	StatusCompleted Status = "completed"
)

// TimeWindow bounds the period under investigation.
type TimeWindow struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// HistoryEntry is an append-only record of something that happened to the
// session, used for auditability.
type HistoryEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"`
	Detail    string    `json:"detail,omitempty"`
}

// Session is the entire state of one investigation. Everything is incremental:
// engines update the session in place and nothing is recomputed from a database.
type Session struct {
	ID         string     `json:"session_id"`
	Question   string     `json:"question"`
	Service    string     `json:"service,omitempty"`
	TimeWindow TimeWindow `json:"time_window"`

	Evidence []*Evidence `json:"evidence"`

	Graph          *Graph          `json:"graph"`
	Hypotheses     []Hypothesis    `json:"hypotheses"`
	Timeline       Timeline        `json:"timeline"`
	Contradictions []Contradiction `json:"contradictions"`
	BlastRadius    BlastRadius     `json:"blast_radius"`

	// RequiredEvidence is the full set the planner currently wants.
	RequiredEvidence []EvidenceRequest `json:"required_evidence"`
	// MissingEvidence is the subset of RequiredEvidence not yet submitted.
	MissingEvidence []EvidenceRequest `json:"missing_evidence"`

	Confidence float64 `json:"confidence"`
	Progress   float64 `json:"progress"`
	Status     Status  `json:"status"`

	History   []HistoryEntry `json:"history"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// HasCategory reports whether at least one piece of evidence of the given
// category has been submitted.
func (s *Session) HasCategory(c Category) bool {
	for _, e := range s.Evidence {
		if e.Category == c {
			return true
		}
	}
	return false
}

// AddHistory appends an audit record.
func (s *Session) AddHistory(action, detail string, now time.Time) {
	s.History = append(s.History, HistoryEntry{Timestamp: now, Action: action, Detail: detail})
}
