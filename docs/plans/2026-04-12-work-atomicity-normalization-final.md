# SDP Work Atomicity Normalization - Final Spec

**Date:** 2026-04-12
**Status:** Proposed Final
**Issue:** `sdplab-ybl6`
**Owner:** Andrei
**Supersedes:** [2026-04-12-work-atomicity-normalization-r2.md](2026-04-12-work-atomicity-normalization-r2.md), [2026-04-12-work-atomicity-normalization-r3.md](2026-04-12-work-atomicity-normalization-r3.md), [2026-04-12-work-atomicity-normalization-r3a-live-state-boundary.md](2026-04-12-work-atomicity-normalization-r3a-live-state-boundary.md)
**Input critique:** [2026-04-12-council-work-atomicity-normalization-r3a-live-state-boundary.md](2026-04-12-council-work-atomicity-normalization-r3a-live-state-boundary.md)

---

## Purpose

This spec normalizes SDP work atomicity so three things stay true at once:

1. workstream structure is machine-readable
2. runtime dispatch is safe enough to execute without reading raw Markdown
3. live Beads state is not falsely snapshotted into static git state

This document is the implementation target.

---

## Decision Summary

SDP adopts the following model.

### Work hierarchy

Allowed shapes:

```text
Feature -> Leaf WS
Feature -> Aggregate WS -> Leaf WS
```

Forbidden shapes:

```text
Feature -> Aggregate WS -> Aggregate WS
Feature -> Leaf WS -> Leaf WS
Feature -> Leaf WS -> Aggregate WS
```

Hard rule:

- maximum workstream nesting depth is `1`

### Execution boundary

- `workgraph.lock.json` is a committed topology-and-bindings artifact
- live issue selection is runtime-derived from Beads state
- `active_issue_id` and any other mutable Beads-derived field are forbidden in
  the committed lock file

### Feature normalization

Each feature is exactly one of:

- `legacy`
- `normalized`
- `mixed_invalid`

Only `normalized` features enter the new runtime contract.

---

## Entity Model

### Feature

Owns:

- user-visible outcome
- acceptance intent
- UAT meaning

Does not own:

- execution slot

### Aggregate workstream

Owns:

- decomposition of one feature slice into `2+` leaf workstreams
- roll-up acceptance only

Does not own:

- direct execution
- independent UAT meaning
- child aggregates

### Leaf workstream

Owns:

- one executable contract slice
- one leaf-scoped execution slot

Does not own:

- child workstreams

### Beads issue

Owns:

- live execution state
- claim state
- findings state
- dependencies

Does not own:

- feature meaning
- topology

---

## Workstream Schema

Canonical frontmatter:

```yaml
---
ws_id: 00-FFF-SS
feature_id: FXXX
status: backlog|open|blocked|done|archived
priority: P0|P1|P2|P3
size: XS|S|M|L
depends_on: []
ws_kind: leaf|aggregate
parent_ws_id: null|00-FFF-SS
dispatch_lifecycle: active|frozen|archived
---
```

Rules:

1. `aggregate` must have `parent_ws_id: null`
2. `leaf` may have `parent_ws_id: null`
3. `leaf.parent_ws_id` may reference only an `aggregate`
4. parent and child must share the same `feature_id`
5. a leaf may have at most one parent
6. only `leaf` workstreams are executable
7. `dispatch_lifecycle = archived` iff `status = archived`
8. `dispatch_lifecycle = frozen` is allowed only for `normalized` features

Authoritative linkage is child-to-parent only. No `child_ws_ids` field exists in
frontmatter.

---

## `## Beads` Contract

Canonical format:

```md
## Beads

- primary: sdplab-123
- finding: sdplab-456
- historical: sdplab-789
```

Allowed roles:

- `primary`
- `finding`
- `historical`

Rules:

1. `leaf` may have exactly one open `primary`
2. `aggregate` must have zero `primary`
3. `leaf` and `aggregate` may have `finding`
4. `historical` is trace only and never dispatchable

Role transition:

- an issue may move to `historical` only after it is closed or explicitly
  superseded in source
- moving an issue to `historical` requires lock regeneration

`.beads-sdp-mapping.jsonl` remains derived helper output only.

