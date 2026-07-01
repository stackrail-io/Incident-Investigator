package builtin

import (
	"github.com/stackrail/incident-investigator/internal/archetype"
	"github.com/stackrail/incident-investigator/internal/model"
	sigpkg "github.com/stackrail/incident-investigator/internal/signals"
)

// DeploymentCaused scores deployment-before-symptom signatures.
type DeploymentCaused struct{}

func (DeploymentCaused) ID() string               { return "deployment-failure" }
func (DeploymentCaused) Name() string             { return "Deployment Failure" }
func (DeploymentCaused) Domain() archetype.Domain { return archetype.DomainApplication }
func (DeploymentCaused) Priority() int            { return 5 }
func (DeploymentCaused) HypothesisID() string     { return "hypothesis-deployment-caused" }
func (DeploymentCaused) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (DeploymentCaused) ExpectedEvidence() []model.Category {
	return []model.Category{
		model.CategoryDeploymentEvents, model.CategoryApplicationLogs,
		model.CategoryAlertEvents, model.CategoryMetrics,
	}
}
func (DeploymentCaused) TypicalSubcauses() []string {
	return []string{"bad code", "bad config", "feature flag", "wrong image", "missing migration"}
}
func (a DeploymentCaused) Score(ctx archetype.ScoreContext) archetype.Candidate {
	sig := ctx.Signals
	s := ctx.Session
	c := archetype.Candidate{
		HypothesisID: a.HypothesisID(),
		Statement:    "A recent deployment introduced the regression that caused the incident.",
	}
	if sig.FirstDeployment == nil {
		return c
	}
	c.Score = 5
	c.Support = append(c.Support, sig.FirstDeployment.ID)
	if sig.DeployBeforeIncident {
		c.Score += 45
		c.Rationale = "A deployment was observed shortly before the first symptom."
		if sig.IncidentOnset != nil {
			c.Support = append(c.Support, sig.IncidentOnset.ID)
		}
	}
	if sig.Keywords["config"] && s.HasCategory(model.CategoryConfigurationChanges) {
		c.Score += 12
		c.Support = append(c.Support, sigpkg.EvidenceMatching(s, func(e *model.Evidence) bool {
			return e.Category == model.CategoryConfigurationChanges
		})...)
	}
	if sig.Keywords["restart"] {
		c.Score += 10
	}
	if ctx.DeployContradicted() {
		c.Score = 2
		c.Rationale = "Deployment timing contradicts a causal role (it happened after onset)."
		if sig.IncidentOnset != nil {
			c.Conflict = append(c.Conflict, sig.IncidentOnset.ID)
		}
	}
	return c
}

func (DeploymentCaused) SeedQuestions() []archetype.QuestionSeed {
	return []archetype.QuestionSeed{
		{
			ID: "deploy-before-errors", Priority: 85,
			Title:       "Did deployment happen before errors?",
			Description: "Establish temporal ordering between the most recent deployment and the first symptom.",
			Requires: []model.Category{
				model.CategoryDeploymentEvents, model.CategoryApplicationLogs, model.CategoryAlertEvents,
			},
			Effects: []archetype.QuestionEffect{
				effect(true, archetype.EffectIncrease, "hypothesis-deployment-caused", 25),
				effect(false, archetype.EffectDecrease, "hypothesis-deployment-caused", 40),
				effect(false, archetype.EffectIncrease, "hypothesis-deployment-unrelated", 20),
			},
		},
		{
			ID: "deploy-after-incident", Priority: 80,
			Title:       "Did deployment occur after incident onset?",
			Description: "A deployment that lands after symptoms began contradicts deploy-caused theories.",
			Requires: []model.Category{
				model.CategoryDeploymentEvents, model.CategoryAlertEvents, model.CategoryApplicationLogs,
			},
			DependsOn: []string{"deploy-before-errors"},
			Effects: []archetype.QuestionEffect{
				effect(true, archetype.EffectIncrease, "hypothesis-deployment-unrelated", 30),
				effect(false, archetype.EffectDecrease, "hypothesis-deployment-unrelated", 15),
				effect(true, archetype.EffectDecrease, "hypothesis-deployment-caused", 35),
			},
		},
		{
			ID: "rollback-restored-service", Priority: 75,
			Title:       "Did rollback restore service?",
			Description: "Recovery immediately after rollback strongly implicates the preceding deployment.",
			Requires:    []model.Category{model.CategoryDeploymentEvents, model.CategoryMetrics},
			DependsOn:   []string{"deploy-before-errors"},
			Effects: []archetype.QuestionEffect{
				effect(true, archetype.EffectIncrease, "hypothesis-deployment-caused", 15),
				effect(false, archetype.EffectDecrease, "hypothesis-deployment-caused", 20),
			},
		},
	}
}

