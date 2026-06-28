package graph

import (
	"time"

	"github.com/stackrail/incident-investigator/internal/model"
)

// ConsistencyChecker validates graph integrity.
type ConsistencyChecker struct{}

// NewConsistencyChecker returns the default checker.
func NewConsistencyChecker() *ConsistencyChecker { return &ConsistencyChecker{} }

// Check returns all consistency issues found.
func (c *ConsistencyChecker) Check(g *InvestigationGraph) model.GraphConsistencyReport {
	var issues []model.GraphIssue
	nodeIDs := map[string]bool{}
	for _, n := range g.store.AllNodes() {
		nodeIDs[n.ID] = true
	}

	edgeKeys := map[string]bool{}
	for _, e := range g.store.AllEdges() {
		if !nodeIDs[e.From] || !nodeIDs[e.To] {
			issues = append(issues, model.GraphIssue{
				Severity: "high", Code: "dangling_reference",
				Message: "Edge " + e.ID + " references missing node.",
			})
		}
		key := e.From + "|" + e.To + "|" + string(e.Type)
		if edgeKeys[key] {
			issues = append(issues, model.GraphIssue{
				Severity: "medium", Code: "duplicate_edge",
				Message: "Duplicate edge: " + key,
			})
		}
		edgeKeys[key] = true
	}

	connected := map[string]bool{}
	for _, e := range g.store.AllEdges() {
		connected[e.From] = true
		connected[e.To] = true
	}
	for _, n := range g.store.AllNodes() {
		if n.Type == model.NodeTypeInvestigation {
			continue
		}
		if !connected[n.ID] {
			issues = append(issues, model.GraphIssue{
				Severity: "low", Code: "orphan_node",
				Message: "Node " + n.ID + " has no edges.",
			})
		}
	}

	if cycle := findDependsCycle(g); cycle != "" {
		issues = append(issues, model.GraphIssue{
			Severity: "high", Code: "cycle",
			Message: "Dependency cycle detected: " + cycle,
		})
	}

	if issues == nil {
		issues = []model.GraphIssue{}
	}
	return model.GraphConsistencyReport{Issues: issues}
}

func findDependsCycle(g *InvestigationGraph) string {
	adj := map[string][]string{}
	for _, e := range g.store.AllEdges() {
		if e.Type == model.EdgeDependsOn {
			adj[e.From] = append(adj[e.From], e.To)
		}
	}
	visited := map[string]bool{}
	stack := map[string]bool{}
	var dfs func(string) string
	dfs = func(n string) string {
		visited[n] = true
		stack[n] = true
		for _, next := range adj[n] {
			if !visited[next] {
				if c := dfs(next); c != "" {
					return c
				}
			} else if stack[next] {
				return n + " -> " + next
			}
		}
		stack[n] = false
		return ""
	}
	for _, n := range g.store.AllNodes() {
		if !visited[n.ID] {
			if c := dfs(n.ID); c != "" {
				return c
			}
		}
	}
	return ""
}

// ExtractSubgraph returns a named subgraph filter.
func ExtractSubgraph(g *InvestigationGraph, name string, nodeFilter func(*model.GraphNode) bool) *model.Subgraph {
	var ids []string
	for _, n := range g.store.AllNodes() {
		if nodeFilter(n) {
			ids = append(ids, n.ID)
		}
	}
	return g.subgraphFromNodes(ids, name)
}

// SubgraphByType extracts nodes of given types.
func SubgraphByType(g *InvestigationGraph, name string, types ...model.NodeType) *model.Subgraph {
	want := map[model.NodeType]bool{}
	for _, t := range types {
		want[t] = true
	}
	return ExtractSubgraph(g, name, func(n *model.GraphNode) bool { return want[n.Type] })
}

func nowUTC() time.Time { return time.Now().UTC() }
