package signals_test

import (
	"testing"
	"time"

	"github.com/stackrail/incident-investigator/internal/model"
	"github.com/stackrail/incident-investigator/internal/signals"
)

func TestEvidenceRulesOutDeploy(t *testing.T) {
	cases := []struct {
		summary string
		payload map[string]any
		want    bool
	}{
		{"No deployment in the incident window", map[string]any{"new_deploy": false}, true},
		{"Deployed checkout-api v2.4.0", nil, false},
		{"Investigated deploy timeline", map[string]any{"deployed": "false"}, true},
	}
	for _, tc := range cases {
		e := &model.Evidence{
			ID: "e1", Timestamp: time.Now(), Category: model.CategoryDeploymentEvents,
			Summary: tc.summary, Payload: tc.payload,
		}
		if got := signals.EvidenceRulesOutDeploy(e); got != tc.want {
			t.Errorf("summary=%q payload=%v: got %v want %v", tc.summary, tc.payload, got, tc.want)
		}
	}
}

func TestAnalyzeIgnoresRuledOutDeployment(t *testing.T) {
	base := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	s := &model.Session{
		Evidence: []*model.Evidence{
			{
				ID: "dep-neg", Timestamp: base, Category: model.CategoryDeploymentEvents,
				Summary: "No deployment in the 24h window before incident", Payload: map[string]any{"new_deploy": false},
			},
			{
				ID: "log-1", Timestamp: base.Add(time.Minute), Category: model.CategoryApplicationLogs,
				Summary: "aws-cni failed to assign IP address", Entity: "aws-node",
			},
			{
				ID: "alert-1", Timestamp: base.Add(2 * time.Minute), Category: model.CategoryAlertEvents,
				Summary: "job.failed monitor triggered", Entity: "identity-compromise-report",
			},
		},
	}
	sig := signals.Analyze(s)
	if sig.FirstDeployment != nil {
		t.Fatalf("FirstDeployment should be nil for ruled-out deploy evidence, got %s", sig.FirstDeployment.ID)
	}
	if !signals.AllDeploymentEvidenceRuledOut(s) {
		t.Fatal("expected all deployment evidence ruled out")
	}
}

func TestAnalyzeIgnoresRuledOutConfig(t *testing.T) {
	s := &model.Session{
		Evidence: []*model.Evidence{
			{
				ID: "cfg-neg", Timestamp: time.Now(), Category: model.CategoryConfigurationChanges,
				Summary: "No configuration change in the incident window",
				Payload: map[string]any{"config_changed": false},
			},
		},
	}
	sig := signals.Analyze(s)
	if sig.Keywords["config"] {
		t.Fatal("config keyword should not fire for ruled-out configuration evidence")
	}
	if signals.AllConfigurationEvidenceRuledOut(s) && signals.HasAffirmativeConfigurationEvidence(s) {
		t.Fatal("contradictory config state")
	}
}
