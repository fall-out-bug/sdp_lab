# Agent Catalog

Status: canonical reference

Canonical design reference:

- `docs/plans/2026-03-15-canonical-sdp-loop-and-agent-stack.md`

This document defines the default SDP agent workflow.

## Canonical SDP Loop

The default loop is:

- `vision`
- `feature`
- `workstream`
- `beads issue`
- early `draft PR`
- `execution`
- findings as `beads issue`
- `QA/UAT`
- clean `PR`

The default agent stack exists only to move work through that loop.

## Canonical Agents

### `vision`

Owns:

- project-level intent
- project map updates
- clarification of what kinds of `feature` belong in the project

Used when:

- project direction is unclear
- a new initiative changes the project map
- the user needs to shape or revise `vision`

Primary output:

- updated `vision`

### `feature`

Owns:

- one `feature`
- feature acceptance criteria
- `workstream` decomposition
- mapping from `workstream` to `beads issue`
- decision on whether a separate `plan` is needed

Used when:

- a `feature` is being created or refined
- a `feature` must be decomposed into executable `workstream`
- linked `beads issue` entries need to be prepared

Primary output:

- accepted `feature`
- linked `workstream`
- linked `beads issue` graph

### `orchestrator`

Owns:

- ready `beads issue` graph
- early `draft PR`
- dependency-aware execution order
- keeping the `PR` moving until clean

Used when:

- execution starts
- ready work must be scheduled
- findings must be routed back into execution
- merge readiness must be evaluated

Primary output:

- active branch and early `draft PR`
- updated execution state
- clean `PR` ready for human merge

### `implementer`

Owns:

- execution of one `beads issue`
- TDD where required by the `workstream`
- production of `evidence`
- `trace` updates and `drift` inputs

Used when:

- one execution unit is ready
- a finding issue is ready to fix

Primary output:

- completed change or explicit blocker
- `evidence`
- `trace` input
- `drift` input

### `reviewer`

Owns:

- engineering review
- verification of quality gates, `trace`, and `evidence`
- conversion of review, CI, and `drift` findings into `beads issue`

Used when:

- execution output is ready for review
- engineering gates need verification
- findings must re-enter the backlog

Primary output:

- pass verdict, or
- typed `beads issue` findings

### `qa`

Owns:

- `QA/UAT`
- validation of code behavior against `feature` intent
- `qa:pass` or `qa:fail`

Used when:

- engineering gates are clean
- the system needs final intent validation before merge-ready state

Primary output:

- `qa:pass` with `UAT evidence`, or
- `qa:fail` with blocking `beads issue`

## Canonical Stage Routing

| Stage | Primary agent | Required result |
|-------|---------------|-----------------|
| `vision` | `vision` | updated project map |
| `feature` shaping | `feature` | accepted `feature` |
| `workstream` + `beads issue` mapping | `feature` | executable graph |
| early `draft PR` | `orchestrator` | active branch and draft PR |
| `execution` | `implementer` | change or blocker |
| review and gates | `reviewer` | pass or typed findings |
| `QA/UAT` | `qa` | `qa:pass` or `qa:fail` |
| merge readiness | `orchestrator` | clean `PR` |

## Optional Advisors

These are not part of the default happy path.

- `oracle` - hard architecture, debugging, security, or tradeoff cases
- `reality` - repo audits and reality checks
- specialist review modes such as security, sre, devops, or ux when a feature truly needs them

Rule:

- if a role does not own a unique SDP transition, it should not be a top-level default agent

## Reduction Rules

Merge or delete when possible:

- planning personas that duplicate `feature`
- supervisor or synthesis roles that duplicate `orchestrator`
- multiple review personas as separate default agents
- market or growth personas on the default engineering path

Canonical target:

- 6 default agents
- small optional advisor bench

## Agent Quality Rule

Every top-level agent must answer three questions clearly:

- what SDP stage it owns
- what SDP entity it updates
- what artifact or verdict it must emit

If an agent cannot answer those three, it should not stay top-level.
