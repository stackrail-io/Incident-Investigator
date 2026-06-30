// Package runtime is the stateful investigation engine.
//
// It owns the session store, orchestrates recompute on every mutation, and
// validates all changes. Extensions register via Option helpers; core logic
// should not change when adding reasoners or archetypes.
//
// Dependency direction: runtime → engine, reasoning, graph, intelligence, model.
// Nothing in model or graph should import runtime.
package runtime
