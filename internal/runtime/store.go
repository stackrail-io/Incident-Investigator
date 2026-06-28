package runtime

import (
	"errors"
	"sync"

	"github.com/stackrail/incident-investigator/internal/model"
)

// ErrSessionNotFound is returned when a session id does not exist.
var ErrSessionNotFound = errors.New("investigation session not found")

// ErrSessionCompleted is returned when mutating a finished investigation.
var ErrSessionCompleted = errors.New("investigation session is already completed")

// ErrEmptyEvidence is returned when submit is called with no evidence.
var ErrEmptyEvidence = errors.New("at least one evidence item is required")

// ErrInvalidEvidence is returned when evidence fails validation.
var ErrInvalidEvidence = errors.New("invalid evidence")

// ErrInvalidTimeWindow is returned when the investigation window is malformed.
var ErrInvalidTimeWindow = errors.New("time_window end must not be before start")

// ErrDuplicateEvidenceID is returned when evidence ids collide.
var ErrDuplicateEvidenceID = errors.New("duplicate evidence id")

// Store persists investigation sessions. The initial implementation is purely
// in-memory; the interface leaves room for durable backends later without
// touching the runtime or MCP layers.
type Store interface {
	Create(s *model.Session) error
	Get(id string) (*model.Session, error)
	Save(s *model.Session) error
	// WithSession runs fn while holding an exclusive lock on the session, enabling
	// atomic read-modify-write without races between concurrent MCP calls.
	WithSession(id string, fn func(*model.Session) error) (*model.Session, error)
}

// MemoryStore keeps one map of sessions guarded by a mutex.
type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*model.Session
}

// NewMemoryStore returns an empty in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{sessions: map[string]*model.Session{}}
}

// Create inserts a new session.
func (m *MemoryStore) Create(s *model.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[s.ID] = s
	return nil
}

// Get returns the session by id.
func (m *MemoryStore) Get(id string) (*model.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return s, nil
}

// Save updates an existing session (no-op distinction from Create for the
// in-memory backend, but meaningful for future durable stores).
func (m *MemoryStore) Save(s *model.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[s.ID] = s
	return nil
}

// WithSession implements Store.
func (m *MemoryStore) WithSession(id string, fn func(*model.Session) error) (*model.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[id]
	if !ok {
		return nil, ErrSessionNotFound
	}
	if err := fn(s); err != nil {
		return nil, err
	}
	return s, nil
}
