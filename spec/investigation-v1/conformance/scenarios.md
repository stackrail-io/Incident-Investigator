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

## Fixture-backed scenarios (reference impl)

| Fixture | Expected leading hypothesis | Min confidence after full evidence |
| ------- | --------------------------- | ---------------------------------- |
| Bad deployment | deployment-caused | ≥ 40 |
| Certificate expiry | certificate-related | ≥ 60 |
| Database outage | database saturation | ≥ 60 |
| DNS outage | network/DNS | ≥ 50 |
| Retry storm | retry/cascade | ≥ 50 |

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
