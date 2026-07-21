# Decisions

Plain-markdown decision log. Superseded by schema-shaped decision records (spec/records.schema.yaml) once Milestone 1 lands.

## 2026-07-21 — First two Go dependencies: `santhosh-tekuri/jsonschema/v6` and `gopkg.in/yaml.v3`

**What:** M4.1 (`atlas validate`) takes Atlas's first two Go dependencies. Schema
validation itself uses `github.com/santhosh-tekuri/jsonschema/v6`. Reading any of the
project's own YAML files (schemas or instances) uses `gopkg.in/yaml.v3`. Module:
`github.com/HilarioNengareJr/atlas`.

**Facts at decision time:**
- Every schema in `spec/` declares `$schema: .../2020-12/schema` explicitly — the
  validator has to support draft 2020-12 for real, not degrade silently on unsupported
  keywords. `xeipuuv/gojsonschema` (the most commonly reached-for Go option) only
  supports draft-07.
- `santhosh-tekuri/jsonschema/v6` is the actively maintained, spec-compliant 2020-12
  implementation; single dependency (pulls in `golang.org/x/text` transitively for
  localized error messages).
- Proved end-to-end before writing any package code (throwaway probe, deleted after):
  decode a real schema YAML file into `any` via `yaml.v3`, register it with
  `Compiler.AddResource(schemaDoc["$id"], schemaDoc)`, `Compile` it, `Validate` a real
  instance (`spec/examples/manifests/architect.yaml`) — passes clean; a deliberately
  broken instance fails with the expected `InstanceLocation`/message. One real bug found
  in the process: `ErrorKind.LocalizedString(nil)` segfaults — a real `*message.Printer`
  is required, not `nil`.
- YAML decodes numbers as Go `int` (not `float64`); confirmed this validates correctly
  against the library's `const`/`type` keyword checks in the same probe — no JSON/YAML
  type-shape mismatch in practice for the schemas as currently written.
- `code-standards.md` already named this exact fork explicitly: *"Go core has no
  framework dependencies worth naming until M4 forces the question — decide then, log it
  in `docs/decisions.md`."* M4 is now.

**Alternatives considered:** (a) hand-roll a validator sized to the schemas' current
(modest) keyword surface — rejected: correct today, but a silent liability the moment a
schema gains a keyword the hand-rolled walker doesn't know about, and the "zero
dependencies" instinct is about avoiding *unnecessary* dependencies, not about refusing
the one M4 explicitly anticipated forcing; (b) `xeipuuv/gojsonschema` — rejected outright,
draft-07 only, would silently mis-validate 2020-12-only keywords.

**Why:** Spend the one dependency `code-standards.md` already said M4 would force, on the
library that actually matches what every schema declares, rather than reinventing a
narrower version of the same thing.

**Follow-up:** If a schema ever needs a keyword this library doesn't support, that's a
new dated decision, not a silent workaround.

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
