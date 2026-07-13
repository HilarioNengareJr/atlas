# Atlas — End-to-End Plan

**Version:** 0.1 (working draft) · **Owner:** Hills · **Date:** July 2026

---

## 1. Thesis

Prompting is the new programming — and right now it is in its spaghetti phase. Plans, conventions, and context are the new source code, but they live as vibes: unversioned, unstructured, evaporating after every agent session. Every era of programming produced a framework that ended its chaos phase. Django did it for web development. Atlas does it for LLM-augmented software development.

**One sentence:** Atlas is an opinionated, installable, agent-agnostic workflow framework that makes AI-assisted development compound — every task leaves the project smarter than it found it.

**Tagline candidates (pick one, then stop changing it):**
- "Django for prompting"
- "Prompting is the new programming. Atlas is what makes it engineering."

**The moat, in one line:** Other tools optimise the *flow* of code. Atlas accumulates a *stock* of understanding.

---

## 2. What Atlas Is (and Refuses to Be)

Atlas is a framework layer over agent runtimes it does not control. It provides conventions, a lifecycle, shared state, and packaging. It is not the agent, not an orchestrator, not a UI.

**Design values (the foundation):**
- Opinionated defaults — the intelligence is in the defaults; every config option is a small betrayal.
- Convention over configuration — transferability falls out of ruthless convention.
- Human approves every plan — the plan is a contract; approval is a code review.
- Lean stable core, churn at the edges — the lifecycle almost never changes; adapters and integrations absorb change.
- Everything declares itself — skills, MCPs, and adapters all state what they are and what they serve, so the framework can reason about the whole.
- Guardrails, never a toll booth — anything that feels like ceremony gets automated or cut.

**v1 explicitly refuses to do:**
- No Counsel skill (adversarial plan reviewer) — parked for v2, when the Map has enough history to make pushback evidence-based.
- No dashboard / web UI — the terminal is where users live; the meta-skill covers introspection.
- No runtime orchestration of the agent (Option B) — v1 rides on hooks, it does not wrap the agent.
- No multi-agent adapters at launch — agnostic architecture, one shipped adapter (Claude Code).
- No auto-fixing meta-skill — v1 diagnoses only; suggestions in v1.5; auto-tuning only once diagnostics have earned trust.
- No behavioral skill testing — v1 ships static validation (`atlas validate`); fixture-based behavioral tests are v2.
- No signals/middleware extension hooks, no admin panel, no project template gallery.

---

## 3. Architecture

### 3.1 The stable core: the lifecycle loop

Five stages, one path all work flows through:

1. **Survey** — ingest the codebase; build/refresh the Map (architecture, conventions, dependencies, sensitive zones).
2. **Chart** — turn a request into a structured plan: steps, footprint (declared blast radius), verification criteria, Map citations. Human approves before anything is built.
3. **Build** — the agent executes the approved plan in scoped chunks, emitting decision records on any deviation or interpretation.
4. **Verify** — run tests; reconcile edited files against the footprint; check the plan was followed, not reinterpreted; surface batched soft conflicts.
5. **Log** — write outcomes, decisions, resolutions, and new conventions back into the Map.

### 3.2 The Map — the stock at the center

The Map is not a sixth stage; it sits at the center of the loop. It is the accumulated stock of shared understanding: plain files in the repo, git-versioned, legible without Atlas installed. Every stage deposits into it. Agents get faster over time on an Atlas project; they stay the same speed forever everywhere else.

### 3.3 Config layer (Atlas's settings.py)

- `atlas.config` — committed to git. Declares which MCPs the project uses and which SDLC stage each serves. An MCP with no stage does not get generated. Configuration is code.
- `.env` — gitignored (enforced by `atlas init`, which refuses to proceed otherwise). PATs, keys, secrets. Secrets are environment. The secrets layer is an interface, not a hardcoded file read — keychain/secrets-manager backends can be added later.
- Generated MCP config — a build artifact, compiled by `atlas sync` from declaration + env. Clearly marked as generated; never hand-edited; regeneration is cheap and idempotent. `atlas init` validates every MCP connection before declaring success.

