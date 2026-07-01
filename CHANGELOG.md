# Changelog

All notable changes to **Incident Investigator** are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-07-01

> **Tag note:** The `v1.0.0` Git tag was deleted and recreated on current `main`
> (`7edf131`). It previously pointed at the 2026-06-28 initial-release commit and
> did not include the archetype library, v1.1 documentation, README refresh, or
> negation-aware evidence fixes. Downloaders and release assets should use the
> tag as it exists on `main` today.

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
- Reasoning engines (all behind swappable interfaces): planner, hypotheses, confidence, contradictions, missing evidence, graph, timeline, blast radius, report, sufficiency, strategy, coverage, importance.
- Investigation goals: `root_cause`, `timeline`, `blast_radius`, `deployment_verification`, `performance_regression`, `availability`, `custom`.
- Reasoning trace, investigation journal, and reasoning metrics.
- Realistic incident fixtures and end-to-end MCP protocol tests.
- Docker image, install scripts, GoReleaser, CI, Claude/Codex plugins.
- **Full 30-category archetype library** — 32 built-in archetypes with `spec/investigation-v1/archetypes.yaml` contract and per-archetype YAML conformance fixtures.
- **Investigation Specification v1** — formal protocol (`spec/investigation-v1/`): information model, lifecycle, operations, extensions, and `SPECIFICATION.md` overview.
- **v1.1 architecture documentation** — `docs/architecture.md`, `docs/philosophy.md`, `docs/design-principles.md`, `docs/development.md`, `docs/extension-apis.md`, `docs/README.md`, eight ADRs, `ARCHITECTURE.md`, `ROADMAP.md`.
- **Public extension package** — `pkg/extension` provider registries; **exporters** — `pkg/export` (Markdown, JSON, Mermaid, GraphML, PlantUML).
- **Example investigations** — `examples/` with seven scenarios and `go test ./examples/...` regression coverage.
- **Event bus** — `internal/events`; **explainability helpers** — `internal/explain`.
- **Negation-aware evidence signals** — ruled-out deployment and configuration findings (`new_deploy: false`, "no deployment", etc.) do not boost deploy-caused or config-change hypotheses.
- **Semantic prompt payloads** — LLM semantic reasoner receives evidence `payload` fields, not only summaries.

### Changed

- **README** — two-minute architecture-first overview with hero diagram; depth moved to `/docs`.
- **Network/DNS hypothesis split** — `hypothesis-network-dns` → `hypothesis-network-failure` + `hypothesis-dns-failure`.
- **Confidence dilution** — 32 archetypes normalize confidence lower; conformance uses `min_lead_margin` and `must_not_lead`.
- **Conformance fixtures** — YAML per archetype; legacy `bad-deployment.json` removed.
- **Configuration vs Kubernetes fixtures** — separate `configuration-drift` and `kubernetes-failure` scenarios.
- **Intelligence pattern library** — 18 built-in patterns.
- **Archetype package layout** — domain-organized `internal/archetype/builtin/`.
- Archetype-seeded root-cause playbook; expanded question coverage across failure modes.

### Fixed

- Signal classification: rollbacks and recovery events no longer misclassified as deployments or incident onset.
- Report root-cause candidates skip refuted hypotheses.
- Session lifecycle: reject evidence after finish, duplicate evidence IDs, invalid time windows; atomic `WithSession` updates.
- MCP input validation and clearer tool errors.
- Plugin `.mcp.json` relative launcher path; install script release parsing.
- Host-LLM sampling only when client advertises `sampling` capability (avoids blocking `submit_evidence`).
- **Category-vs-content paradox** — negative findings in `deployment_events` or `configuration_changes` no longer confirm `deploy-before-errors` or `config-changed` or boost those hypotheses.
- Orthogonal `config` vs `featureflag` signal keywords; resolver coverage for all seed question IDs.

[Unreleased]: https://github.com/stackrail-io/Incident-Investigator/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/stackrail-io/Incident-Investigator/releases/tag/v1.0.0
