#!/usr/bin/env python3
"""Atlas edit stream — Claude Code adapter hook (M2.2 + M2.3 spike).

Wire on PostToolUse with matcher Edit|Write|MultiEdit|NotebookEdit (see
docs/spikes/hooks.md). Reads the hook payload on stdin, normalizes it into an
edit-event record (spec/records.schema.yaml), reconciles against the active
plan's footprint when one exists, and writes the record to .atlas/records/.

Dependency-free by requirement: hook scripts run on user machines before any
Atlas install exists, so stdlib only. The plan is read as JSON
(.atlas/plan.json) for the same reason — no YAML parser in the stdlib.

Never blocks, never exits nonzero on bad input: a broken observability hook
must not break the user's session. Failures go to stderr and the hook exits 0.
"""
import json
import os
import re
import sys
from datetime import datetime, timezone

EDIT_TOOLS = {"Edit", "Write", "MultiEdit", "NotebookEdit"}


def glob_to_regex(glob: str) -> str:
    """Translate a footprint glob to a regex. ** crosses directories, * doesn't."""
    out, i = [], 0
    while i < len(glob):
        c = glob[i]
        if c == "*":
            if glob[i : i + 2] == "**":
                out.append(".*")
                i += 2
                if i < len(glob) and glob[i] == "/":
                    i += 1  # "**/" also matches zero directories
                continue
            out.append("[^/]*")
        elif c == "?":
            out.append("[^/]")
        else:
            out.append(re.escape(c))
        i += 1
    return "^" + "".join(out) + "$"


def in_footprint(rel_path: str, globs: list) -> bool:
    return any(re.match(glob_to_regex(g), rel_path) for g in globs if isinstance(g, str) and g)


def main() -> int:
    try:
        payload = json.load(sys.stdin)
    except Exception as exc:  # malformed payload: observe-only hooks stay silent
        print(f"atlas edit_stream: unreadable payload: {exc}", file=sys.stderr)
        return 0

    tool = payload.get("tool_name", "")
    if tool not in EDIT_TOOLS:
        return 0  # not an edit; matcher should prevent this, belt-and-braces

    tool_input = payload.get("tool_input") or {}
    # NotebookEdit uses notebook_path; the docs pass showed field-name drift
    # on Edit strings — take file identity defensively (hooks.md, verify at wiring).
    file_path = tool_input.get("file_path") or tool_input.get("notebook_path") or ""
    if not file_path:
        print("atlas edit_stream: edit payload without a file path", file=sys.stderr)
        return 0

    project_dir = os.environ.get("CLAUDE_PROJECT_DIR") or payload.get("cwd") or os.getcwd()
    atlas_dir = os.path.join(project_dir, ".atlas")
    records_dir = os.path.join(atlas_dir, "records")
    os.makedirs(records_dir, exist_ok=True)

    rel_path = os.path.relpath(file_path, project_dir) if os.path.isabs(file_path) else file_path

    task = "unplanned"
    footprint = None
    plan_path = os.path.join(atlas_dir, "plan.json")
    if os.path.exists(plan_path):
        try:
            with open(plan_path) as f:
                plan = json.load(f)
            task = plan.get("id", task)
            footprint = plan.get("footprint")
        except Exception as exc:
            print(f"atlas edit_stream: unreadable plan.json: {exc}", file=sys.stderr)

    now = datetime.now(timezone.utc)
    record = {
        "schema_version": 1,
        "id": f"edit-{now.strftime('%Y%m%dt%H%M%S%f')}",
        "type": "edit-event",
        "task": task,
        "skill": "claude-code",
        "created_at": now.strftime("%Y-%m-%dT%H:%M:%SZ"),
        "tool": tool,
        "file_path": file_path,
    }

    flagged = False
    if isinstance(footprint, list) and footprint:
        record["in_footprint"] = in_footprint(rel_path, footprint)
        flagged = not record["in_footprint"]

    record_path = os.path.join(records_dir, record["id"] + ".json")
    with open(record_path, "w") as f:
        json.dump(record, f, indent=2)
        f.write("\n")

    if flagged:
        # PostToolUse cannot block — soft conflict: flag now, batch at Verify.
        print(json.dumps({
            "systemMessage": (
                f"Atlas: edit outside the approved footprint of plan "
                f"'{task}': {rel_path} (soft conflict, batched for Verify)"
            )
        }))
    return 0


if __name__ == "__main__":
    sys.exit(main())
