package engine

import (
	"time"

	"github.com/stackrail/incident-investigator/internal/graph"
	"github.com/stackrail/incident-investigator/internal/model"
)

// InferenceEngine infers edges from temporal and semantic patterns.
type InferenceEngine struct{}

// NewInferenceEngine returns the default inference engine.
func NewInferenceEngine() *InferenceEngine { return &InferenceEngine{} }

// Apply adds inferred edges to the investigation graph.
func (i *InferenceEngine) Apply(g *graph.InvestigationGraph, sig Signals) {
	for _, e := range g.Store().AllEdges() {
		if e.Type == model.EdgeOccurredBefore && !e.Inferred {
			_ = g.AddEdge(&model.GraphEdge{
				From: e.From, To: e.To, Type: model.EdgeCorrelatesWith,
				Confidence: 55, Weight: 0.5, Reason: "Inferred correlation from temporal proximity.",
				EvidenceRefs: e.EvidenceRefs, Inferred: true, Timestamp: time.Now().UTC(),
			})
		}
	}
	if sig.Keywords["database"] {
		for _, n := range g.Store().AllNodes() {
			if n.Type != model.NodeTypeDatabase {
				continue
			}
			for _, h := range g.Store().AllNodes() {
				if h.Type == model.NodeTypeHypothesis {
					_ = g.AddEdge(&model.GraphEdge{
						From: n.ID, To: h.ID, Type: model.EdgeCorrelatesWith,
						Confidence: 50, Inferred: true, Reason: "Inferred database-hypothesis correlation.",
					})
				}
			}
		}
	}
}

// CausalEngine analyzes causal chains in the investigation graph.
type CausalEngine struct{}

// NewCausalEngine returns the default causal engine.
func NewCausalEngine() *CausalEngine { return &CausalEngine{} }

// Analyze annotates branching failure nodes on causal paths.
func (c *CausalEngine) Analyze(g *graph.InvestigationGraph) {
	outCount := map[string]int{}
	for _, e := range g.Store().AllEdges() {
		if e.Type == model.EdgeCauses || e.Type == model.EdgeLikelyCaused {
			outCount[e.From]++
		}
	}
	for id, count := range outCount {
		if count < 2 {
			continue
		}
		n, err := g.Store().GetNode(id)
		if err != nil {
			continue
		}
		if n.Properties == nil {
			n.Properties = map[string]any{}
		}
		n.Properties["branching_failures"] = count
		_ = g.Store().UpdateNode(*n)
	}
}
