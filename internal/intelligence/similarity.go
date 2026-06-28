package intelligence

import (
	"context"
	"math"
	"sort"
	"strings"

	"github.com/stackrail/incident-investigator/internal/model"
)

// SimilarityEngine finds historically similar investigations.
type SimilarityEngine interface {
	FindSimilar(ctx context.Context, req model.SimilarityRequest, archive InvestigationArchive, limit int) ([]model.SimilarityResult, error)
}

// HeuristicSimilarityEngine uses deterministic feature comparison.
type HeuristicSimilarityEngine struct{}

// NewHeuristicSimilarityEngine returns the default similarity engine.
func NewHeuristicSimilarityEngine() *HeuristicSimilarityEngine {
	return &HeuristicSimilarityEngine{}
}

// FindSimilar implements SimilarityEngine.
func (e *HeuristicSimilarityEngine) FindSimilar(_ context.Context, req model.SimilarityRequest, archive InvestigationArchive, limit int) ([]model.SimilarityResult, error) {
	if limit <= 0 {
		limit = 5
	}
	wantCats := map[model.Category]bool{}
	for _, c := range req.EvidenceCategories {
		wantCats[c] = true
	}
	wantHyps := map[string]bool{}
	for _, h := range req.HypothesisIDs {
		wantHyps[h] = true
	}
	if req.LeadingHypothesis != "" {
		wantHyps[req.LeadingHypothesis] = true
	}

	var curGraph *model.GraphView
	var curTimeline model.Timeline
	if req.Session != nil {
		curGraph = req.Session.Graph
		curTimeline = req.Session.Timeline
		if len(wantCats) == 0 {
			wantCats = sessionCategories(req.Session)
		}
	}

	var results []model.SimilarityResult
	for _, snap := range archive.All() {
		if snap.InvestigationID == req.SessionID {
			continue
		}
		score, reasons := scoreSnapshot(req, snap, wantCats, wantHyps, curGraph, curTimeline)
		if score < 0.2 {
			continue
		}
		svc := serviceFromSnapshot(snap)
		results = append(results, model.SimilarityResult{
			InvestigationID: snap.InvestigationID,
			Score:           round1(score * 100),
			MatchingReasons: reasons,
			Question:        questionFromSnapshot(snap),
			Service:         svc,
			Goal:            snap.Goal,
			RootCause:       snap.RootCause,
			Confidence:      snap.Confidence,
			CompletedAt:     snap.CompletedAt,
		})
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	if len(results) > limit {
		results = results[:limit]
	}
	if results == nil {
		results = []model.SimilarityResult{}
	}
	return results, nil
}

func scoreSnapshot(req model.SimilarityRequest, snap *model.InvestigationSnapshot, wantCats map[model.Category]bool, wantHyps map[string]bool, curGraph *model.GraphView, curTimeline model.Timeline) (float64, []string) {
	var score float64
	var reasons []string

	q := questionFromSnapshot(snap)
	qo := questionOverlap(req.Question, q)
	score += qo * 0.2
	if qo > 0.25 {
		reasons = append(reasons, "question similarity")
	}
	if req.Goal != "" && snap.Goal == req.Goal {
		score += 0.12
		reasons = append(reasons, "same goal")
	}
	svc := serviceFromSnapshot(snap)
	if req.Service != "" && svc != "" && strings.EqualFold(req.Service, svc) {
		score += 0.15
		reasons = append(reasons, "same service")
	}
	cj := jaccard(wantCats, categoriesFromSnapshot(snap))
	score += cj * 0.15
	if cj > 0.3 {
		reasons = append(reasons, "evidence categories")
	}
	gj := jaccardEdgeTypes(graphEdgeTypes(curGraph), graphEdgeTypes(snap.KnowledgeGraph))
	score += gj * 0.12
	if gj > 0.25 {
		reasons = append(reasons, "graph topology")
	}
	tj := timelineSimilarity(curTimeline, snap.Timeline)
	score += tj * 0.1
	if tj > 0.3 {
		reasons = append(reasons, "timeline shape")
	}
	hj := jaccardStringSet(wantHyps, hypothesisSet(snap))
	score += hj * 0.1
	if hj > 0.3 {
		reasons = append(reasons, "hypothesis graph")
	}
	if req.LeadingHypothesis != "" && snap.RootCause == req.LeadingHypothesis {
		score += 0.08
		reasons = append(reasons, "root cause match")
	}
	if fpMatch(req, snap) {
		score += 0.08
		reasons = append(reasons, "fingerprint partial match")
	}
	return math.Min(1, score), reasons
}

func questionFromSnapshot(s *model.InvestigationSnapshot) string {
	if s == nil || s.Metadata == nil {
		return ""
	}
	if q, ok := s.Metadata["question"].(string); ok {
		return q
	}
	return ""
}

func fpMatch(req model.SimilarityRequest, snap *model.InvestigationSnapshot) bool {
	if snap == nil {
		return false
	}
	if req.Goal != "" && string(req.Goal) == snap.Fingerprint.Goal {
		return true
	}
	if req.LeadingHypothesis != "" && req.LeadingHypothesis == snap.Fingerprint.RootCause {
		return true
	}
	return false
}

func jaccardEdgeTypes(a, b map[model.EdgeType]bool) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}
	var inter, union int
	seen := map[model.EdgeType]bool{}
	for k := range a {
		seen[k] = true
		union++
		if b[k] {
			inter++
		}
	}
	for k := range b {
		if !seen[k] {
			union++
		}
	}
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

