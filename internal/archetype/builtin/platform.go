package builtin

import (
	"github.com/stackrail/incident-investigator/internal/archetype"
	"github.com/stackrail/incident-investigator/internal/model"
	sigpkg "github.com/stackrail/incident-investigator/internal/signals"
)

// NetworkFailure scores routing and connectivity failures (excluding DNS).
type NetworkFailure struct{}

func (NetworkFailure) ID() string               { return "network-failure" }
func (NetworkFailure) Name() string             { return "Network Failure" }
func (NetworkFailure) Domain() archetype.Domain { return archetype.DomainPlatform }
func (NetworkFailure) Priority() int            { return 5 }
func (NetworkFailure) HypothesisID() string     { return "hypothesis-network-failure" }
func (NetworkFailure) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (NetworkFailure) ExpectedEvidence() []model.Category {
	return []model.Category{model.CategoryNetworkEvents, model.CategoryApplicationLogs}
}
func (NetworkFailure) TypicalSubcauses() []string {
	return []string{"routing", "firewall", "packet loss", "connection refused", "cross-region latency"}
}
func (a NetworkFailure) Score(ctx archetype.ScoreContext) archetype.Candidate {
	sig := ctx.Signals
	s := ctx.Session
	c := archetype.Candidate{
		HypothesisID: a.HypothesisID(),
		Statement:    "A network connectivity failure disrupted service communication.",
	}
	if sig.Keywords["dns"] {
		return c
	}
	if sig.Keywords["network"] {
		c.Score += 38
		c.Rationale = "Network connectivity symptoms appear in the evidence."
		c.Support = append(c.Support, sigpkg.EvidenceMatching(s, func(e *model.Evidence) bool {
			return sigpkg.MatchesAny(sigpkg.Haystack(e), sigpkg.Keywords["network"])
		})...)
	}
	if sig.Keywords["loadbalancer"] {
		c.Score -= 25
		if c.Score < 0 {
			c.Score = 0
		}
	}
	return c
}

func (NetworkFailure) SeedQuestions() []archetype.QuestionSeed {
	return []archetype.QuestionSeed{
		{
			ID: "network-healthy", Priority: 60,
			Title:       "Was network connectivity healthy?",
			Description: "Absence of network symptoms argues against routing and connectivity failures.",
			Requires: []model.Category{model.CategoryNetworkEvents, model.CategoryApplicationLogs},
			Effects: []archetype.QuestionEffect{
				effect(true, archetype.EffectDecrease, "hypothesis-network-failure", 25),
				effect(false, archetype.EffectIncrease, "hypothesis-network-failure", 30),
			},
		},
	}
}

// DNSFailure scores DNS resolution failures.
type DNSFailure struct{}

func (DNSFailure) ID() string               { return "dns-failure" }
func (DNSFailure) Name() string             { return "DNS Failure" }
func (DNSFailure) Domain() archetype.Domain { return archetype.DomainPlatform }
func (DNSFailure) Priority() int            { return 5 }
func (DNSFailure) HypothesisID() string     { return "hypothesis-dns-failure" }
func (DNSFailure) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (DNSFailure) ExpectedEvidence() []model.Category {
	return []model.Category{model.CategoryNetworkEvents, model.CategoryApplicationLogs}
}
func (DNSFailure) TypicalSubcauses() []string {
	return []string{"resolution failure", "wrong record", "ttl issue", "resolver cache"}
}
func (a DNSFailure) Score(ctx archetype.ScoreContext) archetype.Candidate {
	sig := ctx.Signals
	s := ctx.Session
	c := archetype.Candidate{
		HypothesisID: a.HypothesisID(),
		Statement:    "A DNS resolution failure severed connectivity to dependencies.",
	}
	if sig.Keywords["dns"] {
		c.Score += 42
		c.Rationale = "DNS resolution symptoms appear in the evidence."
		c.Support = append(c.Support, sigpkg.EvidenceMatching(s, func(e *model.Evidence) bool {
			return sigpkg.MatchesAny(sigpkg.Haystack(e), sigpkg.Keywords["dns"])
		})...)
	}
	return c
}

func (DNSFailure) SeedQuestions() []archetype.QuestionSeed {
	return []archetype.QuestionSeed{
		{
			ID: "dns-failure", Priority: 75,
			Title:       "Did DNS resolution fail?",
			Description: "NXDOMAIN and name-resolution errors sever connectivity to dependencies.",
			Requires:      []model.Category{model.CategoryNetworkEvents, model.CategoryApplicationLogs},
			TriggerSignal: "dns",
			Effects: []archetype.QuestionEffect{
				effect(true, archetype.EffectIncrease, "hypothesis-dns-failure", 30),
				effect(false, archetype.EffectDecrease, "hypothesis-dns-failure", 20),
			},
		},
	}
}

