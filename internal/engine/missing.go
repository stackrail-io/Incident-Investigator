package engine

import "github.com/stackrail/incident-investigator/internal/model"

// MissingEvidenceDetector determines which of the planner's desired categories
// have not yet been provided and would most improve confidence.
type MissingEvidenceDetector interface {
	Detect(s *model.Session, required []model.EvidenceRequest) []model.EvidenceRequest
}

// HeuristicMissingEvidenceDetector returns desired-but-absent categories.
type HeuristicMissingEvidenceDetector struct{}

// NewHeuristicMissingEvidenceDetector returns the default detector.
func NewHeuristicMissingEvidenceDetector() *HeuristicMissingEvidenceDetector {
	return &HeuristicMissingEvidenceDetector{}
}

// Detect implements MissingEvidenceDetector.
func (d *HeuristicMissingEvidenceDetector) Detect(s *model.Session, required []model.EvidenceRequest) []model.EvidenceRequest {
	var missing []model.EvidenceRequest
	for _, r := range required {
		if !s.HasCategory(r.Category) {
			missing = append(missing, r)
		}
	}
	if missing == nil {
		return []model.EvidenceRequest{}
	}
	return missing
}

// Progress reports how much of the desired evidence has been collected, weighted
// by request priority (0..100).
func Progress(s *model.Session, required []model.EvidenceRequest) float64 {
	if len(required) == 0 {
		return 0
	}
	var total, covered float64
	for _, r := range required {
		w := r.Priority.Weight()
		total += w
		if s.HasCategory(r.Category) {
			covered += w
		}
	}
	if total == 0 {
		return 0
	}
	return round1(covered / total * 100)
}
