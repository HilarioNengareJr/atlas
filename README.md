# ATLAS

Atlas is an opinionated, installable, agent-agnostic workflow framework that makes AI-assisted development compound — every task leaves the project smarter than it found it. Tagline (frozen): **"Django for prompting."**

Atlas is a framework layer over agent runtimes it does not control. It provides conventions, a lifecycle (Survey → Chart → Build → Verify → Log), shared state (the Map), and packaging (skills with manifests, MCP slots, agent adapters). It is not the agent, not an orchestrator, not a UI.

Right now the repo is pre-v0.1: plans, a one-pager, and skeleton directories. Milestone 0 (name, sentence, home) is complete; Milestone 1 (schemas) is next.

---

## The Problem It Solves

Prompting is the new programming — and it is in its spaghetti phase. Plans, conventions, and context are the new source code, but they live as vibes: unversioned, unstructured, evaporating after every agent session. Other tools optimise the *flow* of code; Atlas accumulates a *stock* of understanding. Agents get faster over time on an Atlas project; they stay the same speed forever everywhere else.

---

## Commands

The product surface is a CLI. Seven commands total (if v1 needs a ninth, something is wrong):

```
atlas init      → doctor checks, survey builds the Map, human blesses it
atlas plan      → request → plan with footprint + criteria, citing the Map
atlas build     → execute approved plan; decision stream tails in terminal
atlas verify    → tests, footprint reconciliation, soft-conflict batch
atlas sync      → recompile generated MCP config + project context from the Map
atlas help      → introspection: installed skills, stage coverage, ownership gaps
atlas doctor    → health: secrets, .env gitignored, MCPs reachable, Map staleness
```

Plus `atlas validate` for skill authors; `atlas migrate` reserved (designed in v1, built when first needed).

---

## Core User Flow

### First run

- `atlas init` → doctor checks (secrets, gitignore, MCP connectivity)
- Survey builds the initial Map
- The human is shown the Map and confirms it looks true — the human blesses the foundation

### Every task thereafter

1. Bring a task (typed, or pulled from the Chart slot's issue tracker)
2. `atlas plan` → plan with footprint and criteria, citing the Map
3. Human reviews and approves — the contract moment, formatted like a code review
4. `atlas build` → agent executes; decision stream tails in the terminal; out-of-footprint edits become soft conflicts; contradictions hard-stop with 2–3 option prompts
5. `atlas verify` → reconciliation + checks + batched soft-conflict review
6. `atlas log` → learning deposited into the Map
7. Next task starts from a smarter Map ← this sentence is the demo and the thesis in miniature

### Conflict escalation

- A conflict exists when two of the four sources of truth disagree: approved plan, Map, actual codebase, verification criteria
- Hard conflicts block and prompt with a fixed format: the contradiction, 2–3 resolutions, trade-offs of each — never an open-ended "what should I do?"
- Soft conflicts batch for review at Verify time
- Resolutions are written back into the Map — the same conflict never escalates twice

---

## Data Architecture

### The Map

- The accumulated stock of shared understanding: plain files in the repo, git-versioned, legible without Atlas installed
- Sits at the center of the lifecycle loop; every stage deposits into it
- Never hand-maintained via compiled context — skills write to the Map; compilation projects the Map into context files (one-directional pipe, no second source of truth)

### Records

- Decision records: emitted when the agent deviates from or interprets the approved plan (following the plan verbatim is silent)
- Context-gap records: emitted when a skill had to discover mid-task what its consumed context should have provided
- Both become citizens of the Map via Log

### Config layer

- `atlas.config` — committed; declares MCPs and which SDLC stage each serves
- `.env` — gitignored (enforced by `atlas init`); secrets are environment
- Generated MCP config — a build artifact compiled by `atlas sync`; never hand-edited

---

## Features In Scope

- The five-stage lifecycle loop: Survey, Chart, Build, Verify, Log
- The Map — git-versioned plain files with staleness metadata and ownership matrix
- Eight skills: atlas-survey, atlas-plan, atlas-build, atlas-fix, atlas-verify, atlas-log, atlas-help, atlas-doctor
- Skill manifests: name, stage, consumes, maintains, emits, requires_slots
- Six MCP slot types: Survey, Chart, Build, Verify, Log, Meta
- Decision stream (live terminal tail) + edit stream (hook-captured, footprint-reconciled)
- Hard/soft conflict escalation with fixed prompt format
- Context compilation: project context at `sync`, task context at `build` (footprint-filtered)
- Human-authored escape-hatch section inside compiled context
- Schema versioning from day one (`schema_version: 1`, line one of every schema file)
- `atlas validate` — static skill checking
- One agent adapter: Claude Code (hooks)
- Go core, single binary; thin pip and npm wrappers (`atlasdev` on both registries)
- The 15-minute tutorial

## Features Out of Scope

- Counsel skill (adversarial plan reviewer) — v2, once the Map has history
- Dashboard / web UI
- Runtime orchestration of the agent — v1 rides on hooks, does not wrap the agent
- Multi-agent adapters at launch — one adapter shipped, interface open; second adapter is v1.5
- Auto-fixing meta-skill — v1 diagnoses only
- Behavioral skill testing — v1 ships static validation only; fixture-based tests are v2
- Signals/middleware extension hooks, admin panel, project template gallery
- `atlas-standup` — designated first contrib skill, post-launch
- Live diffing and rollback

---

## Target User

A developer using AI agents (Claude Code first) for real work who:

- Is tired of plans, conventions, and context evaporating between agent sessions
- Wants a repeatable lifecycle instead of ad-hoc prompting
- Wants to approve plans before agents build — and have deviation surfaced, not buried
- Lives in the terminal
- Will tolerate opinionated defaults in exchange for compounding project understanding

---

## Success Criteria

- **Phase 2 (dogfood):** Hills genuinely prefers working through Atlas on real Rain tasks — no skipped stages after week two; escalation hit-rate is defensible
- **Phase 3 (package):** a stranger completes the 15-minute tutorial without help and sees the Map-gets-smarter moment
- **Phase 4 (launch):** first external user; first external skill manifest written against `atlas validate`; first GitHub issue that isn't from a friend
- **The long test:** month-six agents on an Atlas project are measurably faster/safer than day-one agents — if the stock doesn't compound, the thesis is wrong, and knowing that is also success
