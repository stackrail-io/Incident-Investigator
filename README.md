<p align="center">
  <a href="https://github.com/stackrail-io/Incident-Investigator">
    <img src="docs/banner.svg" alt="Incident Investigator — vendor-neutral MCP investigation engine" width="900"/>
  </a>
</p>

<p align="center">
  <a href="https://github.com/stackrail-io/Incident-Investigator/stargazers"><img src="https://img.shields.io/github/stars/stackrail-io/Incident-Investigator?style=for-the-badge&logo=github&label=Stars&color=181717" alt="Stars"/></a>
  <a href="https://github.com/stackrail-io/Incident-Investigator/releases"><img src="https://img.shields.io/github/v/release/stackrail-io/Incident-Investigator?style=for-the-badge&label=Version&color=0ea5e9" alt="Version"/></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-22c55e?style=for-the-badge" alt="License"/></a>
  <a href="https://calendly.com/stackrail/production-audit"><img src="https://img.shields.io/badge/Book%20a%20Demo-StackRail-f97316?style=for-the-badge" alt="Book a Demo"/></a>
</p>

# Incident Investigator

An open-source **AI investigation runtime** exposed as an
[MCP](https://modelcontextprotocol.io) server.

## Why AI assistants struggle with incident investigations

Today's AI assistants can **gather** evidence — query logs, pull metrics, read
alerts — but they cannot reliably **investigate**. They lack a stateful runtime
that continuously answers four questions:

1. **What do I currently know?** — hypotheses, coverage, contradictions
2. **What do I still need?** — blocking questions, highest-value next evidence
3. **Can I answer yet?** — sufficiency, confidence thresholds
4. **Why do I believe this?** — reasoning trace, journal, confidence breakdown

Without this, assistants either stop too early (under-confident guesses) or
overstate conclusions (confident but unsupported). Incident Investigator fills
that gap: it owns **reasoning**, not data collection.

Incident Investigator does **not** connect to Kubernetes, AWS, GitHub, Slack,
Datadog, Prometheus, or any other system. It has no connectors and no vendor
SDKs. Instead, it is a **stateful investigation runtime** that guides an AI agent
through multiple reasoning iterations: it requests the highest-value evidence,
reasons over what the agent submits, tracks confidence evolution, and produces a
final report only when the investigation is sufficient.

It works with any MCP-capable assistant — Claude Code, Codex, Cursor, Goose,
OpenHands, and others.

## Core philosophy

> An investigation is not a collection of evidence — it is a collection of **questions**.
> Evidence only exists to answer questions. Questions validate hypotheses. Hypotheses answer the investigation goal.

| The AI assistant is responsible for                          | The runtime is responsible for                                                 |
| ------------------------------------------------------------ | ----------------------------------------------------------------------------- |
| Deciding which external tools to call                        | Executing the **Investigation Protocol** (playbook-driven)                    |
| Collecting logs, metrics, alerts, deployments, traces, etc.  | Maintaining the **Investigation Plan** — the brain of the investigation       |
| Answering protocol questions with evidence                   | Generating **questions** and **evidence requests** from the playbook          |
| Knowing what CloudWatch / Datadog / Kubernetes are           | Resolving questions → updating hypotheses → computing confidence                |
|                                                              | Building **Investigation Graph**, question graph, reasoning trace, and journal |

The runtime only understands **evidence categories**. It never depends on vendor
schemas.

## Architecture — Investigation Protocol

The runtime executes an **Investigation Protocol**, not just a pipeline of engines.
Every investigation follows the same hierarchy:

```
Investigation
     ↓
   Goal
     ↓
 Questions  ← driven by Investigation Playbooks
     ↓
 Evidence Requests  (linked to specific questions)
     ↓
 Evidence  (submitted by the AI assistant)
     ↓
 Question Resolution
     ↓
 Hypotheses  (updated by playbook rules, not directly by reasoners)
     ↓
 Confidence → Report
```

Protocol lifecycle:

```
Create Investigation → Generate Plan → Create Questions → Generate Evidence Requests
     → Submit Evidence → Resolve Questions → Update Hypotheses
     → Generate Additional Questions → Repeat → Investigation Complete
```

```
   AI Assistant  (gathers evidence to answer protocol questions)
         │
         ▼
   Incident Investigator MCP
         │
         ▼
   Investigation Runtime
         │
         ├── Investigation Plan  (questions, evidence requests, stage, confidence)
         ├── Investigation Playbooks  (declarative question + hypothesis rules)
         ├── **Investigation Graph**  (canonical model — evidence, questions, hypotheses, services, …)
         ├── Question Graph  (question dependencies, protocol view)
         ├── Question Resolution Engine
         ├── Sufficiency / Coverage / Strategy engines
         └── **Multi-Reasoner Orchestrator** → capability reasoners → validated actions
```

## Multi-Reasoner Architecture

Incident Investigator is **not tied to one reasoning strategy**. The runtime
orchestrates multiple independent reasoning engines. Each engine contributes
**observations as declarative actions**; the runtime validates, merges, and applies
them. The Investigation Graph remains the canonical source of truth.

> Reasoners do not investigate. Reasoners make observations. The runtime conducts
> the investigation.

## Investigation Specification v1

Incident Investigator implements **[Investigation Specification v1](spec/investigation-v1/README.md)** —
a vendor-neutral, transport-agnostic protocol for AI-assisted incident investigations
(think **OpenAPI for investigations**).

The spec defines:

- **Information model** — Investigation, Question, Evidence, Hypothesis, Finding, …
- **Canonical lifecycle** — single state machine (`started` → … → `completed`)
- **Protocol operations** — Start, SubmitEvidence, Finish, … independent of MCP
- **Extensions** — graph, multi-reasoner, incident intelligence

MCP tools in this repo are one **binding** of the spec. Other projects can implement
or consume the same protocol over REST, gRPC, or SDKs without sharing this codebase.

```
Investigation
 ├── Questions
 ├── Evidence
 ├── Graph
 ├── Hypotheses
 └── Findings
```

Run portable conformance fixtures:

```bash
go test ./internal/spec/...
```

Fixtures live in [`conformance/fixtures/`](conformance/fixtures/) — transport-agnostic
JSON scenarios any implementation can replay.

```
Investigation Runtime
        │
Investigation Graph
        │
Reasoning Orchestrator
        │
  ┌─────┴─────┬─────────────┬──────────────┐
  │           │             │              │
Temporal   Causal    Hypothesis   Consistency   Semantic
Reasoner   Reasoner  Reasoner     Reasoner      Reasoner
        │
Reasoning Action List
        │
Runtime Validation → Apply → Updated Investigation
```

### Capability reasoners (default)

| Reasoner | Priority | Responsibility |
| -------- | -------- | -------------- |
| **Temporal** | 100 | Timeline ordering, evidence planning, temporal graph edges |
| **Causal** | 90 | Graph-native root cause, blast radius, strongest paths (graph only) |
| **Hypothesis** | 80 | Competing hypotheses, confidence, coverage, strategy, graph rebuild |
| **Consistency** | 70 | Contradictions, impossible sequences, graph integrity |
| **Semantic** | 50 | Host LLM via MCP sampling (Claude/Codex); skipped when client lacks sampling |

Reasoners are classified by **what kind of reasoning they perform**, not how they
are implemented. The **Semantic** reasoner delegates to the MCP host (Claude, Codex)
via [MCP sampling](https://modelcontextprotocol.io/specification/2025-11-25/client/sampling)
when the client supports it — no vendor SDKs or API keys in the server.

### Reasoning actions

Reasoners never mutate session state directly. They emit actions such as
`IncreaseHypothesisConfidence`, `CreateQuestion`, `LinkGraphNodes`,
`MarkContradiction`, `ReplaceHypotheses`, and `UpdateGraph`. The runtime validates
every action (resolved questions, confidence limits, invalid nodes) before apply.

### Orchestration strategies

Configurable: `sequential` (default), `parallel`, `weighted_voting`, `priority`,
`consensus`. Conflicting confidence deltas are merged with weighted averages and
logged in **reasoning cycles**.

### Explainability

Every recompute persists a `ReasoningCycle` (reasoners run, actions applied/rejected,
duration). Use `get_reasoning_cycles` to replay any iteration.

## Incident Intelligence

Incident Investigator no longer performs isolated investigations. Every completed
investigation becomes **reusable knowledge**. Future investigations improve over time by:

- finding similar past incidents
- reusing investigation patterns
- calibrating confidence against historical outcomes
- recommending better evidence and questions
- accelerating root cause analysis

Historical learning lives in a separate **Incident Intelligence** subsystem. The
investigation runtime remains **stateless with respect to history** — it only asks
the Intelligence API questions; it never reads the archive directly.

```
Completed Investigation
        ↓
   Snapshot (immutable)
        ↓
 Pattern Extraction
        ↓
 Investigation Archive
        ↓
 Similarity Search ──→ Pattern Recommendation
        ↓                      ↓
        └──────────→ Investigation Runtime
```

```
                 Investigation Runtime
                        │
             Incident Intelligence API
                        │
      ┌──────────────┬──────────────┬──────────────┐
      │              │              │              │
 Similarity      Pattern Engine   Confidence Engine
                        │
             Investigation Archive
                        │
        Investigation Graph Snapshots
```

### Core components

| Component | Interface | Default implementation |
| --------- | --------- | ---------------------- |
| Archive | `InvestigationArchive` | `MemoryArchive` (in-memory) |
| Similarity | `SimilarityEngine` | `HeuristicSimilarityEngine` (deterministic) |
| Patterns | `PatternEngine` | `HeuristicPatternEngine` (built-in library + history) |
| Calibration | `ConfidenceCalibrator` | `HeuristicCalibrator` (historical accuracy blend) |

When an investigation completes, the runtime stores an immutable
`InvestigationSnapshot` (goal, root cause, timeline, knowledge graph, hypotheses,
evidence summary, fingerprint, metadata). Snapshots are indexed by
`InvestigationFingerprint` (goal, graph/timeline hashes, services, categories) to
accelerate similarity search.

**Similarity** is deterministic — no embeddings or vector databases. The engine
compares goal, question overlap, service, evidence categories, graph topology,
timeline shape, hypothesis overlap, root cause, and fingerprint partial matches.

**Pattern library** includes reusable playbooks such as Deployment Failure,
Certificate Expiry, Database Saturation, and Retry Storm. Patterns recommend
questions and expected evidence categories with confidence scores.

**Confidence calibration** blends raw hypothesis confidence with historical accuracy
for similar investigations and always returns an explanation (`CalibrationExplanation`
with sample size, correct count, and supporting history).

**Lessons learned** and **investigation recommendations** (typical missing evidence,
root causes, questions) are returned alongside similarity matches. The runtime
decides whether to use them.

Future archive backends (PostgreSQL, Neo4j, S3, object storage) plug in behind
`InvestigationArchive` without changing runtime code. There are **no** vector
databases, embedding APIs, or external AI services in the intelligence layer.

### Intelligence API

| API method | Purpose |
| ---------- | ------- |
| `FindSimilarInvestigations(...)` | Match current investigation to archived cases; returns lessons and recommendations |
| `SuggestPatterns(...)` | Recurring investigation patterns from history with recommended questions |
| `CalibrateConfidence(...)` | Adjust raw confidence using historical outcomes with explanation |

The default `MemoryService` archives snapshots when investigations finish.
Calibration also runs automatically during recompute and on `finish_investigation`.

MCP tools: `find_similar_investigations`, `suggest_patterns`, `calibrate_confidence`.

There are **no** connectors. No AWS SDK, no Kubernetes SDK, no GitHub SDK, no
Slack SDK, no Prometheus SDK, no Datadog SDK. Nothing.

Every engine is defined behind a Go interface, so the built-in heuristic
implementations can be swapped for alternatives (e.g. LLM-backed playbooks) without
touching the runtime or MCP layers.

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
| `start_investigation`      | Begin a session. Returns **plan**, **questions**, first **evidence requests**, stage, hypotheses. |
| `submit_evidence`          | Submit evidence to resolve questions. Returns resolved/new questions, updated plan, confidence delta. |
| `get_investigation_status` | Full snapshot: plan, question graph, questions, evidence requests, resolution history, metrics. |
| `get_investigation_plan`   | Return the complete investigation plan.                                                          |
| `list_open_questions`      | Unresolved questions sorted by priority.                                                         |
| `resolve_question`         | Explicitly resolve a question when evidence is conclusive.                                       |
| `explain_investigation`    | **Primary debugging tool** — plan, questions, graph, stage, trace, confidence.                   |
| `explain_reasoning`        | Hypothesis-level reasoning trace, coverage, sufficiency, journal.                                |
| `get_graph`                | Return the full investigation graph (all nodes and typed relationships).                       |
| `query_graph`              | Run graph queries: upstream causes, downstream impact, supporting evidence, contradictions, …  |
| `get_subgraph`             | Extract a filtered subgraph by node type or explicit node ids.                                   |
| `explain_path`             | Explain a causal path between two nodes with supporting evidence per hop.                        |
| `get_reasoning_cycles`     | Replay reasoning iterations: reasoners, actions, applied/rejected, timing.                         |
| `find_similar_investigations` | Find archived investigations similar to the current session; includes lessons and recommendations. |
| `suggest_patterns`         | Suggest recurring investigation patterns from historical data.                                     |
| `calibrate_confidence`     | Calibrate hypothesis confidence using history; returns original, adjusted, and explanation.          |
| `finish_investigation`     | Final report when the investigation is ready to conclude.                                         |

### Investigation goals

Set `goal` on `start_investigation` to change what the strategy engine prioritizes:

| Goal | Prioritizes |
| ---- | ----------- |
| `root_cause` (default) | Deployments, logs, alerts |
| `timeline` | Alerts, deployments, infrastructure events |
| `blast_radius` | Metrics, traces, alerts |
| `deployment_verification` | Deployments, configuration changes |
| `performance_regression` | Metrics, traces, logs |
| `availability` | Alerts, metrics, infrastructure |

### Example: `start_investigation`

Request:

```json
{
  "question": "Why did checkout fail yesterday?",
  "service": "checkout-api",
  "goal": "root_cause",
  "time_window": { "start": "2026-06-27T09:00:00Z", "end": "2026-06-27T09:30:00Z" }
}
```

Response (abridged):

```json
{
  "session_id": "inv-…",
  "status": "collecting_evidence",
  "state": "waiting_for_evidence",
  "goal": "root_cause",
  "strategy": [
    {
      "priority": 1,
      "category": "deployment_events",
      "reason": "Need to determine whether a deployment preceded the incident.",
      "expected_confidence_gain": 35
    }
  ],
  "required_evidence": [ … ]
}
```

## The investigation protocol

Each iteration follows the runtime rules:

1. **Questions** create evidence requests
2. **Evidence requests** gather evidence (via the AI assistant)
3. **Evidence** resolves questions
4. **Resolved questions** update hypotheses (via playbook rules)
5. **Hypotheses** update confidence
6. **Confidence** determines completion

```
start_investigation
   └─> generate investigation plan + initial questions
         └─> list_open_questions / get_investigation_plan
               └─> submit_evidence  (repeat — gather evidence for open questions)
                     └─> questions resolve → hypotheses update → new questions generated
                           └─> explain_investigation (inspect at any point)
                                 └─> finish_investigation when sufficient
```

Use `explain_investigation` as the primary debugging endpoint. The investigation
**journal** records every state change for replay.

Everything is **incremental** and held **in memory** for one investigation —
nothing is recomputed from a database and there is no persistence layer (yet).

## Investigation Playbooks

Playbooks are declarative investigation scripts. The runtime executes playbooks;
reasoners generate them — reasoners never manipulate hypotheses directly.

Example playbook fragment:

```
QUESTION deploy-before-errors
Did deployment happen before errors?
REQUIRES deployment_events application_logs alert_events
IF TRUE Increase hypothesis-deployment-caused 25
IF FALSE Decrease hypothesis-deployment-caused 40

QUESTION config-changed
Was configuration changed?
REQUIRES configuration_changes deployment_events
TRIGGER config
GENERATES pods-restarted
IF TRUE Increase hypothesis-configuration-change 30
```

Default playbooks ship for `root_cause`, `timeline`, and `blast_radius` goals.
See `internal/engine/playbook/` for the full definitions.

## Investigation Graph

The **Investigation Graph** is the canonical representation of an investigation.
Every engine interacts with the graph — evidence, questions, hypotheses, evidence
requests, services, deployments, and conclusions are all **nodes** connected by
**typed relationships**. The graph — not the LLM — is the source of truth.

```
Investigation Runtime
        ↓
Investigation Graph
        ↓
Planner · Reasoners · Timeline · Hypotheses · Reports · Queries
```

### Node types

`investigation`, `question`, `evidence`, `evidence_request`, `hypothesis`,
`service`, `application`, `deployment`, `pod`, `configuration`, `metric`,
`alert`, `trace`, `database`, `api`, `timeline_event`, `incident`, `conclusion`,
`recommendation`, `region`, `cluster`, `customer_impact`, `custom`

### Edge types

Each edge carries **confidence**, **weight**, **timestamp**, **reason**, and
**evidence references**:

`supports`, `contradicts`, `causes`, `triggered`, `depends_on`, `occurred_before`,
`occurred_after`, `generated`, `resolves`, `requests`, `belongs_to`, `observed_on`,
`recovered_by`, `correlates_with`

### Storage

Graph logic is **storage-independent**. The `GraphStore` interface supports
`AddNode`, `UpdateNode`, `DeleteNode`, `GetNode`, `AddEdge`, `RemoveEdge`,
`GetEdges`, and `Traverse`. Only `MemoryGraphStore` ships today — Neo4j,
Memgraph, PostgreSQL, and other backends can be added later without changing
runtime behavior.

### Graph queries

| Query kind | Example target | Returns |
| ---------- | -------------- | ------- |
| `upstream` | `checkout-api` | Deployment → Configuration → … |
| `downstream` | `checkout-api` | Impacted services and effects |
| `supporting_evidence` | hypothesis id | Evidence nodes supporting a hypothesis |
| `contradictions` | hypothesis id | Contradicting evidence |
| `unanswered_questions` | — | Open protocol questions |
| `service_evidence` | `checkout-api` | All evidence linked to a service |
| `blast_radius` | `checkout-api` | Downstream impact subgraph |
| `shortest_causal_path` | `deploy->errors` | Shortest causal chain |
| `strongest_path` | node id | Highest-confidence outgoing causal edge |

Traversal modes (BFS, DFS, shortest path, topological, causal, timeline) are
independent from storage. Subgraphs (deployment, database, hypothesis, timeline,
customer impact) feed into reports.

### Example: `explain_path`

```json
{
  "session_id": "inv-…",
  "from": "deploy",
  "to": "recovery"
}
```

Returns a hop-by-hop causal explanation where every edge references supporting
evidence — e.g. Deployment → Configuration Changed → Pods Restarted → Readiness
Failed → Traffic Shift → API Errors → Rollback → Recovery.

## Install

Pre-built binaries are published on [GitHub Releases](https://github.com/stackrail-io/Incident-Investigator/releases).
Push a `v*` tag (e.g. `v1.0.0`) to trigger the release workflow and publish installers.

### macOS / Linux

```bash
curl -fsSL https://raw.githubusercontent.com/stackrail-io/Incident-Investigator/main/scripts/install.sh | bash
```

Install to a custom directory:

```bash
INSTALL_DIR="$HOME/.local/bin" curl -fsSL .../install.sh | bash
```

Install a specific version:

```bash
INCIDENT_INVESTIGATOR_VERSION=1.0.0 curl -fsSL .../install.sh | bash
```

Verify:

```bash
incident-investigator version
```

### Windows

```powershell
irm https://raw.githubusercontent.com/stackrail-io/Incident-Investigator/main/scripts/install.ps1 | iex
```

Specific version:

```powershell
$env:INCIDENT_INVESTIGATOR_VERSION = "1.0.0"
irm https://raw.githubusercontent.com/stackrail-io/Incident-Investigator/main/scripts/install.ps1 | iex
```

### Docker

Pull and run (stdio MCP server):

```bash
docker pull ghcr.io/stackrail-io/incident-investigator:1.0.0
docker run -i --rm ghcr.io/stackrail-io/incident-investigator:1.0.0
```

Or use the install helper (pulls the image and prints MCP config):

```bash
curl -fsSL https://raw.githubusercontent.com/stackrail-io/Incident-Investigator/main/scripts/install-docker.sh | bash
```

Build locally:

```bash
docker build -t stackrail/incident-investigator:1.0.0 .
docker run -i --rm stackrail/incident-investigator:1.0.0

# Or with compose
docker compose up --build
```

### Claude Code plugin

Install from this repository's marketplace (includes MCP server + investigation skill):

```text
/plugin marketplace add stackrail-io/Incident-Investigator
/plugin install incident-investigator@incident-investigator
/reload-plugins
```

Use the bundled skill:

```text
/incident-investigator:incident-investigation
```

Validate locally before submitting to the [Claude community marketplace](https://platform.claude.com/plugins/submit):

```bash
claude plugin validate plugins/incident-investigator
```

See [plugins/incident-investigator/README.md](plugins/incident-investigator/README.md) for details.

### Codex plugin

```bash
codex plugin marketplace add stackrail-io/Incident-Investigator
codex plugin install incident-investigator --source incident-investigator
```

Browse installed plugins in the Codex TUI with `/plugins`.

The official Codex Plugin Directory does not accept third-party submissions yet; this Git-backed marketplace works today. See [OpenAI's plugin build guide](https://developers.openai.com/codex/plugins/build).

### MCP client configuration

After installing, point your MCP client at the binary or container:

**Native (macOS / Linux)**

```json
{
  "mcpServers": {
    "incident-investigator": {
      "command": "incident-investigator"
    }
  }
}
```

**Windows**

```json
{
  "mcpServers": {
    "incident-investigator": {
      "command": "C:/Users/you/AppData/Local/Programs/incident-investigator/incident-investigator.exe"
    }
  }
}
```

**Docker**

```json
{
  "mcpServers": {
    "incident-investigator": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "ghcr.io/stackrail-io/incident-investigator:1.0.0"]
    }
  }
}
```

> stdout is reserved for the MCP protocol, so all logging is written to stderr.

### Manual install

Download the archive for your platform from [Releases](https://github.com/stackrail-io/Incident-Investigator/releases), extract `incident-investigator` (or `incident-investigator.exe`), and place it on your `PATH`.

### Build from source

Requires Go 1.25+.

```bash
git clone https://github.com/stackrail-io/Incident-Investigator.git
cd Incident-Investigator
go build -o bin/incident-investigator ./cmd/incident-investigator
./bin/incident-investigator
```

Or with `go run` (no install):

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

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and [SECURITY.md](SECURITY.md) for reporting vulnerabilities.

## Build & run (developers)

```bash
go test ./...
go build -o bin/incident-investigator ./cmd/incident-investigator
./bin/incident-investigator version
./bin/incident-investigator
```

Releases are built with [GoReleaser](https://goreleaser.com/) when a `v*` tag is pushed (see `.github/workflows/release.yml`).

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
internal/spec/                     Spec v1 conformance tests (portable JSON fixtures)
spec/investigation-v1/            Investigation Specification v1 (OpenAPI-style schemas)
cmd/incident-investigator/        MCP server entrypoint (stdio)
plugins/incident-investigator/    Claude Code + Codex plugin bundle
.claude-plugin/marketplace.json   Claude marketplace catalog
.agents/plugins/marketplace.json  Codex marketplace catalog
internal/model/                   Vendor-neutral domain types (evidence, graph, …)
internal/reasoning/               Action model, validator, merger, hybrid orchestrator
internal/reasoners/               Capability reasoners (temporal, causal, hypothesis, …)
internal/intelligence/            Incident Intelligence (archive, similarity, patterns, calibration)
internal/intelligence/fixtures/   50+ completed investigation corpus for intelligence tests
internal/graph/                   Investigation graph store, queries, traversal
internal/engine/                  Planner, hypotheses, confidence, contradictions, …
internal/runtime/                 Stateful runtime + in-memory store
internal/mcpserver/               MCP tool definitions and DTOs
internal/fixtures/                Realistic incident scenarios used by tests
```

## Testing

Realistic incident fixtures (`internal/fixtures`) validate the planner, investigation
graph, timeline, hypotheses, and confidence end to end. The intelligence corpus
(`internal/intelligence/fixtures`, 56 completed snapshots) validates similarity
search, pattern extraction, confidence calibration, knowledge reuse, and false-positive
filtering.

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
| Plugin bundle | [plugins/incident-investigator/README.md](plugins/incident-investigator/README.md) |
| Claude marketplace submit | [platform.claude.com/plugins/submit](https://platform.claude.com/plugins/submit) |
| Changelog | [CHANGELOG.md](CHANGELOG.md) |
| Contributing | [CONTRIBUTING.md](CONTRIBUTING.md) |
| Code of conduct | [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) |
| Bug reports | [Open a bug report](https://github.com/stackrail-io/Incident-Investigator/issues/new?template=bug_report.yml) |
| Security | [SECURITY.md](SECURITY.md) |

## License

MIT (see `LICENSE`).
