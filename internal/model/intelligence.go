package model

import "time"

// InvestigationSnapshot is an immutable record of a completed investigation.
type InvestigationSnapshot struct {
	InvestigationID string              `json:"investigation_id"`
	Goal            InvestigationGoal     `json:"goal"`
	RootCause       string              `json:"root_cause,omitempty"`
	Timeline        Timeline            `json:"timeline"`
	KnowledgeGraph  *GraphView          `json:"knowledge_graph,omitempty"`
	Questions       []Question          `json:"questions,omitempty"`
	EvidenceSummary []EvidenceSummary   `json:"evidence_summary"`
	Hypotheses      []Hypothesis        `json:"hypotheses"`
	Confidence      float64             `json:"confidence"`
	Fingerprint     InvestigationFingerprint `json:"fingerprint"`
	Metadata        map[string]any      `json:"metadata,omitempty"`
	CompletedAt     time.Time           `json:"completed_at"`
}

// CompletedInvestigation wraps a snapshot for archival.
type CompletedInvestigation struct {
	Snapshot InvestigationSnapshot `json:"snapshot"`
}

// EvidenceSummary aggregates evidence by category in a snapshot.
type EvidenceSummary struct {
	Category      Category `json:"category"`
	Count         int      `json:"count"`
	SampleSummary string   `json:"sample_summary,omitempty"`
}

// InvestigationFingerprint accelerates similarity search.
type InvestigationFingerprint struct {
	Goal         string     `json:"goal"`
	GraphHash    string     `json:"graph_hash"`
	TimelineHash string     `json:"timeline_hash"`
	RootCause    string     `json:"root_cause,omitempty"`
	Services     []string   `json:"services,omitempty"`
	Categories   []Category `json:"categories,omitempty"`
}

// ArchiveSearchQuery filters archived investigations.
type ArchiveSearchQuery struct {
	Goal      InvestigationGoal `json:"goal,omitempty"`
	Service   string            `json:"service,omitempty"`
	RootCause string            `json:"root_cause,omitempty"`
	Limit     int               `json:"limit,omitempty"`
}

// SimilarityRequest asks the intelligence layer for past investigations like this one.
type SimilarityRequest struct {
	SessionID          string            `json:"session_id,omitempty"`
	Question           string            `json:"question"`
	Service            string            `json:"service,omitempty"`
	Goal               InvestigationGoal `json:"goal,omitempty"`
	LeadingHypothesis  string            `json:"leading_hypothesis,omitempty"`
	EvidenceCategories []Category        `json:"evidence_categories,omitempty"`
	HypothesisIDs      []string          `json:"hypothesis_ids,omitempty"`
	Limit              int               `json:"limit,omitempty"`
	// Active session context (not serialized to MCP).
	Session *Session `json:"-"`
}

// SimilarityResult is one similarity match with explicit reasons.
type SimilarityResult struct {
	InvestigationID string   `json:"investigation_id"`
	Score           float64  `json:"score"`
	MatchingReasons []string `json:"matching_reasons"`
	Question        string   `json:"question,omitempty"`
	Service         string   `json:"service,omitempty"`
	Goal            InvestigationGoal `json:"goal,omitempty"`
	RootCause       string   `json:"root_cause,omitempty"`
	Confidence      float64  `json:"confidence,omitempty"`
	CompletedAt     time.Time `json:"completed_at,omitempty"`
}

// SimilarInvestigation is the legacy MCP-facing similarity match.
type SimilarInvestigation struct {
	SessionID         string            `json:"session_id"`
	Question          string            `json:"question"`
	Service           string            `json:"service,omitempty"`
	Goal              InvestigationGoal `json:"goal"`
	Confidence        float64           `json:"confidence"`
	LeadingHypothesis string            `json:"leading_hypothesis,omitempty"`
	SimilarityScore   float64           `json:"similarity_score"`
	Reason            string            `json:"reason,omitempty"`
	MatchingReasons   []string          `json:"matching_reasons,omitempty"`
	CompletedAt       time.Time         `json:"completed_at,omitempty"`
}

// SimilarityResponse is the result of FindSimilarInvestigations.
type SimilarityResponse struct {
	Matches         []SimilarInvestigation      `json:"matches"`
	Results         []SimilarityResult          `json:"results,omitempty"`
	Lessons         []LessonLearned             `json:"lessons_learned,omitempty"`
	Recommendations *InvestigationRecommendations `json:"recommendations,omitempty"`
}

// GraphPattern describes a recurring graph motif for pattern matching.
type GraphPattern struct {
	EdgeTypes []EdgeType `json:"edge_types,omitempty"`
	NodeTypes []NodeType `json:"node_types,omitempty"`
	Sequence  []string   `json:"sequence,omitempty"`
}

// QuestionTemplate is a reusable question from historical patterns.
type QuestionTemplate struct {
	Text       string     `json:"text"`
	Categories []Category `json:"categories,omitempty"`
	Priority   Priority   `json:"priority,omitempty"`
}

