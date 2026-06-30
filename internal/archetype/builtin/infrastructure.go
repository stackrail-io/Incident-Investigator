package builtin

import (
	"github.com/stackrail/incident-investigator/internal/archetype"
	"github.com/stackrail/incident-investigator/internal/model"
	sigpkg "github.com/stackrail/incident-investigator/internal/signals"
)

// InfrastructureFailure scores cloud, hypervisor, and node-level faults.
type InfrastructureFailure struct{}

func (InfrastructureFailure) ID() string               { return "infrastructure-failure" }
func (InfrastructureFailure) Name() string             { return "Infrastructure Failure" }
func (InfrastructureFailure) Domain() archetype.Domain { return archetype.DomainInfrastructure }
func (InfrastructureFailure) Priority() int            { return 5 }
func (InfrastructureFailure) HypothesisID() string     { return "hypothesis-infrastructure-failure" }
func (InfrastructureFailure) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (InfrastructureFailure) ExpectedEvidence() []model.Category {
	return []model.Category{model.CategoryInfrastructureEvents, model.CategoryAlertEvents, model.CategoryMetrics}
}
func (InfrastructureFailure) TypicalSubcauses() []string {
	return []string{"node unavailable", "zone failure", "hypervisor fault", "hardware alert", "instance reboot"}
}
func (a InfrastructureFailure) Score(ctx archetype.ScoreContext) archetype.Candidate {
	sig := ctx.Signals
	c := scoreWith(ctx, scoreOpts{
		HypothesisID: a.HypothesisID(),
		Statement:    "Cloud or infrastructure failure disrupted underlying compute or storage.",
		Keyword:      "infrastructure",
		Base:         42,
		Penalties: []scorePenalty{
			{When: "kubernetes", Unless: "infrastructure", Amount: 18},
			{When: "regional", Unless: "infrastructure", Amount: 12},
		},
	})
	if sig.Categories[model.CategoryInfrastructureEvents] > 0 {
		c.Score += 10
	}
	return c
}

func (InfrastructureFailure) SeedQuestions() []archetype.QuestionSeed {
	return []archetype.QuestionSeed{{
		ID: "node-unavailable", Priority: 74,
		Title:       "Was a node, hypervisor, or hardware component unavailable?",
		Description: "Cloud events and hardware alerts often precede workload failures.",
		Requires:      []model.Category{model.CategoryInfrastructureEvents, model.CategoryAlertEvents},
		TriggerSignal: "infrastructure",
		Effects: []archetype.QuestionEffect{
			effect(true, archetype.EffectIncrease, "hypothesis-infrastructure-failure", 30),
			effect(false, archetype.EffectDecrease, "hypothesis-infrastructure-failure", 20),
		},
	}}
}

// ResourceExhaustion scores memory, CPU, and restart signatures.
type ResourceExhaustion struct{}

func (ResourceExhaustion) ID() string               { return "resource-exhaustion" }
func (ResourceExhaustion) Name() string             { return "Resource Exhaustion" }
func (ResourceExhaustion) Domain() archetype.Domain { return archetype.DomainInfrastructure }
func (ResourceExhaustion) Priority() int            { return 5 }
func (ResourceExhaustion) HypothesisID() string     { return "hypothesis-resource-exhaustion" }
func (ResourceExhaustion) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (ResourceExhaustion) ExpectedEvidence() []model.Category {
	return []model.Category{model.CategoryMetrics, model.CategoryInfrastructureEvents, model.CategoryApplicationLogs}
}
func (ResourceExhaustion) TypicalSubcauses() []string {
	return []string{"memory leak", "oom", "cpu saturation", "disk full", "file descriptors"}
}
func (a ResourceExhaustion) Score(ctx archetype.ScoreContext) archetype.Candidate {
	sig := ctx.Signals
	s := ctx.Session
	c := archetype.Candidate{
		HypothesisID: a.HypothesisID(),
		Statement:    "Resource exhaustion (memory/CPU) caused crashes or throttling.",
	}
	if sig.Keywords["memory"] {
		c.Score += 35
		c.Rationale = "Memory pressure symptoms appear in the evidence."
		c.Support = append(c.Support, sigpkg.EvidenceMatching(s, func(e *model.Evidence) bool {
			return sigpkg.MatchesAny(sigpkg.Haystack(e), sigpkg.Keywords["memory"])
		})...)
	}
	if sig.Keywords["restart"] {
		c.Score += 12
	}
	return c
}

