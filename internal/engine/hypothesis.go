package engine

import (
	"sort"

	"github.com/stackrail/incident-investigator/internal/model"
)

// HypothesisEngine produces a ranked field of competing explanations. It never
// returns a single hypothesis.
type HypothesisEngine interface {
	Generate(s *model.Session, sig Signals, contradictions []model.Contradiction) []model.Hypothesis
}

// HeuristicHypothesisEngine scores a fixed catalogue of failure archetypes
// against the observed signals, then normalizes them into a probability field.
type HeuristicHypothesisEngine struct{}

// NewHeuristicHypothesisEngine returns the default engine.
func NewHeuristicHypothesisEngine() *HeuristicHypothesisEngine {
	return &HeuristicHypothesisEngine{}
}

type candidate struct {
	id        string
	statement string
	score     float64
	rationale string
	support   []string
	conflict  []string
}

// Generate implements HypothesisEngine.
func (h *HeuristicHypothesisEngine) Generate(s *model.Session, sig Signals, contradictions []model.Contradiction) []model.Hypothesis {
	deployContradicted := hasContradiction(contradictions, "contradiction-deploy-after-incident")

	cands := []*candidate{
		h.deploymentCaused(s, sig, deployContradicted),
		h.deploymentUnrelated(s, sig, deployContradicted),
		h.databaseSaturation(s, sig),
		h.configurationChange(s, sig),
		h.networkOrDNS(s, sig),
		h.certificateExpiry(s, sig),
		h.resourceExhaustion(s, sig),
		h.retryStorm(s, sig),
		h.unknown(s, sig),
	}

	// Drop zero-scored candidates (except we always keep "unknown").
	kept := cands[:0]
	for _, c := range cands {
		if c.score > 0 || c.id == "hypothesis-unknown" {
			kept = append(kept, c)
		}
	}

	var total float64
	for _, c := range kept {
		total += c.score
	}
	if total <= 0 {
		total = 1
	}

	out := make([]model.Hypothesis, 0, len(kept))
	for _, c := range kept {
		out = append(out, model.Hypothesis{
			ID:                  c.id,
			Statement:           c.statement,
			Confidence:          round1(c.score / total * 100),
			Status:              model.StatusProposed,
			Rationale:           c.rationale,
			SupportingEvidence:  nonNil(c.support),
			ConflictingEvidence: nonNil(c.conflict),
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Confidence != out[j].Confidence {
			return out[i].Confidence > out[j].Confidence
		}
		return out[i].ID < out[j].ID
	})

	assignStatuses(out)
	return out
}

func (h *HeuristicHypothesisEngine) deploymentCaused(s *model.Session, sig Signals, contradicted bool) *candidate {
	c := &candidate{
		id:        "hypothesis-deployment-caused",
		statement: "A recent deployment introduced the regression that caused the incident.",
	}
	if sig.FirstDeployment == nil {
		return c
	}
	c.score = 5
	c.support = append(c.support, sig.FirstDeployment.ID)
	if sig.DeployBeforeIncident {
		c.score += 45
		c.rationale = "A deployment was observed shortly before the first symptom."
		if sig.IncidentOnset != nil {
			c.support = append(c.support, sig.IncidentOnset.ID)
		}
	}
	if sig.Keywords["config"] && s.HasCategory(model.CategoryConfigurationChanges) {
		c.score += 12
		c.support = append(c.support, evidenceMatching(s, func(e *model.Evidence) bool {
			return e.Category == model.CategoryConfigurationChanges
		})...)
	}
	if sig.Keywords["restart"] {
		c.score += 10
	}
	if contradicted {
		// Timeline says the deploy happened after the incident.
		c.score = 2
		c.rationale = "Deployment timing contradicts a causal role (it happened after onset)."
		if sig.IncidentOnset != nil {
			c.conflict = append(c.conflict, sig.IncidentOnset.ID)
		}
	}
	return c
}

func (h *HeuristicHypothesisEngine) deploymentUnrelated(s *model.Session, sig Signals, contradicted bool) *candidate {
	c := &candidate{
		id:        "hypothesis-deployment-unrelated",
		statement: "The deployment was unrelated; the incident has another root cause.",
		score:     8,
		rationale: "Maintained as a competing baseline against deployment-blame bias.",
	}
	if contradicted {
		c.score += 40
		c.rationale = "The deployment timestamp falls after the incident began."
	}
	if sig.FirstDeployment == nil {
		// No deployment at all: this archetype is moot, fold into unknown.
		c.score = 0
	}
	return c
}

func (h *HeuristicHypothesisEngine) databaseSaturation(s *model.Session, sig Signals) *candidate {
	c := &candidate{
		id:        "hypothesis-database-saturation",
		statement: "Database saturation or unavailability degraded the service.",
	}
	if sig.Keywords["database"] {
		c.score += 35
		c.rationale = "Database-related symptoms appear in the evidence."
		c.support = append(c.support, evidenceMatching(s, func(e *model.Evidence) bool {
			return matchesAny(haystack(e), signalKeywords["database"])
		})...)
	}
	if s.HasCategory(model.CategoryDatabaseEvents) {
		c.score += 12
	}
	if sig.Keywords["latency"] && s.HasCategory(model.CategoryMetrics) {
		c.score += 8
	}
	return c
}

func (h *HeuristicHypothesisEngine) configurationChange(s *model.Session, sig Signals) *candidate {
	c := &candidate{
		id:        "hypothesis-configuration-change",
		statement: "A configuration or feature-flag change triggered the incident.",
	}
	if sig.Keywords["config"] {
		c.score += 30
		c.rationale = "Configuration-change symptoms appear in the evidence."
		c.support = append(c.support, evidenceMatching(s, func(e *model.Evidence) bool {
			return e.Category == model.CategoryConfigurationChanges || matchesAny(haystack(e), signalKeywords["config"])
		})...)
	}
	return c
}

func (h *HeuristicHypothesisEngine) networkOrDNS(s *model.Session, sig Signals) *candidate {
	c := &candidate{
		id:        "hypothesis-network-dns",
		statement: "A network or DNS failure disrupted connectivity.",
	}
	if sig.Keywords["dns"] {
		c.score += 38
		c.rationale = "DNS resolution symptoms appear in the evidence."
	}
	if sig.Keywords["network"] {
		c.score += 18
		if c.rationale == "" {
			c.rationale = "Network connectivity symptoms appear in the evidence."
		}
	}
	if c.score > 0 {
		c.support = append(c.support, evidenceMatching(s, func(e *model.Evidence) bool {
			t := haystack(e)
			return matchesAny(t, signalKeywords["dns"]) || matchesAny(t, signalKeywords["network"])
		})...)
	}
	return c
}

func (h *HeuristicHypothesisEngine) certificateExpiry(s *model.Session, sig Signals) *candidate {
	c := &candidate{
		id:        "hypothesis-certificate-expiry",
		statement: "An expired or invalid TLS certificate broke secure connections.",
	}
	if sig.Keywords["cert"] {
		c.score += 48
		c.rationale = "TLS/certificate symptoms appear in the evidence."
		c.support = append(c.support, evidenceMatching(s, func(e *model.Evidence) bool {
			return matchesAny(haystack(e), signalKeywords["cert"])
		})...)
	}
	return c
}

func (h *HeuristicHypothesisEngine) resourceExhaustion(s *model.Session, sig Signals) *candidate {
	c := &candidate{
		id:        "hypothesis-resource-exhaustion",
		statement: "Resource exhaustion (memory/CPU) caused crashes or throttling.",
	}
	if sig.Keywords["memory"] {
		c.score += 35
		c.rationale = "Memory pressure symptoms appear in the evidence."
		c.support = append(c.support, evidenceMatching(s, func(e *model.Evidence) bool {
			return matchesAny(haystack(e), signalKeywords["memory"])
		})...)
	}
	if sig.Keywords["restart"] {
		c.score += 12
	}
	return c
}

func (h *HeuristicHypothesisEngine) retryStorm(s *model.Session, sig Signals) *candidate {
	c := &candidate{
		id:        "hypothesis-retry-storm",
		statement: "A retry storm / cascading failure amplified a smaller fault.",
	}
	if sig.Keywords["retry"] {
		c.score += 32
		c.rationale = "Retry-amplification symptoms appear in the evidence."
		c.support = append(c.support, evidenceMatching(s, func(e *model.Evidence) bool {
			return matchesAny(haystack(e), signalKeywords["retry"])
		})...)
	}
	if sig.Keywords["latency"] {
		c.score += 8
	}
	return c
}

func (h *HeuristicHypothesisEngine) unknown(s *model.Session, sig Signals) *candidate {
	// The catch-all is strong when little evidence exists and fades as coverage
	// and signal strength grow.
	score := 40.0 - 5*float64(len(sig.Categories))
	if score < 4 {
		score = 4
	}
	return &candidate{
		id:        "hypothesis-unknown",
		statement: "Root cause is not yet determined; more evidence is required.",
		score:     score,
		rationale: "Reflects residual uncertainty given current evidence coverage.",
	}
}

// assignStatuses promotes the leading hypothesis and refutes clearly-dominated
// ones.
func assignStatuses(hs []model.Hypothesis) {
	for i := range hs {
		switch {
		case i == 0 && hs[i].Confidence >= 60:
			hs[i].Status = model.StatusLeading
		case i == 0 && hs[i].Confidence >= 35:
			hs[i].Status = model.StatusSupported
		case hs[i].Confidence < 8:
			hs[i].Status = model.StatusRefuted
		default:
			hs[i].Status = model.StatusProposed
		}
	}
}

func hasContradiction(cs []model.Contradiction, id string) bool {
	for _, c := range cs {
		if c.ID == id {
			return true
		}
	}
	return false
}

// evidenceMatching returns the de-duplicated ids of evidence satisfying pred.
func evidenceMatching(s *model.Session, pred func(*model.Evidence) bool) []string {
	seen := map[string]bool{}
	var out []string
	for _, e := range sortedByTime(s.Evidence) {
		if pred(e) && !seen[e.ID] {
			seen[e.ID] = true
			out = append(out, e.ID)
		}
	}
	return out
}

func nonNil(in []string) []string {
	if in == nil {
		return []string{}
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

func round1(v float64) float64 {
	return float64(int(v*10+0.5)) / 10
}
