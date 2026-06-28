package graph

import (
	"fmt"
	"strings"

	"github.com/stackrail/incident-investigator/internal/model"
)

// Query executes a built-in graph query.
func (g *InvestigationGraph) Query(q model.GraphQuery) (*model.Subgraph, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 30
	}
	switch q.Kind {
	case model.QueryUpstream:
		return g.queryUpstream(q.Target, limit), nil
	case model.QueryDownstream:
		return g.queryDownstream(q.Target, limit), nil
	case model.QuerySupportingEvidence:
		return g.querySupportingEvidence(q.Target), nil
	case model.QueryContradictions:
		return g.queryContradictions(q.Target), nil
	case model.QueryUnansweredQuestions:
		return g.queryUnansweredQuestions(), nil
	case model.QueryServiceEvidence:
		return g.queryServiceEvidence(q.Target), nil
	case model.QueryBlastRadius:
		return g.queryBlastRadius(q.Target, limit), nil
	case model.QueryShortestCausalPath:
		parts := strings.SplitN(q.Target, "->", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("shortest_causal_path target must be from->to")
		}
		return g.ShortestPath(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])), nil
	case model.QueryStrongestPath:
		return g.strongestPath(q.Target, limit), nil
	default:
		return nil, fmt.Errorf("unknown query kind %q", q.Kind)
	}
}

func (g *InvestigationGraph) queryUpstream(target string, limit int) *model.Subgraph {
	start := resolveNodeID(g, target)
	return g.reverseTraverse(start, limit, "upstream")
}

func (g *InvestigationGraph) queryDownstream(target string, limit int) *model.Subgraph {
	start := resolveNodeID(g, target)
	return g.Traverse(start, TraversalCausal, limit)
}

func (g *InvestigationGraph) reverseTraverse(start string, limit int, name string) *model.Subgraph {
	rev := map[string][]string{}
	for _, e := range g.store.AllEdges() {
		if causalEdgeTypes[e.Type] || e.Type == model.EdgeDependsOn || e.Type == model.EdgeTriggered {
			rev[e.To] = append(rev[e.To], e.From)
		}
	}
	visited := map[string]bool{}
	order := []string{start}
	queue := []string{start}
	for len(queue) > 0 && len(order) < limit {
		id := queue[0]
		queue = queue[1:]
		for _, prev := range rev[id] {
			if !visited[prev] {
				visited[prev] = true
				order = append(order, prev)
				queue = append(queue, prev)
			}
		}
	}
	return g.subgraphFromNodes(order, name)
}

func (g *InvestigationGraph) querySupportingEvidence(hypothesisID string) *model.Subgraph {
	hid := hypothesisID
	if !strings.HasPrefix(hid, "hyp:") {
		hid = "hyp:" + hypothesisID
	}
	var nodeIDs []string
	var edges []*model.GraphEdge
	for _, e := range g.store.AllEdges() {
		if e.To == hid && (e.Type == model.EdgeSupports || e.Type == model.EdgeCorrelatesWith) {
			nodeIDs = append(nodeIDs, e.From, e.To)
			cp := e
			edges = append(edges, &cp)
		}
	}
	sg := g.subgraphFromNodes(unique(nodeIDs), "supporting_evidence")
	sg.Edges = edges
	return sg
}

func (g *InvestigationGraph) queryContradictions(hypothesisID string) *model.Subgraph {
	hid := hypothesisID
	if !strings.HasPrefix(hid, "hyp:") {
		hid = "hyp:" + hypothesisID
	}
	var nodeIDs []string
	for _, e := range g.store.AllEdges() {
		if e.To == hid && e.Type == model.EdgeContradicts {
			nodeIDs = append(nodeIDs, e.From, e.To)
		}
	}
	return g.subgraphFromNodes(unique(nodeIDs), "contradictions")
}

func (g *InvestigationGraph) queryUnansweredQuestions() *model.Subgraph {
	var ids []string
	for _, n := range g.store.AllNodes() {
		if n.Type != model.NodeTypeQuestion {
			continue
		}
		status, _ := n.Properties["status"].(string)
		if status != string(model.QuestionAnswered) && status != string(model.QuestionRejected) {
			ids = append(ids, n.ID)
		}
	}
	return g.subgraphFromNodes(ids, "unanswered_questions")
}

func (g *InvestigationGraph) queryServiceEvidence(service string) *model.Subgraph {
	sid := "svc:" + service
	var ids []string
	for _, e := range g.store.AllEdges() {
		if e.To == sid || e.From == sid {
			ids = append(ids, e.From, e.To)
		}
	}
	ids = append(ids, sid)
	return g.subgraphFromNodes(unique(ids), "service_evidence")
}

func (g *InvestigationGraph) queryBlastRadius(service string, limit int) *model.Subgraph {
	sg := g.queryDownstream("svc:"+service, limit)
	sg.Name = "blast_radius"
	return sg
}

func (g *InvestigationGraph) strongestPath(target string, limit int) *model.Subgraph {
	start := resolveNodeID(g, target)
	type scored struct {
		id    string
		score float64
	}
	var best []scored
	for _, e := range g.store.AllEdges() {
		if e.From == start && causalEdgeTypes[e.Type] {
			best = append(best, scored{e.To, e.Confidence})
		}
	}
	if len(best) == 0 {
		return g.subgraphFromNodes([]string{start}, "strongest_path")
	}
	// pick highest confidence outgoing causal edge
	top := best[0]
	for _, b := range best[1:] {
		if b.score > top.score {
			top = b
		}
	}
	return g.subgraphFromNodes([]string{start, top.id}, "strongest_path")
}

// ExplainPath returns a causal explanation between two nodes.
func (g *InvestigationGraph) ExplainPath(from, to string) *model.PathExplanation {
	fromID := resolveNodeID(g, from)
	toID := resolveNodeID(g, to)
	sg := g.ShortestPath(fromID, toID)
	reason := fmt.Sprintf("Causal path from %s to %s with %d hop(s).", from, to, len(sg.Edges))
	conf := 0.0
	for _, e := range sg.Edges {
		conf += e.Confidence
	}
	if len(sg.Edges) > 0 {
		conf /= float64(len(sg.Edges))
	}
	return &model.PathExplanation{
		From:       from,
		To:         to,
		Reason:     reason,
		Confidence: conf,
		Nodes:      sg.Nodes,
		Edges:      sg.Edges,
	}
}

func resolveNodeID(g *InvestigationGraph, target string) string {
	if g.HasNode(target) {
		return target
	}
	for _, prefix := range []string{"svc:", "hyp:", "ev:", "q:", "inv:"} {
		candidate := prefix + target
		if g.HasNode(candidate) {
			return candidate
		}
	}
	for _, n := range g.store.AllNodes() {
		if n.Label == target || n.RefID == target {
			return n.ID
		}
	}
	return target
}

func unique(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range in {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
