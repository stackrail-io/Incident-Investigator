package playbook_test

import (
	"testing"

	"github.com/stackrail/incident-investigator/internal/engine/playbook"
	"github.com/stackrail/incident-investigator/internal/model"
)

func TestParseRootCausePlaybook(t *testing.T) {
	pb, err := playbook.ForGoal(model.GoalRootCause)
	if err != nil {
		t.Fatalf("ForGoal: %v", err)
	}
	if len(pb.Questions) < 14 {
		t.Fatalf("got %d questions, want at least 14", len(pb.Questions))
	}
	found := map[string]bool{}
	for _, q := range pb.Questions {
		found[q.ID] = true
		if q.ID == "deploy-before-errors" {
			if len(q.Requires) < 2 {
				t.Errorf("deploy-before-errors requires = %v", q.Requires)
			}
			if len(q.Effects) < 2 {
				t.Errorf("expected IF TRUE/FALSE effects")
			}
			if q.Title == "" || q.Priority < 80 {
				t.Errorf("deploy-before-errors metadata: title=%q priority=%d", q.Title, q.Priority)
			}
		}
	}
	for _, id := range []string{
		"deploy-before-errors", "deploy-after-incident", "database-saturated",
		"lock-contention-queue", "lock-timeouts-missing", "certificate-expired",
		"dns-failure", "memory-pressure",
	} {
		if !found[id] {
			t.Errorf("missing question %q", id)
		}
	}
}

func TestParseCustomPlaybook(t *testing.T) {
	src := `
QUESTION q1
Was service healthy?
REQUIRES metrics application_logs
IF TRUE Increase hypothesis-unknown 10
`
	pb, err := playbook.Parse("test", model.GoalCustom, src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(pb.Questions) != 1 {
		t.Fatalf("got %d questions", len(pb.Questions))
	}
	if pb.Questions[0].Title != "Was service healthy?" {
		t.Errorf("title = %q", pb.Questions[0].Title)
	}
}
