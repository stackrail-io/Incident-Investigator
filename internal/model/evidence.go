package model

import "time"

// Evidence is a single, vendor-neutral observation submitted by the AI
// assistant. The engine never depends on vendor schemas: arbitrary detail lives
// in the opaque Payload map.
type Evidence struct {
	ID        string         `json:"id"`
	Timestamp time.Time      `json:"timestamp"`
	Category  Category       `json:"category"`
	Source    string         `json:"source"`
	Entity    string         `json:"entity"`
	Summary   string         `json:"summary"`
	Payload   map[string]any `json:"payload,omitempty"`
}

// EvidenceRequest describes a category of evidence the engine would like the
// assistant to collect next, along with the reasoning behind the request.
type EvidenceRequest struct {
	Category Category `json:"category"`
	Priority Priority `json:"priority"`
	Reason   string   `json:"reason,omitempty"`
}
