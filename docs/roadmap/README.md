# docs/roadmap/

## Canonical

- **[ROADMAP.md](ROADMAP.md)** — active roadmap. Single source of truth for current product direction. Start here.

## Supporting Context (read on demand)

These files document past pivots, audits, or parallel streams. They are **not** the current plan:

| File | Purpose | Status |
|---|---|---|
| [AGENT_PLATFORM_ROADMAP_2026-03-31.md](AGENT_PLATFORM_ROADMAP_2026-03-31.md) | Platform-first reset plan (origin of current direction) | Historical context |
| [UNIFIED_VISION_ROADMAP_2026-03-03.md](UNIFIED_VISION_ROADMAP_2026-03-03.md) | Consolidated execution model across roadmap/index/beads | Historical context |
| [CRITICAL_ROADMAP_REVIEW_2026-03-03.md](CRITICAL_ROADMAP_REVIEW_2026-03-03.md) | Review that triggered the 2026-03-31 reset | Historical context |
| [IMPLEMENTATION_DRIFT_AUDIT_2026-03-03.md](IMPLEMENTATION_DRIFT_AUDIT_2026-03-03.md) | Drift audit feeding into reset | Historical context |
| [STATE_ALIGNMENT_STREAM_ASTAR.md](STATE_ALIGNMENT_STREAM_ASTAR.md) | A* stabilization stream | Parallel stream |
| [MARKET_INTELLIGENCE_OPERATING_LOOP.md](MARKET_INTELLIGENCE_OPERATING_LOOP.md) | Recurring market-driven prioritization process | Operating doc |
| [CONSISTENCY_MITIGATION_POLICY.md](CONSISTENCY_MITIGATION_POLICY.md) | Mandatory drift-check gate before feature planning | Policy |

## Rules

- Only `ROADMAP.md` carries the **CANONICAL** status marker. When the current roadmap is superseded, move the old file to `docs/archive/roadmap/` and update `ROADMAP.md` with the new canonical plan.
- Do not add `_2026-XX-XX.md`-dated roadmap variants alongside `ROADMAP.md`. Capture snapshots by commit, not by file copy.
- For phase-level backlog and workstreams, use `docs/workstreams/INDEX.md` (planning summary) + `bd ready` (live queue).
