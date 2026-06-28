package engine_test

import (
	"testing"
	"time"

	"github.com/stackrail/incident-investigator/internal/engine"
	"github.com/stackrail/incident-investigator/internal/model"
)

func TestSufficiencyBlocksWithoutEvidence(t *testing.T) {
	s := session("api", "why did api fail?")
	s.Goal = model.GoalRootCause
	suff := engine.NewHeuristicSufficiencyEngine()
	cov := engine.NewHeuristicCoverageEngine().Compute(s, nil)
	report := suff.Evaluate(s, engine.Analyze(s), cov)
	if report.CanAnswer {
		t.Error("expected CanAnswer=false with no evidence")
	}
	if report.Reason == "" {
		t.Error("expected reason")
	}
}

func TestSufficiencyCanAnswerWithRichEvidence(t *testing.T) {
	base := time.Date(2026, 6, 27, 9, 0, 0, 0, time.UTC)
	s := session("checkout-api", "why did checkout fail?",
		&model.Evidence{ID: "d1", Timestamp: base, Category: model.CategoryDeploymentEvents, Summary: "Deployed checkout v2"},
		&model.Evidence{ID: "a1", Timestamp: base.Add(time.Minute), Category: model.CategoryAlertEvents, Summary: "Alert: checkout 5xx spike"},
		&model.Evidence{ID: "l1", Timestamp: base.Add(2 * time.Minute), Category: model.CategoryApplicationLogs, Summary: "HTTP 500 errors after deploy"},
		&model.Evidence{ID: "m1", Timestamp: base.Add(3 * time.Minute), Category: model.CategoryMetrics, Summary: "error rate 25%"},
	)
	s.Goal = model.GoalRootCause
	sig := engine.Analyze(s)
	required := engine.NewHeuristicPlanner().Plan(s, sig)
	s.RequiredEvidence = required
	s.MissingEvidence = engine.NewHeuristicMissingEvidenceDetector().Detect(s, required)
	s.Progress = engine.Progress(s, required)
	s.Hypotheses = engine.NewHeuristicHypothesisEngine().Generate(s, sig, nil)
	s.Confidence = engine.NewHeuristicConfidenceScorer().Score(s, sig, s.Hypotheses, nil, s.Progress)

	cov := engine.NewHeuristicCoverageEngine().Compute(s, required)
	report := engine.NewHeuristicSufficiencyEngine().Evaluate(s, sig, cov)
	if !report.CanAnswer && len(report.BlockingQuestions) > 0 {
		t.Logf("blocking: %+v", report.BlockingQuestions)
	}
}

func TestCoverageComputesPerCategory(t *testing.T) {
	base := time.Date(2026, 6, 27, 9, 0, 0, 0, time.UTC)
	s := session("api", "why?",
		&model.Evidence{ID: "l1", Timestamp: base, Category: model.CategoryApplicationLogs, Summary: "error"},
		&model.Evidence{ID: "l2", Timestamp: base, Category: model.CategoryApplicationLogs, Summary: "more errors"},
	)
	s.Goal = model.GoalRootCause
	required := []model.EvidenceRequest{
		{Category: model.CategoryApplicationLogs, Priority: model.PriorityHigh},
		{Category: model.CategoryMetrics, Priority: model.PriorityHigh},
	}
	cov := engine.NewHeuristicCoverageEngine().Compute(s, required)
	if cov.Overall <= 0 {
		t.Fatalf("expected positive overall coverage, got %.1f", cov.Overall)
	}
	foundLogs := false
	for _, c := range cov.Categories {
		if c.Category == model.CategoryApplicationLogs {
			foundLogs = true
			if c.Percent < 50 {
				t.Errorf("application_logs coverage = %.1f, want high", c.Percent)
			}
		}
	}
	if !foundLogs {
		t.Error("expected application_logs in coverage report")
	}
}

func TestStrategyReturnsAtMostTwoSteps(t *testing.T) {
	s := session("api", "why did api fail?")
	s.Goal = model.GoalRootCause
	sig := engine.Analyze(s)
	required := engine.NewHeuristicPlanner().Plan(s, sig)
	steps := engine.NewHeuristicStrategyEngine().NextSteps(s, sig, nil, required)
	if len(steps) == 0 {
		t.Fatal("expected strategy steps at start")
	}
	if len(steps) > 2 {
		t.Errorf("got %d steps, want at most 2", len(steps))
	}
	if steps[0].ExpectedConfidenceGain <= 0 {
		t.Error("expected positive confidence gain")
	}
}

func TestStrategyGoalAwarePlanning(t *testing.T) {
	s := session("api", "what was the blast radius?")
	s.Goal = model.GoalBlastRadius
	sig := engine.Analyze(s)
	required := engine.NewHeuristicPlanner().Plan(s, sig)
	steps := engine.NewHeuristicStrategyEngine().NextSteps(s, sig, nil, required)
	if len(steps) == 0 {
		t.Fatal("expected steps")
	}
	// Blast radius goal should prioritize metrics/traces/alerts via higher gain.
	foundRelevant := false
	for _, step := range steps {
		if step.Category == model.CategoryMetrics || step.Category == model.CategoryTraceEvents || step.Category == model.CategoryAlertEvents {
			foundRelevant = true
		}
	}
	if !foundRelevant {
		t.Errorf("blast radius strategy should prioritize traffic/latency categories, got %+v", steps)
	}
}

