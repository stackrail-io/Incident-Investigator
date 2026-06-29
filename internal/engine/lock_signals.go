package engine

import (
	"strings"
	"time"

	"github.com/stackrail/incident-investigator/internal/model"
)

// lockReleaseWindow is the maximum spread between statement end timestamps that
// still counts as a serialized lock release (queue unblocked together).
const lockReleaseWindow = 2 * time.Second

// LockContention captures vendor-neutral signals of row/table lock waiting.
type LockContention struct {
	Present                bool
	Entity                 string
	HolderIDs              []string
	WaiterIDs              []string
	SerializedRelease      bool
	MissingLockTimeouts    bool
	HealthyDatabaseMetrics bool
}

func analyzeLockContention(ordered []*model.Evidence) LockContention {
	lc := LockContention{}

	byEntity := map[string][]*model.Evidence{}
	for _, e := range ordered {
		if e.Category != model.CategoryDatabaseEvents || e.Entity == "" {
			continue
		}
		byEntity[e.Entity] = append(byEntity[e.Entity], e)
	}

	for entity, evs := range byEntity {
		if len(evs) < 2 {
			continue
		}
		for _, cluster := range clusterByTimestamp(evs, lockReleaseWindow) {
			if len(cluster) < 2 {
				continue
			}
			holders, waiters := partitionLockQueue(cluster)
			if len(holders) == 0 || len(waiters) == 0 {
				continue
			}
			lc.Present = true
			lc.Entity = entity
			lc.SerializedRelease = true
			for _, h := range holders {
				lc.HolderIDs = append(lc.HolderIDs, h.ID)
			}
			for _, w := range waiters {
				lc.WaiterIDs = append(lc.WaiterIDs, w.ID)
			}
			break
		}
		if lc.Present {
			break
		}
	}

	for _, e := range ordered {
		if e.Category != model.CategoryConfigurationChanges {
			continue
		}
		text := haystack(e)
		if mentionsMissingTimeout(text) {
			lc.MissingLockTimeouts = true
			break
		}
	}

	for _, e := range ordered {
		if e.Category != model.CategoryMetrics {
			continue
		}
		if looksLikeHealthyDatabaseMetrics(e) {
			lc.HealthyDatabaseMetrics = true
			break
		}
	}

	return lc
}

func clusterByTimestamp(evs []*model.Evidence, window time.Duration) [][]*model.Evidence {
	if len(evs) == 0 {
		return nil
	}
	sorted := make([]*model.Evidence, len(evs))
	copy(sorted, evs)
	// ordered input is already chronological; re-sort for safety within entity.
	sortSliceByTime(sorted)

	var clusters [][]*model.Evidence
	start := 0
	for i := 1; i <= len(sorted); i++ {
		if i == len(sorted) || sorted[i].Timestamp.Sub(sorted[start].Timestamp) > window {
			if i-start >= 2 {
				clusters = append(clusters, sorted[start:i])
			}
			start = i
		}
	}
	return clusters
}

func sortSliceByTime(evs []*model.Evidence) {
	for i := 1; i < len(evs); i++ {
		for j := i; j > 0 && evs[j].Timestamp.Before(evs[j-1].Timestamp); j-- {
			evs[j], evs[j-1] = evs[j-1], evs[j]
		}
	}
}

func partitionLockQueue(evs []*model.Evidence) (holders, waiters []*model.Evidence) {
	for _, e := range evs {
		rows, ok := payloadInt(e.Payload, "rows_affected")
		if !ok {
			continue
		}
		if rows > 0 {
			holders = append(holders, e)
		} else if rows == 0 {
			waiters = append(waiters, e)
		}
	}
	return holders, waiters
}

func payloadInt(payload map[string]any, key string) (int, bool) {
	if payload == nil {
		return 0, false
	}
	v, ok := payload[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}

func mentionsMissingTimeout(text string) bool {
	hasTimeout := strings.Contains(text, "lock_timeout") || strings.Contains(text, "statement_timeout")
	if !hasTimeout {
		return false
	}
	return strings.Contains(text, "no ") ||
		strings.Contains(text, "without") ||
		strings.Contains(text, "missing") ||
		strings.Contains(text, "not set") ||
		strings.Contains(text, "absent") ||
		strings.Contains(text, "unbounded")
}

func looksLikeHealthyDatabaseMetrics(e *model.Evidence) bool {
	text := haystack(e)
	isDB := matchesAny(text, signalKeywords["database"]) ||
		strings.Contains(text, "postgres") ||
		strings.Contains(text, "mysql") ||
		strings.Contains(text, "sql")
	if !isDB {
		return false
	}
	if strings.Contains(text, "saturat") || strings.Contains(text, "exhaust") {
		return false
	}
	if strings.Contains(text, "100/100") || strings.Contains(text, "cpu 9") || strings.Contains(text, "cpu 8") {
		return false
	}
	return true
}
