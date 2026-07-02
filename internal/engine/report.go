package engine

import (
	"fmt"
	"strings"
	"time"

	"github.com/stackrail/incident-investigator/internal/model"
)

// ReportGenerator assembles the final investigation deliverable.
type ReportGenerator interface {
	Generate(s *model.Session, sig Signals) model.Report
}

// HeuristicReportGenerator builds an executive summary, recommendations and a
// markdown postmortem from the session state.
type HeuristicReportGenerator struct{}

// NewHeuristicReportGenerator returns the default generator.
func NewHeuristicReportGenerator() *HeuristicReportGenerator { return &HeuristicReportGenerator{} }

// recommendationsByHypothesis maps a hypothesis id to concrete follow-ups.
var recommendationsByHypothesis = map[string][]string{
	"hypothesis-deployment-caused": {
		"Roll back the suspect deployment and confirm recovery.",
		"Add canary/progressive delivery and automated rollback gates.",
	},
	"hypothesis-database-saturation": {
		"Investigate database capacity, slow queries and connection-pool limits.",
		"Add circuit breakers and backpressure to protect the database.",
	},
	"hypothesis-lock-contention": {
		"Identify the long-running transaction holding the lock and shorten or split it.",
		"Set statement_timeout and lock_timeout on connection pools to fail fast instead of queuing unbounded.",
	},
	"hypothesis-configuration-change": {
		"Audit the recent configuration/feature-flag change and revert if needed.",
		"Add configuration validation and staged rollout in CI/CD.",
	},
	"hypothesis-network-failure": {
		"Verify routing, firewall rules, and network paths; add synthetic connectivity checks.",
		"Implement resilient retries with jittered backoff for transient failures.",
	},
	"hypothesis-dns-failure": {
		"Verify DNS resolvers, TTL settings, and record correctness.",
		"Add DNS failover and caching resilience for critical dependencies.",
	},
	"hypothesis-certificate-expiry": {
		"Renew the affected certificate and automate renewal (e.g. ACME).",
		"Add certificate-expiry alerting well ahead of expiry.",
	},
	"hypothesis-resource-exhaustion": {
		"Right-size memory/CPU limits and investigate the suspected leak.",
		"Enable autoscaling and add saturation alerts.",
	},
	"hypothesis-retry-storm": {
		"Introduce retry budgets, jittered backoff and circuit breakers.",
		"Add load-shedding to prevent cascading amplification.",
	},
	"hypothesis-dependency-failure": {
		"Verify downstream dependency health and add circuit breakers with fallbacks.",
		"Add synthetic checks and SLOs for critical dependencies.",
	},
	"hypothesis-performance-regression": {
		"Profile the degraded endpoint and compare against the last known-good release.",
		"Add performance regression tests in CI for hot paths.",
	},
	"hypothesis-external-outage": {
		"Confirm vendor status and engage provider support; enable graceful degradation.",
		"Add multi-vendor or multi-region failover where feasible.",
	},
	"hypothesis-auth-failure": {
		"Verify identity-provider health, token expiry, and recent IAM/RBAC changes.",
		"Add auth-failure alerting and automated token rotation.",
	},
	"hypothesis-human-error": {
		"Document the manual change and add guardrails (approval gates, dry-run).",
		"Update runbooks and add blast-radius checks for operational commands.",
	},
	"hypothesis-capacity-planning": {
		"Review autoscaling policies, quotas, and traffic forecasts.",
		"Load-test ahead of expected spikes and pre-warm capacity.",
	},
	"hypothesis-security-incident": {
		"Contain compromised credentials and rotate secrets immediately.",
		"Engage security team for forensics and audit access logs.",
	},
	"hypothesis-infrastructure-failure": {
		"Verify cloud health dashboards and evacuate affected nodes or zones.",
		"Add redundancy across availability zones and automate node replacement.",
	},
	"hypothesis-kubernetes-failure": {
		"Inspect pod events, readiness probes, and scheduler constraints.",
		"Add pod disruption budgets and resource requests/limits.",
	},
	"hypothesis-container-failure": {
		"Verify image tags, registry availability, and runtime logs.",
		"Pin images by digest and add pull-policy safeguards.",
	},
	"hypothesis-storage-failure": {
		"Check volume attach status, disk usage, and IO latency metrics.",
		"Add storage capacity alerts and automated volume expansion.",
	},
	"hypothesis-cache-failure": {
		"Verify cache cluster health and hit-ratio metrics.",
		"Add cache warming and stampede protection on hot keys.",
	},
	"hypothesis-messaging-failure": {
		"Inspect consumer lag, dead-letter queues, and broker health.",
		"Add backpressure and poison-message handling.",
	},
	"hypothesis-load-balancer-failure": {
		"Verify health-check configuration and backend pool membership.",
		"Add multi-layer health checks and graceful connection draining.",
	},
	"hypothesis-api-contract-failure": {
		"Compare API schemas across client and server versions.",
		"Add contract tests and schema validation in CI/CD.",
	},
	"hypothesis-data-corruption": {
		"Run integrity checks and audit recent migrations or writes.",
		"Add checksums and point-in-time recovery procedures.",
	},
	"hypothesis-clock-failure": {
		"Verify NTP sync and clock skew across nodes.",
		"Add clock-drift monitoring and chrony/chronyd health checks.",
	},
	"hypothesis-feature-flag-failure": {
		"Audit recent flag changes and rollout audience targeting.",
		"Add flag change approval gates and canary rollouts.",
	},
	"hypothesis-regional-failure": {
		"Confirm regional cloud health and traffic distribution.",
		"Enable multi-region failover and cross-region load balancing.",
	},
	"hypothesis-dr-failover-failure": {
		"Review failover logs and replication state.",
		"Run regular DR drills and automate failover validation.",
	},
	"hypothesis-observability-failure": {
		"Restore telemetry agents and fill observability gaps.",
		"Add agent health monitoring and minimum coverage SLOs.",
	},
}