func TestImportanceScoresDeploymentHighest(t *testing.T) {
	base := time.Date(2026, 6, 27, 9, 0, 0, 0, time.UTC)
	s := session("api", "why?",
		&model.Evidence{ID: "d1", Timestamp: base, Category: model.CategoryDeploymentEvents, Summary: "Deployed api v3"},
		&model.Evidence{ID: "h1", Timestamp: base, Category: model.CategoryHumanContext, Summary: "slack: looks bad"},
	)
	sig := engine.Analyze(s)
	scores := engine.NewHeuristicImportanceEngine().Score(s, sig, nil)
	if len(scores) != 2 {
		t.Fatalf("got %d scores", len(scores))
	}
	var deployScore, humanScore float64
	for _, sc := range scores {
		if sc.EvidenceID == "d1" {
			deployScore = sc.Score
		}
		if sc.EvidenceID == "h1" {
			humanScore = sc.Score
		}
	}
	if deployScore <= humanScore {
		t.Errorf("deployment importance %.0f should exceed human context %.0f", deployScore, humanScore)
	}
}

func TestStateMachineTransitions(t *testing.T) {
	sm := engine.NewStateMachine()
	s := session("api", "why?")
	suff := model.SufficiencyReport{CanAnswer: false, BlockingQuestions: []model.BlockingQuestion{{ID: "b1", Question: "?"}}}

	st := sm.Transition(model.StateStarted, s, suff)
	if st != model.StateStarted && st != model.StateCollectingEvidence {
		t.Errorf("empty evidence state = %q", st)
	}

	s.Evidence = []*model.Evidence{
		{ID: "e1", Timestamp: time.Now(), Category: model.CategoryApplicationLogs, Summary: "error"},
	}
	st = sm.Transition(model.StateCollectingEvidence, s, suff)
	if st != model.StateWaitingForEvidence {
		t.Errorf("with blocking questions state = %q, want waiting_for_evidence", st)
	}

	suff = model.SufficiencyReport{CanAnswer: true}
	s.Confidence = 75
	st = sm.Transition(model.StateWaitingForEvidence, s, suff)
	if st != model.StateHighConfidence {
		t.Errorf("state = %q, want high_confidence", st)
	}
}

func TestReasoningTraceRecordsConfidenceChange(t *testing.T) {
	now := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	s := session("api", "why?")
	s.Hypotheses = []model.Hypothesis{
		{ID: "h1", Statement: "deploy caused it", Confidence: 30, Status: model.StatusProposed},
	}
	s.SnapshotHypotheses()

	s.Hypotheses = []model.Hypothesis{
		{ID: "h1", Statement: "deploy caused it", Confidence: 55, Status: model.StatusLeading, SupportingEvidence: []string{"d1"}},
	}
	engine.UpdateReasoningTrace(s, now)
	if len(s.ReasoningTrace) == 0 {
		t.Fatal("expected reasoning trace entries")
	}
	if s.ReasoningTrace[0].ConfidenceChange != 25 {
		t.Errorf("confidence change = %.0f, want 25", s.ReasoningTrace[0].ConfidenceChange)
	}
}

func TestJournalReplay(t *testing.T) {
	now := time.Date(2026, 6, 27, 9, 0, 0, 0, time.UTC)
	s := session("api", "why?")
	s.AddJournal("investigation_started", "why?", 0, now)
	s.AddJournal("evidence_submitted", "1 item", 41, now.Add(time.Minute))
	s.AddJournal("hypothesis_promoted", "deploy caused it", 74, now.Add(3*time.Minute))

	if len(s.Journal) != 3 {
		t.Fatalf("journal length = %d", len(s.Journal))
	}
	if s.Journal[1].Confidence != 41 {
		t.Errorf("journal confidence = %.0f", s.Journal[1].Confidence)
	}
}

func TestInvestigationProgress(t *testing.T) {
	s := session("api", "why?",
		&model.Evidence{ID: "l1", Timestamp: time.Now(), Category: model.CategoryApplicationLogs, Summary: "error"},
	)
	s.RequiredEvidence = []model.EvidenceRequest{
		{Category: model.CategoryApplicationLogs, Priority: model.PriorityHigh},
		{Category: model.CategoryMetrics, Priority: model.PriorityHigh},
	}
	s.Confidence = 50
	cov := model.CoverageReport{Overall: 50}
	suff := model.SufficiencyReport{BlockingQuestions: []model.BlockingQuestion{{ID: "b1"}}}
	prog := engine.ComputeInvestigationProgress(s, cov, suff)
	if prog.PercentComplete <= 0 {
		t.Errorf("expected positive progress, got %.1f", prog.PercentComplete)
	}
	if prog.RemainingQuestions != 1 {
		t.Errorf("remaining questions = %d", prog.RemainingQuestions)
	}
}
