# Development Guide

This guide explains how investigations flow through the codebase and how to extend the runtime without modifying core logic.

## Prerequisites

- Go 1.25+
- Familiarity with [architecture](./architecture.md) and [design principles](./design-principles.md)

```bash
git clone https://github.com/stackrail-io/Incident-Investigator.git
cd Incident-Investigator
go test ./...
go build -o bin/incident-investigator ./cmd/incident-investigator
```

## How investigations flow

### 1. Start

`runtime.Start` creates a `Session`, initializes the investigation plan from the goal's playbook, and runs the first `recompute`.

```
StartInput → Session (evidence=[], state=started)
          → recompute → plan, questions, evidence requests, hypotheses
```

### 2. Submit evidence

`runtime.Submit` appends vendor-neutral `Evidence`, deduplicates by id, and recomputes.

```
Evidence[] → validate categories → append → recompute
```

### 3. Recompute (core loop)

See [architecture.md](./architecture.md#recompute-loop). Key files:

| Step | Package | Entry |
| ---- | ------- | ----- |
| Reasoning | `internal/reasoning` | `HybridOrchestrator.Execute` |
| Protocol | `internal/engine/protocol` | `Engine.Run` |
| Hypotheses | `internal/engine` | `HeuristicHypothesisEngine` |
| Confidence | `internal/engine` | `HeuristicConfidenceScorer` |
| Graph | `internal/graph` | `InvestigationGraph` sync |
| State | `internal/engine` | `StateMachine` |

### 4. Resolve questions

`runtime.ResolveQuestion` or automatic resolution during protocol `Run` updates hypothesis confidence via playbook effects.

### 5. Finish

`runtime.Finish` generates a report, marks the session completed, and optionally archives to intelligence.

## Writing a reasoner

1. Implement `reasoning.Reasoner` (or `extension.Reasoner` contract in `pkg/extension`):

```go
type MyReasoner struct{}

func (MyReasoner) Name() string { return "my-reasoner" }
func (MyReasoner) Priority() int { return 50 }
func (MyReasoner) Supports(s *model.Session) bool { return s.Goal == model.GoalRootCause }
func (MyReasoner) Analyze(ctx context.Context, inv *reasoning.Investigation) (*model.ReasoningResult, error) {
    // Read inv.Session, inv.Graph — never mutate
    return &model.ReasoningResult{
        Actions: []model.ReasoningAction{ /* declarative */ },
    }, nil
}
```

2. Register:

```go
reg := extension.NewReasonerRegistry()
reg.Register(MyReasoner{})
rt := runtime.New(runtime.WithReasonerRegistry(reg))
```

3. Add tests in `internal/reasoning` or your package using `internal/fixtures`.

**Rules:** Return actions only. Do not write to session. Use `Supports` to limit scope.

## Creating playbooks

Playbooks live in `internal/engine/playbook`. Define questions declaratively:

```go
pb := &playbook.Playbook{
    ID:   "my-playbook",
    Goal: model.GoalTimeline,
    Questions: []playbook.PlaybookQuestion{{
        ID:       "timeline-start",
        Title:    "When did symptoms begin?",
        Priority: 90,
        Requires: []model.Category{model.CategoryApplicationLogs},
    }},
}
```

Register via `extension.PlaybookRegistry` or extend `playbook.ForGoal`.

Parse DSL with `playbook.Parse` for string-based definitions.

## Adding archetypes

1. Implement `archetype.Archetype` in a new file under `internal/archetype/builtin/` (or your module).
2. Register in `builtin/register.go` or via `extension.ArchetypeRegistry`.
3. Add entry to `spec/investigation-v1/archetypes.yaml`.
4. Add conformance fixture in `spec/investigation-v1/conformance/archetype-fixtures/`.
5. Run `go test ./internal/spec/...`.

Archetypes provide: `Score`, `Applicable`, `SeedQuestions`, `ExpectedEvidence`.

## How graphs work

- `graph.InvestigationGraph` wraps `graph.Store` (default: in-memory).
- Graph builder (`internal/engine/graph.go`) syncs evidence and hypotheses into nodes after recompute.
- Query APIs: `GetGraph`, `QueryGraph`, `GetSubgraph`, `ExplainPath` on runtime.
- Custom persistence: implement `graph.Store`, pass to graph construction (advanced).

## How intelligence works

Optional layer in `internal/intelligence`:

| Component | Role |
| --------- | ---- |
| `InvestigationArchive` | Stores snapshots on finish |
| `SimilarityEngine` | Finds similar past investigations |
| `PatternEngine` | Suggests patterns from library + history |
| `ConfidenceCalibrator` | Adjusts confidence using historical outcomes |

Disable with `intelligence.Noop()` or `runtime.WithIntelligence(intelligence.Noop())`.

## Events

Subscribe to investigation lifecycle:

```go
bus := events.NewBus()
bus.Subscribe(func(e events.Event) {
    log.Printf("%s %s", e.Type, e.InvestigationID)
})
rt := runtime.New(runtime.WithEventBus(bus))
```

Event types: `EvidenceAdded`, `QuestionResolved`, `HypothesisUpdated`, `ReasoningCompleted`, `InvestigationCompleted`, etc.

## Explainability

```go
import "github.com/stackrail/incident-investigator/internal/explain"

exp := explain.WhyHypothesis(session, "hypothesis-deployment-caused")
inc := explain.WhyIncomplete(session)
```

Runtime also exposes `Explain` and `ExplainInvestigation` for full snapshots.

## Testing

| Layer | Command | Focus |
| ----- | ------- | ----- |
| Unit | `go test ./internal/engine/...` | Engines, protocol |
| Spec conformance | `go test ./internal/spec/...` | 32 archetype YAML fixtures |
| Examples | `go test ./examples/...` | End-to-end example investigations |
| MCP E2E | `go test ./internal/mcpserver/...` | Tool handlers |
| Race | `go test -race ./...` | Concurrency |

### Adding a conformance fixture

1. Edit or generate YAML in `spec/investigation-v1/conformance/archetype-fixtures/`.
2. Mark `hand_tuned: true` to skip generator overwrite.
3. `go test ./internal/spec/ -run Conformance`

### Example investigations

See [`examples/README.md`](../examples/README.md). Each example is replayed in `examples/examples_test.go`.

## Project layout

| Path | Responsibility |
| ---- | -------------- |
| `cmd/incident-investigator/` | MCP server binary |
| `pkg/extension/` | Public extension contracts and registries |
| `pkg/export/` | Report and graph exporters |
| `internal/runtime/` | Session lifecycle, recompute orchestration |
| `internal/engine/` | Heuristic engines, protocol, playbooks |
| `internal/reasoning/` | Reasoner orchestration, action validation |
| `internal/reasoners/` | Built-in reasoners |
| `internal/graph/` | Investigation graph store and queries |
| `internal/archetype/` | Failure-mode library |
| `internal/intelligence/` | Optional historical learning |
| `internal/events/` | Internal event bus |
| `internal/explain/` | Explainability helpers |
| `spec/investigation-v1/` | Normative specification |
| `examples/` | Documented investigations + regression tests |

## Changelog and releases

Update `CHANGELOG.md` under **Unreleased** for user-visible changes. See [ROADMAP.md](../ROADMAP.md) for planned work.
