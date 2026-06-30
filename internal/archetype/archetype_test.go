package archetype_test

import (
	"testing"
	"time"

	"github.com/stackrail/incident-investigator/internal/archetype"
	"github.com/stackrail/incident-investigator/internal/archetype/builtin"
	"github.com/stackrail/incident-investigator/internal/engine"
	"github.com/stackrail/incident-investigator/internal/fixtures"
	"github.com/stackrail/incident-investigator/internal/model"
	"github.com/stackrail/incident-investigator/internal/spec"
)

func TestDefaultRegistryMatchesArchetypeSpec(t *testing.T) {
	root, err := fixtures.RepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	doc, err := spec.LoadArchetypes(root)
	if err != nil {
		t.Fatal(err)
	}

	reg := builtin.DefaultRegistry()
	if got := len(reg.All()); got != len(doc.Archetypes) {
		t.Fatalf("archetype count: registry=%d spec=%d", got, len(doc.Archetypes))
	}
	if reg.ByID("deployment-failure") == nil {
		t.Fatal("expected deployment-failure archetype")
	}
	if reg.ByID("unknown-novel") == nil {
		t.Fatal("expected unknown-novel archetype")
	}
}

func TestRegistrySeedQuestions(t *testing.T) {
	reg := builtin.DefaultRegistry()
	seeds := reg.SeedQuestions()

	wantQuestions := 0
	for _, a := range reg.All() {
		wantQuestions += len(a.SeedQuestions())
	}
	if len(seeds) < wantQuestions {
		t.Fatalf("got %d seeded questions, want at least %d", len(seeds), wantQuestions)
	}

	ids := map[string]bool{}
	for _, q := range seeds {
		if ids[q.ID] {
			t.Errorf("duplicate question id %q", q.ID)
		}
		ids[q.ID] = true
	}
	for _, want := range []string{"deploy-before-errors", "lock-contention-queue", "certificate-expired", "dependency-unavailable", "vendor-outage"} {
		if !ids[want] {
			t.Errorf("missing seeded question %q", want)
		}
	}
}

func TestRegistryScoreMatchesHypothesisEngine(t *testing.T) {
	for _, fx := range fixtures.All() {
		fx := fx
		t.Run(fx.Name, func(t *testing.T) {
			var ev []*model.Evidence
			for _, batch := range fx.Batches {
				ev = append(ev, batch...)
			}
			s := &model.Session{
				ID: fx.Name, Question: fx.Question, Service: fx.Service, Evidence: ev,
				Graph: model.NewEmptyGraphView(),
			}
			sig := engine.Analyze(s)
			contradictions := engine.NewHeuristicContradictionDetector().Detect(s, sig)

			viaEngine := engine.NewHeuristicHypothesisEngine().Generate(s, sig, contradictions)
			viaRegistry := builtin.DefaultRegistry().Score(archetype.ScoreContext{
				Session: s, Signals: sig, Contradictions: contradictions,
			})

			if len(viaEngine) != len(viaRegistry) {
				t.Fatalf("hypothesis count mismatch: engine=%d registry=%d", len(viaEngine), len(viaRegistry))
			}
			if viaEngine[0].ID != viaRegistry[0].ID {
				t.Errorf("leading mismatch: engine=%q registry=%q", viaEngine[0].ID, viaRegistry[0].ID)
			}
			if viaEngine[0].Confidence != viaRegistry[0].Confidence {
				t.Errorf("leading confidence mismatch: engine=%.1f registry=%.1f",
					viaEngine[0].Confidence, viaRegistry[0].Confidence)
			}
		})
	}
}

type customProbe struct{}

func (customProbe) ID() string               { return "custom-probe" }
func (customProbe) Name() string             { return "Custom Probe" }
func (customProbe) Domain() archetype.Domain { return archetype.DomainGeneric }
func (customProbe) Priority() int            { return 1 }
func (customProbe) HypothesisID() string     { return "hypothesis-custom-probe" }
func (customProbe) Applicable(archetype.ScoreContext) bool { return true }
func (customProbe) ExpectedEvidence() []model.Category     { return nil }
func (customProbe) TypicalSubcauses() []string               { return nil }
func (customProbe) Score(archetype.ScoreContext) archetype.Candidate {
	return archetype.Candidate{
		HypothesisID: "hypothesis-custom-probe",
		Statement:    "Custom archetype fired.",
		Score:        50,
	}
}
func (customProbe) SeedQuestions() []archetype.QuestionSeed { return nil }

func TestCustomArchetypeRegistration(t *testing.T) {
	reg := builtin.DefaultRegistry()
	reg.Register(customProbe{})

	s := &model.Session{ID: "t", Question: "q?", Evidence: []*model.Evidence{
		{ID: "e1", Timestamp: time.Now(), Category: model.CategoryApplicationLogs, Summary: "error"},
	}}
	sig := engine.Analyze(s)
	hyps := reg.Score(archetype.ScoreContext{Session: s, Signals: sig})

	found := false
	for _, h := range hyps {
		if h.ID == "hypothesis-custom-probe" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected custom archetype in scored field")
	}
}
