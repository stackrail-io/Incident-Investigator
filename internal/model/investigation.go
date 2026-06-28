package model

import "time"

// InvestigationGoal scopes what the investigation optimizes for.
type InvestigationGoal string

const (
	GoalRootCause              InvestigationGoal = "root_cause"
	GoalTimeline               InvestigationGoal = "timeline"
	GoalBlastRadius            InvestigationGoal = "blast_radius"
	GoalDeploymentVerification InvestigationGoal = "deployment_verification"
	GoalPerformanceRegression  InvestigationGoal = "performance_regression"
	GoalAvailability           InvestigationGoal = "availability"
	GoalCustom                 InvestigationGoal = "custom"
)

// DefaultGoal is used when the client does not specify a goal.
const DefaultGoal = GoalRootCause

// Valid reports whether g is a supported investigation goal.
func (g InvestigationGoal) Valid() bool {
	switch g {
	case GoalRootCause, GoalTimeline, GoalBlastRadius, GoalDeploymentVerification,
		GoalPerformanceRegression, GoalAvailability, GoalCustom:
		return true
	default:
		return false
	}
}

// InvestigationState is the explicit lifecycle state of an investigation.
type InvestigationState string

const (
	StateStarted            InvestigationState = "started"
	StateCollectingEvidence InvestigationState = "collecting_evidence"
	StateReasoning          InvestigationState = "reasoning"
	StateWaitingForEvidence InvestigationState = "waiting_for_evidence"
	StateHighConfidence     InvestigationState = "high_confidence"
	StateCompleted          InvestigationState = "completed"
	StateFailed             InvestigationState = "failed"
)

// LegacyStatus maps the explicit state machine to the pre-v0.2 status strings
// so existing MCP clients keep working.
func (st InvestigationState) LegacyStatus() Status {
	switch st {
	case StateHighConfidence:
		return StatusReady
	case StateCompleted:
		return StatusCompleted
	default:
		return StatusCollecting
	}
}

// BlockingQuestion is an unanswered question that prevents a trustworthy conclusion.
type BlockingQuestion struct {
	ID       string `json:"id"`
	Question string `json:"question"`
	Priority Priority `json:"priority"`
	Reason   string `json:"reason,omitempty"`
}

// EvidenceRequirement describes evidence still needed to answer blocking questions.
type EvidenceRequirement struct {
	Category Category `json:"category"`
	Priority Priority `json:"priority"`
	Reason   string   `json:"reason"`
}

// SufficiencyReport is the central decision on whether the investigation can answer.
type SufficiencyReport struct {
	CanAnswer         bool                  `json:"can_answer"`
	OverallConfidence float64               `json:"overall_confidence"`
	Coverage          float64               `json:"coverage"`
	BlockingQuestions []BlockingQuestion    `json:"blocking_questions"`
	MissingEvidence   []EvidenceRequirement `json:"missing_evidence"`
	Reason            string                `json:"reason"`
}

// NextStep is the highest-value evidence the strategy engine recommends next.
type NextStep struct {
	Priority               int      `json:"priority"`
	Category               Category `json:"category"`
	Reason                 string   `json:"reason"`
	ExpectedConfidenceGain float64  `json:"expected_confidence_gain"`
	BlocksHypothesis       []string `json:"blocks_hypothesis,omitempty"`
}

// ReasoningTrace records one step in how confidence evolved.
type ReasoningTrace struct {
	ObservationID        string    `json:"observation_id"`
	HypothesisID         string    `json:"hypothesis_id,omitempty"`
	Reason               string    `json:"reason"`
	ConfidenceChange     float64   `json:"confidence_change"`
	SupportingEvidence   []string  `json:"supporting_evidence,omitempty"`
	ContradictingEvidence []string `json:"contradicting_evidence,omitempty"`
	Timestamp            time.Time `json:"timestamp"`
}

// JournalEntry is one append-only record in the investigation journal.
type JournalEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Event     string    `json:"event"`
	Detail    string    `json:"detail,omitempty"`
	Confidence float64  `json:"confidence,omitempty"`
}

// CategoryCoverage reports how well a category is represented.
type CategoryCoverage struct {
	Category Category `json:"category"`
	Percent  float64  `json:"percent"`
	Count    int      `json:"count"`
}

// CoverageReport summarizes evidence coverage across categories.
type CoverageReport struct {
	Categories []CategoryCoverage `json:"categories"`
	Overall    float64            `json:"overall"`
}

// EvidenceImportance scores how much an evidence item contributed.
type EvidenceImportance struct {
	EvidenceID string  `json:"evidence_id"`
	Category   Category `json:"category"`
	Summary    string  `json:"summary"`
	Score      float64 `json:"score"`
}

// InvestigationProgress is the v0.2 progress model.
type InvestigationProgress struct {
	PercentComplete    float64 `json:"percent_complete"`
	Confidence         float64 `json:"confidence"`
	Coverage           float64 `json:"coverage"`
	RemainingQuestions int     `json:"remaining_questions"`
}

// ConfidenceBreakdown explains how overall confidence was derived.
type ConfidenceBreakdown struct {
	LeadingHypothesis   float64 `json:"leading_hypothesis"`
	Separation          float64 `json:"separation"`
	Corroboration       float64 `json:"corroboration"`
	Temporal            float64 `json:"temporal"`
	CoverageFactor      float64 `json:"coverage_factor"`
	ContradictionPenalty float64 `json:"contradiction_penalty"`
	Overall             float64 `json:"overall"`
}

// ReasoningMetrics exposes operational counters for an investigation.
type ReasoningMetrics struct {
	HypothesisCount        int     `json:"hypothesis_count"`
	RejectedHypotheses     int     `json:"rejected_hypotheses"`
	ContradictionCount     int     `json:"contradiction_count"`
	Coverage               float64 `json:"coverage"`
	EvidenceCount          int     `json:"evidence_count"`
	ReasoningIterations    int     `json:"reasoning_iterations"`
	PlannerIterations      int     `json:"planner_iterations"`
	AverageConfidenceGain  float64 `json:"average_confidence_gain"`
}
