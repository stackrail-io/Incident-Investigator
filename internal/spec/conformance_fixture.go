package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ConformanceFixture is a portable end-to-end investigation scenario.
// Archetype fixtures use YAML; legacy core fixtures may remain JSON.
type ConformanceFixture struct {
	ArchetypeID  string `yaml:"archetype_id" json:"archetype_id,omitempty"`
	ScenarioID   string `yaml:"scenario_id" json:"scenario_id"`
	Tier         string `yaml:"tier" json:"tier"`
	Description  string `yaml:"description" json:"description"`
	Start        struct {
		Question   string `yaml:"question" json:"question"`
		Service    string `yaml:"service" json:"service"`
		Goal       string `yaml:"goal" json:"goal"`
		TimeWindow struct {
			Start string `yaml:"start" json:"start"`
			End   string `yaml:"end" json:"end"`
		} `yaml:"time_window" json:"time_window"`
	} `yaml:"start" json:"start"`
	ExpectAfterStart struct {
		RequiredEvidenceMin int `yaml:"required_evidence_min" json:"required_evidence_min"`
		PlanQuestionsMin    int `yaml:"plan_questions_min" json:"plan_questions_min"`
	} `yaml:"expect_after_start" json:"expect_after_start"`
	EvidenceBatches    [][]ConformanceEvidence `yaml:"evidence_batches" json:"evidence_batches"`
	ExpectAfterAllEvidence struct {
		LeadingHypothesisID string   `yaml:"leading_hypothesis_id" json:"leading_hypothesis_id"`
		MinConfidence       float64  `yaml:"min_confidence" json:"min_confidence"`
		MinLeadMargin       float64  `yaml:"min_lead_margin,omitempty" json:"min_lead_margin,omitempty"`
		MustNotLead         []string `yaml:"must_not_lead,omitempty" json:"must_not_lead,omitempty"`
		MinHypotheses       int      `yaml:"min_hypotheses" json:"min_hypotheses"`
		GraphNodesMin       int      `yaml:"graph_nodes_min" json:"graph_nodes_min"`
	} `yaml:"expect_after_all_evidence" json:"expect_after_all_evidence"`
	ExpectAfterFinish struct {
		State                string   `yaml:"state" json:"state"`
		ReportRequiredFields []string `yaml:"report_required_fields" json:"report_required_fields"`
	} `yaml:"expect_after_finish" json:"expect_after_finish"`
}

// ConformanceEvidence is one evidence item inside a conformance batch.
type ConformanceEvidence struct {
	ID        string         `yaml:"id" json:"id"`
	Timestamp string         `yaml:"timestamp" json:"timestamp"`
	Category  string         `yaml:"category" json:"category"`
	Entity    string         `yaml:"entity" json:"entity"`
	Summary   string         `yaml:"summary" json:"summary"`
	Payload   map[string]any `yaml:"payload,omitempty" json:"payload,omitempty"`
}

// LoadConformanceFixture reads a YAML or JSON conformance scenario.
func LoadConformanceFixture(path string) (*ConformanceFixture, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var fx ConformanceFixture
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &fx); err != nil {
			return nil, fmt.Errorf("parse yaml fixture %s: %w", path, err)
		}
	case ".json":
		if err := json.Unmarshal(data, &fx); err != nil {
			return nil, fmt.Errorf("parse json fixture %s: %w", path, err)
		}
	default:
		return nil, fmt.Errorf("unsupported fixture format: %s", path)
	}
	return &fx, nil
}
