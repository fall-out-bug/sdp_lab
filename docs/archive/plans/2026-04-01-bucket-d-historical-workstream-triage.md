# Bucket D Historical Workstream Triage (2026-04-01)

> **Status:** Done
> **Goal:** Decide which oversized historical workstreams need structural cleanup and which ones are only long, not misleading.

Related:

- `docs/plans/2026-03-31-legacy-drift-cleanup-procedure.md`
- `docs/workstreams/backlog/00-096-02.md`
- `docs/reviews/2026-04-01-F053-workstream-review-history.md`

## Decision

Bucket D is smaller than it looked.

Most oversized files are long because they include one implementation note block and one review block.
That is not ideal, but it is still readable and does not mix planning with multiple audit rounds.

The first real split candidate was `00-053-01`.
It mixed:

- the workstream contract
- the execution report
- round-1 review
- round-2 cross-repo audit findings
- round-4 post-feature audit findings

That file now points to a dedicated review artifact instead of carrying the full audit transcript inline.

## Triage Table

| Workstream | Lines | Why it is long | Decision |
|------------|------:|----------------|----------|
| `00-027-01` | 112 | Large implementation-notes block plus one review section | Keep as-is |
| `00-026-01` | 111 | Provenance notes and one review section | Keep as-is |
| `00-024-01` | 96 | Hook payload examples and one review section | Keep as-is |
| `00-053-01` | 94 | Multiple review rounds and cross-feature audit history mixed into active file | Split audit history out |
| `00-025-01` | 90 | Prompt consolidation notes and one review section | Keep as-is |
| `00-061-04` | 86 | Reference-heavy implementation notes | Keep as-is |
| `00-022-01` | 85 | Example-heavy implementation notes and one review section | Keep as-is |
| `00-001-01` | 82 | Archived schema work with one execution and one review block | Keep as-is |
| `00-001-02` | 80 | Archived CLI work with one execution and one review block | Keep as-is |

## Why only one split now

Doing more would be cleanup theater.

The procedure for Bucket D says to shorten active workstream contracts when they do too many jobs at once.
Only `00-053-01` clearly crossed that line.
The rest are verbose, but they still behave like single-purpose records.

## Action Taken

- created `00-096-02` to record the Bucket D triage batch
- extracted `00-053-01` multi-round audit history into `docs/reviews/2026-04-01-F053-workstream-review-history.md`
- updated dependent F053 workstreams to point at the extracted review artifact

## Residual Risk

The repository still has a broader rule mismatch around `.beads-sdp-mapping.jsonl` coverage versus the current backlog file count.
That is a separate hygiene problem.
It should be handled as its own cleanup task, not folded into Bucket D.
