# LLM Council Raw Log: SDP Work Atomicity Normalization

**Date:** 2026-04-12
**Subject:** [2026-04-12-work-atomicity-normalization.md](2026-04-12-work-atomicity-normalization.md)
**Mode:** Round 1 blind review

---

## Architect — `gpt-5.4`

**Verdict:** CONDITIONAL  
**Domain veto:** NO

Key points:

- critical: no governance for reshaping live work
- major: ambiguity moved, not eliminated, in findings handling
- major: parent workstream vs feature boundary under-specified
- major: adoption plan incomplete and operationally risky
- minor: concurrency rule too weak for real execution

Requested changes:

- add governance matrix for who may create parent WS and split live leaf WS
- add escalation path for disputed classification
- add migration plan and cutover order
- tighten feature vs parent boundary
- add execution-state rules for locking and stale invalidation

Question:

- who exactly may convert an in-flight leaf into a parent, and how is the
  active primary Beads issue migrated without concurrent stale execution?

---

## Critic — `gpt-5.2`

**Verdict:** CONDITIONAL  
**Domain veto:** YES

Key points:

- critical: parent completion override is an unbounded governance escape hatch
- major: findings triage rules are subjective and gameable
- major: “one primary execution issue per leaf” is not enforceable as written
- major: leaf atomic contract definition is not operationalizable
- minor: migration plan missing

Risks called out:

- process theater through “override with rationale”
- scope creep hidden under “same leaf”
- endless retries turning a leaf into a bucket
- mapping ambiguity between helper and canonical sources
- no authority or hard triggers for `leaf -> parent`

Question:

- what exact enforcement design prevents early parent closure and guarantees a
  single primary execution issue per leaf?

---

## Technician — `gpt-5.3-codex`

**Verdict:** CONDITIONAL  
**Domain veto:** YES

Key points:

- critical: no migration protocol, only end-state rules
- critical: “one primary execution issue per leaf” is not implementable as
  written
- major: `ws_kind` alone is insufficient to model a tree
- major: findings split criteria are too subjective
- major: runtime transition behavior is missing

Technically viable parts:

- leaf-only executability
- blocking dispatch of parent workstreams
- keeping findings in Beads by default
- retiring “workstream = always atomic”

Question:

- can the next revision provide schema changes, backfill algorithm, dispatch
  lock semantics, and phased validator rollout with zero orphaned active work?

---

## Philosopher — `gpt-5.4-mini`

**Verdict:** CONDITIONAL  
**Domain veto:** NO

Key points:

- major: leaf atomicity is reclassified, not normalized, if kind can change
  after publication
- major: feature/workstream ownership remains underspecified
- major: findings/remediation rules are still judgment calls dressed as policy
- major: parent completion override is a loophole
- minor: external analogies are rhetorical, not validating evidence

Conceptual support:

- separation of concerns is the right axis
- one primary execution issue per leaf is clean
- in-slice remediation vs new-slice discovery is the right UX boundary

Question:

- what exact creation-time rule determines `parent` vs `leaf`, and can a
  workstream kind remain stable once published?

---

## Pragmatist — `gpt-5.3-codex-spark`

**Verdict:** CONDITIONAL  
**Domain veto:** NO

Key points:

- critical: no transition plan for existing WS/Beads state
- major: feature vs parent WS boundary not formalized
- major: findings rules are not machine-checkable
- major: shipping blast radius is too large for one step
- minor: “one primary issue per leaf” is too rigid without retry lifecycle

What should ship first:

- optional `ws_kind` with deduced default and warning mode
- runtime guard that blocks dispatch of parents
- migration script and report on current repo state

Question:

- can the repo survive a one-sprint migration with no validator hard-fail and
  with automatic reporting/backfill?

---

## Engineer — `gpt-5.4`

**Verdict:** CONDITIONAL  
**Domain veto:** YES

Key points:

- critical: undefined `primary execution issue`
- critical: `ws_kind` is not enough to represent a tree
- major: findings triage is still human judgment, not policy
- major: leaf-to-parent reshaping has no migration protocol
- major: anti-double-dispatch rule needs dispatcher or Beads-side enforcement,
  not validator-only enforcement

Concretely implementable parts:

- add `ws_kind` and block dispatch of parents
- update docs/prompts so only leaf WS is executable atomic task
- require executable mappings to point only to leaf WS
- require canonical parent/child field in workstream metadata
- keep findings in Beads by default

Question:

- what exact machine-readable schema handles parent/child links, issue roles,
  and live reshaping without breaking mappings, evidence, and PR traceability?
