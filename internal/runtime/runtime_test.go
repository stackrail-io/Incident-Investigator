package runtime_test

import (
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/stackrail/incident-investigator/internal/fixtures"
	"github.com/stackrail/incident-investigator/internal/model"
	"github.com/stackrail/incident-investigator/internal/runtime"
)

func fixedClock() func() time.Time {
	t := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	return func() time.Time { return t }
}

func newRuntime() *runtime.Runtime {
	return runtime.New(runtime.WithClock(fixedClock()))
}

// TestFixtures replays every realistic incident fixture end to end and asserts
// the planner, timeline, graph, hypotheses and confidence all behave.
func TestFixtures(t *testing.T) {
	for _, fx := range fixtures.All() {
		fx := fx
		t.Run(fx.Name, func(t *testing.T) {
			rt := newRuntime()

			sess, err := rt.Start(runtime.StartInput{
				Question:   fx.Question,
				Service:    fx.Service,
				TimeWindow: fx.Window,
			})
			if err != nil {
				t.Fatalf("Start: %v", err)
			}
			if len(sess.RequiredEvidence) == 0 {
				t.Fatalf("expected the planner to request evidence at start")
			}

			total := 0
			for _, batch := range fx.Batches {
				total += len(batch)
				sess, err = rt.Submit(sess.ID, batch)
				if err != nil {
					t.Fatalf("Submit: %v", err)
				}
			}

			// Leading hypothesis must match expectation.
			if len(sess.Hypotheses) == 0 {
				t.Fatalf("expected hypotheses")
			}
			if got := sess.Hypotheses[0].ID; got != fx.ExpectLeading {
				t.Errorf("leading hypothesis = %q, want %q\nfield: %s", got, fx.ExpectLeading, dumpHypotheses(sess.Hypotheses))
			}

			// Hypothesis confidences must form a probability field (~100%).
			if sum := sumConfidence(sess.Hypotheses); math.Abs(sum-100) > 1.0 {
				t.Errorf("hypothesis confidences sum to %.2f, want ~100", sum)
			}

			// Timeline references every piece of evidence.
			if len(sess.Timeline) != total {
				t.Errorf("timeline length = %d, want %d", len(sess.Timeline), total)
			}
			if !timelineSorted(sess.Timeline) {
				t.Errorf("timeline is not chronologically ordered")
			}

			// Graph must have nodes and at least one edge.
			if len(sess.Graph.Nodes()) == 0 {
				t.Errorf("expected graph nodes")
			}
			if len(sess.Graph.Edges()) == 0 {
				t.Errorf("expected graph edges")
			}

			// Confidence must be meaningful once evidence exists.
			if sess.Confidence <= 0 {
				t.Errorf("expected positive confidence, got %.2f", sess.Confidence)
			}

			// Finish must produce a usable report.
			report, finished, err := rt.Finish(sess.ID)
			if err != nil {
				t.Fatalf("Finish: %v", err)
			}
			if finished.Status != model.StatusCompleted {
				t.Errorf("status after finish = %q, want completed", finished.Status)
			}
			if report.Postmortem == "" {
				t.Errorf("expected non-empty postmortem")
			}
			if len(report.RootCauseCandidates) == 0 {
				t.Errorf("expected root cause candidates")
			}
			if report.RootCauseCandidates[0].ID != fx.ExpectLeading {
				t.Errorf("top root cause = %q, want %q", report.RootCauseCandidates[0].ID, fx.ExpectLeading)
			}
		})
	}
}

