package intelligence

import (
	"context"
	"sync"

	"github.com/stackrail/incident-investigator/internal/model"
)

// MemoryService is the default Incident Intelligence implementation composing
// archive, similarity, pattern, and calibration engines.
type MemoryService struct {
	mu         sync.RWMutex
	archive    InvestigationArchive
	similarity SimilarityEngine
	patterns   PatternEngine
	calibrator ConfidenceCalibrator
	metrics    *Metrics
}

// NewMemoryService returns a wired in-memory intelligence service.
func NewMemoryService() *MemoryService {
	patterns := NewHeuristicPatternEngine()
	svc := &MemoryService{
		archive:    NewMemoryArchive(),
		similarity: NewHeuristicSimilarityEngine(),
		patterns:   patterns,
		calibrator: NewHeuristicCalibrator(),
		metrics:    newMetrics(),
	}
	svc.metrics.setPatternCount(len(patterns.Library()))
	return svc
}

// NewMemoryServiceWithArchive constructs a service with a custom archive (tests).
func NewMemoryServiceWithArchive(archive InvestigationArchive) *MemoryService {
	patterns := NewHeuristicPatternEngine()
	return &MemoryService{
		archive:    archive,
		similarity: NewHeuristicSimilarityEngine(),
		patterns:   patterns,
		calibrator: NewHeuristicCalibrator(),
		metrics:    newMetrics(),
	}
}

// Metrics returns operational counters.
func (m *MemoryService) Metrics() IntelligenceMetricsSnapshot {
	return m.metrics.snapshot()
}

// FindSimilarInvestigations implements Intelligence.
func (m *MemoryService) FindSimilarInvestigations(ctx context.Context, req model.SimilarityRequest) (*model.SimilarityResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	limit := req.Limit
	if limit <= 0 {
		limit = 5
	}
	if req.Session != nil && len(req.HypothesisIDs) == 0 {
		for _, h := range req.Session.Hypotheses {
			req.HypothesisIDs = append(req.HypothesisIDs, h.ID)
		}
	}

	results, err := m.similarity.FindSimilar(ctx, req, m.archive, limit)
	if err != nil {
		return nil, err
	}
	var avg float64
	for _, r := range results {
		avg += r.Score
	}
	if len(results) > 0 {
		avg /= float64(len(results))
	}
	m.metrics.recordSimilarity(avg, len(results))

	lessons := ExtractLessons(results, m.archive)
	patterns, _ := m.patterns.Suggest(ctx, model.PatternRequest{
		SessionID: req.SessionID, Question: req.Question, Service: req.Service, Goal: req.Goal, Session: req.Session,
	}, m.archive, 5)
	m.metrics.recordPatternReuse(len(patterns))

	return &model.SimilarityResponse{
		Matches:         toSimilarInvestigations(results),
		Results:         results,
		Lessons:         lessons,
		Recommendations: BuildRecommendations(results, patterns, lessons, m.archive),
	}, nil
}

// SuggestPatterns implements Intelligence.
func (m *MemoryService) SuggestPatterns(ctx context.Context, req model.PatternRequest) (*model.PatternResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	limit := req.Limit
	if limit <= 0 {
		limit = 5
	}
	patterns, err := m.patterns.Suggest(ctx, req, m.archive, limit)
	if err != nil {
		return nil, err
	}
	m.metrics.recordPatternReuse(len(patterns))
	if patterns == nil {
		patterns = []model.SuggestedPattern{}
	}
	return &model.PatternResponse{Patterns: patterns}, nil
}

// CalibrateConfidence implements Intelligence.
func (m *MemoryService) CalibrateConfidence(_ context.Context, req model.CalibrationRequest) (*model.CalibrationResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if req.RecordCompleted && req.Session != nil {
		storeSession(m.archive, req.Session)
		m.metrics.recordStore()
	}

	resp := CalibrateFromRequest(req, m.archive.All(), m.calibrator)
	m.metrics.recordCalibration()
	return resp, nil
}
