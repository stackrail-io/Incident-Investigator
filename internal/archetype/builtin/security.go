package builtin

import (
	"github.com/stackrail/incident-investigator/internal/archetype"
	"github.com/stackrail/incident-investigator/internal/model"
	sigpkg "github.com/stackrail/incident-investigator/internal/signals"
)

// SecurityIncident scores security breaches distinct from certificate expiry.
type SecurityIncident struct{}

func (SecurityIncident) ID() string               { return "security-incident" }
func (SecurityIncident) Name() string             { return "Security Incident" }
func (SecurityIncident) Domain() archetype.Domain { return archetype.DomainSecurity }
func (SecurityIncident) Priority() int            { return 5 }
func (SecurityIncident) HypothesisID() string     { return "hypothesis-security-incident" }
func (SecurityIncident) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (SecurityIncident) ExpectedEvidence() []model.Category {
	return []model.Category{model.CategorySecurityEvents, model.CategoryApplicationLogs, model.CategoryHumanContext}
}
func (SecurityIncident) TypicalSubcauses() []string {
	return []string{"credential compromise", "unauthorized access", "exploit", "data exfiltration"}
}

func (SecurityIncident) SeedQuestions() []archetype.QuestionSeed {
	return []archetype.QuestionSeed{{
		ID: "security-breach", Priority: 84,
		Title:       "Was unauthorized access or a security breach detected?",
		Description: "Credential compromise and exploits can cause outages or data exposure.",
		Requires:      []model.Category{model.CategorySecurityEvents, model.CategoryApplicationLogs},
		TriggerSignal: "security",
		Effects: []archetype.QuestionEffect{
			effect(true, archetype.EffectIncrease, "hypothesis-security-incident", 35),
			effect(false, archetype.EffectDecrease, "hypothesis-security-incident", 25),
		},
	}}
}

func (a SecurityIncident) Score(ctx archetype.ScoreContext) archetype.Candidate {
	sig := ctx.Signals
	s := ctx.Session
	c := archetype.Candidate{
		HypothesisID: a.HypothesisID(),
		Statement:    "A security incident such as unauthorized access or compromise caused the outage.",
	}
	if sig.Keywords["security"] {
		c.Score += 46
		c.Rationale = "Security incident symptoms appear in the evidence."
		c.Support = append(c.Support, sigpkg.EvidenceMatching(s, func(e *model.Evidence) bool {
			return sigpkg.MatchesAny(sigpkg.Haystack(e), sigpkg.Keywords["security"])
		})...)
	}
	if s.HasCategory(model.CategorySecurityEvents) && !sig.Keywords["cert"] {
		c.Score += 10
	}
	return c
}