---

## Feature Normalization Classes

### `legacy`

- excluded from `workgraph.lock.json`
- continues on the legacy dispatch path

### `normalized`

- validates this spec
- emits into `workgraph.lock.json`
- dispatches only through the new runtime contract

### `mixed_invalid`

Examples:

- partial `ws_kind` adoption inside one feature
- mixed legacy and strict `## Beads` syntax inside one feature
- malformed aggregate/leaf tree

Rules:

- excluded from `workgraph.lock.json`
- non-dispatchable
- validator error, not warning

Machine-checkable conditions:

1. any active workstream in the feature lacks `ws_kind`
2. any active workstream in the feature lacks `dispatch_lifecycle`
3. strict `## Beads` parsing fails for any active workstream
4. a `leaf` references a non-aggregate parent
5. a workstream references a parent in another feature
6. aggregate depth exceeds `1`
7. an `aggregate` declares a `primary`
8. an open `leaf` has `0` or `>1` bound primaries
9. topology contains orphan or cyclic parentage

---

## Lock File Contract

Path:

`/Users/fall_out_bug/projects/vibe_coding/sdp_lab/.sdp/workgraph.lock.json`

The committed lock file contains only topology, bindings, lifecycle state, and
policy versions.

`lifecycle_state` in the lock file is copied from source `dispatch_lifecycle`.

Minimum schema:

```json
{
  "schema_version": 1,
  "source_inputs_hash": "sha256:...",
  "policy_versions": {
    "normalization": "v1",
    "dispatch_resolution": "v1",
    "aggregate_status": "shadow-v1|enforced-v1"
  },
  "features": []
}
```

Normalized feature entry:

```json
{
  "feature_id": "F109",
  "mode": "normalized",
  "workstreams": []
}
```

Leaf entry:

```json
{
  "ws_id": "00-109-01",
  "ws_kind": "leaf",
  "parent_ws_id": "00-109-00",
  "children": [],
  "lifecycle_state": "active",
  "declared_status": "open",
  "bound_primary_issue_id": "sdplab-123",
  "finding_issue_ids": ["sdplab-456"],
  "historical_issue_ids": ["sdplab-101"]
}
```

Aggregate entry:

```json
{
  "ws_id": "00-109-00",
  "ws_kind": "aggregate",
  "parent_ws_id": null,
  "children": ["00-109-01", "00-109-02"],
  "lifecycle_state": "active",
  "declared_status": "open",
  "derived_status": "open",
  "aggregate_finding_issue_ids": ["sdplab-777"],
  "historical_issue_ids": ["sdplab-706"]
}
```

The committed lock file must not contain:

- `active_issue_id`
- any ranking result over live issues
- any field derived from current Beads claim state
- any field derived from current Beads openness
- any field derived from current Beads priority changes

`bound_primary_issue_id` is the issue id declared as `primary` in the
workstream's `## Beads` section at compile time. It is a static binding, not a
live-state field.

---

## `source_inputs_hash` Contract

The lock file is fresh only when its `source_inputs_hash` matches the current
authoritative source inputs.

### Included inputs

For each `normalized` feature, the compiler includes only:

1. workstream frontmatter fields:
   - `ws_id`
   - `feature_id`
   - `status`
   - `ws_kind`
   - `parent_ws_id`
   - `dispatch_lifecycle`
2. parsed `## Beads` role bindings:
   - `primary`
   - `finding`
   - `historical`
3. compiler policy versions:
   - `schema_version`
   - `normalization`
   - `dispatch_resolution`
   - `aggregate_status`

### Excluded inputs

The hash must explicitly exclude:

- Markdown prose outside `## Beads`
- acceptance criteria text
- comments and whitespace
- ordering of keys in source files
- `priority`
- `size`
- `depends_on`
- `.beads-sdp-mapping.jsonl`
- live Beads state
- generated artifacts
- timestamps
- git commit ids
- environment variables

### Canonicalization

The compiler must build one canonical JSON object before hashing:

