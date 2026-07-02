package export

import (
	"time"

	"github.com/stackrail/incident-investigator/internal/model"
)

// FromSession builds an export snapshot from runtime session state.
func FromSession(s *model.Session, report *model.Report) *Snapshot {
	if s == nil {
		return nil
	}
	snap := &Snapshot{
		InvestigationID: s.ID,
		Question:        s.Question,
		Service:         s.Service,
		State:           string(s.State),
		Confidence:      s.Confidence,
		ExportedAt:      time.Now().UTC(),
	}
	for _, e := range s.Evidence {
		if e == nil {
			continue
		}
		snap.Evidence = append(snap.Evidence, Evidence{
			ID: e.ID, Timestamp: e.Timestamp, Category: string(e.Category),
			Entity: e.Entity, Summary: e.Summary,
		})
	}
	for _, h := range s.Hypotheses {
		snap.Hypotheses = append(snap.Hypotheses, Hypothesis{
			ID: h.ID, Statement: h.Statement, Confidence: h.Confidence,
			Status: string(h.Status), Rationale: h.Rationale,
		})
	}
	for _, e := range s.Timeline {
		snap.Timeline = append(snap.Timeline, TimelineEntry{
			Timestamp: e.Timestamp, Description: e.Description,
			Category: string(e.Category), Entity: e.Entity, EvidenceRefs: e.EvidenceRefs,
		})
	}
	if s.Graph != nil {
		for _, n := range s.Graph.Nodes {
			snap.Graph.Nodes = append(snap.Graph.Nodes, GraphNode{ID: n.ID, Type: string(n.Type), Label: n.Label})
		}
		for _, e := range s.Graph.Edges {
			snap.Graph.Edges = append(snap.Graph.Edges, GraphEdge{From: e.From, To: e.To, Type: string(e.Type)})
		}
	}
	if report != nil {
		snap.Report = &Report{
			ExecutiveSummary: report.ExecutiveSummary,
			Confidence:       report.Confidence,
			Postmortem:       report.Postmortem,
			Recommendations:  report.Recommendations,
		}
	}
	return snap
}
