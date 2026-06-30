package extension

// Reasoner proposes declarative reasoning actions from a read-only investigation view.
//
// Expected methods (implemented in internal/reasoning.Reasoner):
//
//	Name() string
//	Priority() int
//	Supports(session) bool
//	Analyze(ctx, investigation) (*ReasoningResult, error)
//
// Ownership: read-only access to investigation state; never mutate session or graph.
type Reasoner interface {
	NamedProvider
	Priority() int
}

// GraphStore persists investigation graph nodes and edges.
//
// Expected methods (implemented in internal/graph.Store):
//
//	AddNode, UpdateNode, DeleteNode, GetNode
//	AddEdge, RemoveEdge, GetEdges
//	Traverse(query) (*Subgraph, error)
//
// Ownership: graph structure only.
type GraphStore interface {
	NamedProvider
}

// InvestigationArchive stores completed investigation snapshots.
//
// Expected methods (implemented in internal/intelligence.InvestigationArchive):
//
//	Archive(snapshot) error
//	List(filter) ([]Snapshot, error)
//	Get(id) (*Snapshot, error)
//
// Ownership: historical data separate from active sessions.
type InvestigationArchive interface {
	NamedProvider
}

// PatternProvider suggests investigation patterns from a library and archive.
type PatternProvider interface {
	NamedProvider
}

// PlaybookProvider supplies declarative playbooks for investigation goals.
type PlaybookProvider interface {
	NamedProvider
}

// ArchetypeProvider supplies failure-mode templates for scoring and seed questions.
type ArchetypeProvider interface {
	NamedProvider
}

// ReportGenerator assembles the final investigation report.
//
// Lifecycle: invoked once on Finish; must not mutate session.
type ReportGenerator interface {
	NamedProvider
}

// ConfidenceProvider scores investigation confidence from hypotheses and coverage.
type ConfidenceProvider interface {
	NamedProvider
}

// SimilarityProvider finds historically similar investigations.
type SimilarityProvider interface {
	NamedProvider
}
