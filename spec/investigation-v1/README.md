# Investigation Specification v1

**Investigation Specification v1** is a vendor-neutral, transport-agnostic protocol
for AI-assisted incident investigations. It defines the information model,
lifecycle, and operations that any compliant runtime must implement.

Incident Investigator is the **reference implementation** of this specification.
Its MCP tools are one **binding** of the protocol — not the protocol itself.

## Design goals

1. **Separation of specification and implementation** — Go types and MCP handlers
   conform to the spec; they do not define it.
2. **Vendor neutrality** — evidence is categorized abstractly; assistants map
   CloudWatch, Datadog, Kubernetes, etc. into normative categories.
3. **Question-driven investigation** — investigations progress by resolving
   questions, not by ad-hoc chat turns.
4. **Graph as source of truth** — hypotheses, evidence, and causal relationships
   live in a structured investigation graph.
5. **Extensibility** — core spec is small; graph, reasoning, and intelligence
   are optional extension modules.

## Specification documents

| Document | Purpose |
| -------- | ------- |
| [jsonschema/investigation-v1.schema.json](./jsonschema/investigation-v1.schema.json) | Bundled JSON Schema (draft 2020-12) for tooling |
| [model.yaml](./model.yaml) | Core entities: Investigation, Question, Evidence, Hypothesis, Finding, … |
| [lifecycle.yaml](./lifecycle.yaml) | Canonical state machine and stage annotations |
| [operations.yaml](./operations.yaml) | Protocol operations (Start, SubmitEvidence, Finish, …) |
| [categories.yaml](./categories.yaml) | Normative evidence categories and priorities |
| [archetypes.yaml](./archetypes.yaml) | Normative failure-mode archetype library and conformance fixtures |
| [extensions/graph-v1.yaml](./extensions/graph-v1.yaml) | Investigation graph node/edge types and queries |
| [extensions/reasoning-v1.yaml](./extensions/reasoning-v1.yaml) | Reasoning actions, cycles, reasoner interface |
| [extensions/intelligence-v1.yaml](./extensions/intelligence-v1.yaml) | Historical learning API (similarity, patterns, calibration) |

## Core entities

```
Investigation
 ├── metadata        (id, question, goal, service, time_window)
 ├── lifecycle       (state, stage, status, timestamps)
 ├── plan            (questions, evidence_requests, resolution_history)
 ├── questions[]
 ├── evidence[]
 ├── evidence_requests[]     (planner + protocol views)
 ├── graph             (nodes, edges)
 ├── hypotheses[]
 ├── findings[]        (reasoner observations)
 ├── recommendations[]
 ├── contradictions[]
 ├── timeline[]
 ├── confidence / coverage / sufficiency
 └── report            (on completion)
```

### Entity relationships

| From | Relationship | To |
| ---- | ------------ | -- |
| Investigation | contains | Question, Evidence, Hypothesis, Finding |
| Question | depends_on | Question |
| Question | requires | EvidenceCategory |
| ProtocolEvidenceRequest | fulfills | Question |
| Evidence | supports / contradicts | Hypothesis |
| Evidence | referenced_by | Finding |
| Hypothesis | competes_with | Hypothesis |
| GraphNode | linked_by | GraphEdge |
| Finding | emitted_by | Reasoner (extension) |
| Recommendation | derived_from | Finding, Hypothesis |

## Canonical lifecycle

The **normative lifecycle** is a single state machine. Implementations may expose
legacy aliases for backward compatibility.

```
started
   ↓
planning                    (stage annotation)
   ↓
collecting_evidence
   ↓
reasoning
   ↓
waiting_for_evidence  ←──────┐
   ↓                         │ (more evidence submitted)
high_confidence              │
   ↓                         │
completed ←──────────────────┘ (early finish allowed)
```

Terminal states: `completed`, `failed`.

See [lifecycle.yaml](./lifecycle.yaml) for transitions, guards, and legacy mappings.

## Protocol operations

Operations are transport-independent. Bindings map them to concrete APIs.

| Operation | Intent | MCP binding (reference impl) |
| --------- | ------ | --------------------------- |
| `StartInvestigation` | Begin investigation; return plan + initial requests | `start_investigation` |
| `SubmitEvidence` | Ingest evidence; trigger reasoning cycle | `submit_evidence` |
| `GetInvestigation` | Read current snapshot | `get_investigation_status` |
| `ResolveQuestion` | Explicitly resolve a protocol question | `resolve_question` |
| `ListOpenQuestions` | Unresolved questions by priority | `list_open_questions` |
| `FinishInvestigation` | Conclude; produce report | `finish_investigation` |
| `ExplainInvestigation` | Protocol-centric debug snapshot | `explain_investigation` |
| `ExplainReasoning` | Hypothesis/trace debug snapshot | `explain_reasoning` |

Extension operations (graph, reasoning, intelligence) are defined in
[operations.yaml](./operations.yaml) and extension modules.

## Versioning

- Spec version: `investigation-spec/v1`
- Media type (proposed): `application/vnd.investigation.v1+json`
- Breaking changes require a new major version (`v2`)
- Extensions version independently (`graph/v1`, `reasoning/v1`)

## Conformance

An implementation **conforms to Investigation Specification v1** if it:

1. Accepts evidence only in [normative categories](./categories.yaml)
2. Maintains an Investigation resource matching [model.yaml](./model.yaml)
3. Implements the **required operations** in [operations.yaml](./operations.yaml)
4. Transitions lifecycle states per [lifecycle.yaml](./lifecycle.yaml)
5. Returns a `Report` on `FinishInvestigation` matching the spec schema
6. Does not require vendor-specific connectors in the core runtime

Optional conformance tiers:

| Tier | Adds |
| ---- | ---- |
| **Core v1** | Required operations + model + lifecycle |
| **Graph v1** | Graph extension + query operations |
| **Reasoning v1** | Reasoning cycles, action validation |
| **Intelligence v1** | Similarity, patterns, calibration API |

The reference implementation passes Core + Graph + Reasoning + Intelligence tiers.
Conformance tests live in `internal/` fixtures and E2E MCP tests. See
[conformance/scenarios.md](./conformance/scenarios.md) for normative scenarios.

## Reference implementation mapping

| Spec term | Reference impl (`internal/model`) |
| --------- | -------------------------------- |
| `investigation_id` | `Session.ID` (`session_id` in JSON) |
| `state` | `InvestigationState` |
| `stage` | `InvestigationPlan.CurrentStage` |
| `status` (legacy) | `Status` — derived from `state` |
| `EvidenceRequest` | `EvidenceRequest` (planner) + `ProtocolEvidenceRequest` (protocol) |

### Known normalization (v1.0 reference impl)

The reference implementation exposes three overlapping lifecycle views. Spec
consumers should prefer **`state`** as canonical:

| Canonical `state` | Legacy `status` | Typical `stage` |
| ----------------- | --------------- | --------------- |
| `started` | `collecting_evidence` | `planning` |
| `collecting_evidence` | `collecting_evidence` | `evidence_collection` |
| `reasoning` | `collecting_evidence` | `hypothesis_evaluation` |
| `waiting_for_evidence` | `collecting_evidence` | `need_more_evidence` |
| `high_confidence` | `ready_to_conclude` | `hypothesis_evaluation` |
| `completed` | `completed` | `completed` |

Future reference impl releases may deprecate `status` in favor of `state` only.

## Contributing to the spec

Spec changes require:

1. Update YAML schemas in this directory
2. Note breaking vs non-breaking change in CHANGELOG
3. Update reference implementation or document intentional divergence
4. Add/adjust conformance scenario if behavior changes
