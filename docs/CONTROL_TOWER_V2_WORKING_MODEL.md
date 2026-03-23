# Control Tower V2 — Working Model

Status: working model
Date: 2026-03-23
Scope: observable control tower joining board, orchestrator, trace, and friction visibility

## Purpose

SDP control tower should not stop at:
- a board that shows statuses
- an orchestrator that silently mutates state
- isolated execution traces

The real target is a single observable control surface where a colleague can:
1. drop work onto the board
2. see how the orchestrator matures and routes it
3. get asked only for meaningful clarification/approval
4. watch execution move forward
5. inspect the full trace from intake to deployment
6. see where agents and delivery flow keep generating friction

That is Control Tower V2.

---

## Product promise

A colleague should be able to say:

> “I threw a task onto the board. The system shaped it, asked me only when needed, showed me where it is, and I can trace everything from the original ticket to deploy — including where the agents screwed up.”

If SDP cannot do that, the tower is still incomplete.

---

## V2 thesis

Control tower UX is not just UI polish.
It is the combination of:
- **board semantics**
- **orchestrator observability**
- **end-to-end trace**
- **friction telemetry**

These four together create trust.

---

## 1. Board semantics

The board is not a decorative kanban.
It is the primary visibility surface for the control object.

### A board card must show
At minimum:
- current state
- why it is in that state
- who/what is expected next
- recommended next action
- execution linkage
- latest result summary
- visible friction markers

### A board card is not
- not just a title in a column
- not just a YAML projection
- not the native execution workbench

### Board state buckets
Minimum useful buckets:
- inbox
- clarifying
- needs_input
- ready
- executing
- reviewing
- blocked
- done
- deployed
- rolled_back

Even if some of these are implemented later, V2 should model them explicitly.

### Board summary should answer
- what needs attention now?
- what is waiting on a human?
- what is blocked?
- what is ready to move?
- what is executing?
- what was recently completed or deployed?

---

## 2. Orchestrator observability

The orchestrator must stop feeling like hidden magic.

### The user should be able to see
- last orchestrator action
- why that action happened
- what the orchestrator recommends next
- why the card was not advanced further
- what input/approval is missing

### Example observable actions
- moved card from `inbox` to `clarifying`
- requested human clarification
- marked card `ready`
- dispatched card into execution
- ingested execution result
- moved card to `blocked`
- recommended park / retry / escalation

### Minimum V2 fields to make visible somewhere
- `last_orchestrator_action`
- `last_orchestrator_reason`
- `last_orchestrator_at`
- `recommended_next_action`
- `recommended_next_reason`

The underlying representation can evolve later, but V2 requires this observability model.

---

## 3. End-to-end trace model

V2 requires a visible chain from original work source to delivery outcome.

## Canonical trace chain

### Stage 1 — Intake/source
Source can be:
- ticket
- chat message
- form/request
- note
- manually created card

Minimum visibility:
- source reference(s)
- raw request
- intake artifact

### Stage 2 — Shaping
Visible shaping artifacts:
- normalized intent
- scope
- decisions made
- questions asked
- approvals requested
- ready-gate completion

### Stage 3 — Execution bridge
Visible linkage:
- dispatch event
- linked Beads IDs
- executor role/agent
- execution packet path

### Stage 4 — Review / verification
Visible outcomes:
- review attempts
- verification verdicts
- findings/open risks
- required follow-up

### Stage 5 — Delivery / deploy
Visible outcomes:
- deployment status
- release/deploy event refs
- environment target
- rollout state

### Stage 6 — Rollback / follow-up
Visible outcomes:
- rollback event
- incident/failure refs
- hotfix/follow-up linkage
- re-opened card or successor card if needed

### V2 rule
A colleague should be able to traverse this chain without reconstructing it manually from random files.

---

## 4. Friction telemetry model

V2 must expose not only state but **where the system hurts**.

## Why
A smooth board with invisible repeated failure is fake UX.
Great UX makes friction legible.

### Friction classes

#### A. Agent friction
How often the execution/review loop struggles.

Examples:
- review failure count
- rework cycle count
- execution retry count
- clarification loop count
- blocked cycle count

#### B. Delivery friction
How often the path to deploy fails or regresses.

Examples:
- deploy failure count
- rollback count
- failed verification count
- hotfix follow-up count
- release delay events

#### C. Process friction
How sticky or slow the control flow is.

Examples:
- time in current state
- total waiting_on_human duration
- dispatch latency
- ready age
- blocked age
- number of human interruptions

### V2 visibility rule
Friction should appear:
- on the card detail view
- in board summary markers when important
- in operator summary surfaces

Not every metric needs to be everywhere.
But the existence of friction must not be hidden.

---

## 5. Core surfaces in V2

### A. Operator Home
One-screen summary:
- attention now
- waiting on human
- blocked
- ready
- executing
- deployment/release trouble
- top friction hotspots
- next best action

### B. Project Control View
For one project:
- grouped state buckets
- bottlenecks
- active executions
- pending human/admin asks
- recent completions/deployments
- project friction summary

### C. Card Detail View
One card as a living control object:
- source → shaping → execution → review → deploy trace
- current missing thing
- orchestrator history summary
- friction counters/markers
- next action

### D. Delivery Trace View (later)
Cross-card / per-card path to deploy and rollback.

---

## 6. Action model

A V2 surface is incomplete if it only displays state.

Important actions to support near the card/summary:
- clarify
- request input
- mark ready
- execute / dispatch
- resume
- ingest result
- mark review outcome
- register deployment outcome
- register rollback / follow-up

### Rule
The user should not need to read state in one place and guess the next move in another.

---

## 7. Data/contract implications

V2 likely requires additional fields or event structures, but they should be introduced thinly.

Likely additions over time:
- source refs
- lifecycle event log
- orchestrator action summary fields
- review attempt counters
- execution attempt counters
- deploy status / deploy refs
- rollback count / rollback refs
- friction counters or derived summaries

### Important constraint
Do not preemptively explode the schema.
Add only the fields needed to support actual V2 visibility slices.

---

## 8. Implementation sequence

### Slice 1 — V2 working model and steering
This doc.

### Slice 2 — observable orchestrator + card detail contract
Add the minimum fields/views needed to show:
- source refs
- last orchestrator action/reason
- latest execution/review state
- basic friction counters

### Slice 3 — board summary upgrade
Make board/project views show:
- actionability
- waiting reason
- execution/deploy linkage
- friction markers

### Slice 4 — release/rollback trace integration
Close the chain from card to deployment outcome.

---

## 9. Non-goals right now

- fancy front-end for its own sake
- event-sourcing rewrite
- analytics warehouse
- perfect metrics taxonomy before implementation
- fully generalized observability platform

This is still a thin-slice control-tower evolution.

---

## Short formula

Control Tower V2 =
**board visibility + orchestrator observability + end-to-end trace + friction telemetry**

If those four are visible, colleagues trust the system.
If any of them are hidden, the UX is lying.
