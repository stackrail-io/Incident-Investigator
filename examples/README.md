# Examples

Complete example investigations demonstrate how evidence flows through the runtime and serve as **documentation and regression tests**.

Run all example tests:

```bash
go test ./examples/...
```

## Layout

Each example directory contains:

| File | Purpose |
| ---- | ------- |
| `README.md` | Scenario narrative |
| `evidence/` | JSON evidence batches (submitted in order) |
| `expected-report.md` | Key phrases expected in final report |
| `expected-graph.json` | Minimum graph node/edge counts |
| `expected-questions.json` | Plan question count expectations |
| `expected-findings.json` | Leading hypothesis assertion |
| `expected-reasoning.md` | Reasoning cycle expectations |

Examples map to archetype conformance fixtures in `spec/investigation-v1/conformance/archetype-fixtures/`.

## Scenarios

| Example | Archetype fixture | Leading hypothesis |
| ------- | ----------------- | ------------------ |
| [deployment-failure](./deployment-failure/) | `deployment-failure.yaml` | `hypothesis-deployment-caused` |
| [certificate-expiry](./certificate-expiry/) | `certificate-tls-failure.yaml` | `hypothesis-certificate-expiry` |
| [dns-outage](./dns-outage/) | `dns-failure.yaml` | `hypothesis-dns-failure` |
| [retry-storm](./retry-storm/) | `retry-storm.yaml` | `hypothesis-retry-storm` |
| [database-deadlock](./database-deadlock/) | `database-lock-contention.yaml` | `hypothesis-lock-contention` |
| [memory-leak](./memory-leak/) | `resource-exhaustion.yaml` | `hypothesis-resource-exhaustion` |
| [regional-outage](./regional-outage/) | `regional-failure.yaml` | `hypothesis-regional-failure` |

## Exporting results

After running an investigation, use `pkg/export` to render Markdown, JSON, Mermaid, GraphML, or PlantUML from session state.
