package intelligence

import (
	"context"
	"math"
	"sort"
	"strings"

	"github.com/stackrail/incident-investigator/internal/model"
)

// PatternEngine discovers and recommends investigation patterns.
type PatternEngine interface {
	Library() []model.InvestigationPattern
	ExtractFromSnapshot(snap *model.InvestigationSnapshot) []model.InvestigationPattern
	Suggest(ctx context.Context, req model.PatternRequest, archive InvestigationArchive, limit int) ([]model.SuggestedPattern, error)
}

// HeuristicPatternEngine mines patterns from the library and archive.
type HeuristicPatternEngine struct {
	library []model.InvestigationPattern
}

// NewHeuristicPatternEngine returns the engine with built-in patterns.
func NewHeuristicPatternEngine() *HeuristicPatternEngine {
	return &HeuristicPatternEngine{library: DefaultPatternLibrary()}
}

// Library implements PatternEngine.
func (p *HeuristicPatternEngine) Library() []model.InvestigationPattern {
	return append([]model.InvestigationPattern(nil), p.library...)
}

// ExtractFromSnapshot implements PatternEngine.
func (p *HeuristicPatternEngine) ExtractFromSnapshot(snap *model.InvestigationSnapshot) []model.InvestigationPattern {
	if snap == nil {
		return nil
	}
	var matched []model.InvestigationPattern
	for _, pat := range p.library {
		if patternMatchesSnapshot(pat, snap) {
			cp := pat
			cp.Occurrences = 1
			matched = append(matched, cp)
		}
	}
	return matched
}

// Suggest implements PatternEngine.
func (p *HeuristicPatternEngine) Suggest(_ context.Context, req model.PatternRequest, archive InvestigationArchive, limit int) ([]model.SuggestedPattern, error) {
	if limit <= 0 {
		limit = 5
	}

	occurrences := map[string]int{}
	for _, snap := range archive.All() {
		if req.Goal != "" && snap.Goal != req.Goal {
			continue
		}
		for _, pat := range p.ExtractFromSnapshot(snap) {
			occurrences[pat.ID]++
		}
		// Also count by root cause signature.
		if snap.RootCause != "" {
			occurrences["mined-"+snap.RootCause]++
		}
	}

	type scored struct {
		pat   model.InvestigationPattern
		score float64
		count int
	}
	var candidates []scored
	for _, pat := range p.library {
		if req.Goal != "" {
			// patterns are goal-agnostic but filter by typical root cause relevance
		}
		count := occurrences[pat.ID]
		if count == 0 && !patternMatchesRequest(pat, req) {
			continue
		}
		if count == 0 {
			count = 1
		}
		conf := math.Min(95, float64(count)*18+basePatternScore(pat, req))
		candidates = append(candidates, scored{pat: pat, score: conf, count: count})
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].count > candidates[j].count
	})

	var total float64
	for _, c := range candidates {
		total += c.score
	}

	var patterns []model.SuggestedPattern
	for i, c := range candidates {
		if i >= limit {
			break
		}
		conf := c.score
		if total > 0 {
			conf = c.score / total * 100
		}
		patterns = append(patterns, model.SuggestedPattern{
			ID:                   c.pat.ID,
			Name:                 c.pat.Name,
			Description:          c.pat.Description,
			Confidence:           round1(conf),
			EvidenceCategories:   c.pat.ExpectedEvidence,
			HypothesisHint:       c.pat.TypicalRootCause,
			RecommendedQuestions: c.pat.RecommendedQuestions,
			Occurrences:          c.count,
			Reason:               "Pattern library match with historical occurrences.",
		})
	}
	if patterns == nil {
		patterns = []model.SuggestedPattern{}
	}
	return patterns, nil
}

func basePatternScore(pat model.InvestigationPattern, req model.PatternRequest) float64 {
	if req.Session == nil {
		return 10
	}
	score := 10.0
	text := strings.ToLower(req.Question + " " + req.Service)
	for _, label := range pat.Trigger.Sequence {
		if strings.Contains(text, label) {
			score += 15
		}
	}
	return score
}

func patternMatchesRequest(pat model.InvestigationPattern, req model.PatternRequest) bool {
	if req.Session == nil {
		return false
	}
	snap := BuildSnapshot(req.Session)
	return patternMatchesSnapshot(pat, &snap)
}