func (ResourceExhaustion) SeedQuestions() []archetype.QuestionSeed {
	return []archetype.QuestionSeed{
		{
			ID: "pods-restarted", Priority: 65,
			Title:       "Did pods restart or crash loop?",
			Description: "Restarts after a config change point to misconfiguration or resource limits.",
			Requires:      []model.Category{model.CategoryInfrastructureEvents, model.CategoryApplicationLogs},
			DependsOn:     []string{"config-changed"},
			TriggerSignal: "restart",
			Generates:     []string{"traffic-shifted"},
			Effects: []archetype.QuestionEffect{
				effect(true, archetype.EffectIncrease, "hypothesis-resource-exhaustion", 20),
			},
		},
		{
			ID: "memory-pressure", Priority: 68,
			Title:       "Was memory pressure or OOM observed?",
			Description: "Heap growth, OOM kills, and eviction events indicate resource exhaustion.",
			Requires:      []model.Category{model.CategoryMetrics, model.CategoryInfrastructureEvents, model.CategoryApplicationLogs},
			TriggerSignal: "memory",
			Effects: []archetype.QuestionEffect{
				effect(true, archetype.EffectIncrease, "hypothesis-resource-exhaustion", 30),
				effect(false, archetype.EffectDecrease, "hypothesis-resource-exhaustion", 20),
			},
		},
	}
}

// StorageFailure scores disk, volume, and mount failures.
type StorageFailure struct{}

func (StorageFailure) ID() string               { return "storage-failure" }
func (StorageFailure) Name() string             { return "Storage Failure" }
func (StorageFailure) Domain() archetype.Domain { return archetype.DomainInfrastructure }
func (StorageFailure) Priority() int            { return 4 }
func (StorageFailure) HypothesisID() string     { return "hypothesis-storage-failure" }
func (StorageFailure) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (StorageFailure) ExpectedEvidence() []model.Category {
	return []model.Category{model.CategoryInfrastructureEvents, model.CategoryMetrics, model.CategoryAlertEvents}
}
func (StorageFailure) TypicalSubcauses() []string {
	return []string{"disk full", "mount failed", "io latency", "ebs unavailable", "pvc bound failure"}
}
func (a StorageFailure) Score(ctx archetype.ScoreContext) archetype.Candidate {
	sig := ctx.Signals
	c := scoreWith(ctx, scoreOpts{
		HypothesisID: a.HypothesisID(),
		Statement:    "Storage unavailability or saturation caused read/write failures.",
		Keyword:      "storage",
		Base:         42,
		Penalties: []scorePenalty{
			{When: "memory", Amount: 20},
		},
	})
	if sig.Categories[model.CategoryInfrastructureEvents] > 0 {
		c.Score += 8
	}
	return c
}

func (StorageFailure) SeedQuestions() []archetype.QuestionSeed {
	return []archetype.QuestionSeed{{
		ID: "storage-unavailable", Priority: 70,
		Title:       "Was storage unavailable, full, or experiencing high IO latency?",
		Description: "Disk, EBS, PVC, and SAN faults block reads and writes.",
		Requires:      []model.Category{model.CategoryInfrastructureEvents, model.CategoryMetrics},
		TriggerSignal: "storage",
		Effects: []archetype.QuestionEffect{
			effect(true, archetype.EffectIncrease, "hypothesis-storage-failure", 30),
			effect(false, archetype.EffectDecrease, "hypothesis-storage-failure", 20),
		},
	}}
}

// RegionalFailure scores availability-zone and region outages.
type RegionalFailure struct{}

func (RegionalFailure) ID() string               { return "regional-failure" }
func (RegionalFailure) Name() string             { return "Regional / Availability Zone Failure" }
func (RegionalFailure) Domain() archetype.Domain { return archetype.DomainInfrastructure }
func (RegionalFailure) Priority() int            { return 4 }
func (RegionalFailure) HypothesisID() string     { return "hypothesis-regional-failure" }
func (RegionalFailure) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (RegionalFailure) ExpectedEvidence() []model.Category {
	return []model.Category{model.CategoryInfrastructureEvents, model.CategoryNetworkEvents, model.CategoryMetrics}
}
func (RegionalFailure) TypicalSubcauses() []string {
	return []string{"az failure", "single-region dependency", "capacity loss in region"}
}
func (a RegionalFailure) Score(ctx archetype.ScoreContext) archetype.Candidate {
	sig := ctx.Signals
	c := scoreWith(ctx, scoreOpts{
		HypothesisID: a.HypothesisID(),
		Statement:    "A regional or availability-zone failure isolated capacity.",
		Keyword:      "regional",
		Base:         42,
		Penalties: []scorePenalty{
			{When: "infrastructure", Unless: "regional", Amount: 14},
		},
	})
	if sig.Categories[model.CategoryInfrastructureEvents] > 0 {
		c.Score += 8
	}
	return c
}

func (RegionalFailure) SeedQuestions() []archetype.QuestionSeed {
	return []archetype.QuestionSeed{{
		ID: "az-failure", Priority: 76,
		Title:       "Did an availability zone or region fail?",
		Description: "Regional outages isolate capacity when workloads are single-region.",
		Requires:      []model.Category{model.CategoryInfrastructureEvents, model.CategoryNetworkEvents},
		TriggerSignal: "regional",
		Effects: []archetype.QuestionEffect{
			effect(true, archetype.EffectIncrease, "hypothesis-regional-failure", 32),
			effect(false, archetype.EffectDecrease, "hypothesis-regional-failure", 22),
		},
	}}
}
