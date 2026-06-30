package engine

import (
	"testing"

	"github.com/stackrail/incident-investigator/internal/archetype/builtin"
	"github.com/stackrail/incident-investigator/internal/model"
)

func TestRecommendationsCoverRootCauseHypotheses(t *testing.T) {
	for _, a := range builtin.DefaultRegistry().All() {
		id := a.HypothesisID()
		switch id {
		case "hypothesis-unknown", "hypothesis-deployment-unrelated":
			continue
		}
		if _, ok := recommendationsByHypothesis[id]; !ok {
			t.Errorf("recommendationsByHypothesis missing %q (%s)", id, a.ID())
		}
	}
}

func TestCategorySupportsHypothesisCoversRootCauseHypotheses(t *testing.T) {
	categories := []model.Category{
		model.CategoryDeploymentEvents,
		model.CategoryConfigurationChanges,
		model.CategoryApplicationLogs,
		model.CategoryDatabaseEvents,
		model.CategoryMetrics,
		model.CategoryTraceEvents,
		model.CategoryNetworkEvents,
		model.CategorySecurityEvents,
		model.CategoryInfrastructureEvents,
		model.CategoryAlertEvents,
		model.CategoryHumanContext,
	}
	for _, a := range builtin.DefaultRegistry().All() {
		id := a.HypothesisID()
		switch id {
		case "hypothesis-unknown", "hypothesis-deployment-unrelated":
			continue
		}
		supported := false
		for _, cat := range categories {
			if categorySupportsHypothesis(cat, id) {
				supported = true
				break
			}
		}
		if !supported {
			t.Errorf("categorySupportsHypothesis has no category mapping for %q (%s)", id, a.ID())
		}
	}
}