### 3.4 Observability pair

- **Decision stream** — a decision is recordable when the agent deviates from or interprets the approved plan. Following the plan verbatim is silent. Records are structured (what, alternatives, why, plan clause) and become citizens of the Map via Log. v1 delivery: a live tail in the terminal.
- **Edit stream** — file edits captured at the moment of action (via agent hooks, not a watcher daemon) and reconciled against the plan's declared footprint. Inside the footprint: silent. Outside it: soft conflict. Touching Map-marked sensitive zones: hard conflict. v1 scope: capture and reconcile only — no live diff UI, no rollback, no mid-edit intervention.

### 3.5 Conflict escalation and the attention budget

A conflict exists when two of Atlas's four sources of truth disagree: the approved plan, the Map, the actual codebase, and the verification criteria. Triggers: plan is impossible against the code; plan violates a Map convention; plan self-contradicts; Verify fails and the agent proposes to "fix" it by changing the plan's intent.

- **Hard conflicts** block and prompt the human with a fixed format: the contradiction, two or three resolutions, the trade-offs of each. Never an open-ended "what should I do?"
- **Soft conflicts** batch for review at Verify time. Interruption is spent only on the hard stuff.
- Resolutions are written back into the Map — the same conflict never escalates twice. Resolution becomes precedent.
- If a task generates more than a couple of escalations, Atlas says so: that is a Chart failure, not a reason to answer faster.
- Deferred (designed, not built): demote conflict categories to soft when humans consistently pick the agent's first-choice resolution.

### 3.6 Agent-agnostic by architecture

The core speaks only in Atlas primitives — plans, decision records, edit events, conflicts, Map entries — and never knows which agent produced them. One boundary, the **agent adapter**, translates a specific ecosystem into those primitives. Each adapter **declares its capabilities** (edit events? decision visibility? interruption?) and Atlas degrades gracefully and honestly per ecosystem. v1 ships exactly one adapter: Claude Code (richest interception surface via hooks — it stretches the whole interface). The second adapter is the first item of v1.5 and is the test of whether the boundary was drawn right. Public claim: "agent-agnostic by architecture; ships with Claude Code support; adapter interface open."

### 3.7 Skills — Atlas's Django apps

A skill is a packaged unit of capability that plugs into any Atlas project because it obeys the conventions. The keystone is the **manifest**: every skill declares its name, the single lifecycle stage it serves, what it reads from the Map, what records it emits, which MCP slots it requires — plus two duty fields:

- **`consumes`** — which context sections / Map entry types the skill reads to do its job.
- **`maintains`** — which Map entry types the skill is responsible for keeping true.

Skills never write compiled context files directly — they write to the Map; compilation projects the Map into context. One-directional pipe, no second source of truth.

**The ownership matrix (the anti-staleness mechanism):** every Map entry type must have at least one skill declaring `maintains` over it — an unmaintained entry type is staleness by design, detectable statically by `atlas validate`. An entry type nothing `consumes` is dead weight, flagged by `atlas help`. Freshness stops being a hope and becomes a checkable property: `atlas help` can report "UI registry consumed by build, maintainer last ran N tasks ago — agents are working from a stale inventory."

The manifest makes the meta-skill's gap analysis mechanical, lets `init` validate requirements up front, and makes skills distributable. v1 rules: one stage per skill; no skill-to-skill dependencies. Design principle extended: everything declares itself — not just what it is, but what it owes.

### 3.8 The time dimension (what makes it a proper framework)

- **Schema versioning from day one.** Every schema file's first line: `schema_version: 1`. Non-negotiable — versioning cannot be retrofitted onto artifacts that never declared a version.
- **`atlas migrate` (designed in v1, built when first needed).** The Map is never broken by an upgrade; accumulated understanding always has an upgrade path. The thesis dies at the first version bump otherwise.
- **`atlas validate` (built in v1).** Static checking for skills: manifest well-formed, Map dependencies exist in schema, emitted records parse, required MCP slots declared. A type-checker for skills.
- **Compatibility promise (one paragraph in the docs).** What semver means for Atlas; how long deprecated manifest fields keep working; the Map is never broken without a migration path.

