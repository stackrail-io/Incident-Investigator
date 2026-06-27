<p align="center">
  <a href="https://github.com/stackrail/incident-investigator">
    <img src="docs/banner.svg" alt="Incident Investigator — vendor-neutral MCP investigation engine" width="900"/>
  </a>
</p>

<p align="center">
  <a href="https://github.com/stackrail/incident-investigator/stargazers"><img src="https://img.shields.io/github/stars/stackrail/incident-investigator?style=for-the-badge&logo=github&label=Stars&color=181717" alt="Stars"/></a>
  <a href="https://github.com/stackrail/incident-investigator/releases"><img src="https://img.shields.io/badge/version-0.1.0-0ea5e9?style=for-the-badge" alt="Version"/></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-22c55e?style=for-the-badge" alt="License"/></a>
  <a href="https://stackrail.io/demo"><img src="https://img.shields.io/badge/Book%20a%20Demo-StackRail-f97316?style=for-the-badge" alt="Book a Demo"/></a>
  <a href="https://stackrail.io/aurora"><img src="https://img.shields.io/badge/Try%20Aurora%20Live-→-a855f7?style=for-the-badge" alt="Try Aurora Live"/></a>
</p>

# Incident Investigator

An open-source, **vendor-neutral incident investigation engine** exposed as an
[MCP](https://modelcontextprotocol.io) server.

Incident Investigator does **not** connect to Kubernetes, AWS, GitHub, Slack,
Datadog, Prometheus, or any other system. It has no connectors and no vendor
SDKs. Instead, it is a **stateful investigation runtime** that guides an AI agent
through an investigation: it requests the evidence it needs, reasons over the
evidence the agent submits, builds an evidence graph, generates competing
hypotheses, and produces a final investigation report.

It works with any MCP-capable assistant — Claude Code, Codex, Cursor, Goose,
OpenHands, and others.

## Core philosophy

> The AI assistant gathers evidence. The Investigation Engine reasons over evidence.

| The AI assistant is responsible for                          | The engine is responsible for                                                 |
| ------------------------------------------------------------ | ----------------------------------------------------------------------------- |
| Deciding which external tools to call                        | Maintaining investigation state                                               |
| Collecting logs, metrics, alerts, deployments, traces, etc.  | Requesting additional evidence (dynamic planner)                              |
| Knowing what CloudWatch / Datadog / Kubernetes are           | Building & correlating an evidence graph                                      |
|                                                              | Generating competing hypotheses + confidence scoring                          |
|                                                              | Contradiction detection, missing-evidence detection, blast-radius estimation  |
|                                                              | Timeline generation and postmortem / report generation                       |

The engine only understands **evidence categories**. It never depends on vendor
schemas.

## Architecture

```
                MCP Server  (cmd/incident-investigator)
                     │
            Investigation Runtime  (internal/runtime)
                     │
   ┌─────────────────┼──────────────────────────────┐
 Session          Planner                         Engines (internal/engine)
 (model)        (what to collect next)            ├─ Reasoner / Signals
   │                 │                            ├─ Hypothesis Engine
 Evidence Store   Evidence Graph   Timeline       ├─ Confidence Engine
 (in-memory)         │                            ├─ Contradiction Engine
   │             Hypotheses + Confidence          ├─ Missing-Evidence Engine
 History          Report Generator                ├─ Blast-Radius Estimator
                                                  └─ Timeline / Graph builders
```

There are **no** connectors. No AWS SDK, no Kubernetes SDK, no GitHub SDK, no
Slack SDK, no Prometheus SDK, no Datadog SDK. Nothing.

Every engine is defined behind a Go interface (see `internal/runtime`'s
`Engines` struct), so the built-in heuristic implementations can be swapped for
alternatives (e.g. LLM-backed) without touching the runtime or MCP layers.

## Evidence categories

The engine recognizes only these vendor-neutral categories:

`application_logs`, `infrastructure_events`, `deployment_events`, `alert_events`,
`metrics`, `trace_events`, `configuration_changes`, `network_events`,
`database_events`, `security_events`, `human_context`, `custom`.

## Evidence model

Every object the assistant submits is normalized into:

```json
{
  "id": "uuid",
  "timestamp": "2026-06-27T09:01:00Z",
  "category": "application_logs",
  "source": "provided_by_client",
  "entity": "checkout-api",
  "summary": "Database connection timeout",
  "payload": {}
}
```

`payload` is opaque — put anything in it. The engine never parses vendor formats;
it reasons over `category`, `timestamp`, `entity`, and `summary`, and surfaces
`payload` keys like `region`, `customer`, and `api`/`endpoint` for blast-radius
estimation.

## MCP tools

| Tool                       | Purpose                                                                                          |
| -------------------------- | ------------------------------------------------------------------------------------------------ |
| `start_investigation`      | Begin a session. Returns `session_id`, `status`, and the `required_evidence` to collect first.   |
| `submit_evidence`          | Add evidence. Returns updated `progress`, `confidence`, `missing_evidence`, `next_required_evidence`, `updated_hypotheses`, `contradictions`. |
| `get_investigation_status` | Current hypotheses, confidence, graph, timeline, missing evidence, blast radius, progress.       |
| `finish_investigation`     | Final report: executive summary, timeline, evidence, hypotheses, root-cause candidates, graph, blast radius, contradictions, missing evidence, recommendations, confidence, and a markdown postmortem. |

### Example: `start_investigation`

Request:

```json
{
  "question": "Why did checkout fail yesterday?",
  "service": "checkout-api",
  "time_window": { "start": "2026-06-27T09:00:00Z", "end": "2026-06-27T09:30:00Z" }
}
```

Response (abridged):

```json
{
  "session_id": "inv-…",
  "status": "collecting_evidence",
  "required_evidence": [
    { "category": "deployment_events", "priority": "high", "reason": "Need to determine whether a deployment preceded the incident." },
    { "category": "application_logs",  "priority": "high", "reason": "Need application logs to characterize the failure mode." },
    { "category": "metrics",           "priority": "medium" }
  ]
}
```

## The investigation lifecycle

```
start_investigation
   └─> planner determines required evidence
         └─> submit_evidence  (repeat)
               └─> planner re-evaluates, hypotheses & confidence update
                     └─> confidence sufficient?  ── no ──> keep collecting
                                                  └─ yes ─> finish_investigation
```

Everything is **incremental** and held **in memory** for one investigation —
nothing is recomputed from a database and there is no persistence layer (yet).

## Build & run

Requires Go 1.25+.

```bash
# Build
go build -o bin/incident-investigator ./cmd/incident-investigator

# Run the MCP server (speaks MCP over stdio; logs go to stderr)
./bin/incident-investigator
```

### Use it from an MCP client

The server communicates over **stdio**. Point any MCP client at the binary.

Claude Code / Cursor style `mcp.json`:

```json
{
  "mcpServers": {
    "incident-investigator": {
      "command": "/absolute/path/to/bin/incident-investigator"
    }
  }
}
```

Or with `go run`:

```json
{
  "mcpServers": {
    "incident-investigator": {
      "command": "go",
      "args": ["run", "github.com/stackrail/incident-investigator/cmd/incident-investigator"]
    }
  }
}
```

> stdout is reserved for the MCP protocol, so all logging is written to stderr.

### Docker

```bash
# Build image
docker build -t stackrail/incident-investigator:0.1.0 .

# Run (stdio MCP server)
docker run -i --rm stackrail/incident-investigator:0.1.0

# Or with compose
docker compose up --build
```

Point your MCP client at the container (example for Cursor / Claude Code):

```json
{
  "mcpServers": {
    "incident-investigator": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "stackrail/incident-investigator:0.1.0"]
    }
  }
}
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and [SECURITY.md](SECURITY.md) for reporting vulnerabilities.

## How the reasoning works

The built-in engines are deterministic, rule-based heuristics that operate purely
on the abstract signals extracted from evidence (`internal/engine/signals.go`):

- **Planner** — starts with a baseline (deployments, logs, alerts, metrics) and
  *dynamically* expands the request set as evidence arrives (e.g. once a
  deployment appears, it asks for configuration changes and traces).
- **Hypothesis engine** — always produces *competing* hypotheses
  (deployment-caused, deployment-unrelated, database saturation, configuration
  change, network/DNS, certificate expiry, resource exhaustion, retry storm,
  unknown), scored and normalized into a probability field.
- **Confidence engine** — rises with independent agreement, multi-category
  corroboration, and clean temporal ordering; falls with contradictions and
  missing critical evidence.
- **Contradiction engine** — flags impossible/inconsistent sequences such as a
  deployment timestamped *after* the incident began, recovery before onset, and
  duplicate evidence.
- **Blast-radius estimator** — derives affected services/regions/customers/APIs
  from entities and well-known payload keys.

## Project layout

```
cmd/incident-investigator/   MCP server entrypoint (stdio)
internal/model/              Vendor-neutral domain types (evidence, graph, …)
internal/engine/             Planner, hypotheses, confidence, contradictions, …
internal/runtime/            Stateful runtime + in-memory store
internal/mcpserver/          MCP tool definitions and DTOs
internal/fixtures/           Realistic incident scenarios used by tests
```

## Testing

Realistic incident fixtures (`internal/fixtures`) validate the planner, evidence
graph, timeline, hypotheses, and confidence end to end:

- Bad deployment
- Database outage
- Certificate expiry
- DNS outage
- Kubernetes restart loop
- Memory leak
- Retry storm

```bash
go test ./...
```

There is also an end-to-end MCP test that drives the real protocol over an
in-memory transport (`internal/mcpserver/server_test.go`).

## Out of scope (intentionally not implemented)

Connectors, authentication, RBAC, a UI, and streaming telemetry are explicitly
**out of scope**. The engine's only job is to reason over evidence.

## Community

| Resource | Link |
| -------- | ---- |
| Changelog | [CHANGELOG.md](CHANGELOG.md) |
| Contributing | [CONTRIBUTING.md](CONTRIBUTING.md) |
| Code of conduct | [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) |
| Bug reports | [Open a bug report](https://github.com/stackrail/incident-investigator/issues/new?template=bug_report.yml) |
| Security | [SECURITY.md](SECURITY.md) |

## License

MIT (see `LICENSE`).
