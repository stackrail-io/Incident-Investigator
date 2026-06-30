package builtin

import (
	"github.com/stackrail/incident-investigator/internal/archetype"
	"github.com/stackrail/incident-investigator/internal/model"
	sigpkg "github.com/stackrail/incident-investigator/internal/signals"
)

// RetryStorm scores retry amplification and cascading failures.
type RetryStorm struct{}

func (RetryStorm) ID() string               { return "retry-storm" }
func (RetryStorm) Name() string             { return "Retry Storm / Cascading Failure" }
func (RetryStorm) Domain() archetype.Domain { return archetype.DomainOperations }
func (RetryStorm) Priority() int            { return 5 }
func (RetryStorm) HypothesisID() string     { return "hypothesis-retry-storm" }
func (RetryStorm) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (RetryStorm) ExpectedEvidence() []model.Category {
	return []model.Category{model.CategoryApplicationLogs, model.CategoryMetrics, model.CategoryTraceEvents}
}
func (RetryStorm) TypicalSubcauses() []string {
	return []string{"aggressive retries", "missing circuit breaker", "cascading timeout", "thundering herd"}
}
func (a RetryStorm) Score(ctx archetype.ScoreContext) archetype.Candidate {
	sig := ctx.Signals
	s := ctx.Session
	c := archetype.Candidate{
		HypothesisID: a.HypothesisID(),
		Statement:    "A retry storm / cascading failure amplified a smaller fault.",
	}
	if sig.Keywords["retry"] {
		c.Score += 32
		c.Rationale = "Retry-amplification symptoms appear in the evidence."
		c.Support = append(c.Support, sigpkg.EvidenceMatching(s, func(e *model.Evidence) bool {
			return sigpkg.MatchesAny(sigpkg.Haystack(e), sigpkg.Keywords["retry"])
		})...)
	}
	if sig.Keywords["latency"] {
		c.Score += 8
	}
	return c
}

func (RetryStorm) SeedQuestions() []archetype.QuestionSeed {
	return []archetype.QuestionSeed{
		{
			ID: "latency-before-retries", Priority: 60,
			Title:       "Did latency begin before retries amplified?",
			Description: "Retries that follow latency suggest amplification; latency that follows retries does not.",
			Requires: []model.Category{model.CategoryMetrics, model.CategoryTraceEvents},
			Effects: []archetype.QuestionEffect{
				effect(true, archetype.EffectIncrease, "hypothesis-retry-storm", 20),
				effect(false, archetype.EffectDecrease, "hypothesis-retry-storm", 25),
			},
		},
		{
			ID: "traffic-shifted", Priority: 55,
			Title:       "Did traffic or load shift during the incident?",
			Description: "Sudden traffic shifts can expose latent bottlenecks and amplify retries.",
			Requires:  []model.Category{model.CategoryMetrics, model.CategoryTraceEvents},
			DependsOn: []string{"pods-restarted"},
			Effects: []archetype.QuestionEffect{
				effect(true, archetype.EffectIncrease, "hypothesis-retry-storm", 15),
			},
		},
	}
}

// DependencyFailure scores downstream or third-party dependency unavailability.
type DependencyFailure struct{}

func (DependencyFailure) ID() string               { return "dependency-failure" }
func (DependencyFailure) Name() string             { return "Dependency Failure" }
func (DependencyFailure) Domain() archetype.Domain { return archetype.DomainOperations }
func (DependencyFailure) Priority() int            { return 5 }
func (DependencyFailure) HypothesisID() string     { return "hypothesis-dependency-failure" }
func (DependencyFailure) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (DependencyFailure) ExpectedEvidence() []model.Category {
	return []model.Category{
		model.CategoryApplicationLogs, model.CategoryTraceEvents,
		model.CategoryMetrics, model.CategoryAlertEvents,
	}
}
func (DependencyFailure) TypicalSubcauses() []string {
	return []string{"downstream timeout", "service unavailable", "circuit open", "queue backlog"}
}
func (a DependencyFailure) Score(ctx archetype.ScoreContext) archetype.Candidate {
	sig := ctx.Signals
	s := ctx.Session
	c := archetype.Candidate{
		HypothesisID: a.HypothesisID(),
		Statement:    "A downstream dependency failure caused the incident.",
	}
	if sig.Keywords["dependency"] {
		c.Score += 38
		c.Rationale = "Downstream or dependency failure symptoms appear in the evidence."
		c.Support = append(c.Support, sigpkg.EvidenceMatching(s, func(e *model.Evidence) bool {
			return sigpkg.MatchesAny(sigpkg.Haystack(e), sigpkg.Keywords["dependency"])
		})...)
	}
	if sig.Keywords["error"] && s.HasCategory(model.CategoryTraceEvents) {
		c.Score += 12
		if c.Rationale == "" {
			c.Rationale = "Errors correlated with distributed trace evidence suggest a dependency fault."
		}
	}
	if sig.Keywords["latency"] && sig.Keywords["dependency"] {
		c.Score += 8
	}
	if sig.Keywords["retry"] {
		c.Score -= 30
		if c.Score < 0 {
			c.Score = 0
		}
	}
	if sig.Keywords["external"] {
		c.Score -= 25
		if c.Score < 0 {
			c.Score = 0
		}
	}
	return c
}

