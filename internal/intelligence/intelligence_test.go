package intelligence_test

import (
	"context"
	"testing"

	"github.com/stackrail/incident-investigator/internal/intelligence"
	intelligencefixtures "github.com/stackrail/incident-investigator/internal/intelligence/fixtures"
	"github.com/stackrail/incident-investigator/internal/model"
)

func TestCorpusSize(t *testing.T) {
	corpus := intelligencefixtures.CompletedCorpus()
	if len(corpus) < 50 {
		t.Fatalf("expected 50+ snapshots, got %d", len(corpus))
	}
}

func TestPatternLibrarySize(t *testing.T) {
	lib := intelligence.DefaultPatternLibrary()
	if len(lib) < 15 {
		t.Fatalf("expected at least 15 investigation patterns, got %d", len(lib))
	}
	roots := map[string]bool{}
	for _, p := range lib {
		if p.TypicalRootCause != "" {
			roots[p.TypicalRootCause] = true
		}
	}
	if len(roots) < 15 {
		t.Fatalf("expected patterns for at least 15 root causes, got %d", len(roots))
	}
}

func TestArchiveStoreAndFind(t *testing.T) {
	arch := intelligence.NewMemoryArchive()
	intelligencefixtures.LoadCorpusIntoArchive(arch)
	if arch.Count() < 50 {
		t.Fatalf("expected 50+ archived, got %d", arch.Count())
	}
	first := intelligencefixtures.CompletedCorpus()[0]
	found, err := arch.Find(first.InvestigationID)
	if err != nil {
		t.Fatal(err)
	}
	if found.RootCause != first.RootCause {
		t.Errorf("root cause mismatch")
	}
}

func TestSimilaritySearchWithCorpus(t *testing.T) {
	arch := intelligence.NewMemoryArchive()
	intelligencefixtures.LoadCorpusIntoArchive(arch)
	svc := intelligence.NewMemoryServiceWithArchive(arch)

	resp, err := svc.FindSimilarInvestigations(context.Background(), model.SimilarityRequest{
		Question: "Why did checkout fail yesterday?",
		Service:  "checkout-api",
		Goal:     model.GoalRootCause,
		Limit:    5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Matches) == 0 {
		t.Fatal("expected similar matches")
	}
	if len(resp.Lessons) == 0 && len(resp.Results) > 0 {
		t.Log("lessons optional when matches exist")
	}
}

func TestPatternExtraction(t *testing.T) {
	svc := intelligence.NewMemoryService()
	arch := intelligence.NewMemoryArchive()
	intelligencefixtures.LoadCorpusIntoArchive(arch)
	svc = intelligence.NewMemoryServiceWithArchive(arch)

	resp, err := svc.SuggestPatterns(context.Background(), model.PatternRequest{
		Question: "Why did checkout fail?",
		Service:  "checkout-api",
		Goal:     model.GoalRootCause,
		Limit:    5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Patterns) == 0 {
		t.Fatal("expected patterns from library")
	}
	foundDeployment := false
	for _, p := range resp.Patterns {
		if p.Name == "Deployment Failure Pattern" {
			foundDeployment = true
		}
	}
	if !foundDeployment {
		t.Error("expected deployment failure pattern in suggestions")
	}
}

func TestConfidenceCalibrationExplained(t *testing.T) {
	arch := intelligence.NewMemoryArchive()
	intelligencefixtures.LoadCorpusIntoArchive(arch)
	svc := intelligence.NewMemoryServiceWithArchive(arch)

	resp, err := svc.CalibrateConfidence(context.Background(), model.CalibrationRequest{
		HypothesisID:  "hypothesis-deployment-caused",
		RawConfidence: 82,
		Service:       "checkout-api",
		Goal:          model.GoalRootCause,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.OriginalConfidence != 82 {
		t.Errorf("original=%v", resp.OriginalConfidence)
	}
	if resp.Reason == "" {
		t.Error("expected calibration reason")
	}
	if resp.HistoricalSampleSize == 0 {
		t.Error("expected historical sample")
	}
}

func TestFalsePositiveSimilarityFiltered(t *testing.T) {
	arch := intelligence.NewMemoryArchive()
	intelligencefixtures.LoadCorpusIntoArchive(arch)
	svc := intelligence.NewMemoryServiceWithArchive(arch)

	resp, err := svc.FindSimilarInvestigations(context.Background(), model.SimilarityRequest{
		Question: "completely unrelated quantum physics question",
		Service:  "nonexistent-service-xyz",
		Goal:     model.GoalCustom,
		Limit:    5,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range resp.Matches {
		if m.SimilarityScore > 50 {
			t.Errorf("unexpected high similarity %.1f for unrelated query", m.SimilarityScore)
		}
	}
}

func TestKnowledgeReuseRegression(t *testing.T) {
	svc := intelligence.NewMemoryService()
	ctx := context.Background()
	_, _ = svc.CalibrateConfidence(ctx, model.CalibrationRequest{
		RecordCompleted: true,
		Session:         completedSession("reg-1", "checkout-api", "Why did checkout fail?", 88, "hypothesis-deployment-caused", model.CategoryDeploymentEvents),
	})
	resp, err := svc.FindSimilarInvestigations(ctx, model.SimilarityRequest{
		Question: "Why did checkout fail?", Service: "checkout-api", Goal: model.GoalRootCause, Limit: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Matches) == 0 {
		t.Fatal("expected match after archive")
	}
}

func completedSession(id, service, question string, conf float64, hyp string, cats ...model.Category) *model.Session {
	s := &model.Session{
		ID: id, Question: question, Service: service, Goal: model.GoalRootCause,
		Status: model.StatusCompleted, Confidence: conf,
		Hypotheses: []model.Hypothesis{{ID: hyp, Confidence: conf}},
	}
	for i, c := range cats {
		s.Evidence = append(s.Evidence, &model.Evidence{ID: "e" + string(rune('a'+i)), Category: c, Summary: "obs"})
	}
	return s
}
