---
description: Run a structured incident investigation using the Incident Investigator MCP engine. Use when investigating outages, regressions, deployments, latency spikes, or when the user asks for a postmortem, root cause analysis, or evidence-based incident timeline.
---

# Incident investigation workflow

You are conducting a **vendor-neutral** investigation. The Incident Investigator engine
reasons over evidence; **you** gather it from the user's tools (logs, metrics, alerts,
deployments, traces, chat, etc.).

## 1. Start

Call `start_investigation` with:

- `question` — what happened (e.g. "Why did checkout fail yesterday?")
- `service` — primary service under investigation (optional)
- `time_window` — incident window if known (optional)

Save the returned `session_id`.

## 2. Collect evidence

For each item in `required_evidence` / `next_required_evidence`:

1. Use the user's available tools to fetch real data (CloudWatch, Datadog, kubectl,
   GitHub, Slack, etc.). The engine does not connect to these systems.
2. Map each observation to a **vendor-neutral category**:
   `application_logs`, `deployment_events`, `alert_events`, `metrics`,
   `trace_events`, `configuration_changes`, `network_events`, `database_events`,
   `security_events`, `human_context`, or `custom`.
3. Call `submit_evidence` with normalized objects:

```json
{
  "session_id": "...",
  "evidence": [
    {
      "timestamp": "2026-06-27T09:01:00Z",
      "category": "deployment_events",
      "entity": "checkout-api",
      "summary": "Deployed checkout-api v2.4.0 to production",
      "payload": { "region": "us-east-1", "version": "v2.4.0" }
    }
  ]
}
```

Repeat until `progress` and `confidence` are sufficient or the user wants to conclude.

## 3. Monitor

Use `get_investigation_status` to inspect hypotheses, contradictions, timeline, and
missing evidence. Resolve contradictions by collecting clarifying evidence.

## 4. Finish

Call `finish_investigation` and present:

- Executive summary and leading root-cause candidates
- Timeline with evidence references
- Contradictions and missing evidence
- Blast radius and recommendations
- The markdown postmortem

Never invent evidence. If data is unavailable, say so and note it in missing evidence.
