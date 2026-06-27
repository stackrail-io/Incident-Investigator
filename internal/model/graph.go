package model

import "sort"

// NodeType enumerates the kinds of vertices in the evidence graph.
type NodeType string

const (
	NodeEvidence   NodeType = "evidence"
	NodeHypothesis NodeType = "hypothesis"
	NodeService    NodeType = "service"
	NodeDeployment NodeType = "deployment"
	NodeAlert      NodeType = "alert"
	NodeMetric     NodeType = "metric"
)

// EdgeType enumerates the kinds of directed relationships in the graph.
type EdgeType string

const (
	EdgeOccurredBefore EdgeType = "occurred_before"
	EdgeLikelyCaused   EdgeType = "likely_caused"
	EdgeSupports       EdgeType = "supports"
	EdgeContradicts    EdgeType = "contradicts"
	EdgeRecoveredAfter EdgeType = "recovered_after"
)

// Node is a vertex in the evidence graph.
type Node struct {
	ID    string   `json:"id"`
	Type  NodeType `json:"type"`
	Label string   `json:"label"`
	// RefID references the underlying object (e.g. an evidence id or hypothesis id).
	RefID string `json:"ref_id,omitempty"`
}

// Edge is a directed, confidence-weighted relationship between two nodes.
type Edge struct {
	From         string   `json:"from"`
	To           string   `json:"to"`
	Type         EdgeType `json:"type"`
	Confidence   float64  `json:"confidence"`
	Reason       string   `json:"reason,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

// Graph is a directed graph of evidence, hypotheses and the entities involved.
type Graph struct {
	nodes map[string]*Node
	edges []*Edge
}

// NewGraph returns an empty graph.
func NewGraph() *Graph {
	return &Graph{nodes: map[string]*Node{}}
}

// AddNode inserts a node if it is not already present (idempotent by ID).
func (g *Graph) AddNode(n *Node) {
	if g.nodes == nil {
		g.nodes = map[string]*Node{}
	}
	if _, ok := g.nodes[n.ID]; !ok {
		g.nodes[n.ID] = n
	}
}

// HasNode reports whether a node with the given ID exists.
func (g *Graph) HasNode(id string) bool {
	_, ok := g.nodes[id]
	return ok
}

// AddEdge appends a directed edge. Callers are responsible for ensuring the
// referenced nodes exist.
func (g *Graph) AddEdge(e *Edge) {
	g.edges = append(g.edges, e)
}

// Reset clears the graph so it can be rebuilt from scratch.
func (g *Graph) Reset() {
	g.nodes = map[string]*Node{}
	g.edges = nil
}

// Nodes returns the nodes sorted deterministically by ID.
func (g *Graph) Nodes() []*Node {
	out := make([]*Node, 0, len(g.nodes))
	for _, n := range g.nodes {
		out = append(out, n)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Edges returns the edges in insertion order.
func (g *Graph) Edges() []*Edge {
	out := make([]*Edge, len(g.edges))
	copy(out, g.edges)
	return out
}

// GraphView is the exported, serializable projection of a Graph. It exists so
// that JSON-schema inference (used by the MCP layer) sees concrete properties
// rather than the Graph's unexported maps.
type GraphView struct {
	Nodes []*Node `json:"nodes"`
	Edges []*Edge `json:"edges"`
}

// View returns a stable, serializable snapshot of the graph.
func (g *Graph) View() *GraphView {
	if g == nil {
		return &GraphView{Nodes: []*Node{}, Edges: []*Edge{}}
	}
	v := &GraphView{Nodes: g.Nodes(), Edges: g.Edges()}
	if v.Nodes == nil {
		v.Nodes = []*Node{}
	}
	if v.Edges == nil {
		v.Edges = []*Edge{}
	}
	return v
}

// MarshalJSON renders the graph as a stable {nodes, edges} object.
func (g *Graph) MarshalJSON() ([]byte, error) {
	return jsonMarshal(g.View())
}
