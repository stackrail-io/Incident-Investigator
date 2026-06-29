# Changelog

All notable changes to **Incident Investigator** are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Investigation archetype registry (Phase 1)** — `internal/archetype/` defines the `Archetype` interface and `Registry`; built-in failure modes live in `internal/archetype/builtin/`. `HeuristicHypothesisEngine` delegates scoring to the registry with no behavior change. Signal extraction moved to `internal/signals/` to support extensible archetypes without import cycles.
- **Archetype-seeded root-cause playbook (Phase 2)** — root-cause investigation questions are contributed by each built-in archetype via `SeedQuestions()` and assembled by `Registry.SeedQuestions()`; the monolithic `rootCausePlaybook` DSL string is removed.
- **Expanded root-cause playbook** — covers all failure archetypes (deployment ordering, database saturation vs lock contention, certificate expiry, DNS, memory pressure) with TITLE/DESCRIPTION/PRIORITY metadata and signal-triggered question chains.
- **Investigation Specification v1** — formal protocol (`spec/investigation-v1/`): information model, lifecycle, operations, and extensions (graph, reasoning, intelligence).
- **Semantic reasoner** uses the MCP host LLM (Claude, Codex) via MCP sampling during evidence recomputation when the client supports it.

### Fixed

- Only attach the host-LLM sampling backend when the client advertised the `sampling` capability during initialization. Previously every reasoning tool call (e.g. `submit_evidence`) issued a `sampling/createMessage` request to clients that never declared sampling support, blocking the call until it errored or the connection dropped. The semantic reasoner now skips cleanly when sampling is unavailable (debug-logged at `semantic reasoner skipped`).

## [1.0.0] - 2026-06-28

### Added

- Initial open-source release of the Incident Investigator MCP server.
- Vendor-neutral evidence model with twelve supported categories.
- **AI investigation runtime** — stateful protocol for multi-iteration investigations that continuously answers: what is known, what is still needed, whether the investigation can conclude, and why the runtime believes its conclusions.
- Stateful investigation runtime with in-memory session storage and explicit state machine (`started` → `collecting_evidence` → `reasoning` → `waiting_for_evidence` → `high_confidence` → `completed`).
- MCP tools: `start_investigation`, `submit_evidence`, `get_investigation_status`, `finish_investigation`, `explain_reasoning`, `explain_investigation`, `get_investigation_plan`, `resolve_question`, `list_open_questions`, `get_graph`, `query_graph`, `get_subgraph`, `explain_path`, `get_reasoning_cycles`, `find_similar_investigations`, `suggest_patterns`, `calibrate_confidence`.
- **Multi-Reasoner Architecture** — pluggable capability reasoners that emit declarative actions; runtime validates, merges, and applies changes.
- Reasoner interface (`Name`, `Priority`, `Supports`, `Analyze`) returning `ReasoningResult` with actions, findings, and metrics — reasoners never mutate session state.
- Declarative reasoning actions: `IncreaseHypothesisConfidence`, `CreateQuestion`, `LinkGraphNodes`, `MarkContradiction`, `ReplaceHypotheses`, `UpdateGraph`, and more.
- Hybrid orchestrator with sequential, parallel, weighted voting, priority, and consensus strategies.
- Conflict resolution for competing confidence deltas (weighted merge with full cycle logging).
- Reasoning cycles persisted on session and journal for replay via `get_reasoning_cycles`.
- Reasoner registry for plugin registration without runtime code changes.
- Built-in capability reasoners: **Temporal**, **Causal**, **Hypothesis**, **Consistency**, **Semantic** (host LLM via MCP sampling when supported).
- **Incident Intelligence** — separate subsystem that learns from completed investigations without coupling history to the runtime.
- `InvestigationArchive` interface with `MemoryArchive`; immutable `InvestigationSnapshot` and `InvestigationFingerprint` on completion.
- Deterministic `SimilarityEngine` comparing goal, questions, graph topology, timeline, evidence categories, hypotheses, and root cause (no embeddings).
- `PatternEngine` with built-in playbooks (deployment failure, certificate expiry, database saturation, retry storm) and pattern recommendations.
- `ConfidenceCalibrator` with self-explaining `CalibrationExplanation`; historical accuracy blended into session confidence.
- Lessons learned and investigation recommendations (typical missing evidence, root causes, questions) bundled with similarity results.
- Intelligence metrics (stored investigations, pattern count, similarity queries, calibrations).
- 56-snapshot intelligence test corpus validating similarity, patterns, calibration, knowledge reuse, and false-positive filtering.
- MCP tools: `find_similar_investigations`, `suggest_patterns`, `calibrate_confidence`.
- **Investigation Protocol** — question-driven investigation lifecycle with Investigation Plan as the central state object.
- **Investigation Playbooks** — declarative question definitions with hypothesis effects (default playbooks for root cause, timeline, blast radius).
- Question model with statuses (`unknown`, `waiting_for_evidence`, `partially_answered`, `answered`, `rejected`).
- Protocol evidence requests linked to specific questions with expected confidence gain.
- Question resolution engine with confirmed/rejected/insufficient/unknown outcomes.
- Question graph (question dependencies, protocol view separate from investigation graph).
- **Investigation Graph** — canonical in-memory graph replacing the evidence graph as the central data model.
- `GraphStore` interface with `MemoryGraphStore` implementation (storage-independent; no external graph database).
- Rich node types (investigation, question, evidence, hypothesis, service, deployment, …) and typed edges (supports, contradicts, causes, occurred_before, …) with confidence, weight, timestamp, reason, and evidence refs.
- Graph query engine: upstream/downstream causes, supporting evidence, contradictions, unanswered questions, service evidence, blast radius, shortest causal path, strongest path.
- Graph traversal (BFS, DFS, shortest path, topological, causal, timeline) independent from storage.
- Inference engine (temporal and semantic edge inference with confidence) and causal engine (branching/cascading failure detection).
- Graph consistency checks (orphan nodes, duplicate edges, cycles, dangling references).
- Subgraph extraction for reports (by node type or explicit ids).
- `explain_path` causal explanations with evidence per hop.
- Dynamic question generation as evidence arrives (playbook `TRIGGER` and `GENERATES` rules).
- Investigation stages: planning, question generation, evidence collection, question resolution, hypothesis evaluation, need more evidence, completed.
- Protocol metrics (total/resolved/pending questions, evidence request completion, average resolution confidence).
- Reasoning engines (all behind swappable interfaces):
  - Dynamic evidence planner
  - Competing hypothesis generation
  - Confidence scoring
  - Contradiction detection
  - Missing-evidence detection
  - Investigation graph builder (replaces evidence graph builder)
  - Timeline generator
  - Blast-radius estimator
  - Postmortem / report generator
  - Evidence sufficiency engine (central `CanAnswer` decision with blocking questions)
  - Investigation strategy engine (highest-value next 1–2 evidence items with expected confidence gain)
  - Evidence coverage engine (per-category coverage percentages)
  - Evidence importance scoring
