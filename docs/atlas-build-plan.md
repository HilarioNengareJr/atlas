# Atlas — Build Plan

**Version:** 0.1 · **Companion to:** `atlas-plan.md` (the what/why) · **This document:** the how/when
**Working constraints:** evenings + weekends, alongside probation-era workload. Unit of work = **one session ≈ 2–3 focused hours.** Assume ~3 sessions/week sustainable → ~45–50 sessions through November. The plan budgets 40, leaving slack, because slack is not optional.

**Meta-rule:** Atlas is built using Atlas's own discipline. Each task below is a mini-plan with a footprint and done-criteria. Outcomes, decisions, and skipped steps get logged in the Obsidian vault after every session — that log is the embryonic Map of the Atlas project itself, and it is also the dogfooding data.

---

## Milestone 0 — Foundation (Week of Jul 13 · 2 sessions)

| # | Task | Deliverable | Done when | Depends on |
|---|---|---|---|---|
| 0.1 | Registry + name check | Decision: final name | Name free on PyPI **and** npm (or conflicts accepted consciously); GitHub org/repo name reserved; decision logged | — |
| 0.2 | One-pager | `docs/one-pager.md` | Lifted from plan §1–2; one tagline chosen and frozen; read aloud once without wincing | 0.1 |
| 0.3 | Repo skeleton | `atlas/` repo, private | `spec/`, `docs/`, `skills/`, `adapters/` dirs; plan + one-pager committed; README stub with the one sentence | 0.1 |

**Exit gate M0:** the project has a name, a sentence, and a home. *Hard rule: no M1 work before 0.1 — attachment to an unavailable name compounds daily.*

---

## Milestone 1 — Schemas (Jul 14–26 · 5 sessions)

| # | Task | Deliverable | Done when | Depends on |
|---|---|---|---|---|
| 1.1 | Skill manifest schema | `spec/manifest.schema.yaml` | Fields: name, stage, consumes, maintains, emits, requires_slots; `schema_version: 1` line one; open questions punted to `spec/PUNTS.md` | 0.3 |
| 1.2 | Decision record + context-gap record schemas | `spec/records.schema.yaml` | Minimal fields survive the "useful in 6 months?" test; plan-clause addressing scheme chosen | 1.1 |
| 1.3 | Plan schema | `spec/plan.schema.yaml` | Addressable steps; footprint as globs; verification criteria machine-checkable in principle | 1.2 |
| 1.4 | Map schema | `spec/map.schema.yaml` | Entry types: architecture, conventions, decisions, sensitive-zones, components, library-manifest, standup-ledger-slot; directory-of-files layout; staleness metadata; path-scoping field (feeds context selection) | 1.3 |
| 1.5 | **The retrofit test** | 3 hand-written manifests | `/architect`, `/implement`, `/fix` from Flow expressed in schema 1.1; every hesitation recorded as a schema bug; config schema transcribed as warm-down (`spec/config.schema.yaml`) | 1.1–1.4 |

**Exit gate M1:** four+ versioned schemas, three real manifests, a punt list. → **Submit for adversarial review** (Claude session: break the schemas). Fixes are session 1.6 if needed.
**Predicted finding (verify or refute):** existing skills consume much, maintain nothing — context freshness has been Hills' manual job.

---

## Milestone 2 — The Hooks Spike (Jul 27–Aug 2 · 3 sessions) ⚠ LOAD-BEARING

| # | Task | Deliverable | Done when | Depends on |
|---|---|---|---|---|
| 2.1 | Hook inventory | `docs/spikes/hooks.md` | Claude Code hook surface enumerated: which events fire, what payloads carry, edit + decision interception feasibility mapped | 0.3 |
| 2.2 | Edit-stream proof | Working hook script | File edits captured at action time, written as records matching schema 1.2, on a throwaway repo | 2.1, 1.2 |
| 2.3 | Footprint reconcile proof | Working check | Given a plan file with globs, out-of-footprint edits flagged; in-footprint edits silent | 2.2, 1.3 |

**Exit gate M2 — the first kill-gate:** if hooks cannot deliver the streams without an orchestrator, **stop and redesign** before any core code: options are (a) verify-time-only reconciliation for v1, or (b) accept a thin wrapper. Decide consciously, update plan §3.4, log the decision. Do not drift into Option B by accident.

---

## Milestone 3 — Atlas-as-Files Dogfood (Aug 3–31 · 8 sessions + ambient daily use)

| # | Task | Deliverable | Done when | Depends on |
|---|---|---|---|---|
| 3.1 | Hand-build the Map for one real repo | `map/` in a Rain project (or 101skins) | Survey done manually per schema 1.4; architecture, conventions, sensitive zones populated; feels true | 1.4 |
| 3.2 | Rewrite 3 Flow skills as Atlas skills | `skills/atlas-plan`, `atlas-build`, `atlas-fix` (markdown-era versions) | Each reads the Map, produces schema-conformant plans/records; manifests validate by eyeball | 1.5, 3.1 |
| 3.3 | Wire hook scripts from M2 into daily work | Streams live | Decision + edit records accumulating on real tasks | 2.3, 3.2 |
| 3.4–3.7 | **Four weeks of real use** (ambient + 1 review session/week) | Weekly dogfood log | Each week answers, in writing: *what did I skip when rushed? when interrupted, was it right? did the Map help or lie?* | 3.3 |
| 3.8 | Dogfood verdict | `docs/dogfood-verdict.md` | Ceremony list (cut/automate), escalation hit-rate, Map-truth assessment; plan §3 amended where reality disagreed | 3.4–3.7 |

