# LLM Council Raw Log: SDP Work Atomicity Normalization

**Date:** 2026-04-12
**Round:** 1 blind review
**Subject:** [2026-04-12-work-atomicity-normalization.md](2026-04-12-work-atomicity-normalization.md)
**Status:** Protocol-compliant council rerun with the intended mixed model roster

---

## Model Roster

| Role | Model | Runtime | Status |
|---|---|---|---|
| Architect | `gpt-5.4` | `codex-rescue` via Claude/Codex companion | responded |
| Critic | `google/gemini-3.1-pro-preview` | OpenRouter | responded |
| Technician | `deepseek/deepseek-v3.2` | OpenRouter | responded |
| Philosopher | `moonshotai/kimi-k2.5` | OpenRouter | responded |
| Pragmatist | `minimax/minimax-m2.7` | OpenRouter | responded |
| Engineer | `xiaomi/mimo-v2-pro` | OpenRouter | responded |

---

## Architect — `gpt-5.4` via `codex-rescue`

**Verdict:** CONDITIONAL ACCEPT
**Domain veto:** NO

Main points:

- Option B is the correct direction
- current operational spec is not tight enough to adopt
- strongest issues:
  - unresolved isomorphism between `Feature` and `parent workstream`
  - findings routing is still heuristic, not rule
  - no migration path or phased rollout
  - undefined `primary` boundary conditions
  - no rollback/arbitration for `leaf -> parent`

Key question:

- who has override authority for parent completion, and how is override
  normalization prevented?

---

## Critic — `google/gemini-3.1-pro-preview`

**Verdict:** REJECT IN CURRENT FORM
**Domain veto:** YES

Main points:

- process integrity veto
- parent completion override is a fatal governance loophole
- remediation routing remains subjective and gameable
- mid-flight leaf-to-parent conversion can corrupt state
- no mechanism prevents “zombie leaf” endless sequential remediation

Key question:

- how does the system mechanically stop endless sequential Beads issues from
  turning one leaf into hidden parent scope?

---

## Technician — `deepseek/deepseek-v3.2`

**Verdict:** technically feasible, high migration cost, underspecified runtime
**Domain veto:** NO

Main points:

- no technical impossibility
- but serious underspecification:
  - state transitions and locking
  - derived parent completion runtime
  - validator/runtime split
  - historical mapping ambiguity
  - missing migration path

Key question:

- what exact mechanism enforces one primary execution issue per leaf and manages
  atomic `leaf -> parent` conversion in a multi-actor system?

---

## Philosopher — `moonshotai/kimi-k2.5`

**Verdict:** REJECT
**Protocol veto authority:** none for this role

Main points:

- the memo may rename ambiguity instead of removing it
- `atomicity` is being conflated with `dispatchability`
- the model is not truly recursive, just a rigid two-tier split
- a leaf with repeated execution episodes is not really atomic
- `Feature` and `parent WS` may be redundant containers
- Workstream primacy may already be fiction if only Beads truly execute

Key question:

- when a leaf accumulates repeated issue history, who has authority to declare
  it was never atomic in the first place?

---

## Pragmatist — `minimax/minimax-m2.7`

**Verdict:** Option B is over-engineered for the stated problem
**Protocol veto authority:** none for this role

Main points:

- the memo solves classification, not execution
- docs normalization and findings rules should ship first
- tooling and schema migration should be deferred
- `ws_kind` and invariants are too heavy as a first move
- parent WS may not deserve first-class schema treatment yet

What should ship first:

- revised `docs/TERMS.md`
- standalone findings/remediation rules
- one-page dispatcher guide

Key question:

- if humans can still mis-declare `leaf`, what is the real enforcement
  mechanism?

---

## Engineer — `xiaomi/mimo-v2-pro`

**Verdict:** REJECT
**Domain veto:** YES

Main points:

- current memo lacks exact machine-readable schema
- no exact validator logic
- mapping file semantics undefined
- migration/backfill missing
- findings policy still too vague to encode

Concretely implementable pieces:

- `ws_kind: parent|leaf`
- block dispatch of parent WS
- require executable mappings to point only at leaf WS
- require canonical parent/child field

Key question:

- what exact frontmatter and mapping schema implements Option B without forcing
  implementers to invent the missing details?

---

## Raw Notes

This round intentionally stopped after blind review.

Reason:

- the draft is missing explicit contracts, not merely suffering from
  perspective disagreement
- further rebuttal on the same memo would be lower-signal than revising the
  spec itself
