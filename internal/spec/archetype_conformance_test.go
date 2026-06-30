package spec_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stackrail/incident-investigator/internal/archetype"
	"github.com/stackrail/incident-investigator/internal/archetype/builtin"
	"github.com/stackrail/incident-investigator/internal/spec"
)

func TestArchetypesYAMLMatchesBuiltinRegistry(t *testing.T) {
	root := repoRoot(t)
	doc, err := spec.LoadArchetypes(root)
	if err != nil {
		t.Fatal(err)
	}

	reg := builtin.DefaultRegistry()
	byID := map[string]archetype.Archetype{}
	for _, a := range reg.All() {
		byID[a.ID()] = a
	}

	if len(doc.Archetypes) != len(reg.All()) {
		t.Fatalf("archetype count: spec=%d registry=%d", len(doc.Archetypes), len(reg.All()))
	}

	seen := map[string]bool{}
	for _, want := range doc.Archetypes {
		if seen[want.ID] {
			t.Fatalf("duplicate archetype id in spec: %q", want.ID)
		}
		seen[want.ID] = true

		got := byID[want.ID]
		if got == nil {
			t.Fatalf("registry missing archetype %q from spec", want.ID)
		}
		if got.Name() != want.Name {
			t.Errorf("%s name: registry=%q spec=%q", want.ID, got.Name(), want.Name)
		}
		if string(got.Domain()) != want.Domain {
			t.Errorf("%s domain: registry=%q spec=%q", want.ID, got.Domain(), want.Domain)
		}
		if got.HypothesisID() != want.HypothesisID {
			t.Errorf("%s hypothesis_id: registry=%q spec=%q", want.ID, got.HypothesisID(), want.HypothesisID)
		}
		if got.Priority() != want.Priority {
			t.Errorf("%s priority: registry=%d spec=%d", want.ID, got.Priority(), want.Priority)
		}
		if want.ConformanceFixture == "" {
			t.Errorf("%s: missing conformance_fixture in spec", want.ID)
			continue
		}
		fixPath := filepath.Join(root, want.ConformanceFixture)
		if _, err := os.Stat(fixPath); err != nil {
			t.Errorf("%s conformance fixture missing at %s: %v", want.ID, want.ConformanceFixture, err)
		}
		ext := filepath.Ext(want.ConformanceFixture)
		if ext != ".yaml" && ext != ".yml" {
			t.Errorf("%s: archetype conformance fixtures must be YAML, got %q", want.ID, ext)
		}
	}
}

func TestArchetypeConformanceFixtures(t *testing.T) {
	root := repoRoot(t)
	doc, err := spec.LoadArchetypes(root)
	if err != nil {
		t.Fatal(err)
	}

	for _, arch := range doc.Archetypes {
		arch := arch
		t.Run(arch.ID, func(t *testing.T) {
			path := filepath.Join(root, arch.ConformanceFixture)
			runArchetypeConformanceFixture(t, path, arch.ID, arch.HypothesisID)
		})
	}
}

func runArchetypeConformanceFixture(t *testing.T, path, archetypeID, hypothesisID string) {
	t.Helper()
	fx, err := spec.LoadConformanceFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	if fx.ArchetypeID != archetypeID {
		t.Fatalf("fixture archetype_id=%q, spec=%q", fx.ArchetypeID, archetypeID)
	}
	if fx.ExpectAfterAllEvidence.LeadingHypothesisID != hypothesisID {
		t.Fatalf("fixture leading_hypothesis_id=%q, spec hypothesis_id=%q",
			fx.ExpectAfterAllEvidence.LeadingHypothesisID, hypothesisID)
	}
	runConformanceFixtureData(t, fx)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	return filepath.Join(filepath.Dir(file), "..", "..")
}
