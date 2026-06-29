package builtin

import (
	"github.com/stackrail/incident-investigator/internal/archetype"
	"github.com/stackrail/incident-investigator/internal/model"
	sigpkg "github.com/stackrail/incident-investigator/internal/signals"
)

// DatabaseSaturation scores connection-pool, CPU, and I/O saturation.
type DatabaseSaturation struct{}

func (DatabaseSaturation) ID() string               { return "database-saturation" }
func (DatabaseSaturation) Name() string             { return "Database Saturation" }
func (DatabaseSaturation) Domain() archetype.Domain { return archetype.DomainData }
func (DatabaseSaturation) Priority() int            { return 5 }
func (DatabaseSaturation) HypothesisID() string     { return "hypothesis-database-saturation" }
func (DatabaseSaturation) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (DatabaseSaturation) ExpectedEvidence() []model.Category {
	return []model.Category{
		model.CategoryDatabaseEvents, model.CategoryMetrics, model.CategoryApplicationLogs,
	}
}
func (DatabaseSaturation) TypicalSubcauses() []string {
	return []string{"connection exhaustion", "slow queries", "replica lag", "failover", "cpu saturation"}
}
func (a DatabaseSaturation) Score(ctx archetype.ScoreContext) archetype.Candidate {
	sig := ctx.Signals
	s := ctx.Session
	c := archetype.Candidate{
		HypothesisID: a.HypothesisID(),
		Statement:    "Database saturation or unavailability degraded the service.",
	}
	if sig.Keywords["database"] {
		c.Score += 35
		c.Rationale = "Database-related symptoms appear in the evidence."
		c.Support = append(c.Support, sigpkg.EvidenceMatching(s, func(e *model.Evidence) bool {
			return sigpkg.MatchesAny(sigpkg.Haystack(e), sigpkg.Keywords["database"])
		})...)
	}
	if s.HasCategory(model.CategoryDatabaseEvents) {
		c.Score += 12
	}
	if sig.Keywords["latency"] && s.HasCategory(model.CategoryMetrics) {
		c.Score += 8
	}
	if sig.Lock.Present {
		c.Score -= 25
		if c.Score < 0 {
			c.Score = 0
		}
		c.Conflict = append(c.Conflict, sig.Lock.WaiterIDs...)
	}
	if sig.Lock.HealthyDatabaseMetrics {
		c.Score -= 15
		if c.Score < 0 {
			c.Score = 0
		}
	}
	return c
}

// LockContention scores row/table lock-wait queues.
type LockContention struct{}

func (LockContention) ID() string               { return "database-lock-contention" }
func (LockContention) Name() string             { return "Database Lock Contention" }
func (LockContention) Domain() archetype.Domain { return archetype.DomainData }
func (LockContention) Priority() int            { return 5 }
func (LockContention) HypothesisID() string     { return "hypothesis-lock-contention" }
func (LockContention) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (LockContention) ExpectedEvidence() []model.Category {
	return []model.Category{
		model.CategoryDatabaseEvents, model.CategoryTraceEvents,
		model.CategoryConfigurationChanges, model.CategoryMetrics,
	}
}
func (LockContention) TypicalSubcauses() []string {
	return []string{"row lock", "long transaction", "missing lock_timeout", "hot primary key"}
}
func (a LockContention) Score(ctx archetype.ScoreContext) archetype.Candidate {
	sig := ctx.Signals
	s := ctx.Session
	c := archetype.Candidate{
		HypothesisID: a.HypothesisID(),
		Statement:    "Database lock contention on a hot row or table blocked concurrent writers.",
	}
	if !sig.Lock.Present {
		return c
	}
	c.Score += 35
	c.Rationale = "Multiple database statements on the same entity completed together, indicating a lock queue."
	c.Support = append(c.Support, sig.Lock.HolderIDs...)
	c.Support = append(c.Support, sig.Lock.WaiterIDs...)
	if sig.Lock.SerializedRelease {
		c.Score += 20
	}
	if sig.Lock.MissingLockTimeouts {
		c.Score += 15
		c.Rationale += " Missing statement/lock timeouts allowed blocked writers to wait unbounded."
		c.Support = append(c.Support, sigpkg.EvidenceMatching(s, func(e *model.Evidence) bool {
			return e.Category == model.CategoryConfigurationChanges &&
				sigpkg.MentionsMissingTimeout(sigpkg.Haystack(e))
		})...)
	}
	if sig.Lock.HealthyDatabaseMetrics {
		c.Score += 12
		c.Support = append(c.Support, sigpkg.EvidenceMatching(s, func(e *model.Evidence) bool {
			return e.Category == model.CategoryMetrics && sigpkg.LooksLikeHealthyDatabaseMetrics(e)
		})...)
	}
	if sig.Keywords["latency"] {
		c.Score += 8
	}
	if s.HasCategory(model.CategoryTraceEvents) {
		c.Score += 6
		c.Support = append(c.Support, sigpkg.EvidenceMatching(s, func(e *model.Evidence) bool {
			return e.Category == model.CategoryTraceEvents
		})...)
	}
	return c
}