### 3.9 Context compilation — how the Map reaches the agent

The Map stores understanding; agents read context windows. Context files bridge the two, and they are **compiled artifacts** — same discipline as the generated MCP config: Map is source, context files are build outputs, the adapter owns the target format (CLAUDE.md + skill files for Claude Code). Never hand-edited; regeneration idempotent; drift between what Atlas knows and what the agent sees is structurally impossible.

Two compilation moments:
- **Project context** — standing layer (architecture summary, conventions, sensitive zones). Recompiled by `atlas sync` whenever the Map changes.
- **Task context** — per-task layer: the approved plan, its footprint, and only the Map entries relevant to that footprint. Assembled at `atlas build` time, discarded after. The footprint is the relevance filter — one contract, two functions (edit reconciliation + context selection).

Why selection is non-negotiable: the Map grows forever, context windows don't. Without footprint-driven selection, accumulated understanding becomes context bloat and the compounding thesis self-destructs at scale. Selection converts the stock into usable flow.

Human escape hatch: a declared human-authored section (or verbatim-included source file) inside compiled context, so teams can add quirks and warnings without editing generated files. Machine-derived and human-authored context coexist with an explicit boundary — same philosophy as config vs. env.

**Canonical sections of compiled context** — rule: every section names its source of truth and update trigger; no section is ever hand-maintained (except the declared human-authored escape hatch):

| Section | Source of truth | Compiled at | Notes |
|---|---|---|---|
| Project overview | Map (survey summary) | `sync` | |
| Architecture | Map architecture entries | `sync` | |
| Code standards | Map conventions | `sync` | Same entries Verify checks — agent is briefed on the rules it's judged by |
| UI registry | Map component entries (path-scoped) | `sync` | **Conditional** — included only when survey detects a UI stack; targets duplicate-component failure |
| Library docs | Map library manifest (pinned versions + learned quirks) | `sync` | **Pointer, not payload** — docs themselves fetched live via Build slot MCP; never copied into the Map |
| Build plan | Approved plan artifact | `build` | Task context only; discarded after |
| Progress | Derived from decision records + log entries of active task | `build` / live | Never hand-maintained — if it can't be derived, it's cut |

**The skills↔context loop:** context briefs skills; skills write learning to the Map; compilation refreshes context; the next run starts better briefed. Two signals drive it: *corrections* (Map-vs-code disagreement — already a conflict trigger) and **context-gap records** — when a skill had to discover information mid-task that its consumed context should have provided, it emits a gap record; the responsible maintainer (per the ownership matrix) fills the hole. Context is not only corrected by skills — it is completed by what skills were missing.

**Stack-aware profiles:** survey detects the project type and switches conditional sections on/off. Nobody configures section inclusion; detection decides. Every section must earn its tokens.

Provenance: `~/myCodingW/Flow` context templates are this feature in embryo — Atlas automates an already-trusted manual practice.

---

## 4. The v1 Catalog

### 4.1 Skills (eight, after folding)

| Stage | Skill | Purpose | Ancestor in ~/myCodingW/Flow |
|---|---|---|---|
| Survey | `atlas-survey` | Build/refresh the Map (folds in conventions extraction) | `/import-context`, `/tell-me-about` |
| Chart | `atlas-plan` | Request → plan with footprint + criteria + citations (folds in the review ceremony) | `/architect` |
| Build | `atlas-build` | Execute approved plan in scoped chunks; emit decision records | `/implement` |
| Build | `atlas-fix` | Constrained repair: tighter footprint, minimal-change bias | `/fix` |
| Verify | `atlas-verify` | Tests, footprint reconciliation, plan-drift check, soft-conflict batch | — |
| Log | `atlas-log` | Deposit outcomes, decisions, conventions into the Map | — |
| Meta | `atlas-help` | Live introspection: what's installed, how it maps to this directory, gaps and redundancies | `/chat-says` |
| Meta | `atlas-doctor` | Health: Map staleness, secrets present, MCPs reachable, .env gitignored | — |

Roughly five of eight already exist in embryo in the Flow directory — the build is more retrofit than greenfield.

