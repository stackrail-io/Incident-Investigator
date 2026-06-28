package model

import "time"

// NodeType classifies investigation graph nodes.
type NodeType string

// EdgeType classifies investigation graph edges.
type EdgeType string

// Investigation node types — every entity in an investigation is a node.
const (
	NodeTypeInvestigation    NodeType = "investigation"
	NodeTypeQuestion         NodeType = "question"
	NodeTypeEvidenceRequest  NodeType = "evidence_request"
	NodeTypeHypothesis       NodeType = "hypothesis"
	NodeTypeApplication      NodeType = "application"
	NodeTypePod              NodeType = "pod"
	NodeTypeConfiguration    NodeType = "configuration"
	NodeTypeTrace            NodeType = "trace"
	NodeTypeDatabase         NodeType = "database"
	NodeTypeAPI              NodeType = "api"
	NodeTypeTimelineEvent    NodeType = "timeline_event"
	NodeTypeIncident         NodeType = "incident"
	NodeTypeConclusion       NodeType = "conclusion"
	NodeTypeRecommendation   NodeType = "recommendation"
	NodeTypeRegion           NodeType = "region"
	NodeTypeCluster          NodeType = "cluster"
	NodeTypeCustomerImpact   NodeType = "customer_impact"
	NodeTypeCustom           NodeType = "custom"

	// Legacy aliases retained for compatibility.
	NodeEvidence   NodeType = "evidence"
	NodeService    NodeType = "service"
	NodeDeployment NodeType = "deployment"
	NodeAlert      NodeType = "alert"
	NodeMetric     NodeType = "metric"
)

// Investigation edge types.
const (
	EdgeSupports       EdgeType = "supports"
	EdgeContradicts    EdgeType = "contradicts"
	EdgeCauses         EdgeType = "causes"
	EdgeTriggered      EdgeType = "triggered"
	EdgeDependsOn      EdgeType = "depends_on"
	EdgeOccurredBefore EdgeType = "occurred_before"
	EdgeOccurredAfter  EdgeType = "occurred_after"
	EdgeGenerated      EdgeType = "generated"
	EdgeResolves       EdgeType = "resolves"
	EdgeRequests       EdgeType = "requests"
	EdgeBelongsTo      EdgeType = "belongs_to"
	EdgeObservedOn     EdgeType = "observed_on"
	EdgeRecoveredBy     EdgeType = "recovered_by"
	EdgeCorrelatesWith EdgeType = "correlates_with"

	// Legacy aliases.
	EdgeLikelyCaused   EdgeType = "likely_caused"
	EdgeRecoveredAfter EdgeType = "recovered_after"
)

// GraphNode is a vertex in the investigation graph.
type GraphNode struct {
	ID         string         `json:"id"`
	Type       NodeType       `json:"type"`
	Label      string         `json:"label"`
	RefID      string         `json:"ref_id,omitempty"`
	Properties map[string]any `json:"properties,omitempty"`
}

// GraphEdge is a directed, confidence-weighted relationship.
type GraphEdge struct {
	ID           string    `json:"id"`
	From         string    `json:"from"`
	To           string    `json:"to"`
	Type         EdgeType  `json:"type"`
	Confidence   float64   `json:"confidence"`
	Weight       float64   `json:"weight,omitempty"`
	Timestamp    time.Time `json:"timestamp,omitempty"`
	Reason       string    `json:"reason,omitempty"`
	EvidenceRefs []string  `json:"evidence_refs,omitempty"`
	Inferred     bool      `json:"inferred,omitempty"`
}

// Subgraph is a filtered extract of the investigation graph.
type Subgraph struct {
	Name  string       `json:"name,omitempty"`
	Nodes []*GraphNode `json:"nodes"`
	Edges []*GraphEdge `json:"edges"`
}

// GraphView is the serializable projection of an investigation graph.
type GraphView struct {
	Nodes []*GraphNode `json:"nodes"`
	Edges []*GraphEdge `json:"edges"`
}

// NewEmptyGraphView returns an empty graph snapshot.
func NewEmptyGraphView() *GraphView {
	return &GraphView{
		Nodes: []*GraphNode{},
		Edges: []*GraphEdge{},
	}
}

// PathExplanation is a causal path with evidence references per hop.
type PathExplanation struct {
	From       string       `json:"from"`
	To         string       `json:"to"`
	Reason     string       `json:"reason"`
	Confidence float64      `json:"confidence"`
	Nodes      []*GraphNode `json:"nodes"`
	Edges      []*GraphEdge `json:"edges"`
}

// GraphConsistencyReport lists graph integrity issues.
type GraphConsistencyReport struct {
	Issues []GraphIssue `json:"issues"`
}

// GraphIssue describes one consistency problem.
type GraphIssue struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
}

// GraphQueryKind identifies a built-in graph query.
type GraphQueryKind string

const (
	QueryUpstream           GraphQueryKind = "upstream"
	QueryDownstream         GraphQueryKind = "downstream"
	QuerySupportingEvidence GraphQueryKind = "supporting_evidence"
	QueryContradictions     GraphQueryKind = "contradictions"
	QueryUnansweredQuestions GraphQueryKind = "unanswered_questions"
	QueryServiceEvidence    GraphQueryKind = "service_evidence"
	QueryBlastRadius        GraphQueryKind = "blast_radius"
	QueryShortestCausalPath GraphQueryKind = "shortest_causal_path"
	QueryStrongestPath      GraphQueryKind = "strongest_path"
)

// GraphQuery is a graph query request.
type GraphQuery struct {
	Kind   GraphQueryKind `json:"kind"`
	Target string         `json:"target"`
	Limit  int            `json:"limit,omitempty"`
}