// Generate implements ReportGenerator.
func (g *HeuristicReportGenerator) Generate(s *model.Session, sig Signals) model.Report {
	report := model.Report{
		SessionID:       s.ID,
		Question:        s.Question,
		Timeline:        s.Timeline,
		Evidence:        s.Evidence,
		Hypotheses:      s.Hypotheses,
		Graph:           s.Graph,
		BlastRadius:     s.BlastRadius,
		Contradictions:  s.Contradictions,
		MissingEvidence: s.MissingEvidence,
		Confidence:      s.Confidence,
	}

	report.RootCauseCandidates = topCandidates(s.Hypotheses, 3)
	report.ExecutiveSummary = g.executiveSummary(s)
	report.Recommendations = g.recommendations(s)
	report.Postmortem = g.postmortem(s, report)

	return report
}

func topCandidates(hs []model.Hypothesis, n int) []model.Hypothesis {
	out := make([]model.Hypothesis, 0, n)
	for _, h := range hs {
		if h.Status == model.StatusRefuted {
			continue
		}
		out = append(out, h)
		if len(out) >= n {
			break
		}
	}
	return out
}

func (g *HeuristicReportGenerator) executiveSummary(s *model.Session) string {
	if len(s.Hypotheses) == 0 {
		return "No evidence has been submitted yet; no conclusion can be drawn."
	}
	lead := s.Hypotheses[0]
	var b strings.Builder
	fmt.Fprintf(&b, "Investigating: %q.", s.Question)
	fmt.Fprintf(&b, " Leading explanation (%.0f%% relative likelihood): %s",
		lead.Confidence, lead.Statement)
	fmt.Fprintf(&b, " Overall investigation confidence is %.0f%% based on %d evidence item(s) across %d categor(y/ies).",
		s.Confidence, len(s.Evidence), countCategories(s))
	if len(s.Contradictions) > 0 {
		fmt.Fprintf(&b, " Note: %d contradiction(s) were detected and should be resolved.", len(s.Contradictions))
	}
	if len(s.MissingEvidence) > 0 {
		fmt.Fprintf(&b, " %d additional evidence categor(y/ies) would strengthen the conclusion.", len(s.MissingEvidence))
	}
	return b.String()
}

func (g *HeuristicReportGenerator) recommendations(s *model.Session) []string {
	var recs []string
	if len(s.Hypotheses) > 0 {
		lead := s.Hypotheses[0]
		recs = append(recs, recommendationsByHypothesis[lead.ID]...)
	}
	if len(s.MissingEvidence) > 0 {
		cats := make([]string, 0, len(s.MissingEvidence))
		for _, m := range s.MissingEvidence {
			cats = append(cats, string(m.Category))
		}
		recs = append(recs, "Collect missing evidence to raise confidence: "+strings.Join(cats, ", ")+".")
	}
	if len(s.Contradictions) > 0 {
		recs = append(recs, "Resolve the detected contradictions before finalizing the root cause.")
	}
	if len(recs) == 0 {
		recs = append(recs, "Continue monitoring; no specific remediation is indicated by current evidence.")
	}
	return recs
}

