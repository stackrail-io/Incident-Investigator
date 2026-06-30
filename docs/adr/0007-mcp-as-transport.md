# ADR 0007: MCP as default transport

## Status

Accepted

## Context

Investigations are driven by AI assistants that already speak MCP to tools. A custom RPC would duplicate client integration work.

## Decision

Ship the reference implementation as an **MCP server** over stdio. The Investigation Specification is transport-agnostic; MCP is one binding.

## Consequences

**Positive:** Works with Claude, Codex, Cursor, and other MCP hosts without custom SDKs.

**Negative:** Non-MCP integrations must wrap or reimplement operations from the spec.
