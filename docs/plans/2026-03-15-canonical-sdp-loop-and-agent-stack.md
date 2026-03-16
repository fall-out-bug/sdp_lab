# Canonical SDP Loop and Agent Stack

> **Status:** Draft design
> **Date:** 2026-03-15
> **Goal:** Define one canonical SDP path from `vision` and `feature` to clean `PR`, then reduce the agent/skill zoo around that path.

---

## 1. Problem Statement

Today SDP has most of the right pieces:

- `vision`
- `feature`
- `workstream`
- `beads issue`
- `execution`
- `evidence`
- `trace`
- `drift`
- `PR`

The problem is not missing concepts. The problem is weak control flow.

Observed failure mode:

- too many sources of truth
- too many optional paths
- `OmO` behaves like a free-form orchestrator instead of an SDP runtime
- `PR` appears too late
- review findings do not cleanly re-enter the same execution loop
- `QA/UAT` is implied, but not a first-class stage

Result: SDP feels like protocol plus artifacts, not AI for SDLC.

---

## 2. Canonical SDP Entities

Use one ownership model and stop letting entities overlap.

| Entity | Owns | Must not own |
|--------|------|--------------|
| `vision` | project map, feature map, top-level intent | execution order, review findings |
| `feature` | user-facing outcome, acceptance intent, UAT intent | low-level execution graph |
| `workstream` | execution contract for one change slice | scheduling, review queue |
| `beads issue` | execution unit, dependency graph, findings queue | product meaning for whole feature |
| `PR` | integration surface, review surface, merge surface | project planning |
| `evidence` | proof of completion for execution and review gates | backlog state |
| `trace` | linkage across entities and artifacts | acceptance itself |
| `drift` | verdict on divergence from feature/workstream intent | scheduling |

Rule: `feature` owns meaning, `workstream` owns contract, `beads issue` owns execution, `PR` owns integration.

---

## 3. Canonical Happy Path

### 3.1 `vision`

`vision` is interactive. SDP works with the user until the project map is clear enough to support `feature` creation.

`vision` must answer:

- what the project is trying to achieve
- what kinds of `feature` belong in the project
- what success looks like at project level
- which constraints are stable enough to shape future work

### 3.2 `feature`

`feature` is also interactive. SDP works with the user until the feature has enough shape to be executed.

Every `feature` must define:

- expected user-visible outcome
- acceptance criteria
- explicit non-goals or out-of-scope notes
- expected `QA/UAT` path

If the system cannot state acceptance criteria, the `feature` is not ready.

### 3.3 `workstream`

`feature` is decomposed into `workstream` files.

Every `workstream` must define:

- goal
- scope boundary
- acceptance criteria
- expected `evidence`
- `drift` notes for changes that would require feature re-alignment

`workstream` is a contract, not a queue.

### 3.4 `beads issue` mapping

Every executable unit is a `beads issue` linked back to one `feature` and one `workstream`.

`beads issue` owns:

- dependency edges
- ready/blocked status
- implementation or findings source
- blocking vs non-blocking priority

If the dependency graph in `beads` is sufficient, a separate `plan` is optional.

`plan` is only needed for:

- ambiguous execution
- risky migration
- cross-cutting change
- non-trivial external integration
- complicated `QA/UAT`

### 3.5 Early `draft PR`

The `PR` is created early.

Rule:

- create a `draft PR` at the start of the first blocking `workstream`
- if no blocking `workstream` exists, create it at the first meaningful code or doc change tied to the `feature`

Why:

- integration surface exists from the start
- `trace` starts early instead of being reconstructed later
- review surface appears before the system thinks it is done
- TDD becomes visible: first the branch is red, then green, then cleaned up

### 3.6 `execution`

The orchestrator walks the ready `beads issue` graph.

For each ready issue it must:

1. confirm linked `feature` and `workstream`
2. execute the work
3. collect `evidence`
4. update `trace`
5. emit `drift` verdict
6. move to next ready issue or stop on blocker

