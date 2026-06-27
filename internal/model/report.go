package model

// Report is the final investigation deliverable returned by finish_investigation.
type Report struct {
	SessionID           string            `json:"session_id"`
	Question            string            `json:"question"`
	ExecutiveSummary    string            `json:"executive_summary"`
	Timeline            Timeline          `json:"timeline"`
	Evidence            []*Evidence       `json:"evidence"`
	Hypotheses          []Hypothesis      `json:"hypotheses"`
	RootCauseCandidates []Hypothesis      `json:"root_cause_candidates"`
	Graph               *GraphView        `json:"graph"`
	BlastRadius         BlastRadius       `json:"blast_radius"`
	Contradictions      []Contradiction   `json:"contradictions"`
	MissingEvidence     []EvidenceRequest `json:"missing_evidence"`
	Recommendations     []string          `json:"recommendations"`
	Confidence          float64           `json:"confidence"`
	Postmortem          string            `json:"postmortem"`
}
