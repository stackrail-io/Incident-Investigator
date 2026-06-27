package model

// HypothesisStatus captures where a hypothesis sits in the investigation.
type HypothesisStatus string

const (
	StatusProposed  HypothesisStatus = "proposed"
	StatusSupported HypothesisStatus = "supported"
	StatusLeading   HypothesisStatus = "leading"
	StatusRefuted   HypothesisStatus = "refuted"
	StatusConfirmed HypothesisStatus = "confirmed"
)

// Hypothesis is one competing explanation for the incident. The engine never
// produces a single hypothesis; it always maintains a ranked field of them.
type Hypothesis struct {
	ID                  string           `json:"id"`
	Statement           string           `json:"statement"`
	Confidence          float64          `json:"confidence"`
	Status              HypothesisStatus `json:"status"`
	Rationale           string           `json:"rationale,omitempty"`
	SupportingEvidence  []string         `json:"supporting_evidence"`
	ConflictingEvidence []string         `json:"conflicting_evidence"`
}
