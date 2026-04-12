# SDP Work Atomicity Normalization — Revision 2

**Date:** 2026-04-12
**Status:** Proposed
**Issue:** `sdplab-0c5`
**Owner:** Андрей
**Supersedes:** [2026-04-12-work-atomicity-normalization.md](2026-04-12-work-atomicity-normalization.md)
**Input critique:** [2026-04-12-council-work-atomicity-normalization.md](2026-04-12-council-work-atomicity-normalization.md)

---

## Why Revision 2 Exists

Revision 1 got the diagnosis right and the contract wrong.

The valid council round accepted the problem statement but rejected the memo as
an adoption-ready spec.

The repeated blockers were concrete:

1. no machine-readable tree schema
2. no exact issue-role model
3. no migration protocol
4. no governance for live reshaping
5. findings triage still depended on judgment instead of a decision rule
6. `override with rationale` was a governance hole

Revision 2 addresses those gaps directly.

---

## Decision Summary

SDP should normalize work atomicity with a **bounded workstream hierarchy**, not
with loose “recursive workstream” language.

Target model:

- `Feature` owns outcome and UAT
- `Workstream` remains the SDP planning and contract unit
- `Beads issue` remains the live execution unit
- only `leaf` workstreams are executable
- `aggregate` workstreams are non-executable decomposition containers

This revision also changes two important points from Revision 1:

1. `parent` is renamed to `aggregate`
   Reason: it better describes the role and reduces “shadow feature” confusion.
2. `override with rationale` is removed
   Reason: the council correctly treated it as a process-theater loophole.

---

## What Must Stay True

Any acceptable model must keep these invariants:

1. `Feature` stays the user-facing outcome container.
2. `Beads` stays the live queue and dependency graph.
3. `workstream` stays first-class and does not collapse into a disposable note.
4. review, CI, drift, and `QA/UAT` findings stay inside the same overall system.
5. the model must be shallow enough to reason about without inventing ontology
   on every session.

---

## Normalized Hierarchy

This is **not** arbitrary recursion.

It is a bounded hierarchy with one optional aggregation layer.

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

- max workstream nesting depth is `1`

This avoids “epic hell” and keeps the planner model tractable.

---

## Entity Ownership

| Entity | Owns | Must not own |
|---|---|---|
| `Feature` | user-visible outcome, acceptance intent, UAT path | execution queue |
| `Aggregate WS` | decomposition of one feature slice into 2+ leaf workstreams | direct execution, independent UAT |
| `Leaf WS` | one executable contract slice | planned child workstreams |
| `Beads issue` | live execution state, findings state, dependencies, claim state | feature meaning |

Mechanical discriminator:

- if something has independent user-visible outcome or UAT meaning, it is a
  `Feature`, not an `Aggregate WS`
- if something exists only to break one feature slice into 2+ executable leaves,
  it is an `Aggregate WS`

---

## Exact Workstream Schema

Revision 2 makes frontmatter explicit.

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
---
```

### Field rules

#### `ws_kind`

Allowed values:

- `leaf`
- `aggregate`

#### `parent_ws_id`

Authoritative direction of linkage is **child -> parent**.

Rules:

1. `aggregate` workstream must have `parent_ws_id: null`
2. `leaf` workstream may have `parent_ws_id: null`
3. `leaf.parent_ws_id` may reference only an `aggregate` workstream
4. a referenced aggregate must share the same `feature_id`
5. a leaf may have at most one parent

No `child_ws_ids` field exists in frontmatter.

Children are derived by scanning all workstreams with matching `parent_ws_id`.

This removes dual-authority drift.

### Aggregate WS rules

An `aggregate` workstream:

1. is never dispatchable
2. must resolve to `2+` child leaf workstreams before it can leave `backlog`
3. must not have `Scope Files` that pretend to be executable implementation scope
4. must define roll-up acceptance only

### Leaf WS rules

A `leaf` workstream:

1. is the only executable workstream kind
2. must not have child workstreams
3. must define one executable acceptance contract
4. may exist directly under a feature or under one aggregate workstream

---

## Exact `## Beads` Contract

