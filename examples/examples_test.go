package examples

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stackrail/incident-investigator/internal/events"
	"github.com/stackrail/incident-investigator/internal/fixtures"
	"github.com/stackrail/incident-investigator/internal/model"
	"github.com/stackrail/incident-investigator/internal/runtime"
	"github.com/stackrail/incident-investigator/internal/spec"
)

// example maps a public example name to its archetype conformance fixture.
type example struct {
	Name           string
	FixtureID      string
	LeadingHyp     string
	MinConfidence  float64
	ReportContains []string
}

var catalog = []example{
	{Name: "deployment-failure", FixtureID: "deployment-failure", LeadingHyp: "hypothesis-deployment-caused", MinConfidence: 15, ReportContains: []string{"deploy"}},
	{Name: "certificate-expiry", FixtureID: "certificate-tls-failure", LeadingHyp: "hypothesis-certificate-expiry", MinConfidence: 15, ReportContains: []string{"certificate", "TLS"}},
	{Name: "dns-outage", FixtureID: "dns-failure", LeadingHyp: "hypothesis-dns-failure", MinConfidence: 15, ReportContains: []string{"DNS"}},
	{Name: "retry-storm", FixtureID: "retry-storm", LeadingHyp: "hypothesis-retry-storm", MinConfidence: 10},
	{Name: "database-deadlock", FixtureID: "database-lock-contention", LeadingHyp: "hypothesis-lock-contention", MinConfidence: 15, ReportContains: []string{"lock"}},
	{Name: "memory-leak", FixtureID: "resource-exhaustion", LeadingHyp: "hypothesis-resource-exhaustion", MinConfidence: 10, ReportContains: []string{"memory", "OOM"}},
	{Name: "regional-outage", FixtureID: "regional-failure", LeadingHyp: "hypothesis-regional-failure", MinConfidence: 10},
}

func TestExampleInvestigations(t *testing.T) {
	root, err := fixtures.RepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	for _, ex := range catalog {
		t.Run(ex.Name, func(t *testing.T) {
			runExample(t, root, ex)
		})
	}
}

