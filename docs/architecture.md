# Architecture

Incident Investigator is a **vendor-neutral investigation runtime**. LLMs and MCP clients collect evidence; the runtime conducts the investigation.

This document describes how components fit together, who owns state, and where extension points live.

## System overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│  MCP client / AI assistant                                              │
│  (Claude, Codex, Cursor, custom agents)                                 │
│                                                                         │
│  Responsibility: gather evidence from external systems                  │
└───────────────────────────────┬─────────────────────────────────────────┘
                                │ evidence, question answers
                                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  Evidence collection (out of scope for this project)                    │
│  Logs, metrics, deployments, traces, tickets — any vendor schema        │
└───────────────────────────────┬─────────────────────────────────────────┘
                                │ vendor-neutral Evidence payloads
                                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  Investigation Protocol                                                 │
│  Goal → Questions → Evidence Requests → Resolution → Hypotheses         │
└───────────────────────────────┬─────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  Question Engine                                                        │
│  Playbooks, archetype seeds, open questions, sufficiency                  │
└───────────────────────────────┬─────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  Investigation Graph (canonical state view)                             │
│  Evidence, hypotheses, timeline, causal links                           │
└───────────────────────────────┬─────────────────────────────────────────┘
                                │ read-only Investigation view
                                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  Reasoning Runtime (multi-reasoner)                                     │
│  Reasoners propose declarative actions; runtime validates and applies     │
└───────────────────────────────┬─────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  Incident Intelligence (optional)                                       │
│  Archive, similarity, patterns, confidence calibration                  │
└───────────────────────────────┬─────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  Findings & Report                                                      │
│  Exporters: Markdown, JSON, Mermaid, GraphML, PlantUML                 │
└─────────────────────────────────────────────────────────────────────────┘
```

## Recompute loop

Every mutation (`Start`, `Submit`, `ResolveQuestion`, `Finish`) triggers a **recompute**:

1. **Reasoning orchestrator** — run registered reasoners; validate and apply `ReasoningAction`s
2. **Protocol engine** — resolve questions from evidence signals; update playbook effects
3. **Heuristic engines** — hypotheses, confidence, contradictions, coverage, sufficiency
4. **Graph builder** — sync graph nodes/edges from session state
5. **Intelligence** (optional) — calibrate confidence from historical archive
6. **State machine** — transition investigation lifecycle state

Reasoners and protocol logic **never write directly to each other**. The runtime owns the session and applies all changes.

## Ownership boundaries

| Component | Owns | Must not |
| --------- | ---- | -------- |
| **Runtime** (`internal/runtime`) | Session store, recompute orchestration, validation | Embed vendor SDKs or call external APIs |
| **Investigation Graph** (`internal/graph`) | Node/edge persistence, traversal, path explanation | Mutate hypotheses or confidence directly |
| **Reasoners** (`internal/reasoners`) | Proposing `ReasoningAction`s from read-only views | Mutate `Session`, graph, or evidence |
| **Protocol engine** (`internal/engine/protocol`) | Question resolution, playbook effects on hypotheses | Bypass runtime validation |
| **Playbooks** (`internal/engine/playbook`) | Declarative question definitions | Execute side effects outside runtime |
| **Archetypes** (`internal/archetype`) | Failure-mode templates, scoring, seed questions | Store investigation state |
| **Intelligence** (`internal/intelligence`) | Historical archive, similarity, patterns | Block core investigation when disabled |
| **MCP server** (`internal/mcpserver`) | Transport, DTO mapping | Business logic beyond validation |

**The graph is the canonical structural view of an investigation.** Hypotheses, evidence, and timeline events are mirrored into graph nodes; queries and exporters read from graph + session.

**Evidence is immutable.** Once submitted, evidence is never edited—only appended (deduplicated by id).

**Runtime validates everything.** Reasoning actions pass through a validator before the applier touches session state.

## Package dependency direction

Dependencies flow **inward** toward domain types, never upward toward transport:

```
cmd/incident-investigator
        │
        ▼
internal/mcpserver ──► internal/runtime ──► internal/engine
        │                    │                    │
        │                    ├── internal/graph   │
        │                    ├── internal/reasoning│
        │                    └── internal/intelligence
        │
        ▼
internal/model  ◄── internal/archetype, internal/signals, internal/fixtures
        ▲
pkg/extension, pkg/export  (public contracts; reference impl in internal/)
```

## Extension model

Third-party code extends the runtime by **registering providers**—not by forking core logic.

| Extension | Register via | Package |
| --------- | ------------ | ------- |
| Reasoner | `extension.ReasonerRegistry` → `runtime.WithReasonerRegistry` | `pkg/extension` |
| Archetype | `extension.ArchetypeRegistry` → hypothesis engine wiring | `pkg/extension` |
| Playbook | `extension.PlaybookRegistry` | `pkg/extension` |
| Report | `extension.ReportRegistry` → `runtime.WithReportGenerator` | `pkg/extension` |
| Graph store | `extension.GraphStore` implementation | `pkg/extension` |
| Intelligence archive | `extension.InvestigationArchive` | `pkg/extension` |
| Pattern / similarity | `extension.PatternRegistry`, `extension.SimilarityRegistry` | `pkg/extension` |
| Exporters | `export.Registry` | `pkg/export` |

See [extension-apis.md](./extension-apis.md) and [development.md](./development.md).

## Event model

The runtime publishes typed events on an internal bus (`internal/events`). Components may subscribe to react to investigation lifecycle changes without tight coupling.

Events include: `EvidenceAdded`, `QuestionCreated`, `QuestionResolved`, `HypothesisCreated`, `HypothesisUpdated`, `ReasoningCompleted`, `InvestigationCompleted`, `PatternMatched`.

## Specification

The normative contract lives in [`spec/investigation-v1/`](../spec/investigation-v1/). Implementations in any language should conform to that specification; the Go runtime is one reference implementation.

## Related documents

- [Philosophy](./philosophy.md)
- [Design principles](./design-principles.md)
- [Extension APIs](./extension-apis.md)
- [Development guide](./development.md)
- [Architecture Decision Records](./adr/README.md)
- [Investigation Specification](../spec/investigation-v1/SPECIFICATION.md)
