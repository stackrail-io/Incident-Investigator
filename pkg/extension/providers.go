package extension

// ReasonerRegistry holds reasoning providers. Implementations satisfy
// internal/reasoning.Reasoner when built in-tree.
//
// Lifecycle: runtime invokes Supports → Analyze on each recompute; actions
// are validated and applied by the orchestrator. Reasoners must not mutate state.
type ReasonerRegistry = Registry[NamedProvider]

// NewReasonerRegistry returns an empty reasoner registry.
func NewReasonerRegistry() *ReasonerRegistry {
	return NewRegistry[NamedProvider]()
}

// GraphStoreRegistry holds graph persistence providers.
type GraphStoreRegistry = Registry[NamedProvider]

// NewGraphStoreRegistry returns an empty graph store registry.
func NewGraphStoreRegistry() *GraphStoreRegistry {
	return NewRegistry[NamedProvider]()
}

// ArchetypeRegistry holds failure-mode archetype providers.
type ArchetypeRegistry = Registry[NamedProvider]

// NewArchetypeRegistry returns an empty archetype registry.
func NewArchetypeRegistry() *ArchetypeRegistry {
	return NewRegistry[NamedProvider]()
}

// PlaybookRegistry holds investigation playbook providers.
type PlaybookRegistry = Registry[NamedProvider]

// NewPlaybookRegistry returns an empty playbook registry.
func NewPlaybookRegistry() *PlaybookRegistry {
	return NewRegistry[NamedProvider]()
}

// ReportRegistry holds report generator providers.
type ReportRegistry = Registry[NamedProvider]

// NewReportRegistry returns an empty report registry.
func NewReportRegistry() *ReportRegistry {
	return NewRegistry[NamedProvider]()
}

// PatternRegistry holds pattern suggestion providers.
type PatternRegistry = Registry[NamedProvider]

// NewPatternRegistry returns an empty pattern registry.
func NewPatternRegistry() *PatternRegistry {
	return NewRegistry[NamedProvider]()
}

// SimilarityRegistry holds similarity engine providers.
type SimilarityRegistry = Registry[NamedProvider]

// NewSimilarityRegistry returns an empty similarity registry.
func NewSimilarityRegistry() *SimilarityRegistry {
	return NewRegistry[NamedProvider]()
}

// NamedProvider is the minimal contract for registry entries documented in
// pkg/extension. Concrete types add domain methods in internal packages.
type NamedProvider interface {
	Provider
}

// Bundle groups provider registries for one-shot runtime wiring.
type Bundle struct {
	Reasoners   *ReasonerRegistry
	Archetypes  *ArchetypeRegistry
	Playbooks   *PlaybookRegistry
	Reports     *ReportRegistry
	Patterns    *PatternRegistry
	Similarity  *SimilarityRegistry
	GraphStores *GraphStoreRegistry
}

// NewBundle returns registries with empty defaults.
func NewBundle() *Bundle {
	return &Bundle{
		Reasoners:   NewReasonerRegistry(),
		Archetypes:  NewArchetypeRegistry(),
		Playbooks:   NewPlaybookRegistry(),
		Reports:     NewReportRegistry(),
		Patterns:    NewPatternRegistry(),
		Similarity:  NewSimilarityRegistry(),
		GraphStores: NewGraphStoreRegistry(),
	}
}
