# LLM Council Report: SDP Work Atomicity Normalization - Revision 3

**Date:** 2026-04-12
**Rounds:** 1 of 5
**Subject:** [2026-04-12-work-atomicity-normalization-r3.md](2026-04-12-work-atomicity-normalization-r3.md)
**Consensus:** NOT REACHED
**Decision Owner:** PENDING

---

## Models

| Role | Model | Runtime |
|---|---|---|
| Architect | `openai/gpt-5.4` | OpenRouter |
| Critic | `google/gemini-3.1-pro-preview` | OpenRouter |
| Technician | `deepseek/deepseek-v3.2` | OpenRouter |
| Philosopher | `anthropic/claude-opus-4.6` | OpenRouter |
| Pragmatist | `minimax/minimax-m2.7` | OpenRouter |
| Engineer | `xiaomi/mimo-v2-pro` | OpenRouter |

All 6 roles responded in the final valid round.

Validity note:

- discarded architect attempt via `codex-rescue` was excluded because the answer
  identified itself as a simulation rather than a distinct model runtime
- discarded philosopher attempts via `moonshotai/kimi-k2.5` were excluded
  because the provider returned reasoning-only or truncated output instead of a
  usable council answer

---

## Executive Result

Revision 3 improved the design materially, but it did not reach adoption-ready
consensus.

Council split:

- `5/6` roles returned `CONDITIONAL_ACCEPT`
- `1/6` role returned `REJECT`
- active formal vetoes: `Critic`

This is not a cosmetic veto. The critic's objection attacks the core runtime
boundary:

- whether `active_issue_id` belongs in the static lock file at all
- whether a git-committed artifact may safely snapshot live Beads execution
  state

If the critic is right, Revision 3 still bakes live queue state into static git
state and recreates a split-brain at a different layer.

Council conclusion:

1. Revision 3 closed most of the Round 2 ambiguity
2. Revision 3 did **not** fully settle the compiled-state versus live-state
   boundary
3. adoption should wait for one more patch pass on runtime state ownership

---

## What Revision 3 Clearly Fixed

Strong convergence emerged on these improvements.

### C1: Feature-level migration classification is the right move

Consensus.

The switch to `legacy | normalized | mixed_invalid` was seen as a real fix for
the Revision 2 migration contradiction.

### C2: Runtime authority separation is much better

Consensus.

Moving dispatch away from live Markdown parsing and toward a compiled lock file
was seen as the right structural direction.

### C3: Leaf execution semantics are finally explicit

Consensus.

The single execution slot and deterministic ordering for `primary` versus
`finding` resolved a real ambiguity from Round 2.

### C4: Reshape protocol is stricter and safer

Consensus with caveats.

Freeze-before-replace is widely seen as the correct shape. The remaining debate
is about enforcement timing and failure semantics, not about the need for a
reshape guard.

### C5: Aggregate completion is closer to machine-enforceable

Consensus with caveats.

The council agreed that Revision 3 is far better than Revision 2 on aggregate
status, but some roles still dislike the human-maintained `declared_status`
mirror and want the derivation rules rewritten more explicitly.

---

## Remaining Blocking Questions

### B1: Static lock file versus live execution state

**Severity:** CRITICAL  
**Raised by:** Critic, Architect, Technician, Engineer

This is the main unresolved dispute.

Revision 3 stores `active_issue_id` in `workgraph.lock.json`, but
`active_issue_id` is derived from live Beads properties such as:

- open findings
- blocking versus non-blocking findings
- priority
- creation order

The critic argues this is still broken:

- a new blocking finding can appear in Beads after the lock file is generated
- the static lock file then points to stale execution state
- runtime may execute the wrong issue unless git regenerates first

Other roles were softer, but still asked for tighter rules on:

- which inputs are topology and belong in the lock file
- which inputs are live state and must be resolved at dispatch time

This is the only issue that still carries a formal veto.

### B2: Lock identity and generation lifecycle are underspecified

