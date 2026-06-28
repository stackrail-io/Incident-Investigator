package graph_test

import (
	"context"
	"testing"
	"time"

	"github.com/stackrail/incident-investigator/internal/engine"
	"github.com/stackrail/incident-investigator/internal/fixtures"
	"github.com/stackrail/incident-investigator/internal/graph"
	"github.com/stackrail/incident-investigator/internal/model"
	"github.com/stackrail/incident-investigator/internal/runtime"
)

func buildDeploymentGraph(t *testing.T) *graph.InvestigationGraph {
	t.Helper()
	rt := runtime.New()
	fix := fixtures.BadDeployment()
	sess, err := rt.Start(context.Background(), runtime.StartInput{
		Question:   fix.Question,
		Service:    fix.Service,
		TimeWindow: fix.Window,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, batch := range fix.Batches {
		if _, err := rt.Submit(context.Background(), sess.ID, batch); err != nil {
			t.Fatal(err)
		}
	}
	sess, err = rt.Get(context.Background(), sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	return graph.FromView(sess.Graph)
}

func TestGraphConstruction(t *testing.T) {
	g := buildDeploymentGraph(t)
	if len(g.Nodes()) == 0 {
		t.Fatal("expected nodes")
	}
	if len(g.Edges()) == 0 {
		t.Fatal("expected edges")
	}
	hasInv := false
	for _, n := range g.Nodes() {
		if n.Type == model.NodeTypeInvestigation {
			hasInv = true
		}
	}
	if !hasInv {
		t.Error("missing investigation root node")
	}
}

func TestFromViewRoundTrip(t *testing.T) {
	g := buildDeploymentGraph(t)
	v := g.View()
	loaded := graph.FromView(v)
	if len(loaded.Nodes()) != len(v.Nodes) {
		t.Fatalf("node count mismatch: %d vs %d", len(loaded.Nodes()), len(v.Nodes))
	}
}

func TestShortestPath(t *testing.T) {
	g := graph.NewGraph()
	_ = g.AddNode(&model.GraphNode{ID: "a", Type: model.NodeDeployment, Label: "deploy"})
	_ = g.AddNode(&model.GraphNode{ID: "b", Type: model.NodeTypeConfiguration, Label: "config"})
	_ = g.AddNode(&model.GraphNode{ID: "c", Type: model.NodeEvidence, Label: "errors"})
	_ = g.AddEdge(&model.GraphEdge{From: "a", To: "b", Type: model.EdgeCauses, Confidence: 80})
	_ = g.AddEdge(&model.GraphEdge{From: "b", To: "c", Type: model.EdgeCauses, Confidence: 70})

	sg := g.ShortestPath("a", "c")
	if len(sg.Nodes) != 3 {
		t.Fatalf("path nodes = %d, want 3", len(sg.Nodes))
	}
}

func TestQueryUpstream(t *testing.T) {
	g := graph.NewGraph()
	_ = g.AddNode(&model.GraphNode{ID: "svc:checkout-api", Type: model.NodeService, Label: "checkout-api"})
	_ = g.AddNode(&model.GraphNode{ID: "dep", Type: model.NodeDeployment, Label: "deploy"})
	_ = g.AddNode(&model.GraphNode{ID: "cfg", Type: model.NodeTypeConfiguration, Label: "config"})
	_ = g.AddEdge(&model.GraphEdge{From: "dep", To: "cfg", Type: model.EdgeCauses, Confidence: 90})
	_ = g.AddEdge(&model.GraphEdge{From: "cfg", To: "svc:checkout-api", Type: model.EdgeCauses, Confidence: 85})

	sg, err := g.Query(model.GraphQuery{Kind: model.QueryUpstream, Target: "checkout-api"})
	if err != nil {
		t.Fatal(err)
	}
	if len(sg.Nodes) < 2 {
		t.Fatalf("upstream nodes = %d, want at least 2", len(sg.Nodes))
	}
}

func TestSupportingEvidenceQuery(t *testing.T) {
	g := buildDeploymentGraph(t)
	var hypID string
	for _, n := range g.Nodes() {
		if n.Type == model.NodeTypeHypothesis {
			hypID = n.ID
			break
		}
	}
	if hypID == "" {
		t.Fatal("no hypothesis node")
	}
	sg, err := g.Query(model.GraphQuery{Kind: model.QuerySupportingEvidence, Target: hypID})
	if err != nil {
		t.Fatal(err)
	}
	if len(sg.Edges) == 0 {
		t.Error("expected supporting edges")
	}
}

func TestSubgraphByType(t *testing.T) {
	g := buildDeploymentGraph(t)
	sg := graph.SubgraphByType(g, "evidence", model.NodeEvidence)
	if len(sg.Nodes) == 0 {
		t.Error("expected evidence nodes in subgraph")
	}
}

func TestConsistencyChecker(t *testing.T) {
	g := graph.NewGraph()
	_ = g.AddNode(&model.GraphNode{ID: "orphan", Type: model.NodeEvidence, Label: "lonely"})
	report := graph.NewConsistencyChecker().Check(g)
	if len(report.Issues) == 0 {
		t.Error("expected orphan node issue")
	}
}

func TestInferenceEngine(t *testing.T) {
	g := graph.NewGraph()
	_ = g.AddNode(&model.GraphNode{ID: "e1", Type: model.NodeEvidence, Label: "deploy"})
	_ = g.AddNode(&model.GraphNode{ID: "e2", Type: model.NodeEvidence, Label: "latency"})
	_ = g.AddEdge(&model.GraphEdge{
		From: "e1", To: "e2", Type: model.EdgeOccurredBefore,
		Confidence: 100, Timestamp: time.Now().UTC(),
	})
	engine.NewInferenceEngine().Apply(g, engine.Signals{})
	found := false
	for _, e := range g.Edges() {
		if e.Inferred && e.Type == model.EdgeCorrelatesWith {
			found = true
		}
	}
	if !found {
		t.Error("expected inferred correlation edge")
	}
}

func TestCausalEngineBranching(t *testing.T) {
	g := graph.NewGraph()
	_ = g.AddNode(&model.GraphNode{ID: "root", Type: model.NodeDeployment, Label: "deploy"})
	_ = g.AddNode(&model.GraphNode{ID: "a", Type: model.NodeEvidence, Label: "a"})
	_ = g.AddNode(&model.GraphNode{ID: "b", Type: model.NodeEvidence, Label: "b"})
	_ = g.AddEdge(&model.GraphEdge{From: "root", To: "a", Type: model.EdgeCauses, Confidence: 80})
	_ = g.AddEdge(&model.GraphEdge{From: "root", To: "b", Type: model.EdgeCauses, Confidence: 75})
	engine.NewCausalEngine().Analyze(g)
	n, _ := g.Store().GetNode("root")
	if n.Properties["branching_failures"] != 2 {
		t.Errorf("branching_failures = %v, want 2", n.Properties["branching_failures"])
	}
}

func TestExplainPath(t *testing.T) {
	g := graph.NewGraph()
	_ = g.AddNode(&model.GraphNode{ID: "dep", Type: model.NodeDeployment, Label: "Deployment"})
	_ = g.AddNode(&model.GraphNode{ID: "err", Type: model.NodeEvidence, Label: "API Errors"})
	_ = g.AddEdge(&model.GraphEdge{From: "dep", To: "err", Type: model.EdgeCauses, Confidence: 83, EvidenceRefs: []string{"ev-1"}})
	ex := g.ExplainPath("dep", "err")
	if ex == nil || len(ex.Edges) == 0 {
		t.Fatal("expected path explanation")
	}
	if ex.Confidence <= 0 {
		t.Error("expected positive path confidence")
	}
}

func TestLargeInvestigationGraph(t *testing.T) {
	rt := runtime.New()
	for _, fix := range fixtures.All() {
		sess, err := rt.Start(context.Background(), runtime.StartInput{
			Question:   fix.Question,
			Service:    fix.Service,
			TimeWindow: fix.Window,
		})
		if err != nil {
			t.Fatalf("%s start: %v", fix.Name, err)
		}
		for _, batch := range fix.Batches {
			if _, err := rt.Submit(context.Background(), sess.ID, batch); err != nil {
				t.Fatalf("%s submit: %v", fix.Name, err)
			}
		}
		sess, _ = rt.Get(context.Background(), sess.ID)
		if sess.Graph == nil || len(sess.Graph.Nodes) < 3 {
			t.Errorf("%s: graph too small", fix.Name)
		}
	}
}
