package reasoners

import (
	"context"
	"fmt"
	"strings"

	"github.com/stackrail/incident-investigator/internal/engine"
	"github.com/stackrail/incident-investigator/internal/graph"
	"github.com/stackrail/incident-investigator/internal/model"
	"github.com/stackrail/incident-investigator/internal/reasoning"
)

func asSignals(v any) engine.Signals {
	if s, ok := v.(engine.Signals); ok {
		return s
	}
	return engine.Signals{}
}

// TemporalReasoner handles ordering, timelines, and temporal validation.
type TemporalReasoner struct {
	engines engine.RuntimeEngines
}

// NewTemporalReasoner returns the temporal reasoner.
func NewTemporalReasoner(eng engine.RuntimeEngines) *TemporalReasoner {
	return &TemporalReasoner{engines: eng}
}

func (t *TemporalReasoner) Name() string                  { return "temporal" }
func (t *TemporalReasoner) Priority() int               { return 100 }
func (t *TemporalReasoner) Supports(_ *model.Session) bool { return true }

func (t *TemporalReasoner) Analyze(ctx context.Context, inv *reasoning.Investigation) (*model.ReasoningResult, error) {
	_ = ctx
	s := inv.Session
	sig := asSignals(inv.Signals)
	e := t.engines

	required := e.Planner.Plan(s, sig)
	missing := e.Missing.Detect(s, required)
	progress := engine.Progress(s, required)
	timeline := e.Timeline.Build(s)

	var actions []model.ReasoningAction
	actions = append(actions,
		model.ReasoningAction{
			Type: model.ActionSetEvidencePlan, Reasoner: t.Name(),
			RequiredEvidence: required, MissingEvidence: missing,
			Reason: "Planner evidence requirements from temporal signals.",
		},
		model.ReasoningAction{
			Type: model.ActionSetTimeline, Reasoner: t.Name(),
			Timeline: timeline, Reason: "Chronological timeline from evidence ordering.",
		},
		model.ReasoningAction{
			Type: model.ActionSetProgress, Reasoner: t.Name(),
			Progress: progress,
		},
	)

	var findings []model.Finding
	if sig.DeployBeforeIncident {
		findings = append(findings, model.Finding{
			Type: "temporal_ordering", Summary: "Deployment preceded incident onset.",
			Confidence: 85, Reason: "Evidence timestamps show deploy-before-error ordering.",
		})
	}
	if sig.Keywords["rollback"] && sig.Keywords["recovery"] {
		findings = append(findings, model.Finding{
			Type: "temporal_ordering", Summary: "Rollback preceded recovery.",
			Confidence: 80,
		})
	}

	if inv.Graph != nil {
		for i := 1; i < len(timeline); i++ {
			prev, cur := timeline[i-1], timeline[i]
			if len(prev.EvidenceRefs) == 0 || len(cur.EvidenceRefs) == 0 {
				continue
			}
			from, to := "ev:"+prev.EvidenceRefs[0], "ev:"+cur.EvidenceRefs[0]
			if inv.Graph.HasNode(from) && inv.Graph.HasNode(to) {
				actions = append(actions, model.ReasoningAction{
					Type: model.ActionLinkGraphNodes, Reasoner: t.Name(),
					GraphEdge: &model.GraphEdge{
						From: from, To: to, Type: model.EdgeOccurredBefore,
						Confidence: 100, Timestamp: cur.Timestamp,
						Reason: "Inferred from timeline ordering.",
					},
				})
			}
		}
	}

	return &model.ReasoningResult{
		Reasoner: t.Name(), Confidence: s.Confidence,
		Actions: actions, Findings: findings,
	}, nil
}

// CausalReasoner performs graph-native cause-and-effect analysis.
type CausalReasoner struct{}

// NewCausalReasoner returns the causal reasoner.
func NewCausalReasoner() *CausalReasoner { return &CausalReasoner{} }

func (c *CausalReasoner) Name() string  { return "causal" }
func (c *CausalReasoner) Priority() int { return 90 }
func (c *CausalReasoner) Supports(s *model.Session) bool {
	return s != nil && s.Graph != nil && len(s.Graph.Nodes) > 0
}