**Severity:** MAJOR  
**Raised by:** Architect, Critic, Technician, Pragmatist

Revision 3 says normalized features are not dispatchable if the lock file does
not match `HEAD`, but multiple roles called this incomplete.

Open questions:

- what exactly counts as a match
- how lock generation is triggered
- whether the lock file is committed or generated on demand
- whether the `git_commit` field is self-referential or refers to input state

The critic explicitly called out a possible git hash paradox if the lock file
tries to embed the commit that contains the lock file itself.

### B3: Claim and reshape failure semantics are still too loose

**Severity:** MAJOR  
**Raised by:** Architect, Critic, Philosopher, Pragmatist, Engineer

Revision 3 improved pre-claim and post-claim logic, but the council still wants
more exact runtime behavior:

- what is the atomic claim primitive
- what happens after revalidation failure
- whether retries are automatic or manual
- how reshape freeze interacts with already-running work
- whether mode transitions require quiescence before authority changes

The reshape protocol is directionally accepted. The missing part is exact
failure behavior.

### B4: Aggregate status still has DX and completeness concerns

**Severity:** MAJOR  
**Raised by:** Critic, Philosopher, Pragmatist

Two complaints remained:

1. the rule set for aggregate derivation should be written as a clearer,
   exhaustive decision table
2. requiring `declared_status == derived_status` may create human-toil if the
   system already derives the machine truth

This is not a structural veto, but it is a credible DX warning.

### B5: Implementation contracts are still too thin

**Severity:** MAJOR  
**Raised by:** Technician, Engineer, Pragmatist

Multiple roles converged on missing delivery contracts:

- full JSON schema for `workgraph.lock.json`
- validator error catalog
- dispatcher state machine
- exact source and legal values for finding metadata
- explicit meaning of `historical_issue_ids`

The council no longer sees ontology as the main blocker. It now sees missing
machine contracts as the main implementation risk.

---

## Role Summary

### Architect

`CONDITIONAL_ACCEPT`, no veto.

Main position:

- runtime and migration design is now plausible
- remaining work is mostly about strict identity rules, atomic claim authority,
  and explicit frozen/dispatchable state

### Critic

`REJECT`, formal veto.

Main position:

- Revision 3 still snapshots live Beads execution state into static git state
- `active_issue_id` should be derived at runtime from live Beads state, not
  compiled into the lock file
- aggregate declared status is still governance theater

### Technician

`CONDITIONAL_ACCEPT`, no veto.

Main position:

- design direction is correct
- operational semantics are still incomplete around lock refresh, rollback,
  mixed-feature recovery, and concurrent compiler runs

### Philosopher

`CONDITIONAL_ACCEPT`, no veto.

Main position:

- the ontology is now mostly coherent
- remaining issues are about rule completeness, especially aggregate
  derivation and failure semantics

### Pragmatist

`CONDITIONAL_ACCEPT`, no veto.

Main position:

- Revision 3 is close enough to adopt
- but Phase 3/Phase 4 need explicit generation triggers, rollback triggers, and
  legacy exit criteria

### Engineer

`CONDITIONAL_ACCEPT`, no veto.

Main position:

- the design is now machine-oriented instead of purely prose-oriented
- but implementation should not start before schema, validator, and dispatcher
  contracts are explicit

---

## Decision

Revision 3 should **not** be adopted as-is.

It is close, but one core design question remains unresolved:

- is `workgraph.lock.json` a topology artifact only
- or does it also own volatile live execution selection such as
  `active_issue_id`

Until that is answered, the council does not have real consensus on runtime
state ownership.

Recommended next move:

1. write one short patch revision focused only on the live-state boundary
2. decide whether `active_issue_id` is compiled or runtime-derived
3. define lock identity/generation rules and claim failure semantics in the same
   patch
4. avoid reopening hierarchy or terminology again

If that patch resolves the critic's split-brain objection, the rest of the
council feedback looks patch-level rather than architectural.
