package intelligence

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/stackrail/incident-investigator/internal/model"
)

// InvestigationArchive persists immutable investigation snapshots.
type InvestigationArchive interface {
	Store(*model.CompletedInvestigation) error
	Find(id string) (*model.InvestigationSnapshot, error)
	Search(query model.ArchiveSearchQuery) ([]*model.InvestigationSnapshot, error)
	All() []*model.InvestigationSnapshot
	Count() int
}

// MemoryArchive is an in-memory investigation archive.
type MemoryArchive struct {
	mu        sync.RWMutex
	snapshots map[string]*model.InvestigationSnapshot
	order     []string
}

// NewMemoryArchive returns an empty archive.
func NewMemoryArchive() *MemoryArchive {
	return &MemoryArchive{snapshots: map[string]*model.InvestigationSnapshot{}}
}

// Store implements InvestigationArchive.
func (a *MemoryArchive) Store(c *model.CompletedInvestigation) error {
	if c == nil {
		return fmt.Errorf("completed investigation is nil")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	id := c.Snapshot.InvestigationID
	if id == "" {
		return fmt.Errorf("investigation id required")
	}
	if _, exists := a.snapshots[id]; exists {
		return nil
	}
	snap := c.Snapshot
	a.snapshots[id] = &snap
	a.order = append(a.order, id)
	return nil
}

// Find implements InvestigationArchive.
func (a *MemoryArchive) Find(id string) (*model.InvestigationSnapshot, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	s, ok := a.snapshots[id]
	if !ok {
		return nil, fmt.Errorf("investigation %q not found", id)
	}
	cp := *s
	return &cp, nil
}

// Search implements InvestigationArchive.
func (a *MemoryArchive) Search(q model.ArchiveSearchQuery) ([]*model.InvestigationSnapshot, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	var out []*model.InvestigationSnapshot
	for _, id := range a.order {
		s := a.snapshots[id]
		if q.Goal != "" && s.Goal != q.Goal {
			continue
		}
		if q.Service != "" && !strings.EqualFold(q.Service, serviceFromSnapshot(s)) {
			continue
		}
		if q.RootCause != "" && s.RootCause != q.RootCause {
			continue
		}
		cp := *s
		out = append(out, &cp)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

// All implements InvestigationArchive.
func (a *MemoryArchive) All() []*model.InvestigationSnapshot {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make([]*model.InvestigationSnapshot, 0, len(a.order))
	for _, id := range a.order {
		cp := *a.snapshots[id]
		out = append(out, &cp)
	}
	return out
}

// Count implements InvestigationArchive.
func (a *MemoryArchive) Count() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.snapshots)
}

func serviceFromSnapshot(s *model.InvestigationSnapshot) string {
	if s == nil || s.Metadata == nil {
		return ""
	}
	if v, ok := s.Metadata["service"].(string); ok {
		return v
	}
	return ""
}

func storeSession(archive InvestigationArchive, s *model.Session) {
	if s == nil || archive == nil {
		return
	}
	snap := BuildSnapshot(s)
	_ = archive.Store(&model.CompletedInvestigation{Snapshot: snap})
}

func snapshotTime(s *model.InvestigationSnapshot) time.Time {
	if s == nil {
		return time.Time{}
	}
	return s.CompletedAt
}
