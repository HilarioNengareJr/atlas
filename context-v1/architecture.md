# Atlas — Architecture Log

This file accumulates the implementation plan for each unit of work. Newest entries at the bottom.

## Implementation Plan — Milestone 0: name, sentence, home (2026-07-11)

### What we are building
The foundation of the Atlas project in `/Users/hilariojuniornengare/atlas`: a settled install name (checked against PyPI and npm, decision logged), a one-pager with one frozen tagline lifted from `atlas-plan.md` §1–2, and a private GitHub repo (`HilarioNengareJr/atlas`) holding the skeleton directories, both plan documents, the one-pager, and a README stub carrying the one sentence.

### Language we agreed on
- **Project name**: "Atlas" — the brand; not what the registry check decides.
- **Install name**: the PyPI/npm package name; the plan's own candidate is `atlas-dev` (plan §6). This is what task 0.1 checks and what the logged decision covers.
- **Reserved (GitHub)**: creating the private repo `HilarioNengareJr/atlas` is the reservation; repo names are per-account, so this holds regardless of the registry outcome.
- **Decision logged**: a dated entry in `docs/decisions.md` — plain markdown, since schema-shaped decision records don't exist until Milestone 1.
- **Committed**: `git init` in `~/atlas`, one initial commit, pushed to the new private remote.

### Decisions made
- **Name scope** (user-confirmed): project name stays Atlas; the registry check targets the install name with `atlas-dev` as the leading candidate; the GitHub repo is `HilarioNengareJr/atlas` either way. If `atlas-dev` is also taken on either registry, that is a hard conflict — stop and put the options to the user; do not pick a fallback name unilaterally.
- **Tagline** (deferred to build, user's call at the one-pager step): one of the two candidates from plan §1 — "Django for prompting" or "Prompting is the new programming. Atlas is what makes it engineering."
- **Registry checks are read-only**: `pip index` / registry JSON endpoints and `npm view` — no publishing, no placeholder packages.

### Assumptions
- Both plan docs live in `docs/`: `docs/atlas-plan.md` (the Jul 10 copy from `~/SKINS/cycle/`) and `docs/atlas-build-plan.md` (the pasted build plan, saved to disk as part of this work).
- Empty skeleton dirs (`spec/`, `skills/`, `adapters/`) get a `.gitkeep` so git tracks them.
- `context/` (this file plus the cycle's tracking artifacts) is committed — plans-as-artifacts fits Atlas's own philosophy.
- The Obsidian-vault session log stays the user's manual ritual; the repo's `docs/decisions.md` is the in-repo record.

### How to build it
1. **Name check (0.1 — hard-ordered first).** Query PyPI and npm for `atlas` (for the record — expected taken) and `atlas-dev` (the candidate). Present the facts; user decides the install name; write the dated decision to `docs/decisions.md`.
2. **Reserve the home (0.1).** `git init` in `~/atlas`; create the private repo `HilarioNengareJr/atlas` via `gh repo create`; wire the remote. No push yet — pushing is the ship stage.
3. **One-pager (0.2).** Write `docs/one-pager.md` from `atlas-plan.md` §1–2: thesis paragraph, the one sentence, the chosen tagline (user picks between the two candidates), the moat line, what Atlas is / refuses to be, design values. One page, no invention — lifted, trimmed, frozen.
4. **Skeleton (0.3).** Create `spec/`, `docs/`, `skills/`, `adapters/` (+ `.gitkeep` where empty); copy `docs/atlas-plan.md`; save `docs/atlas-build-plan.md`; write `README.md` stub containing the one sentence and the private-status note.
5. **Verification.** All four dirs exist; `docs/decisions.md` has the name decision; the one-pager contains exactly one tagline; README carries the one sentence; `git status` clean after the ship-stage commit.
