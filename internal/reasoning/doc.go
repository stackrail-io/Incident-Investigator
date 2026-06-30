// Package reasoning orchestrates stateless reasoners and applies validated actions.
//
// Reasoners receive read-only Investigation views and return ReasoningAction slices.
// The hybrid orchestrator merges results; the applier mutates session state.
//
// Dependency direction: reasoning → model, graph. Never import runtime.
package reasoning
