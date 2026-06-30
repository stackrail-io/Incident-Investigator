# Architecture Decision Records

This directory records significant architectural decisions for Incident Investigator.

| ADR | Title |
| --- | ----- |
| [0001](./0001-investigation-graph.md) | Investigation Graph as canonical structure |
| [0002](./0002-question-first-protocol.md) | Question-first investigation protocol |
| [0003](./0003-immutable-evidence.md) | Immutable evidence |
| [0004](./0004-declarative-reasoning-actions.md) | Declarative reasoning actions |
| [0005](./0005-reasoners-never-mutate-runtime.md) | Reasoners never mutate runtime state |
| [0006](./0006-no-vendor-integrations.md) | No vendor integrations in core |
| [0007](./0007-mcp-as-transport.md) | MCP as default transport |
| [0008](./0008-optional-intelligence.md) | Incident Intelligence is optional |

## Format

Each ADR follows:

- **Status** — proposed, accepted, deprecated, superseded
- **Context** — forces at play
- **Decision** — what we chose
- **Consequences** — positive and negative outcomes

## Adding an ADR

1. Copy the template from an existing ADR.
2. Number sequentially.
3. Link from this index and from relevant docs.
