# ADR 0001: Investigation Graph as canonical structure

## Status

Accepted

## Context

Investigations connect evidence, hypotheses, services, deployments, and timeline events. Without a single structural model, each engine maintains its own adjacency and path queries become inconsistent.

## Decision

Introduce an **Investigation Graph** as the canonical representation of relationships. Evidence and hypotheses sync into graph nodes; traversal and path explanation read from the graph store.

## Consequences

**Positive:** Unified queries (`upstream`, `downstream`, `ExplainPath`); exporters can emit Mermaid/GraphML from one source.

**Negative:** Dual maintenance between session lists and graph until sync is complete each recompute.
