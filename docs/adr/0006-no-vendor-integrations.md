# ADR 0006: No vendor integrations in core

## Status

Accepted

## Context

Incident tools often embed Datadog, Kubernetes, or Slack SDKs, coupling releases to vendor APIs and credentials.

## Decision

The core runtime has **no connectors**. It accepts vendor-neutral evidence categories only. Clients gather data; the runtime investigates.

## Consequences

**Positive:** Vendor-neutral; smaller attack surface; portable specification.

**Negative:** Clients must map vendor schemas to evidence payloads.