```json
{
  "schema_version": 1,
  "policy_versions": { "...": "..." },
  "features": [
    {
      "feature_id": "F109",
      "workstreams": [
        {
          "ws_id": "00-109-01",
          "frontmatter": {
            "feature_id": "F109",
            "dispatch_lifecycle": "active",
            "parent_ws_id": "00-109-00",
            "status": "open",
            "ws_id": "00-109-01",
            "ws_kind": "leaf"
          },
          "beads": {
            "finding": ["sdplab-456"],
            "historical": ["sdplab-101"],
            "primary": ["sdplab-123"]
          }
        }
      ]
    }
  ]
}
```

Canonicalization rules:

1. sort features by `feature_id`
2. sort workstreams by `ws_id`
3. sort all issue id arrays lexically
4. sort all object keys lexically
5. use UTF-8 JSON with `\n` line endings
6. hash the canonical serialized bytes as `sha256`

Dispatch is allowed only when:

1. normalized source files are clean in the working tree
2. recomputed `source_inputs_hash` exactly matches the lock file

Malformed input rule:

- if any normalized feature fails frontmatter validation or strict `## Beads`
  parsing, compilation fails for that feature and it is classified as
  `mixed_invalid`

---

## Runtime Beads Query Contract

Runtime may query only issue ids already bound by the lock file.

For each bound issue id, the Beads adapter must return this normalized shape:

```json
{
  "id": "sdplab-123",
  "is_open": true,
  "is_claimed": false,
  "priority": 1,
  "created_at": "2026-04-12T15:22:25Z",
  "status": "open"
}
```

Required semantics:

1. `priority` is integer `0..4`
2. `created_at` is RFC3339 UTC
3. `is_open` is true for non-terminal issues only
4. `is_claimed` reflects current live claim ownership

### Binding visibility rule

Only bound issue ids may affect normalized dispatch.

Consequences:

1. unbound Beads issues are ignored by active issue resolution
2. if the adapter can detect unbound issues that reference the same workstream,
   it should emit `WARN unbound_issue_ignored`
3. a newly created finding becomes dispatchable only after:
   - it is added to `## Beads`
   - the lock file is regenerated

### Blocking policy

For `dispatch_resolution = v1`:

- `priority 0..1` => blocking finding
- `priority 2..4` => non-blocking finding

This is an explicit v1 heuristic, not timeless truth. A future policy version
may replace it with native blocker metadata once Beads supports it.

### Query failure rule

If the adapter cannot produce a complete response for all bound issues on the
target leaf, dispatch fails closed.

### Adapter error contract

On query failure, the adapter must return:

```json
{
  "code": "beads_query_failed",
  "leaf_ws_id": "00-109-01",
  "issue_ids": ["sdplab-123", "sdplab-456"],
  "reason": "timeout|not_found|transport|invalid_payload"
}
```

---

## Active Issue Resolution

Runtime derives the active issue at dispatch time, not at compile time.

For a leaf workstream:

1. highest-priority open blocking `finding`
2. current open `primary`
3. highest-priority open non-blocking `finding`
4. none

Tie-breakers:

1. older `created_at`
2. lower lexical issue id

If no bound issue is active, the leaf is not dispatchable.

---

## Claim And Revalidation State Machine

### Pre-claim checks

The dispatcher must confirm:

1. feature mode is `normalized`
2. leaf `lifecycle_state = active`
3. lock freshness check passes
4. Beads query succeeds for the full bound issue set
5. resolved active issue exists
6. no other bound issue on the same leaf is claimed

### Claim primitive

The dispatcher claims the resolved active issue in Beads.

Today, Beads issue claim is the only atomic ownership primitive assumed by this
spec.

### Claim release primitive

The Beads adapter must expose a release operation for a previously acquired
claim.

### Post-claim revalidation

Immediately after claim, the dispatcher must re-read:

1. lock freshness
2. leaf lifecycle state
3. live Beads query for the full bound issue set
4. resolved active issue under current live data

Dispatch aborts and releases the claim if any of these are true:

- lock freshness fails
- leaf is no longer `active`
- Beads query fails
- resolved active issue changed
- another bound issue on the same leaf is now claimed

### Failure mode

Revalidation is fail-closed.

Required behavior:

1. release the claim if one was acquired
2. emit `dispatch_aborted_revalidation`
3. do not begin execution work

