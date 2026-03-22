# Project Control Panel — Immediate Backlog

Status: working backlog
Date: 2026-03-22
Related:
- `docs/PROJECT_CONTROL_PANEL_WORKING_MODEL.md`
- `docs/specs/project-registry.yaml`
- `schema/contracts/beads-queue-view.schema.json`

## Goal

Turn the control panel idea into a sequence of concrete implementation slices.

---

## P0 — define the source of truth split

### 1. Define entity boundaries
Need explicit split between:
- `FeatureCard` (portfolio/control-panel level)
- `Beads issue` (execution task level)
- `SDP artifacts` (process evidence level)

Deliverable:
- small schema note or contract draft for `FeatureCard`

### 2. Define status vocabulary
Need canonical status set:
- `inbox`
- `clarifying`
- `ready`
- `executing`
- `reviewing`
- `blocked`
- `done`
- `parked`
- `needs_input`

Deliverable:
- status enum contract

---

## P1 — build the thin board first

### 3. Project list from registry
Use `docs/specs/project-registry.yaml` as initial project source.

Deliverable:
- panel can list projects from registry

### 4. Per-project feature inbox storage
Need simple durable storage for raw feature cards.

Candidates:
- JSONL per project
- one shared JSON file
- Beads issue with special label (not preferred for raw intake)

Recommended first cut:
- local JSONL or structured file owned by SDP

Deliverable:
- create/read/update feature cards without requiring full Beads decomposition

### 5. Read-only execution summary widget
Use existing Beads queue view concepts to show:
- ready count
- blocked count
- in-progress count
- next action

Deliverable:
- project board can show execution-side health without mixing it with raw intake cards

---

## P2 — orchestrator-assisted shaping

### 6. Clarification action
Orchestrator takes an inbox card and produces:
- normalized intent
- scope in/out
- risk guess
- open questions
- recommended next step

Deliverable:
- card moves `inbox -> clarifying` or `inbox -> ready`

### 7. Ready gate
Define the minimum condition for `ready`.

Deliverable:
- explicit readiness checklist in code/docs

### 8. Human question lane
Need first-class support for:
- waiting on answer
- preserving unanswered clarification questions

Deliverable:
- `needs_input` state with visible pending questions

---

## P3 — bridge ready cards into execution

### 9. Ready card -> feature issue/task creation
Once ready, create:
- feature-level Beads issue
- optional child tasks/workstreams later

Deliverable:
- one command/action to convert a ready feature card into execution state

### 10. Attach SDP expectations
At execution handoff, derive:
- task type
- risk level
- minimum artifact bundle

Deliverable:
- ready card becomes execution packet skeleton

---

## P4 — operator visibility and stale control

### 11. Stale card detection
Need visibility for:
- inbox cards untouched too long
- clarifying cards waiting too long
- blocked cards with no movement

Deliverable:
- stale markers / reminders

### 12. Portfolio dashboard
Across all projects, show:
- waiting on human
- ready to execute
- blocked by dependency
- active execution count

Deliverable:
- one-screen portfolio control view

---

## Suggested first implementation slice

If we want the fastest useful result, do this first:

1. define `FeatureCard` schema
2. add per-project inbox storage
3. render per-project board columns
4. add orchestrator `clarify` action
5. add `mark_ready` action
6. show existing Beads execution summary separately

That already gives:
- project boards
- raw feature intake
- clarification loop
- ready queue
- separation between planning intake and execution tracking