func runExample(t *testing.T, root string, ex example) {
	t.Helper()
	path := filepath.Join(root, "spec/investigation-v1/conformance/archetype-fixtures", ex.FixtureID+".yaml")
	fx, err := spec.LoadConformanceFixture(path)
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	fix, err := fixtures.FromConformance(fx)
	if err != nil {
		t.Fatalf("from conformance: %v", err)
	}

	rt := runtime.New()
	ctx := context.Background()
	sess, err := rt.Start(ctx, runtime.StartInput{
		Question:   fix.Question,
		Service:    fix.Service,
		TimeWindow: fix.Window,
		Goal:       model.GoalRootCause,
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	assertExpectedQuestions(t, root, ex.Name, sess)

	for _, batch := range fix.Batches {
		sess, err = rt.Submit(ctx, sess.ID, batch)
		if err != nil {
			t.Fatalf("submit: %v", err)
		}
	}

	if leading := leadingHypothesis(sess); leading == nil || leading.ID != ex.LeadingHyp {
		got := "<none>"
		if leading != nil {
			got = leading.ID
		}
		t.Fatalf("leading hypothesis: got %s want %s", got, ex.LeadingHyp)
	}
	if ex.MinConfidence > 0 && sess.Confidence < ex.MinConfidence {
		t.Fatalf("confidence %.1f below minimum %.1f", sess.Confidence, ex.MinConfidence)
	}

	assertExpectedGraph(t, root, ex.Name, sess)
	assertExpectedFindings(t, root, ex.Name, ex.LeadingHyp)

	report, _, err := rt.Finish(ctx, sess.ID)
	if err != nil {
		t.Fatalf("finish: %v", err)
	}
	for _, phrase := range ex.ReportContains {
		if !strings.Contains(strings.ToLower(report.ExecutiveSummary), strings.ToLower(phrase)) &&
			!strings.Contains(strings.ToLower(report.Postmortem), strings.ToLower(phrase)) {
			t.Errorf("report missing expected phrase %q", phrase)
		}
	}
	assertExpectedReport(t, root, ex.Name, report)
}

func leadingHypothesis(s *model.Session) *model.Hypothesis {
	if s == nil || len(s.Hypotheses) == 0 {
		return nil
	}
	best := &s.Hypotheses[0]
	for i := 1; i < len(s.Hypotheses); i++ {
		if s.Hypotheses[i].Confidence > best.Confidence {
			best = &s.Hypotheses[i]
		}
	}
	return best
}

type graphExpect struct {
	MinNodes int `json:"min_nodes"`
	MinEdges int `json:"min_edges"`
}

func assertExpectedGraph(t *testing.T, root, name string, s *model.Session) {
	t.Helper()
	path := filepath.Join(root, "examples", name, "expected-graph.json")
	var exp graphExpect
	if err := readJSON(path, &exp); err != nil {
		t.Fatalf("expected-graph: %v", err)
	}
	nodes, edges := 0, 0
	if s.Graph != nil {
		nodes = len(s.Graph.Nodes)
		edges = len(s.Graph.Edges)
	}
	if nodes < exp.MinNodes {
		t.Errorf("graph nodes: got %d want >= %d", nodes, exp.MinNodes)
	}
	if edges < exp.MinEdges {
		t.Errorf("graph edges: got %d want >= %d", edges, exp.MinEdges)
	}
}

type questionsExpect struct {
	MinPlanQuestions int `json:"min_plan_questions"`
}

func assertExpectedQuestions(t *testing.T, root, name string, s *model.Session) {
	t.Helper()
	path := filepath.Join(root, "examples", name, "expected-questions.json")
	var exp questionsExpect
	if err := readJSON(path, &exp); err != nil {
		t.Fatalf("expected-questions: %v", err)
	}
	count := 0
	if s.Plan != nil {
		count = len(s.Plan.Questions)
	}
	if count < exp.MinPlanQuestions {
		t.Errorf("plan questions: got %d want >= %d", count, exp.MinPlanQuestions)
	}
}

type findingsExpect struct {
	LeadingHypothesisID string `json:"leading_hypothesis_id"`
}

func assertExpectedFindings(t *testing.T, root, name, leading string) {
	t.Helper()
	path := filepath.Join(root, "examples", name, "expected-findings.json")
	var exp findingsExpect
	if err := readJSON(path, &exp); err != nil {
		t.Fatalf("expected-findings: %v", err)
	}
	if exp.LeadingHypothesisID != "" && exp.LeadingHypothesisID != leading {
		t.Errorf("expected findings leading %s got %s", exp.LeadingHypothesisID, leading)
	}
}

func assertExpectedReport(t *testing.T, root, name string, report model.Report) {
	t.Helper()
	path := filepath.Join(root, "examples", name, "expected-report.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected-report: %v", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.Contains(report.ExecutiveSummary, line) && !strings.Contains(report.Postmortem, line) {
			t.Errorf("report missing expected content: %q", line)
		}
	}
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// TestExampleEventBus verifies the runtime publishes lifecycle events.
func TestExampleEventBus(t *testing.T) {
	fix := fixtures.BadDeployment()
	rt := runtime.New()
	var types []string
	rt.EventBus().Subscribe(func(e events.Event) {
		types = append(types, string(e.Type))
	})
	ctx := context.Background()
	sess, err := rt.Start(ctx, runtime.StartInput{
		Question: fix.Question, Service: fix.Service, TimeWindow: fix.Window, Goal: model.GoalRootCause,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, batch := range fix.Batches {
		if _, err := rt.Submit(ctx, sess.ID, batch); err != nil {
			t.Fatal(err)
		}
	}
	if len(types) == 0 {
		t.Fatal("expected events")
	}
}
