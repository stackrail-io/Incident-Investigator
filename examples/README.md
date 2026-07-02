# Examples

Complete, replayable investigations with **realistic evidence JSON** — the same shape you submit via MCP `submit_evidence`.

Each example is self-contained:

```
examples/<name>/
  investigation.json     # start_investigation parameters
  evidence/
    01-*.json            # first submit_evidence batch (with payloads)
    02-*.json            # second batch
    ...
  expected-findings.json
  expected-questions.json
  expected-graph.json
  expected-report.md     # full RCA report (chronological evidence timeline)
  expected-reasoning.md
  README.md
```

## Run tests

```bash
go test ./examples/...
```

Tests load evidence from `examples/<name>/evidence/*.json` — not from hidden fixture shortcuts.

## Regenerate from conformance fixtures

After editing archetype YAML fixtures:

```bash
go run internal/spec/cmd/gen-examples/main.go
```

Hand-tuned scenarios (`database-lock-contention`) keep rich payloads from the spec fixture.

## Scenarios

| Example | Incident |
| ------- | -------- |
| [deployment-failure](deployment-failure/) | Bad deploy → 5xx spike → rollback recovery |
| [certificate-expiry](certificate-expiry/) | Expired TLS cert on API gateway |
| [dns-outage](dns-outage/) | NXDOMAIN for `db.internal` |
| [retry-storm](retry-storm/) | Gateway retry amplification |
| [database-deadlock](database-deadlock/) | Row lock queue with healthy DB metrics |
| [memory-leak](memory-leak/) | Heap growth → OOMKill |
| [regional-outage](regional-outage/) | Single-AZ impairment |