Allowed outcomes for one execution step:

- `done` with `evidence`
- `blocked` with exact blocker
- `needs_clarification` with one exact question

### 3.7 Review cycle

All review outputs re-enter the same loop as `beads issue`.

This includes:

- code review comments
- CI failures
- `drift` findings
- `PR` gate failures

No new top-level entity is required.

Instead, review-derived `beads issue` entries need stronger metadata:

- `source = review | ci | drift | qa`
- `feature_id`
- `workstream_id`
- `pr_url` or review artifact link
- `blocking = true|false`
- `evidence_ref` when available

### 3.8 `QA/UAT`

After the `PR` is clean on engineering gates, SDP enters `QA/UAT`.

`QA/UAT` takes:

- `vision`
- `feature`
- linked `workstream` acceptance criteria
- current `PR`
- current `evidence`

`QA/UAT` returns one of two verdicts:

- `qa:pass` with `UAT evidence`
- `qa:fail` with new blocking `beads issue`

After `qa:fail`, the orchestrator goes back to the same graph.
After `qa:pass`, the `PR` is ready for human merge.

### 3.9 Merge

Human still decides to merge.

But for SDP, the canonical terminal state for change work is:

- clean `PR`
- passed engineering gates
- `drift` verdict recorded
- `QA/UAT` verdict recorded

That is the real "done" state.

---

## 4. Required Artifacts Per Stage

| Stage | Required artifact | Required verdict |
|-------|-------------------|------------------|
| `vision` | updated project map | project intent is clear enough for `feature` work |
| `feature` | feature definition | acceptance is clear enough to decompose |
| `workstream` | workstream file | contract is clear enough to execute |
| `beads issue` | linked issue with dependencies | ready or blocked |
| `execution` | code/doc/runtime change + `evidence` | done, blocked, or needs clarification |
| `PR` | early `draft PR`, later reviewable PR | gate pass/fail |
| review | findings as `beads issue` | blocking/non-blocking |
| `QA/UAT` | `UAT evidence` | `qa:pass` or `qa:fail` |
| completion | merged PR or human-approved clean PR | feature complete or follow-up required |

---

## 5. Trace and Drift

### 5.1 `trace`

`trace` must be derived from the loop, not maintained as free-form prose.

Minimum chain:

- `vision -> feature`
- `feature -> workstream`
- `workstream -> beads issue`
- `beads issue -> branch`
- `branch -> PR`
- `beads issue -> evidence`
- `PR -> review findings`
- `QA/UAT -> evidence`

### 5.2 `drift`

`drift` is checked at the end of every execution cycle and again before `QA/UAT`.

Allowed verdicts:

- `no_drift`
- `accepted_drift`
- `unacceptable_drift`

If `unacceptable_drift`, SDP must create blocking `beads issue` entries or force `feature/workstream` alignment before continuing.

---

## 6. Canonical Agent Stack

The current agent zoo is larger than the canonical SDP loop needs.

Keep a small default stack and move the rest to optional advisors.

### 6.1 Canonical agents

#### 1. `vision` agent

Purpose:

- work with the user on project-level intent
- update project map
- identify or refine `feature` candidates

#### 2. `feature` agent

Purpose:

- work with the user on one `feature`
- generate or update `workstream` files
- map `workstream` to `beads issue`
- decide whether a separate `plan` is needed

This absorbs most of what a standalone design/planning agent used to do on the happy path.

#### 3. `orchestrator` agent

Purpose:

- own the ready `beads issue` graph
- open the early `draft PR`
- dispatch execution in dependency order
- keep the `PR` moving until clean

This is the runtime heart of SDP.

#### 4. `implementer` agent

Purpose:

- execute one `beads issue`
- follow TDD when code changes are involved
- return `evidence`, `trace` updates, and `drift` verdict inputs

