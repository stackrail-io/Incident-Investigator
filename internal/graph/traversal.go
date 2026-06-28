package graph

import (
	"sort"

	"github.com/stackrail/incident-investigator/internal/model"
)

// TraversalMode selects a graph walk strategy.
type TraversalMode string

const (
	TraversalBFS          TraversalMode = "bfs"
	TraversalDFS          TraversalMode = "dfs"
	TraversalShortestPath TraversalMode = "shortest_path"
	TraversalTopological  TraversalMode = "topological"
	TraversalCausal       TraversalMode = "causal"
	TraversalTimeline     TraversalMode = "timeline"
)

var causalEdgeTypes = map[model.EdgeType]bool{
	model.EdgeCauses:       true,
	model.EdgeLikelyCaused: true,
	model.EdgeTriggered:    true,
}

var temporalEdgeTypes = map[model.EdgeType]bool{
	model.EdgeOccurredBefore: true,
	model.EdgeOccurredAfter:  true,
}

// Traverse walks the graph from start using the given mode and edge filter.
func (g *InvestigationGraph) Traverse(start string, mode TraversalMode, limit int) *model.Subgraph {
	if limit <= 0 {
		limit = 50
	}
	adj := g.adjacency(mode)
	visited := map[string]bool{}
	order := []string{}

	switch mode {
	case TraversalDFS:
		var dfs func(string)
		dfs = func(id string) {
			if visited[id] || len(order) >= limit {
				return
			}
			visited[id] = true
			order = append(order, id)
			for _, next := range adj[id] {
				dfs(next)
			}
		}
		dfs(start)
	default:
		queue := []string{start}
		for len(queue) > 0 && len(order) < limit {
			id := queue[0]
			queue = queue[1:]
			if visited[id] {
				continue
			}
			visited[id] = true
			order = append(order, id)
			for _, next := range adj[id] {
				if !visited[next] {
					queue = append(queue, next)
				}
			}
		}
	}

	return g.subgraphFromNodes(order, "")
}

func (g *InvestigationGraph) adjacency(mode TraversalMode) map[string][]string {
	adj := map[string][]string{}
	for _, e := range g.store.AllEdges() {
		if mode == TraversalCausal && !causalEdgeTypes[e.Type] {
			continue
		}
		if mode == TraversalTimeline && !temporalEdgeTypes[e.Type] {
			continue
		}
		adj[e.From] = append(adj[e.From], e.To)
	}
	return adj
}

// ShortestPath finds a shortest path between two nodes (BFS).
func (g *InvestigationGraph) ShortestPath(from, to string) *model.Subgraph {
	if from == to {
		return g.subgraphFromNodes([]string{from}, "shortest_path")
	}
	adj := map[string][]string{}
	for _, e := range g.store.AllEdges() {
		adj[e.From] = append(adj[e.From], e.To)
	}
	parent := map[string]string{}
	visited := map[string]bool{from: true}
	queue := []string{from}
	found := false
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, next := range adj[cur] {
			if visited[next] {
				continue
			}
			visited[next] = true
			parent[next] = cur
			if next == to {
				found = true
				break
			}
			queue = append(queue, next)
		}
		if found {
			break
		}
	}
	if !found {
		return &model.Subgraph{Name: "shortest_path", Nodes: []*model.GraphNode{}, Edges: []*model.GraphEdge{}}
	}
	path := []string{to}
	for cur := to; cur != from; {
		cur = parent[cur]
		path = append([]string{cur}, path...)
	}
	return g.subgraphFromNodes(path, "shortest_path")
}

// SubgraphFromNodeIDs extracts a subgraph containing the given node ids.
func (g *InvestigationGraph) SubgraphFromNodeIDs(nodeIDs []string, name string) *model.Subgraph {
	return g.subgraphFromNodes(nodeIDs, name)
}

func (g *InvestigationGraph) subgraphFromNodes(nodeIDs []string, name string) *model.Subgraph {
	want := map[string]bool{}
	for _, id := range nodeIDs {
		want[id] = true
	}
	var nodes []*model.GraphNode
	for _, n := range g.store.AllNodes() {
		if want[n.ID] {
			cp := *n
			nodes = append(nodes, &cp)
		}
	}
	var edges []*model.GraphEdge
	for _, e := range g.store.AllEdges() {
		if want[e.From] && want[e.To] {
			cp := e
			edges = append(edges, &cp)
		}
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	return &model.Subgraph{Name: name, Nodes: nodes, Edges: edges}
}
