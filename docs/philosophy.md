# Philosophy

Incident Investigator exists because **investigation is a discipline**, not a chat skill.

This document explains *why* the project is built the way it is. For *how*, see [architecture](architecture.md) and [design principles](design-principles.md).

---

## Investigations are question-driven

An incident is not solved by collecting everything and hoping a pattern emerges. It is solved by answering specific questions in a deliberate order:

- Did a deployment precede the failure?
- Is the database saturated or merely slow?
- Does the timeline contradict the leading theory?

**Evidence does not speak for itself.** It answers questions. Questions test hypotheses. Hypotheses answer the goal.

When investigations drift into unstructured narration—"here are some logs, what do you think?"—they become non-reproducible. Two engineers, or two model runs, can reach different conclusions from the same facts. Question-driven investigation makes progress **inspectable**: you can see what was asked, what was answered, and what remains open.

That is why Incident Investigator treats the investigation plan—not the evidence pile—as the center of gravity.

---

## AI should gather evidence, not own reasoning

Large language models are extraordinary at **access**: reading logs, summarizing alerts, translating vendor APIs into human language, deciding which tool to call next. They are unreliable at **custody**: maintaining competing theories, tracking what would change your mind, knowing when you have enough to conclude, and explaining why you believe what you believe.

Those failures are not bugs in a particular model. They are structural. A stateless completion over a growing transcript is not an investigation runtime.

Incident Investigator draws a bright line:

| The assistant | The runtime |
| ------------- | ----------- |
| Chooses which systems to query | Owns the investigation state |
| Submits evidence in neutral categories | Maintains hypotheses and confidence |
| Answers protocol questions | Decides sufficiency and next questions |
| Operates in the messy vendor world | Reasons only over what was submitted |

The assistant is the **field agent**. The runtime is the **investigator**. Confusing the two produces confident reports that cannot be audited—and audits matter more than fluency when production is down.

---

## Explainability matters more than autonomy

Autonomy is seductive. "Let the AI investigate end to end" sounds efficient until something goes wrong: a wrong root cause, a missed rollback window, a blameless postmortem that is not blameless because no one can reconstruct the reasoning.

**An investigation you cannot explain is an investigation you cannot trust.**

Explainability is not a feature bolted onto output. It is the reason the runtime exists:

- Why does this hypothesis lead?
- Why not the alternative?
- Why is confidence at this level?
- Why are we not done yet?
- What evidence would change the conclusion?

A system optimized for autonomy hides these answers in weights and prompts. A system optimized for explainability makes them **first-class state**—journal entries, reasoning traces, graph paths, sufficiency reports.

We would rather say "insufficient evidence" than say "probably the deploy" without a path a human can follow. Restraint is a feature.

---

## Deterministic reasoning and LLM reasoning are complements

Heuristic engines are predictable. They encode domain patterns—deploy-before-error ordering, certificate expiry signals, lock-contention cues—with tests that replay the same way every time. That predictability is how you build **conformance**: 32 archetype fixtures, regression tests, spec compliance.

LLMs are flexible. They can synthesize across noisy logs, notice metaphors in summaries, and propose connections no rule author anticipated. That flexibility is valuable—and **dangerous** without a runtime that validates, merges, and records what was accepted.

Incident Investigator does not ask you to choose one or the other:

- **Deterministic reasoners** anchor the investigation in reproducible logic.
- **Semantic reasoners** may contribute when a host LLM is available—but they propose **actions**, not truth.
- The **runtime** decides what becomes state.

LLM reasoning is optional. Explainable conclusions are not. The philosophy is partnership: machines for structure, models for breadth, humans for judgment—with a clear audit trail between them.

---

## An investigation runtime, not an AI agent

An **agent** improvises. It plans, acts, observes, and plans again inside a loop it owns. That is the right abstraction for open-ended tasks.

An **investigation runtime** commits to a protocol. It defines entities (question, evidence, hypothesis), lifecycle (started → reasoning → completed), and operations (submit, resolve, finish). Extensions register; core behavior stays stable.

Incident Investigator is deliberately the second thing.

It is not competing with Claude or Codex to be a better chat partner. It is the **engine underneath**—the same relationship Git has to your editor, or Kubernetes to your containers, or Terraform to your cloud console. Those tools do not replace judgment; they **encode** it so teams can share, review, and improve it.

We believe production incidents deserve that kind of infrastructure: not another autonomous actor, but a **neutral runtime** that turns evidence into accountable conclusions.

---

## What this means for contributors

When you propose a change, ask:

1. Does it strengthen the question-driven protocol, or bypass it?
2. Does it keep reasoning in the runtime, or leak it into the client?
3. Does it make conclusions more explainable, or more opaque?
4. Does it preserve deterministic replay, or only work in one model's head?
5. Does it belong in core, or in an extension that registers without forking?

If the answer aligns with this philosophy, it probably belongs. If it trades explainability for autonomy, or conflates the assistant with the investigator, it probably does not—no matter how impressive the demo looks.

---

## Related reading

- [Design principles](design-principles.md) — architectural contracts derived from this philosophy
- [Architecture](architecture.md) — how components implement it
- [Investigation Specification](../spec/investigation-v1/SPECIFICATION.md) — the portable protocol contract
