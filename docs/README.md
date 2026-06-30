# Documentation

Start here after the [README](../README.md). Everything beyond a two-minute overview lives in this tree.

## Understand the project

| Doc | Read when you want to… |
| --- | ---------------------- |
| [Philosophy](philosophy.md) | Understand *why* — questions over evidence, runtime vs agent |
| [Architecture](architecture.md) | See components, ownership, and data flow |
| [Design principles](design-principles.md) | Learn the architectural contracts |
| [ADRs](adr/README.md) | Review specific design decisions |

## Build and extend

| Doc | Read when you want to… |
| --- | ---------------------- |
| [Development guide](development.md) | Run tests, add reasoners, archetypes, playbooks |
| [Extension APIs](extension-apis.md) | Register providers without forking core |
| [Examples](../examples/README.md) | Replay complete investigations |
| [Investigation Specification](../spec/investigation-v1/SPECIFICATION.md) | Implement the protocol in another language |

## Reference

- **MCP tools** — `incident-investigator help` or [plugins/incident-investigator](../plugins/incident-investigator/)
- **Spec YAML** — [spec/investigation-v1/](../spec/investigation-v1/)
- **Conformance** — [spec/investigation-v1/conformance/](../spec/investigation-v1/conformance/)
- **Contributing** — [CONTRIBUTING.md](../CONTRIBUTING.md)
- **Roadmap** — [ROADMAP.md](../ROADMAP.md)

## Testing

```bash
go test ./...
go test ./internal/spec/...   # 32 archetype conformance fixtures
go test ./examples/...        # end-to-end example investigations
```