#### 5. `reviewer` agent

Purpose:

- validate code quality, traceability, evidence, and gate status
- convert findings into `beads issue`
- keep the review loop inside SDP instead of in ad hoc comments

#### 6. `qa` agent

Purpose:

- drive `QA/UAT`
- validate behavior against `feature` intent
- produce `qa:pass` or `qa:fail`

### 6.2 Optional agents

These are no longer part of the default happy path. They are exception-path advisors.

- `oracle` for hard architecture/debug/security tradeoffs
- `reality` for codebase audits and drift reality checks
- specialist personas such as security, sre, devops, ux when a feature truly needs them

### 6.3 What to merge or delete

Default rule: if an agent does not own a unique SDP transition, it should not be a top-level agent.

That means:

- market analyst, growth strategist, business analyst, risk analyst should not be separate default agents
- planner should not be separate from `feature` unless the work is unusually risky or ambiguous
- multiple review personas should be modes of `reviewer`, not independent default agents
- synthesis/supervisor agents should disappear from the happy path; the orchestrator already owns flow control

Canonical target: 6 default agents plus a small exception bench.

---

## 7. Canonical Skill Surface

Skills should mirror the canonical SDP loop, not the historical prompt tree.

### 7.1 User-facing skills

Keep these as the main public surface:

- `@vision` - define or revise project direction
- `@feature` - define one feature, produce/update workstreams and `beads` mapping
- `@oneshot` - execute feature through the `beads` graph and keep the `PR` moving
- `@review` - run engineering review and convert findings into `beads issue`
- `@qa` - run `QA/UAT` against the current `PR`
- `@deploy` - optional release/deploy path after merge readiness

### 7.2 Internal or conditional skills

Keep these, but stop treating them as core user entry points:

- `@build` - execute one workstream or one `beads issue`
- `@debug` - systematic failure analysis
- `@issue` - classify incoming bug/failure work
- `@reality` / `@reality-check` - reality and drift checks

### 7.3 Skills to absorb or demote

- `@idea` becomes internal to `@vision` and `@feature`
- `@design` becomes internal to `@feature` unless manual decomposition is explicitly requested
- `@plan` stays implicit and conditional, not a mandatory public step
- skills that only duplicate docs, old branch models, or phantom commands should be deleted

### 7.4 Skill quality rule

Every surviving skill must answer three questions clearly:

- when is it the right entry point
- what SDP entity does it update
- what artifact or verdict does it emit

If a skill cannot answer those three, it is not stable enough to keep.

### 7.5 Canonical stage routing

The default zoo behavior should be deterministic.

| Stage | Primary agent | Optional advisors | Primary skill | `sdp-process` support | Required output |
|-------|---------------|-------------------|---------------|-----------------------|-----------------|
| `vision` | `vision` | `reality`, `oracle` | `@vision` | `vision.get`, `vision.update` | updated project map |
| `feature` shaping | `feature` | `oracle`, `reality` | `@feature` | `feature.get`, `feature.update` | accepted feature definition |
| `workstream` + `beads issue` mapping | `feature` | `reality` | `@feature` with internal design path | `workstream.list`, `workstream.get`, `workstream.upsert`, `beads.link_workstream` | linked workstreams and dependency graph |
| branch + early `draft PR` | `orchestrator` | none | `@oneshot` | `pr.ensure_draft`, `trace.render` | branch plus early draft PR |
| `execution` | `implementer` | `oracle` on exception path | `@build` or `@oneshot` | `evidence.template`, `trace.render`, `drift.check` | completed change or explicit blocker |
| review and engineering gates | `reviewer` | security/sre/devops modes inside review | `@review` | `beads.create_finding`, `trace.render`, `drift.check` | findings as typed `beads issue` or pass verdict |
| `QA/UAT` | `qa` | `oracle` when intent is disputed | `@qa` | `qa.checklist`, `qa.record_verdict` | `qa:pass` or `qa:fail` with evidence |
| merge readiness | `orchestrator` | none | `@oneshot` or explicit closeout path | `trace.render`, `drift.check` | clean PR ready for human merge |

