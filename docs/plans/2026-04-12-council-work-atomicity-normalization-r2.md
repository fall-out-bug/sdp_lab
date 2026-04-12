# LLM Council Report: SDP Work Atomicity Normalization — Revision 2

**Date:** 2026-04-12
**Rounds:** 1 of 5
**Subject:** [2026-04-12-work-atomicity-normalization-r2.md](2026-04-12-work-atomicity-normalization-r2.md)
**Consensus:** NOT REACHED
**Decision Owner:** PENDING

---

## Models

| Role | Model | Runtime |
|---|---|---|
| Architect | `gpt-5.4` | `codex-rescue` via Claude/Codex companion |
| Critic | `google/gemini-3.1-pro-preview` | OpenRouter |
| Technician | `deepseek/deepseek-v3.2` | OpenRouter |
| Philosopher | `moonshotai/kimi-k2.5` | OpenRouter |
| Pragmatist | `minimax/minimax-m2.7` | OpenRouter |
| Engineer | `xiaomi/mimo-v2-pro` | OpenRouter |

All 6 roles responded.

---

## Executive Result

Revision 2 is materially stronger than Revision 1.

Shift in council posture:

- `Architect`: from conditional support with major structural blockers to
  **conditional accept**
- `Engineer`: from **veto** to **no veto**
- `Pragmatist`: from “over-engineered” to **conditional ship**
- `Critic`: still **veto**
- `Technician`: still **veto**
- `Philosopher`: still conceptually opposed, but this role has no formal veto

This means Revision 2 successfully closed a large part of the original criticism
set, but did not close the runtime/migration layer.

Formal vetoes still active:

- `Critic`
- `Technician`

Council conclusion:

1. Revision 2 is a real improvement
2. the model is now much closer to implementation-grade
3. adoption is still blocked by runtime state and migration contradictions

---

## What Revision 2 Fixed

Strong convergence emerged on these improvements:

### C1: Bounded hierarchy is better than loose recursion

Consensus.

The explicit max depth of `1` and forbidden-shape rules were seen as a real
improvement over Revision 1.

### C2: Machine-readable schema is much stronger

Consensus.

Compared to Revision 1, the council agreed that Revision 2 finally provides:

- real frontmatter fields
- clear child-to-parent authority
- stricter role naming
- a parseable `## Beads` contract

### C3: The findings decision table is materially better

Consensus with conditions.

Every role agreed that the new decision table is better than the old
judgment-heavy prose.

### C4: Removing `override with rationale` was correct

Strong consensus.

No role argued to bring it back.

---

## Remaining Blocking Issues

### I1: TOCTOU between git workstream state and live Beads claim state

**Severity:** CRITICAL  
**Raised by:** Critic, Technician, Architect

This is now the biggest blocker.

Core problem:

- workstream topology and issue roles live in git-managed documents
- claims live in Beads runtime state
- reshape requires checking one system and mutating the other

The critic's main attack:

- a dispatcher can claim a leaf issue while a reshape PR is merging
- after merge, that claim becomes orphaned against an invalidated topology

The technician's version of the same objection:

- the runtime lock is not atomic
- docs + re-read + claim is still TOCTOU

### I2: Migration protocol still contradicts runtime expectations

**Severity:** CRITICAL  
**Raised by:** Technician, Pragmatist, Architect

The council accepted phased migration in principle, but found one major flaw:

- Phase 2 says “warn mode” with old and new artifacts coexisting
- Phase 4 runtime guard assumes canonical workstream and `## Beads` state

This means the spec still has not fully answered:

- what exact parser behavior applies to old files
- which phases may safely run against mixed-format backlog
- what rollback or dry-run exists before hard enforcement

### I3: Aggregate WS is still ontologically disputed

**Severity:** MAJOR  
**Raised by:** Philosopher, partially Pragmatist

The philosopher's core argument:

- `aggregate` is a planning artifact, not a true workstream
- keeping both `aggregate` and `leaf` under one `workstream` umbrella may still
  preserve a category error

