package builtin

import (
	"github.com/stackrail/incident-investigator/internal/archetype"
	"github.com/stackrail/incident-investigator/internal/model"
)

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

func (ExternalOutage) SeedQuestions() []archetype.QuestionSeed {
	return []archetype.QuestionSeed{
		{
			ID: "vendor-outage", Priority: 76,
			Title:       "Did a third-party or vendor outage occur?",
			Description: "External provider failures propagate as internal timeouts and errors.",
			Requires:      []model.Category{model.CategoryApplicationLogs, model.CategoryNetworkEvents},
			TriggerSignal: "external",
			Effects: []archetype.QuestionEffect{
				effect(true, archetype.EffectIncrease, "hypothesis-external-outage", 32),
				effect(false, archetype.EffectDecrease, "hypothesis-external-outage", 22),
			},
		},
	}
}

func (AuthFailure) SeedQuestions() []archetype.QuestionSeed {
	return []archetype.QuestionSeed{
		{
			ID: "auth-failure", Priority: 72,
			Title:       "Did authentication or authorization fail?",
			Description: "Token expiry, IAM changes, and identity-provider faults block access.",
			Requires:      []model.Category{model.CategorySecurityEvents, model.CategoryApplicationLogs},
			TriggerSignal: "auth",
			Effects: []archetype.QuestionEffect{
				effect(true, archetype.EffectIncrease, "hypothesis-auth-failure", 30),
				effect(false, archetype.EffectDecrease, "hypothesis-auth-failure", 20),
			},
		},
	}
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

func (SecurityIncident) SeedQuestions() []archetype.QuestionSeed {
	return []archetype.QuestionSeed{
		{
			ID: "security-breach", Priority: 84,
			Title:       "Was unauthorized access or a security breach detected?",
			Description: "Credential compromise and exploits can cause outages or data exposure.",
			Requires:      []model.Category{model.CategorySecurityEvents, model.CategoryApplicationLogs},
			TriggerSignal: "security",
			Effects: []archetype.QuestionEffect{
				effect(true, archetype.EffectIncrease, "hypothesis-security-incident", 35),
				effect(false, archetype.EffectDecrease, "hypothesis-security-incident", 25),
			},
		},
	}
}
