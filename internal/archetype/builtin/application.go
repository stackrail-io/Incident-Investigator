package builtin

import (
	"github.com/stackrail/incident-investigator/internal/archetype"
	"github.com/stackrail/incident-investigator/internal/model"
	sigpkg "github.com/stackrail/incident-investigator/internal/signals"
)

// DeploymentCaused scores deployment-before-symptom signatures.
type DeploymentCaused struct{}

func (DeploymentCaused) ID() string           { return "deployment-failure" }
func (DeploymentCaused) Name() string         { return "Deployment Failure" }
func (DeploymentCaused) Domain() archetype.Domain { return archetype.DomainApplication }
func (DeploymentCaused) Priority() int        { return 5 }
func (DeploymentCaused) HypothesisID() string { return "hypothesis-deployment-caused" }
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

// DeploymentUnrelated competes against deployment-blame bias.
type DeploymentUnrelated struct{}

func (DeploymentUnrelated) ID() string           { return "deployment-unrelated" }
func (DeploymentUnrelated) Name() string         { return "Deployment Unrelated" }
func (DeploymentUnrelated) Domain() archetype.Domain { return archetype.DomainApplication }
func (DeploymentUnrelated) Priority() int      { return 3 }
func (DeploymentUnrelated) HypothesisID() string { return "hypothesis-deployment-unrelated" }
func (DeploymentUnrelated) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (DeploymentUnrelated) ExpectedEvidence() []model.Category {
	return []model.Category{model.CategoryDeploymentEvents, model.CategoryAlertEvents}
}
func (DeploymentUnrelated) TypicalSubcauses() []string { return nil }
func (a DeploymentUnrelated) Score(ctx archetype.ScoreContext) archetype.Candidate {
	sig := ctx.Signals
	c := archetype.Candidate{
		HypothesisID: a.HypothesisID(),
		Statement:    "The deployment was unrelated; the incident has another root cause.",
		Score:        8,
		Rationale:    "Maintained as a competing baseline against deployment-blame bias.",
	}
	if ctx.DeployContradicted() {
		c.Score += 40
		c.Rationale = "The deployment timestamp falls after the incident began."
	}
	if sig.FirstDeployment == nil {
		c.Score = 0
	}
	return c
}

// ConfigurationChange scores config and feature-flag changes.
type ConfigurationChange struct{}

func (ConfigurationChange) ID() string           { return "configuration-drift" }
func (ConfigurationChange) Name() string         { return "Configuration Drift" }
func (ConfigurationChange) Domain() archetype.Domain { return archetype.DomainApplication }
func (ConfigurationChange) Priority() int        { return 5 }
func (ConfigurationChange) HypothesisID() string { return "hypothesis-configuration-change" }
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
	if sig.Keywords["config"] {
		c.Score += 30
		c.Rationale = "Configuration-change symptoms appear in the evidence."
		c.Support = append(c.Support, sigpkg.EvidenceMatching(s, func(e *model.Evidence) bool {
			return e.Category == model.CategoryConfigurationChanges ||
				sigpkg.MatchesAny(sigpkg.Haystack(e), sigpkg.Keywords["config"])
		})...)
	}
	return c
}

// NetworkDNS scores network and DNS connectivity failures.
type NetworkDNS struct{}

func (NetworkDNS) ID() string           { return "network-failure" }
func (NetworkDNS) Name() string         { return "Network / DNS Failure" }
func (NetworkDNS) Domain() archetype.Domain { return archetype.DomainPlatform }
func (NetworkDNS) Priority() int        { return 5 }
func (NetworkDNS) HypothesisID() string { return "hypothesis-network-dns" }
func (NetworkDNS) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (NetworkDNS) ExpectedEvidence() []model.Category {
	return []model.Category{model.CategoryNetworkEvents, model.CategoryApplicationLogs}
}
func (NetworkDNS) TypicalSubcauses() []string {
	return []string{"dns resolution", "routing", "firewall", "packet loss", "load balancer"}
}
func (a NetworkDNS) Score(ctx archetype.ScoreContext) archetype.Candidate {
	sig := ctx.Signals
	s := ctx.Session
	c := archetype.Candidate{
		HypothesisID: a.HypothesisID(),
		Statement:    "A network or DNS failure disrupted connectivity.",
	}
	if sig.Keywords["dns"] {
		c.Score += 38
		c.Rationale = "DNS resolution symptoms appear in the evidence."
	}
	if sig.Keywords["network"] {
		c.Score += 18
		if c.Rationale == "" {
			c.Rationale = "Network connectivity symptoms appear in the evidence."
		}
	}
	if c.Score > 0 {
		c.Support = append(c.Support, sigpkg.EvidenceMatching(s, func(e *model.Evidence) bool {
			t := sigpkg.Haystack(e)
			return sigpkg.MatchesAny(t, sigpkg.Keywords["dns"]) || sigpkg.MatchesAny(t, sigpkg.Keywords["network"])
		})...)
	}
	return c
}

