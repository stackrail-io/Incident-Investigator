package engine

import (
	"github.com/stackrail/incident-investigator/internal/model"
	sig "github.com/stackrail/incident-investigator/internal/signals"
)

// Signals is a distilled, vendor-neutral summary of what the evidence implies.
type Signals = sig.Signals

// LockContention captures vendor-neutral row/table lock-wait signals.
type LockContention = sig.LockContention

// Analyze walks session evidence and extracts high-level signals.
func Analyze(s *model.Session) Signals { return sig.Analyze(s) }

// sortedByTime is kept for engine-internal callers during migration.
func sortedByTime(in []*model.Evidence) []*model.Evidence { return sig.SortedByTime(in) }

func haystack(e *model.Evidence) string { return sig.Haystack(e) }

func matchesAny(text string, words []string) bool { return sig.MatchesAny(text, words) }

var signalKeywords = sig.Keywords

func evidenceMatching(s *model.Session, pred func(*model.Evidence) bool) []string {
	return sig.EvidenceMatching(s, pred)
}

func mentionsMissingTimeout(text string) bool { return sig.MentionsMissingTimeout(text) }

func looksLikeHealthyDatabaseMetrics(e *model.Evidence) bool {
	return sig.LooksLikeHealthyDatabaseMetrics(e)
}

// Round1 rounds to one decimal place (shared across engine scoring).
func Round1(v float64) float64 {
	return float64(int(v*10+0.5)) / 10
}

func round1(v float64) float64 { return Round1(v) }
