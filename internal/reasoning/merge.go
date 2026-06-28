package reasoning

import (
	"sort"

	"github.com/stackrail/incident-investigator/internal/model"
)

// Merger combines actions from multiple reasoners and resolves conflicts.
type Merger struct {
	Strategy model.OrchestrationStrategy
}

// NewMerger returns a merger with the given strategy.
func NewMerger(strategy model.OrchestrationStrategy) *Merger {
	if strategy == "" {
		strategy = model.StrategySequential
	}
	return &Merger{Strategy: strategy}
}

type weightedAction struct {
	action model.ReasoningAction
	weight int
}

// Merge combines reasoner results into a single action list.
func (m *Merger) Merge(results []model.ReasoningResult) []model.ReasoningAction {
	if len(results) == 0 {
		return nil
	}
	switch m.Strategy {
	case model.StrategyWeightedVote, model.StrategyConsensus:
		return m.mergeWeighted(results)
	default:
		return m.mergeSequential(results)
	}
}

func (m *Merger) mergeSequential(results []model.ReasoningResult) []model.ReasoningAction {
	sorted := append([]model.ReasoningResult(nil), results...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return reasonerPriority(sorted[i].Reasoner) > reasonerPriority(sorted[j].Reasoner)
	})
	var out []model.ReasoningAction
	for _, r := range sorted {
		out = append(out, r.Actions...)
	}
	return out
}

func (m *Merger) mergeWeighted(results []model.ReasoningResult) []model.ReasoningAction {
	// Non-confidence actions: append in priority order (dedupe replace-style).
	var passthrough []model.ReasoningAction
	confDeltas := map[string][]weightedAction{}

	for _, r := range results {
		w := reasonerPriority(r.Reasoner)
		for _, a := range r.Actions {
			switch a.Type {
			case model.ActionIncreaseHypothesisConfidence, model.ActionDecreaseHypothesisConfidence:
				confDeltas[a.HypothesisID] = append(confDeltas[a.HypothesisID], weightedAction{a, w})
			case model.ActionReplaceHypotheses, model.ActionUpdateGraph, model.ActionSetTimeline:
				passthrough = replaceAction(passthrough, a)
			default:
				passthrough = append(passthrough, a)
			}
		}
	}

	for hid, items := range confDeltas {
		delta := weightedAverageDelta(items)
		if delta == 0 {
			continue
		}
		act := model.ReasoningAction{
			Type:         model.ActionIncreaseHypothesisConfidence,
			Reasoner:     "orchestrator",
			Reason:       "Weighted merge of reasoner confidence deltas.",
			HypothesisID: hid,
			Delta:        delta,
		}
		if delta < 0 {
			act.Type = model.ActionDecreaseHypothesisConfidence
			act.Delta = -delta
		}
		passthrough = append(passthrough, act)
	}
	return passthrough
}

func weightedAverageDelta(items []weightedAction) float64 {
	var sum, wsum float64
	for _, it := range items {
		d := it.action.Delta
		if it.action.Type == model.ActionDecreaseHypothesisConfidence {
			d = -d
		}
		w := float64(it.weight)
		sum += d * w
		wsum += w
	}
	if wsum == 0 {
		return 0
	}
	return sum / wsum
}

func replaceAction(list []model.ReasoningAction, a model.ReasoningAction) []model.ReasoningAction {
	for i, existing := range list {
		if existing.Type == a.Type {
			list[i] = a
			return list
		}
	}
	return append(list, a)
}

func reasonerPriority(name string) int {
	switch name {
	case "temporal":
		return 100
	case "causal":
		return 90
	case "hypothesis":
		return 80
	case "consistency":
		return 70
	case "semantic":
		return 50
	default:
		return 10
	}
}
