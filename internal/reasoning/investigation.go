package reasoning

import (
	"context"

	"github.com/stackrail/incident-investigator/internal/graph"
	"github.com/stackrail/incident-investigator/internal/model"
)

// Investigation is a read-only view passed to reasoners during analysis.
// Reasoners must not mutate Session; they return actions instead.
type Investigation struct {
	Session *model.Session
	Signals any
	Graph   *graph.InvestigationGraph
	Engines any
}

// Reasoner proposes observations as declarative actions.
type Reasoner interface {
	Name() string
	Priority() int
	Supports(session *model.Session) bool
	Analyze(ctx context.Context, inv *Investigation) (*model.ReasoningResult, error)
}

// Orchestrator executes reasoners and applies validated actions.
type Orchestrator interface {
	Execute(ctx context.Context, inv *Investigation) (*model.ReasoningCycle, error)
}
