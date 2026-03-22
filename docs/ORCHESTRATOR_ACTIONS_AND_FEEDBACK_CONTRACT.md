# Orchestrator Actions & Feedback Contract

Status: working model
Date: 2026-03-22
Scope: orchestrator-driven state changes and human/admin feedback loops for the SDP control tower
Related:
- `docs/PROJECT_CONTROL_PANEL_WORKING_MODEL.md`
- `docs/FEATURE_CARD_CONTRACT_WORKING_MODEL.md`
- `docs/BEADS_GASTOWN_SDP_CONTROL_TOWER_INTEGRATION_PLAN.md`
- `schema/contracts/feature-card.schema.json`
- `schema/contracts/project-board-snapshot.schema.json`
- `schema/contracts/portfolio-board-snapshot.schema.json`

## Goal

Define how the orchestrator acts on control-tower state and when it must surface feedback requests, updates, or decision points to humans/admins.

The key principle is:

- the orchestrator is the default actor
- humans/admins are exception and decision loops
- the board visualizes this, but does not define it

---

## 1. Control philosophy

### Default mode
The orchestrator should continue autonomously whenever it has enough information and authority to do so safely.

### Human/admin loop
The orchestrator should interrupt autonomy only when one of these is true:
- missing domain/product clarification
- explicit decision required
- approval required
- safety/risk threshold crossed
- external dependency cannot be resolved autonomously

### Board role
The board is a visualization of:
- orchestrator actions
- active agents
- pending feedback requests
- pending admin actions
- current execution state

It is not the primary source of action logic.

---

## 2. Canonical orchestrator actions

These are the primary actions the orchestrator may perform on a `FeatureCard`.

### Intake and shaping actions
- `create_card`
- `start_trace`
- `clarify_card`
- `normalize_intent`
- `set_scope`
- `set_risk`
- `set_acceptance_shape`
- `mark_needs_input`
- `mark_ready`
- `park_card`
- `reopen_card`

### Execution-bridge actions
- `create_beads_feature`
- `link_beads_issue`
- `create_workstream_tasks`
- `attach_artifact_expectations`
- `spawn_execution`
- `mark_executing`

### Review and completion actions
- `collect_results`
- `request_review`
- `request_verification`
- `mark_reviewing`
- `mark_blocked`
- `mark_done`
- `create_follow_up`

### Feedback/output actions
- `request_author_feedback`
- `request_admin_action`
- `send_author_update`
- `send_admin_update`
- `record_decision`
- `record_escalation`

---

## 3. Action classes

### A. Autonomous actions
These do not require explicit human/admin input.

Examples:
- `create_card`
- `start_trace`
- `clarify_card`
- `normalize_intent`
- `set_scope`
- `set_risk`
- `attach_artifact_expectations`
- `send_author_update`
- `send_admin_update`

### B. Clarification-gated actions
These require missing information to be resolved first.

Examples:
- `mark_ready`
- `create_beads_feature`
- `spawn_execution`

### C. Approval/decision-gated actions
These require explicit human/admin decision.

Examples:
- risky scope expansion
- policy override
- release-sensitive go/no-go
- destructive or externally impactful actions

### D. Escalation actions
These happen when autonomy stalls or safety/risk exceeds normal bounds.

Examples:
- `request_admin_action`
- `record_escalation`
- `mark_blocked`

---

## 4. When feedback is required

### Require author feedback when
- product intent is ambiguous
- acceptance criteria are underspecified
- multiple reasonable feature interpretations exist
- tradeoff between scope options must be chosen
- UX/product direction needs owner input

### Require admin action when
- approval is required for risky/external action
- security/privacy concern appears
- policy override is needed
- project-level priority or cross-project arbitration is needed
- system cannot resolve conflict autonomously

### Do not ask when
- the system can safely infer and continue
- the clarification is trivial and low-risk
- feedback would only create unnecessary friction

---

## 5. Canonical feedback loop fields

These fields in `FeatureCard` represent outward-facing interaction needs.

### `needs_feedback_from`
Who needs to respond:
- `author`
- `admin`
- `human`

### `feedback_request`
Concrete question(s) that need answering.

### `decision_required`
Decision(s) the system cannot make alone.

### `author_update`
Human-readable update intended for the request author.

### `admin_action_required`
Explicit action/approval/escalation needed from admin/operator.

### `active_agents`
Who is currently working on the card.

---

## 6. Message quality rules

When the orchestrator surfaces feedback to a human/admin, the message should be:

- concise
- decision-oriented
- explicit about why the system is blocked or pausing
- explicit about the next step after the answer

### Good example
"Need one product decision before marking this ready: should personal and project reminders share the same escalation thresholds? Once answered, I can move this into execution planning."

### Bad example
"Please provide more details."

---

## 7. State transition expectations

### `inbox -> clarifying`
Default orchestrator move after intake.

### `clarifying -> needs_input`
Use only when a concrete unanswered question blocks safe shaping.

### `clarifying -> ready`
Use when ready gate passes and no unresolved critical questions remain.

### `ready -> executing`
Use when execution objects are created and active work begins.

### `executing -> reviewing`
Use when implementation exists and review/verification is now primary.

### `executing -> blocked`
Use when execution cannot continue autonomously.

### `reviewing -> done`
Use when checks/review/acceptance are satisfied.

### `reviewing -> needs_input`
Use if final acceptance or owner feedback is required.

---

## 8. Feedback loop templates

### Template: author clarification request
- what is being built
- what is unclear
- which decision is needed
- what the system will do after the answer

### Template: admin action request
- what operation or risk triggered the request
- why admin action is needed
- what options exist
- what happens after approval/rejection

### Template: author update
- current state
- what the system has already done
- what remains
- whether any response is needed

---

## 9. Suggested default orchestrator policy

### Policy 1
Try to move the card forward without asking a question unless ambiguity actually blocks safe progress.

### Policy 2
When asking for input, ask the smallest possible high-leverage question.

### Policy 3
When escalating to admin, provide a decision packet, not a vague alert.

### Policy 4
Always record why autonomy stopped.

### Policy 5
If the answer arrives, resume automatically where possible.

---

## 10. Example flow

### Raw intake
Human creates a feature card with a rough request.

### Orchestrator actions
- `create_card`
- `start_trace`
- `clarify_card`
- `normalize_intent`
- `set_scope`

### If ambiguity remains
- `mark_needs_input`
- `request_author_feedback`

FeatureCard fields update:
- `needs_feedback_from: [author]`
- `feedback_request: [...]`
- `author_update: [...]`

### After answer arrives
- `reopen_card`
- `clarify_card`
- `mark_ready`
- `create_beads_feature`
- `spawn_execution`

---

## 11. Relationship to the board

The board should render these things clearly:
- card status
- active agents
- who owes feedback
- what exact feedback/decision is needed
- current recommended next step

This makes the board useful for humans/admins without making it the engine of orchestration.

---

## 12. Short formula

- orchestrator acts by default
- humans/admins answer by exception
- board visualizes the state of that loop
- `FeatureCard` carries the feedback contract
