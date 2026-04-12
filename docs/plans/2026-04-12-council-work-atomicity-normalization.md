# LLM Council Report: SDP Work Atomicity Normalization

**Date:** 2026-04-12
**Rounds:** 1 of 5
**Subject:** [2026-04-12-work-atomicity-normalization.md](2026-04-12-work-atomicity-normalization.md)
**Consensus:** NOT REACHED
**Decision Owner:** PENDING
**Status:** This report supersedes the earlier invalid local-only pseudo-council.

---

## Models

| Role | Model | Runtime |
|---|---|---|
| Architect | `gpt-5.4` via `codex-rescue` | Codex companion through `claude --agent codex-rescue` |
| Critic | `google/gemini-3.1-pro-preview` | OpenRouter |
| Technician | `deepseek/deepseek-v3.2` | OpenRouter |
| Philosopher | `moonshotai/kimi-k2.5` | OpenRouter |
| Pragmatist | `minimax/minimax-m2.7` | OpenRouter |
| Engineer | `xiaomi/mimo-v2-pro` | OpenRouter |

All 6 roles responded.

---

## Executive Result

This was a proper blind review round with the intended model roster.

The result is harsher than the invalid draft council was:

- `Architect`: conditional accept
- `Technician`: technically feasible, but underspecified
- `Critic`: reject in current form
- `Philosopher`: reject
- `Pragmatist`: over-engineered for the stated problem
- `Engineer`: reject

Protocol note:

- `Critic` and `Engineer` raised formal domain vetoes
- `Philosopher` used veto language in content, but that role has no protocol
  veto authority
- `Pragmatist` has no protocol veto authority either

Council conclusion:

1. the drift diagnosis is correct;
2. the current memo is **not adoption-ready**;
3. the council did **not** converge on a single final normalization strategy.

---

## Consensus

### C1: The current SDP model really is split

Consensus.

All roles agreed that the repo currently mixes incompatible meanings:

- `workstream` as atomic task
- `workstream` as contract only
- `beads issue` as real execution atom

No role defended the status quo as healthy.

### C2: The current memo is too abstract to adopt

Consensus.

Every role found the same structural gap:

- the memo states end-state rules
- but not the machine-readable schema, migration protocol, or enforcement model

This was the dominant blocker.

### C3: The findings/remediation rules are still too subjective

Consensus.

Every role attacked some version of the same problem:

- “scope boundary unchanged”
- “new acceptance contract”
- “problem re-shaping”
- “execution quality”

These are useful human intuitions, but not yet enforceable policy.

### C4: `override with rationale` is unacceptable in current form

Strong consensus.

This clause was treated as a governance escape hatch, not as a harmless safety
valve.

Default council recommendation: remove it from the next revision.

---

## Split

### S1: Is Option B the right end-state, or is it over-engineered?

No consensus.

**Architect + Technician**: Option B is directionally right.

- keep `Feature`
- keep `workstream`
- keep `Beads`
- make only executable leaf workstreams dispatchable

**Pragmatist**: Option B solves the wrong layer first.

- ship docs normalization and findings rules first
- defer schema migration and tooling

**Philosopher**: Option B preserves the dualism instead of resolving it.

- if only Beads issues execute, Workstream primacy may already be fiction

**Engineer + Critic**: direction is plausible, but current memo is too
underspecified to endorse as an engineering spec.

### S2: Is `parent workstream` a real first-class concept or a shadow feature?

No consensus.

This was a recurring structural objection:

- the memo does not yet prove a hard boundary between `Feature` and `parent WS`
- several roles argued that parent WS could easily become “feature in disguise”

---

## Blocking Issues

### I1: No machine-readable tree schema

**Severity:** CRITICAL  
**Raised by:** Technician, Engineer, Architect

`ws_kind: parent|leaf` is not enough.

Missing:

- `parent_ws_id` or equivalent canonical linkage
- child ordering rules
- cycle prevention
- completion derivation rules

### I2: `primary execution issue` is undefined

**Severity:** CRITICAL  
**Raised by:** Critic, Engineer, Technician

