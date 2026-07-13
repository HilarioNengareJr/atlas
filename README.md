# Atlas

> **Django for prompting.**

Atlas is an opinionated, installable, agent-agnostic workflow framework that makes AI-assisted development compound — every task leaves the project smarter than it found it.

**Status:** private, pre-v0.1. The what/why lives in [`docs/atlas-plan.md`](docs/atlas-plan.md); the how/when in [`docs/atlas-build-plan.md`](docs/atlas-build-plan.md); the short version in [`docs/one-pager.md`](docs/one-pager.md).

Install name (reserved, nothing published yet): `atlasdev` — see [`docs/decisions.md`](docs/decisions.md).

## Layout

- `spec/` — versioned schemas (manifest, records, plan, Map, config)
- `docs/` — plans, one-pager, decisions, spikes
- `skills/` — Atlas skills (markdown-era first, per Milestone 3)
- `adapters/` — agent adapters (Claude Code first)
