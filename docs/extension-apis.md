# Extension APIs

Public extension contracts live in [`pkg/extension`](../pkg/extension/). The reference implementation wires built-in providers in `internal/`; third parties register replacements without modifying runtime source.

## Quick start

```go
import (
    "github.com/stackrail/incident-investigator/internal/runtime"
    "github.com/stackrail/incident-investigator/pkg/extension"
)

reg := extension.NewReasonerRegistry()
reg.Register(myReasoner)

rt := runtime.New(
    runtime.WithReasonerRegistry(reg),
)
```

## Interfaces

### Reasoner

**Purpose:** Propose observations as declarative `ReasoningAction`s from a read-only investigation view.

**Lifecycle:**

1. Runtime builds `Investigation` view (session + graph + signals)
2. For each registered reasoner where `Supports(session)` is true, call `Analyze`
3. Validator checks actions; applier merges into session
4. Runtime publishes `ReasoningCompleted` event

**Ownership:** Reasoner must not mutate session, graph, or evidence.

### GraphStore

**Purpose:** Persist investigation graph nodes and edges.

**Lifecycle:** Graph builder adds/updates nodes after each recompute. Query APIs traverse the store.

**Ownership:** Store owns graph structure only—not hypotheses list or confidence on session.

### InvestigationArchive

**Purpose:** Store completed investigation snapshots for intelligence features.

**Lifecycle:** On `Finish`, runtime optionally archives a snapshot. Similarity and pattern engines read from archive.

**Ownership:** Archive is separate from active session store.

### PatternProvider / PatternEngine

**Purpose:** Suggest investigation patterns from library and historical data.

**Lifecycle:** Called on demand via `SuggestPatterns` MCP tool or intelligence API.

### PlaybookProvider

**Purpose:** Supply declarative playbooks for investigation goals.

**Lifecycle:** Protocol engine loads playbook at investigation start; questions materialize from playbook + archetype seeds.

### ArchetypeProvider

**Purpose:** Failure-mode templates for hypothesis scoring and seed questions.

**Lifecycle:** Hypothesis engine scores all applicable archetypes each recompute; root-cause playbook merges seed questions.

### ReportGenerator

**Purpose:** Assemble final `Report` on `Finish`.

**Lifecycle:** Called once when investigation completes; does not mutate session.

### ConfidenceProvider

**Purpose:** Score session confidence from hypotheses, contradictions, and coverage.

**Lifecycle:** Invoked each recompute after hypothesis updates.

### SimilarityProvider

**Purpose:** Find historically similar investigations from archive.

**Lifecycle:** On-demand; optional at finish for calibration.

## Registries

| Registry | Package type | Default wiring |
| -------- | ------------ | -------------- |
| `ReasonerRegistry` | `pkg/extension` | `internal/reasoners.DefaultRegistry` |
| `ArchetypeRegistry` | `pkg/extension` | `internal/archetype/builtin.DefaultRegistry` |
| `PlaybookRegistry` | `pkg/extension` | `internal/engine/playbook.ForGoal` |
| `ReportRegistry` | `pkg/extension` | `internal/engine` heuristic generator |
| `PatternRegistry` | `pkg/extension` | `internal/intelligence` |
| `SimilarityRegistry` | `pkg/extension` | `internal/intelligence` |

Registries use **register-by-name** semantics: later registration replaces earlier entries with the same name.

## Runtime options

| Option | Effect |
| ------ | ------ |
| `WithReasonerRegistry` | Replace reasoner set |
| `WithEngines` | Replace heuristic engines (planner, confidence, report, …) |
| `WithOrchestrator` | Replace reasoning orchestrator |
| `WithEventBus` | Subscribe to investigation events |
| `WithIntelligence` | Replace intelligence implementation |

## Exporters

Report and graph export live in [`pkg/export`](../pkg/export/). Exporters are decoupled from runtime—they accept `export.InvestigationSnapshot` built from a completed or in-progress session.

Formats: Markdown, JSON, Mermaid, GraphML, PlantUML.

## Conformance

Implementing the Investigation Specification in another language requires:

1. [`spec/investigation-v1/SPECIFICATION.md`](../spec/investigation-v1/SPECIFICATION.md) — entity definitions
2. [`spec/investigation-v1/conformance/`](../spec/investigation-v1/conformance/) — replay fixtures
3. [`examples/`](../examples/) — end-to-end expected outputs
