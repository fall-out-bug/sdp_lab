# SDP Work Atomicity Normalization - Revision 3a

**Date:** 2026-04-12
**Status:** Proposed
**Issue:** `sdplab-44k`
**Owner:** Andrei
**Focus:** live-state boundary for `workgraph.lock.json`
**Supersedes:** [2026-04-12-work-atomicity-normalization-r3.md](2026-04-12-work-atomicity-normalization-r3.md) on runtime state ownership only
**Input critique:** [2026-04-12-council-work-atomicity-normalization-r3.md](2026-04-12-council-work-atomicity-normalization-r3.md)

---

## Why This Patch Exists

Round 3 narrowed the remaining dispute to one hard question:

- does `workgraph.lock.json` own volatile execution selection such as
  `active_issue_id`
- or does it own only topology and issue bindings, while runtime selection stays
  live in Beads

This patch answers only that question and the two adjacent contracts it touches:

1. lock identity
2. claim/revalidation boundary

It does **not** reopen hierarchy, migration phases, or terminology.

---

## Problem Statement

Revision 3 compiled `active_issue_id` into the lock file.

That field is derived from mutable Beads state:

- which findings are open
- which findings are blocking
- Beads priority
- `created_at`

This creates a structural mismatch:

- the lock file is git-managed and intentionally stable between commits
- Beads issue state is live runtime state and can change between commits

If a new blocking finding appears after lock generation, a static
`active_issue_id` can go stale without any topology change.

That is the critic's strongest objection, and it is correct.

---

## Decision Summary

`workgraph.lock.json` must be a **topology-and-bindings artifact**, not a
snapshot of live queue state.

### Therefore

- `active_issue_id` is removed from the committed lock file
- any field derived from mutable Beads execution state is removed from the
  committed lock file
- runtime derives the current active issue from live Beads state, but only
  within the issue bindings permitted by the lock file

This preserves both halves of the contract:

1. git owns structure
2. Beads owns live execution state

---

## Rejected Options

### Option A: Keep `active_issue_id` in the lock file

Rejected.

Reason:

- it snapshots mutable queue state into static git state
- it requires git regeneration for routine live finding changes
- it recreates split-brain at a different layer

### Option B: Remove the lock file and derive everything at runtime

Rejected.

Reason:

- it brings runtime back to live Markdown parsing or equivalent dynamic graph
  assembly
- it reopens parser drift, performance, and topology ambiguity

### Option C: Static topology, live issue selection

Accepted.

Reason:

- topology is stable enough to compile
- execution selection is volatile enough to stay runtime-derived
- the authority split becomes clear instead of blended

---

## Static Versus Live Boundary

### Committed lock file may contain

- feature normalization class
- workstream topology
- `ws_kind`
- `parent_ws_id`
- child adjacency
- lifecycle state such as `active|frozen|archived`
- bound issue ids from `## Beads`
  - `primary_issue_id`
  - `finding_issue_ids`
  - `historical_issue_ids`
  - `aggregate_finding_issue_ids`
- policy versions
  - dispatch policy version
  - aggregate status policy version

### Committed lock file must not contain

- `active_issue_id`
- any ranking result over live issues
- any field derived from current Beads openness, blocking, claim, or priority
- any field whose value changes when Beads changes but git does not

This is the hard boundary.

---

## Runtime Resolution Contract

Dispatch becomes a two-source evaluation:

1. load compiled lock file
2. query live Beads state only for issue ids bound by that lock file

For a leaf workstream, runtime derives the active issue with this order:

1. highest-priority open blocking `finding`
2. current open `primary`
3. highest-priority open non-blocking `finding`
4. none

Tie-breakers:

1. older `created_at`
2. lower lexical issue id

This logic stays the same as Revision 3.

What changes is **where** it executes:

- not at compile time
- at dispatch time

---

## Claim And Revalidation Contract

### Pre-claim

The dispatcher must confirm:

1. the feature is `normalized`
2. the leaf is `active`, not `frozen|archived`
3. the chosen issue is currently active under the runtime resolution algorithm
4. no other bound issue on the same leaf is already claimed

### Claim primitive

Beads claim on the chosen issue remains the atomic ownership primitive available
today.

If Beads cannot enforce leaf-wide exclusivity directly, the dispatcher must
treat any existing claim on another bound issue of the same leaf as a hard
dispatch failure after live re-read.

### Post-claim revalidation

Immediately after claim, the dispatcher re-reads:

1. lock identity
2. live Beads state for the same bound issue set

If the chosen issue is no longer active, execution aborts and the claim is
released before work starts.

This does not create global atomicity. It creates an explicit safety barrier
without baking live state into git.

---

## Lock Identity Contract

Revision 3's `git_commit` wording was too loose.

The committed lock file should identify its **inputs**, not try to describe the
commit that contains itself.

Recommended field:

```json
{
  "source_inputs_hash": "sha256:..."
}
```

The hash is computed over the normalized feature's authoritative source inputs:

- workstream frontmatter
- `## Beads` role blocks
- any compiler config that affects graph or policy interpretation

Dispatch is allowed only when:

1. checkout is clean for normalized workstream sources
2. `source_inputs_hash` matches current source inputs

This removes the git-hash paradox and ties runtime to actual source state.

---

## Aggregate Status Consequence

This patch is not primarily about aggregate DX, but one nearby ambiguity must be
closed because it touches the static/live boundary.

Rule:

- `declared_status != derived_status` is a compile error for normalized features

Consequence:

- the feature does not emit into the lock file
- it is treated as non-dispatchable until fixed

This avoids a half-valid static artifact.

---

## Implementation Shape

Revision 3a implies one architecture change:

### Compiler output

The compiler emits:

- topology
- bindings
- policy versions
- source input hash

### Dispatcher input

The dispatcher combines:

- compiled lock file
- live Beads state for bound issues

### Optional observability artifact

If the system wants a user-facing snapshot such as "current active issue", it
should be emitted as an ephemeral runtime view or logs artifact, not committed
as source-controlled lock state.

---

## Adoption Recommendation

Revision 3 should be patched in this direction:

1. remove `active_issue_id` from the committed lock schema
2. move issue selection to runtime resolution against live Beads
3. replace `git_commit` freshness with `source_inputs_hash`
4. treat aggregate declared/derived mismatch as compile failure

If the council agrees with this boundary, the remaining work becomes normal
implementation detail, not architectural dispute.
