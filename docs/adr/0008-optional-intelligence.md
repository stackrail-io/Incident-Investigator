# ADR 0008: Incident Intelligence is optional

## Status

Accepted

## Context

Historical learning (similarity, patterns, calibration) requires storage and privacy considerations not every deployment wants.

## Decision

**Intelligence is a pluggable layer** behind `intelligence.Intelligence`. Default enables in-memory archive; `Noop()` disables without affecting protocol, graph, or reasoning.

## Consequences

**Positive:** Air-gapped and greenfield deployments work out of the box; intelligence upgrades are additive.

**Negative:** Calibration benefits require opt-in archive population.
