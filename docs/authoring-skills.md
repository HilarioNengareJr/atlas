# Writing an Atlas Skill Manifest

A skill without a manifest is a prompt in a folder. The manifest is what lets
Atlas reason about your skill alongside every other one — which stage it serves,
what it reads, and **what it owes**.

Every field is checked by `atlas validate`, so you can be wrong out loud rather
than wrong silently.

---

## The shortest complete manifest

Skills live at `skills/<name>/manifest.yaml`. That path is how `atlas validate`
knows the file is a manifest — put it elsewhere and it won't be discovered.

```yaml
schema_version: 1
name: atlas-survey
description: Read a repo cold and build the initial Map.
stage: survey
consumes: []
maintains:
  - architecture
  - conventions
  - components
emits:
  - context-gap
requires_slots:
  - survey
```

Seven fields. Six are required — `description` is the only optional one.

---

## The fields

### `schema_version` — always `1`

Line one, every time. Non-negotiable: versioning cannot be retrofitted onto
artifacts that never declared a version. See
[`compatibility.md`](compatibility.md).

### `name` — kebab-case

Pattern: `^[a-z][a-z0-9]*(-[a-z0-9]+)*$`. Atlas core skills use the `atlas-`
prefix. Whether contrib skills need a namespace like `author/skill` is an open
question (PUNTS.md P1) — for now, any kebab-case name validates.

### `description` — optional, but write one

The manifest is for machines, which is why this is optional. `atlas help` reads
it, though, and an undescribed skill is unfriendly to the human trying to work
out what they installed. One line. (PUNTS.md P2 — this may become required.)

### `stage` — exactly one

`survey` · `chart` · `build` · `verify` · `log` · `meta`

**A skill that wants two stages is two skills.** This is v1 law, and the schema
enforces it by shape.

The stage implies the artifact you produce — a `chart` skill produces a plan —
which is why there is no `produces` field. Adding one would let a chart skill
claim it produces something else, which is worse than not asking.

There is also **no dependency field at all**. Skill-to-skill dependencies don't
exist in v1, enforced by the field's absence. If your skill needs another skill
to have run first, express that through the Map, not through a dependency edge.

### `consumes` — what you read

Map entry types, plus the special value `approved-plan` for build- and
verify-stage skills that read the plan artifact itself.

The seven Map entry types:

```
architecture · conventions · decisions · sensitive-zones
components · library-manifest · standup-ledger-slot
```

An entry type that nothing consumes is dead weight, and `atlas help` flags it.

### `maintains` — what you owe

This is the field that makes Atlas work, and the one people leave empty.

`maintains` is the list of Map entry types **you are responsible for keeping
true**. Not what you write once — what you keep correct over time. An empty list
is legal for any single skill. But across all installed skills, every entry type
needs at least one maintainer, or that type goes stale with nobody accountable.

That's the **ownership matrix**, and `atlas validate` checks it across every
manifest in `skills/`. Run it on a repo with no maintainers and you get:

```
Map entry type "architecture"
  what: No installed skill declares "architecture" in its `maintains` list.
  why:  An entry type nothing maintains goes stale with no one responsible for
        keeping it true — staleness by design, not by accident.
  next: Add "architecture" to some skill's `maintains` list, or remove the entry
        type from map.schema.yaml if it's genuinely unused.
```

Seven of those is what this repo prints today, because no skills are installed
yet.

**Don't fix that by claiming maintainership you won't honour.** A manifest that
lies is worse than one that's modest — the matrix looks complete and the Map
rots anyway. Claim it when it's true.

### `emits` — `decision`, `context-gap`, or neither

Record types your skill can produce. Empty is legal and common: **following the
plan verbatim is silent.** A record exists only when something deviated or
something was missing.

- **`decision`** — you deviated from or interpreted the approved plan.
- **`context-gap`** — you had to discover something mid-task that your `consumes`
  context should have given you. This is the signal that feeds back into the Map.

`edit-event` is a third record type but it isn't listed here — adapters emit
those, not skills. See [`adapters.md`](adapters.md).

### `requires_slots` — which MCPs you need

Same six names as the stages: `survey` · `chart` · `build` · `verify` · `log` ·
`meta`. These are MCP **slot types**, declared in `atlas.config` and checked by
`atlas init` up front. No slot, no config entry — your skill won't start with a
missing dependency and discover it halfway through.

---

## Check your work

```bash
atlas validate                          # whole repo: schemas + ownership matrix + P8/P10
atlas validate skills/my-skill/manifest.yaml   # one file
atlas validate --json                   # machine-readable findings
```

Exit codes: `0` clean, `1` findings, `2` couldn't run.

Every finding follows the same three-part shape — **what's wrong → why Atlas
cares → what to do next**. If you ever get a finding that doesn't tell you what
to do next, that's a bug worth reporting.

One current limitation: `atlas validate` finds its schemas by walking up for
`spec/manifest.schema.yaml`, which only exists inside the Atlas repo itself. In
your own project it will exit 2 until the schemas are embedded in the binary
(PUNTS.md P14).

---

## Worked example

Here is the real Flow `/architect` skill written as a manifest, from
[`../spec/examples/manifests/architect.yaml`](../spec/examples/manifests/architect.yaml):

```yaml
schema_version: 1
name: architect
description: >-
  Think through a feature with the developer before code — align language,
  surface decisions, write the implementation plan.
stage: chart
consumes:
  - architecture
  - conventions
  - components
  - library-manifest
maintains: []
emits: []
requires_slots: []
```

Note `maintains: []` and `emits: []`. This skill reads four entry types and keeps
none of them true — and that is recorded honestly rather than dressed up. It was
also the predicted finding of the retrofit test that produced this file:
**existing skills consume much and maintain nothing.** Keeping context fresh had
been a manual human job all along.

If your first manifest looks like this one, you've found something real about
your skill, not made a mistake filling in a form.

Two more worked examples sit alongside it:
[`build.yaml`](../spec/examples/manifests/build.yaml) and
[`review.yaml`](../spec/examples/manifests/review.yaml).
