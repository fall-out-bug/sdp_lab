# SDP Work Atomicity Normalization - Revision 3

**Date:** 2026-04-12
**Status:** Proposed
**Issue:** `sdplab-68v`
**Owner:** Andrei
**Supersedes:** [2026-04-12-work-atomicity-normalization-r2.md](2026-04-12-work-atomicity-normalization-r2.md)
**Input critique:** [2026-04-12-council-work-atomicity-normalization-r2.md](2026-04-12-council-work-atomicity-normalization-r2.md)

---

## Why Revision 3 Exists

Revision 2 fixed most of the modeling drift, but it still left runtime safety
underspecified.

Round 2 criticism converged on four concrete failures:

1. reshape is unsafe under concurrent live claims
2. mixed old/new workstream formats still create contradictory parser behavior
3. `primary` versus `finding` execution semantics are still ambiguous
4. aggregate completion is not yet fully derived and enforceable

Revision 3 addresses only those runtime and migration gaps.

It does **not** reopen the ontology debate.

---

## Scope And Non-Goals

### In scope

- runtime authority and dispatch inputs
- reshape safety under live Beads claims
- migration behavior for legacy, normalized, and invalid mixed features
- exact serialized execution rule for `primary` and `finding`
- derived status contract for aggregate workstreams
- dry-run and rollback rules before hard enforcement

### Out of scope

- changing `Feature -> Aggregate WS -> Leaf WS`
- renaming entities again
- changing Beads itself into a different tracker
- solving every historical backlog inconsistency in one pass

---

## Decision Summary

Revision 3 introduces one new machine artifact and one new migration rule.

### New machine artifact

`/Users/fall_out_bug/projects/vibe_coding/sdp_lab/.sdp/workgraph.lock.json`

This file is generated from canonical workstream documents and becomes the
**only runtime input** for dispatch, reshape guards, and status derivation.

### New migration rule

Normalization is decided **per feature**, not per individual workstream.

Each feature must be exactly one of:

- `legacy`
- `normalized`
- `mixed_invalid`

Only `normalized` features participate in the new runtime contract.

This removes the Phase 2/Phase 4 contradiction from Revision 2.

---

## Runtime Authority Model

Revision 3 separates authoring truth from runtime truth without creating dual
authority.

### Authoring truth

Human-edited source files remain:

- workstream frontmatter
- `## Beads` role block
- acceptance criteria text

These files are the only place humans edit workstream meaning.

### Runtime truth

`workgraph.lock.json` is a generated projection of authoring truth.

The dispatcher, validator, and reshape guard must never parse raw Markdown at
runtime. They must read the lock file only.

Hard rules:

1. if the lock file is missing, normalized features are not dispatchable
2. if the lock file does not match `HEAD`, normalized features are not
   dispatchable
3. `.beads-sdp-mapping.jsonl` remains derived helper output only
4. the dispatcher never derives parent/child structure by scanning docs live

This addresses both TOCTOU and performance objections from Round 2.

---

## Feature Normalization Classes

The compiler classifies each feature before writing the lock file.

### `legacy`

All active workstreams under the feature still use the pre-normalization
contract.

Rules:

1. excluded from `workgraph.lock.json`
2. not eligible for aggregate/leaf runtime enforcement
3. may continue through the legacy path until migrated

### `normalized`

All active workstreams under the feature validate the bounded hierarchy and
strict `## Beads` syntax from Revision 2.

Rules:

1. included in `workgraph.lock.json`
2. eligible for new dispatch and reshape rules
3. aggregate status is derived
4. leaf execution is serialized

### `mixed_invalid`

The feature contains partial normalization, such as:

- some workstreams with `ws_kind`, others without
- strict `## Beads` syntax on some files and legacy syntax on others
- aggregate references inside a feature that cannot compile into one valid tree

Rules:

1. excluded from `workgraph.lock.json`
2. dispatch blocked for that feature
3. validator error, not warning
4. migration must finish or be reverted back to fully `legacy`

This removes ambiguous mixed-mode runtime behavior.

---

## Lock File Contract

`workgraph.lock.json` is a compiled snapshot, not an editable planning file.

Minimum top-level fields:

```json
{
  "schema_version": 1,
  "git_commit": "abc123",
  "generated_at": "2026-04-12T15:00:00Z",
  "features": []
}
```

Each normalized feature entry must contain:

```json
{
  "feature_id": "F109",
  "mode": "normalized",
  "aggregate_ws_ids": ["00-109-00"],
  "leaf_ws_ids": ["00-109-01", "00-109-02"],
  "workstreams": []
}
```

Each workstream entry must contain resolved runtime fields:

```json
{
  "ws_id": "00-109-01",
  "ws_kind": "leaf",
  "parent_ws_id": "00-109-00",
  "children": [],
  "declared_status": "open",
  "derived_status": "open",
  "primary_issue_id": "sdplab-123",
  "finding_issue_ids": ["sdplab-456"],
  "historical_issue_ids": ["sdplab-101"],
  "active_issue_id": "sdplab-123",
  "execution_policy": "serialized"
}
```

For aggregate workstreams:

```json
{
  "ws_id": "00-109-00",
  "ws_kind": "aggregate",
  "parent_ws_id": null,
  "children": ["00-109-01", "00-109-02"],
  "declared_status": "open",
  "derived_status": "open",
  "aggregate_finding_issue_ids": ["sdplab-777"]
}
```

The compiler must precompute adjacency and derived status so runtime callers do
not scan the backlog.

---

## Serialized Execution Contract

