package intelligence

import "sync"

// Metrics tracks incident intelligence operations.
type Metrics struct {
	mu                   sync.Mutex
	StoredInvestigations int
	PatternCount         int
	AverageSimilarity    float64
	PatternReuse         int
	CalibrationsRun      int
	SimilarityQueries    int
	similaritySum        float64
	similarityCount      int
}

func newMetrics() *Metrics { return &Metrics{} }

func (m *Metrics) recordStore() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.StoredInvestigations++
}

func (m *Metrics) recordSimilarity(avgScore float64, matches int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SimilarityQueries++
	if matches > 0 {
		m.similaritySum += avgScore
		m.similarityCount++
		m.AverageSimilarity = m.similaritySum / float64(m.similarityCount)
	}
}

func (m *Metrics) recordCalibration() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CalibrationsRun++
}

func (m *Metrics) recordPatternReuse(n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PatternReuse += n
}

func (m *Metrics) setPatternCount(n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PatternCount = n
}

func (m *Metrics) snapshot() IntelligenceMetricsSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	return IntelligenceMetricsSnapshot{
		StoredInvestigations: m.StoredInvestigations,
		PatternCount:         m.PatternCount,
		AverageSimilarity:    round1(m.AverageSimilarity),
		PatternReuse:         m.PatternReuse,
		CalibrationsRun:      m.CalibrationsRun,
		SimilarityQueries:    m.SimilarityQueries,
	}
}

// IntelligenceMetricsSnapshot is a point-in-time metrics export.
type IntelligenceMetricsSnapshot struct {
	StoredInvestigations int     `json:"stored_investigations"`
	PatternCount         int     `json:"pattern_count"`
	AverageSimilarity    float64 `json:"average_similarity"`
	PatternReuse         int     `json:"pattern_reuse"`
	CalibrationsRun      int     `json:"calibrations_run"`
	SimilarityQueries    int     `json:"similarity_queries"`
}
