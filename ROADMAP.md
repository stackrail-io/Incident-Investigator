# Roadmap

## v1.1 (current) — Framework maturity

- Architecture-first documentation (`docs/`, ADRs, specification)
- Public extension package (`pkg/extension`) and exporters (`pkg/export`)
- Example investigations with regression tests (`examples/`)
- Internal event bus and explainability helpers
- Provider registration via runtime options

## v1.2 — Extension ergonomics

- Promote `pkg/model` for out-of-tree extension authors
- Shared resolver question ID registry (replace regex coverage test)
- Durable `GraphStore` and `runtime.Store` reference implementations
- Additional conformance tiers for reasoning and intelligence

## v2.0 — Ecosystem

- Plugin loading convention (without vendor SDKs in core)
- Additional transport bindings beyond MCP (gRPC/REST from spec)
- Expanded archetype library with community contributions

## Non-goals

- Cloud vendor connectors (Datadog, Kubernetes SDK, Slack, …)
- Hosted SaaS or authentication
- UI/dashboard in core repository

See [CONTRIBUTING.md](CONTRIBUTING.md) to propose changes.
