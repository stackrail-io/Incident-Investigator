# Design Principles

These principles are **architectural contracts**. They guide implementation decisions and conformance tests. Violating them is a breaking change.

For the *why* behind these rules, read [philosophy](philosophy.md).

## 1. Runtime owns investigations

The investigation runtime is the single authority for session state, lifecycle transitions, and when an investigation may conclude. Clients submit evidence and receive plans; they do not mutate hypotheses or confidence directly.

## 2. Questions drive investigations

An investigation is a structured set of **questions**, not a bag of evidence. Evidence exists only to answer questions. Questions validate hypotheses. Hypotheses answer the investigation goal.

## 3. Evidence is immutable

Submitted evidence is append-only. Corrections arrive as new evidence items with explicit references, not in-place edits. This preserves auditability and replay.

## 4. Graph is canonical

The **Investigation Graph** is the structural source of truth for relationships between evidence, hypotheses, services, and timeline events. Engines sync into the graph; exporters and path queries read from it.

## 5. Reasoners are stateless

Reasoners receive a **read-only** investigation view. They return declarative `ReasoningAction`s. The runtime validates and applies actions. Reasoners never hold per-investigation mutable state.

## 6. History is optional

Incident Intelligence (archive, similarity, patterns, calibration) enhances investigations but is **not required**. `intelligence.Noop()` disables the layer without affecting core protocol behavior.

## 7. Playbooks are declarative

Investigation playbooks describe questions, dependencies, required evidence categories, and hypothesis effects. They do not execute imperative code at runtime beyond what the protocol engine interprets.

## 8. Storage is replaceable

Session storage (`runtime.Store`) and graph storage (`graph.Store`) are interfaces. The default is in-memory; production deployments may plug in durable backends without changing engines.

## 9. LLM reasoning is optional

Heuristic engines power the default runtime. Semantic reasoners may call an LLM via a pluggable `reasoning.LLM` or MCP host sampling—but investigations must remain explainable without an LLM.

## 10. Every conclusion must be explainable

For any hypothesis, confidence score, or incomplete status, the runtime must provide an explanation path:

- Why this hypothesis leads
- Why another hypothesis does not
- Why confidence is at its current level
- Why the investigation is incomplete
- Why more evidence is required

Use `Explain`, `ExplainInvestigation`, and `explain` package helpers.

## 11. No vendor integrations

The runtime understands **evidence categories**, not Datadog queries or Kubernetes APIs. Connectors belong in the client layer that gathers evidence.

## 12. Extension via registration

New capabilities register through provider registries (`pkg/extension`). Core runtime code should not require modification when adding a reasoner, archetype, or exporter.

## 13. Specification-first

Behavior that affects interoperability is defined in [`spec/investigation-v1/`](../spec/investigation-v1/) before implementation. Conformance fixtures are the regression contract.

## 14. Vendor-neutral vocabulary

Domain language uses investigation protocol terms—question, evidence, hypothesis, finding—not chat or agent terminology in core APIs.
