# Canonical SDP Rollout Plan

> **Status:** Draft plan
> **Date:** 2026-03-15
> **Goal:** Move SDP from the current workflow and agent/skill zoo to the canonical loop without a greenfield rewrite.

Related:

- [2026-03-15-canonical-sdp-loop-and-agent-stack.md](2026-03-15-canonical-sdp-loop-and-agent-stack.md)
- [../../AGENTS.md](../../AGENTS.md)

---

## 1. Rollout Principles

- keep existing work moving while the loop is tightened
- prefer migration and aliasing over abrupt deletion
- make the happy path stronger before deleting escape hatches
- move state into SDP runtime contracts, not prompt prose
- measure success by clean `PR` throughput, not by component count

---

## 2. Target End State

Canonical happy path:

- user shapes `vision`
- user shapes `feature`
- SDP writes `workstream` contracts and linked `beads issue`
- orchestrator opens early `draft PR`
- orchestrator executes ready `beads issue`
- review, CI, `drift`, and `QA/UAT` findings re-enter as typed `beads issue`
- `QA/UAT` returns `qa:pass` or `qa:fail`
- user receives a clean `PR` to merge

Canonical default stack:

- agents: `vision`, `feature`, `orchestrator`, `implementer`, `reviewer`, `qa`
- public skills: `@vision`, `@feature`, `@oneshot`, `@review`, `@qa`, `@deploy`
- process support: internal `sdp-process` MCP

---

## 3. Success Criteria

The rollout is working when all of the following are true:

- a `feature` can be executed through one canonical path without requiring prompt archaeology
- the first blocking `workstream` or first meaningful change opens a `draft PR`
- review, CI, `drift`, and `QA/UAT` findings all re-enter as typed `beads issue`
- `QA/UAT` verdict exists before SDP calls a change ready for merge
- default top-level agents are reduced to the canonical 6
- public skill surface is reduced to the canonical happy path
- operators can answer "what is next" and "why is this blocked" from SDP state, not from memory

Recommended metrics:

- time from first `beads issue` claim to `draft PR`
- time from `draft PR` to clean `PR`
- blocking findings per `feature`
- reopen rate after `qa:fail`
- token spend per clean `PR`

---

## 4. Phase Plan

### Phase 0: Terminology and docs alignment

**Goal:** stop semantic drift before changing runtime behavior.

Changes:

- align `AGENTS.md` to the canonical loop
- align canonical design docs and runbooks to SDP terms only
- document early `draft PR`, findings-as-`beads issue`, and `QA/UAT`
- identify docs that still describe the old loop

Primary files:

- `AGENTS.md`
- `docs/SDP_OPERATOR_WORKFLOW.md`
- `docs/REAL_FEATURE_TO_PR_RUNBOOK.md`
- `docs/reference/agent-catalog.md`
- `docs/reference/skills.md`

Exit criteria:

- one documented canonical loop exists
- no primary doc contradicts the canonical loop on `PR`, review findings, or `QA/UAT`

Risks:

- docs move faster than runtime

Rollback:

- keep old runbooks as legacy references until runtime catches up

### Phase 1: Canonical public surface

**Goal:** make the happy path obvious.

Changes:

- promote `@vision`, `@feature`, `@oneshot`, `@review`, `@qa`, `@deploy`
- demote `@idea`, `@design`, and explicit `plan` to internal or advanced paths
- publish one decision tree for when to use each public skill

Exit criteria:

- operators can describe the default path without referencing internal skills
- docs stop advertising redundant public entry points

Risks:

- old habits keep sending users through the old surface

Rollback:

- keep legacy skills as aliases while the docs and prompts are cut over

### Phase 2: Typed `beads issue` findings

**Goal:** put review, CI, `drift`, and `QA/UAT` findings back into the same execution graph.

Changes:

- define typed finding metadata in `beads issue`
- require `source`, `feature`, `workstream`, `blocking`, and `PR`/artifact references
- update `beads` integration docs and adapters
- make reviewer and QA paths emit findings as `beads issue`

Primary contract:

- `docs/protocol/BEADS_FINDINGS_CONTRACT.md`

Exit criteria:

- no blocking review or QA result exists outside `beads issue`
- the orchestrator can resume work from findings without manual translation

Risks:

- historical issues without metadata remain in the system

Rollback:

- support mixed mode: legacy issue entries plus typed entries, with typed entries required for new work

### Phase 3: Early `draft PR` and orchestrator PR ownership

**Goal:** make `PR` part of the runtime from the start of execution.

Changes:

