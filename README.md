<p align="center">
  <a href="https://github.com/stackrail-io/Incident-Investigator">
    <img src="docs/banner.svg" alt="Incident Investigator" width="900"/>
  </a>
</p>

<p align="center">
  <a href="https://github.com/stackrail-io/Incident-Investigator/stargazers"><img src="https://img.shields.io/github/stars/stackrail-io/Incident-Investigator?style=for-the-badge&logo=github&label=Stars&color=181717" alt="Stars"/></a>
  <a href="https://github.com/stackrail-io/Incident-Investigator/releases"><img src="https://img.shields.io/github/v/release/stackrail-io/Incident-Investigator?style=for-the-badge&label=Version&color=0ea5e9" alt="Version"/></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-22c55e?style=for-the-badge" alt="License"/></a>
  <a href="spec/investigation-v1/SPECIFICATION.md"><img src="https://img.shields.io/badge/spec-v1-8b5cf6?style=for-the-badge" alt="Spec v1"/></a>
</p>

# Incident Investigator

**LLMs collect evidence. Incident Investigator conducts investigations.**

An open-source, vendor-neutral **investigation runtime** exposed as an [MCP](https://modelcontextprotocol.io) server. It is not another AI agent—it is the stateful engine that turns evidence into structured conclusions.

## Why this exists

AI assistants are good at calling tools—querying logs, pulling metrics, reading alerts. They are unreliable at **investigating**: maintaining competing hypotheses, knowing what evidence is still missing, tracking confidence over time, and explaining conclusions.

Incident Investigator owns that runtime:

| Layer | Responsibility |
| ----- | ---------------- |
| **Investigation Protocol** | Goal → questions → evidence requests → resolution → hypotheses |
| **Question Engine** | Playbooks and archetypes drive what to ask next |
| **Investigation Graph** | Canonical structure of evidence, causes, and timeline |
| **Multi-Reasoner Runtime** | Stateless reasoners propose actions; runtime applies them |
| **Incident Intelligence** | Optional archive, similarity, patterns, calibration |

The assistant gathers evidence. The runtime investigates.

## How this differs

### Not an AI agent

Agents decide which APIs to call and improvise reasoning in prompts. Incident Investigator is a **deterministic investigation framework** with a formal spec, conformance tests, and explainable state.

### Why not just prompt Claude?

Prompt-only investigations lack:

- Immutable evidence and audit journal
- Structured question graph with sufficiency rules
- Competing hypotheses with normalized confidence
- Replayable conformance scenarios (32 archetypes)

Prompts are not a substitute for a runtime that enforces the protocol.

### Why not LangGraph?

LangGraph orchestrates arbitrary LLM workflows. Incident Investigator implements a **domain-specific investigation protocol**—questions, evidence categories, hypothesis fields, graph queries—with or without an LLM. Reasoners are pluggable; the protocol is fixed.

### Why MCP?

MCP is the integration surface assistants already use. The [Investigation Specification](spec/investigation-v1/SPECIFICATION.md) is transport-agnostic; MCP is the reference binding.

## Architecture

```
Evidence (from assistant) → Protocol → Questions → Graph → Reasoners → Intelligence → Findings
```

```
┌──────────────┐     ┌─────────────────────────────────────────┐
│ MCP client   │────►│ Incident Investigator runtime             │
│ (assistant)  │     │  protocol · graph · reasoners · report  │
└──────────────┘     └─────────────────────────────────────────┘
        │                          ▲
        │ collects                 │ vendor-neutral Evidence
        ▼                          │
   Datadog, K8s, …            (no connectors in core)
```

**Ownership:** The runtime owns session state. Reasoners never mutate state. The graph is canonical. Evidence is append-only.

Full design: [docs/architecture.md](docs/architecture.md) · [philosophy](docs/philosophy.md) · [design principles](docs/design-principles.md) · [ADRs](docs/adr/README.md)

## Extend without forking

Register providers—do not patch core:

```go
reg := extension.NewReasonerRegistry()
// reg.Register(myReasoner) — implement internal/reasoning.Reasoner in-tree
rt := runtime.New(runtime.WithReasonerRegistry(myRegistry))
```

| Extension | Registry |
| --------- | -------- |
| Reasoner | `extension.ReasonerRegistry` |
| Archetype | `extension.ArchetypeRegistry` |
| Playbook | `extension.PlaybookRegistry` |
| Report | `extension.ReportRegistry` |
| Graph store | `extension.GraphStoreRegistry` |
| Intelligence | `runtime.WithIntelligence` |

Docs: [extension APIs](docs/extension-apis.md) · [development guide](docs/development.md)

## Investigation Specification v1

Implementation-independent contract in [`spec/investigation-v1/`](spec/investigation-v1/):

- [SPECIFICATION.md](spec/investigation-v1/SPECIFICATION.md) — entities, lifecycle, protocol
- [archetypes.yaml](spec/investigation-v1/archetypes.yaml) — 32 failure-mode archetypes
- [conformance fixtures](spec/investigation-v1/conformance/archetype-fixtures/) — replay scenarios

Another language can implement the spec and pass the same fixtures.

## Examples

Seven complete investigations with regression tests:

```bash
go test ./examples/...
```

| Example | Scenario |
| ------- | -------- |
| [deployment-failure](examples/deployment-failure/) | Deploy preceded errors |
| [certificate-expiry](examples/certificate-expiry/) | TLS certificate expired |
| [dns-outage](examples/dns-outage/) | DNS resolution failure |
| [retry-storm](examples/retry-storm/) | Retry amplification |
| [database-deadlock](examples/database-deadlock/) | Lock contention |
| [memory-leak](examples/memory-leak/) | OOM / resource exhaustion |
| [regional-outage](examples/regional-outage/) | Availability zone failure |

## Quick start

```bash
# Install
go install github.com/stackrail/incident-investigator/cmd/incident-investigator@latest

# Or from source
git clone https://github.com/stackrail-io/Incident-Investigator.git
cd Incident-Investigator
go test ./...
go build -o bin/incident-investigator ./cmd/incident-investigator
```

Configure as an MCP server in Claude Code, Codex, or Cursor. See [plugins/incident-investigator/](plugins/incident-investigator/).

## MCP tools (reference binding)

| Tool | Protocol operation |
| ---- | ------------------ |
| `start_investigation` | Start |
| `submit_evidence` | Submit |
| `finish_investigation` | Finish |
| `explain_investigation` | Debug snapshot |
| `get_graph` / `query_graph` | Graph extension |
| `find_similar_investigations` | Intelligence extension |

Full tool list and payloads: run `incident-investigator help` or see prior README sections in git history for MCP examples.

## Project layout

| Path | Role |
| ---- | ---- |
| `cmd/incident-investigator/` | MCP server binary |
| `pkg/extension/` | Public extension contracts and registries |
| `pkg/export/` | Markdown, JSON, Mermaid, GraphML, PlantUML exporters |
| `internal/runtime/` | Session lifecycle and recompute orchestration |
| `internal/engine/` | Protocol, playbooks, heuristic engines |
| `internal/reasoning/` | Reasoner orchestration |
| `internal/graph/` | Investigation graph |
| `internal/archetype/` | Failure-mode library (32 built-ins) |
| `internal/intelligence/` | Optional historical learning |
| `spec/investigation-v1/` | Normative specification |
| `examples/` | Documented investigations + E2E tests |
| `docs/` | Architecture, ADRs, development guide |

## Testing

```bash
go test ./...              # unit + conformance + examples
go test ./internal/spec/... # 32 archetype fixtures
go test ./examples/...      # example investigations
go test -race ./...
```

## Explainability

Every decision has an explanation path:

```go
explain.WhyHypothesis(session, "hypothesis-deployment-caused")
explain.WhyConfidence(session)
explain.WhyIncomplete(session)
explain.WhyMoreEvidence(session)
```

Runtime APIs: `explain_investigation`, `explain`, graph `explain_path`.

## Out of scope

No Kubernetes SDKs, Datadog, Grafana, Slack, databases, authentication, or hosted service in core. Connectors belong in the assistant layer.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) and [ROADMAP.md](ROADMAP.md).

## License

MIT — see [LICENSE](LICENSE).
