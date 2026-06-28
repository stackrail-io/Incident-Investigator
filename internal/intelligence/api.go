package intelligence

import (
	"context"

	"github.com/stackrail/incident-investigator/internal/model"
)

// Intelligence is the Incident Intelligence API exposed to the investigation runtime.
// The runtime asks questions; it never knows how similarity, patterns, or calibration
// are implemented or where historical data is stored.
type Intelligence interface {
	FindSimilarInvestigations(ctx context.Context, req model.SimilarityRequest) (*model.SimilarityResponse, error)
	SuggestPatterns(ctx context.Context, req model.PatternRequest) (*model.PatternResponse, error)
	CalibrateConfidence(ctx context.Context, req model.CalibrationRequest) (*model.CalibrationResponse, error)
}

// Noop returns an intelligence implementation that passes confidence through unchanged.
func Noop() Intelligence { return noopIntelligence{} }

type noopIntelligence struct{}

func (noopIntelligence) FindSimilarInvestigations(_ context.Context, _ model.SimilarityRequest) (*model.SimilarityResponse, error) {
	return &model.SimilarityResponse{Matches: []model.SimilarInvestigation{}}, nil
}

func (noopIntelligence) SuggestPatterns(_ context.Context, _ model.PatternRequest) (*model.PatternResponse, error) {
	return &model.PatternResponse{Patterns: []model.SuggestedPattern{}}, nil
}

func (noopIntelligence) CalibrateConfidence(_ context.Context, req model.CalibrationRequest) (*model.CalibrationResponse, error) {
	return &model.CalibrationResponse{
		OriginalConfidence:   req.RawConfidence,
		CalibratedConfidence: req.RawConfidence,
		Delta:                0,
		Reason:               "Intelligence layer disabled.",
	}, nil
}
