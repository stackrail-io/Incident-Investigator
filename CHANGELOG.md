# Changelog

All notable changes to **Incident Investigator** are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Nothing yet.

## [0.1.0] - 2026-06-28

### Added

- Initial open-source release of the Incident Investigator MCP server.
- Vendor-neutral evidence model with twelve supported categories.
- Stateful investigation runtime with in-memory session storage.
- MCP tools: `start_investigation`, `submit_evidence`, `get_investigation_status`, `finish_investigation`.
- Reasoning engines (all behind swappable interfaces):
  - Dynamic evidence planner
  - Competing hypothesis generation
  - Confidence scoring
  - Contradiction detection
  - Missing-evidence detection
  - Evidence graph builder
  - Timeline generator
  - Blast-radius estimator
  - Postmortem / report generator
- Realistic incident fixtures and tests (bad deployment, database outage, certificate expiry, DNS outage, Kubernetes restart loop, memory leak, retry storm).
- End-to-end MCP protocol test over an in-memory transport.
- Docker image and `docker-compose` for running the MCP server.

[Unreleased]: https://github.com/stackrail/incident-investigator/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/stackrail/incident-investigator/releases/tag/v0.1.0
