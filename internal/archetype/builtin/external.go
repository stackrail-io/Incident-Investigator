package builtin

import (
	"github.com/stackrail/incident-investigator/internal/archetype"
	"github.com/stackrail/incident-investigator/internal/model"
	sigpkg "github.com/stackrail/incident-investigator/internal/signals"
)

// ExternalOutage scores third-party or SaaS provider outages.
type ExternalOutage struct{}

func (ExternalOutage) ID() string               { return "external-outage" }
func (ExternalOutage) Name() string             { return "External Service Outage" }
func (ExternalOutage) Domain() archetype.Domain { return archetype.DomainExternal }
func (ExternalOutage) Priority() int            { return 5 }
func (ExternalOutage) HypothesisID() string     { return "hypothesis-external-outage" }
func (ExternalOutage) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (ExternalOutage) ExpectedEvidence() []model.Category {
	return []model.Category{model.CategoryApplicationLogs, model.CategoryNetworkEvents, model.CategoryAlertEvents}
}
func (ExternalOutage) TypicalSubcauses() []string {
	return []string{"vendor outage", "regional provider failure", "api unavailable", "sla breach"}
}

func (ExternalOutage) SeedQuestions() []archetype.QuestionSeed {
	return []archetype.QuestionSeed{{
		ID: "vendor-outage", Priority: 76,
		Title:       "Did a third-party or vendor outage occur?",
		Description: "External provider failures propagate as internal timeouts and errors.",
		Requires:      []model.Category{model.CategoryApplicationLogs, model.CategoryNetworkEvents},
		TriggerSignal: "external",
		Effects: []archetype.QuestionEffect{
			effect(true, archetype.EffectIncrease, "hypothesis-external-outage", 32),
			effect(false, archetype.EffectDecrease, "hypothesis-external-outage", 22),
		},
	}}
}

func (a ExternalOutage) Score(ctx archetype.ScoreContext) archetype.Candidate {
	sig := ctx.Signals
	s := ctx.Session
	c := archetype.Candidate{
		HypothesisID: a.HypothesisID(),
		Statement:    "An external or third-party service outage caused the incident.",
	}
	if sig.Keywords["external"] {
		c.Score += 42
		c.Rationale = "Third-party or vendor outage symptoms appear in the evidence."
		c.Support = append(c.Support, sigpkg.EvidenceMatching(s, func(e *model.Evidence) bool {
			return sigpkg.MatchesAny(sigpkg.Haystack(e), sigpkg.Keywords["external"])
		})...)
	}
	return c
}
