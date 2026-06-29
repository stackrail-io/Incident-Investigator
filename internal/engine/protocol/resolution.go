package protocol

import (
	"fmt"
	"strings"

	"github.com/stackrail/incident-investigator/internal/engine"
	"github.com/stackrail/incident-investigator/internal/model"
)

// ResolutionEngine resolves protocol questions from evidence and signals.
type ResolutionEngine struct{}

// NewResolutionEngine returns the default resolver.
func NewResolutionEngine() *ResolutionEngine { return &ResolutionEngine{} }

// Resolve returns a resolution when enough evidence exists; nil otherwise.
func (r *ResolutionEngine) Resolve(q *model.Question, s *model.Session, sig engine.Signals) *model.QuestionResolution {
	if len(q.RequiredEvidence) == 0 {
		return nil
	}
	for _, c := range q.RequiredEvidence {
		if !s.HasCategory(c) {
			return nil
		}
	}

	support := evidenceForCategories(s, q.RequiredEvidence)
	res := &model.QuestionResolution{
		QuestionID:         q.ID,
		SupportingEvidence: support,
	}

	switch q.ID {
	case "deploy-before-errors":
		if sig.DeployBeforeIncident {
			res.Status = model.ResolutionConfirmed
			res.Confidence = 88
			res.Reason = "Deployment preceded the first symptom."
		} else if sig.DeployAfterIncident {
			res.Status = model.ResolutionRejected
			res.Confidence = 85
			res.Reason = "Deployment occurred after incident onset."
			res.ContradictingEvidence = support
		} else {
			res.Status = model.ResolutionInsufficientEvidence
			res.Confidence = 40
			res.Reason = "Temporal ordering between deployment and symptoms is inconclusive."
		}
	case "deploy-after-incident":
		if sig.DeployAfterIncident {
			res.Status = model.ResolutionConfirmed
			res.Confidence = 90
			res.Reason = "Deployment occurred after incident onset; deployment is unlikely to be root cause."
		} else if sig.DeployBeforeIncident {
			res.Status = model.ResolutionRejected
			res.Confidence = 85
			res.Reason = "Deployment preceded symptoms; deploy-caused theories remain plausible."
		} else {
			res.Status = model.ResolutionInsufficientEvidence
			res.Confidence = 35
			res.Reason = "Need deployment and symptom evidence to establish ordering."
		}
	case "rollback-restored-service":
		if sig.Recovery != nil {
			res.Status = model.ResolutionConfirmed
			res.Confidence = 91
			res.Reason = "Recovery evidence observed after rollback or remediation."
			if sig.Recovery != nil {
				res.SupportingEvidence = append(res.SupportingEvidence, sig.Recovery.ID)
			}
		} else {
			res.Status = model.ResolutionInsufficientEvidence
			res.Confidence = 35
			res.Reason = "No recovery evidence submitted yet."
		}
	case "database-healthy":
		if sig.Lock.Present {
			res.Status = model.ResolutionConfirmed
			res.Confidence = 85
			res.Reason = "Database capacity appears healthy; lock contention rather than saturation."
		} else if sig.Keywords["database"] && hasCategoryError(s, model.CategoryDatabaseEvents) {
			res.Status = model.ResolutionRejected
			res.Confidence = 87
			res.Reason = "Database events indicate saturation or errors."
		} else if s.HasCategory(model.CategoryDatabaseEvents) {
			res.Status = model.ResolutionConfirmed
			res.Confidence = 80
			res.Reason = "Database events show no saturation signals."
		} else {
			res.Status = model.ResolutionInsufficientEvidence
			res.Confidence = 30
			res.Reason = "Insufficient database evidence."
		}
	case "database-saturated":
		if sig.Lock.Present || sig.Lock.HealthyDatabaseMetrics {
			res.Status = model.ResolutionRejected
			res.Confidence = 84
			res.Reason = "Database capacity appears healthy or lock contention is present."
		} else if hasCategoryError(s, model.CategoryDatabaseEvents) || hasSaturationMetrics(s) {
			res.Status = model.ResolutionConfirmed
			res.Confidence = 88
			res.Reason = "Database events or metrics indicate saturation."
		} else if s.HasCategory(model.CategoryDatabaseEvents) {
			res.Status = model.ResolutionRejected
			res.Confidence = 72
			res.Reason = "Database events present without saturation signals."
		} else {
			res.Status = model.ResolutionInsufficientEvidence
			res.Confidence = 30
			res.Reason = "Need database events and metrics to assess saturation."
		}
	case "lock-contention-queue":
		if sig.Lock.Present {
			res.Status = model.ResolutionConfirmed
			res.Confidence = 90
			res.Reason = "Multiple database statements on the same entity completed together, indicating a lock queue."
			res.SupportingEvidence = append(res.SupportingEvidence, sig.Lock.HolderIDs...)
			res.SupportingEvidence = append(res.SupportingEvidence, sig.Lock.WaiterIDs...)
		} else if s.HasCategory(model.CategoryDatabaseEvents) {
			res.Status = model.ResolutionRejected
			res.Confidence = 75
			res.Reason = "Database events present but no lock-queue signature detected."
		} else {
			res.Status = model.ResolutionInsufficientEvidence
			res.Confidence = 30
			res.Reason = "Need database and trace evidence to assess lock contention."
		}
	case "lock-timeouts-missing":
		if sig.Lock.MissingLockTimeouts {
			res.Status = model.ResolutionConfirmed
			res.Confidence = 86
			res.Reason = "Configuration lacks statement_timeout or lock_timeout."
		} else if s.HasCategory(model.CategoryConfigurationChanges) {
			res.Status = model.ResolutionRejected
			res.Confidence = 70
			res.Reason = "No missing lock-timeout configuration detected."
		} else {
			res.Status = model.ResolutionInsufficientEvidence
			res.Confidence = 30
			res.Reason = "Need configuration evidence to assess lock timeouts."
		}
	case "latency-before-retries":
		if sig.Keywords["latency"] && sig.Keywords["retry"] {
			res.Status = model.ResolutionConfirmed
			res.Confidence = 78
			res.Reason = "Both latency and retry signals present in evidence."
		} else {
			res.Status = model.ResolutionInsufficientEvidence
			res.Confidence = 40
			res.Reason = "Need corroborating latency and retry evidence."
		}
	case "config-changed":
		if s.HasCategory(model.CategoryConfigurationChanges) || sig.Keywords["config"] {
			res.Status = model.ResolutionConfirmed
			res.Confidence = 82
			res.Reason = "Configuration change evidence present."
		} else {
			res.Status = model.ResolutionRejected
			res.Confidence = 70
			res.Reason = "No configuration change evidence found."
		}
	case "pods-restarted":
		if sig.Keywords["restart"] || s.HasCategory(model.CategoryInfrastructureEvents) {
			res.Status = model.ResolutionConfirmed
			res.Confidence = 84
			res.Reason = "Infrastructure or restart evidence present."
		} else {
			res.Status = model.ResolutionInsufficientEvidence
			res.Confidence = 35
			res.Reason = "No restart evidence yet."
		}
	case "traffic-shifted":
		if sig.Keywords["latency"] || sig.Categories[model.CategoryMetrics] > 0 {
			res.Status = model.ResolutionConfirmed
			res.Confidence = 75
			res.Reason = "Metrics suggest traffic or latency shift."
		} else {
			res.Status = model.ResolutionInsufficientEvidence
			res.Confidence = 30
			res.Reason = "Need metrics to confirm traffic shift."
		}
	case "memory-pressure":
		if sig.Keywords["memory"] || sig.Keywords["restart"] {
			res.Status = model.ResolutionConfirmed
			res.Confidence = 86
			res.Reason = "Memory pressure or restart signals present in evidence."
		} else if s.HasCategory(model.CategoryInfrastructureEvents) || s.HasCategory(model.CategoryMetrics) {
			res.Status = model.ResolutionRejected
			res.Confidence = 72
			res.Reason = "No memory pressure or OOM signals detected."
		} else {
			res.Status = model.ResolutionInsufficientEvidence
			res.Confidence = 30
			res.Reason = "Need metrics and infrastructure evidence to assess memory pressure."
		}
	case "dns-failure":
		if sig.Keywords["dns"] {
			res.Status = model.ResolutionConfirmed
			res.Confidence = 90
			res.Reason = "DNS resolution failure symptoms detected."
		} else if s.HasCategory(model.CategoryNetworkEvents) {
			res.Status = model.ResolutionRejected
			res.Confidence = 75
			res.Reason = "Network events present without DNS failure signals."
		} else {
			res.Status = model.ResolutionInsufficientEvidence
			res.Confidence = 30
			res.Reason = "Need network evidence to assess DNS failures."
		}
	case "certificate-expired":
		if sig.Keywords["cert"] {
			res.Status = model.ResolutionConfirmed
			res.Confidence = 92
			res.Reason = "TLS or certificate failure symptoms detected."
		} else if s.HasCategory(model.CategorySecurityEvents) {
			res.Status = model.ResolutionRejected
			res.Confidence = 70
			res.Reason = "Security events present without certificate expiry signals."
		} else {
			res.Status = model.ResolutionInsufficientEvidence
			res.Confidence = 30
			res.Reason = "Need security and log evidence to assess certificate health."
		}
	case "network-healthy":
		if sig.Keywords["dns"] || sig.Keywords["network"] {
			res.Status = model.ResolutionRejected
			res.Confidence = 86
			res.Reason = "Network or DNS symptoms detected."
		} else if s.HasCategory(model.CategoryNetworkEvents) {
			res.Status = model.ResolutionConfirmed
			res.Confidence = 78
			res.Reason = "Network events show no failure signals."
		} else {
			res.Status = model.ResolutionInsufficientEvidence
			res.Confidence = 35
			res.Reason = "Need network events to confirm."
		}
	case "dependency-unavailable":
		if sig.Keywords["dependency"] {
			res.Status = model.ResolutionConfirmed
			res.Confidence = 88
			res.Reason = "Downstream dependency failure symptoms detected."
		} else if s.HasCategory(model.CategoryTraceEvents) {
			res.Status = model.ResolutionRejected
			res.Confidence = 72
			res.Reason = "Trace evidence present without dependency failure signals."
		} else {
			res.Status = model.ResolutionInsufficientEvidence
			res.Confidence = 30
			res.Reason = "Need logs and traces to assess dependency health."
		}
	case "latency-regression":
		if sig.Keywords["performance"] || (sig.Keywords["latency"] && !sig.Keywords["database"] && !sig.Keywords["retry"]) {
			res.Status = model.ResolutionConfirmed
			res.Confidence = 82
			res.Reason = "Performance regression symptoms detected."
		} else {
			res.Status = model.ResolutionInsufficientEvidence
			res.Confidence = 35
			res.Reason = "Need metrics and traces to confirm a regression."
		}
	case "vendor-outage":
		if sig.Keywords["external"] {
			res.Status = model.ResolutionConfirmed
			res.Confidence = 90
			res.Reason = "Third-party or vendor outage symptoms detected."
		} else {
			res.Status = model.ResolutionInsufficientEvidence
			res.Confidence = 30
			res.Reason = "Need external provider evidence."
		}
	case "auth-failure":
		if sig.Keywords["auth"] {
			res.Status = model.ResolutionConfirmed
			res.Confidence = 89
			res.Reason = "Authentication or authorization failure detected."
		} else {
			res.Status = model.ResolutionInsufficientEvidence
			res.Confidence = 30
			res.Reason = "Need security and log evidence for auth failures."
		}
	case "manual-change":
		if sig.Keywords["human"] || s.HasCategory(model.CategoryHumanContext) {
			res.Status = model.ResolutionConfirmed
			res.Confidence = 87
			res.Reason = "Human or operational error evidence present."
		} else {
			res.Status = model.ResolutionInsufficientEvidence
			res.Confidence = 30
			res.Reason = "Need human context evidence."
		}
	case "capacity-exceeded":
		if sig.Keywords["capacity"] {
			res.Status = model.ResolutionConfirmed
			res.Confidence = 85
			res.Reason = "Capacity or autoscaling stress detected."
		} else {
			res.Status = model.ResolutionInsufficientEvidence
			res.Confidence = 30
			res.Reason = "Need metrics and infrastructure evidence."
		}
	case "security-breach":
		if sig.Keywords["security"] {
			res.Status = model.ResolutionConfirmed
			res.Confidence = 91
			res.Reason = "Security incident symptoms detected."
		} else if s.HasCategory(model.CategorySecurityEvents) {
			res.Status = model.ResolutionRejected
			res.Confidence = 70
			res.Reason = "Security events without breach indicators."
		} else {
			res.Status = model.ResolutionInsufficientEvidence
			res.Confidence = 30
			res.Reason = "Need security event evidence."
		}
	default:
		res.Status = model.ResolutionConfirmed
		res.Confidence = 70
		res.Reason = "Required evidence categories are present."
	}
	if res.Status == model.ResolutionInsufficientEvidence {
		return nil
	}
	return res
}

func evidenceForCategories(s *model.Session, cats []model.Category) []string {
	want := map[model.Category]bool{}
	for _, c := range cats {
		want[c] = true
	}
	var ids []string
	for _, e := range s.Evidence {
		if want[e.Category] {
			ids = append(ids, e.ID)
		}
	}
	return ids
}

func hasCategoryError(s *model.Session, cat model.Category) bool {
	for _, e := range s.Evidence {
		if e.Category != cat {
			continue
		}
		text := strings.ToLower(e.Summary)
		if strings.Contains(text, "saturat") || strings.Contains(text, "timeout") ||
			strings.Contains(text, "error") || strings.Contains(text, "exhaust") {
			return true
		}
	}
	return false
}

func hasSaturationMetrics(s *model.Session) bool {
	for _, e := range s.Evidence {
		if e.Category != model.CategoryMetrics {
			continue
		}
		text := strings.ToLower(e.Summary + " " + e.Entity)
		for _, k := range e.Payload {
			text += " " + strings.ToLower(fmt.Sprint(k))
		}
		if strings.Contains(text, "saturat") || strings.Contains(text, "100/100") ||
			strings.Contains(text, "exhaust") || strings.Contains(text, "cpu 9") {
			return true
		}
	}
	return false
}
