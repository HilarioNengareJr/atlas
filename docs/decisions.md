# Decisions

Plain-markdown decision log. Superseded by schema-shaped decision records (spec/records.schema.yaml) once Milestone 1 lands.

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
