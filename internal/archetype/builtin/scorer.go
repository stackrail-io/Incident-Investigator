package builtin

import (
	"github.com/stackrail/incident-investigator/internal/archetype"
	"github.com/stackrail/incident-investigator/internal/model"
	sigpkg "github.com/stackrail/incident-investigator/internal/signals"
)

type scoreBoost struct {
	When   string
	Amount float64
}

type scorePenalty struct {
	When   string
	Unless string
	Amount float64
}

type scoreOpts struct {
	HypothesisID string
	Statement    string
	Keyword      string
	Base         float64
	Rationale    string
	Boosts       []scoreBoost
	Penalties    []scorePenalty
	SkipIf       []string
	RequireAny   []string
}

func scoreKeyword(ctx archetype.ScoreContext, hypID, statement, keyword string, base float64) archetype.Candidate {
	return scoreWith(ctx, scoreOpts{
		HypothesisID: hypID,
		Statement:    statement,
		Keyword:      keyword,
		Base:         base,
	})
}

func scoreWith(ctx archetype.ScoreContext, opts scoreOpts) archetype.Candidate {
	sig := ctx.Signals
	s := ctx.Session
	c := archetype.Candidate{HypothesisID: opts.HypothesisID, Statement: opts.Statement}

	for _, kw := range opts.SkipIf {
		if sig.Keywords[kw] {
			return c
		}
	}
	if len(opts.RequireAny) > 0 {
		matched := false
		for _, kw := range opts.RequireAny {
			if sig.Keywords[kw] {
				matched = true
				break
			}
		}
		if !matched {
			return c
		}
	}
	if opts.Keyword != "" && !sig.Keywords[opts.Keyword] {
		return c
	}

	c.Score = opts.Base
	if opts.Rationale != "" {
		c.Rationale = opts.Rationale
	} else if opts.Keyword != "" {
		c.Rationale = "Evidence matches " + opts.Keyword + " failure signatures."
	}
	if opts.Keyword != "" {
		c.Support = append(c.Support, sigpkg.EvidenceMatching(s, func(e *model.Evidence) bool {
			return sigpkg.MatchesAny(sigpkg.Haystack(e), sigpkg.Keywords[opts.Keyword])
		})...)
	}

	for _, b := range opts.Boosts {
		if sig.Keywords[b.When] {
			c.Score += b.Amount
		}
	}
	for _, p := range opts.Penalties {
		if !sig.Keywords[p.When] {
			continue
		}
		if p.Unless != "" && sig.Keywords[p.Unless] {
			continue
		}
		c.Score -= p.Amount
	}
	if c.Score < 0 {
		c.Score = 0
	}
	return c
}
