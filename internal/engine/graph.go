package engine

import (
	"fmt"
	"time"

	"github.com/stackrail/incident-investigator/internal/graph"
	"github.com/stackrail/incident-investigator/internal/model"
)

// GraphBuilder (re)builds the investigation graph from the current session state.
type GraphBuilder interface {
	Build(s *model.Session, sig Signals) *graph.InvestigationGraph
}

// HeuristicGraphBuilder constructs the full investigation graph.
type HeuristicGraphBuilder struct{}

// NewHeuristicGraphBuilder returns the default builder.
func NewHeuristicGraphBuilder() *HeuristicGraphBuilder { return &HeuristicGraphBuilder{} }

func evidenceNodeID(id string) string   { return "ev:" + id }
func entityNodeID(name string) string   { return "svc:" + name }
func hypothesisNodeID(id string) string { return "hyp:" + id }
func questionNodeID(id string) string   { return "q:" + id }

func nodeTypeForEvidence(c model.Category) model.NodeType {
	switch c {
	case model.CategoryDeploymentEvents:
		return model.NodeDeployment
	case model.CategoryAlertEvents:
		return model.NodeAlert
	case model.CategoryMetrics:
		return model.NodeMetric
	case model.CategoryConfigurationChanges:
		return model.NodeTypeConfiguration
	case model.CategoryDatabaseEvents:
		return model.NodeTypeDatabase
	case model.CategoryTraceEvents:
		return model.NodeTypeTrace
	default:
		return model.NodeEvidence
	}
}

