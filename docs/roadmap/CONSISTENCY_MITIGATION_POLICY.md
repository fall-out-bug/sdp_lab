# Repository Consistency Mitigation Policy

Status: active
Updated: 2026-03-03

## 1. Problem statement

The main operational risk is management drift between:

- `docs/roadmap/ROADMAP.md`
- `docs/workstreams/INDEX.md`
- `docs/workstreams/backlog/*.md` frontmatter
- `.beads-sdp-mapping.jsonl`

If these artifacts diverge, release planning and queue governance become untrustworthy.

## 2. Enforcement model

Primary gate (always available):

- `python3 scripts/check_repo_consistency.py --json`

Convenience wrapper:

- `./scripts/run_consistency_checks.sh`

CI gate:

- `consistency-gate` in `.github/workflows/ci.yml`

## 3. Source-of-truth rules

1. `ROADMAP.md` is feature-phase authority.
2. `INDEX.md` is workstream-status authority.
3. Backlog `status` must match INDEX status for the same WS ID.
4. ROADMAP cannot reference WS IDs that do not exist in backlog.
5. Mapping file line count must equal backlog file count.
6. Workstream marked `done` must have all acceptance checkboxes completed.

## 4. Current baseline (after reconciliation)

- Status mismatch errors: 0
- Phantom roadmap WS references: 0
- Mapping count mismatch: 0
- Remaining warnings: done workstreams with unchecked acceptance criteria

Warnings currently tracked:

- `docs/workstreams/backlog/00-026-01.md`
- `docs/workstreams/backlog/00-059-01.md`
- `docs/workstreams/backlog/00-059-02.md`
- `docs/workstreams/backlog/00-061-01.md`
- `docs/workstreams/backlog/00-061-02.md`
- `docs/workstreams/backlog/00-067-01.md`

## 5. Mitigation plan

Phase A (immediate):

- Keep warnings non-blocking in CI to avoid false hard-fail while data is cleaned.

Phase B (next):

- Clean warning set by either:
  - completing checkbox evidence for done workstreams, or
  - reverting status to non-done where evidence is not available.

Phase C (hardening):

- Enable strict mode (`--strict-ac`) in CI after warning set reaches zero.

Mapping format note:

- `.beads-sdp-mapping.jsonl` legacy line-format differences are treated as non-blocking informational debt while semantic mapping integrity (count and ID linkage) remains enforced.

## 6. Operational procedure

Before planning new features:

1. Run `./scripts/run_consistency_checks.sh`.
2. If errors > 0, stop planning and fix consistency first.
3. If warnings > 0, track them explicitly in cleanup queue.
