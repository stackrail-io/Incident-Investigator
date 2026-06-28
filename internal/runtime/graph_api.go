package runtime

import (
	"fmt"

	"github.com/stackrail/incident-investigator/internal/graph"
	"github.com/stackrail/incident-investigator/internal/model"
)

// GraphQueryInput is the input to query_graph.
type GraphQueryInput struct {
	Kind   model.GraphQueryKind
	Target string
	Limit  int
}

// SubgraphInput selects a named subgraph extract.
type SubgraphInput struct {
	Name     string
	NodeType string
	NodeIDs  []string
}

// GetReasoningCycles returns persisted reasoning cycles for replay.
func (r *Runtime) GetReasoningCycles(sessionID string) ([]model.ReasoningCycle, error) {
	s, err := r.store.Get(sessionID)
	if err != nil {
		return nil, err
	}
	if s.ReasoningCycles == nil {
		return []model.ReasoningCycle{}, nil
	}
	return s.ReasoningCycles, nil
}

func (r *Runtime) loadGraph(sessionID string) (*model.Session, *graph.InvestigationGraph, error) {
	s, err := r.store.Get(sessionID)
	if err != nil {
		return nil, nil, err
	}
	return s, graph.FromView(s.Graph), nil
}

// GetGraph returns the full investigation graph for a session.
func (r *Runtime) GetGraph(sessionID string) (*model.GraphView, error) {
	s, err := r.store.Get(sessionID)
	if err != nil {
		return nil, err
	}
	if s.Graph == nil {
		return model.NewEmptyGraphView(), nil
	}
	return s.Graph, nil
}

// QueryGraph executes a built-in graph query against the session graph.
func (r *Runtime) QueryGraph(sessionID string, in GraphQueryInput) (*model.Subgraph, error) {
	_, g, err := r.loadGraph(sessionID)
	if err != nil {
		return nil, err
	}
	return g.Query(model.GraphQuery{Kind: in.Kind, Target: in.Target, Limit: in.Limit})
}

// GetSubgraph returns a filtered subgraph by node type or explicit node ids.
func (r *Runtime) GetSubgraph(sessionID string, in SubgraphInput) (*model.Subgraph, error) {
	_, g, err := r.loadGraph(sessionID)
	if err != nil {
		return nil, err
	}
	if len(in.NodeIDs) > 0 {
		return g.SubgraphFromNodeIDs(in.NodeIDs, in.Name), nil
	}
	if in.NodeType != "" {
		return graph.SubgraphByType(g, in.Name, model.NodeType(in.NodeType)), nil
	}
	return nil, fmt.Errorf("subgraph requires node_type or node_ids")
}

// ExplainPath returns a causal explanation between two graph nodes.
func (r *Runtime) ExplainPath(sessionID, from, to string) (*model.PathExplanation, error) {
	_, g, err := r.loadGraph(sessionID)
	if err != nil {
		return nil, err
	}
	return g.ExplainPath(from, to), nil
}

// CheckGraphConsistency returns integrity issues for the session graph.
func (r *Runtime) CheckGraphConsistency(sessionID string) (*model.GraphConsistencyReport, error) {
	_, g, err := r.loadGraph(sessionID)
	if err != nil {
		return nil, err
	}
	report := graph.NewConsistencyChecker().Check(g)
	return &report, nil
}
