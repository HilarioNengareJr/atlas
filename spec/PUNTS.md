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
