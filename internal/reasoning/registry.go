package reasoning

import (
	"fmt"
	"sort"
	"sync"
)

// Registry holds registered reasoners.
type Registry struct {
	mu        sync.RWMutex
	reasoners map[string]Reasoner
	order     []string
}

// NewRegistry returns an empty reasoner registry.
func NewRegistry() *Registry {
	return &Registry{reasoners: map[string]Reasoner{}}
}

// Register adds a reasoner. Later registrations with the same name replace earlier ones.
func (r *Registry) Register(reasoner Reasoner) {
	if reasoner == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	name := reasoner.Name()
	if _, exists := r.reasoners[name]; !exists {
		r.order = append(r.order, name)
	}
	r.reasoners[name] = reasoner
}

// List returns registered reasoners sorted by priority (descending).
func (r *Registry) List() []Reasoner {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Reasoner, 0, len(r.reasoners))
	for _, name := range r.order {
		if rr, ok := r.reasoners[name]; ok {
			out = append(out, rr)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Priority() > out[j].Priority()
	})
	return out
}

// Get returns a reasoner by name.
func (r *Registry) Get(name string) (Reasoner, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rr, ok := r.reasoners[name]
	if !ok {
		return nil, fmt.Errorf("reasoner %q not registered", name)
	}
	return rr, nil
}
