package archetype

import (
	"sort"

	"github.com/stackrail/incident-investigator/internal/model"
)

// Registry holds registered investigation archetypes.
type Registry struct {
	order []Archetype
	byID  map[string]Archetype
}

// NewRegistry builds a registry from the given archetypes.
func NewRegistry(archetypes ...Archetype) *Registry {
	r := &Registry{byID: make(map[string]Archetype, len(archetypes))}
	for _, a := range archetypes {
		r.Register(a)
	}
	return r
}

// Register adds an archetype to the registry.
func (r *Registry) Register(a Archetype) {
	if r.byID == nil {
		r.byID = make(map[string]Archetype)
	}
	if _, exists := r.byID[a.ID()]; !exists {
		r.order = append(r.order, a)
	}
	r.byID[a.ID()] = a
}

// All returns archetypes in registration order.
func (r *Registry) All() []Archetype {
	out := make([]Archetype, len(r.order))
	copy(out, r.order)
	return out
}

// ByID looks up an archetype by its library id.
func (r *Registry) ByID(id string) Archetype {
	return r.byID[id]
}

// Score evaluates every applicable archetype and returns a ranked hypothesis field.
func (r *Registry) Score(ctx ScoreContext) []model.Hypothesis {
	var cands []Candidate
	for _, a := range r.order {
		if !a.Applicable(ctx) {
			continue
		}
		cands = append(cands, a.Score(ctx))
	}
	return candidatesToHypotheses(cands)
}

// SeedQuestions merges question seeds from every registered archetype, deduped by id.
func (r *Registry) SeedQuestions() []QuestionSeed {
	seen := map[string]bool{}
	var out []QuestionSeed
	for _, a := range r.order {
		for _, q := range a.SeedQuestions() {
			if seen[q.ID] {
				continue
			}
			seen[q.ID] = true
			out = append(out, q)
		}
	}
	return out
}

func candidatesToHypotheses(cands []Candidate) []model.Hypothesis {
	kept := cands[:0]
	for _, c := range cands {
		if c.Score > 0 || c.AlwaysKeep {
			kept = append(kept, c)
		}
	}

	var total float64
	for _, c := range kept {
		total += c.Score
	}
	if total <= 0 {
		total = 1
	}

	out := make([]model.Hypothesis, 0, len(kept))
	for _, c := range kept {
		out = append(out, model.Hypothesis{
			ID:                  c.HypothesisID,
			Statement:           c.Statement,
			Confidence:          round1(c.Score / total * 100),
			Status:              model.StatusProposed,
			Rationale:           c.Rationale,
			SupportingEvidence:  nonNil(c.Support),
			ConflictingEvidence: nonNil(c.Conflict),
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
