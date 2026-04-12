# SDP Work Atomicity Normalization

**Date:** 2026-04-12
**Status:** Proposed
**Issue:** `sdplab-nx2`
**Owner:** Андрей
**Scope:** SDP planning and execution model

---

## Problem

SDP currently uses the same words for different things.

That drift is no longer cosmetic. It changes how work is shaped, assigned, and
verified.

Current repo statements conflict:

| Source | Current statement | Tension introduced |
|---|---|---|
| `docs/TERMS.md` | `workstream` is an atomic task | implies direct executability |
| `docs/reference/design-spec.md` | large workstreams must split | implies workstream atomicity is mandatory |
| `AGENTS.md` | one workstream can accumulate more than one Beads issue over time | implies workstream is not the live atomic unit |
| `AGENTS.md` / `docs/reference/skills.md` | execution walks the ready Beads issue graph | implies Beads issue is the live execution atom |
| `docs/archive/plans/2026-03-15-canonical-sdp-loop-and-agent-stack.md` | `workstream` owns contract, `beads issue` owns execution | implies two different layers already exist |

Result:

1. the same `workstream` is treated as atomic in some docs and non-atomic in others;
2. `beads issue` is sometimes a mirror of a workstream and sometimes the real
   execution truth;
3. findings and remediation work do not have a clear rule for when they reopen
   the current slice versus create a new slice.

This is not a wording problem. It is a control-flow problem.

---

## What Must Stay True

Any normalization is invalid if it breaks these constraints:

1. `Feature` remains the user-facing outcome container.
2. `Beads` remains the live execution queue and dependency graph.
3. `workstream` remains a first-class SDP artifact, not a disposable note.
4. review, CI, drift, and `QA/UAT` findings must re-enter the same system.
5. the model must stay simple enough for agents and humans to use without
   interpreting philosophy every session.

---

## Atomicity, Normalized

SDP needs one exact definition:

> A workstream is atomic only when it is a leaf workstream: one coherent change
> slice with one scope boundary, one acceptance contract, and one primary
> executable path.

Corollary:

- not every `workstream` is atomic;
- every executable `workstream` must be atomic;
- a non-atomic `workstream` is a parent container and must not be dispatched
  directly.

This keeps `workstream` as the core planning object without pretending every WS
file is immediately executable.

---

## Options

### Option A: Restore strict atomic workstream, treat Beads as execution mirror

Model:

- `Feature -> Workstream`
- every `workstream` is atomic and directly executable
- `beads issue` is just the live queue representation of that same unit

Pros:

- simplest mental model on paper
- preserves old wording almost unchanged
- easy to explain

Cons:

- does not fit current findings loop well
- forces awkward handling for remediation and discovered work
- conflicts with the reality that one workstream often spawns several execution
  episodes over time
- pushes complexity into undocumented exceptions instead of the model

Verdict:

- attractive as doctrine
- weak as operating model

### Option B: Recursive workstream, atomicity only at leaf level

Model:

- `Feature -> Workstream tree`
- a `workstream` can be either `parent` or `leaf`
- only `leaf workstream` is executable
- live Beads issues point to leaf workstreams

Pros:

- matches how decomposition actually happens
- preserves `workstream` as the core SDP unit
- lets Beads stay the execution graph without replacing workstreams
- gives a clean place for remediation and discovered child slices

Cons:

- docs, validators, and templates must change
- parent vs leaf semantics must be explicit everywhere
- old shorthand "workstream = atomic task" must be retired

Verdict:

- best fit for current SDP reality
- cheapest honest normalization

### Option C: Make Beads issue the only atomic work item, keep workstream as contract only

Model:

- `Feature -> Workstream`
- `Beads issue` is the only executable atom
- `workstream` is a contract/spec layer that may map to many issues

Pros:

- aligns with the current execution engine
- clean for queueing and findings
- technically close to what some docs already describe

Cons:

- demotes workstream from "unit of work" to "planning annotation"
- weakens SDP's own language and templates
- invites Beads-first behavior and more drift

Verdict:

- coherent, but undermines the intended primacy of workstreams

---

## Recommendation

Adopt **Option B: recursive workstream with leaf-only atomicity**.

This is the only option that keeps all three of these true at once:

1. `workstream` remains a first-class SDP unit;
2. `Beads` remains the live execution graph;
3. decomposition and remediation stop living in undocumented exceptions.

This is also the closest match to the direction already implied by the repo:

- `Feature` owns outcome
- `workstream` owns contract
- `beads issue` owns live execution state

The missing piece is not a new entity. The missing piece is a normalized rule
for when a workstream is executable.

---

## Normalized Model

### Entity rules

| Entity | Owns | Must not own |
|---|---|---|
| `Feature` | outcome, acceptance intent, UAT intent | execution graph |
| `Parent workstream` | decomposition boundary, roll-up acceptance, child ordering | direct execution |
| `Leaf workstream` | atomic contract for one change slice | multi-slice planning |
| `Beads issue` | live execution state, dependencies, findings, blocking status | feature meaning |