// CertificateExpiry scores TLS and certificate failures.
type CertificateExpiry struct{}

func (CertificateExpiry) ID() string               { return "certificate-tls-failure" }
func (CertificateExpiry) Name() string             { return "Certificate / TLS Failure" }
func (CertificateExpiry) Domain() archetype.Domain { return archetype.DomainPlatform }
func (CertificateExpiry) Priority() int            { return 5 }
func (CertificateExpiry) HypothesisID() string     { return "hypothesis-certificate-expiry" }
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

// KubernetesFailure scores scheduler, readiness, and pod lifecycle faults.
type KubernetesFailure struct{}

func (KubernetesFailure) ID() string               { return "kubernetes-failure" }
func (KubernetesFailure) Name() string             { return "Kubernetes Failure" }
func (KubernetesFailure) Domain() archetype.Domain { return archetype.DomainPlatform }
func (KubernetesFailure) Priority() int            { return 5 }
func (KubernetesFailure) HypothesisID() string     { return "hypothesis-kubernetes-failure" }
func (KubernetesFailure) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (KubernetesFailure) ExpectedEvidence() []model.Category {
	return []model.Category{model.CategoryInfrastructureEvents, model.CategoryApplicationLogs, model.CategoryAlertEvents}
}
func (KubernetesFailure) TypicalSubcauses() []string {
	return []string{"crashloop", "eviction", "scheduling failure", "readiness probe", "image pull"}
}
func (a KubernetesFailure) Score(ctx archetype.ScoreContext) archetype.Candidate {
	return scoreWith(ctx, scoreOpts{
		HypothesisID: a.HypothesisID(),
		Statement:    "A Kubernetes control-plane or workload lifecycle failure caused the incident.",
		Keyword:      "kubernetes",
		Base:         40,
		Boosts: []scoreBoost{
			{When: "restart", Amount: 12},
		},
		Penalties: []scorePenalty{
			{When: "config", Unless: "kubernetes", Amount: 15},
			{When: "container", Unless: "kubernetes", Amount: 8},
		},
	})
}

func (KubernetesFailure) SeedQuestions() []archetype.QuestionSeed {
	return []archetype.QuestionSeed{{
		ID: "k8s-crashloop", Priority: 72,
		Title:       "Did pods crash loop, fail readiness, or fail scheduling?",
		Description: "Kubernetes lifecycle faults surface as restarts, evictions, or pending pods.",
		Requires:      []model.Category{model.CategoryInfrastructureEvents, model.CategoryApplicationLogs},
		TriggerSignal: "kubernetes",
		Effects: []archetype.QuestionEffect{
			effect(true, archetype.EffectIncrease, "hypothesis-kubernetes-failure", 28),
			effect(false, archetype.EffectDecrease, "hypothesis-kubernetes-failure", 18),
		},
	}}
}

// ContainerFailure scores container runtime and image/registry faults.
type ContainerFailure struct{}

func (ContainerFailure) ID() string               { return "container-failure" }
func (ContainerFailure) Name() string             { return "Container Failure" }
func (ContainerFailure) Domain() archetype.Domain { return archetype.DomainPlatform }
func (ContainerFailure) Priority() int            { return 4 }
func (ContainerFailure) HypothesisID() string     { return "hypothesis-container-failure" }
func (ContainerFailure) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (ContainerFailure) ExpectedEvidence() []model.Category {
	return []model.Category{model.CategoryInfrastructureEvents, model.CategoryApplicationLogs}
}
func (ContainerFailure) TypicalSubcauses() []string {
	return []string{"image pull failed", "wrong image", "oci runtime error", "registry unavailable"}
}
func (a ContainerFailure) Score(ctx archetype.ScoreContext) archetype.Candidate {
	return scoreWith(ctx, scoreOpts{
		HypothesisID: a.HypothesisID(),
		Statement:    "A container runtime or image failure prevented workloads from starting.",
		Keyword:      "container",
		Base:         40,
		Boosts: []scoreBoost{
			{When: "restart", Amount: 8},
		},
		Penalties: []scorePenalty{
			{When: "kubernetes", Unless: "container", Amount: 12},
		},
	})
}

