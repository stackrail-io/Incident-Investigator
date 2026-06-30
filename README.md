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

```
        Claude / Codex
              │
              ▼
      Collect Evidence
              │
              ▼
   Incident Investigator
   ├── Protocol
   ├── Questions
   ├── Investigation Graph
   ├── Reasoners
   └── Intelligence
              │
              ▼
    Structured Findings
              │
              ▼
        Claude / Codex
```

An open-source **investigation runtime** ([MCP](https://modelcontextprotocol.io) server). Not an AI agent—a stateful engine that turns vendor-neutral evidence into hypotheses, confidence, and explainable conclusions.

The assistant gathers from Datadog, Kubernetes, logs, or anywhere else. The runtime owns the investigation: what to ask next, what you still need, when you can conclude, and why.

**No connectors in core.** Evidence categories in; structured findings out.

## Quick start

```bash
go install github.com/stackrail/incident-investigator/cmd/incident-investigator@latest
```

Wire it as an MCP server in Claude Code, Codex, or Cursor → [plugins/incident-investigator/](plugins/incident-investigator/).

```bash
git clone https://github.com/stackrail-io/Incident-Investigator.git && cd Incident-Investigator
go test ./...
```

## Documentation

| | |
| --- | --- |
| [docs/README.md](docs/README.md) | **Documentation index** |
| [Philosophy](docs/philosophy.md) | Why the project exists |
| [Architecture](docs/architecture.md) | How it fits together |
| [Development](docs/development.md) | Extend reasoners, archetypes, playbooks |
| [Specification](spec/investigation-v1/SPECIFICATION.md) | Portable protocol contract |
| [Examples](examples/) | Seven replayable investigations |

## Contributing

[CONTRIBUTING.md](CONTRIBUTING.md) · [ROADMAP.md](ROADMAP.md)

## License

MIT — [LICENSE](LICENSE)