// Build implements GraphBuilder.
func (b *HeuristicGraphBuilder) Build(s *model.Session, sig Signals) *graph.InvestigationGraph {
	g := graph.NewInvestigationGraph()
	g.AddNode(&model.GraphNode{
		ID: "inv:" + s.ID, Type: model.NodeTypeInvestigation, Label: s.Question, RefID: s.ID,
		Properties: map[string]any{"goal": string(s.Goal), "status": string(s.Status)},
	})

	if s.Service != "" {
		g.AddNode(&model.GraphNode{ID: entityNodeID(s.Service), Type: model.NodeService, Label: s.Service, RefID: s.Service})
		_ = g.AddEdge(&model.GraphEdge{
			From: "inv:" + s.ID, To: entityNodeID(s.Service),
			Type: model.EdgeBelongsTo, Confidence: 100, Reason: "Investigation targets this service.",
		})
	}

	for name := range sig.Entities {
		g.AddNode(&model.GraphNode{ID: entityNodeID(name), Type: model.NodeService, Label: name, RefID: name})
	}

	ordered := sortedByTime(s.Evidence)
	for _, e := range ordered {
		g.AddNode(&model.GraphNode{
			ID: evidenceNodeID(e.ID), Type: nodeTypeForEvidence(e.Category),
			Label: e.Summary, RefID: e.ID,
			Properties: map[string]any{
				"category": string(e.Category), "timestamp": e.Timestamp.Format(time.RFC3339),
			},
		})
		_ = g.AddEdge(&model.GraphEdge{
			From: "inv:" + s.ID, To: evidenceNodeID(e.ID),
			Type: model.EdgeGenerated, Confidence: 100, EvidenceRefs: []string{e.ID},
		})
		if e.Entity != "" {
			_ = g.AddEdge(&model.GraphEdge{
				From: evidenceNodeID(e.ID), To: entityNodeID(e.Entity),
				Type: model.EdgeObservedOn, Confidence: 100, EvidenceRefs: []string{e.ID},
			})
		}
	}

	for i := 1; i < len(ordered); i++ {
		prev, cur := ordered[i-1], ordered[i]
		_ = g.AddEdge(&model.GraphEdge{
			From: evidenceNodeID(prev.ID), To: evidenceNodeID(cur.ID),
			Type: model.EdgeOccurredBefore, Confidence: 100, Weight: 1, Timestamp: cur.Timestamp,
			Reason: fmt.Sprintf("%s precedes %s", prev.Timestamp.Format("15:04:05"), cur.Timestamp.Format("15:04:05")),
			EvidenceRefs: []string{prev.ID, cur.ID},
		})
	}

	if sig.DeployBeforeIncident && sig.FirstDeployment != nil && sig.IncidentOnset != nil {
		_ = g.AddEdge(&model.GraphEdge{
			From: evidenceNodeID(sig.FirstDeployment.ID), To: evidenceNodeID(sig.IncidentOnset.ID),
			Type: model.EdgeCauses, Confidence: 70,
			Reason: "Deployment immediately precedes the first symptom.",
			EvidenceRefs: []string{sig.FirstDeployment.ID, sig.IncidentOnset.ID},
		})
	}
	if sig.Recovery != nil && sig.IncidentOnset != nil {
		_ = g.AddEdge(&model.GraphEdge{
			From: evidenceNodeID(sig.Recovery.ID), To: evidenceNodeID(sig.IncidentOnset.ID),
			Type: model.EdgeRecoveredBy, Confidence: 80,
			Reason: "Service recovered after this remediation.",
			EvidenceRefs: []string{sig.Recovery.ID, sig.IncidentOnset.ID},
		})
	}

	for _, hyp := range s.Hypotheses {
		g.AddNode(&model.GraphNode{
			ID: hypothesisNodeID(hyp.ID), Type: model.NodeTypeHypothesis,
			Label: hyp.Statement, RefID: hyp.ID,
			Properties: map[string]any{"confidence": hyp.Confidence, "status": string(hyp.Status)},
		})
		for _, ref := range hyp.SupportingEvidence {
			if g.HasNode(evidenceNodeID(ref)) {
				_ = g.AddEdge(&model.GraphEdge{
					From: evidenceNodeID(ref), To: hypothesisNodeID(hyp.ID),
					Type: model.EdgeSupports, Confidence: hyp.Confidence,
					EvidenceRefs: []string{ref},
				})
			}
		}
		for _, ref := range hyp.ConflictingEvidence {
			if g.HasNode(evidenceNodeID(ref)) {
				_ = g.AddEdge(&model.GraphEdge{
					From: evidenceNodeID(ref), To: hypothesisNodeID(hyp.ID),
					Type: model.EdgeContradicts, Confidence: 60, EvidenceRefs: []string{ref},
				})
			}
		}
	}

	if s.Plan != nil {
		addProtocolNodes(g, s)
	}

	NewInferenceEngine().Apply(g, sig)
	NewCausalEngine().Analyze(g)
	return g
}

func addProtocolNodes(g *graph.InvestigationGraph, s *model.Session) {
	for _, q := range s.Plan.Questions {
		g.AddNode(&model.GraphNode{
			ID: questionNodeID(q.ID), Type: model.NodeTypeQuestion, Label: q.Title, RefID: q.ID,
			Properties: map[string]any{"status": string(q.Status), "priority": q.Priority},
		})
		_ = g.AddEdge(&model.GraphEdge{
			From: "inv:" + s.ID, To: questionNodeID(q.ID),
			Type: model.EdgeGenerated, Confidence: 100,
		})
		for _, dep := range q.DependsOn {
			_ = g.AddEdge(&model.GraphEdge{
				From: questionNodeID(dep), To: questionNodeID(q.ID),
				Type: model.EdgeDependsOn, Confidence: 100,
			})
		}
	}
	for _, req := range s.Plan.EvidenceRequests {
		g.AddNode(&model.GraphNode{
			ID: "req:" + req.ID, Type: model.NodeTypeEvidenceRequest,
			Label: req.Reason, RefID: req.ID,
		})
		_ = g.AddEdge(&model.GraphEdge{
			From: questionNodeID(req.QuestionID), To: "req:" + req.ID,
			Type: model.EdgeRequests, Confidence: req.ExpectedConfidenceGain, Reason: req.Reason,
		})
	}
}
