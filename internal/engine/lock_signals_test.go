package engine_test

import (
	"testing"
	"time"

	"github.com/stackrail/incident-investigator/internal/engine"
	"github.com/stackrail/incident-investigator/internal/fixtures"
	"github.com/stackrail/incident-investigator/internal/model"
)

func TestLockContentionSignals(t *testing.T) {
	fx := fixtures.LockContention()
	var ev []*model.Evidence
	for _, batch := range fx.Batches {
		ev = append(ev, batch...)
	}
	s := session(fx.Service, fx.Question, ev...)
	sig := engine.Analyze(s)

	if !sig.Lock.Present {
		t.Fatal("expected lock contention signals")
	}
	if sig.Lock.Entity != "identity_provider:pk-42" {
		t.Errorf("entity = %q, want identity_provider:pk-42", sig.Lock.Entity)
	}
	if len(sig.Lock.HolderIDs) != 1 || len(sig.Lock.WaiterIDs) != 3 {
		t.Errorf("holder/waiter counts = %d/%d, want 1/3", len(sig.Lock.HolderIDs), len(sig.Lock.WaiterIDs))
	}
	if !sig.Lock.MissingLockTimeouts {
		t.Error("expected missing lock timeout signal")
	}
	if !sig.Lock.HealthyDatabaseMetrics {
		t.Error("expected healthy database metrics signal")
	}
}

func TestLockContentionHypothesisLeads(t *testing.T) {
	fx := fixtures.LockContention()
	var ev []*model.Evidence
	for _, batch := range fx.Batches {
		ev = append(ev, batch...)
	}
	s := session(fx.Service, fx.Question, ev...)
	sig := engine.Analyze(s)
	hyps := engine.NewHeuristicHypothesisEngine().Generate(s, sig, nil)

	if len(hyps) == 0 {
		t.Fatal("expected hypotheses")
	}
	if hyps[0].ID != "hypothesis-lock-contention" {
		t.Fatalf("leading = %q, want hypothesis-lock-contention; field: %v", hyps[0].ID, hyps[0].Confidence)
	}
	sat := findHypothesis(hyps, "hypothesis-database-saturation")
	if sat == nil {
		t.Fatal("expected database-saturation to remain in field")
	}
	if sat.Confidence > hyps[0].Confidence {
		t.Errorf("saturation %.1f should not beat lock contention %.1f", sat.Confidence, hyps[0].Confidence)
	}
}

func TestDatabaseOutageStillLeadsSaturation(t *testing.T) {
	fx := fixtures.DatabaseOutage()
	var ev []*model.Evidence
	for _, batch := range fx.Batches {
		ev = append(ev, batch...)
	}
	s := session(fx.Service, fx.Question, ev...)
	sig := engine.Analyze(s)
	if sig.Lock.Present {
		t.Fatal("database outage fixture should not trigger lock contention")
	}
	hyps := engine.NewHeuristicHypothesisEngine().Generate(s, sig, nil)
	if hyps[0].ID != "hypothesis-database-saturation" {
		t.Errorf("leading = %q, want hypothesis-database-saturation", hyps[0].ID)
	}
}

func findHypothesis(hyps []model.Hypothesis, id string) *model.Hypothesis {
	for i := range hyps {
		if hyps[i].ID == id {
			return &hyps[i]
		}
	}
	return nil
}

func TestLockContentionRequiresRowsAffected(t *testing.T) {
	base := time.Date(2026, 6, 27, 9, 0, 0, 0, time.UTC)
	release := base.Add(10 * time.Minute)
	s := session("auth-api", "why slow writes?",
		&model.Evidence{ID: "db-1", Timestamp: release, Category: model.CategoryDatabaseEvents, Entity: "row:1", Summary: "slow update"},
		&model.Evidence{ID: "db-2", Timestamp: release, Category: model.CategoryDatabaseEvents, Entity: "row:1", Summary: "slow update"},
	)
	sig := engine.Analyze(s)
	if sig.Lock.Present {
		t.Error("expected no lock queue without rows_affected payload")
	}
}