func (c *CausalReasoner) Analyze(ctx context.Context, inv *reasoning.Investigation) (*model.ReasoningResult, error) {
	_ = ctx
	g := inv.Graph
	if g == nil {
		return &model.ReasoningResult{Reasoner: c.Name()}, nil
	}

	var actions []model.ReasoningAction
	var findings []model.Finding

	if inv.Session.Service != "" {
		sg, _ := g.Query(model.GraphQuery{Kind: model.QueryUpstream, Target: inv.Session.Service, Limit: 10})
		if sg != nil && len(sg.Nodes) > 1 {
			findings = append(findings, model.Finding{
				Type: "upstream_causes", Summary: fmt.Sprintf("Found %d upstream nodes for %s.", len(sg.Nodes), inv.Session.Service),
				Refs: nodeLabels(sg.Nodes),
			})
		}
		sgDown, _ := g.Query(model.GraphQuery{Kind: model.QueryBlastRadius, Target: inv.Session.Service, Limit: 15})
		if sgDown != nil && len(sgDown.Nodes) > 1 {
			findings = append(findings, model.Finding{
				Type: "downstream_impact", Summary: fmt.Sprintf("Blast radius spans %d nodes.", len(sgDown.Nodes)),
			})
		}
	}

	for _, h := range inv.Session.Hypotheses {
		if h.Status == model.StatusRefuted {
			continue
		}
		sg, _ := g.Query(model.GraphQuery{Kind: model.QueryStrongestPath, Target: "hyp:" + strings.TrimPrefix(h.ID, "hypothesis-"), Limit: 5})
		if sg == nil {
			sg, _ = g.Query(model.GraphQuery{Kind: model.QueryStrongestPath, Target: h.ID, Limit: 5})
		}
		if sg != nil && len(sg.Edges) > 0 {
			edge := sg.Edges[0]
			actions = append(actions, model.ReasoningAction{
				Type: model.ActionIncreaseHypothesisConfidence, Reasoner: c.Name(),
				HypothesisID: h.ID, Delta: edge.Confidence / 10,
				Reason: fmt.Sprintf("Strongest causal path via %s.", edge.Type),
			})
		}
	}

	for _, n := range g.Nodes() {
		if n.Properties == nil {
			continue
		}
		if bf, ok := n.Properties["branching_failures"].(int); ok && bf >= 2 {
			findings = append(findings, model.Finding{
				Type: "cascading_failure", Summary: fmt.Sprintf("Branching failure at %s (%d paths).", n.Label, bf),
				Confidence: 75,
			})
		}
	}

	for _, h := range inv.Session.Hypotheses {
		sg, _ := g.Query(model.GraphQuery{Kind: model.QueryContradictions, Target: h.ID})
		if sg != nil && len(sg.Edges) > 0 {
			findings = append(findings, model.Finding{
				Type: "contradictory_path", Summary: fmt.Sprintf("Contradictory evidence path for %s.", h.ID),
				Refs: []string{h.ID},
			})
		}
	}

	return &model.ReasoningResult{
		Reasoner: c.Name(), Confidence: inv.Session.Confidence,
		Actions: actions, Findings: findings,
	}, nil
}

// HypothesisReasoner generates and scores competing hypotheses.
type HypothesisReasoner struct {
	engines engine.RuntimeEngines
}

// NewHypothesisReasoner returns the hypothesis reasoner.
func NewHypothesisReasoner(eng engine.RuntimeEngines) *HypothesisReasoner {
	return &HypothesisReasoner{engines: eng}
}

func (h *HypothesisReasoner) Name() string                  { return "hypothesis" }
func (h *HypothesisReasoner) Priority() int               { return 80 }
func (h *HypothesisReasoner) Supports(_ *model.Session) bool { return true }

