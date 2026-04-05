# Canonical Happy Path

Status: canonical reference

This is the shortest stable description of how SDP is supposed to work from intake to delivery.

Use this doc when the question is:

- what is the default SDP product path?
- how do board, Beads, skills, CLI, PR, and QA/UAT fit together?
- what is the difference between local onboarding and full operator mode?

For rationale and rollout details, see [../plans/2026-04-05-canonical-sdp-happy-path-consistency.md](../plans/2026-04-05-canonical-sdp-happy-path-consistency.md).

## Product Promise

SDP should support one coherent story:

1. work enters a board-backed queue
2. SDP shapes it with progressive disclosure
3. agents execute through a visible workflow
4. findings loop back into the same queue
5. the feature reaches a clean delivery path with proof

Everything else is a control surface or a disclosure layer for that same story.

## Two Modes

### Local Mode

Use this when a user wants SDP inside one repo and is not yet running a shared queue.

Default path:

1. install SDP
2. `sdp init`
3. `sdp doctor`
4. shape work locally
5. execute and verify locally

Characteristics:

- `Beads` is optional
- there may be no visible board UI
- skills and CLI are local control surfaces

### Operator Mode

Use this when SDP is presented as a queue or board system that carries work to deploy.

Default path:

1. task enters board-backed queue
2. clarification and shaping
3. executable graph becomes ready
4. early draft PR exists
5. agents execute
6. findings re-enter the queue
7. QA/UAT passes
8. human approves merge or delivery step

Characteristics:

- `Beads` is required
- the board is a projection over Beads-backed operational truth
- PR, findings, QA/UAT, and delivery are first-class stages

## Source of Truth

| Layer | Canonical truth |
|---|---|
| queue state | `Beads` |
| semantic intent | `feature` and `workstream` artifacts |
| integration state | branch + early `draft PR` |
| proof | `evidence`, `trace`, `drift`, `QA/UAT` artifacts |
| board/status views | derived projections |
| skills and CLI | control surfaces |

Rule:

- the board is not an independent store
- the board is a visibility surface over Beads-backed truth

Reference:

- [../decisions/ADR-BEADS-FIRST-SOURCE-OF-TRUTH.md](../decisions/ADR-BEADS-FIRST-SOURCE-OF-TRUTH.md)

## Canonical Stages

| Stage | Result |
|---|---|
| install and bootstrap | repo is ready to use SDP |
| intake | task exists in queue and is visible |
| clarification and shaping | accepted `feature`, `workstream`, and linked Beads graph |
| execution setup | branch and early `draft PR` exist |
| execution | change and execution evidence exist |
| findings loop | review, CI, drift, and QA findings re-enter queue |
| QA/UAT | `qa:pass` or `qa:fail` is explicit |
| delivery | clean PR plus approval or delivery proof |

## Control Surfaces

### Board and queue

Use for:

- visibility
- intake
- attention routing
- status and blockers

### Skills

Use for:

- guided journeys
- clarification
- progressive disclosure
- agent-driven execution

### CLI

Use for:

- explicit terminal control
- scripting
- machine-readable state
- deterministic automation

Rule:

- skills and CLI are not separate workflows
- they are two ways to drive the same stage model

## Current Canonical References

| Need | Doc |
|---|---|
| repo orientation | [project-map.md](project-map.md) |
| happy path overview | [canonical-happy-path.md](canonical-happy-path.md) |
| operator execution loop | [../SDP_OPERATOR_WORKFLOW.md](../SDP_OPERATOR_WORKFLOW.md) |
| agent ownership | [agent-catalog.md](agent-catalog.md) |
| skill surface | [skills.md](skills.md) |
| design rationale | [../plans/2026-04-05-canonical-sdp-happy-path-consistency.md](../plans/2026-04-05-canonical-sdp-happy-path-consistency.md) |

## What This Doc Does Not Do

- it does not replace public onboarding in `sdp/docs/QUICKSTART.md`
- it does not define exact CLI syntax
- it does not replace historical design rationale

It defines the stable internal model that other docs must follow.
