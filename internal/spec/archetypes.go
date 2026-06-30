package spec

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ArchetypesDocument is the parsed investigation-v1 archetypes contract.
type ArchetypesDocument struct {
	Components struct {
		Schemas map[string]any `yaml:"schemas"`
	} `yaml:"components"`
	Archetypes []ArchetypeSpec `yaml:"x-investigation-archetypes"`
}

// ArchetypeSpec is one normative failure-mode archetype.
type ArchetypeSpec struct {
	ID                 string   `yaml:"id"`
	Name               string   `yaml:"name"`
	Domain             string   `yaml:"domain"`
	HypothesisID       string   `yaml:"hypothesis_id"`
	Priority           int      `yaml:"priority"`
	TaxonomyRank       int      `yaml:"taxonomy_rank,omitempty"`
	SignalTriggers     []string `yaml:"signal_triggers,omitempty"`
	ExpectedEvidence   []string `yaml:"expected_evidence,omitempty"`
	TypicalSubcauses   []string `yaml:"typical_subcauses,omitempty"`
	ConformanceFixture string   `yaml:"conformance_fixture"`
	Notes              string   `yaml:"notes,omitempty"`
}

// LoadArchetypes reads spec/investigation-v1/archetypes.yaml from repoRoot.
func LoadArchetypes(repoRoot string) (*ArchetypesDocument, error) {
	path := filepath.Join(repoRoot, "spec", "investigation-v1", "archetypes.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read archetypes.yaml: %w", err)
	}
	var doc ArchetypesDocument
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse archetypes.yaml: %w", err)
	}
	if len(doc.Archetypes) == 0 {
		return nil, fmt.Errorf("archetypes.yaml: x-investigation-archetypes is empty")
	}
	return &doc, nil
}

// ArchetypeFixturesDir returns the path to per-archetype conformance fixtures.
func ArchetypeFixturesDir(repoRoot string) string {
	return filepath.Join(repoRoot, "spec", "investigation-v1", "conformance", "archetype-fixtures")
}
