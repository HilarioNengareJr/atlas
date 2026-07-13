# Atlas

> **Django for prompting.**

## Thesis

Prompting is the new programming — and right now it is in its spaghetti phase. Plans, conventions, and context are the new source code, but they live as vibes: unversioned, unstructured, evaporating after every agent session. Every era of programming produced a framework that ended its chaos phase. Django did it for web development. Atlas does it for LLM-augmented software development.

**One sentence:** Atlas is an opinionated, installable, agent-agnostic workflow framework that makes AI-assisted development compound — every task leaves the project smarter than it found it.

**The moat, in one line:** Other tools optimise the *flow* of code. Atlas accumulates a *stock* of understanding.

## What Atlas Is (and Refuses to Be)

Atlas is a framework layer over agent runtimes it does not control. It provides conventions, a lifecycle, shared state, and packaging. It is not the agent, not an orchestrator, not a UI.

**Design values:**

- **Opinionated defaults** — the intelligence is in the defaults; every config option is a small betrayal.
- **Convention over configuration** — transferability falls out of ruthless convention.
- **Human approves every plan** — the plan is a contract; approval is a code review.
- **Lean stable core, churn at the edges** — the lifecycle almost never changes; adapters and integrations absorb change.
- **Everything declares itself** — skills, MCPs, and adapters all state what they are and what they serve, so the framework can reason about the whole.
- **Guardrails, never a toll booth** — anything that feels like ceremony gets automated or cut.

**v1 explicitly refuses to do:**

- No Counsel skill (adversarial plan reviewer) — parked for v2, when the Map has enough history to make pushback evidence-based.
- No dashboard / web UI — the terminal is where users live; the meta-skill covers introspection.
- No runtime orchestration of the agent — v1 rides on hooks, it does not wrap the agent.
- No multi-agent adapters at launch — agnostic architecture, one shipped adapter (Claude Code).
- No auto-fixing meta-skill — v1 diagnoses only.
- No behavioral skill testing — v1 ships static validation (`atlas validate`).
- No signals/middleware extension hooks, no admin panel, no project template gallery.

---

*Install (when it ships): `pip install atlasdev` / `npm install -g atlasdev`.*