// ConfigurationChange scores config and feature-flag changes.
type ConfigurationChange struct{}

func (ConfigurationChange) ID() string               { return "configuration-drift" }
func (ConfigurationChange) Name() string             { return "Configuration Drift" }
func (ConfigurationChange) Domain() archetype.Domain { return archetype.DomainApplication }
func (ConfigurationChange) Priority() int            { return 5 }
func (ConfigurationChange) HypothesisID() string     { return "hypothesis-configuration-change" }
func (ConfigurationChange) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (ConfigurationChange) ExpectedEvidence() []model.Category {
	return []model.Category{model.CategoryConfigurationChanges, model.CategoryDeploymentEvents}
}
func (ConfigurationChange) TypicalSubcauses() []string {
	return []string{"configmap change", "secret rotation", "env var", "feature flag", "helm values"}
}
func (a ConfigurationChange) Score(ctx archetype.ScoreContext) archetype.Candidate {
	sig := ctx.Signals
	s := ctx.Session
	c := archetype.Candidate{
		HypothesisID: a.HypothesisID(),
		Statement:    "A configuration or feature-flag change triggered the incident.",
	}
	if sig.Keywords["config"] && !sig.Keywords["featureflag"] && sigpkg.HasAffirmativeConfigurationEvidence(s) {
		c.Score += 30
		c.Rationale = "Configuration-change symptoms appear in the evidence."
		c.Support = append(c.Support, sigpkg.EvidenceMatching(s, func(e *model.Evidence) bool {
			if sigpkg.EvidenceRulesOutConfig(e) {
				return false
			}
			return e.Category == model.CategoryConfigurationChanges ||
				sigpkg.MatchesAny(sigpkg.Haystack(e), sigpkg.Keywords["config"])
		})...)
	}
	if sigpkg.HasAffirmativeConfigurationEvidence(s) && !sig.Keywords["kubernetes"] {
		c.Score += 12
		if c.Rationale == "" {
			c.Rationale = "A configuration change was recorded before symptoms appeared."
		}
	}
	if sig.Keywords["kubernetes"] {
		c.Score -= 20
		if c.Score < 0 {
			c.Score = 0
		}
	}
	return c
}

func (ConfigurationChange) SeedQuestions() []archetype.QuestionSeed {
	return []archetype.QuestionSeed{
		{
			ID: "config-changed", Priority: 70,
			Title:       "Was configuration changed?",
			Description: "Feature flags, env vars, and connection-pool settings are common incident triggers.",
			Requires:      []model.Category{model.CategoryConfigurationChanges, model.CategoryDeploymentEvents},
			TriggerSignal: "config",
			Generates:     []string{"pods-restarted"},
			Effects: []archetype.QuestionEffect{
				effect(true, archetype.EffectIncrease, "hypothesis-configuration-change", 30),
				effect(false, archetype.EffectDecrease, "hypothesis-configuration-change", 15),
			},
		},
	}
}

// PerformanceRegression scores latency or throughput regression without saturation.
type PerformanceRegression struct{}

