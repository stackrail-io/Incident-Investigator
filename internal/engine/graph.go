package engine

import (
	"fmt"

	"github.com/stackrail/incident-investigator/internal/model"
)

// GraphBuilder (re)builds the evidence graph from the current session state.
type GraphBuilder interface {
	Build(s *model.Session, sig Signals) *model.Graph
}

// HeuristicGraphBuilder constructs nodes for evidence/hypotheses/entities and
// connects them with confidence-weighted edges.
type HeuristicGraphBuilder struct{}

// NewHeuristicGraphBuilder returns the default builder.
func NewHeuristicGraphBuilder() *HeuristicGraphBuilder { return &HeuristicGraphBuilder{} }

func evidenceNodeID(id string) string   { return "ev:" + id }
func entityNodeID(name string) string   { return "svc:" + name }
func hypothesisNodeID(id string) string { return "hyp:" + id }

func nodeTypeForEvidence(c model.Category) model.NodeType {
	switch c {
	case model.CategoryDeploymentEvents:
		return model.NodeDeployment
	case model.CategoryAlertEvents:
		return model.NodeAlert
	case model.CategoryMetrics:
		return model.NodeMetric
	default:
		return model.NodeEvidence
	}
}

// Build implements GraphBuilder.
func (b *HeuristicGraphBuilder) Build(s *model.Session, sig Signals) *model.Graph {
	g := model.NewGraph()
	ordered := sortedByTime(s.Evidence)

	// Entity (service) nodes.
	for name := range sig.Entities {
		g.AddNode(&model.Node{ID: entityNodeID(name), Type: model.NodeService, Label: name, RefID: name})
	}
	if s.Service != "" {
		g.AddNode(&model.Node{ID: entityNodeID(s.Service), Type: model.NodeService, Label: s.Service, RefID: s.Service})
	}

	// Evidence nodes + entity association edges.
	for _, e := range ordered {
		g.AddNode(&model.Node{
			ID:    evidenceNodeID(e.ID),
			Type:  nodeTypeForEvidence(e.Category),
			Label: e.Summary,
			RefID: e.ID,
		})
		if e.Entity != "" {
			g.AddEdge(&model.Edge{
				From:         evidenceNodeID(e.ID),
				To:           entityNodeID(e.Entity),
				Type:         model.EdgeSupports,
				Confidence:   100,
				Reason:       "Evidence concerns this entity.",
				EvidenceRefs: []string{e.ID},
			})
		}
	}

	// Temporal chain: occurred_before between consecutive observations.
	for i := 1; i < len(ordered); i++ {
		prev, cur := ordered[i-1], ordered[i]
		g.AddEdge(&model.Edge{
			From:         evidenceNodeID(prev.ID),
			To:           evidenceNodeID(cur.ID),
			Type:         model.EdgeOccurredBefore,
			Confidence:   100,
			Reason:       fmt.Sprintf("%s precedes %s", prev.Timestamp.Format("15:04:05"), cur.Timestamp.Format("15:04:05")),
			EvidenceRefs: []string{prev.ID, cur.ID},
		})
	}

	// Causal edge: a deployment before the incident likely caused the onset.
	if sig.DeployBeforeIncident && sig.FirstDeployment != nil && sig.IncidentOnset != nil {
		g.AddEdge(&model.Edge{
			From:         evidenceNodeID(sig.FirstDeployment.ID),
			To:           evidenceNodeID(sig.IncidentOnset.ID),
			Type:         model.EdgeLikelyCaused,
			Confidence:   70,
			Reason:       "Deployment immediately precedes the first symptom.",
			EvidenceRefs: []string{sig.FirstDeployment.ID, sig.IncidentOnset.ID},
		})
	}

	// Recovery edge.
	if sig.Recovery != nil && sig.IncidentOnset != nil {
		g.AddEdge(&model.Edge{
			From:         evidenceNodeID(sig.IncidentOnset.ID),
			To:           evidenceNodeID(sig.Recovery.ID),
			Type:         model.EdgeRecoveredAfter,
			Confidence:   80,
			Reason:       "Service recovered after this remediation.",
			EvidenceRefs: []string{sig.IncidentOnset.ID, sig.Recovery.ID},
		})
	}

	// Hypothesis nodes + supports/contradicts edges.
	for _, hyp := range s.Hypotheses {
		g.AddNode(&model.Node{
			ID:    hypothesisNodeID(hyp.ID),
			Type:  model.NodeHypothesis,
			Label: hyp.Statement,
			RefID: hyp.ID,
		})
		for _, ref := range hyp.SupportingEvidence {
			if !g.HasNode(evidenceNodeID(ref)) {
				continue
			}
			g.AddEdge(&model.Edge{
				From:         evidenceNodeID(ref),
				To:           hypothesisNodeID(hyp.ID),
				Type:         model.EdgeSupports,
				Confidence:   hyp.Confidence,
				Reason:       "Evidence supports this hypothesis.",
				EvidenceRefs: []string{ref},
			})
		}
		for _, ref := range hyp.ConflictingEvidence {
			if !g.HasNode(evidenceNodeID(ref)) {
				continue
			}
			g.AddEdge(&model.Edge{
				From:         evidenceNodeID(ref),
				To:           hypothesisNodeID(hyp.ID),
				Type:         model.EdgeContradicts,
				Confidence:   60,
				Reason:       "Evidence conflicts with this hypothesis.",
				EvidenceRefs: []string{ref},
			})
		}
	}

	return g
}