This was not a formal blocker for most roles, but the objection remains alive.

### I4: Aggregate status and completion semantics are still incomplete

**Severity:** MAJOR  
**Raised by:** Architect, Technician, Critic

Revision 2 says aggregate completion is roll-up only, but still leaves open:

- how frontmatter `status` is derived or validated
- whether aggregate-level findings block aggregate completion
- whether status drift is allowed between derived and declared state

### I5: Primary/finding serialization semantics are still unclear

**Severity:** MAJOR  
**Raised by:** Architect, Critic

Revision 2 improved lock language, but the council still wants a direct answer:

- can a `finding` be worked while the `primary` is claimed?
- if not, that is a strong serialization constraint and should be explicit
- if yes, the lock model must distinguish primary vs finding claim slots

---

## Round 2 Split

### S1: Is Revision 2 now adoption-ready as a spec?

No consensus, but progress.

**Architect**: yes, conditionally.

**Pragmatist**: mostly yes, if later phases are split and rollback defined.

**Engineer**: largely yes from a machine-readable perspective, but some fields
still need exact encoding.

**Critic + Technician**: no, because runtime and migration still create broken
states.

### S2: Is `aggregate` still the right entity?

No consensus.

Most roles tolerated it as a practical design.

The philosopher still rejects it as an ontological hybrid.

---

## Role Summaries

### Architect

Best result in Round 2.

Main message:

- Revision 2 closes most Round 1 blockers
- remaining issues are migration of existing `## Beads` sections, aggregate
  status derivation, and explicit serialization semantics

### Critic

Still hard negative.

Main message:

- reshape protocol is unsafe under concurrent live claims
- split between YAML topology and Markdown Beads roles is fragile
- leaf lock can deadlock remediation
- aggregate-level findings are not fully accounted for

### Technician

Still formal veto.

Main message:

- runtime and migration are self-contradictory
- strict syntax plus warn mode is not fully specified
- runtime guard depends on canonical state that migration explicitly delays

### Philosopher

Still conceptually opposed.

Main message:

- Revision 2 fixed syntax but not the deeper dual-authority problem
- `aggregate` may still be the wrong kind of object

### Pragmatist

Clearly softer than Round 1.

Main message:

- Phases 1-3 are shippable
- Phase 4 needs rollback and maybe dry-run subphase
- Phase 5 should probably start with new workstreams only

### Engineer

Biggest positive movement.

Main message:

- spec is now partially machine-readable and implementation can begin
- remaining issues are exact encoding of discriminator logic, planning
  authority, migration gates, and some validator details

---

## Council Recommendation

Do **not** go straight to implementation of the whole model.

But also do **not** throw Revision 2 away.

Revision 2 should become the base for **Revision 3**, focused narrowly on the
runtime contract and migration contract.

### Revision 3 should answer

1. **Runtime state authority**
   - how reshape and dispatch avoid TOCTOU
   - whether Beads roles stay in Markdown or move into a more atomic artifact

2. **Serialization semantics**
   - whether `primary` and `finding` are mutually exclusive claim slots
   - if yes, say so explicitly
   - if no, define separate lock semantics

3. **Migration parser behavior**
   - exact behavior on old `## Beads` syntax
   - exact success criteria for moving from warn mode to enforcement

4. **Aggregate completion semantics**
   - aggregate-level finding treatment
   - derived status rule

5. **Phase 4 rollback**
   - explicit off-switch or dry-run mode before blocking dispatch

---

## Decision Needed From Owner

Choose one:

1. proceed to Revision 3 before any implementation
2. freeze Revision 2 as “good enough” and implement only Phases 1-3
3. reject the aggregate model and reopen the ontology question

---

## Audit Log

Raw role responses: [2026-04-12-council-work-atomicity-normalization-r2-raw.md](2026-04-12-council-work-atomicity-normalization-r2-raw.md)
