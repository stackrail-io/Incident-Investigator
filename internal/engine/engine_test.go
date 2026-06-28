package engine_test

import (
	"testing"
	"time"

	"github.com/stackrail/incident-investigator/internal/engine"
	"github.com/stackrail/incident-investigator/internal/model"
)

func session(service, question string, ev ...*model.Evidence) *model.Session {
	return &model.Session{
		ID:       "test",
		Question: question,
		Service:  service,
		Evidence: ev,
		Graph:    model.NewGraph(),
	}
}

// TestPlannerIsDynamic verifies the planner asks for more, sharper evidence once
// a deployment shows up.
func TestPlannerIsDynamic(t *testing.T) {
	p := engine.NewHeuristicPlanner()

	empty := session("checkout-api", "why did checkout fail?")
	before := p.Plan(empty, engine.Analyze(empty))

	withDeploy := session("checkout-api", "why did checkout fail?",
		&model.Evidence{ID: "d1", Timestamp: time.Now(), Category: model.CategoryDeploymentEvents, Summary: "Deployed v2"},
	)
	after := p.Plan(withDeploy, engine.Analyze(withDeploy))

	if len(after) <= len(before) {
		t.Errorf("expected the planner to request more evidence after a deployment: before=%d after=%d", len(before), len(after))
	}
	if !requests(after, model.CategoryConfigurationChanges) {
		t.Errorf("expected configuration_changes to be requested after a deployment")
	}
}

// TestHypothesesAlwaysCompete ensures the engine never returns a single
// hypothesis and that confidences normalize to ~100%.
func TestHypothesesAlwaysCompete(t *testing.T) {
	h := engine.NewHeuristicHypothesisEngine()
	s := session("orders-api", "why are orders timing out?",
		&model.Evidence{ID: "l1", Timestamp: time.Now(), Category: model.CategoryApplicationLogs, Summary: "database connection pool exhausted; query timeout"},
	)
	sig := engine.Analyze(s)
	hyps := h.Generate(s, sig, nil)

	if len(hyps) < 2 {
		t.Fatalf("expected multiple competing hypotheses, got %d", len(hyps))
	}
	var sum float64
	for _, hp := range hyps {
		sum += hp.Confidence
	}
	if sum < 99 || sum > 101 {
		t.Errorf("confidences should sum to ~100, got %.2f", sum)
	}
	if hyps[0].ID != "hypothesis-database-saturation" {
		t.Errorf("expected database hypothesis to lead, got %q", hyps[0].ID)
	}
}

// TestConfidenceRewardsCorroboration checks that multi-category agreement yields
// higher confidence than a single log line.
func TestConfidenceRewardsCorroboration(t *testing.T) {
	rt := func(ev ...*model.Evidence) float64 {
		s := session("orders-api", "why are orders failing?", ev...)
		sig := engine.Analyze(s)
		contradictions := engine.NewHeuristicContradictionDetector().Detect(s, sig)
		hyps := engine.NewHeuristicHypothesisEngine().Generate(s, sig, contradictions)
		s.Hypotheses = hyps
		required := engine.NewHeuristicPlanner().Plan(s, sig)
		prog := engine.Progress(s, required)
		return engine.NewHeuristicConfidenceScorer().Score(s, sig, hyps, contradictions, prog)
	}

	now := time.Now()
	sparse := rt(
		&model.Evidence{ID: "l1", Timestamp: now, Category: model.CategoryApplicationLogs, Entity: "orders-api", Summary: "database query timeout"},
	)
	rich := rt(
		&model.Evidence{ID: "d1", Timestamp: now, Category: model.CategoryDeploymentEvents, Entity: "orders-api", Summary: "Deployed orders v2"},
		&model.Evidence{ID: "l1", Timestamp: now.Add(time.Minute), Category: model.CategoryApplicationLogs, Entity: "orders-api", Summary: "database query timeout after deploy"},
		&model.Evidence{ID: "m1", Timestamp: now.Add(2 * time.Minute), Category: model.CategoryMetrics, Entity: "orders-api", Summary: "error rate 20%"},
		&model.Evidence{ID: "db1", Timestamp: now.Add(3 * time.Minute), Category: model.CategoryDatabaseEvents, Entity: "orders-db", Summary: "connection pool saturated"},
	)

	if rich <= sparse {
		t.Errorf("expected richer evidence to yield higher confidence: sparse=%.2f rich=%.2f", sparse, rich)
	}
}

func requests(reqs []model.EvidenceRequest, c model.Category) bool {
	for _, r := range reqs {
		if r.Category == c {
			return true
		}
	}
	return false
}

func TestRollbackNotFirstDeployment(t *testing.T) {
	base := time.Date(2026, 6, 27, 9, 0, 0, 0, time.UTC)
	s := session("checkout-api", "why did checkout fail?",
		&model.Evidence{ID: "dep", Timestamp: base, Category: model.CategoryDeploymentEvents, Summary: "Deployed checkout-api v2.4.0"},
		&model.Evidence{ID: "rb", Timestamp: base.Add(5 * time.Minute), Category: model.CategoryDeploymentEvents, Summary: "Rollback checkout-api to v2.3.9 after errors"},
	)
	sig := engine.Analyze(s)
	if sig.FirstDeployment == nil || sig.FirstDeployment.ID != "dep" {
		t.Fatalf("FirstDeployment = %v, want dep", sig.FirstDeployment)
	}
	if sig.Recovery == nil || sig.Recovery.ID != "rb" {
		t.Fatalf("Recovery = %v, want rb", sig.Recovery)
	}
}

func TestRecoveryKeywordsDoNotMarkDeploy(t *testing.T) {
	base := time.Date(2026, 6, 27, 9, 0, 0, 0, time.UTC)
	s := session("api", "why did api fail?",
		&model.Evidence{ID: "log", Timestamp: base, Category: model.CategoryApplicationLogs, Summary: "HTTP 500 errors"},
		&model.Evidence{ID: "rec", Timestamp: base.Add(time.Minute), Category: model.CategoryApplicationLogs, Summary: "Service recovered; incident resolved"},
	)
	sig := engine.Analyze(s)
	if sig.FirstDeployment != nil {
		t.Errorf("expected no deployment, got %v", sig.FirstDeployment)
	}
	if sig.Recovery == nil || sig.Recovery.ID != "rec" {
		t.Fatalf("Recovery = %v, want rec", sig.Recovery)
	}
}

func TestTopCandidatesSkipsRefuted(t *testing.T) {
	s := session("api", "why did api fail?")
	s.Hypotheses = []model.Hypothesis{
		{ID: "h1", Statement: "refuted", Status: model.StatusRefuted, Confidence: 50},
		{ID: "h2", Statement: "active", Status: model.StatusLeading, Confidence: 30},
		{ID: "h3", Statement: "also active", Status: model.StatusSupported, Confidence: 20},
	}
	report := engine.NewHeuristicReportGenerator().Generate(s, engine.Analyze(s))
	if len(report.RootCauseCandidates) != 2 {
		t.Fatalf("got %d candidates, want 2", len(report.RootCauseCandidates))
	}
	if report.RootCauseCandidates[0].ID != "h2" {
		t.Errorf("first candidate = %q, want h2", report.RootCauseCandidates[0].ID)
	}
}
