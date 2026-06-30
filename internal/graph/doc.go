// Package graph implements the Investigation Graph store, traversal, and queries.
//
// The graph is the canonical structural view of an investigation. Graph stores
// persist nodes and edges; they do not own hypothesis confidence or evidence lists.
//
// Dependency direction: graph → model only.
package graph
