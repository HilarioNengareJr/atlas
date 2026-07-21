# Agent Adapters

**Status: one adapter shipped (Claude Code), interface open, boundary unproven.**
Read the honesty section before building against this.

Atlas's core speaks only in Atlas primitives — plans, decision records, edit
events, conflicts, Map entries — and never learns which agent produced them. One
boundary, the **agent adapter**, translates a specific agent ecosystem into those
primitives. v1 ships exactly one adapter: Claude Code, chosen because its hook
surface is the richest available and therefore stretches the whole interface. The
second adapter is the first item of v1.5, and it is the test of whether this
boundary was drawn in the right place.

Public claim, stated precisely: *agent-agnostic by architecture; ships with
Claude Code support; adapter interface open.*

---

## What an adapter is responsible for

An adapter does three jobs and no others.

### 1. Capture the edit stream

Every file the agent writes becomes an `edit-event` record
([`spec/records.schema.yaml`](../spec/records.schema.yaml)), captured **at the
moment of action** — not reconstructed afterwards from a diff. This is the
load-bearing half of Atlas's observability: Verify reconciles what the agent
actually touched against the footprint the approved plan declared.

An `edit-event` needs:

```yaml
schema_version: 1
id: e7                      # kebab-case, unique within the task
type: edit-event
task: plan-2026-07-21-a     # or "unplanned" when no plan is active
skill: claude-code          # the adapter's own name, for hook-captured events
created_at: 2026-07-21T19:04:11Z
tool: Edit                  # see "The tool enum problem" below
file_path: /abs/path/to/file.go
in_footprint: false         # omit if there's no active plan to reconcile against
```

`in_footprint: false` is a **soft conflict** — it flags and batches for review at
Verify time. It does not block. That's the correct severity for v1 and it is not
a limitation to work around.

### 2. Declare capabilities honestly

Not every agent exposes the same surface. An adapter states what it can actually
do, and Atlas degrades gracefully and openly per ecosystem rather than pretending
uniformity. The three capabilities that matter:

| Capability | Question it answers |
|---|---|
| **Edit events** | Can you see file writes at action time? |
| **Decision visibility** | Can you see the agent's reasoning as it happens? |
| **Interruption** | Can you block an action before it lands? |

For Claude Code today: edit events **yes** (PostToolUse), decision visibility
**no** (no hook exposes agent reasoning; the transcript is only inspectable
between turns), interruption **not used** (PostToolUse cannot block, and
PreToolUse `deny` on sensitive zones is deferred past v1).

Because decision visibility is unavailable, the **decision stream is
skill-emitted, not adapter-captured** — the Build skill narrates its own
deviations as `decision` records. Adapters emit `edit-event` and nothing else.
An adapter that could see reasoning would still not emit decisions in v1; that
split is deliberate, so Verify always has two independently-sourced sides to
reconcile.

### 3. Own the context target format

The Map is the source; context files are compiled build outputs. The adapter owns
what that output looks like for its ecosystem — for Claude Code, `CLAUDE.md` plus
skill files. Regeneration is idempotent and the files are never hand-edited, so
drift between what Atlas knows and what the agent sees is structurally
impossible.

---

## What the interface actually is right now

**There is no Go interface.** No `type Adapter interface` exists in this repo,
and pretending otherwise would be the kind of aspirational documentation Atlas
exists to stop.

What exists is a contract by artifact: the shipped adapter is a standalone
Python script wired into Claude Code's hook config, which writes schema-valid
records into `.atlas/records/`. The core reads those records. The seam is the
record file, not a function signature.

See [`../adapters/claude-code/README.md`](../adapters/claude-code/README.md) for
the working example and its wiring, and
[`spikes/hooks.md`](spikes/hooks.md) for the hook surface it was built against.

Design rules that example obeys, which a second adapter should copy:

- **Stdlib only.** An adapter must not add install prerequisites to the user's
  project.
- **Always exit 0.** Observability must never break the session it observes. A
  broken adapter loses records; it does not lose the user's work.
- **Read only what you need.** The Claude Code hook reads `file_path` and nothing
  else from the edit payload — deliberately staying off field names that a
  fast-moving product might rename.

---

## Known problems for the second adapter

Writing this document surfaced two, and they're recorded here rather than
discovered by whoever builds adapter number two.

### The tool enum problem

`edit_event.tool` in `records.schema.yaml` is a **closed enum** of Claude Code's
tool names:

```yaml
tool:
  enum: [Edit, Write, MultiEdit, NotebookEdit]
```

Those are one vendor's product names sitting in a schema the core owns. A second
adapter whose agent calls the operation `write_file`, `apply_patch`, or anything
else cannot emit a valid record without either lying about the tool name or
changing a core schema — which is exactly the coupling the adapter boundary is
supposed to prevent.

This is the sharpest available test of whether the boundary is real, and it is
tracked as **P15** in [`../spec/PUNTS.md`](../spec/PUNTS.md).

### Records have no agreed location

`.atlas/records/` is what the M2 spike used. It is a candidate, not an adopted
convention — nothing in the schemas states where plan or record instances live on
disk, which is why `atlas validate` needs an explicit `--schema` override to check
one. Tracked as P11. A second adapter should expect this to move.

---

## Building one (v1.5)

The interface is open, not stable. If you're building an adapter before v1.5
lands, the order that will waste the least of your time:

1. Map your agent's hook or event surface — what fires, what payload it carries,
   whether it can block. Write it down the way [`spikes/hooks.md`](spikes/hooks.md)
   does.
2. Answer the three capability questions honestly. A "no" is fine and expected; a
   wrong "yes" produces a Map that lies.
3. Emit `edit-event` records at action time. Validate them with
   `atlas validate --schema=records <file>`.
4. Open an issue about the tool enum before you work around it. Your workaround
   is data about where the boundary really belongs.