Revision 3 makes leaf atomicity explicit.

### Rule

A leaf workstream has exactly **one execution slot** across `primary` and all
open `finding` issues.

That slot is represented by `active_issue_id` in the lock file.

Consequences:

1. `primary` and `finding` work for the same leaf must not run concurrently
2. if concurrency is required, the leaf was mis-shaped and must be reshaped
3. a finding does not create a second execution lane

### Active issue selection

For a normalized leaf, the compiler derives `active_issue_id` with this order:

1. highest-priority open blocking `finding`
2. current open `primary`
3. highest-priority open non-blocking `finding`
4. `null` if none are open

Tie-breaker for multiple findings at the same priority:

1. older `created_at`
2. lower lexical Beads id as a deterministic fallback

This resolves the Round 2 ambiguity directly.

---

## Aggregate Status Derivation

Aggregate workstreams are never directly executed, so their status must be
derived.

The compiler derives aggregate status with this order:

1. `archived` if the aggregate is explicitly archived
2. `done` if all child leaves are `done|archived` and there are no open
   aggregate-level findings
3. `blocked` if any open blocking aggregate-level finding exists
4. `blocked` if every non-terminal child is `blocked`
5. `open` if any child is `open|blocked|done` but the aggregate is not yet done
6. `backlog` otherwise

Validator rule:

- aggregate `declared_status` must match `derived_status`

This means frontmatter stays readable for humans, but the machine verdict is not
subjective.

---

## Dispatch Contract

Dispatch for normalized features reads `workgraph.lock.json` and live Beads
state.

The dispatcher must perform both checks:

### Pre-claim validation

Before claiming `active_issue_id`, confirm:

1. feature is present in the lock file as `normalized`
2. targeted issue equals current `active_issue_id`
3. no other issue on the same leaf is currently claimed
4. the owning workstream and all ancestors are not archived

### Post-claim revalidation

Immediately after claim, re-read:

1. current `HEAD`
2. current lock file
3. live Beads state for the same leaf

If either changes invalidate the claim, the dispatcher must abort execution and
release the claim before work begins.

This is the runtime answer to the TOCTOU objection. The system does not pretend
to be globally atomic; it enforces a safe revalidation barrier before execution.

---

## Reshape Protocol

A normalized feature may be reshaped only through a two-step cutover.

### Step 1: Freeze commit

The reshape PR first lands a freeze state for affected workstreams.

Required outcomes of the freeze commit:

1. affected leaf workstreams become non-dispatchable in the next lock file
2. their current `primary` issues move to `historical` or are explicitly marked
   non-ready in Beads
3. no new child leaves are active yet
4. the lock file records the frozen topology at a new `git_commit`

### Step 2: Replacement commit

Only after the freeze state is on the base branch may the replacement topology
land.

Required outcomes of the replacement commit:

1. new leaf workstreams exist and compile cleanly
2. each new open leaf has exactly one `primary`
3. old frozen workstreams are terminal or archived
4. the lock file advances to the replacement topology

### Hard safety rule

If any affected issue remains actively claimed during Step 1, reshape is blocked.

This is intentionally strict. If the team needs in-flight concurrent reshape,
the current atomicity model is already broken.

---

## Migration Protocol

Revision 3 replaces fuzzy warn-mode language with explicit feature states.

### Phase 1: Audit

Run validators and classify every feature as:

- `legacy`
- `normalized`
- `mixed_invalid`

Outcome:

- no runtime behavior changes

### Phase 2: Compile In Shadow Mode

Generate `workgraph.lock.json` for `normalized` features only.

Outcome:

- dispatcher may read the lock file for diagnostics
- runtime does not yet block legacy features

### Phase 3: Enforce On Normalized Features

Normalized features must dispatch only through the lock file path.

Outcome:

- legacy features continue on the old path
- mixed-invalid features are blocked

### Phase 4: Reshape Guard

Enable freeze-before-replace enforcement for normalized features.

Outcome:

- new normalized reshapes must follow the two-step cutover

### Phase 5: Global Cutover

Only after the active backlog no longer contains `legacy` features may the
legacy runtime path be removed.

This answers the Round 2 contradiction directly: runtime guard applies to
normalized features only, not to partially migrated backlog.

---

## Rollback And Dry-Run

Round 2 was correct to demand an exit ramp.

### Dry-run

Before enabling Phase 3 or Phase 4, the validator must support:

- compile lock file without dispatch enforcement
- report feature classification
- report leaf slot conflicts
- report aggregate status mismatches

### Rollback

If a normalized feature fails under the new runtime, rollback means:

1. revert the normalization commit set for that feature
2. regenerate the lock file
3. return the feature to a fully `legacy` or fully `normalized` state

Partial rollback into `mixed_invalid` is forbidden.

---

## Guard Rails

Revision 3 adds these non-negotiable guard rails:

1. `mixed_invalid` features are never dispatchable
2. normalized dispatch never reads raw Markdown
3. a leaf never has more than one active execution issue
4. reshape never lands in one commit
5. aggregate completion is derived, not hand-waved

If any of these guard rails feel too strict, the likely problem is not the
guard rail. The likely problem is that the workstream shape is still too large
or too ambiguous.

---

## Adoption Decision

Revision 3 makes the contract narrow enough to implement.

If the council accepts this revision, the next implementation slice should be:

1. compiler for `workgraph.lock.json`
2. validators for feature classification and slot conflicts
3. dispatcher revalidation barrier
4. freeze-before-replace reshape checks

The next step should **not** be another terminology rewrite.
