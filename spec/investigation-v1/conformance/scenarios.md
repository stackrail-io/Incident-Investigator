# Conformance Scenarios — Investigation Spec v1

These scenarios define expected behavior for conforming implementations.
The reference implementation validates them via `internal/fixtures` and E2E MCP tests.

## Core tier

### SC-001 — Start investigation

**Given** a valid question and optional goal  
**When** `StartInvestigation` is called  
**Then**

- Returns `investigation_id`
- `state` is `started` or `planning`
- `plan.questions` is non-empty for default playbooks
- `required_evidence` or `evidence_requests` lists initial collection targets

### SC-002 — Submit evidence triggers reasoning

**Given** an active investigation  
**When** one or more `Evidence` items are submitted  
**Then**

- Evidence is appended (deduplicated by id)
- `confidence` and `progress` are updated
- `hypotheses` is non-empty after sufficient deployment/log evidence
- Completed investigations reject further evidence

### SC-003 — Finish produces report

**Given** an investigation with evidence  
**When** `FinishInvestigation` is called  
**Then**

- `state` becomes `completed`
- Returns `Report` with `executive_summary`, `hypotheses`, `confidence`
- Further mutations are rejected

### SC-004 — Evidence categories are normative

**Given** evidence with category `not_a_real_category`  
**When** submitted  
**Then** implementation rejects or maps to `custom` (reference impl: reject at MCP layer)

## Archetype conformance (normative)

Each entry in [archetypes.yaml](../archetypes.yaml) declares a portable conformance
fixture under `conformance/archetype-fixtures/<archetype-id>.yaml`. Conforming
implementations MUST classify the fixture evidence with the declared
`hypothesis_id` as the **leading hypothesis by confidence** after all evidence
batches are submitted (not merely the first entry in a hypothesis list).

Each fixture's `expect_after_all_evidence` block may include:

| Field | Meaning |
| ----- | ------- |
| `leading_hypothesis_id` | Required. Hypothesis that must rank first by confidence. |
| `min_confidence` | Minimum session confidence after all batches. |
| `min_lead_margin` | Minimum confidence gap over the runner-up (default **3.0** for core archetypes; **0** for `unknown-novel`). |
| `must_not_lead` | Hypothesis ids that must not rank first or tie the leader (default **`hypothesis-unknown`**). |
| `min_hypotheses` | Minimum hypothesis count. |
| `graph_nodes_min` | Minimum graph node count. |

The reference implementation validates:

- `TestArchetypesYAMLMatchesBuiltinRegistry` — spec ↔ built-in registry parity
- `TestArchetypeConformanceFixtures` — one end-to-end scenario per archetype

Regenerate fixtures after editing archetype evidence templates:

```bash
go run internal/spec/cmd/gen-archetype-fixtures/main.go
```

**Warning:** the generator overwrites every file in `conformance/archetype-fixtures/` except fixtures marked `hand_tuned: true` (for example `database-lock-contention.yaml`). Edit hand-tuned fixtures directly; keep the flag unless you intend to replace them with generator output.

## Fixture-backed scenarios (reference impl)

Core scenarios are covered by per-archetype YAML fixtures in `conformance/archetype-fixtures/`. See [Archetype conformance](#archetype-conformance-normative) above.

| Archetype fixture | Expected leading hypothesis |
| ----------------- | --------------------------- |
| `deployment-failure.yaml` | `hypothesis-deployment-caused` |
| `configuration-drift.yaml` | `hypothesis-configuration-change` |
| `kubernetes-failure.yaml` | `hypothesis-kubernetes-failure` |
| `certificate-tls-failure.yaml` | `hypothesis-certificate-expiry` |
| `database-saturation.yaml` | `hypothesis-database-saturation` |
| `dns-failure.yaml` | `hypothesis-dns-failure` |
| `retry-storm.yaml` | `hypothesis-retry-storm` |

## Graph tier (optional)

### SG-001 — Graph populated after evidence

**When** evidence is submitted  
**Then** `graph.nodes` includes evidence and hypothesis nodes

### SG-002 — Query upstream causes

**When** `QueryGraph` with `kind=upstream`  
**Then** returns connected subgraph

## Reasoning tier (optional)

### SR-001 — Reasoning cycle recorded

**When** evidence triggers recompute  
**Then** `reasoning_cycles` gains an entry with `applied_actions`

## Intelligence tier (optional)

### SI-001 — Archive on finish

**When** investigation finishes with intelligence enabled  
**Then** snapshot is stored and findable via `FindSimilarInvestigations`
