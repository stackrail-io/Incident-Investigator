package intelligence

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/stackrail/incident-investigator/internal/model"
)

// BuildSnapshot creates an immutable snapshot from a completed session.
func BuildSnapshot(s *model.Session) model.InvestigationSnapshot {
	snap := model.InvestigationSnapshot{
		InvestigationID: s.ID,
		Goal:            s.Goal,
		Timeline:        s.Timeline,
		KnowledgeGraph:  s.Graph,
		Hypotheses:      append([]model.Hypothesis(nil), s.Hypotheses...),
		Confidence:      s.Confidence,
		CompletedAt:     s.UpdatedAt,
		Metadata:        map[string]any{},
	}
	if s.Plan != nil {
		snap.Questions = append([]model.Question(nil), s.Plan.Questions...)
	}
	if s.Service != "" {
		snap.Metadata["service"] = s.Service
	}
	if s.Question != "" {
		snap.Metadata["question"] = s.Question
	}
	if len(s.BlastRadius.Services) > 0 {
		snap.Metadata["blast_services"] = s.BlastRadius.Services
	}
	snap.EvidenceSummary = summarizeEvidence(s.Evidence)
	snap.RootCause = leadingHypothesisID(s.Hypotheses)
	snap.Fingerprint = ComputeFingerprint(&snap)
	return snap
}

func summarizeEvidence(evidence []*model.Evidence) []model.EvidenceSummary {
	counts := map[model.Category]int{}
	samples := map[model.Category]string{}
	for _, e := range evidence {
		if e == nil {
			continue
		}
		counts[e.Category]++
		if samples[e.Category] == "" {
			samples[e.Category] = e.Summary
		}
	}
	var cats []model.Category
	for c := range counts {
		cats = append(cats, c)
	}
	sort.Slice(cats, func(i, j int) bool { return cats[i] < cats[j] })
	out := make([]model.EvidenceSummary, 0, len(cats))
	for _, c := range cats {
		out = append(out, model.EvidenceSummary{
			Category: c, Count: counts[c], SampleSummary: samples[c],
		})
	}
	return out
}

func leadingHypothesisID(hs []model.Hypothesis) string {
	if len(hs) == 0 {
		return ""
	}
	return hs[0].ID
}

// ComputeFingerprint derives a deterministic fingerprint from a snapshot.
func ComputeFingerprint(s *model.InvestigationSnapshot) model.InvestigationFingerprint {
	fp := model.InvestigationFingerprint{
		Goal:      string(s.Goal),
		RootCause: s.RootCause,
	}
	if s.Metadata != nil {
		if svc, ok := s.Metadata["service"].(string); ok && svc != "" {
			fp.Services = []string{svc}
		}
	}
	for _, sum := range s.EvidenceSummary {
		fp.Categories = append(fp.Categories, sum.Category)
	}
	sort.Slice(fp.Categories, func(i, j int) bool { return fp.Categories[i] < fp.Categories[j] })
	fp.GraphHash = hashGraph(s.KnowledgeGraph)
	fp.TimelineHash = hashTimeline(s.Timeline)
	return fp
}

func hashGraph(g *model.GraphView) string {
	if g == nil {
		return ""
	}
	var parts []string
	for _, e := range g.Edges {
		if e != nil {
			parts = append(parts, string(e.Type)+":"+e.From+"->"+e.To)
		}
	}
	sort.Strings(parts)
	return hashStrings(parts)
}

func hashTimeline(t model.Timeline) string {
	var parts []string
	for _, e := range t {
		parts = append(parts, fmt.Sprintf("%s:%s", e.Timestamp.Format("15:04"), string(e.Category)))
	}
	return hashStrings(parts)
}

func hashStrings(parts []string) string {
	h := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(h[:8])
}

func categoriesFromSnapshot(s *model.InvestigationSnapshot) map[model.Category]bool {
	out := map[model.Category]bool{}
	for _, sum := range s.EvidenceSummary {
		out[sum.Category] = true
	}
	return out
}

func hypothesisSet(s *model.InvestigationSnapshot) map[string]bool {
	out := map[string]bool{}
	for _, h := range s.Hypotheses {
		out[h.ID] = true
	}
	return out
}

func graphEdgeTypes(g *model.GraphView) map[model.EdgeType]bool {
	out := map[model.EdgeType]bool{}
	if g == nil {
		return out
	}
	for _, e := range g.Edges {
		if e != nil {
			out[e.Type] = true
		}
	}
	return out
}

func timelineCategories(t model.Timeline) []model.Category {
	var out []model.Category
	for _, e := range t {
		out = append(out, e.Category)
	}
	return out
}
