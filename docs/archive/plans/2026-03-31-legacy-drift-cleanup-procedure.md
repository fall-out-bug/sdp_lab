# Legacy Drift Cleanup Procedure (2026-03-31)

> **Status:** Active
> **Goal:** Reduce historical protocol and backlog drift without derailing the platform-first execution lane.

Related:

- `docs/plans/2026-03-31-platform-backlog-reset.md`
- `docs/workstreams/INDEX.md`
- `docs/roadmap/ROADMAP.md`

## Rule

Legacy drift cleanup is a support lane.
It should unblock planning trust.
It must not become the default execution path again.

## Drift buckets

Current protocol-check findings fall into four buckets.

### Bucket A: Placeholder workstreams with invalid frontmatter

Examples:

- `00-069-06` ... `00-069-15`

Symptoms:

- missing `size`
- missing `depends_on`
- missing `## Acceptance Criteria`
- missing `## Beads`

Treatment:

- either materialize them into valid workstreams
- or archive/remove them if they are fake placeholders

### Bucket B: Feature/index mismatch

Examples:

- `F053`, `F054`, `F055`, `F056`

Symptoms:

- backlog files exist
- `ROADMAP.md` or `INDEX.md` says the features are reserved, absent, or intentionally not materialized

Treatment:

- choose one truth:
  - re-materialize those features in roadmap and index
  - or archive/move the files out of the active backlog

No half-state allowed.

### Bucket C: Missing or invalid `## Beads` sections

Examples:

- many `00-053-*` files
- several `00-054-*`, `00-055-*`, `00-056-*` files

Symptoms:

- missing `## Beads`
- empty Beads section
- non-`sdplab-*` references

Treatment:

- add valid `sdplab-*` references where work remains active
- mark the work historical and move it out of active backlog if the underlying beads no longer exist

### Bucket D: Historical review logs embedded in active workstreams

Examples:

- long `F053` workstreams with execution history and multiple audit rounds

Symptoms:

- active backlog file is doing too many jobs at once
- planning, execution log, and audit transcript are mixed together

Treatment:

- keep the workstream contract short
- move heavy audit history into review artifacts if still needed

## Procedure

Run cleanup in this order.

1. Bucket A first.
   Reason: these are objective schema failures and the cheapest wins.
2. Bucket B second.
   Reason: roadmap/index authority must stop contradicting backlog reality.
3. Bucket C third.
   Reason: once feature authority is clean, beads references can be repaired systematically.
4. Bucket D last.
   Reason: structure cleanup is lower leverage than hard protocol mismatches.

## Batch rules

- One bucket family per workstream batch.
- Cap each cleanup batch to roughly 5-10 files.
- Re-run `go run ./cmd/sdp-protocol-check --format json` after every batch.
- Do not mix platform implementation code with legacy cleanup in one commit.
- If a bucket requires product-direction judgment, stop and record the decision before bulk edits.

## Verification checkpoints

For each cleanup batch:

- protocol check count for the target bucket decreases
- no new warnings are introduced in unrelated buckets
- `INDEX.md` and `ROADMAP.md` remain aligned for touched features
- touched workstreams include valid frontmatter and `## Beads`

## First cleanup batch

Start with Bucket A:

- `00-069-06` ... `00-069-10`

Reason:

- these are mechanically broken placeholder files
- they are easy to normalize or archive
- they reduce hard protocol-check errors immediately

## Exit criteria

- no active backlog file fails required frontmatter
- no active backlog feature is absent from `INDEX.md` and `ROADMAP.md`
- every active workstream has a valid `## Beads` section
- protocol-check failures are reduced to intentional, documented residuals
