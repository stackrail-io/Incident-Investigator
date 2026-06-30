# ADR 0005: Reasoners never mutate runtime state

## Status

Accepted

## Context

Letting reasoners write directly to sessions caused ordering bugs and untestable side effects in early designs.

## Decision

Reasoners receive a **read-only** `Investigation` view. Only the runtime applier mutates session, graph, and evidence.

## Consequences

**Positive:** Clear ownership; parallel reasoner execution possible; third-party reasoners are sandboxed by contract.

**Negative:** Reasoners cannot perform optimizations that require direct writes.