func jaccardStringSet(a map[string]bool, b map[string]bool) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}
	var inter, union int
	seen := map[string]bool{}
	for k := range a {
		seen[k] = true
		union++
		if b[k] {
			inter++
		}
	}
	for k := range b {
		if !seen[k] {
			union++
		}
	}
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

func timelineSimilarity(a, b model.Timeline) float64 {
	ca := timelineCategories(a)
	cb := timelineCategories(b)
	if len(ca) == 0 || len(cb) == 0 {
		return 0
	}
	setA := map[model.Category]bool{}
	for _, c := range ca {
		setA[c] = true
	}
	var inter int
	for _, c := range cb {
		if setA[c] {
			inter++
		}
	}
	denom := len(ca)
	if len(cb) > denom {
		denom = len(cb)
	}
	return float64(inter) / float64(denom)
}

func sessionCategories(s *model.Session) map[model.Category]bool {
	out := map[model.Category]bool{}
	for _, e := range s.Evidence {
		if e != nil {
			out[e.Category] = true
		}
	}
	return out
}

func tokenizeQuestion(q string) map[string]bool {
	tokens := map[string]bool{}
	for _, w := range strings.Fields(strings.ToLower(q)) {
		w = strings.Trim(w, "?!.,\"'")
		if len(w) > 2 {
			tokens[w] = true
		}
	}
	return tokens
}

func questionOverlap(a, b string) float64 {
	ta, tb := tokenizeQuestion(a), tokenizeQuestion(b)
	if len(ta) == 0 || len(tb) == 0 {
		return 0
	}
	var inter int
	for k := range ta {
		if tb[k] {
			inter++
		}
	}
	denom := len(ta)
	if len(tb) > denom {
		denom = len(tb)
	}
	return float64(inter) / float64(denom)
}

func jaccard(a, b map[model.Category]bool) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}
	var inter, union int
	seen := map[model.Category]bool{}
	for k := range a {
		seen[k] = true
		union++
		if b[k] {
			inter++
		}
	}
	for k := range b {
		if !seen[k] {
			union++
		}
	}
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}

func toSimilarInvestigations(results []model.SimilarityResult) []model.SimilarInvestigation {
	out := make([]model.SimilarInvestigation, len(results))
	for i, r := range results {
		reason := strings.Join(r.MatchingReasons, ", ")
		out[i] = model.SimilarInvestigation{
			SessionID: r.InvestigationID, Question: r.Question, Service: r.Service,
			Goal: r.Goal, Confidence: r.Confidence, LeadingHypothesis: r.RootCause,
			SimilarityScore: r.Score, Reason: reason, MatchingReasons: r.MatchingReasons,
			CompletedAt: r.CompletedAt,
		}
	}
	return out
}
