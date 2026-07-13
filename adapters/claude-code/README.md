# Claude Code Adapter (embryo)

The one adapter v1 ships (plan §3.6). Translates Claude Code's hook surface
into Atlas primitives — the core never learns where events come from.

Current contents (M2 spike deliverables):

- `hooks/edit_stream.py` — PostToolUse hook: normalizes edit payloads into
  `edit-event` records (`spec/records.schema.yaml`), reconciles against
  `.atlas/plan.json`'s footprint, writes to `.atlas/records/`. Out-of-footprint
  edits get a `systemMessage` flag (soft conflict — PostToolUse cannot block,
  which is the correct severity for v1).

## Wiring (manual until `atlas init` exists)

In the target project's `.claude/settings.json`:

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

For a project that consumes Atlas rather than being Atlas, point the command
at wherever the adapter lives and set `CLAUDE_PROJECT_DIR` accordingly.

## Verified vs. reported

The payload shapes this adapter handles follow `docs/spikes/hooks.md`. Two
items are docs-reported but not yet verified against a live hook (do this at
M3.3 wiring): the exact Edit string field names (`old_string`/`new_string`
vs `original_str`/`new_str` — the script never reads them, only `file_path`,
precisely to stay off that moving surface) and NotebookEdit's `notebook_path`
(handled defensively).

Design rules the script obeys: stdlib only (no install prerequisites),
exit 0 always (observability must never break the session), plan read as
JSON because stdlib has no YAML (the Go binary absorbs this at M4).
