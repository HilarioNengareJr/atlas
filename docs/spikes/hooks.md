# Spike — Claude Code Hooks (M2.1)

**Question this spike answers:** can Claude Code hooks deliver Atlas's two
streams — edit stream and decision stream — without an orchestrator wrapping
the agent? (Exit gate M2; plan §3.4, §8.2.)

**Verdict up front: YES for the edit stream, YES-BY-DESIGN for the decision
stream.** Edits are hookable at the moment of action with full payloads.
Decisions have no dedicated hook event — but Atlas's design never needed one:
decision records are emitted by skills during Build; hooks contribute the
edit stream and checkpoint moments. No orchestrator required. Kill-gate
passed; no redesign of §3.4.

Sources: code.claude.com/docs/en/hooks, /hooks-guide, /agent-sdk/hooks,
fetched 2026-07-13 via live docs (not training memory). Items marked
**[verify at wiring]** must be confirmed against a real hook before M3.3.

---

## Hook events that matter to Atlas

| Event | Fires | Can block | Atlas use |
|---|---|---|---|
| `PreToolUse` | before a tool runs | yes (`permissionDecision`: allow/deny/ask) | later: hard-conflict on sensitive-zone edits (v1.5+) |
| `PostToolUse` | after a tool succeeds | no | **the edit stream** — capture + reconcile |
| `PostToolUseFailure` | after a tool fails | no | ignore failed edits (nothing changed on disk) |
| `UserPromptSubmit` | before each user prompt processes | yes | checkpoint: task switching |
| `Stop` | when the agent finishes a turn | yes | checkpoint: batch soft-conflict summary at Verify |
| `SessionStart` | once per session | no | load current task/plan context |
| `PreCompact` | before context compaction | yes | snapshot opportunity |

Events on every payload: `session_id`, `transcript_path` (JSONL of the
conversation), `cwd`, `hook_event_name`, plus `tool_name`/`tool_input` on the
tool events and `tool_output` on PostToolUse.

## Edit payloads

Matcher for the edit stream: `Edit|Write|MultiEdit|NotebookEdit` on
`PostToolUse`.

- `Write` → `tool_input.file_path`, `tool_input.content`
- `Edit` → `tool_input.file_path`, old/new strings
- `MultiEdit` → `tool_input.file_path`, `tool_input.edits[]`
- `NotebookEdit` → `tool_input.notebook_path` (not `file_path`) **[verify at wiring]**

**Field-name discrepancy [verify at wiring]:** the docs pass reported Edit's
strings as `original_str`/`new_str`; the CLI's own Edit tool takes
`old_string`/`new_string`, and hook payloads mirror `tool_input` verbatim, so
`old_string`/`new_string` is almost certainly what arrives. The adapter
script tolerates both. This is exactly the fast-moving-surface risk plan
§8.2 holds — the adapter absorbs it, the core never sees it.

## Configuration shape

Project-scoped, in `.claude/settings.json`:

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Edit|Write|MultiEdit|NotebookEdit",
        "hooks": [
          {
            "type": "command",
            "command": "python3 \"${CLAUDE_PROJECT_DIR}/adapters/claude-code/hooks/edit_stream.py\""
          }
        ]
      }
    ]
  }
}
```

`${CLAUDE_PROJECT_DIR}` resolves to the project root. Payload arrives as JSON
on stdin. Exit 0 = success; stdout JSON can carry `systemMessage` (shown to
the user) — the reconciler uses it to flag out-of-footprint edits without
blocking (PostToolUse cannot block, which suits soft conflicts: flag, batch,
review at Verify).

## Output semantics (the parts Atlas relies on)

- Exit 0 + JSON stdout → parsed; `systemMessage` surfaces a warning.
- Exit 2 → blocking error (only on blockable events).
- Hooks run as subprocesses, parallel per event, default timeout 600s. No
  persistent state between invocations — state lives in files (`.atlas/`),
  which is Atlas's model anyway.

## Decision stream assessment

There is no "agent made a decision" hook event, and the transcript
(`transcript_path`) is only inspectable between turns. Atlas's design already
accounts for this: **decision records are emitted by the Build-stage skill as
part of its instructions** (schema-shaped, per `spec/records.schema.yaml`),
not intercepted by machinery. Hooks add the *involuntary* stream (edits —
which cannot be forgotten or embellished) while skills own the *narrated*
stream (decisions). The two reconcile at Verify: an out-of-footprint edit
with no decision record is precisely a soft conflict.

Conscious acceptance, logged here per the kill-gate instructions: v1 does
not attempt hook-side decision detection. Revisit only if M3 dogfooding shows
skills under-reporting deviations.

## Limitations that shaped the scripts

- PostToolUse observes; it cannot block — soft conflicts only. Hard
  conflicts (sensitive zones) need PreToolUse `deny`, deferred past v1.
- Payload `tool_input` shapes are product surface, not contract — the
  adapter normalizes them into Atlas primitives (edit-event records) so the
  core never touches them (plan §3.6).
- Hook scripts must be dependency-free (they run on user machines before any
  Atlas install) — stdlib Python only in the spike; the Go binary absorbs
  this at M4.

## Proofs

- **2.2 edit stream:** `adapters/claude-code/hooks/edit_stream.py` — reads a
  PostToolUse payload on stdin, normalizes it, writes an `edit-event` record
  (JSON, validating against `spec/records.schema.yaml`) to
  `.atlas/records/`. Proven on a throwaway repo with synthetic payloads for
  all four tools and both Edit field-name variants.
- **2.3 footprint reconcile:** same script — if `.atlas/plan.json` exists,
  the edit's path is matched against the plan's `footprint` globs
  (`**`-aware). In-footprint: silent record. Out-of-footprint: record with
  `in_footprint: false` plus a `systemMessage` flag. No plan: record written
  unreconciled.
