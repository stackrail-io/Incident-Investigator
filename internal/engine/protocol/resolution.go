package protocol

import (
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