func (DependencyFailure) SeedQuestions() []archetype.QuestionSeed {
	return []archetype.QuestionSeed{
		{
			ID: "dependency-unavailable", Priority: 74,
			Title:       "Was a downstream dependency unavailable?",
			Description: "Timeouts and errors from upstream callers often indicate a failed dependency.",
			Requires:      []model.Category{model.CategoryApplicationLogs, model.CategoryTraceEvents},
			TriggerSignal: "dependency",
			Effects: []archetype.QuestionEffect{
				effect(true, archetype.EffectIncrease, "hypothesis-dependency-failure", 30),
				effect(false, archetype.EffectDecrease, "hypothesis-dependency-failure", 20),
			},
		},
	}
}

// MessagingFailure scores queue and broker failures.
type MessagingFailure struct{}

func (MessagingFailure) ID() string               { return "messaging-failure" }
func (MessagingFailure) Name() string             { return "Messaging Failure" }
func (MessagingFailure) Domain() archetype.Domain { return archetype.DomainOperations }
func (MessagingFailure) Priority() int            { return 4 }
func (MessagingFailure) HypothesisID() string     { return "hypothesis-messaging-failure" }
func (MessagingFailure) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (MessagingFailure) ExpectedEvidence() []model.Category {
	return []model.Category{model.CategoryApplicationLogs, model.CategoryMetrics, model.CategoryTraceEvents}
}
func (MessagingFailure) TypicalSubcauses() []string {
	return []string{"consumer lag", "queue buildup", "poison message", "broker unavailable"}
}
func (a MessagingFailure) Score(ctx archetype.ScoreContext) archetype.Candidate {
	sig := ctx.Signals
	c := scoreWith(ctx, scoreOpts{
		HypothesisID: a.HypothesisID(),
		Statement:    "A messaging queue or broker failure disrupted asynchronous processing.",
		Keyword:      "messaging",
		Base:         42,
		Penalties: []scorePenalty{
			{When: "retry", Unless: "messaging", Amount: 22},
		},
	})
	if sig.Categories[model.CategoryMetrics] > 0 {
		c.Score += 8
	}
	return c
}

func (MessagingFailure) SeedQuestions() []archetype.QuestionSeed {
	return []archetype.QuestionSeed{{
		ID: "consumer-lag", Priority: 68,
		Title:       "Is consumer lag or queue buildup present?",
		Description: "Kafka, RabbitMQ, and SQS backlogs indicate messaging-layer faults.",
		Requires:      []model.Category{model.CategoryApplicationLogs, model.CategoryMetrics},
		TriggerSignal: "messaging",
		Effects: []archetype.QuestionEffect{
			effect(true, archetype.EffectIncrease, "hypothesis-messaging-failure", 28),
			effect(false, archetype.EffectDecrease, "hypothesis-messaging-failure", 18),
		},
	}}
}

// CapacityPlanning scores traffic spikes and autoscaling failures.
type CapacityPlanning struct{}

func (CapacityPlanning) ID() string               { return "capacity-planning" }
func (CapacityPlanning) Name() string             { return "Capacity Planning Failure" }
func (CapacityPlanning) Domain() archetype.Domain { return archetype.DomainOperations }
func (CapacityPlanning) Priority() int            { return 4 }
func (CapacityPlanning) HypothesisID() string     { return "hypothesis-capacity-planning" }
func (CapacityPlanning) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (CapacityPlanning) ExpectedEvidence() []model.Category {
	return []model.Category{model.CategoryMetrics, model.CategoryInfrastructureEvents, model.CategoryAlertEvents}
}
func (CapacityPlanning) TypicalSubcauses() []string {
	return []string{"traffic spike", "autoscaling delayed", "quota exceeded", "under-provisioned"}
}
func (a CapacityPlanning) Score(ctx archetype.ScoreContext) archetype.Candidate {
	sig := ctx.Signals
	s := ctx.Session
	c := archetype.Candidate{
		HypothesisID: a.HypothesisID(),
		Statement:    "Insufficient capacity or delayed scaling failed to absorb load.",
	}
	if sig.Keywords["capacity"] {
		c.Score += 38
		c.Rationale = "Capacity or autoscaling stress symptoms appear in the evidence."
		c.Support = append(c.Support, sigpkg.EvidenceMatching(s, func(e *model.Evidence) bool {
			return sigpkg.MatchesAny(sigpkg.Haystack(e), sigpkg.Keywords["capacity"])
		})...)
	}
	if sig.Keywords["latency"] && s.HasCategory(model.CategoryMetrics) && !sig.Keywords["database"] {
		c.Score += 10
	}
	return c
}