- codify early `draft PR` creation at first blocking `workstream` or first meaningful change
- make the orchestrator own the feature `PR`
- attach `trace` and `evidence` updates to the active `PR`
- update `REAL_FEATURE_TO_PR_RUNBOOK` and related scripts as needed

Exit criteria:

- new feature execution creates `draft PR` early by default
- operators no longer wait until the end to see review and integration state

Risks:

- current scripts may assume publish happens only near completion

Rollback:

- allow a temporary compatibility mode where the orchestrator can still publish late, but warn loudly in docs and status output

### Phase 4: `QA/UAT` as a first-class stage

**Goal:** stop treating `QA/UAT` as a manual afterthought.

Changes:

- define `qa:pass` and `qa:fail` as explicit SDP verdicts
- define required `UAT evidence`
- make `qa:fail` generate blocking `beads issue`
- update runbooks and gates to require `QA/UAT` before merge-ready state

Exit criteria:

- clean `PR` is not called ready until `QA/UAT` verdict exists
- `qa:fail` paths are visible in the same backlog and state model as implementation work

Risks:

- teams may treat QA as slow extra bureaucracy instead of final intent validation

Rollback:

- allow `QA/UAT` to start as a documented manual gate with standardized evidence before automating more of it

### Phase 5: Canonical agent and skill reduction

**Goal:** reduce the zoo only after the happy path is strong enough.

Changes:

- reduce default agents to `vision`, `feature`, `orchestrator`, `implementer`, `reviewer`, `qa`
- move specialist roles to optional advisor bench
- merge redundant skill responsibilities into the canonical public surface
- delete or rewrite skills with phantom commands, wrong language assumptions, or duplicate flow logic

Exit criteria:

- default docs, prompts, and workflows refer to the canonical 6 agents
- public skill surface is the canonical happy path
- non-canonical agents are clearly marked as optional or legacy

Risks:

- deleting too early can strand useful but undocumented workflows

Rollback:

- deprecate before deleting; keep compatibility shims for one cycle

### Phase 6: `sdp-process` MCP

**Goal:** stop making agents infer process from markdown and prompt bloat.

Changes:

- define a read-first `sdp-process` MCP over SDP state
- expose structured operations for `vision`, `feature`, `workstream`, `beads issue`, `PR`, `trace`, `drift`, `evidence`, and `QA/UAT`
- start with read and checklist operations, then add safe write operations

Suggested first operations:

- `vision.get`
- `feature.get`
- `workstream.list`
- `beads.ready`
- `pr.ensure_draft`
- `trace.render`
- `drift.check`
- `evidence.template`
- `qa.checklist`

Later write operations:

- `vision.update`
- `feature.update`
- `workstream.upsert`
- `beads.link_workstream`
- `beads.create_finding`
- `qa.record_verdict`

Exit criteria:

- agents can answer current state and next action from MCP-backed SDP state
- prompt instructions shrink because process lookup is structured

Risks:

- MCP becomes a second source of truth

Rollback:

- keep MCP as a thin state access layer over existing SDP artifacts; do not let it invent state of its own

---

## 5. Migration Order

Recommended order:

1. docs alignment
2. canonical public surface
3. typed `beads issue` findings
4. early `draft PR`
5. `QA/UAT` stage
6. zoo reduction
7. `sdp-process` MCP

Reasoning:

- docs and surface must stabilize before deleting anything
- findings and PR ownership tighten the loop before agent reduction
- `QA/UAT` must exist before the system can call a `PR` truly clean
- MCP should land after the state model is stable enough to expose cleanly

---

## 6. What Not To Do

- do not start with a greenfield rewrite
- do not add more top-level agents before the canonical 6 are stable
- do not make MCP a competing state store
- do not delete legacy skills before the happy path is visibly better
- do not automate `QA/UAT` into nonsense; keep it grounded in feature intent and evidence

---

## 7. First Concrete Workstreams

The first implementation slice should produce visible closure, not more architecture paper.

Recommended first workstreams:

1. update core docs and references to the canonical loop
2. define typed finding fields for `beads issue`
3. update orchestrator flow for early `draft PR`
4. define `QA/UAT` verdict contract and evidence shape
5. rewrite agent and skill references to the canonical surface

That sequence tightens the loop before it starts deleting the zoo.

---

## 8. Rollout Summary

SDP should not become "more agentic."

It should become more deterministic:

- one canonical path
- one default agent stack
- one public skill surface
- one findings loop through `beads issue`
- one `PR` that appears early and stays alive
- one `QA/UAT` verdict before merge-ready state

That is how the zoo becomes an SDLC machine instead of a collection of clever parts.
