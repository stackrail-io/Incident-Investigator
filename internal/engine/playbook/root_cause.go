package playbook

import (
	"github.com/stackrail/incident-investigator/internal/archetype"
	"github.com/stackrail/incident-investigator/internal/archetype/builtin"
	"github.com/stackrail/incident-investigator/internal/model"
)

// rootCauseFromRegistry assembles the root-cause playbook from archetype question seeds.
func rootCauseFromRegistry() *Playbook {
	seeds := builtin.DefaultRegistry().SeedQuestions()
	questions := make([]PlaybookQuestion, len(seeds))
	for i, seed := range seeds {
		questions[i] = questionFromSeed(seed)
	}
	return &Playbook{
		ID:        "root-cause-default",
		Goal:      model.GoalRootCause,
		Questions: questions,
	}
}

func questionFromSeed(seed archetype.QuestionSeed) PlaybookQuestion {
	pq := PlaybookQuestion{
		ID:            seed.ID,
		Title:         seed.Title,
		Description:   seed.Description,
		Priority:      seed.Priority,
		Requires:      append([]model.Category(nil), seed.Requires...),
		DependsOn:     append([]string(nil), seed.DependsOn...),
		TriggerSignal: seed.TriggerSignal,
		Generates:     append([]string(nil), seed.Generates...),
	}
	if pq.Priority == 0 {
		pq.Priority = 50
	}
	for _, eff := range seed.Effects {
		pq.Effects = append(pq.Effects, PlaybookEffect{
			WhenTrue:     eff.WhenTrue,
			Action:       EffectAction(eff.Action),
			HypothesisID: eff.HypothesisID,
			Amount:       eff.Amount,
		})
	}
	return pq
}
