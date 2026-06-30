# Investigation Specification v1

**Status:** Normative  
**Version:** 1.0.0  
**Scope:** Implementation-independent contract for AI-assisted incident investigations

This document defines the Investigation Protocol in plain language. Machine-readable schemas live alongside this file in YAML and JSON Schema form.

## Purpose

Incident Investigator specifies **how** to conduct an investigation—not **how** to collect evidence from vendors. Any runtime (Go, Rust, TypeScript) that implements this specification can interoperate with MCP clients, REST gateways, or batch replay tools using the same payloads.

## Core entities

### Investigation

The aggregate root. An investigation has:

- A natural-language **question** and optional **service** and **time window**
- An investigation **goal** (root cause, timeline, blast radius, …)
- Lifecycle **state** (started → reasoning → completed)
- Collections of evidence, hypotheses, questions, and findings
- Derived **confidence** and **progress**

See `model.yaml` → `Investigation`.

### Question

A structured inquiry that drives evidence collection. Questions have:

- **id**, **title**, **description**, **priority**
- **status** (open, resolved, …)
- **required_evidence** categories
- Optional **depends_on** other questions
- **resolution** when answered

Investigations are collections of questions—not unstructured chat.

### Evidence

An immutable, vendor-neutral observation:

- **id**, **timestamp**, **category**, **summary**
- Optional **entity** (service/host) and **payload** (opaque vendor detail)

Evidence is append-only after submission.

### Evidence Request

A planner- or protocol-level request for a category of evidence:

- **category**, **priority**, **reason**
- Protocol variant links to a **question_id** (`ProtocolEvidenceRequest`)

### Hypothesis

A competing explanation:

- **id**, **statement**, **confidence**, **status**
- **supporting_evidence** and **conflicting_evidence** references
- Multiple hypotheses are always maintained; one may **lead** by confidence

### Finding

An observation emitted by a reasoner:

- **type**, **summary**, optional **confidence**, **reason**, **refs**

Findings inform humans and downstream systems; they do not directly mutate state.

### Recommendation

Actionable follow-up text, optionally attributed to a **source** reasoner or report section.

### Reasoning Action

A declarative mutation proposed by a reasoner and applied by the runtime after validation. Examples: replace hypotheses, set confidence, link graph nodes, create recommendations.

Reasoners return actions; they never write session state directly.

### Investigation Graph

A directed graph of investigation entities:

- **Nodes:** evidence, hypotheses, services, timeline events, …
- **Edges:** supports, contradicts, caused, occurred_before, …

The graph is the canonical structural view. See `extensions/graph-v1.yaml`.

### Investigation Plan

The active protocol artifact:

- **goal**, **current_stage**
- **questions**, **evidence_requests**, **resolution_history**
- Plan **confidence**

## Lifecycle

Defined in `lifecycle.yaml`:

```
started → planning → collecting_evidence → reasoning → ready_to_conclude → completed
```

Questions transition independently (open → resolved). Investigations may **fail** or be **cancelled** in extended profiles.

## Protocol

Defined in `operations.yaml`. Core operations:

| Operation | Effect |
| --------- | ------ |
| StartInvestigation | Create investigation, generate plan and initial questions |
| SubmitEvidence | Append evidence, recompute derived state |
| ResolveQuestion | Explicitly resolve a protocol question |
| GetInvestigation | Return current state |
| FinishInvestigation | Generate report, mark completed |

MCP tool names map to these operations in the reference implementation.

## Extensions

Optional capability profiles:

| Extension | File | Capability |
| --------- | ---- | ---------- |
| Graph v1 | `extensions/graph-v1.yaml` | Graph queries, subgraph, path explanation |
| Reasoning v1 | `extensions/reasoning-v1.yaml` | Multi-reasoner cycles, action log |
| Intelligence v1 | `extensions/intelligence-v1.yaml` | Archive, similarity, patterns, calibration |

Implementations declare supported extensions; core protocol works without them.

## Conformance

Three tiers (see `conformance/scenarios.md`):

1. **Core** — start, submit, finish, category validation
2. **Graph** — graph populated, upstream queries
3. **Reasoning / Intelligence** — cycles recorded, archive on finish

**Archetype fixtures** (`conformance/archetype-fixtures/`) define 32 end-to-end scenarios with expected leading hypotheses.

## Archetype library

`archetypes.yaml` declares 32 vendor-neutral failure modes. Each archetype specifies:

- Domain, priority, hypothesis id
- Signal triggers and expected evidence categories
- Conformance fixture path

## Evidence categories

`categories.yaml` is the normative taxonomy. Unknown categories must be rejected or mapped to `custom` by policy.

## JSON Schema

`jsonschema/investigation-v1.schema.json` aggregates schemas for validation tooling.

## Implementing in another language

1. Implement entities from `model.yaml`
2. Implement lifecycle from `lifecycle.yaml`
3. Implement operations from `operations.yaml`
4. Pass archetype conformance fixtures
5. Optionally implement extensions

Reference implementation: Go module `github.com/stackrail/incident-investigator`.

## Related documents

- [Spec README](./README.md)
- [Conformance scenarios](./conformance/scenarios.md)
- [Architecture](../../docs/architecture.md)
- [Design principles](../../docs/design-principles.md)
