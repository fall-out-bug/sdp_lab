# LLM Council Report: SDP Work Atomicity Normalization - Final Spec

**Date:** 2026-04-12
**Rounds:** 1 of 5
**Subject:** [2026-04-12-work-atomicity-normalization-final.md](2026-04-12-work-atomicity-normalization-final.md)
**Consensus:** ACCEPT WITH PATCHES
**Decision Owner:** PENDING

---

## Models

| Role | Model | Runtime |
|---|---|---|
| Architect | `anthropic/claude-sonnet-4.6` | OpenRouter |
| Critic | `mistralai/mistral-large-2512` | OpenRouter |
| Technician | `deepseek/deepseek-v3.2` | OpenRouter |
| Philosopher | `anthropic/claude-opus-4.6` | OpenRouter |
| Pragmatist | `minimax/minimax-m2.7` | OpenRouter |
| Engineer | `xiaomi/mimo-v2-pro` | OpenRouter |

All 6 roles responded in the final valid amended pass.

Validity note:

- user requested not to use `openai/gpt-5.4` on OpenRouter; the architect role
  therefore used `anthropic/claude-sonnet-4.6`
- this report covers only the amended final spec pass
- earlier council rounds against `r2`, `r3`, `r3a`, and one stalled sequential
  attempt on the amended final spec are excluded from this artifact

---

## Executive Result

The amended final spec passed council validation without vetoes.

Council split:

- `1/6` roles returned `ACCEPT`
- `5/6` roles returned `CONDITIONAL_ACCEPT`
- `0/6` roles returned `REJECT`
- `0/6` formal vetoes remain

Main conclusion:

`The final spec is architecturally accepted.`

What remains is not a new design dispute. What remains is a set of operational
and implementation clarifications around:

- leaf-wide exclusivity
- leaked-claim recovery
- adapter behavior and bulk query semantics
- rollout mechanics for aggregate enforcement

The council no longer disputes the core model:

- bounded hierarchy
- topology-only committed lock file
- runtime-derived live issue selection
- fail-closed dispatcher

---

## What The Council Now Accepts

Strong convergence emerged on these points.

### C1: Static versus live boundary is correct

Consensus.

The council accepted that:

- `workgraph.lock.json` owns structure and bindings
- Beads owns live execution state
- `active_issue_id` does not belong in committed state

### C2: `source_inputs_hash` is now a real contract

Consensus.

The amended spec now states:

- exact included inputs
- exact excluded inputs
- canonical ordering and serialization

This closed one of the most repeated gaps from `r3a`.

### C3: The dispatch state machine is credible

Consensus with caveats.

Pre-claim, claim, post-claim revalidation, fail-closed behavior, and retry
rules are now concrete enough that multiple roles called the spec
implementation-ready in architecture terms.

### C4: Migration and reshape are conceptually sound

Consensus.

The council accepted:

- `legacy | normalized | mixed_invalid`
- freeze-before-replace
- `dispatch_lifecycle` in the hash
- `shadow-v1 | enforced-v1` as the aggregate enforcement path

### C5: The spec finally says one consistent thing

Consensus.

The philosopher explicitly accepted the spec as coherent end-to-end: structure,
liveness, ownership, and causality now line up instead of fighting each other.

---

## Remaining Patch-Level Concerns

### I1: Leaf-wide exclusivity is still only best-effort

**Severity:** MAJOR  
**Raised by:** Critic, Technician

This is the most serious remaining concern.

The current spec:

- documents the race honestly
- adds local advisory lock mitigation
- relies on post-claim revalidation and observability

But some roles still want a stronger primitive, such as:

- leaf-scoped lock/lease
- authoritative leaf claim token
- stronger adapter support

No one vetoed on this point, but it remains the main operational weakness.

### I2: Leaked-claim recovery needs a named operator path

**Severity:** MAJOR  
**Raised by:** Architect, Pragmatist

The spec says leaked claims must be manually resolved, but the council wants a
clear recovery path:

- what exact command or adapter operation releases a leaked claim
- what audit trail it produces
- how operators discover and clear leaked ownership safely

### I3: Adapter/query contract still needs one more layer of precision

**Severity:** MAJOR  
**Raised by:** Engineer, Pragmatist

The council wants explicit treatment of:

- bulk query semantics for bound issue sets
- structured adapter error payloads in runtime logs/metrics
- exact failure behavior when Beads is unavailable
- whether `is_open` and `status` must always agree

These are interface-shaping details, not architecture disputes.

### I4: Aggregate enforcement rollout still needs harder operational rules

**Severity:** MAJOR  
**Raised by:** Critic, Pragmatist

The council accepted `shadow-v1 -> enforced-v1` conceptually, but some roles
still want stronger rollout mechanics:

- more explicit warning sink/output
- clearer enforcement deadline or trigger
- less room for repositories to linger in warning mode indefinitely

### I5: Minor naming and behavior clarifications remain

**Severity:** MINOR  
**Raised by:** Architect, Philosopher, Engineer

Examples:

- explicitly define `leaf_conflict`
- make it clearer that aggregate `children` in the lock are compiler-derived
- state bulk-query and malformed-input behavior even more directly

These are cleanup items rather than design gaps.

---

## Role Summary

### Architect

`CONDITIONAL_ACCEPT`, no veto.

Main position:

- the core architecture is now sound
- remaining work is operational: leaked claims, freeze ownership, advisory lock
  details, shadow-warning output

### Critic

`CONDITIONAL_ACCEPT`, no veto.

Main position:

- the split-brain problem is resolved
- the remaining concern is operational brittleness, especially leaf exclusivity,
  freeze timing, and Beads dependency

### Technician

`CONDITIONAL_ACCEPT`, no veto.

Main position:

- compiler/runtime contracts are much stronger
- leaf-scoped exclusivity is the only remaining blocker-like concern

### Philosopher

`ACCEPT`, no veto.

Main position:

- the spec is now conceptually coherent
- remaining tensions are policy choices, not contradictions

### Pragmatist

`CONDITIONAL_ACCEPT`, no veto.

Main position:

- the spec is shippable
- but operator-facing error handling and rollout clarity should be finished
  before teams rely on it

### Engineer

`CONDITIONAL_ACCEPT`, no veto.

Main position:

- the schema and contracts are close enough to build against
- final implementation would benefit from sharper adapter and revalidation
  wording

---

## Decision

The final spec should be treated as the accepted implementation target, with one
important qualifier:

- it is accepted at the architectural level
- it still deserves one short polish pass for operational readiness

Recommended next move:

1. keep [2026-04-12-work-atomicity-normalization-final.md](2026-04-12-work-atomicity-normalization-final.md) as the canonical base
2. open one small follow-up for:
   - leaked-claim recovery
   - adapter bulk-query and error semantics
   - leaf-wide exclusivity hardening or explicit non-goal
   - aggregate rollout wording
3. start implementation against the current spec while that follow-up closes the
   last operational gaps

This does not need another architecture council unless the team chooses to add a
new primitive such as:

- leaf-scoped distributed lock
- freeze sidecar file
- stale-data dispatch fallback

Those would be new design moves. The current spec itself is already accepted.