### 4.2 MCP slots (six types; projects fill them, Atlas defines them)

- **Survey slot** — codebase intelligence: git history, repo structure, dependency graphs.
- **Chart slot** — work intake: issue tracker (Linear / Jira / GitLab issues).
- **Build slot** — usually empty; docs/package-registry retrieval (Context7-style) when plans need current library knowledge.
- **Verify slot** — test runner, CI status, linters.
- **Log slot** — docs sync; outbound notification (e.g., Slack "task landed").
- **Meta slot** — secrets validation, connectivity checks for `atlas-doctor`.

Rule: no slot, no config entry. Do not add a seventh slot type until dogfooding demands it.

---

## 5. The Workflow, End to End

**First run:** `atlas init` → doctor checks (secrets, gitignore, MCP connectivity) → survey builds the initial Map → the human is shown the Map and confirms it looks true. The human blesses the foundation.

**Every task thereafter:**
1. Bring a task (typed, or pulled from the Chart slot's issue tracker).
2. `atlas plan` → plan with footprint and criteria, citing the Map.
3. Human reviews and approves — the contract moment, formatted like a code review.
4. `atlas build` → agent executes; decision stream tails in the terminal; out-of-footprint edits become soft conflicts; contradictions hard-stop with 2–3 option prompts.
5. `atlas verify` → reconciliation + checks + batched soft-conflict review.
6. `atlas log` → learning deposited.
7. Next task starts from a smarter Map. ← This sentence is the demo, the tutorial's final beat, and the thesis in miniature.

---

## 6. Installation and Interface

- `pip install atlas-dev` / `npm install -g atlas-dev` — **name pending registry check; do this early, it can invalidate branding.**
- One core codebase (Go — single binary), thin pip and npm wrappers. Never write the core twice.
- **Seven commands total:** `init`, `plan`, `build`, `verify`, `sync`, `help`, `doctor` (+ `validate` for skill authors; `migrate` reserved). If v1 needs a ninth, something is wrong.
- `init` asks only for missing secrets. Every question it asks is a DX tax — count them like a performance budget.

**DX standard (Django-grade, testable):**
- The 15-minute tutorial is the product: `init` on a real small repo, one feature through the full loop, ending on the visible moment the Map gets smarter.
- Error messages are the interface: every interruption states what's wrong, why Atlas cares, and what to do next — in that order, every time.
- Docs are half the codebase — and they double as the "prompting is the new programming" writing project. The framework is the argument; the writing is the argument; build them together.

---

## 7. Phases and Timeline (evenings/weekends pace)

**Phase 0 — Lock the definition (July, ~1 week)**
- Finalize the one-pager from this plan: the sentence, values, mechanisms, refuses-list.
- Registry name check on PyPI and npm.
- Exit: one page, one tagline, name secured.

**Phase 1 — Schemas (July, 1–2 evenings + iteration)**
Drafting order (smallest and most testable first), every file opening with `schema_version: 1`:
1. Skill manifest (~30 min) — forces: mandatory vs optional fields; one stage per skill; no inter-skill deps.
2. Decision record (~30 min) — forces: minimal fields useful six months later; how a record addresses a plan clause.
3. Plan schema (~45 min) — forces: addressable steps; footprint as globs; checkable verification criteria.
4. Map schema (~60 min, the boss fight) — forces: entry types (only what the other schemas proved necessary); directory of files, not one file; staleness metadata (created-by-task, last-confirmed).
5. Config schema — transcription of what's already designed.
- Same-day reality test: hand-write manifests for `/architect`, `/implement`, `/fix` from Flow. Every hesitation is a schema bug found free.
- Exit: four+ schema files versioned; three real manifests validate by eyeball; punt-list written.
- Then: adversarial review of the drafts (break them with renamed files mid-task, overlapping record claims, self-contradicting Map entries).

**Phase 2 — Prototype on real work (August)**
- **Hooks spike first** — load-bearing for both streams: confirm Claude Code hooks can capture decisions and edits without an orchestrator.
- Run Atlas-as-files on actual Rain/SKNS work for several weeks.
- Dogfooding questions, tracked honestly:
  - Which steps do I skip when tired and rushing? (Skipped = automate or cut.)
  - When Atlas interrupted me, was it right to? (Track the hit rate.)
- Exit: workflow proven or amended; ceremony identified and removed.

**Phase 3 — Package (September–October)**
- Go core, `init`/`sync`/`doctor`/`validate` first, then the loop commands.
- Thin pip/npm wrappers; generated-artifact discipline; `.env` enforcement.
- Exit: `pip install` → 15-minute tutorial works on a fresh repo.

**Phase 4 — Open it up (October–November)**
- Public GitHub, MIT or Apache license, README with the 30-second before/after, short demo video (the `atlas help` introspection moment is the demo).
- Compatibility-promise paragraph in the docs.
- Launch quietly: use it publicly on a real project; write the essays; share in Claude Code / AI-dev communities.

**v1.5 (designed, deferred):** second agent adapter (the boundary test) · meta-skill suggestions · soft-demotion tuning for conflicts.
**v2 parking lot:** Counsel (flagship, once the Map has history) · **`atlas-standup` — designated first contrib skill, built post-launch against the public manifest contract to prove the ecosystem story: reads diffs + decision/log records, generates standup summary and backlog items, pushes to Chart slot (ClickUp/Linear/etc.)** · behavioral skill testing on fixture repos · dashboard/UI · live diffing and rollback.

---

## 8. Risks, Held Consciously

1. **The Map is the thesis and the biggest unknown.** Stale documentation is the graveyard of every similar tool. Whether agents can reliably write back understanding that stays true is untested — this is why schemas come first and dogfooding is non-negotiable. Structural mitigation: the ownership matrix (§3.7) — every entry type has a declared maintainer, staleness is statically detectable, and `atlas help` reports maintainer recency.
2. **Hooks dependency.** An interception surface Atlas doesn't control, from a fast-moving product. Mitigation: the adapter boundary — hook integration stays a thin adapter; the core never knows where events come from.
3. **Crowded, fast-moving space.** Spec-driven and planning-layer tools are multiplying. The differentiators (stock thesis, conflict taxonomy, footprint contract) are real but the novelty window is months. Ship the small true version fast.
4. **Toll-booth risk.** Policy resistance is a law of systems: the moment Atlas feels like ceremony, devs route around it, and a bypassed workflow tool is dead. The dogfooding skip-test is the defense.
5. **Escalation fatigue.** Over-prompting gets rubber-stamped, and rubber-stamping launders bad decisions as human-approved. The hard/soft split and the attention budget are the defense.
6. **Context bloat at scale.** The Map grows forever; context windows don't. If footprint-driven selection is weak, month-six agents get slower and noisier, inverting the thesis. Selection quality is a first-class engineering concern, not plumbing.
7. **Ambition/design gap.** "For everyone" vs. one agent, one workflow, one opinion. The honest public claim: agnostic by architecture, one adapter shipped, interface open. Django started as a newspaper's internal tool that happened to be right.

---

## 9. Success Criteria

- **Phase 2:** Hills genuinely prefers working through Atlas on real Rain tasks — no skipped stages after week two; escalation hit-rate is defensible.
- **Phase 3:** A stranger completes the 15-minute tutorial without help and sees the Map-gets-smarter moment.
- **Phase 4:** First external user; first external skill manifest written against `atlas validate`; first GitHub issue that isn't from a friend.
- **The long test:** month-six agents on an Atlas project are measurably faster/safer than day-one agents. If the stock doesn't compound, the thesis is wrong — and knowing that is also success.

---

## 10. Immediate Next Actions (this week)

1. Registry check the name (PyPI + npm) — before attachment deepens.
2. Draft the one-pager from Sections 1–2 of this document; pick the tagline; stop changing it.
3. Schema session per Phase 1 — one evening, timeboxed, punt-list allowed.
4. Retrofit manifests onto three Flow skills — the one-hour reality test.
5. Bring schema drafts back for adversarial review.

*Schemas that are 80% right and touched reality beat perfect ones that are still theoretical — `atlas migrate` exists so v1 is allowed to be wrong later.*
