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
	if sig.Keywords["dr"] {
		c.Score -= 15
		if c.Score < 0 {
			c.Score = 0
		}
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

func (DatabaseSaturation) SeedQuestions() []archetype.QuestionSeed {
	return []archetype.QuestionSeed{
		{
			ID: "database-healthy", Priority: 70,
			Title:       "Was database capacity healthy?",
			Description: "Healthy connections and CPU argue against saturation; lock contention can still cause latency.",
			Requires: []model.Category{
				model.CategoryDatabaseEvents, model.CategoryMetrics, model.CategoryApplicationLogs,
			},
			TriggerSignal: "database",
			Effects: []archetype.QuestionEffect{
				effect(true, archetype.EffectDecrease, "hypothesis-database-saturation", 30),
				effect(false, archetype.EffectIncrease, "hypothesis-database-saturation", 35),
			},
		},
		{
			ID: "database-saturated", Priority: 72,
			Title:       "Were database connections, CPU, or I/O saturated?",
			Description: "Pool exhaustion, high CPU, or replica lag indicate capacity saturation rather than lock waiting.",
			Requires:    []model.Category{model.CategoryDatabaseEvents, model.CategoryMetrics},
			DependsOn:   []string{"database-healthy"},
			Effects: []archetype.QuestionEffect{
				effect(true, archetype.EffectIncrease, "hypothesis-database-saturation", 30),
				effect(false, archetype.EffectDecrease, "hypothesis-database-saturation", 20),
				effect(true, archetype.EffectDecrease, "hypothesis-lock-contention", 20),
			},
		},
	}
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

func (LockContention) SeedQuestions() []archetype.QuestionSeed {
	return []archetype.QuestionSeed{
		{
			ID: "lock-contention-queue", Priority: 78,
			Title:       "Were database writes blocked on the same row?",
			Description: "Multiple statements on one entity completing together indicate a lock queue, not saturation.",
			Requires:      []model.Category{model.CategoryDatabaseEvents, model.CategoryTraceEvents},
			TriggerSignal: "database",
			Effects: []archetype.QuestionEffect{
				effect(true, archetype.EffectIncrease, "hypothesis-lock-contention", 25),
				effect(false, archetype.EffectDecrease, "hypothesis-lock-contention", 20),
				effect(true, archetype.EffectDecrease, "hypothesis-database-saturation", 25),
			},
		},
		{
			ID: "lock-timeouts-missing", Priority: 65,
			Title:       "Are statement or lock timeouts configured?",
			Description: "Missing lock_timeout or statement_timeout lets blocked writers wait unbounded behind a holder.",
			Requires:  []model.Category{model.CategoryConfigurationChanges, model.CategoryDatabaseEvents},
			DependsOn: []string{"lock-contention-queue"},
			Effects: []archetype.QuestionEffect{
				effect(true, archetype.EffectIncrease, "hypothesis-lock-contention", 15),
				effect(false, archetype.EffectDecrease, "hypothesis-lock-contention", 10),
			},
		},
	}
}

// CacheFailure scores Redis/Memcached outages and stampedes.
type CacheFailure struct{}

func (CacheFailure) ID() string               { return "cache-failure" }
func (CacheFailure) Name() string             { return "Cache Failure" }
func (CacheFailure) Domain() archetype.Domain { return archetype.DomainData }
func (CacheFailure) Priority() int            { return 4 }
func (CacheFailure) HypothesisID() string     { return "hypothesis-cache-failure" }
func (CacheFailure) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (CacheFailure) ExpectedEvidence() []model.Category {
	return []model.Category{model.CategoryMetrics, model.CategoryApplicationLogs, model.CategoryAlertEvents}
}
func (CacheFailure) TypicalSubcauses() []string {
	return []string{"cache unavailable", "cache stampede", "memory pressure", "eviction storm"}
}
func (a CacheFailure) Score(ctx archetype.ScoreContext) archetype.Candidate {
	sig := ctx.Signals
	c := scoreWith(ctx, scoreOpts{
		HypothesisID: a.HypothesisID(),
		Statement:    "Cache unavailability or stampede degraded service performance.",
		Keyword:      "cache",
		Base:         40,
		Penalties: []scorePenalty{
			{When: "database", Amount: 18},
		},
	})
	if sig.Categories[model.CategoryMetrics] > 0 {
		c.Score += 8
	}
	return c
}

func (CacheFailure) SeedQuestions() []archetype.QuestionSeed {
	return []archetype.QuestionSeed{{
		ID: "cache-unavailable", Priority: 66,
		Title:       "Was the cache unavailable or experiencing a stampede?",
		Description: "Redis/Memcached outages and eviction storms spike origin load.",
		Requires:      []model.Category{model.CategoryMetrics, model.CategoryApplicationLogs},
		TriggerSignal: "cache",
		Effects: []archetype.QuestionEffect{
			effect(true, archetype.EffectIncrease, "hypothesis-cache-failure", 28),
			effect(false, archetype.EffectDecrease, "hypothesis-cache-failure", 18),
		},
	}}
}

// DataCorruption scores integrity and partial-write failures.
type DataCorruption struct{}

func (DataCorruption) ID() string               { return "data-corruption" }
func (DataCorruption) Name() string             { return "Data Corruption" }
func (DataCorruption) Domain() archetype.Domain { return archetype.DomainData }
func (DataCorruption) Priority() int            { return 4 }
func (DataCorruption) HypothesisID() string     { return "hypothesis-data-corruption" }
func (DataCorruption) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (DataCorruption) ExpectedEvidence() []model.Category {
	return []model.Category{model.CategoryDatabaseEvents, model.CategoryApplicationLogs, model.CategoryHumanContext}
}
func (DataCorruption) TypicalSubcauses() []string {
	return []string{"partial write", "checksum mismatch", "migration bug", "replication lag corruption"}
}
func (a DataCorruption) Score(ctx archetype.ScoreContext) archetype.Candidate {
	return scoreWith(ctx, scoreOpts{
		HypothesisID: a.HypothesisID(),
		Statement:    "Data corruption or partial writes compromised integrity.",
		Keyword:      "datacorruption",
		Base:         44,
		Penalties: []scorePenalty{
			{When: "human", Unless: "datacorruption", Amount: 20},
		},
	})
}

func (DataCorruption) SeedQuestions() []archetype.QuestionSeed {
	return []archetype.QuestionSeed{{
		ID: "data-corrupted", Priority: 74,
		Title:       "Is data corruption or partial-write damage suspected?",
		Description: "Checksum failures and audit anomalies indicate integrity loss.",
		Requires:      []model.Category{model.CategoryDatabaseEvents, model.CategoryApplicationLogs},
		TriggerSignal: "datacorruption",
		Effects: []archetype.QuestionEffect{
			effect(true, archetype.EffectIncrease, "hypothesis-data-corruption", 32),
			effect(false, archetype.EffectDecrease, "hypothesis-data-corruption", 22),
		},
	}}
}
