package builtin

import (
	"github.com/stackrail/incident-investigator/internal/archetype"
	"github.com/stackrail/incident-investigator/internal/model"
)

// DeploymentUnrelated competes against deployment-blame bias.
type DeploymentUnrelated struct{}

func (DeploymentUnrelated) ID() string               { return "deployment-unrelated" }
func (DeploymentUnrelated) Name() string             { return "Deployment Unrelated" }
func (DeploymentUnrelated) Domain() archetype.Domain { return archetype.DomainApplication }
func (DeploymentUnrelated) Priority() int            { return 3 }
func (DeploymentUnrelated) HypothesisID() string     { return "hypothesis-deployment-unrelated" }
func (DeploymentUnrelated) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (DeploymentUnrelated) ExpectedEvidence() []model.Category {
	return []model.Category{model.CategoryDeploymentEvents, model.CategoryAlertEvents}
}
func (DeploymentUnrelated) TypicalSubcauses() []string { return nil }
func (DeploymentUnrelated) SeedQuestions() []archetype.QuestionSeed { return nil }

func (a DeploymentUnrelated) Score(ctx archetype.ScoreContext) archetype.Candidate {
	sig := ctx.Signals
	c := archetype.Candidate{
		HypothesisID: a.HypothesisID(),
		Statement:    "The deployment was unrelated; the incident has another root cause.",
		Score:        8,
		Rationale:    "Maintained as a competing baseline against deployment-blame bias.",
	}
	if ctx.DeployContradicted() {
		c.Score += 40
		c.Rationale = "The deployment timestamp falls after the incident began."
	}
	if sig.FirstDeployment == nil {
		c.Score = 0
	}
	return c
}

// Unknown is the catch-all that fades as evidence coverage grows.
type Unknown struct{}

func (Unknown) ID() string               { return "unknown-novel" }
func (Unknown) Name() string             { return "Unknown / Novel Failure" }
func (Unknown) Domain() archetype.Domain { return archetype.DomainGeneric }
func (Unknown) Priority() int            { return 5 }
func (Unknown) HypothesisID() string     { return "hypothesis-unknown" }
func (Unknown) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (Unknown) ExpectedEvidence() []model.Category { return nil }
func (Unknown) TypicalSubcauses() []string         { return nil }
func (Unknown) SeedQuestions() []archetype.QuestionSeed { return nil }

func (a Unknown) Score(ctx archetype.ScoreContext) archetype.Candidate {
	score := 40.0 - 5*float64(len(ctx.Signals.Categories))
	if score < 4 {
		score = 4
	}
	return archetype.Candidate{
		HypothesisID: a.HypothesisID(),
		Statement:    "Root cause is not yet determined; more evidence is required.",
		Score:        score,
		Rationale:    "Reflects residual uncertainty given current evidence coverage.",
		AlwaysKeep:   true,
	}
}
