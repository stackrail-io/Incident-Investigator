# Contributing to Incident Investigator

Thank you for your interest in contributing. Incident Investigator is a
vendor-neutral investigation engine — we welcome fixes, tests, documentation,
and engine improvements that stay true to that philosophy.

## Before you start

- Read the [README](README.md) and understand the core rule: **no infrastructure
  connectors**. The engine reasons over evidence; the AI assistant gathers it.
- Check [open issues](https://github.com/stackrail/incident-investigator/issues)
  and [pull requests](https://github.com/stackrail/incident-investigator/pulls)
  to avoid duplicate work.

## How to report bugs

Use the [Bug report issue template](https://github.com/stackrail/incident-investigator/issues/new?template=bug_report.yml).

Include:

- What you expected vs. what happened
- Steps to reproduce (MCP tool calls, evidence payloads, session id if relevant)
- Version (`incident-investigator` logs `version` on startup, or see badge in README)
- Go version and how you run the server (binary, `go run`, Docker)
- Relevant logs from **stderr** (stdout is reserved for MCP)

For small, obvious fixes, a pull request with a test is often faster than filing
an issue first.

## How to report security issues

**Do not open a public GitHub issue for security vulnerabilities.**

See [SECURITY.md](SECURITY.md) for our responsible disclosure process.

## Development setup

Requirements: **Go 1.25+**

```bash
git clone https://github.com/stackrail/incident-investigator.git
cd incident-investigator
go test ./...
go build -o bin/incident-investigator ./cmd/incident-investigator
```

### Run tests

```bash
go test ./...
go test -race ./...
```

Add or update tests when you change engine behavior. Realistic fixtures live in
`internal/fixtures/` — extend them when adding new incident archetypes.

### Project layout

| Path | Purpose |
| ---- | ------- |
| `cmd/incident-investigator/` | MCP server entrypoint |
| `internal/model/` | Domain types |
| `internal/engine/` | Planner, hypotheses, confidence, etc. (interfaces + heuristics) |
| `internal/runtime/` | Session lifecycle and store |
| `internal/mcpserver/` | MCP tool wiring |
| `internal/fixtures/` | Incident scenarios for tests |

## Making changes

1. Fork the repository and create a branch from `main`.
2. Make focused changes — one logical change per pull request.
3. Run `go test ./...` and `go vet ./...`.
4. Update [CHANGELOG.md](CHANGELOG.md) under **Unreleased** (Keep a Changelog format).
5. Open a pull request with a clear description and test plan.

### Engine changes

New reasoning logic should:

- Live behind an existing interface in `internal/engine/` when possible
- Remain vendor-neutral (no CloudWatch/Datadog/K8s-specific parsing)
- Include fixture-based tests that assert planner, hypotheses, timeline, graph, or confidence behavior

### MCP tool changes

MCP tool schemas are inferred from Go types. If you change DTOs in
`internal/mcpserver/`, update the end-to-end test in
`internal/mcpserver/server_test.go`.

## Pull request checklist

- [ ] Tests pass locally (`go test ./...`)
- [ ] New behavior has test coverage where appropriate
- [ ] CHANGELOG.md updated
- [ ] No new external connectors or vendor SDKs
- [ ] README updated if user-facing behavior changed

## Code of conduct

This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md). By
participating, you agree to uphold it.

## Questions

- [GitHub Discussions](https://github.com/stackrail/incident-investigator/discussions) (if enabled)
- [Discord community](https://discord.gg/stackrail)
- [Book a demo](https://stackrail.io/demo) for StackRail product questions