func (ContainerFailure) SeedQuestions() []archetype.QuestionSeed {
	return []archetype.QuestionSeed{{
		ID: "image-pull-failed", Priority: 68,
		Title:       "Did a container image pull or runtime error occur?",
		Description: "Registry outages and OCI runtime faults prevent pods from starting.",
		Requires:      []model.Category{model.CategoryInfrastructureEvents, model.CategoryApplicationLogs},
		TriggerSignal: "container",
		Effects: []archetype.QuestionEffect{
			effect(true, archetype.EffectIncrease, "hypothesis-container-failure", 28),
			effect(false, archetype.EffectDecrease, "hypothesis-container-failure", 18),
		},
	}}
}

// LoadBalancerFailure scores proxy and load-balancer faults.
type LoadBalancerFailure struct{}

func (LoadBalancerFailure) ID() string               { return "load-balancer-failure" }
func (LoadBalancerFailure) Name() string             { return "Load Balancer / Proxy Failure" }
func (LoadBalancerFailure) Domain() archetype.Domain { return archetype.DomainPlatform }
func (LoadBalancerFailure) Priority() int            { return 4 }
func (LoadBalancerFailure) HypothesisID() string     { return "hypothesis-load-balancer-failure" }
func (LoadBalancerFailure) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (LoadBalancerFailure) ExpectedEvidence() []model.Category {
	return []model.Category{model.CategoryNetworkEvents, model.CategoryApplicationLogs, model.CategoryMetrics}
}
func (LoadBalancerFailure) TypicalSubcauses() []string {
	return []string{"health check failed", "routing misconfiguration", "tls termination", "backend pool empty"}
}
func (a LoadBalancerFailure) Score(ctx archetype.ScoreContext) archetype.Candidate {
	return scoreWith(ctx, scoreOpts{
		HypothesisID: a.HypothesisID(),
		Statement:    "A load balancer or proxy failure disrupted traffic routing.",
		Keyword:      "loadbalancer",
		Base:         40,
		SkipIf:       []string{"dns"},
		Penalties: []scorePenalty{
			{When: "network", Unless: "loadbalancer", Amount: 10},
		},
	})
}

func (LoadBalancerFailure) SeedQuestions() []archetype.QuestionSeed {
	return []archetype.QuestionSeed{{
		ID: "lb-health-check", Priority: 70,
		Title:       "Did load-balancer health checks or routing fail?",
		Description: "Proxy and ALB faults drop healthy backends or misroute traffic.",
		Requires:      []model.Category{model.CategoryNetworkEvents, model.CategoryApplicationLogs},
		TriggerSignal: "loadbalancer",
		Effects: []archetype.QuestionEffect{
			effect(true, archetype.EffectIncrease, "hypothesis-load-balancer-failure", 28),
			effect(false, archetype.EffectDecrease, "hypothesis-load-balancer-failure", 18),
		},
	}}
}

// ClockFailure scores NTP drift and time-sync issues.
type ClockFailure struct{}

func (ClockFailure) ID() string               { return "clock-failure" }
func (ClockFailure) Name() string             { return "Clock / Time Failure" }
func (ClockFailure) Domain() archetype.Domain { return archetype.DomainPlatform }
func (ClockFailure) Priority() int            { return 3 }
func (ClockFailure) HypothesisID() string     { return "hypothesis-clock-failure" }
func (ClockFailure) Applicable(ctx archetype.ScoreContext) bool {
	return alwaysApplicable(ctx)
}
func (ClockFailure) ExpectedEvidence() []model.Category {
	return []model.Category{model.CategoryInfrastructureEvents, model.CategorySecurityEvents, model.CategoryApplicationLogs}
}
func (ClockFailure) TypicalSubcauses() []string {
	return []string{"clock drift", "ntp failure", "token validation skew", "leader election skew"}
}
func (a ClockFailure) Score(ctx archetype.ScoreContext) archetype.Candidate {
	return scoreKeyword(ctx, a.HypothesisID(),
		"Clock skew or time synchronization failure caused validation or ordering faults.",
		"clock", 40)
}

func (ClockFailure) SeedQuestions() []archetype.QuestionSeed {
	return []archetype.QuestionSeed{{
		ID: "clock-drift", Priority: 62,
		Title:       "Was clock drift or NTP sync failure observed?",
		Description: "Time skew breaks token validation, TLS, and distributed coordination.",
		Requires:      []model.Category{model.CategoryInfrastructureEvents, model.CategorySecurityEvents},
		TriggerSignal: "clock",
		Effects: []archetype.QuestionEffect{
			effect(true, archetype.EffectIncrease, "hypothesis-clock-failure", 28),
			effect(false, archetype.EffectDecrease, "hypothesis-clock-failure", 18),
		},
	}}
}
