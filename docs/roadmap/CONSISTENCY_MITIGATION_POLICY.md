# Repository Consistency Mitigation Policy

Status: active
Updated: 2026-04-04

## 1. Problem statement

The main operational risk is management drift between:

- `docs/roadmap/ROADMAP.md`
- `docs/workstreams/INDEX.md`
- `docs/workstreams/backlog/*.md` frontmatter
- workstream `## Beads` sections
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

1. `ROADMAP.md` is feature-phase planning authority.
2. `INDEX.md` is the planning summary for feature and workstream status.
3. Backlog file frontmatter and body are the canonical workstream record, including the live `## Beads` links.
4. Beads remains the live execution queue for ready/blocked/in-progress work.
5. Backlog `status` must match INDEX status for the same WS ID.
6. ROADMAP cannot reference WS IDs that do not exist in backlog.
7. `.beads-sdp-mapping.jsonl` is helper lookup data, not a required 1:1 mirror of every backlog file.
8. Mapping rows must use the normalized `sdp_id` / `beads_id` shape when present.
9. Workstream marked `done` must have all acceptance checkboxes completed.

## 4. Current baseline (after reconciliation)

- Status mismatch errors: 0
- Phantom roadmap WS references: 0
- Mapping schema drift: 0
- Current helper coverage: 151 mapping rows for 209 backlog files
- Remaining warnings: 0

Current check output (`python3 scripts/check_repo_consistency.py --json`):

- `roadmap_ws_refs = 47`
- `index_ws_status_rows = 151`
- `backlog_files = 209`
- `beads_mapping_count = 151`

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

- `.beads-sdp-mapping.jsonl` no longer uses count parity as a consistency rule.
- Partial historical coverage is acceptable as long as canonical workstream files keep the live Beads links and helper rows that do exist use the normalized schema.

## 6. Operational procedure

Before planning new features:

1. Run `./scripts/run_consistency_checks.sh`.
2. If errors > 0, stop planning and fix consistency first.
3. If warnings > 0, track them explicitly in cleanup queue.