func patternMatchesSnapshot(pat model.InvestigationPattern, snap *model.InvestigationSnapshot) bool {
	if snap.RootCause != "" && pat.TypicalRootCause != "" {
		if strings.Contains(snap.RootCause, strings.TrimPrefix(pat.TypicalRootCause, "hypothesis-")) ||
			snap.RootCause == pat.TypicalRootCause {
			return true
		}
	}
	for _, want := range pat.ExpectedEvidence {
		found := false
		for _, sum := range snap.EvidenceSummary {
			if sum.Category == want {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if len(pat.Trigger.Sequence) > 0 && snap.KnowledgeGraph != nil {
		labels := graphNodeLabels(snap.KnowledgeGraph)
		matched := 0
		for _, seq := range pat.Trigger.Sequence {
			for _, l := range labels {
				if strings.Contains(strings.ToLower(l), seq) {
					matched++
					break
				}
			}
		}
		return matched >= len(pat.Trigger.Sequence)/2
	}
	return len(pat.ExpectedEvidence) > 0
}

func graphNodeLabels(g *model.GraphView) []string {
	var out []string
	if g == nil {
		return out
	}
	for _, n := range g.Nodes {
		if n != nil {
			out = append(out, n.Label)
		}
	}
	return out
}

// DefaultPatternLibrary returns built-in investigation patterns.
func DefaultPatternLibrary() []model.InvestigationPattern {
	return []model.InvestigationPattern{
		{
			ID: "pattern-deployment-failure", Name: "Deployment Failure Pattern",
			Description: "Deployment precedes pod restarts, latency spike, errors, rollback, and recovery.",
			Trigger: model.GraphPattern{
				Sequence: []string{"deploy", "latency", "error", "rollback", "recover"},
				EdgeTypes: []model.EdgeType{model.EdgeCauses, model.EdgeOccurredBefore},
			},
			RecommendedQuestions: []model.QuestionTemplate{
				{Text: "Did deployment happen before errors?", Categories: []model.Category{model.CategoryDeploymentEvents, model.CategoryApplicationLogs}},
				{Text: "Was there a rollback?", Categories: []model.Category{model.CategoryDeploymentEvents}},
			},
			ExpectedEvidence: []model.Category{
				model.CategoryDeploymentEvents, model.CategoryApplicationLogs,
				model.CategoryMetrics, model.CategoryAlertEvents,
			},
			TypicalRootCause: "hypothesis-deployment-caused",
		},
		{
			ID: "pattern-certificate-expiry", Name: "Certificate Expiry Pattern",
			Description: "Certificate expiry leads to TLS errors, connection failures, renewal, and recovery.",
			Trigger: model.GraphPattern{
				Sequence: []string{"certificate", "tls", "connection", "renewal"},
			},
			RecommendedQuestions: []model.QuestionTemplate{
				{Text: "Did a certificate expire?", Categories: []model.Category{model.CategorySecurityEvents, model.CategoryApplicationLogs}},
			},
			ExpectedEvidence: []model.Category{
				model.CategorySecurityEvents, model.CategoryApplicationLogs, model.CategoryNetworkEvents,
			},
			TypicalRootCause: "hypothesis-certificate-expiry",
		},
		{
			ID: "pattern-database-saturation", Name: "Database Saturation Pattern",
			Description: "Database saturation causes query timeouts, connection pool exhaustion, and service degradation.",
			Trigger: model.GraphPattern{
				Sequence: []string{"database", "timeout", "saturation"},
			},
			RecommendedQuestions: []model.QuestionTemplate{
				{Text: "Are database connections exhausted?", Categories: []model.Category{model.CategoryDatabaseEvents, model.CategoryMetrics}},
			},
			ExpectedEvidence: []model.Category{
				model.CategoryDatabaseEvents, model.CategoryMetrics, model.CategoryApplicationLogs,
			},
			TypicalRootCause: "hypothesis-database-saturation",
		},
		{
			ID: "pattern-lock-contention", Name: "Lock Contention Pattern",
			Description: "Row or table lock contention causes serialized writes, long wait queues, and latency spikes while database capacity remains healthy.",
			Trigger: model.GraphPattern{
				Sequence: []string{"database", "lock", "latency"},
			},
			RecommendedQuestions: []model.QuestionTemplate{
				{Text: "Were multiple statements blocked on the same row?", Categories: []model.Category{model.CategoryDatabaseEvents, model.CategoryTraceEvents}},
				{Text: "Are lock timeouts configured?", Categories: []model.Category{model.CategoryConfigurationChanges}},
			},
			ExpectedEvidence: []model.Category{
				model.CategoryDatabaseEvents, model.CategoryTraceEvents,
				model.CategoryConfigurationChanges, model.CategoryMetrics,
			},
			TypicalRootCause: "hypothesis-lock-contention",
		},
		{
			ID: "pattern-retry-storm", Name: "Retry Storm Pattern",
			Description: "Retry amplification causes cascading latency and error spikes across services.",
			Trigger: model.GraphPattern{
				Sequence: []string{"retry", "latency", "error"},
			},
			ExpectedEvidence: []model.Category{
				model.CategoryApplicationLogs, model.CategoryMetrics, model.CategoryTraceEvents,
			},
			TypicalRootCause: "hypothesis-retry-storm",
		},
		{
			ID: "pattern-dependency-failure", Name: "Dependency Failure Pattern",
			Description: "Downstream dependency timeouts and errors cascade to callers.",
			Trigger: model.GraphPattern{Sequence: []string{"dependency", "timeout", "error"}},
			ExpectedEvidence: []model.Category{
				model.CategoryApplicationLogs, model.CategoryTraceEvents, model.CategoryAlertEvents,
			},
			TypicalRootCause: "hypothesis-dependency-failure",
		},
		{
			ID: "pattern-external-outage", Name: "External Outage Pattern",
			Description: "Third-party or vendor outages cause internal API failures.",
			Trigger: model.GraphPattern{Sequence: []string{"external", "vendor", "outage"}},
			ExpectedEvidence: []model.Category{
				model.CategoryApplicationLogs, model.CategoryNetworkEvents,
			},
			TypicalRootCause: "hypothesis-external-outage",
		},
	}
}
