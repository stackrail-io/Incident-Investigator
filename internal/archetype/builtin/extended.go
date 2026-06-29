package builtin

import (
	"github.com/stackrail/incident-investigator/internal/archetype"
	"github.com/stackrail/incident-investigator/internal/model"
	sigpkg "github.com/stackrail/incident-investigator/internal/signals"
)

// DependencyFailure scores downstream or third-party dependency unavailability.
type DependencyFailure struct{}

func (DependencyFailure) ID() string               { return "dependency-failure" }
func (DependencyFailure) Name() string             { return "Dependency Failure" }
func (DependencyFailure) Domain() archetype.Domain { return archetype.DomainOperations }
func (DependencyFailure) Priority() int          { return 5 }
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
	return c
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

// ExternalOutage scores third-party or SaaS provider outages.
type ExternalOutage struct{}

func (ExternalOutage) ID() string               { return "external-outage" }
func (ExternalOutage) Name() string             { return "External Service Outage" }
func (ExternalOutage) Domain() archetype.Domain { return archetype.DomainExternal }
func (ExternalOutage) Priority() int          { return 5 }
func (ExternalOutage) HypothesisID() string   { return "hypothesis-external-outage" }
func (ExternalOutage) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (ExternalOutage) ExpectedEvidence() []model.Category {
	return []model.Category{model.CategoryApplicationLogs, model.CategoryNetworkEvents, model.CategoryAlertEvents}
}
func (ExternalOutage) TypicalSubcauses() []string {
	return []string{"vendor outage", "regional provider failure", "api unavailable", "sla breach"}
}
func (a ExternalOutage) Score(ctx archetype.ScoreContext) archetype.Candidate {
	sig := ctx.Signals
	s := ctx.Session
	c := archetype.Candidate{
		HypothesisID: a.HypothesisID(),
		Statement:    "An external or third-party service outage caused the incident.",
	}
	if sig.Keywords["external"] {
		c.Score += 42
		c.Rationale = "Third-party or vendor outage symptoms appear in the evidence."
		c.Support = append(c.Support, sigpkg.EvidenceMatching(s, func(e *model.Evidence) bool {
			return sigpkg.MatchesAny(sigpkg.Haystack(e), sigpkg.Keywords["external"])
		})...)
	}
	return c
}

// AuthFailure scores authentication and authorization failures.
type AuthFailure struct{}

func (AuthFailure) ID() string               { return "auth-failure" }
func (AuthFailure) Name() string             { return "Authentication / Authorization Failure" }
func (AuthFailure) Domain() archetype.Domain { return archetype.DomainPlatform }
func (AuthFailure) Priority() int            { return 4 }
func (AuthFailure) HypothesisID() string     { return "hypothesis-auth-failure" }
func (AuthFailure) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (AuthFailure) ExpectedEvidence() []model.Category {
	return []model.Category{model.CategorySecurityEvents, model.CategoryApplicationLogs, model.CategoryAlertEvents}
}
func (AuthFailure) TypicalSubcauses() []string {
	return []string{"expired token", "permission change", "identity provider outage", "rbac misconfiguration"}
}
func (a AuthFailure) Score(ctx archetype.ScoreContext) archetype.Candidate {
	sig := ctx.Signals
	s := ctx.Session
	c := archetype.Candidate{
		HypothesisID: a.HypothesisID(),
		Statement:    "An authentication or authorization failure blocked legitimate access.",
	}
	if sig.Keywords["auth"] {
		c.Score += 44
		c.Rationale = "Authentication or authorization symptoms appear in the evidence."
		c.Support = append(c.Support, sigpkg.EvidenceMatching(s, func(e *model.Evidence) bool {
			return sigpkg.MatchesAny(sigpkg.Haystack(e), sigpkg.Keywords["auth"])
		})...)
	}
	return c
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

// SecurityIncident scores security breaches distinct from certificate expiry.
type SecurityIncident struct{}

func (SecurityIncident) ID() string               { return "security-incident" }
func (SecurityIncident) Name() string             { return "Security Incident" }
func (SecurityIncident) Domain() archetype.Domain { return archetype.DomainSecurity }
func (SecurityIncident) Priority() int            { return 5 }
func (SecurityIncident) HypothesisID() string     { return "hypothesis-security-incident" }
func (SecurityIncident) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (SecurityIncident) ExpectedEvidence() []model.Category {
	return []model.Category{model.CategorySecurityEvents, model.CategoryApplicationLogs, model.CategoryHumanContext}
}
func (SecurityIncident) TypicalSubcauses() []string {
	return []string{"credential compromise", "unauthorized access", "exploit", "data exfiltration"}
}
func (a SecurityIncident) Score(ctx archetype.ScoreContext) archetype.Candidate {
	sig := ctx.Signals
	s := ctx.Session
	c := archetype.Candidate{
		HypothesisID: a.HypothesisID(),
		Statement:    "A security incident such as unauthorized access or compromise caused the outage.",
	}
	if sig.Keywords["security"] {
		c.Score += 46
		c.Rationale = "Security incident symptoms appear in the evidence."
		c.Support = append(c.Support, sigpkg.EvidenceMatching(s, func(e *model.Evidence) bool {
			return sigpkg.MatchesAny(sigpkg.Haystack(e), sigpkg.Keywords["security"])
		})...)
	}
	if s.HasCategory(model.CategorySecurityEvents) && !sig.Keywords["cert"] {
		c.Score += 10
	}
	return c
}
