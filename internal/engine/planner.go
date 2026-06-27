package engine

import (
	"sort"
	"strings"

	"github.com/stackrail/incident-investigator/internal/model"
)

// Planner decides which categories of evidence the assistant should collect.
// It is dynamic: the desired set changes as evidence arrives.
type Planner interface {
	// Plan returns the full set of evidence the engine currently wants, given
	// the question and whatever evidence has already been submitted.
	Plan(s *model.Session, sig Signals) []model.EvidenceRequest
}

// HeuristicPlanner is a rule-based, vendor-neutral planner.
type HeuristicPlanner struct{}

// NewHeuristicPlanner returns the default planner.
func NewHeuristicPlanner() *HeuristicPlanner { return &HeuristicPlanner{} }

// Plan implements Planner.
func (p *HeuristicPlanner) Plan(s *model.Session, sig Signals) []model.EvidenceRequest {
	// requests is keyed by category so later (more specific) rules can upgrade
	// the priority or reason of an earlier request.
	requests := map[model.Category]model.EvidenceRequest{}
	add := func(c model.Category, pr model.Priority, reason string) {
		if existing, ok := requests[c]; ok {
			// Keep the strongest priority and the most specific reason.
			if pr.Weight() > existing.Priority.Weight() {
				existing.Priority = pr
			}
			if reason != "" {
				existing.Reason = reason
			}
			requests[c] = existing
			return
		}
		requests[c] = model.EvidenceRequest{Category: c, Priority: pr, Reason: reason}
	}

	// Baseline plan: every investigation starts by anchoring the timeline.
	add(model.CategoryDeploymentEvents, model.PriorityHigh,
		"Need to determine whether a deployment preceded the incident.")
	add(model.CategoryApplicationLogs, model.PriorityHigh,
		"Need application logs to characterize the failure mode.")
	add(model.CategoryAlertEvents, model.PriorityMedium,
		"Need alerts to establish incident onset and severity.")
	add(model.CategoryMetrics, model.PriorityMedium,
		"Need metrics to assess saturation and system health.")

	// Question-driven hints.
	q := strings.ToLower(s.Question + " " + s.Service)
	if matchesAny(q, signalKeywords["database"]) {
		add(model.CategoryDatabaseEvents, model.PriorityHigh,
			"The question references the database; collect database events.")
	}
	if matchesAny(q, signalKeywords["latency"]) {
		add(model.CategoryTraceEvents, model.PriorityMedium,
			"Latency questions are best localized with distributed traces.")
	}

	// Evidence-driven, dynamic expansion. This is what makes the planner feel
	// alive: as evidence arrives, the next questions sharpen.
	if sig.FirstDeployment != nil {
		add(model.CategoryConfigurationChanges, model.PriorityHigh,
			"A deployment was observed near the incident; configuration changes are a likely trigger.")
		add(model.CategoryTraceEvents, model.PriorityMedium,
			"Traces can localize which dependency broke after the deployment.")
	}
	if sig.Keywords["database"] || sig.Categories[model.CategoryMetrics] > 0 {
		add(model.CategoryDatabaseEvents, model.PriorityHigh,
			"Logs or metrics suggest database involvement; collect database events.")
	}
	if sig.Keywords["dns"] || sig.Keywords["network"] {
		add(model.CategoryNetworkEvents, model.PriorityHigh,
			"Network or DNS symptoms detected; collect network events.")
	}
	if sig.Keywords["cert"] {
		add(model.CategorySecurityEvents, model.PriorityHigh,
			"Certificate/TLS symptoms detected; collect security events.")
	}
	if sig.Keywords["restart"] || sig.Keywords["memory"] {
		add(model.CategoryInfrastructureEvents, model.PriorityMedium,
			"Restart or memory symptoms detected; collect infrastructure events.")
	}
	if sig.IncidentOnset != nil {
		add(model.CategoryHumanContext, model.PriorityLow,
			"Human context (chat, runbooks) can corroborate the timeline.")
	}

	return sortRequests(requests)
}

// sortRequests flattens the map into a deterministic, priority-ordered slice.
func sortRequests(m map[model.Category]model.EvidenceRequest) []model.EvidenceRequest {
	out := make([]model.EvidenceRequest, 0, len(m))
	for _, r := range m {
		out = append(out, r)
	}
	sort.SliceStable(out, func(i, j int) bool {
		wi, wj := out[i].Priority.Weight(), out[j].Priority.Weight()
		if wi != wj {
			return wi > wj
		}
		return out[i].Category < out[j].Category
	})
	return out
}
