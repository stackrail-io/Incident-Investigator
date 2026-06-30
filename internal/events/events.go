// Package events provides an internal event bus for investigation lifecycle
// notifications. Components subscribe to events instead of calling each other
// directly.
//
// The runtime publishes events; subscribers must not mutate session state.
package events

import (
	"sync"
	"time"
)

// Type identifies an investigation lifecycle event.
type Type string

const (
	EvidenceAdded          Type = "EvidenceAdded"
	QuestionCreated        Type = "QuestionCreated"
	QuestionResolved       Type = "QuestionResolved"
	HypothesisCreated      Type = "HypothesisCreated"
	HypothesisUpdated      Type = "HypothesisUpdated"
	ReasoningCompleted     Type = "ReasoningCompleted"
	InvestigationCompleted Type = "InvestigationCompleted"
	PatternMatched         Type = "PatternMatched"
)

// Event is a single lifecycle notification.
type Event struct {
	Type            Type      `json:"type"`
	InvestigationID string    `json:"investigation_id"`
	Timestamp       time.Time `json:"timestamp"`
	Detail          string    `json:"detail,omitempty"`
	Payload         any       `json:"payload,omitempty"`
}

// Handler reacts to an event.
type Handler func(Event)

// Bus dispatches events to subscribers.
type Bus struct {
	mu       sync.RWMutex
	handlers []Handler
}

// NewBus returns an event bus.
func NewBus() *Bus {
	return &Bus{}
}

// Subscribe registers a handler. Handlers run synchronously in registration order.
func (b *Bus) Subscribe(h Handler) {
	if h == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers = append(b.handlers, h)
}

// Publish delivers an event to all subscribers.
func (b *Bus) Publish(e Event) {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}
	b.mu.RLock()
	handlers := append([]Handler(nil), b.handlers...)
	b.mu.RUnlock()
	for _, h := range handlers {
		h(e)
	}
}