The memo makes this the core invariant without defining:

- where `primary` lives
- how it differs from `finding` or `historical`
- how it is assigned
- how double-dispatch is blocked
- how retries/reopens interact with the invariant

### I3: No migration protocol

**Severity:** CRITICAL  
**Raised by:** Pragmatist, Technician, Architect, Engineer

The memo lists required repo changes but no cutover design:

- no backfill plan for current WS files
- no warn-first vs fail-later rollout
- no compatibility window
- no sequencing across docs, prompts, validators, and queue tooling

### I4: Findings triage is not operational policy yet

**Severity:** MAJOR  
**Raised by:** all roles

The current rules still depend on judgment calls and will recreate drift during
triage even if the nouns are cleaned up.

### I5: Live reshaping governance is undefined

**Severity:** MAJOR  
**Raised by:** Architect, Technician, Engineer, Critic

The memo allows `leaf -> parent` reshaping but does not define:

- who may trigger it
- who approves it
- what happens to the active Beads issue
- how traceability, PR linkage, and evidence survive the transition
- how stale execution against the old leaf is invalidated

### I6: Parent completion override is a governance hole

**Severity:** CRITICAL  
**Raised by:** Critic, Philosopher, Engineer, Architect

The council treated this as a bypass channel for process theater.

---

## Role Summaries

### Architect

Directionally positive on Option B, but blocked on:

- no governance for reshaping live work
- no hard feature vs parent boundary
- no migration plan

### Critic

Hardest negative review.

Main attack:

- parent completion override destroys the trust chain
- findings triage is gameable
- “single primary issue” is not enforceable as written

### Technician

No technical-impossibility veto, but very strong execution warning.

Main attack:

- state transitions, locks, and parent completion are unspecified state-machine
  behavior, not small details

### Philosopher

The strongest conceptual challenge.

Main attack:

- the memo may be renaming ambiguity instead of removing it
- `atomicity` and `dispatchability` are being conflated
- `Feature` and `parent WS` may be redundant containers

### Pragmatist

The strongest scope-pressure review.

Main attack:

- the memo tries to ship docs + schema + validators + mapping rules together
- a smaller normalization should land first

### Engineer

Strong implementation-blocker critique.

Main attack:

- no exact schema
- no validator algorithm
- no exact mapping semantics
- no migration/backfill spec

---

## Council Recommendation

Do **not** adopt the current memo as canonical SDP policy.

The next useful step is not another debate round on the same artifact.

It is **Revision 2** with explicit contracts.

### Revision 2 must add

1. **Tree schema**
   - exact frontmatter fields
   - canonical parent/child linkage
   - max depth rule
   - feature vs parent discriminator

2. **Issue role model**
   - exact role set such as `primary | finding | historical`
   - exact location of that metadata
   - exact rule for “one active primary issue per leaf”

3. **Migration protocol**
   - current backlog backfill
   - compatibility window
   - warn-first rollout
   - hard-enforcement cutover sequence

4. **Reshaping governance**
   - who may propose `leaf -> parent`
   - who approves
   - what happens to active issues and mappings
   - stale execution invalidation

5. **Decision table for findings**
   - unchanged acceptance → stay in leaf
   - acceptance changed → new leaf
   - oversized slice discovered mid-flight → reshape path

### Revision 2 should remove

- `override with rationale` in its current form

---

## Decision Needed From Owner

Choose one:

1. keep pursuing Option B, but only through Revision 2
2. pivot to a narrower docs-and-dispatch normalization first
3. reopen the larger ontology question and reconsider Beads-first execution

---

## Why The Council Stopped After One Round

This was a valid stop, not an abandoned process.

Reason:

- the blockers are not disagreements over subtle trade-offs
- they are missing contracts in the artifact itself

A rebuttal round on the same memo would mostly produce repetition.

The right next input for council is a revised spec, not more argument on an
underspecified one.

---

## Audit Log

Raw role responses: [2026-04-12-council-work-atomicity-normalization-raw.md](2026-04-12-council-work-atomicity-normalization-raw.md)