**Exit gate M3 — the second kill-gate:** if after four weeks Hills does not *prefer* working through the loop, the toll-booth risk is real. Fix the workflow before building the CLI — a CLI wrapped around ceremony is polished ceremony. This gate cannot be gamed; the skip-log is the evidence.

---

## Milestone 4 — Core CLI (Sep 1–Oct 4 · 12 sessions)

Language: Go, single binary. Build order follows dependency, not glamour:

| # | Task | Sessions | Done when |
|---|---|---|---|
| 4.1 | Schema loading + validation engine | 2 | All `spec/` schemas parsed; `atlas validate` passes the three M1 manifests, fails corrupted ones |
| 4.2 | `atlas doctor` | 1 | Secrets present, `.env` gitignore enforced (refuses otherwise), MCP reachability, Map staleness heuristic |
| 4.3 | Config → artifact compiler | 2 | `atlas sync` generates MCP config from declaration + env; idempotent; artifact clearly marked generated |
| 4.4 | Context compiler | 2 | Project context from Map per §3.9 table; task context assembled by footprint filter; human-authored section included verbatim |
| 4.5 | `atlas init` | 2 | Detect stack → ask only missing secrets → doctor → generate → install core skill files → run survey skill → present Map for blessing. Question count logged as DX budget |
| 4.6 | Loop commands: `plan`, `build`, `verify` | 2 | Thin orchestration of skill files + hook wiring + record I/O; conflict prompts follow fixed format (what/options/trade-offs) |
| 4.7 | `atlas help` (meta-skill) | 1 | Reads manifests + Map + config; reports installed skills, stage coverage, ownership matrix gaps, maintainer recency |

**Exit gate M4:** full loop runs end-to-end on the dogfood repo via the binary. Error messages audited against the standard: *what's wrong → why Atlas cares → what to do next.*

---

## Milestone 5 — Packaging + Tutorial (Oct 5–25 · 6 sessions)

| # | Task | Sessions | Done when |
|---|---|---|---|
| 5.1 | pip wrapper | 1 | `pip install <name>` on a clean machine yields working `atlas` |
| 5.2 | npm wrapper | 1 | Same via npm; both wrappers ship the one Go binary |
| 5.3 | The 15-minute tutorial | 2 | Written against a small public sample repo; ends on the visible Map-gets-smarter beat; timed at ≤15 min by following it literally |
| 5.4 | Docs pass | 2 | One-pager → README; compatibility-promise paragraph; adapter interface documented (for v1.5 authors); manifest authoring guide |

**Exit gate M5 — the stranger test:** one person who is not Hills (Jess is the obvious candidate) completes the tutorial unaided. Every place they stall is a P1 bug in docs or DX, fixed before launch.

---

## Milestone 6 — Launch (Oct 26–Nov 15 · 3 sessions + ambient)

| # | Task | Done when |
|---|---|---|
| 6.1 | Repo public, license (MIT or Apache-2 — decide, don't deliberate), v0.1.0 tagged | Links live |
| 6.2 | Demo video (≤3 min): init → plan → conflict prompt → `atlas help` introspection moment | Recorded, imperfect, published |
| 6.3 | Launch essay: "Prompting is the new programming — and it's in its spaghetti phase" | Published; shared in Claude Code/AI-dev communities; first entry of the writing project |

**Post-launch, in order:** `atlas-standup` per its scope doc (the ecosystem proof + second essay) → second agent adapter (the boundary test) → revisit parking lot.

---

## Critical Path & Parallelization

```
0.1 → M1 schemas → M3 dogfood → M4 core → M5 packaging → M6 launch
              ↘ M2 spike ↗
```
- M2 can start after 1.2/1.3 exist in draft — spike and late schemas interleave within the same weeks.
- 5.3/5.4 (docs) can begin during M4's back half.
- Nothing else parallelizes; respect the line. The dogfood month (M3) is the schedule's anchor — it is calendar-bound (four real weeks), not effort-bound, so it cannot be compressed by working harder.

## Cadence

Suggested weekly shape: two weekday evening sessions (build tasks) + one weekend session (review/log/next-week plan). The weekend session is also where the dogfood questions get answered in writing — skipping it is how the kill-gates get gamed.

## Kill/Pivot Gates, Collected

1. **M0:** name conflicts → rename now, cheaply.
2. **M2:** hooks insufficient → conscious redesign of streams, not accidental orchestrator.
3. **M3:** loop feels like toll booth → fix workflow before CLI; if unfixable, the honest outcome is "Flow stays a personal system" — smaller, still valuable, and better learned in August than November.
4. **M5:** stranger fails tutorial → DX is not done, launch waits.

## Risk Budget

The 40-session budget against ~45–50 available leaves ~20% slack. Slack burns first on: M2 surprises, M3-driven schema rework, M4.6 (loop orchestration is the least predictable build task). If slack exhausts, cut from M5 polish and M6 video quality — never from M3 duration or kill-gate honesty.

---

*First action: task 0.1, tonight, ten minutes. Everything else is downstream of a name.*
