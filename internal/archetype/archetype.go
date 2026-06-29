package archetype

import (
	"github.com/stackrail/incident-investigator/internal/model"
	sigpkg "github.com/stackrail/incident-investigator/internal/signals"
)

// Domain groups failure archetypes for library organization.
type Domain string

const (
	DomainApplication    Domain = "application"
	DomainInfrastructure Domain = "infrastructure"
	DomainData           Domain = "data"
	DomainPlatform       Domain = "platform"
	DomainOperations     Domain = "operations"
	DomainSecurity       Domain = "security"
	DomainExternal       Domain = "external"
	DomainGeneric        Domain = "generic"
)

// Candidate is a scored hypothesis before normalization into a probability field.
type Candidate struct {
	HypothesisID string
	Statement    string
	Score        float64
	Rationale    string
	Support      []string
	Conflict     []string
	AlwaysKeep   bool
}

// ScoreContext carries session state into archetype scoring.
type ScoreContext struct {
	Session        *model.Session
	Signals        sigpkg.Signals
	Contradictions []model.Contradiction
}

// DeployContradicted reports whether timeline evidence contradicts deploy-caused theories.
func (ctx ScoreContext) DeployContradicted() bool {
	for _, c := range ctx.Contradictions {
		if c.ID == "contradiction-deploy-after-incident" {
			return true
		}
	}
	return false
}

// Archetype is a reusable failure-mode template. Built-in archetypes ship in the
// default library; enterprises can register extensions without changing the runtime.
type Archetype interface {
	ID() string
	Name() string
	Domain() Domain
	Priority() int
	HypothesisID() string
	Applicable(ctx ScoreContext) bool
	Score(ctx ScoreContext) Candidate
	SeedQuestions() []QuestionSeed
	ExpectedEvidence() []model.Category
	TypicalSubcauses() []string
}