If claim release cannot be confirmed, the dispatcher must emit
`dispatch_claim_release_failed` and treat the claim as leaked until manually
resolved.

### Retry rule

Automatic retry is allowed only for `leaf_conflict` failures.

Rules:

1. at most `1` automatic retry
2. jittered backoff `250ms..750ms`
3. no automatic retry on lock mismatch
4. no automatic retry on Beads query failure

### Minimum observability

Required counters:

- `dispatch_attempt_total`
- `dispatch_success_total`
- `dispatch_aborted_revalidation_total`
- `dispatch_leaf_conflict_total`
- `dispatch_claim_release_failed_total`
- `dispatch_beads_query_failed_total`

Required log codes:

- `unbound_issue_ignored`
- `lock_freshness_failure`
- `leaf_conflict`
- `dispatch_aborted_revalidation`
- `dispatch_claim_release_failed`
- `beads_query_failed`

---

## Known Limitation: Leaf-Wide Exclusivity

Until Beads supports leaf-scoped exclusivity directly, leaf-wide exclusivity is
best-effort.

Possible failure:

- dispatcher A claims one bound issue on a leaf
- dispatcher B claims another bound issue on the same leaf
- both abort on post-claim revalidation

This is accepted as a known limitation for `dispatch_resolution = v1`.

Interim mitigation:

- a dispatcher SHOULD take a local advisory lock keyed by `ws_id` around
  pre-claim, claim, and post-claim revalidation
- this mitigation reduces same-host contention but is not authoritative across
  multiple hosts or runners

Required observability:

- increment `dispatch_leaf_conflict_total`
- log `WARN` with leaf id and competing issue ids

This is not grounds to reintroduce live-state fields into the committed lock.

---

## Aggregate Status Contract

Aggregate workstreams are non-executable and derived.

### Decision table

| Order | Condition | `derived_status` |
|---|---|---|
| 1 | declared aggregate is `archived` | `archived` |
| 2 | all child leaves are `done|archived` and no open aggregate findings exist | `done` |
| 3 | any open blocking aggregate finding exists | `blocked` |
| 4 | all non-terminal child leaves are `blocked` | `blocked` |
| 5 | any child leaf is `open|blocked|done` and row 2 did not match | `open` |
| 6 | all child leaves are `backlog` | `backlog` |

### Migration policy

For `aggregate_status = shadow-v1`:

- mismatch between `declared_status` and `derived_status` is warning only
- compiler still emits the feature
- lock file uses `derived_status` as machine truth

For `aggregate_status = enforced-v1`:

- mismatch is compile error
- the feature does not emit into the lock file
- the feature is non-dispatchable until fixed

This rollout rule applies only to `normalized` features.

Rollout trigger:

- newly normalized features SHOULD default to `enforced-v1`
- a repository may remain on `shadow-v1` only while shadow warnings still occur
  on existing normalized features
- the repository should flip to `enforced-v1` after one clean shadow cycle with
  no aggregate mismatch warnings in CI

---

## Reshape Contract

Normalized features reshape through freeze-before-replace.

### Step 1: Freeze

Required outcomes:

1. affected leaves become `lifecycle_state = frozen`
   and `dispatch_lifecycle: frozen` in source
2. freeze state is present in the committed lock file
3. no new replacement leaves are active yet

Freeze safety note:

- because `dispatch_lifecycle` is part of `source_inputs_hash`, any freeze
  change invalidates the prior lock file
- a dispatcher holding the old lock must fail the freshness check during
  post-claim revalidation and abort

### Step 2: Replace

Required outcomes:

1. replacement topology compiles cleanly
2. each new open leaf has exactly one `primary`
3. replaced leaves are terminal or archived

### Safety rule

If any affected bound issue is actively claimed, reshape must not proceed.

If a freeze lands after a dispatcher selected a leaf but before work begins, the
dispatcher must abort during post-claim revalidation because leaf lifecycle is no
longer `active`.

---

## Implementation Order

Implementation should proceed in this order:

1. lock compiler and canonical hash
2. live Beads adapter for the normalized query contract
3. dispatcher state machine with fail-closed revalidation
4. aggregate shadow enforcement
5. aggregate enforced mode

The next step is implementation, not another terminology rewrite.
