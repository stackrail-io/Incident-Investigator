# ADR 0002: Question-first investigation protocol

## Status

Accepted

## Context

AI assistants naturally accumulate evidence without structure. Investigations stall when nothing drives what to collect next or when conclusions are reached without answered questions.

## Decision

Model investigations as **questions** linked to evidence requests and hypothesis effects. Playbooks declare questions; the protocol engine resolves them from submitted evidence.

## Consequences

**Positive:** Clear sufficiency criteria; explainable progress; MCP tools map naturally to plan/questions.

**Negative:** More ceremony than free-form chat; playbook authoring required per goal.
