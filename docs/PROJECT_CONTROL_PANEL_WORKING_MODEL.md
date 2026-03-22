# Project Control Panel — Working Model

Status: working model
Date: 2026-03-22
Scope: multi-project intake, clarification, and execution visibility across SDP + Beads + orchestrator

## Goal

Define a practical control panel model for managing multiple projects where:
- the human can quickly drop in feature ideas
- the orchestrator can clarify and shape them
- features move toward ready status instead of rotting in raw intake
- execution state is visible across projects

This is not a full product spec for the final dashboard.
It is the current working model for what the control surface should do.

---

## 1. Problem to solve

Right now the pieces exist separately:
- **Beads** tracks issues and dependencies
- **SDP** defines process/evidence/artifact expectations
- there are existing efforts around intake, workstreams, feature shortcuts, and scheduler/orchestrator flows
- multiple projects already exist in the registry (`beads`, `openclaw`, `opencode`, `kubeopencode`, `sdp`, `sdp_dev`)

But what is still missing as a clean human-facing surface is:

1. a project-level board view
2. a feature intake inbox
3. a clarification loop before execution
4. a "ready for build" state that is explicit
5. a consistent way to see what is blocked, in progress, and waiting on human input

---

## 2. Core concept

The control panel should be a **portfolio + project board layer** over Beads and SDP.

### Human mental model
The human should be able to do two things easily:

1. **Drop a feature idea into a project inbox**
2. **See where every project feature sits in the flow**

### System mental model
The system should treat each incoming item as moving through a small state machine:

```text
inbox -> clarifying -> shaped -> ready -> executing -> verifying -> done
```

With optional side exits:

```text
inbox -> parked
clarifying -> waiting-human
executing -> blocked
verifying -> needs-changes
```

---

## 3. What the control panel is actually showing

The control panel is not a replacement for Beads.
It is a **view + orchestration layer** on top of Beads state plus SDP state.

### Primary entities

#### A. Project
A top-level container such as:
- beads
- openclaw
- opencode
- kubeopencode
- sdp
- sdp_dev
- later: gastown, wasteland, etc.

#### B. Feature card
A human-facing feature/request card.
This is the thing a human throws onto a board.

#### C. Execution tasks
These are the lower-level Beads issues/workstreams spawned from a shaped feature.

### Important distinction
A **feature card** is not the same as an execution task.

The feature card exists earlier and at a higher level.
Only after clarification/shaping does it fan out into:
- Beads issues
- workstreams
- artifact expectations
- execution paths

---

## 4. Recommended board columns

Each project should have a board view with these default columns:

### `Inbox`
Raw feature ideas and requests.
Minimal structure.
Fast capture.

### `Clarifying`
Orchestrator is actively refining the request.
Waiting for:
- scope clarification
- acceptance shape
- risk framing
- missing context

### `Ready`
Feature is shaped enough to execute.
Has:
- objective
- scope
- initial acceptance
- recommended next action
- enough context to decompose into work

### `Executing`
Execution tasks/workstreams are in progress.

### `Review / Verify`
Implementation exists, now waiting on:
- review
- verification
- QA/UAT
- evidence completion

### `Blocked`
Cannot proceed due to:
- dependency
- missing decision
- external blocker
- waiting on human

### `Done`
Accepted and complete.

### Optional side columns
- `Parked` — intentionally deferred
- `Needs input` — explicit human action required

---

## 5. Intake contract for new feature cards

A human should be able to add a new feature card with a very small amount of info.

### Minimum intake fields
- `project_id`
- `title`
- `raw_request`

### Recommended extra fields
- `why_now`
- `links`
- `rough_priority`
- `rough_risk`

### Example
```yaml
project_id: openclaw
title: unify reminder escalation rules
raw_request: make reminders escalate more intelligently across projects and personal tasks
why_now: current reminder behavior is inconsistent and easy to ignore
links:
  - https://...
rough_priority: high
rough_risk: medium
```

### Key principle
Do **not** require full SDP artifacts at inbox time.
Inbox must be lightweight or it will never be used.

---

## 6. Clarification loop

This is where the orchestrator becomes essential.

### Responsibility of the orchestrator
For each card in `Inbox`, the orchestrator should try to move it toward `Ready` by:
- extracting intent
- detecting ambiguity
- asking targeted questions only when needed
- proposing initial scope boundaries
- identifying likely affected repo/area
- proposing task type and risk level
- deciding whether the card is ready or still under-specified