func (CapacityPlanning) SeedQuestions() []archetype.QuestionSeed {
	return []archetype.QuestionSeed{
		{
			ID: "capacity-exceeded", Priority: 66,
			Title:       "Was capacity exceeded or scaling delayed?",
			Description: "Traffic spikes and autoscaling lag produce latency without code changes.",
			Requires:      []model.Category{model.CategoryMetrics, model.CategoryInfrastructureEvents},
			TriggerSignal: "capacity",
			Effects: []archetype.QuestionEffect{
				effect(true, archetype.EffectIncrease, "hypothesis-capacity-planning", 28),
				effect(false, archetype.EffectDecrease, "hypothesis-capacity-planning", 18),
			},
		},
	}
}

// DRFailoverFailure scores disaster-recovery and failover faults.
type DRFailoverFailure struct{}

func (DRFailoverFailure) ID() string               { return "dr-failover-failure" }
func (DRFailoverFailure) Name() string             { return "Disaster Recovery / Failover Failure" }
func (DRFailoverFailure) Domain() archetype.Domain { return archetype.DomainOperations }
func (DRFailoverFailure) Priority() int            { return 3 }
func (DRFailoverFailure) HypothesisID() string     { return "hypothesis-dr-failover-failure" }
func (DRFailoverFailure) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (DRFailoverFailure) ExpectedEvidence() []model.Category {
	return []model.Category{model.CategoryInfrastructureEvents, model.CategoryDatabaseEvents, model.CategoryApplicationLogs}
}
func (DRFailoverFailure) TypicalSubcauses() []string {
	return []string{"failover incomplete", "split brain", "dr drill gone wrong", "replication lag"}
}
func (a DRFailoverFailure) Score(ctx archetype.ScoreContext) archetype.Candidate {
	sig := ctx.Signals
	c := scoreWith(ctx, scoreOpts{
		HypothesisID: a.HypothesisID(),
		Statement:    "A disaster-recovery or failover procedure failed or was incomplete.",
		Keyword:      "dr",
		Base:         40,
		Penalties: []scorePenalty{
			{When: "database", Unless: "dr", Amount: 18},
		},
	})
	if sig.Categories[model.CategoryInfrastructureEvents] > 0 {
		c.Score += 8
	}
	return c
}

func (DRFailoverFailure) SeedQuestions() []archetype.QuestionSeed {
	return []archetype.QuestionSeed{{
		ID: "failover-incomplete", Priority: 64,
		Title:       "Did failover or disaster recovery fail or stall?",
		Description: "Incomplete failover and split-brain leave services partially unavailable.",
		Requires:      []model.Category{model.CategoryInfrastructureEvents, model.CategoryDatabaseEvents},
		TriggerSignal: "dr",
		Effects: []archetype.QuestionEffect{
			effect(true, archetype.EffectIncrease, "hypothesis-dr-failover-failure", 28),
			effect(false, archetype.EffectDecrease, "hypothesis-dr-failover-failure", 18),
		},
	}}
}

// ObservabilityFailure scores missing telemetry and blind spots.
type ObservabilityFailure struct{}

func (ObservabilityFailure) ID() string               { return "observability-failure" }
func (ObservabilityFailure) Name() string             { return "Observability Failure" }
func (ObservabilityFailure) Domain() archetype.Domain { return archetype.DomainOperations }
func (ObservabilityFailure) Priority() int            { return 3 }
func (ObservabilityFailure) HypothesisID() string     { return "hypothesis-observability-failure" }
func (ObservabilityFailure) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (ObservabilityFailure) ExpectedEvidence() []model.Category {
	return []model.Category{model.CategoryHumanContext, model.CategoryApplicationLogs, model.CategoryMetrics}
}
func (ObservabilityFailure) TypicalSubcauses() []string {
	return []string{"agent failure", "sampling gap", "missing logs", "missing metrics"}
}
func (a ObservabilityFailure) Score(ctx archetype.ScoreContext) archetype.Candidate {
	return scoreWith(ctx, scoreOpts{
		HypothesisID: a.HypothesisID(),
		Statement:    "Observability gaps created blind spots that delayed detection or diagnosis.",
		Keyword:      "observability",
		Base:         36,
		Penalties: []scorePenalty{
			{When: "human", Unless: "observability", Amount: 15},
		},
	})
}

func (ObservabilityFailure) SeedQuestions() []archetype.QuestionSeed {
	return []archetype.QuestionSeed{{
		ID: "observability-gap", Priority: 58,
		Title:       "Are logs, metrics, or traces missing for the incident window?",
		Description: "Telemetry gaps create blind spots that delay root-cause identification.",
		Requires:      []model.Category{model.CategoryHumanContext, model.CategoryApplicationLogs},
		TriggerSignal: "observability",
		Effects: []archetype.QuestionEffect{
			effect(true, archetype.EffectIncrease, "hypothesis-observability-failure", 24),
			effect(false, archetype.EffectDecrease, "hypothesis-observability-failure", 14),
		},
	}}
}