The canonical live issue linkage stays in the workstream file, but the syntax is
now strict and machine-readable.

Required format:

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

### Role meaning

#### `primary`

The current execution baton for a leaf workstream.

Rules:

1. only `leaf` workstreams may have a `primary`
2. an open leaf must have exactly one `primary`
3. an `aggregate` workstream must have zero `primary` entries
4. `primary` is the only issue that may represent planned implementation of the
   leaf itself

#### `finding`

A review/CI/drift/QA-derived issue that still belongs to the same owning leaf.

Rules:

1. `finding` may exist only for a `leaf` or `aggregate` already in the backlog
2. `finding` never changes `ws_kind` by itself
3. `finding` may trigger reshape or new-leaf creation only through the decision
   table below

#### `historical`

Closed or superseded issue retained for traceability.

Rules:

1. never dispatchable
2. used for replaced primaries, superseded findings, and reshaped history

### Mapping helper rule

`.beads-sdp-mapping.jsonl` remains a **derived helper**, not the canonical source.

It contains only:

- `ws_id`
- current `primary` issue id

Validators must derive it from `## Beads`, not invent parallel truth.

---

## Dispatch And Lock Rules

Revision 2 separates three ideas:

1. canonical implementation baton = `primary`
2. subordinate remediation = `finding`
3. actual live claim lock = Beads owner/claim state

### Dispatchability

Allowed direct execution targets:

- a `primary` issue on a `leaf`
- a `finding` issue listed on that same `leaf`

Forbidden direct execution targets:

- any issue attached to an `aggregate`
- any issue not listed in the owning workstream's `## Beads` section
- any `historical` issue

### Leaf execution lock

At runtime, a dispatcher must treat a leaf workstream as a single execution
lock domain.

Before executing a `primary` or `finding`, the dispatcher must:

1. resolve the owning `leaf` workstream
2. re-read that workstream file
3. confirm the issue id and role are still listed
4. confirm `ws_kind == leaf`
5. confirm there is no other already-claimed open `primary|finding` issue for
   the same leaf
6. then claim the issue in Beads and proceed

If any of these checks fail, dispatch aborts.

This is an explicit runtime guard, not a doc-only promise.

### What Revision 2 does not claim

Revision 2 does **not** claim perfect transactional single-writer semantics from
docs validators alone.

It requires:

- validator checks before merge
- runtime re-check before dispatch
- Beads claim state as the live lock signal

If later SDP needs stricter transactional guarantees, that is a dispatcher/Beads
capability upgrade, not a reason to keep the ontology vague.

---

## Aggregate Completion

`Aggregate WS` completion is strict roll-up.

It is complete only when:

1. all child leaf workstreams are `done`, and
2. there are no open blocking findings on those children

There is no free-form manual override in this revision.

If manual exceptional closure is ever needed in the future, it must be designed
as a separate audited protocol, not hidden in one sentence.

---

## Findings Decision Table

Revision 2 replaces vague heuristics with a concrete decision table.

### Decision Question 1

Does resolving this finding require editing the current leaf workstream's
`Goal` or `Acceptance Criteria` sections?

- **Yes** → create a **new leaf workstream**
- **No** → continue to Question 2

Reason:

- if the contract text must change, this is not the same executable slice

### Decision Question 2

Can the current leaf be completed with at most **one additional executable
issue** beyond the already-open work on that leaf?

- **Yes** → keep it as a **finding** under the same leaf
- **No** → continue to Question 3

Reason:

- this is the concrete stop-line against “zombie leaf” endless remediation

### Decision Question 3

Is the additional work still in service of the same feature slice, but now
requires decomposition into `2+` executable leaves?

- **Yes** → **reshape** current leaf into an aggregate workstream
- **No** → create a **new sibling leaf workstream**

### Examples

