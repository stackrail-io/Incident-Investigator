package archetype

import "github.com/stackrail/incident-investigator/internal/model"

// EffectAction is a playbook outcome on a hypothesis.
type EffectAction string

const (
	EffectIncrease EffectAction = "Increase"
	EffectDecrease EffectAction = "Decrease"
)

// QuestionEffect applies when a seeded question resolves true/false.
type QuestionEffect struct {
	WhenTrue     bool
	Action       EffectAction
	HypothesisID string
	Amount       float64
}

// QuestionSeed is a declarative investigation question contributed by an archetype.
type QuestionSeed struct {
	ID            string
	Title         string
	Description   string
	Priority      int
	Requires      []model.Category
	DependsOn     []string
	Effects       []QuestionEffect
	TriggerSignal string
	Generates     []string
}