// CertificateExpiry scores TLS and certificate failures.
type CertificateExpiry struct{}

func (CertificateExpiry) ID() string           { return "certificate-tls-failure" }
func (CertificateExpiry) Name() string         { return "Certificate / TLS Failure" }
func (CertificateExpiry) Domain() archetype.Domain { return archetype.DomainPlatform }
func (CertificateExpiry) Priority() int        { return 5 }
func (CertificateExpiry) HypothesisID() string { return "hypothesis-certificate-expiry" }
func (CertificateExpiry) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (CertificateExpiry) ExpectedEvidence() []model.Category {
	return []model.Category{model.CategorySecurityEvents, model.CategoryApplicationLogs, model.CategoryNetworkEvents}
}
func (CertificateExpiry) TypicalSubcauses() []string {
	return []string{"expired certificate", "wrong certificate", "ca change", "trust issue"}
}
func (a CertificateExpiry) Score(ctx archetype.ScoreContext) archetype.Candidate {
	sig := ctx.Signals
	s := ctx.Session
	c := archetype.Candidate{
		HypothesisID: a.HypothesisID(),
		Statement:    "An expired or invalid TLS certificate broke secure connections.",
	}
	if sig.Keywords["cert"] {
		c.Score += 48
		c.Rationale = "TLS/certificate symptoms appear in the evidence."
		c.Support = append(c.Support, sigpkg.EvidenceMatching(s, func(e *model.Evidence) bool {
			return sigpkg.MatchesAny(sigpkg.Haystack(e), sigpkg.Keywords["cert"])
		})...)
	}
	return c
}

// ResourceExhaustion scores memory, CPU, and restart signatures.
type ResourceExhaustion struct{}

func (ResourceExhaustion) ID() string           { return "resource-exhaustion" }
func (ResourceExhaustion) Name() string         { return "Resource Exhaustion" }
func (ResourceExhaustion) Domain() archetype.Domain { return archetype.DomainInfrastructure }
func (ResourceExhaustion) Priority() int        { return 5 }
func (ResourceExhaustion) HypothesisID() string { return "hypothesis-resource-exhaustion" }
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

// RetryStorm scores retry amplification and cascading failures.
type RetryStorm struct{}

func (RetryStorm) ID() string           { return "retry-storm" }
func (RetryStorm) Name() string         { return "Retry Storm / Cascading Failure" }
func (RetryStorm) Domain() archetype.Domain { return archetype.DomainOperations }
func (RetryStorm) Priority() int        { return 5 }
func (RetryStorm) HypothesisID() string { return "hypothesis-retry-storm" }
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

// Unknown is the catch-all that fades as evidence coverage grows.
type Unknown struct{}

func (Unknown) ID() string           { return "unknown-novel" }
func (Unknown) Name() string         { return "Unknown / Novel Failure" }
func (Unknown) Domain() archetype.Domain { return archetype.DomainGeneric }
func (Unknown) Priority() int        { return 5 }
func (Unknown) HypothesisID() string { return "hypothesis-unknown" }
func (Unknown) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (Unknown) ExpectedEvidence() []model.Category { return nil }
func (Unknown) TypicalSubcauses() []string           { return nil }
func (a Unknown) Score(ctx archetype.ScoreContext) archetype.Candidate {
	score := 40.0 - 5*float64(len(ctx.Signals.Categories))
	if score < 4 {
		score = 4
	}
	return archetype.Candidate{
		HypothesisID: a.HypothesisID(),
		Statement:    "Root cause is not yet determined; more evidence is required.",
		Score:        score,
		Rationale:    "Reflects residual uncertainty given current evidence coverage.",
		AlwaysKeep:   true,
	}
}