| Situation | Decision |
|---|---|
| CI failure on existing code path, no WS text changes, one fix ticket | same leaf, `finding` |
| Review says AC is incomplete and a new acceptance checkbox is needed | new leaf |
| Bug reveals three independently reviewable remediation slices under same feature slice | reshape to aggregate |
| New adjacent problem discovered that can ship separately | new sibling leaf |

---

## Live Reshaping Governance

Revision 2 removes silent or automatic reshaping.

### Who may propose reshape

Anyone may propose reshape:

- executor
- reviewer
- CI/drift/QA agent
- human operator

But proposal happens only by creating a Beads issue or explicit planning note.

### Who may mutate topology

Only planning authority may mutate workstream topology:

- user / decision owner
- designated feature owner
- explicit planning command path such as `@feature` / `@design`

Execution agents do **not** silently rewrite `ws_kind` or `parent_ws_id`.

### Reshape protocol

A live leaf may be reshaped only through this sequence:

1. raise or confirm a reshape finding
2. stop new dispatch against the current leaf
3. ensure no `primary|finding` issue for that leaf is currently claimed
4. move current `primary` issue to `historical`
5. change current workstream `ws_kind: leaf -> aggregate`
6. create `2+` child leaf workstreams with `parent_ws_id` set
7. create new `primary` issues for those child leaves
8. export Beads and rerun protocol checks

This means:

- no in-flight leaf-to-aggregate mutation while execution is active
- no orphaned active primary issue
- no stale executor running against an invalidated contract

---

## Migration Protocol

Revision 2 adds a phased rollout instead of big-bang replacement.

### Phase 1: Docs normalization

Ship:

- terminology change
- bounded hierarchy rule
- decision table for findings

Do not enforce yet.

### Phase 2: Schema introduction in warn mode

Ship:

- `ws_kind`
- `parent_ws_id`
- strict `## Beads` role syntax

Behavior:

- missing `ws_kind` is treated as implicit `leaf`
- validators warn, not fail

### Phase 3: Active backlog backfill

Backfill only:

- active features
- active workstreams
- any workstream touched in new PRs

Historical untouched backlog stays in compatibility mode temporarily.

### Phase 4: Runtime guard

Ship dispatcher changes:

- block dispatch of `aggregate`
- require issue role to be listed in the owning `## Beads` section
- enforce leaf lock re-check before execution

### Phase 5: Hard enforcement for new and modified workstreams

Validators fail if a new or modified workstream:

- omits `ws_kind`
- has illegal `parent_ws_id`
- has illegal `## Beads` role syntax
- assigns `primary` to an aggregate
- assigns multiple `primary` issues to a leaf

### Phase 6: Historical cleanup

Optional cleanup sprint:

- backfill the rest of the old backlog
- remove remaining compatibility warnings

---

## Validator Rules

Revision 2 expects these checks:

1. `aggregate` must have `parent_ws_id: null`
2. `leaf.parent_ws_id` may reference only an `aggregate`
3. no workstream depth > 1
4. `aggregate` cannot have a `primary` issue
5. open `leaf` must have exactly one `primary`
6. no duplicate issue id across roles within one workstream
7. `.beads-sdp-mapping.jsonl` must match current `primary` entries for leaves
8. modified `aggregate` must resolve to at least two child leaves before leaving
   `backlog`

---

## What This Revision Intentionally Defers

Not part of this memo:

- changing Beads' underlying claim semantics
- new database-backed execution map
- arbitrary-depth workstream trees
- auto-reshape without planning authority

These are separate capabilities, not prerequisites for atomicity normalization.

---

## Council Questions For Revision 2

1. Is bounded hierarchy a better normalization than the previous loose recursive
   framing?
2. Is the `## Beads` role contract strong enough, or should issue roles move
   into frontmatter or another canonical artifact?
3. Is the findings decision table now concrete enough for consistent agent
   behavior?
4. Is the reshape protocol sufficiently strict to avoid live-state corruption?
5. Is the phased migration realistic, or still too heavy for the repo's current
   state?
