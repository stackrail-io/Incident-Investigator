# Database lock contention

**Service:** `auth-api`  
**Question:** Why did renameIdentityProvider writes spike to 66s p95?

## What happened

A long-running `DELETE` on `identity_provider:pk-42` held a row lock for **222.9s**. Three `UPDATE` statements queued behind it. Database **CPU and connections stayed healthy** — this is lock queueing, not saturation.

## Evidence batches

1. `01-config-gap.json` — pool missing `statement_timeout` / `lock_timeout`
2. `02-latency-alert.json` — p95 breach on write path
3. `03-db-metrics.json` — Postgres 12/100 connections, CPU 15%
4. `04-lock-queue.json` — traces + `pg_locks`-style events with `duration_ms` payloads

## Expected outcome

Leading hypothesis: **`hypothesis-lock-contention`** (not database saturation).
