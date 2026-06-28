package runtime_test

import (
	"context"
	"testing"

	"github.com/stackrail/incident-investigator/internal/fixtures"
	"github.com/stackrail/incident-investigator/internal/intelligence"
	"github.com/stackrail/incident-investigator/internal/runtime"
)

func TestIntelligenceFindSimilarAfterFinish(t *testing.T) {
	intel := intelligence.NewMemoryService()
	rt := runtime.New(
		runtime.WithClock(fixedClock()),
		runtime.WithIntelligence(intel),
	)

	// Archive two completed investigations.
	for _, fx := range []fixtures.Fixture{fixtures.BadDeployment(), fixtures.DatabaseOutage()} {
		sess, err := rt.Start(runtime.StartInput{
			Question: fx.Question, Service: fx.Service, TimeWindow: fx.Window,
		})
		if err != nil {
			t.Fatal(err)
		}
		for _, batch := range fx.Batches {
			if _, err := rt.Submit(sess.ID, batch); err != nil {
				t.Fatal(err)
			}
		}
		if _, _, err := rt.Finish(sess.ID); err != nil {
			t.Fatal(err)
		}
	}

	// New investigation similar to bad deployment.
	sess, err := rt.Start(runtime.StartInput{
		Question:   "Why did checkout fail yesterday?",
		Service:    "checkout-api",
		TimeWindow: fixtures.BadDeployment().Window,
	})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := rt.FindSimilarInvestigations(context.Background(), sess.ID, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Matches) == 0 {
		t.Fatal("expected similar archived investigations")
	}

	patterns, err := rt.SuggestPatterns(context.Background(), sess.ID, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(patterns.Patterns) == 0 {
		t.Fatal("expected suggested patterns from archive")
	}
}
