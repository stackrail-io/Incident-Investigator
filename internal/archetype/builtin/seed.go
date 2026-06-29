package builtin

import (
	"github.com/stackrail/incident-investigator/internal/archetype"
	"github.com/stackrail/incident-investigator/internal/model"
)

func effect(whenTrue bool, action archetype.EffectAction, hyp string, amount float64) archetype.QuestionEffect {
	return archetype.QuestionEffect{WhenTrue: whenTrue, Action: action, HypothesisID: hyp, Amount: amount}
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

func (DeploymentUnrelated) SeedQuestions() []archetype.QuestionSeed { return nil }

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

func (NetworkDNS) SeedQuestions() []archetype.QuestionSeed {
	return []archetype.QuestionSeed{
		{
			ID: "dns-failure", Priority: 75,
			Title:       "Did DNS resolution fail?",
			Description: "NXDOMAIN and name-resolution errors sever connectivity to dependencies.",
			Requires:      []model.Category{model.CategoryNetworkEvents, model.CategoryApplicationLogs},
			TriggerSignal: "dns",
			Effects: []archetype.QuestionEffect{
				effect(true, archetype.EffectIncrease, "hypothesis-network-dns", 30),
				effect(false, archetype.EffectDecrease, "hypothesis-network-dns", 20),
			},
		},
		{
			ID: "network-healthy", Priority: 60,
			Title:       "Was network connectivity healthy?",
			Description: "Absence of network or DNS symptoms argues against connectivity failures.",
			Requires: []model.Category{model.CategoryNetworkEvents, model.CategoryApplicationLogs},
			Effects: []archetype.QuestionEffect{
				effect(true, archetype.EffectDecrease, "hypothesis-network-dns", 25),
				effect(false, archetype.EffectIncrease, "hypothesis-network-dns", 30),
			},
		},
	}
}

func (CertificateExpiry) SeedQuestions() []archetype.QuestionSeed {
	return []archetype.QuestionSeed{
		{
			ID: "certificate-expired", Priority: 82,
			Title:       "Did a TLS certificate expire or become invalid?",
			Description: "Expired or mis-issued certificates break TLS handshakes across all clients.",
			Requires:      []model.Category{model.CategorySecurityEvents, model.CategoryApplicationLogs},
			TriggerSignal: "cert",
			Effects: []archetype.QuestionEffect{
				effect(true, archetype.EffectIncrease, "hypothesis-certificate-expiry", 35),
				effect(false, archetype.EffectDecrease, "hypothesis-certificate-expiry", 25),
			},
		},
	}
}

func (Unknown) SeedQuestions() []archetype.QuestionSeed { return nil }
