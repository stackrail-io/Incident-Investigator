package protocol_test

import (
	"testing"
	"time"

	"github.com/stackrail/incident-investigator/internal/engine"
	"github.com/stackrail/incident-investigator/internal/engine/protocol"
	"github.com/stackrail/incident-investigator/internal/model"
)

func testSession() *model.Session {
	return &model.Session{
		ID:       "inv-test",
		Question: "Why did checkout fail?",
		Goal:     model.GoalRootCause,
		Evidence: []*model.Evidence{},
	}
}

func TestProtocolCreatesPlanAndQuestions(t *testing.T) {
	s := testSession()
	pe, err := protocol.NewEngine(s.Goal)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	sig := engine.Analyze(s)
	pe.Run(s, sig, time.Now(), 0)

	if s.Plan == nil {
		t.Fatal("expected plan")
	}
	if len(s.Plan.Questions) == 0 {
		t.Fatal("expected questions")
	}
	if len(s.Plan.EvidenceRequests) == 0 {
		t.Fatal("expected evidence requests")
	}
	if s.Plan.CurrentStage != model.StageEvidenceCollection {
		t.Errorf("stage = %q", s.Plan.CurrentStage)
	}
}

func TestQuestionResolutionDeployBeforeErrors(t *testing.T) {
	base := time.Date(2026, 6, 27, 9, 0, 0, 0, time.UTC)
	s := testSession()
	s.Evidence = []*model.Evidence{
		{ID: "d1", Timestamp: base, Category: model.CategoryDeploymentEvents, Summary: "Deployed checkout v2"},
		{ID: "a1", Timestamp: base.Add(time.Minute), Category: model.CategoryAlertEvents, Summary: "Alert: 5xx spike"},
		{ID: "l1", Timestamp: base.Add(2 * time.Minute), Category: model.CategoryApplicationLogs, Summary: "HTTP 500 errors"},
	}

	pe, _ := protocol.NewEngine(s.Goal)
	sig := engine.Analyze(s)
	turn := pe.Run(s, sig, time.Now(), 0)

	if len(turn.ResolvedQuestions) == 0 {
		t.Fatal("expected at least one resolved question")
	}
	found := false
	for _, q := range s.Plan.Questions {
		if q.ID == "deploy-before-errors" && q.Status == model.QuestionAnswered {
			found = true
		}
	}
	if !found {
		t.Errorf("deploy-before-errors not answered; questions=%+v", s.Plan.Questions)
	}
}

func TestQuestionGraphDependencies(t *testing.T) {
	base := time.Date(2026, 6, 27, 9, 0, 0, 0, time.UTC)
	s := testSession()
	s.Evidence = []*model.Evidence{
		{ID: "d1", Timestamp: base, Category: model.CategoryDeploymentEvents, Summary: "Deployed checkout v2"},
		{ID: "a1", Timestamp: base.Add(time.Minute), Category: model.CategoryAlertEvents, Summary: "Alert: 5xx spike"},
		{ID: "l1", Timestamp: base.Add(2 * time.Minute), Category: model.CategoryApplicationLogs, Summary: "HTTP 500 errors"},
	}

	pe, _ := protocol.NewEngine(s.Goal)
	sig := engine.Analyze(s)
	pe.Run(s, sig, time.Now(), 0)

	if len(s.QuestionGraph.Edges) == 0 {
		t.Fatal("expected question graph edges after resolving deploy question")
	}
	foundDep := false
	for _, e := range s.QuestionGraph.Edges {
		if e.Relation == "depends_on" {
			foundDep = true
		}
	}
	if !foundDep {
		t.Errorf("expected dependency edges, got %+v", s.QuestionGraph.Edges)
	}
}

func TestDynamicQuestionGeneration(t *testing.T) {
	base := time.Date(2026, 6, 27, 9, 0, 0, 0, time.UTC)
	s := testSession()
	s.Evidence = []*model.Evidence{
		{ID: "c1", Timestamp: base, Category: model.CategoryConfigurationChanges, Summary: "Feature flag enabled bad path"},
	}

	pe, _ := protocol.NewEngine(s.Goal)
	sig := engine.Analyze(s)
	turn := pe.Run(s, sig, time.Now(), 0)

	foundConfig := false
	for _, q := range s.Plan.Questions {
		if q.ID == "config-changed" {
			foundConfig = true
		}
	}
	if !foundConfig {
		t.Errorf("expected config-changed question, new=%+v all=%+v", turn.NewQuestions, s.Plan.Questions)
	}
}

func TestListOpenQuestionsSorted(t *testing.T) {
	s := testSession()
	pe, _ := protocol.NewEngine(s.Goal)
	pe.Run(s, engine.Analyze(s), time.Now(), 0)

	open := protocol.ListOpenQuestions(s.Plan)
	if len(open) == 0 {
		t.Fatal("expected open questions")
	}
	for i := 1; i < len(open); i++ {
		if open[i].Priority > open[i-1].Priority {
			t.Errorf("questions not sorted by priority: %d before %d", open[i-1].Priority, open[i].Priority)
		}
	}
}

func TestResolveQuestionExplicit(t *testing.T) {
	s := testSession()
	pe, _ := protocol.NewEngine(s.Goal)
	pe.Run(s, engine.Analyze(s), time.Now(), 0)

	res, err := pe.ResolveQuestion(s, "deploy-before-errors", true, "confirmed by operator", time.Now())
	if err != nil {
		t.Fatalf("ResolveQuestion: %v", err)
	}
	if res.Status != model.ResolutionConfirmed {
		t.Errorf("status = %q", res.Status)
	}
}

func TestPlaybookEffectsAdjustHypotheses(t *testing.T) {
	base := time.Date(2026, 6, 27, 9, 0, 0, 0, time.UTC)
	s := testSession()
	s.Evidence = []*model.Evidence{
		{ID: "d1", Timestamp: base, Category: model.CategoryDeploymentEvents, Summary: "Deployed checkout v2"},
		{ID: "a1", Timestamp: base.Add(time.Minute), Category: model.CategoryAlertEvents, Summary: "Alert: 5xx spike"},
		{ID: "l1", Timestamp: base.Add(2 * time.Minute), Category: model.CategoryApplicationLogs, Summary: "HTTP 500 errors"},
	}
	s.Hypotheses = []model.Hypothesis{
		{ID: "hypothesis-deployment-caused", Confidence: 40, Status: model.StatusLeading},
		{ID: "hypothesis-unknown", Confidence: 60, Status: model.StatusSupported},
	}

	pe, _ := protocol.NewEngine(s.Goal)
	pe.Run(s, engine.Analyze(s), time.Now(), 0)

	for _, h := range s.Hypotheses {
		if h.ID == "hypothesis-deployment-caused" && h.Confidence <= 40 {
			t.Errorf("expected deployment hypothesis confidence to increase, got %.1f", h.Confidence)
		}
	}
}
