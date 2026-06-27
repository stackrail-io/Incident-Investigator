package model

// Contradiction records an inconsistency the engine detected in the evidence,
// such as a deployment whose timestamp falls after the incident already began.
type Contradiction struct {
	ID           string   `json:"id"`
	Description  string   `json:"description"`
	Severity     string   `json:"severity"`
	EvidenceRefs []string `json:"evidence_refs"`
}
