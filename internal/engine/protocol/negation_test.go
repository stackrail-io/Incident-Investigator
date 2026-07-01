package protocol_test

import (
	"testing"
	"time"

	"github.com/stackrail/incident-investigator/internal/engine"
	"github.com/stackrail/incident-investigator/internal/engine/protocol"
	"github.com/stackrail/incident-investigator/internal/model"
)

func TestRuledOutDeploymentRejectsDeployBeforeErrors(t *testing.T) {
	base := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	s := &model.Session{
		Goal: model.GoalRootCause,
		Evidence: []*model.Evidence{
			{
				ID: "dep-neg", Timestamp: base, Category: model.CategoryDeploymentEvents,
				Summary: "No deployment in window", Payload: map[string]any{"new_deploy": false},
			},
			{ID: "log-1", Timestamp: base, Category: model.CategoryApplicationLogs, Summary: "CNI IP exhaustion"},
			{ID: "alert-1", Timestamp: base, Category: model.CategoryAlertEvents, Summary: "job.failed"},
		},
		Plan: &model.InvestigationPlan{
			Questions: []model.Question{{
				ID: "deploy-before-errors", Status: model.QuestionWaitingForEvidence,
				RequiredEvidence: []model.Category{
					model.CategoryDeploymentEvents, model.CategoryApplicationLogs, model.CategoryAlertEvents,
				},
			}},
		},
	}
	q := s.Plan.Questions[0]
	res := protocol.NewResolutionEngine().Resolve(&q, s, engine.Analyze(s))
	if res == nil {
		t.Fatal("expected resolution")
	}
	if res.Status != model.ResolutionRejected {
		t.Fatalf("status=%s want rejected", res.Status)
	}
}

func TestRuledOutConfigRejectsConfigChanged(t *testing.T) {
	s := &model.Session{
		Evidence: []*model.Evidence{
			{
				ID: "cfg-neg", Timestamp: time.Now(), Category: model.CategoryConfigurationChanges,
				Summary: "No config change", Payload: map[string]any{"config_changed": false},
			},
			{ID: "dep-1", Timestamp: time.Now(), Category: model.CategoryDeploymentEvents, Summary: "deploy timeline checked"},
		},
		Plan: &model.InvestigationPlan{
			Questions: []model.Question{{
				ID: "config-changed", Status: model.QuestionWaitingForEvidence,
				RequiredEvidence: []model.Category{model.CategoryConfigurationChanges, model.CategoryDeploymentEvents},
			}},
		},
	}
	q := s.Plan.Questions[0]
	res := protocol.NewResolutionEngine().Resolve(&q, s, engine.Analyze(s))
	if res == nil {
		t.Fatal("expected resolution")
	}
	if res.Status != model.ResolutionRejected {
		t.Fatalf("status=%s want rejected", res.Status)
	}
}

func TestAffirmativeConfigStillConfirms(t *testing.T) {
	s := &model.Session{
		Evidence: []*model.Evidence{
			{ID: "cfg-1", Timestamp: time.Now(), Category: model.CategoryConfigurationChanges, Summary: "ConfigMap checkout timeout lowered"},
			{ID: "dep-1", Timestamp: time.Now(), Category: model.CategoryDeploymentEvents, Summary: "Deployed checkout v2"},
		},
		Plan: &model.InvestigationPlan{
			Questions: []model.Question{{
				ID: "config-changed", Status: model.QuestionWaitingForEvidence,
				RequiredEvidence: []model.Category{model.CategoryConfigurationChanges, model.CategoryDeploymentEvents},
			}},
		},
	}
	q := s.Plan.Questions[0]
	res := protocol.NewResolutionEngine().Resolve(&q, s, engine.Analyze(s))
	if res == nil || res.Status != model.ResolutionConfirmed {
		t.Fatalf("want confirmed, got %+v", res)
	}
}
