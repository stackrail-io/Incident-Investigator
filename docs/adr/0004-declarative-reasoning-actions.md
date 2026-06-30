# ADR 0004: Declarative reasoning actions

## Status

Accepted

## Context

Multiple reasoners (temporal, causal, semantic) must contribute without racing to mutate shared state.

## Decision

Reasoners return **declarative `ReasoningAction`s**. A validator and applier merge actions in priority order under runtime control.

## Consequences

**Positive:** Composable reasoners; auditable action log; testable validation rules.

**Negative:** Action schema must evolve carefully; some logic is harder to express declaratively.
