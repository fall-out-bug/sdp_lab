# LLM Council Report: SDP Work Atomicity Normalization - Revision 3a

**Date:** 2026-04-12
**Rounds:** 1 of 5
**Subject:** [2026-04-12-work-atomicity-normalization-r3a-live-state-boundary.md](2026-04-12-work-atomicity-normalization-r3a-live-state-boundary.md)
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

All 6 roles responded in the final valid round.

Validity note:

- user requested not to use `openai/gpt-5.4` on OpenRouter in this round; the
  architect role therefore used `anthropic/claude-sonnet-4.6`
- a first sequential run was discarded because it stalled before producing a
  complete final roster
- an earlier critic attempt on `google/gemini-3.1-pro-preview` was excluded
  because the provider repeatedly returned truncated or incomplete output

---

## Executive Result

This round materially changed the council posture.

Council split:

- `3/6` roles returned `ACCEPT`
- `3/6` roles returned `CONDITIONAL_ACCEPT`
- `0/6` roles returned `REJECT`
- `0/6` formal vetoes remain

Main conclusion:

`Revision 3a resolves the prior architectural blocker.`

The council now broadly agrees with the core boundary:

- committed lock file owns topology and issue bindings
- live issue selection stays runtime-derived from Beads

The dispute is no longer about architecture. It is now about specification
completeness and runtime guard rails.

---

## What Revision 3a Clearly Fixed

Strong convergence emerged on five points.

### C1: `active_issue_id` does not belong in the committed lock

Consensus.

Every role agreed that removing `active_issue_id` from
`workgraph.lock.json` resolves the core split-brain from Revision 3.

### C2: Static versus live authority is now coherent

Consensus.

The new boundary was consistently endorsed:

- git owns stable structure
- Beads owns mutable execution state

### C3: `source_inputs_hash` is better than `git_commit`

Consensus.

The council agreed that Revision 3's loose `git_commit` field was the wrong
freshness primitive and that input hashing is directionally correct.

### C4: Runtime issue selection belongs at dispatch time

Consensus.

No role argued to keep compile-time issue selection in the committed lock file.
Even the critic moved from veto to acceptance once the design stopped snapshotting
live queue state into git.

### C5: The patch stayed narrow

Consensus.

Multiple roles explicitly praised Revision 3a for staying on the one disputed
boundary instead of reopening hierarchy, migration phases, or terminology.

---

## Remaining Open Items

The remaining concerns are now patch-level.

### I1: `source_inputs_hash` needs a strict canonical contract

**Severity:** MAJOR  
**Raised by:** Architect, Pragmatist, Engineer

This was the most repeated remaining issue.

The council wants an explicit answer to:

- which exact inputs are hashed
- which inputs are explicitly excluded
- how those inputs are normalized and serialized

Without that, independent implementations could compute different freshness
hashes for the same feature state.

### I2: Revalidation failure semantics need to be fail-closed and explicit

**Severity:** MAJOR  
**Raised by:** Architect, Pragmatist, Engineer

Revision 3a says to abort if the chosen issue is no longer active, but several
roles want the contract completed:

- what happens if Beads query fails
- what happens if the lock identity changed between selection and claim
- whether retry is automatic or manual
- what should be logged or surfaced to operators

The direction is accepted. The failure path still needs tightening.

### I3: Leaf-wide exclusivity remains a known limitation

**Severity:** MAJOR  
**Raised by:** Architect, Critic

The critic no longer vetoes the patch, but still flags a real race:

- if Beads cannot enforce leaf-wide exclusivity atomically
- two dispatchers can still claim different bound issues on the same leaf
- both may then abort on revalidation and potentially livelock on retries

The council now treats this as a known limitation rather than a blocker, but it
should be documented honestly and observed in runtime metrics.

### I4: Aggregate declared/derived mismatch rule still needs migration guidance

**Severity:** MAJOR  
**Raised by:** Pragmatist, Critic

The compile-error rule is cleaner than a half-valid lock file, but the council
still wants the adoption rule tightened:

- does the compile error apply only to fully normalized features
- is there any grace window for existing violations
- is this a hard block immediately or after cutoff

This is mostly a rollout concern, not a design objection.

### I5: Runtime query contract needs to be machine-readable

**Severity:** MAJOR  
**Raised by:** Engineer

The dispatcher now depends on a live Beads query contract for:

- blocking status
- priority
- creation time
- issue-id tie-breakers

The council wants that contract written down explicitly so compiler and
dispatcher authors do not drift.

---

## Role Summary

### Architect

`CONDITIONAL_ACCEPT`, no veto.

Main position:

- the blocker is resolved
- remaining work is mostly around hash identity, lifecycle re-checks, and the
  query surface needed for leaf-wide exclusivity

### Critic

`ACCEPT`, no veto.

Main position:

- the split-brain objection is gone once live issue selection leaves the lock
- remaining discomfort is about livelock risk and aggregate-status DX, not the
  core boundary

### Technician

`ACCEPT`, no veto.

Main position:

- the static/live boundary is now correct
- remaining work is straightforward implementation rather than architectural
  dispute

### Philosopher

`ACCEPT`, no veto.

Main position:

- the ontology is now principled because the design finally distinguishes
  author-caused structure from world-caused selection
- only minor clarification remains

### Pragmatist

`CONDITIONAL_ACCEPT`, no veto.

Main position:

- this is the right narrow patch
- but adoption should add explicit hash scope, revalidation failure rules, and
  a migration path for aggregate compile errors

### Engineer

`ACCEPT`, no veto.

Main position:

- architecture is ready
- implementation needs a formal Beads query contract, hash canonicalization,
  and dispatcher state machine

---

## Decision

Revision 3a should be treated as the accepted direction for the live-state
boundary.

The council no longer disputes the core design:

- `workgraph.lock.json` should contain topology and bindings only
- live active issue selection should be runtime-derived from Beads

Recommended next move:

1. patch the memo once more with explicit `source_inputs_hash` inputs
2. define fail-closed revalidation semantics
3. document leaf-wide exclusivity as a known limitation unless Beads can
   provide a stronger primitive
4. write the machine-readable Beads query contract and dispatcher state machine

That is now an implementation-readiness pass, not a new architecture debate.
