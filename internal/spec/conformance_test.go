package spec_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stackrail/incident-investigator/internal/model"
	runtimepkg "github.com/stackrail/incident-investigator/internal/runtime"
	"github.com/stackrail/incident-investigator/internal/spec"
)

func runConformanceFixture(t *testing.T, path string) {
	t.Helper()
	fx, err := spec.LoadConformanceFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	runConformanceFixtureData(t, fx)
}

func runConformanceFixtureData(t *testing.T, fx *spec.ConformanceFixture) {
	t.Helper()

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

	for i, batch := range fx.EvidenceBatches {
		ev := make([]*model.Evidence, len(batch))
		for j, item := range batch {
			ts, err := time.Parse(time.RFC3339, item.Timestamp)
			if err != nil {
				t.Fatalf("batch %d item %d timestamp: %v", i, j, err)
			}
			ev[j] = &model.Evidence{
				ID:        item.ID,
				Timestamp: ts,
				Category:  model.Category(item.Category),
				Entity:    item.Entity,
				Summary:   item.Summary,
				Payload:   item.Payload,
				Source:    "provided_by_client",
			}
		}
		sess, err = rt.Submit(ctx, sess.ID, ev)
		if err != nil {
			t.Fatalf("SubmitEvidence batch %d: %v", i, err)
		}
	}

	if len(sess.Hypotheses) < fx.ExpectAfterAllEvidence.MinHypotheses {
		t.Errorf("hypotheses: got %d, want >= %d", len(sess.Hypotheses), fx.ExpectAfterAllEvidence.MinHypotheses)
	}
	assertHypothesisLeadQuality(t, sess.Hypotheses, fx)
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

func assertHypothesisLeadQuality(t *testing.T, hyps []model.Hypothesis, fx *spec.ConformanceFixture) {
	t.Helper()
	if len(hyps) == 0 {
		t.Fatal("expected non-empty hypothesis field")
	}

	wantID := fx.ExpectAfterAllEvidence.LeadingHypothesisID
	leader, ok := hypothesisByID(hyps, wantID)
	if !ok {
		t.Fatalf("expected leader %q not found in hypothesis field", wantID)
	}

	runnerUpID, runnerUpConf := strongestOther(hyps, wantID)
	if runnerUpConf > leader.Confidence {
		t.Errorf("leading hypothesis: %q=%.1f should beat runner-up %q=%.1f",
			wantID, leader.Confidence, runnerUpID, runnerUpConf)
	}

	for _, id := range fx.EffectiveMustNotLead() {
		if h, ok := hypothesisByID(hyps, id); ok && h.Confidence >= leader.Confidence {
			t.Errorf("hypothesis %q must not lead or tie leader %q (%.1f vs %.1f)",
				id, wantID, h.Confidence, leader.Confidence)
		}
	}

	margin := fx.EffectiveLeadMargin()
	if margin <= 0 {
		return
	}
	gap := leader.Confidence - runnerUpConf
	if gap < margin {
		t.Errorf("lead margin: got %.1f, want >= %.1f (leader %q=%.1f, runner-up %q=%.1f)",
			gap, margin, wantID, leader.Confidence, runnerUpID, runnerUpConf)
	}
}

func hypothesisByID(hyps []model.Hypothesis, id string) (model.Hypothesis, bool) {
	for _, h := range hyps {
		if h.ID == id {
			return h, true
		}
	}
	return model.Hypothesis{}, false
}

func strongestOther(hyps []model.Hypothesis, excludeID string) (string, float64) {
	bestID := ""
	var best float64
	for _, h := range hyps {
		if h.ID == excludeID {
			continue
		}
		if h.Confidence > best {
			best = h.Confidence
			bestID = h.ID
		}
	}
	return bestID, best
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