func (PerformanceRegression) ID() string               { return "performance-regression" }
func (PerformanceRegression) Name() string             { return "Performance Regression" }
func (PerformanceRegression) Domain() archetype.Domain { return archetype.DomainApplication }
func (PerformanceRegression) Priority() int            { return 5 }
func (PerformanceRegression) HypothesisID() string     { return "hypothesis-performance-regression" }
func (PerformanceRegression) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (PerformanceRegression) ExpectedEvidence() []model.Category {
	return []model.Category{model.CategoryMetrics, model.CategoryTraceEvents, model.CategoryApplicationLogs}
}
func (PerformanceRegression) TypicalSubcauses() []string {
	return []string{"code regression", "hot path", "lock contention", "gc pause", "inefficient query"}
}
func (a PerformanceRegression) Score(ctx archetype.ScoreContext) archetype.Candidate {
	sig := ctx.Signals
	s := ctx.Session
	c := archetype.Candidate{
		HypothesisID: a.HypothesisID(),
		Statement:    "A performance regression degraded latency or throughput.",
	}
	if sig.Keywords["performance"] {
		c.Score += 36
		c.Rationale = "Performance regression symptoms appear in the evidence."
		c.Support = append(c.Support, sigpkg.EvidenceMatching(s, func(e *model.Evidence) bool {
			return sigpkg.MatchesAny(sigpkg.Haystack(e), sigpkg.Keywords["performance"])
		})...)
	}
	if sig.Keywords["latency"] && s.HasCategory(model.CategoryMetrics) && !sig.Keywords["database"] && !sig.Keywords["retry"] {
		c.Score += 22
		if c.Rationale == "" {
			c.Rationale = "Latency degradation appears without database saturation or retry amplification."
		}
	}
	return c
}

func (PerformanceRegression) SeedQuestions() []archetype.QuestionSeed {
	return []archetype.QuestionSeed{
		{
			ID: "latency-regression", Priority: 68,
			Title:       "Did latency regress without saturation or retries?",
			Description: "Isolated latency spikes can indicate a code or query regression.",
			Requires:      []model.Category{model.CategoryMetrics, model.CategoryTraceEvents},
			TriggerSignal: "performance",
			Effects: []archetype.QuestionEffect{
				effect(true, archetype.EffectIncrease, "hypothesis-performance-regression", 28),
				effect(false, archetype.EffectDecrease, "hypothesis-performance-regression", 18),
			},
		},
	}
}

// HumanError scores manual or operational mistakes.
type HumanError struct{}

func (HumanError) ID() string               { return "human-error" }
func (HumanError) Name() string             { return "Human / Operational Error" }
func (HumanError) Domain() archetype.Domain { return archetype.DomainApplication }
func (HumanError) Priority() int            { return 5 }
func (HumanError) HypothesisID() string     { return "hypothesis-human-error" }
func (HumanError) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (HumanError) ExpectedEvidence() []model.Category {
	return []model.Category{model.CategoryHumanContext, model.CategoryConfigurationChanges, model.CategoryApplicationLogs}
}
func (HumanError) TypicalSubcauses() []string {
	return []string{"manual change", "wrong procedure", "runbook skipped", "fat finger"}
}
func (a HumanError) Score(ctx archetype.ScoreContext) archetype.Candidate {
	sig := ctx.Signals
	s := ctx.Session
	c := archetype.Candidate{
		HypothesisID: a.HypothesisID(),
		Statement:    "A human or operational error triggered the incident.",
	}
	if s.HasCategory(model.CategoryHumanContext) {
		c.Score += 40
		c.Rationale = "Human context evidence describes an operational mistake."
		c.Support = append(c.Support, sigpkg.EvidenceMatching(s, func(e *model.Evidence) bool {
			return e.Category == model.CategoryHumanContext
		})...)
	}
	if sig.Keywords["human"] {
		c.Score += 28
		if c.Rationale == "" {
			c.Rationale = "Evidence suggests a manual or operational error."
		}
	}
	return c
}

func (HumanError) SeedQuestions() []archetype.QuestionSeed {
	return []archetype.QuestionSeed{
		{
			ID: "manual-change", Priority: 70,
			Title:       "Was a manual or operational change involved?",
			Description: "Operator actions and runbook deviations are common root causes.",
			Requires:      []model.Category{model.CategoryHumanContext, model.CategoryApplicationLogs},
			TriggerSignal: "human",
			Effects: []archetype.QuestionEffect{
				effect(true, archetype.EffectIncrease, "hypothesis-human-error", 32),
				effect(false, archetype.EffectDecrease, "hypothesis-human-error", 18),
			},
		},
	}
}

