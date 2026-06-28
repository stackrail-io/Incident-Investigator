// Package intelligencefixtures provides completed investigation snapshots for intelligence tests.
package intelligencefixtures

import (
	"fmt"
	"time"

	"github.com/stackrail/incident-investigator/internal/intelligence"
	"github.com/stackrail/incident-investigator/internal/model"
	incfix "github.com/stackrail/incident-investigator/internal/fixtures"
)

// CompletedCorpus returns 50+ completed investigation snapshots derived from
// built-in incident fixtures and deterministic variations.
func CompletedCorpus() []model.InvestigationSnapshot {
	var out []model.InvestigationSnapshot
	baseTime := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)

	for i, fx := range incfix.All() {
		variants := []struct {
			suffix   string
			service  string
			question string
			conf     float64
		}{
			{"", fx.Service, fx.Question, 75 + float64(i%10)},
			{"-b", fx.Service, fx.Question + " (prod)", 80 + float64(i%5)},
			{"-c", fx.Service + "-replica", fx.Question, 70 + float64(i%8)},
			{"-d", fx.Service, "Why did " + fx.Service + " degrade?", 85},
			{"-e", fx.Service, fx.Question, 90 - float64(i%3)},
			{"-f", fx.Service, fx.Question, 65 + float64(i%12)},
			{"-g", fx.Service + "-canary", fx.Question, 78},
			{"-h", fx.Service, fx.Question, 82},
		}
		for j, v := range variants {
			id := fmt.Sprintf("arch-%s-%d%s", fx.Name, i, v.suffix)
			s := sessionFromFixture(fx, id, v.service, v.question, v.conf, baseTime.Add(time.Duration(i*8+j)*time.Hour))
			snap := intelligence.BuildSnapshot(s)
			out = append(out, snap)
		}
	}
	return out
}

func sessionFromFixture(fx incfix.Fixture, id, service, question string, conf float64, completed time.Time) *model.Session {
	var evidence []*model.Evidence
	for _, batch := range fx.Batches {
		evidence = append(evidence, batch...)
	}
	hs := []model.Hypothesis{{
		ID: fx.ExpectLeading, Statement: fx.ExpectLeading, Confidence: conf, Status: model.StatusLeading,
	}}
	return &model.Session{
		ID: id, Question: question, Service: service, Goal: model.GoalRootCause,
		Status: model.StatusCompleted, State: model.StateCompleted,
		Evidence: evidence, Hypotheses: hs, Confidence: conf,
		UpdatedAt: completed, CreatedAt: completed.Add(-time.Hour),
		Graph: model.NewEmptyGraphView(),
	}
}

// LoadCorpusIntoArchive stores the corpus into an archive.
func LoadCorpusIntoArchive(archive intelligence.InvestigationArchive) {
	for _, snap := range CompletedCorpus() {
		s := snap
		_ = archive.Store(&model.CompletedInvestigation{Snapshot: s})
	}
}
