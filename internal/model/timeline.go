package model

import "time"

// TimelineEntry is a single moment in the reconstructed incident timeline. Every
// entry references the evidence that supports it.
type TimelineEntry struct {
	Timestamp    time.Time `json:"timestamp"`
	Description  string    `json:"description"`
	Category     Category  `json:"category"`
	Entity       string    `json:"entity,omitempty"`
	EvidenceRefs []string  `json:"evidence_refs"`
}

// Timeline is an ordered sequence of entries.
type Timeline []TimelineEntry