// APIContractFailure scores schema and version mismatches.
type APIContractFailure struct{}

func (APIContractFailure) ID() string               { return "api-contract-failure" }
func (APIContractFailure) Name() string             { return "API Contract Failure" }
func (APIContractFailure) Domain() archetype.Domain { return archetype.DomainApplication }
func (APIContractFailure) Priority() int            { return 4 }
func (APIContractFailure) HypothesisID() string     { return "hypothesis-api-contract-failure" }
func (APIContractFailure) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (APIContractFailure) ExpectedEvidence() []model.Category {
	return []model.Category{model.CategoryApplicationLogs, model.CategoryTraceEvents, model.CategoryDeploymentEvents}
}
func (APIContractFailure) TypicalSubcauses() []string {
	return []string{"breaking change", "version mismatch", "serialization error", "schema drift"}
}
func (a APIContractFailure) Score(ctx archetype.ScoreContext) archetype.Candidate {
	return scoreKeyword(ctx, a.HypothesisID(),
		"An API contract or schema mismatch broke client-server compatibility.",
		"apicontract", 42)
}

func (APIContractFailure) SeedQuestions() []archetype.QuestionSeed {
	return []archetype.QuestionSeed{{
		ID: "api-breaking-change", Priority: 72,
		Title:       "Was there a breaking API or schema change?",
		Description: "Version mismatches and serialization errors surface as 4xx/5xx at boundaries.",
		Requires:      []model.Category{model.CategoryApplicationLogs, model.CategoryDeploymentEvents},
		TriggerSignal: "apicontract",
		Effects: []archetype.QuestionEffect{
			effect(true, archetype.EffectIncrease, "hypothesis-api-contract-failure", 30),
			effect(false, archetype.EffectDecrease, "hypothesis-api-contract-failure", 20),
		},
	}}
}

// FeatureFlagFailure scores feature-flag rollout mistakes.
type FeatureFlagFailure struct{}

func (FeatureFlagFailure) ID() string               { return "feature-flag-failure" }
func (FeatureFlagFailure) Name() string             { return "Feature Flag Failure" }
func (FeatureFlagFailure) Domain() archetype.Domain { return archetype.DomainApplication }
func (FeatureFlagFailure) Priority() int            { return 4 }
func (FeatureFlagFailure) HypothesisID() string     { return "hypothesis-feature-flag-failure" }
func (FeatureFlagFailure) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (FeatureFlagFailure) ExpectedEvidence() []model.Category {
	return []model.Category{model.CategoryConfigurationChanges, model.CategoryDeploymentEvents, model.CategoryApplicationLogs}
}
func (FeatureFlagFailure) TypicalSubcauses() []string {
	return []string{"flag enabled for wrong audience", "gradual rollout bug", "stale flag state"}
}
func (a FeatureFlagFailure) Score(ctx archetype.ScoreContext) archetype.Candidate {
	return scoreWith(ctx, scoreOpts{
		HypothesisID: a.HypothesisID(),
		Statement:    "A feature-flag change or rollout mistake introduced the regression.",
		Keyword:      "featureflag",
		Base:         44,
		Penalties: []scorePenalty{
			{When: "config", Unless: "featureflag", Amount: 15},
		},
	})
}

func (FeatureFlagFailure) SeedQuestions() []archetype.QuestionSeed {
	return []archetype.QuestionSeed{{
		ID: "feature-flag-changed", Priority: 71,
		Title:       "Was a feature flag enabled, disabled, or rolled out?",
		Description: "Flag changes can affect subsets of traffic without a full deployment.",
		Requires:      []model.Category{model.CategoryConfigurationChanges, model.CategoryDeploymentEvents},
		TriggerSignal: "featureflag",
		Effects: []archetype.QuestionEffect{
			effect(true, archetype.EffectIncrease, "hypothesis-feature-flag-failure", 30),
			effect(false, archetype.EffectDecrease, "hypothesis-feature-flag-failure", 18),
			effect(true, archetype.EffectDecrease, "hypothesis-configuration-change", 15),
		},
	}}
}
