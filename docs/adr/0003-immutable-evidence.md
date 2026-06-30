# ADR 0003: Immutable evidence

## Status

Accepted

## Context

Mutable evidence breaks replay, audit trails, and conformance tests. Clients may need to correct mistakes.

## Decision

Evidence is **append-only** after submission. Corrections are new evidence items. Duplicate ids are rejected.

## Consequences

**Positive:** Deterministic replay; journal integrity; simpler concurrency.

**Negative:** Clients must issue superseding evidence explicitly.
