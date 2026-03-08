# Single Release Feature Selection

> **Status:** Decision made
> **Date:** 2026-03-08
> **Goal:** Choose exactly one useful feature to polish for the next OSS-facing SDP release

---

## Decision

We should implement and polish **guided next-step UX** as the single release feature.

This means one coherent user-facing slice:

- versioned `status` and `instructions` contracts
- unified `sdp status` and `sdp next` experience
- deterministic ready/blocked/next guidance
- structured failure recovery hints

Primary workstreams:

- `docs/workstreams/backlog/00-068-04.md`
- `docs/workstreams/backlog/00-068-05.md`
- `docs/workstreams/backlog/00-069-04.md`
- `docs/workstreams/backlog/00-069-05.md`

---

## The Two Lenses We Compared

### 1. What could move to OSS now, but is still weak?

Best answer: **`sdp-evidence` as a first-class OSS binary**.

Why it scored high:

- strong technical core already exists in `sdp/`
- docs and hooks already assume it matters
- release gap is mostly packaging and consolidation

Why it is **not** the chosen feature:

- it is more release-packaging than product differentiation
- there are still two evidence implementations across repos
- it improves trust packaging, but not day-one usability

### 2. What is the most promising feature to build right now?

Best answer: **guided next-step UX**.

Why it scored highest:

- public CLI already exposes `sdp status` and `sdp next`
- code and tests already exist, but the current experience is weak
- this is a direct answer to the biggest OSS UX question: "what do I do next?"
- it aligns with the OpenSpec-inspired direction without forcing bigger architectural bets yet

---

## Why Guided Next-Step UX Wins

### User value

It reduces first-run confusion immediately. A good OSS tool should not require users to memorize the workflow before it becomes helpful.

### Existing momentum

The public CLI already contains the right surface area:

- `sdp/sdp-plugin/cmd/sdp/status_text.go`
- `sdp/sdp-plugin/cmd/sdp/next.go`
- `sdp/sdp-plugin/cmd/sdp/next_test.go`

The problem is not absence. The problem is weak integration and weak polish.

### Scope discipline

This is big enough to matter, but small enough to ship well:

- better than choosing a huge integration like OpenSpec import end-to-end
- more product-visible than contract-only work
- less risky than cross-repo evidence consolidation right now

### Release story

It is easy to explain in one sentence:

> SDP tells you what is ready, what is blocked, and what to do next.

That is stronger OSS positioning than "we also packaged another validator binary."

---

## Current Weaknesses to Fix

### In `sdp status`

Current implementation is too naive:

- manual JSON printing in `sdp/sdp-plugin/cmd/sdp/status_text.go`
- backlog file counting instead of real ready/blocked semantics
- primitive `NextAction` heuristic
- no stable schema for automation

### In `sdp next`

The command exists and is tested, but it is not yet the canonical UX center:

- status and next-step logic are split
- help/status/recovery paths do not share one contract surface
- there is no clearly documented machine-readable interface for external agents

### In OSS positioning

The public README already mentions `sdp status`, but it does not yet sell a strong next-step story:

- `sdp/README.md`
- `sdp/sdp-plugin/README.md`

---

## Release-Ready Definition

This feature is done only when all of the following are true:

1. `sdp status` returns a stable JSON shape backed by explicit contracts.
2. `sdp next` and `sdp status` use the same recommendation source.
3. Guidance distinguishes `ready`, `blocked`, and `recovery` states.
4. Every next-step recommendation includes rationale.
5. Failure states return actionable recovery guidance instead of generic errors.
6. Docs explain the feature in OSS-facing terms with examples.

---

## What We Explicitly Defer

### Defer 1: `sdp-evidence` packaging

Important, but not the single feature for this cycle.

Reason: it should follow as a trust-surface packaging improvement after the product UX gets sharper.

### Defer 2: OpenSpec import

Promising, but too early as the one polished release feature.

Reason: it needs contracts and next-step semantics first, otherwise import lands into a weak user experience.

### Defer 3: memory/drift storytelling

Valuable, but secondary to first-run clarity.

Reason: users need to understand what SDP wants them to do before advanced continuity features matter.

---

## Implementation Shape

### Phase A - Contracts first

Ship:

- `00-068-04` status and instructions contracts
- `00-068-05` import payload contract only if needed to avoid contract churn later

### Phase B - Canonical next-step UX

Ship:

- `00-069-04` guided next-step status and help
- `00-069-05` structured failure guidance and walkthrough

### Phase C - OSS release polish

Polish:

- README positioning in `sdp/README.md`
- CLI examples in `sdp/sdp-plugin/README.md`
- quickstart examples showing status, blocked state, and recovery flow

---

## Final Recommendation

Choose **guided next-step UX** as the one feature.

Not because it is the easiest thing to do.

Because it best combines:

- real user value
- existing implementation momentum
- manageable release scope
- strong OSS differentiation

If we ship only one thing next, it should be the feature that makes SDP feel immediately understandable.

---

## References

- `docs/architecture/REPO-BOUNDARY.md`
- `docs/plans/2026-03-08-openspec-integration-plan.md`
- `docs/workstreams/backlog/00-068-04.md`
- `docs/workstreams/backlog/00-069-04.md`
- `sdp/README.md`
- `sdp/sdp-plugin/README.md`
- `sdp/sdp-plugin/cmd/sdp/status_text.go`
- `sdp/sdp-plugin/cmd/sdp/next.go`
- `sdp/sdp-plugin/cmd/sdp/next_test.go`