### Outputs of clarification
A clarified card should produce at least:
- normalized intent
- task type
- risk guess
- scope in/out
- acceptance shape
- recommended next step

This is effectively a thin **task brief** for the feature card.

### State transitions
```text
Inbox -> Clarifying
Clarifying -> Ready
Clarifying -> Needs input
Clarifying -> Parked
```

---

## 7. What “Ready” means

A feature card is `Ready` when it is shaped enough that execution can begin without guessing the core objective.

### Minimum readiness standard
A ready feature should have:
- clear intent
- explicit scope or non-goals
- rough acceptance criteria
- target project/repo
- risk estimate
- recommended next action

### Not required yet
At `Ready`, we do **not** necessarily need:
- full workstream decomposition
- complete implementation plan
- full evidence bundle

Those come later.

---

## 8. How this maps to Beads and SDP

### Beads remains source of truth for executable tasks
Once a feature becomes `Ready`, the system can generate or link:
- feature-level Beads issue
- child tasks/workstreams
- dependencies
- findings loop entries

### SDP governs process expectations
Once a feature is shaped and moving to execution, SDP determines:
- required artifact bundle
- required checks
- acceptance and verification expectations

### The control panel sits above both
It answers:
- what project is this in?
- what state is this feature in?
- is this still raw intake, ready, executing, blocked, or done?

---

## 9. Suggested canonical data model for the control panel

### `ProjectCard`
```yaml
project_id: openclaw
name: OpenClaw
beads_prefix: openclaw
repo_url: https://github.com/openclaw/openclaw
board_enabled: true
```

### `FeatureCard`
```yaml
id: feature-openclaw-2026-03-22-001
project_id: openclaw
title: unify reminder escalation rules
status: clarifying
raw_request: make reminders escalate more intelligently across projects and personal tasks
normalized_intent: unify reminder escalation policy across reminder-producing flows
scope_in:
  - reminder prioritization
  - escalation levels
scope_out:
  - calendar sync redesign
risk_level: medium
recommended_next_step: ask 2 clarification questions about channels and severity thresholds
linked_beads_ids: []
linked_artifacts: []
```

---

## 10. Existing building blocks already present

The current code/docs base already has useful pieces:

### Project registry
`docs/specs/project-registry.yaml`

This can seed the project list for the panel.

### Beads queue view schema
`schema/contracts/beads-queue-view.schema.json`

This is already close to a board widget for executable tasks.

### Feature-task contract examples
- `specs/examples/feature-task-contract.example.json`
- `specs/examples/feature-task-snapshot.example.json`

These already model execution-side feature contracts and snapshots.

### Feature shortcut runbook
`docs/FEATURE_SHORTCUT_RUNBOOK.md`

This already points toward the execution-side shortcut after shaping.

---

## 11. Recommended phased implementation

### Phase 1 — thin control board
Build a simple board that shows, per project:
- inbox cards
- clarifying cards
- ready cards
- active execution summary from Beads
- blocked cards

At this stage, the panel can be mostly a read/write wrapper around JSON + Beads.

### Phase 2 — orchestrator-assisted intake
Add orchestrator actions:
- clarify card
- mark ready
- ask human question
- spawn feature decomposition

### Phase 3 — execution bridge
From a `Ready` card, generate:
- feature-level Beads issue
- workstream/task structure
- initial SDP artifact requirements

### Phase 4 — dashboard enrichment
Add:
- per-project metrics
- stale card detection
- waiting-on-human queues
- ready-to-execute recommendations
- blocked dependency visualization

---

## 12. Recommended immediate product shape

The simplest useful version is:

### Portfolio home
Shows all projects and counts:
- inbox
- clarifying
- ready
- executing
- blocked

### Project board
Per project Kanban:
- Inbox
- Clarifying
- Ready
- Executing
- Blocked
- Done

### Feature drawer/card detail
Shows:
- raw request
- normalized intent
- scope
- acceptance shape
- open questions
- linked Beads tasks
- next recommended action

### Orchestrator actions
- Clarify
- Mark Ready
- Park
- Create execution tasks
- Re-open for clarification

---

## 13. Short formula

### Human wants
"накидать фичи по проектам и видеть, что с ними происходит"

### System should do
- collect raw feature cards cheaply
- let the orchestrator shape them
- mark when they become execution-ready
- bridge them into Beads + SDP only after they are mature enough

### Therefore
The control panel should be:
- **project board layer** for humans
- **intake/clarification surface** for orchestrator
- **bridge into Beads + SDP** for execution
