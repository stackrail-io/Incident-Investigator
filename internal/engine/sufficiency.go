package engine

import (
	"fmt"

	"github.com/stackrail/incident-investigator/internal/model"
)

// SufficiencyEngine decides whether the investigation has enough evidence to answer.
type SufficiencyEngine interface {
	Evaluate(s *model.Session, sig Signals, coverage model.CoverageReport) model.SufficiencyReport
}

// HeuristicSufficiencyEngine is the default central decision maker.
type HeuristicSufficiencyEngine struct {
	MinConfidence float64
	MinCoverage   float64
}

// NewHeuristicSufficiencyEngine returns the default sufficiency engine.
func NewHeuristicSufficiencyEngine() *HeuristicSufficiencyEngine {
	return &HeuristicSufficiencyEngine{MinConfidence: 70, MinCoverage: 65}
}

// Evaluate implements SufficiencyEngine.
func (e *HeuristicSufficiencyEngine) Evaluate(s *model.Session, sig Signals, coverage model.CoverageReport) model.SufficiencyReport {
	report := model.SufficiencyReport{
		OverallConfidence: s.Confidence,
		Coverage:          coverage.Overall,
	}

	report.BlockingQuestions = deriveBlockingQuestions(s, sig)
	report.MissingEvidence = missingRequirements(s.MissingEvidence)

	canAnswer := len(s.Evidence) > 0 &&
		s.Confidence >= e.MinConfidence &&
		coverage.Overall >= e.MinCoverage &&
		len(report.BlockingQuestions) == 0 &&
		!hasHighSeverityContradiction(s.Contradictions)

	report.CanAnswer = canAnswer
	report.Reason = sufficiencyReason(canAnswer, s, coverage, report.BlockingQuestions)
	return report
}

func deriveBlockingQuestions(s *model.Session, sig Signals) []model.BlockingQuestion {
	var out []model.BlockingQuestion

	if sig.FirstDeployment != nil && sig.IncidentOnset == nil {
		out = append(out, model.BlockingQuestion{
			ID:       "blocking-deploy-before-errors",
			Question: "Did deployment happen before errors?",
			Priority: model.PriorityHigh,
			Reason:   "A deployment was observed but incident onset is not yet established.",
		})
	}
	if sig.FirstDeployment != nil && sig.Recovery == nil && sig.DeployBeforeIncident {
		out = append(out, model.BlockingQuestion{
			ID:       "blocking-rollback-success",
			Question: "Was rollback successful?",
			Priority: model.PriorityMedium,
			Reason:   "Deployment preceded symptoms; recovery evidence would confirm or refute the deploy theory.",
		})
	}
	if sig.Keywords["database"] && !s.HasCategory(model.CategoryDatabaseEvents) {
		out = append(out, model.BlockingQuestion{
			ID:       "blocking-database-health",
			Question: "Was database healthy?",
			Priority: model.PriorityHigh,
			Reason:   "Database symptoms detected but no database events submitted.",
		})
	}
	if sig.Keywords["latency"] && sig.Keywords["retry"] && !s.HasCategory(model.CategoryMetrics) {
		out = append(out, model.BlockingQuestion{
			ID:       "blocking-latency-before-retries",
			Question: "Did latency begin before retries?",
			Priority: model.PriorityMedium,
			Reason:   "Retry and latency signals need metrics to establish ordering.",
		})
	}
	if len(s.Hypotheses) >= 2 {
		lead := s.Hypotheses[0]
		runner := s.Hypotheses[1]
		if lead.Confidence-runner.Confidence < 15 {
			out = append(out, model.BlockingQuestion{
				ID:       "blocking-hypothesis-separation",
				Question: fmt.Sprintf("Can %q be distinguished from %q?", lead.Statement, runner.Statement),
				Priority: model.PriorityHigh,
				Reason:   "Leading hypotheses are too close in confidence to conclude.",
			})
		}
	}
	if out == nil {
		return []model.BlockingQuestion{}
	}
	return out
}

func missingRequirements(missing []model.EvidenceRequest) []model.EvidenceRequirement {
	out := make([]model.EvidenceRequirement, 0, len(missing))
	for _, m := range missing {
		out = append(out, model.EvidenceRequirement{
			Category: m.Category,
			Priority: m.Priority,
			Reason:   m.Reason,
		})
	}
	if out == nil {
		return []model.EvidenceRequirement{}
	}
	return out
}

func hasHighSeverityContradiction(cs []model.Contradiction) bool {
	for _, c := range cs {
		if c.Severity == "high" {
			return true
		}
	}
	return false
}

func sufficiencyReason(canAnswer bool, s *model.Session, coverage model.CoverageReport, blocking []model.BlockingQuestion) string {
	if canAnswer {
		return fmt.Sprintf("Confidence %.0f%% and coverage %.0f%% meet thresholds with no blocking questions.", s.Confidence, coverage.Overall)
	}
	if len(s.Evidence) == 0 {
		return "No evidence submitted yet."
	}
	if len(blocking) > 0 {
		return fmt.Sprintf("%d blocking question(s) remain before a trustworthy answer.", len(blocking))
	}
	if coverage.Overall < 65 {
		return fmt.Sprintf("Coverage %.0f%% is below the threshold; collect more categories.", coverage.Overall)
	}
	if s.Confidence < 70 {
		return fmt.Sprintf("Confidence %.0f%% is below the threshold.", s.Confidence)
	}
	return "High-severity contradictions must be resolved first."
}
