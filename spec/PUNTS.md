# Punt List

Open schema questions, deliberately not blocking Milestone 1. Each punt names
the schema that raised it and what forces a decision later. Punted ≠ forgotten:
anything resolved gets a dated decision in `docs/decisions.md` and is struck
from this list.

## P1 — Skill name namespacing (manifest)

Core skills use the `atlas-` prefix; the retrofit skills (`architect`,
`build`, `review`) don't. The schema allows any kebab-case name. Does the
ecosystem need namespacing (e.g. `author/skill`) before external skills exist?
**Forces a decision:** first external skill manifest (Phase 4).

## P2 — Is `description` mandatory on manifests? (manifest)

Made optional: the manifest is for machines. But `atlas help` would read it,
and an undescribed skill is unfriendly. **Forces a decision:** building
`atlas help` (M4.7).

## P3 — How plan approval is recorded (plan)

The plan is a contract and approval is the contract moment, but the schema has
no `approved` field. In-file flag? Separate record? CLI state? Recording it
in-file makes the artifact self-contained but mutable after approval.
**Forces a decision:** `atlas plan`/`build` implementation (M4.6).

## P4 — Global record id uniqueness (records)

Record ids are unique within their task only. Cross-task references (e.g. a
decision superseding an older one) would need a global scheme
(`<plan-id>/<record-id>`? uuid?). **Forces a decision:** first skill that
needs to cite a record from another task — likely atlas-log (M3/M4).

## P5 — Per-type structure of Map entry `content` (map)

`content` is free-form (string or object) in v1. Structured fields per entry
type (e.g. components with name/path/props) would help compilation but risk
inventing fields dogfooding won't confirm. **Forces a decision:** M3 dogfood —
whatever structure the hand-built Map keeps reaching for, schema it.

## P6 — `standup-ledger-slot` is dead weight by design (map, manifest)

Reserved for the post-launch atlas-standup skill; in v1 nothing consumes or
maintains it, which is exactly what `atlas help` flags as dead weight. Keep it
reserved (the build plan lists it) or cut it until the skill exists?
**Forces a decision:** M4.7 (`atlas help` would flag it) or the standup scope doc.

## P7 — Is the approved plan the only consumable artifact? (manifest)

The retrofit test showed build/verify skills consume the approved plan, which
is not a Map entry type — `consumes` gained the `approved-plan` value. Are
there other artifacts (compiled context? the punt list itself?) that
manifests should be able to declare, or is one special case the right size?
**Forces a decision:** M3 — rewriting the three Flow skills as Atlas skills
will show what they actually read.

## P8 — Entry-type enum is triplicated (manifest, records, map)

The seven-entry-type list appears verbatim in three self-contained schema
files; map.schema.yaml is designated canonical and all three copies carry
sync comments, but nothing mechanical enforces the three-way match. Options:
a bundling step so `$ref` can cross files, or a consistency check inside
`atlas validate`. **Forces a decision:** M4.1 (the validation engine).

## P9 — Step-id uniqueness is not schema-enforceable (plan)

Found by the M1 adversarial review: two steps sharing `id: s1` validate, which
makes record `clause` citations ambiguous. `uniqueItems` now rejects
byte-identical steps, but per-property uniqueness cannot be expressed in JSON
Schema — the real check is cross-item logic in `atlas validate`.
**Forces a decision:** M4.1 (the validation engine).

## P10 — Map filename↔id agreement is invisible per-file (map)

Found by the M1 adversarial review ("renamed files mid-task" seed): renaming
`map/architecture/loop.yaml` to `zoop.yaml` still validates — the id/filename
convention is a filesystem property no per-file schema sees. This bites at M3,
when the Map is hand-built and hand-edited, before `atlas validate` exists.
Interim guard: the M3 dogfood checklist should include an id↔filename sweep.
**Forces a decision:** M3 setup (interim guard) and M4.1 (real check).

## P11 — Plan/records instance schema dispatch has no location convention (plan, records)

Manifests (`skills/atlas-<name>/manifest.yaml`), Map entries (`map/<type>/<id>.yaml`),
and config (`atlas.config`) all have a stated directory/filename convention, so
`atlas validate` can tell which schema a given file should check against just from its
path. Plan and records instances have no such convention — nothing says where a task's
plan or its emitted records actually live on disk. A previous (skipped) M3 architecture
entry proposed `.atlas/plan.json` (singular, active plan) + `.atlas/plans/` (archive) +
`.atlas/records/`, but M3 was skipped, that convention was never built or tested, and it
uses JSON where the project's naming convention implies YAML instances elsewhere — a
candidate, not an adopted decision. Interim: `atlas validate --schema=plan|records <file>`
takes an explicit override. **Forces a decision:** whenever `atlas plan`/`atlas build`
(M4.6) need a real location to write these artifacts to.

## P12 — MCP reachability is declared, not proven (config)

`atlas doctor` (M4.2) checks that every MCP in `atlas.config` is *declared*
correctly — a slot the schema recognizes, and every name in its `env` list
resolving to a non-empty value. It does not open a connection, so a
perfectly-declared MCP whose server is down, misconfigured, or unreachable
still passes. Proving reachability needs an MCP client in Go, which nothing in
M4 builds, plus a policy on timeouts and on tests that must not touch the
network. **Forces a decision:** whenever a real dogfood session loses time to
an MCP that doctor called healthy.

## P13 — The Map staleness threshold is hardcoded (map, config)

`atlas doctor` flags a Map entry stale at `now - last_confirmed > 30 days`. The
number is a constant in the doctor package: no schema field holds it, and
adding one to `config.schema.yaml` for a single tunable was scope M4.2 did not
own. Different entry types plausibly deserve different windows — `decisions`
ages very differently from `components`. **Forces a decision:** the first time
30 days is visibly wrong on a real Map (M3-style use, or M4.5 `atlas init`).

## P14 — Atlas's schemas are not reachable from a user's repo (spec, all schemas)

`atlas validate` and `atlas doctor` both locate `spec/*.schema.yaml` by walking
upward from the target for `spec/manifest.schema.yaml`. That only ever succeeds
inside the Atlas repo itself — a real Atlas-configured project has an
`atlas.config` and a `map/`, but no `spec/`. M4.1 shipped with this: validate
exits 2 on such a repo. M4.2 refuses to inherit it for the checks that don't
need schemas, so doctor degrades — an unreachable `spec/` becomes one Finding
and every non-schema check still runs. The real fix is `go:embed`-ing the five
schemas into the binary, which is small but raises the question this punt is
really about: when a repo carries its *own* `spec/`, does the repo's copy or
the binary's win? Skew between a user's pinned schemas and a newer binary is
exactly what `atlas migrate` was reserved for. **Forces a decision:** M4.5
(`atlas init`, which must work on a repo that has no Atlas files at all).
