package engine

import (
	"sort"

	"github.com/stackrail/incident-investigator/internal/model"
)

// CoverageEngine computes per-category evidence coverage.
type CoverageEngine interface {
	Compute(s *model.Session, required []model.EvidenceRequest) model.CoverageReport
}

// HeuristicCoverageEngine scores how well each evidence category is represented.
type HeuristicCoverageEngine struct{}

// NewHeuristicCoverageEngine returns the default coverage engine.
func NewHeuristicCoverageEngine() *HeuristicCoverageEngine { return &HeuristicCoverageEngine{} }

// goalCategories returns categories especially relevant to the investigation goal.
func goalCategories(goal model.InvestigationGoal) []model.Category {
	switch goal {
	case model.GoalTimeline:
		return []model.Category{
			model.CategoryAlertEvents, model.CategoryDeploymentEvents,
			model.CategoryInfrastructureEvents, model.CategoryApplicationLogs,
		}
	case model.GoalBlastRadius:
		return []model.Category{
			model.CategoryMetrics, model.CategoryTraceEvents, model.CategoryAlertEvents,
		}
	case model.GoalDeploymentVerification:
		return []model.Category{
			model.CategoryDeploymentEvents, model.CategoryConfigurationChanges,
			model.CategoryApplicationLogs, model.CategoryMetrics,
		}
	case model.GoalPerformanceRegression:
		return []model.Category{
			model.CategoryMetrics, model.CategoryTraceEvents, model.CategoryApplicationLogs,
		}
	case model.GoalAvailability:
		return []model.Category{
			model.CategoryAlertEvents, model.CategoryMetrics, model.CategoryInfrastructureEvents,
		}
	default:
		return []model.Category{
			model.CategoryDeploymentEvents, model.CategoryApplicationLogs,
			model.CategoryAlertEvents, model.CategoryMetrics,
		}
	}
}

// Compute implements CoverageEngine.
func (c *HeuristicCoverageEngine) Compute(s *model.Session, required []model.EvidenceRequest) model.CoverageReport {
	targets := map[model.Category]float64{}
	for _, g := range goalCategories(s.Goal) {
		targets[g] = 3 // desired depth for goal-relevant categories
	}
	for _, r := range required {
		w := r.Priority.Weight()
		if existing, ok := targets[r.Category]; !ok || w > existing {
			targets[r.Category] = w
		}
	}
	for _, e := range s.Evidence {
		if _, ok := targets[e.Category]; !ok {
			targets[e.Category] = 1
		}
	}

	counts := map[model.Category]int{}
	for _, e := range s.Evidence {
		counts[e.Category]++
	}

	categories := make([]model.CategoryCoverage, 0, len(targets))
	var totalWeight, weightedSum float64
	for cat, target := range targets {
		count := counts[cat]
		depth := float64(count) / target
		if depth > 1 {
			depth = 1
		}
		pct := round1(depth * 100)
		categories = append(categories, model.CategoryCoverage{
			Category: cat,
			Percent:  pct,
			Count:    count,
		})
		w := target
		totalWeight += w
		weightedSum += pct * w
	}

	sort.SliceStable(categories, func(i, j int) bool {
		if categories[i].Percent != categories[j].Percent {
			return categories[i].Percent > categories[j].Percent
		}
		return categories[i].Category < categories[j].Category
	})

	overall := 0.0
	if totalWeight > 0 {
		overall = round1(weightedSum / totalWeight)
	}
	return model.CoverageReport{Categories: categories, Overall: overall}
}
