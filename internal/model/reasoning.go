package model

import "time"

// ActionType identifies a declarative reasoning action.
type ActionType string

const (
	ActionIncreaseHypothesisConfidence ActionType = "IncreaseHypothesisConfidence"
	ActionDecreaseHypothesisConfidence ActionType = "DecreaseHypothesisConfidence"
	ActionCreateQuestion               ActionType = "CreateQuestion"
	ActionResolveQuestion              ActionType = "ResolveQuestion"
	ActionRejectQuestion               ActionType = "RejectQuestion"
	ActionCreateEvidenceRequest        ActionType = "CreateEvidenceRequest"
	ActionLinkGraphNodes               ActionType = "LinkGraphNodes"
	ActionCreateHypothesis             ActionType = "CreateHypothesis"
	ActionRejectHypothesis             ActionType = "RejectHypothesis"
	ActionPromoteHypothesis            ActionType = "PromoteHypothesis"
	ActionCreateRecommendation         ActionType = "CreateRecommendation"
	ActionMarkContradiction            ActionType = "MarkContradiction"
	ActionMarkInvestigationComplete    ActionType = "MarkInvestigationComplete"

	// Projection actions — runtime applies derived state from reasoner observations.
	ActionSetContradictions       ActionType = "SetContradictions"
	ActionReplaceHypotheses       ActionType = "ReplaceHypotheses"
	ActionSetTimeline             ActionType = "SetTimeline"
	ActionUpdateGraph             ActionType = "UpdateGraph"
	ActionSetBlastRadius          ActionType = "SetBlastRadius"
	ActionSetEvidencePlan         ActionType = "SetEvidencePlan"
	ActionSetCoverage             ActionType = "SetCoverage"
	ActionSetStrategy             ActionType = "SetStrategy"
	ActionSetSufficiency          ActionType = "SetSufficiency"
	ActionSetConfidence           ActionType = "SetConfidence"
	ActionSetProgress             ActionType = "SetProgress"
	ActionSetInvestigationProgress ActionType = "SetInvestigationProgress"
	ActionSetConfidenceBreakdown  ActionType = "SetConfidenceBreakdown"
	ActionSetEvidenceImportance   ActionType = "SetEvidenceImportance"
)

// ReasoningAction is a declarative change proposed by a reasoner.
// Reasoners emit actions; the runtime validates and applies them.
type ReasoningAction struct {
	Type     ActionType `json:"type"`
	Reasoner string     `json:"reasoner"`
	Reason   string     `json:"reason,omitempty"`

	HypothesisID string      `json:"hypothesis_id,omitempty"`
	Hypothesis   *Hypothesis `json:"hypothesis,omitempty"`
	Delta        float64     `json:"delta,omitempty"`

	QuestionID string    `json:"question_id,omitempty"`
	Question   *Question `json:"question,omitempty"`

	EvidenceRequest *ProtocolEvidenceRequest `json:"evidence_request,omitempty"`
	GraphEdge       *GraphEdge               `json:"graph_edge,omitempty"`
	Contradiction   *Contradiction           `json:"contradiction,omitempty"`
	Contradictions  []Contradiction          `json:"contradictions,omitempty"`
	Recommendation  string                   `json:"recommendation,omitempty"`

	// Projection payloads
	Hypotheses            []Hypothesis           `json:"hypotheses,omitempty"`
	Timeline              Timeline               `json:"timeline,omitempty"`
	Graph                 *GraphView             `json:"graph,omitempty"`
	BlastRadius           BlastRadius            `json:"blast_radius,omitempty"`
	RequiredEvidence      []EvidenceRequest      `json:"required_evidence,omitempty"`
	MissingEvidence       []EvidenceRequest      `json:"missing_evidence,omitempty"`
	Coverage              CoverageReport         `json:"coverage,omitempty"`
	Strategy              []NextStep             `json:"strategy,omitempty"`
	Sufficiency           *SufficiencyReport     `json:"sufficiency,omitempty"`
	Confidence            float64                `json:"confidence,omitempty"`
	Progress              float64                `json:"progress,omitempty"`
	InvestigationProgress *InvestigationProgress `json:"investigation_progress,omitempty"`
	ConfidenceBreakdown   *ConfidenceBreakdown   `json:"confidence_breakdown,omitempty"`
	EvidenceImportance    []EvidenceImportance   `json:"evidence_importance,omitempty"`
}

// ActionType returns the action discriminator (ReasoningAction interface compat).
func (a ReasoningAction) ActionType() string { return string(a.Type) }

// Finding is an observation returned alongside actions.
type Finding struct {
	Type       string  `json:"type"`
	Summary    string  `json:"summary"`
	Confidence float64 `json:"confidence,omitempty"`
	Reason     string  `json:"reason,omitempty"`
	Refs       []string `json:"refs,omitempty"`
}

// ReasonerMetrics captures per-reasoner operational counters.
type ReasonerMetrics struct {
	ExecutionTime        time.Duration `json:"execution_time"`
	QuestionsResolved    int           `json:"questions_resolved"`
	QuestionsCreated     int           `json:"questions_created"`
	HypothesesCreated    int           `json:"hypotheses_created"`
	HypothesesRejected   int           `json:"hypotheses_rejected"`
	AverageConfidenceDelta float64     `json:"average_confidence_delta"`
	ContradictionsFound  int           `json:"contradictions_found"`
	RecommendationsCreated int         `json:"recommendations_created"`
	GraphNodesCreated    int           `json:"graph_nodes_created"`
	GraphEdgesCreated    int           `json:"graph_edges_created"`
	ActionsProposed      int           `json:"actions_proposed"`
}

// ReasoningResult is the output of one reasoner pass.
type ReasoningResult struct {
	Reasoner   string          `json:"reasoner"`
	Confidence float64         `json:"confidence"`
	Actions    []ReasoningAction `json:"actions"`
	Findings   []Finding       `json:"findings"`
	Metrics    ReasonerMetrics `json:"metrics"`
}

// RejectedAction records an action the runtime refused to apply.
type RejectedAction struct {
	Action ReasoningAction `json:"action"`
	Reason string          `json:"reason"`
}

// ReasoningCycle is one full orchestrator iteration.
type ReasoningCycle struct {
	CycleID         string            `json:"cycle_id"`
	StartedAt       time.Time         `json:"started_at"`
	CompletedAt     time.Time         `json:"completed_at"`
	Reasoners       []ReasoningResult `json:"reasoners"`
	AppliedActions  []ReasoningAction `json:"applied_actions"`
	RejectedActions []RejectedAction  `json:"rejected_actions"`
	Duration        time.Duration     `json:"duration"`
	Strategy        string            `json:"strategy,omitempty"`
}

// OrchestrationStrategy selects how reasoners are composed.
type OrchestrationStrategy string

const (
	StrategySequential    OrchestrationStrategy = "sequential"
	StrategyParallel      OrchestrationStrategy = "parallel"
	StrategyWeightedVote  OrchestrationStrategy = "weighted_voting"
	StrategyPriority      OrchestrationStrategy = "priority"
	StrategyConsensus     OrchestrationStrategy = "consensus"
)
