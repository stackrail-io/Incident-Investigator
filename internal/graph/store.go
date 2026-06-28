package graph

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/stackrail/incident-investigator/internal/model"
)

// ErrNodeNotFound is returned when a node id does not exist.
var ErrNodeNotFound = errors.New("graph node not found")

// ErrEdgeNotFound is returned when an edge id does not exist.
var ErrEdgeNotFound = errors.New("graph edge not found")

// Store persists investigation graph nodes and edges.
type Store interface {
	AddNode(node model.GraphNode) error
	UpdateNode(node model.GraphNode) error
	DeleteNode(id string) error
	GetNode(id string) (*model.GraphNode, error)
	AddEdge(edge model.GraphEdge) error
	RemoveEdge(id string) error
	GetEdges(nodeID string) ([]model.GraphEdge, error)
	Traverse(query model.GraphQuery) (*model.Subgraph, error)
	AllNodes() []*model.GraphNode
	AllEdges() []model.GraphEdge
	Reset()
}

// MemoryStore is an in-memory graph store guarded by a mutex.
type MemoryStore struct {
	mu    sync.RWMutex
	nodes map[string]*model.GraphNode
	edges map[string]model.GraphEdge
	order []string
}

// NewMemoryStore returns an empty in-memory graph store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		nodes: map[string]*model.GraphNode{},
		edges: map[string]model.GraphEdge{},
	}
}

func (m *MemoryStore) AddNode(node model.GraphNode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.nodes[node.ID]; !ok {
		n := node
		m.nodes[node.ID] = &n
		m.order = append(m.order, node.ID)
	}
	return nil
}

func (m *MemoryStore) UpdateNode(node model.GraphNode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.nodes[node.ID]; !ok {
		return ErrNodeNotFound
	}
	n := node
	m.nodes[node.ID] = &n
	return nil
}

func (m *MemoryStore) DeleteNode(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.nodes[id]; !ok {
		return ErrNodeNotFound
	}
	delete(m.nodes, id)
	for eid, e := range m.edges {
		if e.From == id || e.To == id {
			delete(m.edges, eid)
		}
	}
	return nil
}

func (m *MemoryStore) GetNode(id string) (*model.GraphNode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n, ok := m.nodes[id]
	if !ok {
		return nil, ErrNodeNotFound
	}
	cp := *n
	return &cp, nil
}

func (m *MemoryStore) AddEdge(edge model.GraphEdge) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if edge.ID == "" {
		edge.ID = fmt.Sprintf("edge-%s-%s-%s", edge.From, edge.To, edge.Type)
	}
	m.edges[edge.ID] = edge
	return nil
}

func (m *MemoryStore) RemoveEdge(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.edges[id]; !ok {
		return ErrEdgeNotFound
	}
	delete(m.edges, id)
	return nil
}

func (m *MemoryStore) GetEdges(nodeID string) ([]model.GraphEdge, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []model.GraphEdge
	for _, e := range m.edges {
		if e.From == nodeID || e.To == nodeID {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (m *MemoryStore) AllNodes() []*model.GraphNode {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*model.GraphNode, 0, len(m.order))
	for _, id := range m.order {
		if n, ok := m.nodes[id]; ok {
			cp := *n
			out = append(out, &cp)
		}
	}
	return out
}

func (m *MemoryStore) AllEdges() []model.GraphEdge {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]model.GraphEdge, 0, len(m.edges))
	for _, e := range m.edges {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (m *MemoryStore) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nodes = map[string]*model.GraphNode{}
	m.edges = map[string]model.GraphEdge{}
	m.order = nil
}

// Traverse delegates to the query engine on an InvestigationGraph wrapper.
func (m *MemoryStore) Traverse(query model.GraphQuery) (*model.Subgraph, error) {
	g := &InvestigationGraph{store: m}
	return g.Query(query)
}

// InvestigationGraph is the canonical in-memory investigation graph.
type InvestigationGraph struct {
	store Store
}

// NewInvestigationGraph returns a graph backed by an in-memory store.
func NewInvestigationGraph() *InvestigationGraph {
	return &InvestigationGraph{store: NewMemoryStore()}
}

// WithStore constructs a graph with a custom store (for future backends).
func WithStore(store Store) *InvestigationGraph {
	return &InvestigationGraph{store: store}
}

// Store exposes the underlying graph store.
func (g *InvestigationGraph) Store() Store { return g.store }

// AddNode inserts a node idempotently.
func (g *InvestigationGraph) AddNode(n *model.GraphNode) error {
	if n == nil {
		return nil
	}
	return g.store.AddNode(*n)
}

// HasNode reports whether a node exists.
func (g *InvestigationGraph) HasNode(id string) bool {
	_, err := g.store.GetNode(id)
	return err == nil
}

// AddEdge appends an edge.
func (g *InvestigationGraph) AddEdge(e *model.GraphEdge) error {
	if e == nil {
		return nil
	}
	return g.store.AddEdge(*e)
}

// Reset clears the graph.
func (g *InvestigationGraph) Reset() { g.store.Reset() }

// Nodes returns all nodes.
func (g *InvestigationGraph) Nodes() []*model.GraphNode { return g.store.AllNodes() }

// Edges returns all edges as pointers.
func (g *InvestigationGraph) Edges() []*model.GraphEdge {
	all := g.store.AllEdges()
	out := make([]*model.GraphEdge, len(all))
	for i := range all {
		e := all[i]
		out[i] = &e
	}
	return out
}

// View returns a serializable snapshot.
func (g *InvestigationGraph) View() *model.GraphView {
	nodes := g.Nodes()
	edges := g.Edges()
	if nodes == nil {
		nodes = []*model.GraphNode{}
	}
	if edges == nil {
		edges = []*model.GraphEdge{}
	}
	return &model.GraphView{Nodes: nodes, Edges: edges}
}

// NewGraph is the legacy factory name for an empty investigation graph.
func NewGraph() *InvestigationGraph { return NewInvestigationGraph() }

// GraphStore is the storage abstraction alias (see Store).
type GraphStore = Store

// MemoryGraphStore is the in-memory Store implementation.
type MemoryGraphStore = MemoryStore

// NewMemoryGraphStore returns an empty in-memory graph store.
func NewMemoryGraphStore() *MemoryGraphStore { return NewMemoryStore() }

// FromView loads a graph snapshot into a queryable InvestigationGraph.
func FromView(v *model.GraphView) *InvestigationGraph {
	g := NewInvestigationGraph()
	if v == nil {
		return g
	}
	for _, n := range v.Nodes {
		if n != nil {
			_ = g.AddNode(n)
		}
	}
	for _, e := range v.Edges {
		if e != nil {
			_ = g.AddEdge(e)
		}
	}
	return g
}

// MarshalJSON renders the graph as {nodes, edges}.
func (g *InvestigationGraph) MarshalJSON() ([]byte, error) {
	return json.Marshal(g.View())
}
