package examples

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stackrail/incident-investigator/internal/events"
	"github.com/stackrail/incident-investigator/internal/model"
	"github.com/stackrail/incident-investigator/internal/runtime"
)

var catalog = []string{
	"deployment-failure",
	"certificate-expiry",
	"dns-outage",
	"retry-storm",
	"database-deadlock",
	"memory-leak",
	"regional-outage",
}

func TestExampleInvestigations(t *testing.T) {
	root, err := RepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range catalog {
		t.Run(name, func(t *testing.T) {
			runExample(t, root, name)
		})
	}
}

func runExample(t *testing.T, root, name string) {
	t.Helper()
	sc, err := Load(root, name)
	if err != nil {
		t.Fatalf("load example: %v", err)
	}
	exp, err := loadFindingsExpect(root, name)
	if err != nil {
		t.Fatalf("expected-findings: %v", err)
	}
	goal, window, err := sc.StartInput()
	if err != nil {
		t.Fatalf("start input: %v", err)
	}

	rt := runtime.New()
	ctx := context.Background()
	sess, err := rt.Start(ctx, runtime.StartInput{
		Question:   sc.Investigation.Question,
		Service:    sc.Investigation.Service,
		TimeWindow: window,
		Goal:       goal,
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	assertExpectedQuestions(t, root, name, sess)

	for i, batch := range sc.Batches {
		sess, err = rt.Submit(ctx, sess.ID, batch)
		if err != nil {
			t.Fatalf("submit batch %d: %v", i+1, err)
		}
	}

	if leading := leadingHypothesis(sess); leading == nil || leading.ID != exp.LeadingHypothesisID {
		got := "<none>"
		if leading != nil {
			got = leading.ID
		}
		t.Fatalf("leading hypothesis: got %s want %s", got, exp.LeadingHypothesisID)
	}
	if exp.MinConfidence > 0 && sess.Confidence < exp.MinConfidence {
		t.Fatalf("confidence %.1f below minimum %.1f", sess.Confidence, exp.MinConfidence)
	}
	if exp.CompetingHypothesesMin > 0 && len(sess.Hypotheses) < exp.CompetingHypothesesMin {
		t.Fatalf("hypotheses %d want >= %d", len(sess.Hypotheses), exp.CompetingHypothesesMin)
	}

	assertExpectedGraph(t, root, name, sess)
	report, _, err := rt.Finish(ctx, sess.ID)
	if err != nil {
		t.Fatalf("finish: %v", err)
	}
	assertRCAReport(t, report, sc)
	assertExpectedReport(t, root, name, report, sc)
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

type findingsExpect struct {
	LeadingHypothesisID    string  `json:"leading_hypothesis_id"`
	MinConfidence          float64 `json:"min_confidence"`
	CompetingHypothesesMin int     `json:"competing_hypotheses_min"`
}

type graphExpect struct {
	MinNodes int `json:"min_nodes"`
	MinEdges int `json:"min_edges"`
}

type questionsExpect struct {
	MinPlanQuestions int `json:"min_plan_questions"`
}

func loadFindingsExpect(root, name string) (findingsExpect, error) {
	var exp findingsExpect
	err := readJSON(filepath.Join(root, "examples", name, "expected-findings.json"), &exp)
	return exp, err
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

func assertRCAReport(t *testing.T, report model.Report, sc *Scenario) {
	t.Helper()
	pm := report.Postmortem
	for _, section := range []string{
		"# Root Cause Analysis:",
		"## Executive Summary",
		"## Chronological Timeline",
		"## Root Cause Analysis",
		"## Recommendations",
	} {
		if !strings.Contains(pm, section) {
			t.Errorf("report missing section %q", section)
		}
	}
	if !strings.Contains(pm, "| Time (UTC) | Evidence | Category | Entity | Event |") {
		t.Error("report missing chronological timeline table")
	}
	evidence := sc.AllEvidence()
	if len(report.Timeline) != len(evidence) {
		t.Errorf("timeline entries %d, want %d evidence items", len(report.Timeline), len(evidence))
	}
	if !timelineSorted(report.Timeline) {
		t.Error("timeline is not chronologically ordered")
	}
	lastIdx := -1
	for _, entry := range report.Timeline {
		if len(entry.EvidenceRefs) == 0 {
			t.Errorf("timeline entry missing evidence ref: %q", entry.Description)
			continue
		}
		ref := "`" + entry.EvidenceRefs[0] + "`"
		idx := strings.Index(pm, ref)
		if idx < 0 {
			t.Errorf("report missing evidence ref %s", entry.EvidenceRefs[0])
			continue
		}
		if idx < lastIdx {
			t.Errorf("evidence %s appears out of chronological order in report", entry.EvidenceRefs[0])
		}
		lastIdx = idx
		if !strings.Contains(pm, entry.Description) {
			t.Errorf("report missing timeline event: %q", entry.Description)
		}
	}
}

func timelineSorted(tl model.Timeline) bool {
	for i := 1; i < len(tl); i++ {
		if tl[i].Timestamp.Before(tl[i-1].Timestamp) {
			return false
		}
	}
	return true
}

func assertExpectedReport(t *testing.T, root, name string, report model.Report, sc *Scenario) {
	t.Helper()
	path := filepath.Join(root, "examples", name, "expected-report.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected-report: %v", err)
	}
	expected := string(data)
	for _, section := range []string{
		"## Chronological Timeline",
		"## Root Cause Analysis",
		"## Recommendations",
	} {
		if !strings.Contains(expected, section) {
			t.Errorf("expected-report.md missing section %q", section)
		}
	}
	for _, e := range sc.AllEvidence() {
		if !strings.Contains(expected, e.Summary) {
			t.Errorf("expected-report.md missing evidence summary: %q", e.Summary)
		}
		if !strings.Contains(expected, "`"+e.ID+"`") {
			t.Errorf("expected-report.md missing evidence id: %q", e.ID)
		}
	}
	if normalizeRCA(expected) != normalizeRCA(report.Postmortem) {
		t.Errorf("report drift from expected-report.md — run: go run internal/spec/cmd/gen-examples/main.go")
	}
}

// normalizeRCA strips confidence percentages so report golden files stay stable
// across minor scoring tweaks.
var rcaPercentRE = regexp.MustCompile(`\d+(\.\d+)?%`)

func normalizeRCA(s string) string {
	var b strings.Builder
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "_Generated by") {
			continue
		}
		line = strings.ReplaceAll(line, "\r", "")
		line = rcaPercentRE.ReplaceAllString(line, "N%")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func TestExampleEvidenceFilesLoad(t *testing.T) {
	root, err := RepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range catalog {
		sc, err := Load(root, name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(sc.Batches) == 0 {
			t.Fatalf("%s: no evidence batches", name)
		}
		total := 0
		for _, b := range sc.Batches {
			total += len(b)
		}
		if total < 2 {
			t.Fatalf("%s: only %d evidence items", name, total)
		}
	}
}

func TestExampleEventBus(t *testing.T) {
	root, err := RepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	sc, err := Load(root, "deployment-failure")
	if err != nil {
		t.Fatal(err)
	}
	goal, window, _ := sc.StartInput()
	rt := runtime.New()
	var types []string
	rt.EventBus().Subscribe(func(e events.Event) {
		types = append(types, string(e.Type))
	})
	ctx := context.Background()
	sess, err := rt.Start(ctx, runtime.StartInput{
		Question: sc.Investigation.Question, Service: sc.Investigation.Service,
		TimeWindow: window, Goal: goal,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, batch := range sc.Batches {
		if _, err := rt.Submit(ctx, sess.ID, batch); err != nil {
			t.Fatal(err)
		}
	}
	if len(types) == 0 {
		t.Fatal("expected events")
	}
}
