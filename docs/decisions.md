# Decisions

Plain-markdown decision log. Superseded by schema-shaped decision records (spec/records.schema.yaml) once Milestone 1 lands.

## 2026-07-13 — M2 kill-gate: hooks deliver the streams; `edit-event` record type added

**What:** The hooks spike (M2) passed its kill-gate: Claude Code hooks deliver the edit stream without an orchestrator — v1 stays on hooks, no wrapper, plan §3.4 unchanged. The decision stream is confirmed as skill-emitted (records narrated by the Build skill), with hooks providing the involuntary edit stream that Verify reconciles against it. To carry edit events, `spec/records.schema.yaml` gained a third record type, `edit-event` (tool, file_path, in_footprint), emitted by adapters rather than skills — the envelope's `skill` field now covers both.

**Facts at decision time:**
- PostToolUse fires per tool call with `tool_name` + `tool_input` (verified against live docs 2026-07-13; payload shapes recorded in `docs/spikes/hooks.md`).
- PostToolUse cannot block — which matches soft-conflict semantics exactly (flag, batch, review at Verify).
- No hook event exposes agent decisions; the transcript is only inspectable between turns.
- Proofs: `adapters/claude-code/hooks/edit_stream.py` produced 7/7 schema-valid records across all four edit tools, both Edit field-name variants, in/out-of-footprint and no-plan cases.

**Alternatives considered:** (a) wrapper process around the agent for true streaming — rejected: v1 law ("rides on hooks, does not wrap"), and nothing in the proofs needed it; (b) keep edit events out of the record schema as ephemeral stream data — rejected: build plan 2.2 explicitly requires edits written as schema 1.2 records, and Log needs them as Map citizens.

**Why:** The edit stream is the load-bearing half (§3.4) and it works with payloads to spare. Skills narrating decisions plus hooks capturing edits gives Verify both sides of the reconciliation without any surface Atlas doesn't control beyond the thin adapter.

**Follow-up:** Verify Edit string field names and NotebookEdit's `notebook_path` against a live hook at M3.3 wiring (`docs/spikes/hooks.md`, "verify at wiring" items). Hard-conflict blocking via PreToolUse `deny` on sensitive zones stays deferred past v1.

## 2026-07-11 — Install name: `atlasdev`

**What:** The package/install name on PyPI and npm is `atlasdev`. The project name stays **Atlas** (branding). The GitHub repo is `HilarioNengareJr/atlas`.

**Facts at decision time:**
- `atlas` — taken on PyPI and npm.
- `atlas-dev` (the plan §6 candidate) — free on PyPI, but squatted on npm by an empty v0.0.1 package (no description, untouched since 2022-04-11).
- `atlas-cli` — free on PyPI, taken on npm. `atlas-framework` — taken on PyPI, free on npm.
- `atlasdev` — free on both.

**Alternatives considered:** (a) `atlas-dev` on PyPI plus an npm abandoned-package dispute — best name if it works, but weeks of uncertainty; (b) split names per registry — permanently inconsistent install line in every doc.

**Why `atlasdev`:** one name everywhere, available today, no dependence on npm's dispute process. Install line: `pip install atlasdev` / `npm install -g atlasdev`.

**Follow-up:** plan §6 references `atlas-dev` — treat `atlasdev` as the settled value from this date forward.