- Reasoning orchestrator with pluggable `Reasoner` interface (`RuleReasoner` default).
- Investigation goals: `root_cause`, `timeline`, `blast_radius`, `deployment_verification`, `performance_regression`, `availability`, `custom`.
- Reasoning trace — replay how hypothesis confidence evolved.
- Investigation journal — append-only event log queryable via status and explain APIs.
- Reasoning metrics (hypothesis count, rejections, iterations, average confidence gain).
- Extended `get_investigation_status` with state, coverage, blocking questions, strategy, reasoning trace, journal, and metrics.
- Optional `goal` field on `start_investigation`.
- Realistic incident fixtures and tests (bad deployment, database outage, certificate expiry, DNS outage, Kubernetes restart loop, memory leak, retry storm).
- End-to-end MCP protocol test over an in-memory transport.
- Docker image and `docker-compose` for running the MCP server.
- Install scripts for macOS, Linux, and Windows (`scripts/install.sh`, `scripts/install.ps1`).
- Docker install helper (`scripts/install-docker.sh`).
- GoReleaser configuration and GitHub Actions release workflow for multi-platform binaries.
- `incident-investigator version` command and centralized build metadata (`internal/version`).
- CI workflow for tests on push and pull requests.
- Claude Code plugin (`.claude-plugin/marketplace.json`, investigation skill, MCP config).
- Codex plugin (`.agents/plugins/marketplace.json`, `.codex-plugin/plugin.json`).
- Plugin bundle at `plugins/incident-investigator/` with MCP launcher script.

### Fixed

- Signal classification: rollbacks and recovery events no longer misclassified as deployments or incident onset.
- Report root-cause candidates skip refuted hypotheses instead of truncating early.
- Session lifecycle: reject evidence submission after finish, duplicate evidence IDs, and invalid time windows.
- Atomic session updates via `WithSession` to prevent races between concurrent MCP calls.
- MCP input validation and clearer tool errors for missing fields and bad evidence.
- Plugin `.mcp.json` uses a relative launcher path for Codex compatibility.
- Install script release version parsing and Windows arm64 detection in `install.ps1`.

[Unreleased]: https://github.com/stackrail-io/Incident-Investigator/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/stackrail-io/Incident-Investigator/releases/tag/v1.0.0
