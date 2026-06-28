# Incident Investigator plugin

Claude Code and Codex plugin bundle for the [Incident Investigator](https://github.com/stackrail-io/Incident-Investigator) MCP server.

## Contents

| Path | Purpose |
| ---- | ------- |
| `.claude-plugin/plugin.json` | Claude Code plugin manifest |
| `.codex-plugin/plugin.json` | Codex plugin manifest |
| `.mcp.json` | MCP server configuration (stdio) |
| `skills/incident-investigation/` | Investigation workflow skill |
| `scripts/run-mcp.sh` | Starts the MCP server (PATH binary or Docker) |

## Prerequisites

Install **one** of:

- Native binary: `curl -fsSL https://raw.githubusercontent.com/stackrail-io/Incident-Investigator/main/scripts/install.sh | bash`
- Docker: `docker pull ghcr.io/stackrail-io/incident-investigator:0.1.0`

The plugin's `run-mcp.sh` wrapper tries the binary on `PATH` first, then Docker.

## Claude Code

```text
/plugin marketplace add stackrail-io/Incident-Investigator
/plugin install incident-investigator@incident-investigator
/reload-plugins
```

Validate before submitting to the community marketplace:

```bash
claude plugin validate plugins/incident-investigator
```

Submit for review: [platform.claude.com/plugins/submit](https://platform.claude.com/plugins/submit)

## Codex

```bash
codex plugin marketplace add stackrail-io/Incident-Investigator
codex plugin install incident-investigator --source incident-investigator
```

Browse in the Codex TUI with `/plugins`.

## Skill

After install, invoke:

```text
/incident-investigator:incident-investigation
```

Or ask Claude/Codex to investigate an incident; the skill is model-invoked when relevant.
