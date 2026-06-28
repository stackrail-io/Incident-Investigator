package runtime

import (
	"context"
	"math"

	"github.com/stackrail/incident-investigator/internal/intelligence"
	"github.com/stackrail/incident-investigator/internal/model"
)

// WithIntelligence overrides the incident intelligence API.
func WithIntelligence(i intelligence.Intelligence) Option {
	return func(r *Runtime) { r.intelligence = i }
}

// FindSimilarInvestigations delegates to the incident intelligence API.
func (r *Runtime) FindSimilarInvestigations(ctx context.Context, sessionID string, limit int) (*model.SimilarityResponse, error) {
	s, err := r.store.Get(sessionID)
	if err != nil {
		return nil, err
	}
	return r.intelligence.FindSimilarInvestigations(ctx, similarityRequestFromSession(s, limit))
}

// SuggestPatterns delegates to the incident intelligence API.
func (r *Runtime) SuggestPatterns(ctx context.Context, sessionID string, limit int) (*model.PatternResponse, error) {
	s, err := r.store.Get(sessionID)
	if err != nil {
		return nil, err
	}
	return r.intelligence.SuggestPatterns(ctx, patternRequestFromSession(s, limit))
}

// CalibrateConfidence delegates to the incident intelligence API.
func (r *Runtime) CalibrateConfidence(ctx context.Context, req model.CalibrationRequest) (*model.CalibrationResponse, error) {
	return r.intelligence.CalibrateConfidence(ctx, req)
}

// CalibrateConfidenceForSession builds a calibration request from a session and delegates.
func (r *Runtime) CalibrateConfidenceForSession(ctx context.Context, sessionID, hypothesisID string) (*model.CalibrationResponse, error) {
	s, err := r.store.Get(sessionID)
	if err != nil {
		return nil, err
	}
	req := calibrationRequestFromSession(s, false)
	req.HypothesisID = hypothesisID
	if hypothesisID != "" {
		for _, h := range s.Hypotheses {
			if h.ID == hypothesisID {
				req.RawConfidence = h.Confidence
				break
			}
		}
	}
	return r.intelligence.CalibrateConfidence(ctx, req)
}

func (r *Runtime) applyIntelligenceCalibration(ctx context.Context, s *model.Session) {
	if r.intelligence == nil {
		return
	}
	cal, err := r.intelligence.CalibrateConfidence(ctx, calibrationRequestFromSession(s, false))
	if err != nil || cal == nil {
		return
	}
	if math.Abs(cal.Delta) >= 0.5 {
		s.Confidence = cal.CalibratedConfidence
		s.AddJournal("intelligence_calibration", cal.Reason, s.Confidence, r.now())
	}
}

func similarityRequestFromSession(s *model.Session, limit int) model.SimilarityRequest {
	req := model.SimilarityRequest{
		SessionID: s.ID,
		Question:  s.Question,
		Service:   s.Service,
		Goal:      s.Goal,
		Limit:     limit,
		Session:   s,
	}
	for c := range sessionCategorySet(s) {
		req.EvidenceCategories = append(req.EvidenceCategories, c)
	}
	if len(s.Hypotheses) > 0 {
		req.LeadingHypothesis = s.Hypotheses[0].ID
	}
	return req
}

func patternRequestFromSession(s *model.Session, limit int) model.PatternRequest {
	return model.PatternRequest{
		SessionID: s.ID,
		Question:  s.Question,
		Service:   s.Service,
		Goal:      s.Goal,
		Limit:     limit,
		Session:   s,
	}
}

func calibrationRequestFromSession(s *model.Session, record bool) model.CalibrationRequest {
	return model.CalibrationRequest{
		SessionID:       s.ID,
		RawConfidence:   s.Confidence,
		Goal:            s.Goal,
		Service:         s.Service,
		HypothesisCount: len(s.Hypotheses),
		EvidenceCount:   len(s.Evidence),
		Coverage:        s.Coverage.Overall,
		RecordCompleted: record,
		Session:         s,
	}
}

func sessionCategorySet(s *model.Session) map[model.Category]bool {
	out := map[model.Category]bool{}
	for _, e := range s.Evidence {
		if e != nil {
			out[e.Category] = true
		}
	}
	return out
}
