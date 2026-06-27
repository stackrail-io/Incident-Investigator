package engine

import (
	"fmt"

	"github.com/stackrail/incident-investigator/internal/model"
)

// ContradictionDetector finds internal inconsistencies in the evidence.
type ContradictionDetector interface {
	Detect(s *model.Session, sig Signals) []model.Contradiction
}

// HeuristicContradictionDetector implements robust, schema-free checks.
type HeuristicContradictionDetector struct{}

// NewHeuristicContradictionDetector returns the default detector.
func NewHeuristicContradictionDetector() *HeuristicContradictionDetector {
	return &HeuristicContradictionDetector{}
}

// Detect implements ContradictionDetector.
func (d *HeuristicContradictionDetector) Detect(s *model.Session, sig Signals) []model.Contradiction {
	var out []model.Contradiction

	// 1. A deployment that happened after the incident already started cannot be
	//    the root cause, yet is often blamed. Surface it explicitly.
	if sig.DeployAfterIncident && sig.FirstDeployment != nil && sig.IncidentOnset != nil {
		out = append(out, model.Contradiction{
			ID: "contradiction-deploy-after-incident",
			Description: fmt.Sprintf(
				"Deployment %q occurred at %s, after the incident began at %s; it cannot be the root cause.",
				sig.FirstDeployment.Summary,
				sig.FirstDeployment.Timestamp.Format("15:04:05"),
				sig.IncidentOnset.Timestamp.Format("15:04:05"),
			),
			Severity:     "high",
			EvidenceRefs: []string{sig.FirstDeployment.ID, sig.IncidentOnset.ID},
		})
	}

	// 2. Recovery that predates the incident onset is an impossible sequence,
	//    usually a clock or data-quality problem.
	if sig.Recovery != nil && sig.IncidentOnset != nil &&
		sig.Recovery.Timestamp.Before(sig.IncidentOnset.Timestamp) {
		out = append(out, model.Contradiction{
			ID:           "contradiction-recovery-before-incident",
			Description:  "Recovery evidence is timestamped before the incident began (impossible sequence; check clocks).",
			Severity:     "medium",
			EvidenceRefs: []string{sig.Recovery.ID, sig.IncidentOnset.ID},
		})
	}

	// 3. A successful deployment with errors but no observed restart/rollout of
	//    pods suggests the deployment never actually reached the runtime.
	if dep := sig.FirstDeployment; dep != nil {
		text := haystack(dep)
		succeeded := matchesAny(text, []string{"success", "succeeded", "completed", "healthy"})
		if succeeded && sig.IncidentOnset != nil && !sig.Keywords["restart"] {
			out = append(out, model.Contradiction{
				ID:           "contradiction-deploy-success-no-restart",
				Description:  "Deployment reported success but no pod restart/rollout was observed despite the incident; the change may not have taken effect.",
				Severity:     "low",
				EvidenceRefs: []string{dep.ID},
			})
		}
	}

	// 4. Duplicate evidence (same category, entity and summary) inflates apparent
	//    corroboration and should be flagged.
	seen := map[string]string{}
	for _, e := range sortedByTime(s.Evidence) {
		key := string(e.Category) + "|" + e.Entity + "|" + e.Summary
		if first, ok := seen[key]; ok {
			out = append(out, model.Contradiction{
				ID:           "contradiction-duplicate-" + e.ID,
				Description:  fmt.Sprintf("Duplicate evidence detected: %q appears more than once.", e.Summary),
				Severity:     "low",
				EvidenceRefs: []string{first, e.ID},
			})
		} else {
			seen[key] = e.ID
		}
	}

	return out
}
