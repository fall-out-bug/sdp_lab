# LLM Council Report: SDP Work Atomicity Normalization

**Date:** 2026-04-12
**Rounds:** 1
**Subject:** [2026-04-12-work-atomicity-normalization.md](2026-04-12-work-atomicity-normalization.md)
**Consensus:** PARTIAL
**Decision Owner:** PENDING

---

## Models

| Role | Model |
|---|---|
| Architect | `gpt-5.4` |
| Critic | `gpt-5.2` |
| Technician | `gpt-5.3-codex` |
| Philosopher | `gpt-5.4-mini` |
| Pragmatist | `gpt-5.3-codex-spark` |
| Engineer | `gpt-5.4` |

Note: current tool inventory exposed 5 distinct model SKUs, so `gpt-5.4` was
reused for `Architect` and `Engineer`.

---

## Executive Result

The council did **not** reject the direction outright.

But it also did **not** approve the memo as a canonical SDP model.

Final round state:

- `0/6` unconditional support
- `6/6` conditional support
- `3` domain vetoes
  - `Critic`
  - `Technician`
  - `Engineer`

The common pattern was consistent:

1. the diagnosis is right;
2. Option B is the strongest direction;
3. the memo is still one layer too abstract to adopt safely.

---

## Council Consensus

### C1: The current problem is real and structural

Consensus.

The council agreed that SDP has a genuine control-flow drift:

- `workstream` is described as atomic in some sources
- `beads issue` is the live execution atom in others
- findings/remediation have no stable normalization rule

This is not a wording cleanup. It is a model conflict.

### C2: Option B is the right direction

Consensus with conditions.

The strongest supported direction remains:

- keep `Feature` as the outcome container
- keep `workstream` as the planning/contract layer
- keep `Beads` as the live execution graph
- make only executable `leaf workstream` dispatchable

No role argued that the current mixed model should remain canonical.

### C3: Parent workstreams must be non-dispatchable

Consensus.

This was the cleanest and least disputed guardrail in the memo.

### C4: Findings should stay in Beads by default

Consensus with conditions.

The council agreed that review/CI/drift/QA findings should usually stay inside
the current leaf workstream unless they clearly create a new independent slice.

However, the current memo does not define that split strongly enough.

---

## Blocking Issues

### I1: No migration protocol for existing WS/Beads state

**Severity:** CRITICAL  
**Raised by:** Pragmatist, Technician, Architect, Critic

The memo defines an end-state but not a safe transition:

- no backfill strategy for current backlog files
- no compatibility window
- no phased validator rollout
- no guarantee that active work is not stranded during cutover

This was the most repeated implementation objection.

### I2: `primary execution issue` is undefined

**Severity:** CRITICAL  
**Raised by:** Technician, Critic, Engineer

The memo makes this the core live invariant, but never defines:

- where `primary` is stored
- how it differs from `finding` or `historical`
- how concurrent claims are prevented
- how stale or reopened issues are handled

Without this, “one primary execution issue per leaf” is a slogan, not an
enforceable rule.

### I3: `ws_kind` is not enough to model a tree

**Severity:** CRITICAL  
**Raised by:** Technician, Engineer

`parent|leaf` alone is insufficient. The council wants explicit tree schema:

- `parent_ws_id` or equivalent authoritative linkage
- child ordering rules
- cycle prevention
- derived completion semantics from real child links, not prose

### I4: Findings triage is still subjective

**Severity:** MAJOR  
**Raised by:** all roles

Phrases such as:

- “scope boundary unchanged”
- “new acceptance contract”
- “can be reviewed independently”

are directionally correct but not operationally strong enough for automation or
consistent human use.

### I5: Governance for reshaping live work is missing

**Severity:** MAJOR  
**Raised by:** Architect, Technician, Engineer

The memo allows `leaf -> parent` reshaping but does not define:

- who has authority to do it
- who approves it
- who migrates the active issue and mappings
- how concurrent execution on the stale leaf is blocked

### I6: Parent completion override is a loophole

**Severity:** CRITICAL  
**Raised by:** Critic, Philosopher, Engineer

The clause:

> parent completion is derived from child completion unless explicitly
> overridden with rationale

was treated as a governance escape hatch.

Council conclusion:

- this should not ship in its current form
- if any override exists at all, it needs hard constraints, explicit authority,
  and auditable evidence

### I7: Feature vs parent workstream boundary is still underspecified

**Severity:** MAJOR  
**Raised by:** Architect, Philosopher, Pragmatist

The council accepted the direction but wants a hard test for:

- when decomposition belongs under a `Feature`
- when a `parent workstream` is legitimate
- how parent acceptance differs from feature acceptance

Without this, parent WS risks becoming a shadow feature.

---

## Minority And Role-Specific Pressure

### Critic veto

The memo still contains too many governance loopholes:

- parent completion override
- fuzzy findings triage
- undefined `primary`
- unclear canonical source for mappings

### Technician veto

The memo is not machine-enforceable yet:

- no schema for parent/child links
- no schema for issue role
- no migration protocol
- no concurrency/dispatch semantics

### Engineer veto

The proposal is implementable only after narrowing:

- define exact metadata fields
- define exact issue-role model
- define exact reshaping protocol
- stop relying on prose where validators need fields

---

## Council Recommendation

Do **not** adopt the memo as canonical SDP policy yet.

Adopt a narrower next step:

### R1: Keep Option B as the working direction

Do not reopen the larger strategic question unless new evidence appears.

### R2: Produce Revision 2 of the memo around four missing contracts

1. **Tree schema**
   - add explicit parent/child linkage
   - define authoritative direction of linkage
   - define completion derivation

2. **Issue role model**
   - define `primary | finding | historical`
   - define live invariants
   - define reopen/retry semantics

3. **Migration protocol**
   - existing backlog backfill
   - compatibility window
   - warn-first then fail-later validator rollout
   - cutover sequence for docs, prompts, validators, queue tooling

4. **Reshaping governance**
   - who may convert `leaf -> parent`
   - approval path
   - issue/mapping/evidence migration custody
   - stale execution invalidation

### R3: Remove or harden the parent completion override

Default recommendation: remove it from the next draft.

### R4: Replace fuzzy findings guidance with a decision table

The next draft should classify at least these cases:

- CI/test/review break under unchanged acceptance
- discovered adjacent scope with its own acceptance
- oversized leaf detected before execution
- oversized leaf detected mid-flight
- urgent fix against active leaf

### R5: Re-run council only after Revision 2 exists

The main blockers are now concrete enough that another round on the same memo
would mostly repeat itself.

---

## Decision Needed From Owner

Choose one:

1. approve Revision 2 work on top of Option B
2. reject Option B and revert to strict atomic workstream doctrine
3. reject workstream primacy and move to Beads-first execution model
4. defer and explicitly accept the current mixed model risk

---

## Round Convergence

| Round | Active models | Unconditional support | Conditional support | Vetoes |
|---|---:|---:|---:|---:|
| 1 | 6/6 | 0 | 6 | 3 |

---

## Audit Log

Raw role responses: [2026-04-12-council-work-atomicity-normalization-raw.md](2026-04-12-council-work-atomicity-normalization-raw.md)
