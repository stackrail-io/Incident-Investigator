package fixtures

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/stackrail/incident-investigator/internal/model"
	"github.com/stackrail/incident-investigator/internal/spec"
)

var (
	repoRootOnce sync.Once
	repoRoot     string
	repoRootErr  error
)

func getRepoRoot() (string, error) {
	repoRootOnce.Do(func() {
		_, file, _, ok := runtime.Caller(0)
		if !ok {
			repoRootErr = fmt.Errorf("fixtures: cannot locate source file")
			return
		}
		repoRoot = filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	})
	return repoRoot, repoRootErr
}

// RepoRoot returns the repository root for tests and tooling.
func RepoRoot() (string, error) {
	return getRepoRoot()
}

// FromConformance converts a spec conformance scenario into a replayable Fixture.
func FromConformance(fx *spec.ConformanceFixture) (Fixture, error) {
	if fx == nil {
		return Fixture{}, fmt.Errorf("nil conformance fixture")
	}
	name := fx.ArchetypeID
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(fx.ScenarioID), ".yaml")
	}
	out := Fixture{
		Name:          strings.ReplaceAll(name, "-", "_"),
		Question:      fx.Start.Question,
		Service:       fx.Start.Service,
		ExpectLeading: fx.ExpectAfterAllEvidence.LeadingHypothesisID,
	}
	if fx.Start.TimeWindow.Start != "" {
		start, err := time.Parse(time.RFC3339, fx.Start.TimeWindow.Start)
		if err != nil {
			return Fixture{}, fmt.Errorf("parse time_window.start: %w", err)
		}
		out.Window.Start = start
	}
	if fx.Start.TimeWindow.End != "" {
		end, err := time.Parse(time.RFC3339, fx.Start.TimeWindow.End)
		if err != nil {
			return Fixture{}, fmt.Errorf("parse time_window.end: %w", err)
		}
		out.Window.End = end
	}
	for _, batch := range fx.EvidenceBatches {
		var evBatch []*model.Evidence
		for _, item := range batch {
			ts, err := time.Parse(time.RFC3339, item.Timestamp)
			if err != nil {
				return Fixture{}, fmt.Errorf("parse evidence %q timestamp: %w", item.ID, err)
			}
			evBatch = append(evBatch, &model.Evidence{
				ID:        item.ID,
				Timestamp: ts,
				Category:  model.Category(item.Category),
				Source:    "provided_by_client",
				Entity:    item.Entity,
				Summary:   item.Summary,
				Payload:   item.Payload,
			})
		}
		out.Batches = append(out.Batches, evBatch)
	}
	return out, nil
}

// LoadByArchetypeID loads the conformance fixture declared for an archetype in archetypes.yaml.
func LoadByArchetypeID(archetypeID string) (Fixture, error) {
	root, err := getRepoRoot()
	if err != nil {
		return Fixture{}, err
	}
	doc, err := spec.LoadArchetypes(root)
	if err != nil {
		return Fixture{}, err
	}
	for _, arch := range doc.Archetypes {
		if arch.ID != archetypeID {
			continue
		}
		fx, err := spec.LoadConformanceFixture(filepath.Join(root, arch.ConformanceFixture))
		if err != nil {
			return Fixture{}, err
		}
		return FromConformance(fx)
	}
	return Fixture{}, fmt.Errorf("archetype %q not found in spec", archetypeID)
}

// AllFromSpec returns every archetype conformance scenario from archetypes.yaml.
func AllFromSpec() ([]Fixture, error) {
	root, err := getRepoRoot()
	if err != nil {
		return nil, err
	}
	doc, err := spec.LoadArchetypes(root)
	if err != nil {
		return nil, err
	}
	out := make([]Fixture, 0, len(doc.Archetypes))
	for _, arch := range doc.Archetypes {
		fx, err := spec.LoadConformanceFixture(filepath.Join(root, arch.ConformanceFixture))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", arch.ID, err)
		}
		fix, err := FromConformance(fx)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", arch.ID, err)
		}
		out = append(out, fix)
	}
	return out, nil
}

func mustLoad(archetypeID string) Fixture {
	fx, err := LoadByArchetypeID(archetypeID)
	if err != nil {
		panic(fmt.Sprintf("fixtures: load %q: %v", archetypeID, err))
	}
	return fx
}
