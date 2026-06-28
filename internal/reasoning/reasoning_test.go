package reasoning_test

import (
	"context"
	"testing"

	"github.com/stackrail/incident-investigator/internal/engine"
	"github.com/stackrail/incident-investigator/internal/model"
	"github.com/stackrail/incident-investigator/internal/reasoners"
	"github.com/stackrail/incident-investigator/internal/reasoning"
)

func testSession() *model.Session {
	return &model.Session{
		ID:       "inv-test",
		Question: "Why did checkout fail?",
		Service:  "checkout-api",
		Goal:     model.GoalRootCause,
		Graph:    model.NewEmptyGraphView(),
		Evidence: []*model.Evidence{{
			ID: "ev-1", Category: model.CategoryDeploymentEvents, Summary: "Deployed v2",
		}},
	}
}

func testEngines() engine.RuntimeEngines {
	return engine.RuntimeEngines{
		Planner:       engine.NewHeuristicPlanner(),
		Hypothesis:    engine.NewHeuristicHypothesisEngine(),
		Confidence:    engine.NewHeuristicConfidenceScorer(),
		Contradiction: engine.NewHeuristicContradictionDetector(),
		Missing:       engine.NewHeuristicMissingEvidenceDetector(),
		Graph:         engine.NewHeuristicGraphBuilder(),
		Timeline:      engine.NewHeuristicTimelineBuilder(),
		Blast:         engine.NewHeuristicBlastRadiusEstimator(),
		Coverage:      engine.NewHeuristicCoverageEngine(),
		Strategy:      engine.NewHeuristicStrategyEngine(),
		Sufficiency:   engine.NewHeuristicSufficiencyEngine(),
		Importance:    engine.NewHeuristicImportanceEngine(),
	}
}

func TestRegistryRegisterAndList(t *testing.T) {
	reg := reasoning.NewRegistry()
	reg.Register(reasoners.NewTemporalReasoner(testEngines()))
	reg.Register(reasoners.NewCausalReasoner())
	list := reg.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 reasoners, got %d", len(list))
	}
	if list[0].Priority() < list[1].Priority() {
		t.Error("expected descending priority order")
	}
}

func TestActionValidationRejectsResolvedQuestion(t *testing.T) {
	s := testSession()
	s.Plan = &model.InvestigationPlan{
		Questions: []model.Question{{ID: "q1", Status: model.QuestionAnswered}},
	}
	v := reasoning.NewValidator()
	err := v.Validate(s, nil, model.ReasoningAction{
		Type: model.ActionResolveQuestion, QuestionID: "q1",
	})
	if err == nil {
		t.Fatal("expected rejection for resolved question")
	}
}

func TestMergerWeightedConfidence(t *testing.T) {
	m := reasoning.NewMerger(model.StrategyWeightedVote)
	results := []model.ReasoningResult{
		{Reasoner: "temporal", Actions: []model.ReasoningAction{{
			Type: model.ActionIncreaseHypothesisConfidence, HypothesisID: "h1", Delta: 10,
		}}},
		{Reasoner: "causal", Actions: []model.ReasoningAction{{
			Type: model.ActionIncreaseHypothesisConfidence, HypothesisID: "h1", Delta: 12,
		}}},
		{Reasoner: "semantic", Actions: []model.ReasoningAction{{
			Type: model.ActionDecreaseHypothesisConfidence, HypothesisID: "h1", Delta: 4,
		}}},
	}
	merged := m.Merge(results)
	if len(merged) != 1 {
		t.Fatalf("expected 1 merged action, got %d", len(merged))
	}
	if merged[0].Delta <= 0 {
		t.Errorf("expected positive merged delta, got %v", merged[0].Delta)
	}
}

func TestHybridOrchestratorDeterministic(t *testing.T) {
	reg := reasoners.DefaultRegistry(testEngines())
	orch := reasoning.NewHybridOrchestrator(reg, reasoning.WithStrategy(model.StrategySequential))
	s := testSession()
	sig := engine.Analyze(s)
	inv := &reasoning.Investigation{Session: s, Signals: sig, Engines: testEngines()}

	c1, err := orch.Execute(context.Background(), inv)
	if err != nil {
		t.Fatal(err)
	}
	if len(c1.AppliedActions) == 0 {
		t.Fatal("expected applied actions")
	}
	if len(s.Hypotheses) == 0 {
		t.Error("expected hypotheses after cycle")
	}
	if s.Graph == nil || len(s.Graph.Nodes) == 0 {
		t.Error("expected populated graph")
	}
}

func TestSemanticReasonerMockLLM(t *testing.T) {
	mock := reasoning.NewMockLLM()
	mock.Response = &reasoning.LLMResponse{
		Findings: []model.ReasoningAction{{
			Type: model.ActionIncreaseHypothesisConfidence,
			HypothesisID: "hypothesis-deployment-caused", Delta: 5, Reason: "Rollback restored traffic.",
		}},
		Recommendations: []string{"Add canary deployment gates."},
	}
	sem := reasoning.NewSemanticReasoner(mock)
	s := testSession()
	s.Hypotheses = []model.Hypothesis{{ID: "hypothesis-deployment-caused", Confidence: 30}}
	res, err := sem.Analyze(context.Background(), &reasoning.Investigation{Session: s})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Actions) < 2 {
		t.Fatalf("expected actions from mock LLM, got %d", len(res.Actions))
	}
	if mock.LastPrompt == "" {
		t.Error("expected prompt to be sent to LLM")
	}
}

func TestParallelExecution(t *testing.T) {
	reg := reasoners.DefaultRegistry(testEngines())
	orch := reasoning.NewHybridOrchestrator(reg, reasoning.WithStrategy(model.StrategyParallel))
	s := testSession()
	inv := &reasoning.Investigation{Session: s, Signals: engine.Analyze(s), Engines: testEngines()}
	cycle, err := orch.Execute(context.Background(), inv)
	if err != nil {
		t.Fatal(err)
	}
	if len(cycle.Reasoners) < 3 {
		t.Errorf("expected multiple reasoners, got %d", len(cycle.Reasoners))
	}
}
