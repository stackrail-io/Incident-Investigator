package engine

import (
	"context"

	"github.com/stackrail/incident-investigator/internal/reasoning"
	"github.com/stackrail/incident-investigator/internal/graph"
	"github.com/stackrail/incident-investigator/internal/model"
)

// ReasoningContext carries session state through a reasoning pass.
type ReasoningContext struct {
	Session *model.Session
	Signals Signals
	Engines RuntimeEngines
}

// RuntimeEngines bundles the engines reasoners read from.
type RuntimeEngines struct {
	Planner       Planner
	Hypothesis    HypothesisEngine
	Confidence    ConfidenceScorer
	Contradiction ContradictionDetector
	Missing       MissingEvidenceDetector
	Graph         GraphBuilder
	Timeline      TimelineBuilder
	Blast         BlastRadiusEstimator
	Coverage      CoverageEngine
	Strategy      StrategyEngine
	Sufficiency   SufficiencyEngine
	Importance    ImportanceEngine
	StateMachine  *StateMachine
}

// Orchestrator runs the multi-reasoner hybrid orchestrator.
type Orchestrator struct {
	hybrid *reasoning.HybridOrchestrator
}

// NewOrchestrator returns an orchestrator. Prefer NewOrchestratorWithEngines for defaults.
func NewOrchestrator(reasoners ...reasoning.Reasoner) *Orchestrator {
	reg := reasoning.NewRegistry()
	for _, r := range reasoners {
		reg.Register(r)
	}
	return &Orchestrator{
		hybrid: reasoning.NewHybridOrchestrator(reg, reasoning.WithStrategy(model.StrategySequential)),
	}
}

// NewOrchestratorWithRegistry returns an orchestrator with a pre-built registry.
func NewOrchestratorWithRegistry(reg *reasoning.Registry, opts ...reasoning.HybridOption) *Orchestrator {
	opts = append([]reasoning.HybridOption{reasoning.WithStrategy(model.StrategySequential)}, opts...)
	return &Orchestrator{
		hybrid: reasoning.NewHybridOrchestrator(reg, opts...),
	}
}

// NewOrchestratorWithEngines returns an orchestrator wired to runtime engines.
// Deprecated: use NewOrchestratorWithRegistry(reasoners.DefaultRegistry(eng)) from runtime wiring.
func NewOrchestratorWithEngines(eng RuntimeEngines, opts ...reasoning.HybridOption) *Orchestrator {
	_ = eng
	reg := reasoning.NewRegistry()
	return NewOrchestratorWithRegistry(reg, opts...)
}

// WithRegistry replaces the reasoner registry.
func (o *Orchestrator) WithRegistry(reg *reasoning.Registry, opts ...reasoning.HybridOption) {
	o.hybrid = reasoning.NewHybridOrchestrator(reg, opts...)
}

// Run executes all reasoners and applies validated actions to the session.
func (o *Orchestrator) Run(ctx context.Context, rtCtx *ReasoningContext) (*model.ReasoningCycle, error) {
	if o.hybrid == nil {
		return nil, nil
	}
	var g *graph.InvestigationGraph
	if rtCtx.Session != nil && rtCtx.Session.Graph != nil {
		g = graph.FromView(rtCtx.Session.Graph)
	}
	inv := &reasoning.Investigation{
		Session: rtCtx.Session,
		Signals: rtCtx.Signals,
		Graph:   g,
		Engines: rtCtx.Engines,
	}
	return o.hybrid.Execute(ctx, inv)
}
