# Compatibility Promise

**In force from v0.1.0. Nothing is published yet, so nothing below binds today.**

Atlas accumulates understanding, and understanding that an upgrade can destroy is
worthless. So the promise is about your files, not our code: **Atlas will never
break your Map without giving you a way to carry it forward.** The binary follows
semver — a major bump is the only place behaviour is allowed to change out from
under you, minor adds, patch fixes. Your artifacts (Map entries, manifests,
plans, records, config) carry their own `schema_version`, which moves
independently of the binary version and only ever goes up by one. When a schema
version bumps, `atlas migrate` upgrades your files in place; when a manifest
field is deprecated, it keeps working — with a warning — until the next major
release, which is at least one full minor cycle of being told. If we ever cannot
give you a migration path, we don't ship the change.

---

## The specifics

### Binary version (semver)

| Bump | Means |
|---|---|
| **Patch** (0.1.**1**) | Bug fix. No behaviour you relied on changes. |
| **Minor** (0.**2**.0) | New commands, new flags, new optional schema fields. Everything that worked before still works. |
| **Major** (**1**.0.0) | Behaviour may change or be removed. Only place a deprecation is allowed to complete. |

Pre-1.0, the minor position acts as the major one — that's standard semver
practice for 0.x, and it means 0.1 → 0.2 may break things. Once 1.0 ships, the
table above is literal.

### Schema version

Every schema file's first line is `schema_version: 1`. This is deliberate and
non-negotiable — versioning cannot be retrofitted onto artifacts that never
declared a version.

- The schema version is **not** the binary version. A binary at v0.4.0 may still
  speak `schema_version: 1`.
- It increments by one, never skips.
- A binary states which schema versions it can read. Reading a newer artifact
  than it understands is a clear error, not a silent misparse.

### Deprecating a manifest field

1. The field keeps working exactly as before.
2. `atlas validate` warns on it, naming the replacement.
3. It stops working no earlier than the next major release.

So the shortest possible life of a deprecated field is one full minor cycle of
warnings. You never find out a field is gone by having it silently ignored.

### The Map

The Map is plain files in your repo, git-versioned, readable without Atlas
installed. That's the real guarantee — the worst case for any Atlas change is
that you read your own files by hand.

`atlas migrate` is **designed in v1 and built when first needed** — the first
schema bump is what builds it, and that bump does not ship until it exists. The
thesis is that a project's understanding compounds; if an upgrade could wipe it,
the thesis dies at the first version bump.

### What is explicitly not promised

- **Generated artifacts.** Compiled MCP config and compiled context files are
  build outputs. Their format can change in any release. Don't hand-edit them and
  you'll never notice.
- **Internal Go packages.** `internal/` is not a public API. Atlas is a CLI, not
  a library.
- **Exit codes and output text before v1.0.** `--json` output stabilises at 1.0;
  until then, script against it at your own risk.
- **The adapter interface before v1.5.** See [`adapters.md`](adapters.md) — one
  adapter has shipped, and the second one is what proves the boundary is in the
  right place. It will move if it's wrong.
