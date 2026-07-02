package examples

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/stackrail/incident-investigator/internal/fixtures"
	"github.com/stackrail/incident-investigator/internal/model"
)

// Investigation describes how to start an example scenario.
type Investigation struct {
	Description string `json:"description"`
	Question    string `json:"question"`
	Service     string `json:"service"`
	Goal        string `json:"goal"`
	TimeWindow  struct {
		Start string `json:"start"`
		End   string `json:"end"`
	} `json:"time_window"`
}

// EvidenceBatch is one submit_evidence call worth of items.
type EvidenceBatch struct {
	Batch       string           `json:"batch"`
	Description string           `json:"description"`
	Evidence    []EvidenceItem   `json:"evidence"`
}

// EvidenceItem is the MCP-facing evidence shape used in example JSON files.
type EvidenceItem struct {
	ID        string         `json:"id"`
	Timestamp string         `json:"timestamp"`
	Category  string         `json:"category"`
	Source    string         `json:"source,omitempty"`
	Entity    string         `json:"entity"`
	Summary   string         `json:"summary"`
	Payload   map[string]any `json:"payload,omitempty"`
}

// Scenario is a loaded example ready to replay through the runtime.
type Scenario struct {
	Name          string
	Investigation Investigation
	Batches       [][]*model.Evidence
}

// Load reads examples/<name>/investigation.json and evidence/*.json.
func Load(repoRoot, name string) (*Scenario, error) {
	dir := filepath.Join(repoRoot, "examples", name)
	invPath := filepath.Join(dir, "investigation.json")
	invData, err := os.ReadFile(invPath)
	if err != nil {
		return nil, fmt.Errorf("read investigation.json: %w", err)
	}
	var inv Investigation
	if err := json.Unmarshal(invData, &inv); err != nil {
		return nil, fmt.Errorf("parse investigation.json: %w", err)
	}
	evDir := filepath.Join(dir, "evidence")
	entries, err := os.ReadDir(evDir)
	if err != nil {
		return nil, fmt.Errorf("read evidence dir: %w", err)
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		files = append(files, e.Name())
	}
	sort.Strings(files)
	if len(files) == 0 {
		return nil, fmt.Errorf("no evidence/*.json in %s", dir)
	}
	var batches [][]*model.Evidence
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(evDir, f))
		if err != nil {
			return nil, err
		}
		var batch EvidenceBatch
		if err := json.Unmarshal(data, &batch); err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
		ev, err := toModelEvidence(batch.Evidence)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
		batches = append(batches, ev)
	}
	return &Scenario{Name: name, Investigation: inv, Batches: batches}, nil
}

// StartInput converts the investigation metadata to runtime input.
func (s *Scenario) StartInput() (model.InvestigationGoal, model.TimeWindow, error) {
	if s == nil {
		return "", model.TimeWindow{}, fmt.Errorf("nil scenario")
	}
	goal := model.InvestigationGoal(s.Investigation.Goal)
	if goal == "" {
		goal = model.GoalRootCause
	}
	if !goal.Valid() {
		return "", model.TimeWindow{}, fmt.Errorf("unknown goal %q", s.Investigation.Goal)
	}
	var window model.TimeWindow
	if s.Investigation.TimeWindow.Start != "" {
		start, err := time.Parse(time.RFC3339, s.Investigation.TimeWindow.Start)
		if err != nil {
			return "", model.TimeWindow{}, fmt.Errorf("time_window.start: %w", err)
		}
		window.Start = start
	}
	if s.Investigation.TimeWindow.End != "" {
		end, err := time.Parse(time.RFC3339, s.Investigation.TimeWindow.End)
		if err != nil {
			return "", model.TimeWindow{}, fmt.Errorf("time_window.end: %w", err)
		}
		window.End = end
	}
	return goal, window, nil
}

func toModelEvidence(items []EvidenceItem) ([]*model.Evidence, error) {
	out := make([]*model.Evidence, 0, len(items))
	for _, item := range items {
		ts, err := time.Parse(time.RFC3339, item.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("evidence %q timestamp: %w", item.ID, err)
		}
		src := item.Source
		if src == "" {
			src = "provided_by_client"
		}
		out = append(out, &model.Evidence{
			ID:        item.ID,
			Timestamp: ts,
			Category:  model.Category(item.Category),
			Source:    src,
			Entity:    item.Entity,
			Summary:   item.Summary,
			Payload:   item.Payload,
		})
	}
	return out, nil
}

// RepoRoot delegates to fixtures.RepoRoot.
func RepoRoot() (string, error) {
	return fixtures.RepoRoot()
}
