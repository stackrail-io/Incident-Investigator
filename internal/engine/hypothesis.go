package engine

import (
	"github.com/stackrail/incident-investigator/internal/archetype"
	"github.com/stackrail/incident-investigator/internal/archetype/builtin"
	"github.com/stackrail/incident-investigator/internal/model"
)

// HypothesisEngine produces a ranked field of competing explanations. It never
// returns a single hypothesis.
type HypothesisEngine interface {
	Generate(s *model.Session, sig Signals, contradictions []model.Contradiction) []model.Hypothesis
}

// HeuristicHypothesisEngine scores registered failure archetypes against the
// observed signals, then normalizes them into a probability field.
type HeuristicHypothesisEngine struct {
	registry *archetype.Registry
}

// NewHeuristicHypothesisEngine returns the default engine.
func NewHeuristicHypothesisEngine() *HeuristicHypothesisEngine {
	return &HeuristicHypothesisEngine{registry: builtin.DefaultRegistry()}
}

// NewHeuristicHypothesisEngineWithRegistry scores hypotheses using a custom archetype library.
func NewHeuristicHypothesisEngineWithRegistry(registry *archetype.Registry) *HeuristicHypothesisEngine {
	if registry == nil {
		registry = builtin.DefaultRegistry()
	}
	return &HeuristicHypothesisEngine{registry: registry}
}

// Generate implements HypothesisEngine.
func (h *HeuristicHypothesisEngine) Generate(s *model.Session, sig Signals, contradictions []model.Contradiction) []model.Hypothesis {
	return h.registry.Score(archetype.ScoreContext{
		Session:        s,
		Signals:        sig,
		Contradictions: contradictions,
	})
}

// Registry exposes the archetype library (useful for tests and extensions).
func (h *HeuristicHypothesisEngine) Registry() *archetype.Registry {
	return h.registry
}
