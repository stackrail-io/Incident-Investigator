package reasoners

import (
	"github.com/stackrail/incident-investigator/internal/engine"
	"github.com/stackrail/incident-investigator/internal/reasoning"
)

// DefaultRegistry registers built-in capability reasoners.
func DefaultRegistry(eng engine.RuntimeEngines) *reasoning.Registry {
	reg := reasoning.NewRegistry()
	reg.Register(NewTemporalReasoner(eng))
	reg.Register(NewCausalReasoner())
	reg.Register(NewHypothesisReasoner(eng))
	reg.Register(NewConsistencyReasoner(eng))
	reg.Register(reasoning.NewSemanticReasoner(reasoning.NewHostLLM()))
	return reg
}