func (g *HeuristicReportGenerator) postmortem(s *model.Session, r model.Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Root Cause Analysis: %s\n\n", s.Question)
	meta := fmt.Sprintf("**Confidence:** %.0f%%", s.Confidence)
	if s.Service != "" {
		meta = fmt.Sprintf("**Service:** %s | %s", s.Service, meta)
	}
	if !s.TimeWindow.Start.IsZero() && !s.TimeWindow.End.IsZero() {
		meta += fmt.Sprintf(" | **Window:** %s → %s",
			s.TimeWindow.Start.UTC().Format(time.RFC3339),
			s.TimeWindow.End.UTC().Format(time.RFC3339))
	}
	fmt.Fprintf(&b, "%s\n\n", meta)
	fmt.Fprintf(&b, "_Generated by Incident Investigator._\n\n")

	b.WriteString("## Executive Summary\n\n")
	b.WriteString(r.ExecutiveSummary + "\n\n")

	b.WriteString("## Chronological Timeline\n\n")
	b.WriteString("Evidence and events in the order they occurred:\n\n")
	if len(s.Timeline) == 0 {
		b.WriteString("_No timeline could be reconstructed._\n\n")
	} else {
		b.WriteString("| Time (UTC) | Evidence | Category | Entity | Event |\n")
		b.WriteString("| --- | --- | --- | --- | --- |\n")
		for _, e := range s.Timeline {
			evID := strings.Join(e.EvidenceRefs, ", ")
			if evID == "" {
				evID = "—"
			} else {
				evID = "`" + evID + "`"
			}
			entity := e.Entity
			if entity == "" {
				entity = "—"
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
				e.Timestamp.UTC().Format(time.RFC3339),
				evID, e.Category, entity, e.Description)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Root Cause Analysis\n\n")
	if len(r.RootCauseCandidates) > 0 {
		lead := r.RootCauseCandidates[0]
		fmt.Fprintf(&b, "**Primary hypothesis:** %s (%.0f%% relative likelihood, %s)\n\n",
			lead.Statement, lead.Confidence, lead.Status)
		b.WriteString(g.rootCauseNarrative(s, lead) + "\n\n")
	} else {
		b.WriteString("_Insufficient evidence to determine a primary root cause._\n\n")
	}

	if len(r.RootCauseCandidates) > 1 {
		b.WriteString("## Alternative Hypotheses\n\n")
		for _, h := range r.RootCauseCandidates[1:] {
			fmt.Fprintf(&b, "- **%s** — %.0f%% (%s)\n", h.Statement, h.Confidence, h.Status)
		}
		b.WriteString("\n")
	}

	if len(s.Contradictions) > 0 {
		b.WriteString("## Contradictions\n\n")
		for _, c := range s.Contradictions {
			fmt.Fprintf(&b, "- [%s] %s\n", c.Severity, c.Description)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Blast Radius\n\n")
	writeScoped(&b, "Services", s.BlastRadius.Services)
	writeScoped(&b, "Regions", s.BlastRadius.Regions)
	writeScoped(&b, "Customers", s.BlastRadius.Customers)
	writeScoped(&b, "APIs", s.BlastRadius.APIs)
	if len(s.BlastRadius.Services) == 0 && len(s.BlastRadius.Regions) == 0 &&
		len(s.BlastRadius.Customers) == 0 && len(s.BlastRadius.APIs) == 0 {
		b.WriteString("_No blast radius could be inferred from current evidence._\n\n")
	}

	if len(s.MissingEvidence) > 0 {
		b.WriteString("## Missing Evidence\n\n")
		for _, m := range s.MissingEvidence {
			fmt.Fprintf(&b, "- %s (%s): %s\n", m.Category, m.Priority, m.Reason)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Recommendations\n\n")
	for _, rec := range r.Recommendations {
		fmt.Fprintf(&b, "- %s\n", rec)
	}

	return b.String()
}

func (g *HeuristicReportGenerator) rootCauseNarrative(s *model.Session, lead model.Hypothesis) string {
	if len(s.Timeline) == 0 {
		return "Submit additional evidence to reconstruct the incident sequence."
	}
	first := s.Timeline[0]
	last := s.Timeline[len(s.Timeline)-1]
	var b strings.Builder
	fmt.Fprintf(&b, "The incident timeline spans %s to %s with %d observed event(s). ",
		first.Timestamp.UTC().Format("15:04 UTC"), last.Timestamp.UTC().Format("15:04 UTC"), len(s.Timeline))
	switch lead.ID {
	case "hypothesis-deployment-caused":
		b.WriteString("A deployment precedes symptom onset and recovery actions align with rollback — consistent with a release-induced regression.")
	case "hypothesis-lock-contention":
		b.WriteString("Healthy database metrics alongside queued writers and long-held locks point to application-level lock contention rather than database saturation.")
	case "hypothesis-dns-failure":
		b.WriteString("DNS resolution failures precede application connection errors, indicating name resolution as the upstream fault.")
	case "hypothesis-certificate-expiry":
		b.WriteString("Certificate expiry signals and TLS handshake failures cluster around the same window, indicating expired credentials.")
	case "hypothesis-retry-storm":
		b.WriteString("Retry amplification in logs and elevated request rates suggest a feedback loop rather than a single downstream hard failure.")
	case "hypothesis-resource-exhaustion":
		b.WriteString("Progressive resource growth followed by OOM or saturation events indicates exhaustion rather than an external dependency outage.")
	case "hypothesis-regional-failure":
		b.WriteString("Regional impairment events align with service impact in the affected geography.")
	default:
		fmt.Fprintf(&b, "The leading explanation (%s) best fits the observed sequence.", lead.Statement)
	}
	return b.String()
}

func writeScoped(b *strings.Builder, title string, items []model.ScopedItem) {
	if len(items) == 0 {
		return
	}
	fmt.Fprintf(b, "**%s:** ", title)
	parts := make([]string, 0, len(items))
	for _, it := range items {
		parts = append(parts, fmt.Sprintf("%s (%.0f%%)", it.Name, it.Confidence))
	}
	b.WriteString(strings.Join(parts, ", ") + "\n\n")
}

func countCategories(s *model.Session) int {
	set := map[model.Category]bool{}
	for _, e := range s.Evidence {
		set[e.Category] = true
	}
	return len(set)
}
