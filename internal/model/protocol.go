package model

import "time"

// InvestigationStage is the current phase of the investigation protocol.
type InvestigationStage string

const (
	StagePlanning            InvestigationStage = "planning"
	StageQuestionGeneration  InvestigationStage = "question_generation"
	StageEvidenceCollection  InvestigationStage = "evidence_collection"
	StageQuestionResolution  InvestigationStage = "question_resolution"
	StageHypothesisEvaluation InvestigationStage = "hypothesis_evaluation"
	StageNeedMoreEvidence    InvestigationStage = "need_more_evidence"
	StageCompleted           InvestigationStage = "completed"
)

// QuestionStatus tracks where a protocol question stands.
type QuestionStatus string

const (
	QuestionUnknown            QuestionStatus = "unknown"
	QuestionWaitingForEvidence QuestionStatus = "waiting_for_evidence"
	QuestionPartiallyAnswered  QuestionStatus = "partially_answered"
	QuestionAnswered           QuestionStatus = "answered"
	QuestionRejected           QuestionStatus = "rejected"
)

// ResolutionStatus is the outcome of resolving a question.
type ResolutionStatus string

const (
	ResolutionConfirmed           ResolutionStatus = "confirmed"
	ResolutionRejected              ResolutionStatus = "rejected"
	ResolutionInsufficientEvidence  ResolutionStatus = "insufficient_evidence"
	ResolutionUnknown               ResolutionStatus = "unknown"
)

// RequestStatus tracks a protocol evidence request lifecycle.
type RequestStatus string

const (
	RequestOpen      RequestStatus = "open"
	RequestFulfilled RequestStatus = "fulfilled"
)

// Question is a protocol question that drives evidence collection.
type Question struct {
	ID                   string             `json:"id"`
	Title                string             `json:"title"`
	Description          string             `json:"description,omitempty"`
	Priority             int                `json:"priority"`
	Status               QuestionStatus     `json:"status"`
	Confidence           float64            `json:"confidence"`
	RequiredEvidence     []Category         `json:"required_evidence"`
	SupportingEvidence   []string           `json:"supporting_evidence,omitempty"`
	ContradictingEvidence []string          `json:"contradicting_evidence,omitempty"`
	Resolution           *QuestionResolution `json:"resolution,omitempty"`
	DependsOn            []string           `json:"depends_on,omitempty"`
}

// QuestionResolution records how a question was resolved.
type QuestionResolution struct {
	QuestionID            string           `json:"question_id"`
	Status                ResolutionStatus `json:"status"`
	Confidence            float64          `json:"confidence"`
	Reason                string           `json:"reason"`
	SupportingEvidence    []string         `json:"supporting_evidence,omitempty"`
	ContradictingEvidence []string         `json:"contradicting_evidence,omitempty"`
	ResolvedAt            time.Time        `json:"resolved_at,omitempty"`
}

// ProtocolEvidenceRequest is evidence requested to answer a specific question.
type ProtocolEvidenceRequest struct {
	ID                     string        `json:"id"`
	QuestionID             string        `json:"question_id"`
	Categories             []Category    `json:"categories"`
	Priority               Priority      `json:"priority"`
	Reason                 string        `json:"reason"`
	ExpectedConfidenceGain float64       `json:"expected_confidence_gain"`
	Status                 RequestStatus `json:"status"`
}

// InvestigationPlan is the brain of an investigation.
type InvestigationPlan struct {
	ID                 string                    `json:"id"`
	Goal               InvestigationGoal         `json:"goal"`
	CurrentStage       InvestigationStage        `json:"current_stage"`
	Questions          []Question                `json:"questions"`
	ActiveHypotheses   []string                  `json:"active_hypotheses"`
	CompletedQuestions []string                  `json:"completed_questions"`
	OpenQuestions      []string                  `json:"open_questions"`
	EvidenceRequests   []ProtocolEvidenceRequest `json:"evidence_requests"`
	Confidence         float64                   `json:"confidence"`
	ResolutionHistory  []QuestionResolution      `json:"resolution_history"`
	PlaybookID         string                    `json:"playbook_id,omitempty"`
}

// QuestionGraphNode is a node in the question dependency graph.
type QuestionGraphNode struct {
	QuestionID string         `json:"question_id"`
	Title      string         `json:"title"`
	Status     QuestionStatus `json:"status"`
}

// QuestionGraphEdge links dependent questions.
type QuestionGraphEdge struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Relation string `json:"relation"`
}

// QuestionGraph models question dependencies as a protocol view.
type QuestionGraph struct {
	Nodes []QuestionGraphNode `json:"nodes"`
	Edges []QuestionGraphEdge `json:"edges"`
}

// ProtocolMetrics exposes investigation-protocol counters.
type ProtocolMetrics struct {
	TotalQuestions              int     `json:"total_questions"`
	ResolvedQuestions           int     `json:"resolved_questions"`
	PendingQuestions            int     `json:"pending_questions"`
	RejectedQuestions           int     `json:"rejected_questions"`
	AverageResolutionConfidence float64 `json:"average_resolution_confidence"`
	EvidenceRequestsCreated     int     `json:"evidence_requests_created"`
	EvidenceRequestsCompleted   int     `json:"evidence_requests_completed"`
	AverageConfidenceGain       float64 `json:"average_confidence_gain"`
	PlannerIterations           int     `json:"planner_iterations"`
}

// ProtocolTurn captures changes from the latest evidence submission.
type ProtocolTurn struct {
	ResolvedQuestions []QuestionResolution `json:"resolved_questions"`
	NewQuestions      []Question           `json:"new_questions"`
	ConfidenceDelta   float64              `json:"confidence_delta"`
}