This routing keeps top-level behavior boring:

- `vision` and `feature` work with the user
- `orchestrator` owns the feature `PR`
- `implementer` executes one unit of work
- `reviewer` turns findings into `beads issue`
- `qa` owns `QA/UAT`

---

## 8. Process-Support MCP

There is room for one process-support MCP, but it must support SDP state, not replace it.

### 8.1 Required internal MCP: `sdp-process`

Purpose: expose structured SDP state and operations so agents stop scraping docs and guessing process.

Suggested operations:

- `vision.get`
- `vision.update`
- `feature.get`
- `feature.update`
- `workstream.list`
- `workstream.get`
- `workstream.upsert`
- `beads.ready`
- `beads.link_workstream`
- `beads.create_finding`
- `pr.ensure_draft`
- `trace.render`
- `drift.check`
- `evidence.template`
- `qa.checklist`
- `qa.record_verdict`

Expected `beads.create_finding` behavior:

- map typed findings into Beads issue fields, labels, and canonical description
- attach source labels such as `review-finding`, `ci-finding`, `drift-finding`, or `qa-finding`
- attach linked `feature` and `workstream`
- attach `blocking` or `non-blocking`
- keep notes supplemental rather than primary state

This should behave more like LSP for SDP state than like a giant process encyclopedia.

### 8.2 Optional external MCP: `process-reference`

Purpose: provide curated guidance for practices such as:

- TDD
- review checklists
- QA/UAT patterns
- release readiness
- incident handling

But this MCP must be advisory only.

Source of truth remains:

- `vision`
- `feature`
- `workstream`
- `beads issue`
- `PR`
- `evidence`
- `trace`
- `drift`

If external process guidance can override SDP state, the system will drift again.

---

## 9. Required Changes

### 9.1 SDP workflow changes

- make early `draft PR` part of the canonical loop
- require review findings to re-enter as typed `beads issue`
- add first-class `QA/UAT` stage after engineering gates
- keep `plan` optional when the `beads` graph is already sufficient

### 9.2 Agent changes

- reduce default agents to the canonical 6
- move most other agents to optional advisor roles
- stop using separate agents for distinctions that belong inside one agent mode

### 9.3 Skill changes

- make `@vision`, `@feature`, `@oneshot`, `@review`, `@qa` the canonical happy-path surface
- demote `@idea`, `@design`, and explicit planning to internal or advanced usage
- delete or rewrite skills that still carry phantom commands, wrong language assumptions, or duplicated workflow logic

### 9.4 Runtime changes

- make the orchestrator own the feature `PR`
- make the orchestrator consume only ready `beads issue`
- require every execution step to emit `evidence`, `trace`, and `drift` inputs

---

## 10. Where the Current Design Went Wrong

This is the author fault line in blunt form.

- too many concepts were allowed to become top-level operating surfaces
- too many agents were created for distinctions that should have stayed as modes or checklists
- skills multiplied faster than the canonical loop got stronger
- process knowledge leaked into prompts, docs, runbooks, and role text instead of one stateful runtime
- `PR` appeared too late, which weakened review, trace, and TDD visibility
- `QA/UAT` was treated like a human afterthought instead of a first-class SDP stage

The fix is not more intelligence.
The fix is one boring loop that closes work reliably.

---

## 11. Canonical Summary

The intended SDP product story is now:

- user shapes `vision`
- user shapes `feature`
- SDP writes `workstream` contracts and maps them to `beads issue`
- orchestrator opens early `draft PR`
- orchestrator executes ready `beads issue`
- review, CI, drift, and QA findings go back into `beads issue`
- `QA/UAT` validates behavior against feature intent
- user receives a clean `PR` to merge

That is the canonical loop.
