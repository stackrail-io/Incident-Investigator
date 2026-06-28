package spec_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stackrail/incident-investigator/internal/model"
	runtimepkg "github.com/stackrail/incident-investigator/internal/runtime"
)

type conformanceFixture struct {
	ScenarioID   string `json:"scenario_id"`
	Tier         string `json:"tier"`
	Description  string `json:"description"`
	Start        struct {
		Question   string `json:"question"`
		Service    string `json:"service"`
		Goal       string `json:"goal"`
		TimeWindow struct {
			Start string `json:"start"`
			End   string `json:"end"`
		} `json:"time_window"`
	} `json:"start"`
	ExpectAfterStart struct {
		RequiredEvidenceMin int `json:"required_evidence_min"`
		PlanQuestionsMin    int `json:"plan_questions_min"`
	} `json:"expect_after_start"`
	EvidenceBatches []json.RawMessage `json:"evidence_batches"`
	ExpectAfterAllEvidence struct {
		LeadingHypothesisID string  `json:"leading_hypothesis_id"`
		MinConfidence       float64 `json:"min_confidence"`
		MinHypotheses       int     `json:"min_hypotheses"`
		GraphNodesMin       int     `json:"graph_nodes_min"`
	} `json:"expect_after_all_evidence"`
	ExpectAfterFinish struct {
		State                string   `json:"state"`
		ReportRequiredFields []string `json:"report_required_fields"`
	} `json:"expect_after_finish"`
}

func TestSpecConformanceFixtures(t *testing.T) {
	dir := conformanceFixturesDir(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read fixtures: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		t.Run(e.Name(), func(t *testing.T) {
			runConformanceFixture(t, filepath.Join(dir, e.Name()))
		})
	}
}

func runConformanceFixture(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fx conformanceFixture
	if err := json.Unmarshal(data, &fx); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	rt := runtimepkg.New(runtimepkg.WithClock(func() time.Time { return now }))
	ctx := context.Background()

	tw := model.TimeWindow{}
	if fx.Start.TimeWindow.Start != "" {
		tw.Start, _ = time.Parse(time.RFC3339, fx.Start.TimeWindow.Start)
	}
	if fx.Start.TimeWindow.End != "" {
		tw.End, _ = time.Parse(time.RFC3339, fx.Start.TimeWindow.End)
	}

	sess, err := rt.Start(ctx, runtimepkg.StartInput{
		Question:   fx.Start.Question,
		Service:    fx.Start.Service,
		Goal:       model.InvestigationGoal(fx.Start.Goal),
		TimeWindow: tw,
	})
	if err != nil {
		t.Fatalf("StartInvestigation: %v", err)
	}
	if len(sess.RequiredEvidence) < fx.ExpectAfterStart.RequiredEvidenceMin {
		t.Errorf("required_evidence: got %d, want >= %d", len(sess.RequiredEvidence), fx.ExpectAfterStart.RequiredEvidenceMin)
	}
	if sess.Plan != nil && len(sess.Plan.Questions) < fx.ExpectAfterStart.PlanQuestionsMin {
		t.Errorf("plan questions: got %d, want >= %d", len(sess.Plan.Questions), fx.ExpectAfterStart.PlanQuestionsMin)
	}

	for i, batchRaw := range fx.EvidenceBatches {
		var batch []model.Evidence
		if err := json.Unmarshal(batchRaw, &batch); err != nil {
			t.Fatalf("batch %d: %v", i, err)
		}
		ev := make([]*model.Evidence, len(batch))
		for j := range batch {
			ev[j] = &batch[j]
		}
		sess, err = rt.Submit(ctx, sess.ID, ev)
		if err != nil {
			t.Fatalf("SubmitEvidence batch %d: %v", i, err)
		}
	}

	if len(sess.Hypotheses) < fx.ExpectAfterAllEvidence.MinHypotheses {
		t.Errorf("hypotheses: got %d, want >= %d", len(sess.Hypotheses), fx.ExpectAfterAllEvidence.MinHypotheses)
	}
	if len(sess.Hypotheses) > 0 && sess.Hypotheses[0].ID != fx.ExpectAfterAllEvidence.LeadingHypothesisID {
		t.Errorf("leading hypothesis: got %q, want %q", sess.Hypotheses[0].ID, fx.ExpectAfterAllEvidence.LeadingHypothesisID)
	}
	if sess.Confidence < fx.ExpectAfterAllEvidence.MinConfidence {
		t.Errorf("confidence: got %.1f, want >= %.1f", sess.Confidence, fx.ExpectAfterAllEvidence.MinConfidence)
	}
	if sess.Graph != nil && len(sess.Graph.Nodes) < fx.ExpectAfterAllEvidence.GraphNodesMin {
		t.Errorf("graph nodes: got %d, want >= %d", len(sess.Graph.Nodes), fx.ExpectAfterAllEvidence.GraphNodesMin)
	}

	report, finished, err := rt.Finish(ctx, sess.ID)
	if err != nil {
		t.Fatalf("FinishInvestigation: %v", err)
	}
	if string(finished.State) != fx.ExpectAfterFinish.State {
		t.Errorf("state after finish: got %q, want %q", finished.State, fx.ExpectAfterFinish.State)
	}
	reportJSON, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	var reportMap map[string]any
	if err := json.Unmarshal(reportJSON, &reportMap); err != nil {
		t.Fatal(err)
	}
	for _, field := range fx.ExpectAfterFinish.ReportRequiredFields {
		if _, ok := reportMap[field]; !ok {
			t.Errorf("report missing required field %q", field)
		}
	}

	// Spec v1: investigation snapshot must serialize core fields.
	snap, err := json.Marshal(sessionToSpecInvestigation(finished))
	if err != nil {
		t.Fatal(err)
	}
	var inv map[string]any
	if err := json.Unmarshal(snap, &inv); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"investigation_id", "question", "state", "evidence", "hypotheses", "confidence"} {
		if _, ok := inv[key]; !ok {
			t.Errorf("Investigation spec snapshot missing %q", key)
		}
	}
}

func sessionToSpecInvestigation(s *model.Session) map[string]any {
	return map[string]any{
		"investigation_id": s.ID,
		"session_id":       s.ID,
		"question":         s.Question,
		"service":          s.Service,
		"goal":             s.Goal,
		"state":            s.State,
		"status":           s.Status,
		"evidence":         s.Evidence,
		"hypotheses":       s.Hypotheses,
		"confidence":       s.Confidence,
		"graph":            s.Graph,
		"plan":             s.Plan,
	}
}

func conformanceFixturesDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "spec", "investigation-v1", "conformance", "fixtures")
}