### Workstream states

Every workstream must declare one of:

- `parent`
- `leaf`

Rules:

1. `parent` workstream cannot be dispatched directly.
2. `leaf` workstream cannot have planned child workstreams.
3. if a `leaf` needs planned decomposition, it was mis-shaped and must be
   converted into `parent`.
4. parent completion is derived from child completion unless explicitly
   overridden with rationale.

### Mapping rules

Live mapping:

- one primary executable `beads issue` maps to exactly one `leaf workstream`

Historical reality:

- over time, the same `leaf workstream` may accumulate several linked Beads
  issues due to retries, findings, or reopened work

Constraint:

- there must never be multiple simultaneous primary execution issues for the
  same leaf workstream

This keeps the live model simple without denying history.

---

## Findings And Remediation Rules

This is the part SDP currently leaves fuzzy.

### Keep the finding inside the same leaf workstream when:

- the original goal is unchanged
- the scope boundary is unchanged
- the failure is execution quality, not problem re-shaping
- examples: test failure, bug in implementation, review fix, CI breakage

Action:

- create a linked finding `beads issue`
- keep the same leaf workstream as the owning contract

### Create a new child or sibling workstream when:

- the fix changes the intended scope
- the work introduces a new acceptance contract
- the original slice was too large or incorrectly shaped
- the discovered work can be reviewed and closed independently

Action:

- create a new `leaf workstream`
- link a new primary Beads issue to that new leaf

### Re-shape the current workstream into a parent when:

- decomposition is not incidental but planned
- multiple new slices are now clearly required
- the current leaf can no longer honestly claim one atomic contract

Action:

- convert current WS to `parent`
- create child leaf workstreams
- move live execution to children

This is the guardrail that prevents "one WS, five hidden tasks" drift.

---

## Operational Invariants

If this model is adopted, the repo should enforce these invariants:

1. every executable queue item links to exactly one `leaf workstream`
2. no ready queue item may point to a `parent workstream`
3. `docs/workstreams/backlog/*.md` must declare `ws_kind: parent|leaf`
4. `.beads-sdp-mapping.jsonl` may only map a primary execution issue to a leaf
   workstream
5. a workstream validator must fail if a `leaf` lists planned child workstreams
6. a parent workstream must have roll-up acceptance, not implementation-level
   file scope pretending to be atomic

---

## Why This Is Better For UX And DX

### UX

- users can finally understand why some work items are directly executable and
  others are just decomposition containers
- findings stop feeling arbitrary
- remediation becomes explainable instead of magical

### DX

- planners stop encoding hidden subtasks in prose
- agents know whether they may execute a workstream directly
- validators can catch bad shaping before execution starts

The key point: this reduces ambiguity at dispatch time.

---

## External Sanity Check

Other systems converge on similar separations even if they use different words:

- `GitHub` and `Linear` allow recursive issues/sub-issues
- `Airflow` and `Prefect` separate workflow container from task atom
- `Paperclip` keeps goals and governance above task checkout
- `DeerFlow` keeps runtime decomposition inside the harness
- `Gastown` separates issue, convoy, and formula/workflow layers

SDP should not try to be the one system where one noun means contract,
container, queue item, and remediation target at the same time.

---

## Required Repo Changes If Accepted

### Docs

- rewrite `docs/TERMS.md` workstream definition
- update `AGENTS.md` to distinguish parent vs leaf workstream
- update `docs/reference/design-spec.md` decomposition rules
- update `docs/reference/skills.md` and agent prompts to forbid dispatching
  parent workstreams
- add one short canonical reference page for normalized atomicity

### Tooling

- extend workstream frontmatter with `ws_kind`
- update validators to enforce leaf-only executability
- update mapping checks so primary execution issues only point to leaf WS
- update planning tools to split oversized leaf WS into child WS instead of
  hiding subtasks in Beads only

### Process

- review/CI/drift findings stay in Beads unless they clearly create a new slice
- planned decomposition must create child workstreams, not invisible execution
  atoms

---

## Questions For LLM Council

1. Is leaf-only atomicity the right normalization, or does it still leave too
   much ambiguity between `parent workstream` and `feature`?
2. Is the proposed rule for findings versus new child workstreams strong enough,
   or does it need stricter triggers?
3. Should a parent workstream ever carry direct evidence of completion, or only
   roll-up evidence from children?
4. Is the live rule "one primary execution issue per leaf workstream" strong
   enough to prevent double-dispatch and hidden parallelism?
5. Which parts of the current repo will break first if this normalization is
   adopted?

---

## Decision Wanted

The Decision Owner should choose one of:

1. accept Option B as the new canonical model
2. reject Option B and restore strict atomic workstream doctrine
3. reject workstream primacy and make Beads issue the only execution atom
4. defer and keep current mixed model with explicit risk acceptance
