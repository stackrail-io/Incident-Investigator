package builtin

import "github.com/stackrail/incident-investigator/internal/archetype"

func effect(whenTrue bool, action archetype.EffectAction, hyp string, amount float64) archetype.QuestionEffect {
	return archetype.QuestionEffect{WhenTrue: whenTrue, Action: action, HypothesisID: hyp, Amount: amount}
}