func (h *HypothesisReasoner) Analyze(ctx context.Context, inv *reasoning.Investigation) (*model.ReasoningResult, error) {
	_ = ctx
	s := inv.Session
	sig := asSignals(inv.Signals)
	e := h.engines

	contradictions := e.Contradiction.Detect(s, sig)
	hypotheses := e.Hypothesis.Generate(s, sig, contradictions)
	required := s.RequiredEvidence
	if len(required) == 0 {
		required = e.Planner.Plan(s, sig)
	}
	progress := s.Progress
	if progress == 0 {
		progress = engine.Progress(s, required)
	}

	confidence := e.Confidence.Score(s, sig, hypotheses, contradictions, progress)
	coverage := e.Coverage.Compute(s, required)
	strategy := e.Strategy.NextSteps(s, sig, hypotheses, required)
	importance := e.Importance.Score(s, sig, hypotheses)
	sufficiency := e.Sufficiency.Evaluate(s, sig, coverage)
	invProgress := engine.ComputeInvestigationProgress(s, coverage, sufficiency)
	breakdown := engine.ComputeConfidenceBreakdown(s, sig, hypotheses, contradictions, progress)

	graphView := s.Graph
	if built := e.Graph.Build(s, sig); built != nil {
		engine.NewInferenceEngine().Apply(built, sig)
		engine.NewCausalEngine().Analyze(built)
		graphView = built.View()
	}
	blast := e.Blast.Estimate(s, sig)

	actions := []model.ReasoningAction{
		{Type: model.ActionReplaceHypotheses, Reasoner: h.Name(), Hypotheses: hypotheses},
		{Type: model.ActionSetConfidence, Reasoner: h.Name(), Confidence: confidence},
		{Type: model.ActionSetCoverage, Reasoner: h.Name(), Coverage: coverage},
		{Type: model.ActionSetStrategy, Reasoner: h.Name(), Strategy: strategy},
		{Type: model.ActionSetSufficiency, Reasoner: h.Name(), Sufficiency: &sufficiency},
		{Type: model.ActionSetInvestigationProgress, Reasoner: h.Name(), InvestigationProgress: &invProgress},
		{Type: model.ActionSetConfidenceBreakdown, Reasoner: h.Name(), ConfidenceBreakdown: &breakdown},
		{Type: model.ActionSetEvidenceImportance, Reasoner: h.Name(), EvidenceImportance: importance},
		{Type: model.ActionUpdateGraph, Reasoner: h.Name(), Graph: graphView},
		{Type: model.ActionSetBlastRadius, Reasoner: h.Name(), BlastRadius: blast},
	}

	if len(hypotheses) > 0 && hypotheses[0].Confidence >= 40 {
		actions = append(actions, model.ReasoningAction{
			Type: model.ActionPromoteHypothesis, Reasoner: h.Name(),
			HypothesisID: hypotheses[0].ID,
			Reason:       "Leading hypothesis by normalized confidence.",
		})
	}

	return &model.ReasoningResult{
		Reasoner: h.Name(), Confidence: confidence, Actions: actions,
		Findings: []model.Finding{{
			Type: "hypothesis_field", Summary: fmt.Sprintf("%d competing hypotheses.", len(hypotheses)),
			Confidence: confidence,
		}},
	}, nil
}

// ConsistencyReasoner detects contradictions and impossible states.
type ConsistencyReasoner struct {
	engines engine.RuntimeEngines
}

// NewConsistencyReasoner returns the consistency reasoner.
func NewConsistencyReasoner(eng engine.RuntimeEngines) *ConsistencyReasoner {
	return &ConsistencyReasoner{engines: eng}
}

func (c *ConsistencyReasoner) Name() string                  { return "consistency" }
func (c *ConsistencyReasoner) Priority() int               { return 70 }
func (c *ConsistencyReasoner) Supports(_ *model.Session) bool { return true }

func (c *ConsistencyReasoner) Analyze(ctx context.Context, inv *reasoning.Investigation) (*model.ReasoningResult, error) {
	_ = ctx
	sig := asSignals(inv.Signals)
	contradictions := c.engines.Contradiction.Detect(inv.Session, sig)
	var actions []model.ReasoningAction
	actions = append(actions, model.ReasoningAction{
		Type: model.ActionSetContradictions, Reasoner: c.Name(),
		Contradictions: contradictions,
	})
	for _, cx := range contradictions {
		if cx.ID == "contradiction-deploy-after-incident" {
			actions = append(actions, model.ReasoningAction{
				Type: model.ActionDecreaseHypothesisConfidence, Reasoner: c.Name(),
				HypothesisID: "hypothesis-deployment-caused", Delta: 15,
				Reason: cx.Description,
			})
		}
	}

	if inv.Graph != nil {
		report := graph.NewConsistencyChecker().Check(inv.Graph)
		for _, issue := range report.Issues {
			if issue.Severity == "high" {
				actions = append(actions, model.ReasoningAction{
					Type: model.ActionCreateRecommendation, Reasoner: c.Name(),
					Recommendation: fmt.Sprintf("Graph consistency: %s", issue.Message),
				})
			}
		}
	}

	findings := make([]model.Finding, 0, len(contradictions))
	for _, cx := range contradictions {
		findings = append(findings, model.Finding{
			Type: "contradiction", Summary: cx.Description, Confidence: 90, Refs: cx.EvidenceRefs,
		})
	}

	return &model.ReasoningResult{
		Reasoner: c.Name(), Confidence: inv.Session.Confidence,
		Actions: actions, Findings: findings,
	}, nil
}

func nodeLabels(nodes []*model.GraphNode) []string {
	out := make([]string, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, n.Label)
	}
	return out
}