// TestProgressIncreasesAsEvidenceArrives verifies the incremental, stateful
// nature of the engine: progress and missing-evidence change as data arrives.
func TestProgressIncreasesAsEvidenceArrives(t *testing.T) {
	rt := newRuntime()
	fx := fixtures.BadDeployment()

	sess, err := rt.Start(runtime.StartInput{Question: fx.Question, Service: fx.Service, TimeWindow: fx.Window})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	startProgress := sess.Progress
	if startProgress != 0 {
		t.Errorf("expected 0 progress before any evidence, got %.2f", startProgress)
	}
	// Baseline categories the planner asks for up front must all be missing now.
	if !categoryMissing(sess.MissingEvidence, model.CategoryDeploymentEvents) {
		t.Errorf("expected deployment_events to be missing initially")
	}

	for _, batch := range fx.Batches {
		sess, err = rt.Submit(sess.ID, batch)
		if err != nil {
			t.Fatalf("Submit: %v", err)
		}
	}

	if sess.Progress <= startProgress {
		t.Errorf("progress did not increase: start=%.2f end=%.2f", startProgress, sess.Progress)
	}
	// Categories that have now been supplied must no longer be reported missing.
	for _, c := range []model.Category{model.CategoryDeploymentEvents, model.CategoryApplicationLogs, model.CategoryMetrics} {
		if categoryMissing(sess.MissingEvidence, c) {
			t.Errorf("category %q should no longer be missing after submission", c)
		}
	}
}

func categoryMissing(reqs []model.EvidenceRequest, c model.Category) bool {
	for _, r := range reqs {
		if r.Category == c {
			return true
		}
	}
	return false
}

// TestDeployAfterIncidentContradiction verifies temporal contradiction
// detection: a deployment timestamped after the incident cannot be the cause,
// and the engine should prefer "deployment unrelated".
func TestDeployAfterIncidentContradiction(t *testing.T) {
	rt := newRuntime()
	base := time.Date(2026, 6, 27, 9, 0, 0, 0, time.UTC)

	sess, err := rt.Start(runtime.StartInput{Question: "Why did the api fail?", Service: "api"})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	evidence := []*model.Evidence{
		{ID: "alert-1", Timestamp: base, Category: model.CategoryAlertEvents, Entity: "api", Summary: "Alert: api 5xx error spike"},
		{ID: "log-1", Timestamp: base.Add(time.Minute), Category: model.CategoryApplicationLogs, Entity: "api", Summary: "HTTP 500 errors on /v1"},
		{ID: "dep-1", Timestamp: base.Add(10 * time.Minute), Category: model.CategoryDeploymentEvents, Entity: "api", Summary: "Deployed api v3"},
	}
	sess, err = rt.Submit(sess.ID, evidence)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	if !hasContradiction(sess.Contradictions, "contradiction-deploy-after-incident") {
		t.Fatalf("expected deploy-after-incident contradiction, got %+v", sess.Contradictions)
	}

	depCaused := findHypothesis(sess.Hypotheses, "hypothesis-deployment-caused")
	depUnrelated := findHypothesis(sess.Hypotheses, "hypothesis-deployment-unrelated")
	if depCaused == nil || depUnrelated == nil {
		t.Fatalf("expected both deployment hypotheses present")
	}
	if depUnrelated.Confidence <= depCaused.Confidence {
		t.Errorf("expected 'deployment unrelated' (%.1f) to beat 'deployment caused' (%.1f)",
			depUnrelated.Confidence, depCaused.Confidence)
	}
}

func TestUnknownSessionErrors(t *testing.T) {
	rt := newRuntime()
	if _, err := rt.Get("does-not-exist"); err == nil {
		t.Errorf("expected error for unknown session")
	}
	if _, err := rt.Submit("does-not-exist", nil); err == nil {
		t.Errorf("expected error for submit to unknown session")
	}
}

// helpers

func sumConfidence(hs []model.Hypothesis) float64 {
	var sum float64
	for _, h := range hs {
		sum += h.Confidence
	}
	return sum
}

func timelineSorted(tl model.Timeline) bool {
	for i := 1; i < len(tl); i++ {
		if tl[i].Timestamp.Before(tl[i-1].Timestamp) {
			return false
		}
	}
	return true
}

func hasContradiction(cs []model.Contradiction, id string) bool {
	for _, c := range cs {
		if c.ID == id {
			return true
		}
	}
	return false
}

func findHypothesis(hs []model.Hypothesis, id string) *model.Hypothesis {
	for i := range hs {
		if hs[i].ID == id {
			return &hs[i]
		}
	}
	return nil
}

func dumpHypotheses(hs []model.Hypothesis) string {
	out := ""
	for _, h := range hs {
		out += "\n  " + h.ID + " " + strconv.FormatFloat(h.Confidence, 'f', 1, 64)
	}
	return out
}
