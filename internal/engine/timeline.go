package engine

import "github.com/stackrail/incident-investigator/internal/model"

// TimelineBuilder reconstructs the chronological story of the incident.
type TimelineBuilder interface {
	Build(s *model.Session) model.Timeline
}

// HeuristicTimelineBuilder orders evidence by time, one entry per observation.
type HeuristicTimelineBuilder struct{}

// NewHeuristicTimelineBuilder returns the default builder.
func NewHeuristicTimelineBuilder() *HeuristicTimelineBuilder { return &HeuristicTimelineBuilder{} }

// Build implements TimelineBuilder. Every entry references the evidence that
// supports it.
func (b *HeuristicTimelineBuilder) Build(s *model.Session) model.Timeline {
	ordered := sortedByTime(s.Evidence)
	tl := make(model.Timeline, 0, len(ordered))
	for _, e := range ordered {
		tl = append(tl, model.TimelineEntry{
			Timestamp:    e.Timestamp,
			Description:  e.Summary,
			Category:     e.Category,
			Entity:       e.Entity,
			EvidenceRefs: []string{e.ID},
		})
	}
	return tl
}