// InvestigationPattern is a reusable investigation playbook mined from history.
type InvestigationPattern struct {
	ID                   string             `json:"id"`
	Name                 string             `json:"name"`
	Description          string             `json:"description"`
	Trigger              GraphPattern       `json:"trigger"`
	RecommendedQuestions []QuestionTemplate `json:"recommended_questions,omitempty"`
	ExpectedEvidence     []Category         `json:"expected_evidence,omitempty"`
	TypicalRootCause     string             `json:"typical_root_cause,omitempty"`
	Occurrences          int                `json:"occurrences,omitempty"`
}

// PatternRequest asks for investigation patterns relevant to the current session.
type PatternRequest struct {
	SessionID string            `json:"session_id,omitempty"`
	Question  string            `json:"question"`
	Service   string            `json:"service,omitempty"`
	Goal      InvestigationGoal `json:"goal,omitempty"`
	Limit     int               `json:"limit,omitempty"`
	Session   *Session          `json:"-"`
}

// SuggestedPattern is a pattern recommendation with confidence.
type SuggestedPattern struct {
	ID                   string             `json:"id"`
	Name                 string             `json:"name"`
	Description          string             `json:"description"`
	Confidence           float64            `json:"confidence"`
	EvidenceCategories   []Category         `json:"evidence_categories,omitempty"`
	HypothesisHint       string             `json:"hypothesis_hint,omitempty"`
	RecommendedQuestions []QuestionTemplate `json:"recommended_questions,omitempty"`
	Occurrences          int                `json:"occurrences"`
	Reason               string             `json:"reason,omitempty"`
}

// PatternResponse is the result of SuggestPatterns.
type PatternResponse struct {
	Patterns []SuggestedPattern `json:"patterns"`
}

// CalibrationRequest asks the intelligence layer to adjust confidence using history.
type CalibrationRequest struct {
	SessionID       string            `json:"session_id,omitempty"`
	HypothesisID    string            `json:"hypothesis_id,omitempty"`
	RawConfidence   float64           `json:"raw_confidence"`
	Goal            InvestigationGoal `json:"goal,omitempty"`
	Service         string            `json:"service,omitempty"`
	HypothesisCount int               `json:"hypothesis_count,omitempty"`
	EvidenceCount   int               `json:"evidence_count,omitempty"`
	Coverage        float64           `json:"coverage,omitempty"`
	RecordCompleted bool              `json:"record_completed,omitempty"`
	Session         *Session          `json:"-"`
}

// CalibrationExplanation documents why confidence was adjusted.
type CalibrationExplanation struct {
	SimilarInvestigations int      `json:"similar_investigations"`
	CorrectCount          int      `json:"correct_count"`
	TotalComparisons      int      `json:"total_comparisons"`
	SupportingHistory     []string `json:"supporting_history,omitempty"`
}

// CalibrationResponse is the result of CalibrateConfidence.
type CalibrationResponse struct {
	OriginalConfidence   float64                 `json:"original_confidence"`
	CalibratedConfidence float64                 `json:"calibrated_confidence"`
	Delta                float64                 `json:"delta"`
	Reason               string                  `json:"reason,omitempty"`
	HistoricalSampleSize int                     `json:"historical_sample_size"`
	HypothesisID         string                  `json:"hypothesis_id,omitempty"`
	Explanation          *CalibrationExplanation `json:"explanation,omitempty"`
}

// LessonLearned is reusable guidance extracted from historical investigations.
type LessonLearned struct {
	ID                  string     `json:"id"`
	Summary             string     `json:"summary"`
	RequiredEvidence    []Category `json:"required_evidence,omitempty"`
	ConfidenceThreshold float64    `json:"confidence_threshold,omitempty"`
	Occurrences         int        `json:"occurrences"`
}

// InvestigationRecommendations bundles intelligence outputs for the runtime.
type InvestigationRecommendations struct {
	SimilarInvestigations  []SimilarityResult   `json:"similar_investigations,omitempty"`
	Patterns               []SuggestedPattern   `json:"patterns,omitempty"`
	TypicalMissingEvidence []Category           `json:"typical_missing_evidence,omitempty"`
	TypicalRootCauses      []string             `json:"typical_root_causes,omitempty"`
	RecommendedQuestions   []QuestionTemplate   `json:"recommended_questions,omitempty"`
	Lessons                []LessonLearned      `json:"lessons_learned,omitempty"`
}

// IntelligenceMetrics tracks incident intelligence operations.
type IntelligenceMetrics struct {
	StoredInvestigations int     `json:"stored_investigations"`
	PatternCount         int     `json:"pattern_count"`
	AverageSimilarity    float64 `json:"average_similarity"`
	PatternReuse         int     `json:"pattern_reuse"`
	CalibrationsRun      int     `json:"calibrations_run"`
	SimilarityQueries    int     `json:"similarity_queries"`
}
