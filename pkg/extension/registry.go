package extension

import (
	"fmt"
	"sort"
	"sync"
)

// Provider is a named extension component.
type Provider interface {
	Name() string
}

// Registry holds providers by name. Later registration replaces same name.
type Registry[T Provider] struct {
	mu        sync.RWMutex
	providers map[string]T
	order     []string
}

// NewRegistry returns an empty provider registry.
func NewRegistry[T Provider]() *Registry[T] {
	return &Registry[T]{providers: map[string]T{}}
}

// Register adds or replaces a provider.
func (r *Registry[T]) Register(p T) {
	if any(p) == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	name := p.Name()
	if _, exists := r.providers[name]; !exists {
		r.order = append(r.order, name)
	}
	r.providers[name] = p
}

// Get returns a provider by name.
func (r *Registry[T]) Get(name string) (T, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	if !ok {
		var zero T
		return zero, fmt.Errorf("provider %q not registered", name)
	}
	return p, nil
}

// Names returns registered provider names in registration order.
func (r *Registry[T]) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := append([]string(nil), r.order...)
	return out
}

// All returns all providers sorted by descending Priority when T implements Prioritized.
type Prioritized interface {
	Provider
	Priority() int
}

// All returns providers. When T implements Prioritized, results are sorted by
// priority descending; otherwise registration order is preserved.
func (r *Registry[T]) All() []T {
	r.mu.RLock()
	defer r.mu.Unlock()
	out := make([]T, 0, len(r.providers))
	for _, name := range r.order {
		if p, ok := r.providers[name]; ok {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return out
	}
	var probe T
	if _, ok := any(probe).(Prioritized); ok {
		sort.SliceStable(out, func(i, j int) bool {
			return any(out[i]).(Prioritized).Priority() > any(out[j]).(Prioritized).Priority()
		})
	}
	return out
}

// Len returns the number of registered providers.
func (r *Registry[T]) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.providers)
}
