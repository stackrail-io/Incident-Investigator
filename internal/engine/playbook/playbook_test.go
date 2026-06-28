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
	if len(pb.Questions) < 5 {
		t.Fatalf("got %d questions, want at least 5", len(pb.Questions))
	}
	found := false
	for _, q := range pb.Questions {
		if q.ID == "deploy-before-errors" {
			found = true
			if len(q.Requires) < 2 {
				t.Errorf("deploy-before-errors requires = %v", q.Requires)
			}
			if len(q.Effects) < 2 {
				t.Errorf("expected IF TRUE/FALSE effects")
			}
		}
	}
	if !found {
		t.Error("deploy-before-errors not in playbook")
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
