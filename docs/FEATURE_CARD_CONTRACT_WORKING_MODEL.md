# FeatureCard Contract — Working Model

Status: working model
Date: 2026-03-22
Scope: control-panel level feature intake and shaping contract
Related:
- `docs/PROJECT_CONTROL_PANEL_WORKING_MODEL.md`
- `docs/PROJECT_CONTROL_PANEL_BACKLOG.md`
- `docs/TASK_ENVELOPE_TO_ARTIFACT_BUNDLE_MAPPING.md`
- `docs/specs/project-registry.yaml`

## Purpose

Define `FeatureCard` as the canonical entity for project control panel intake and shaping.

This contract exists to solve a specific problem:

- a human needs a cheap way to drop feature ideas into a project
- the orchestrator needs a structured object it can clarify and mature
- the system needs a clear bridge from early feature shaping into Beads execution and SDP artifact expectations

`FeatureCard` is that object.

---

## 1. What a FeatureCard is

A `FeatureCard` is a **portfolio/project-board level feature object**.

It is:
- higher-level than a Beads execution task
- earlier than a full execution packet
- lighter than a full SDP artifact bundle
- the main unit used in `Inbox`, `Clarifying`, and `Ready`

### It is NOT
- not a Beads issue
- not a workstream file
- not an evidence envelope
- not a PR object

---

## 2. Lifecycle position

`FeatureCard` typically moves through this state machine:

```text
inbox -> clarifying -> ready -> executing -> reviewing -> done
```

Optional side states:

```text
inbox -> parked
clarifying -> needs_input
executing -> blocked
reviewing -> needs_changes
```

### Important rule

The card remains the human-facing control object even after execution begins.
Execution tasks may be spawned underneath it, but the card remains the board-level summary object.

---

## 3. Minimal required fields

These fields are required for every `FeatureCard`.

```yaml
id: feature-<project>-<date>-<seq>
project_id: <project id from registry>
title: <short human-facing title>
status: inbox|clarifying|ready|executing|reviewing|blocked|done|parked|needs_input
raw_request: <original request text>
created_at: <iso timestamp>
updated_at: <iso timestamp>
```

### Field notes

#### `id`
Stable control-panel identifier.
Not the same as Beads ID.

#### `project_id`
Must resolve to a known project in `project-registry.yaml`.

#### `title`
Short card title, human-facing.

#### `status`
Board state of the card.

#### `raw_request`
The original human request, preserved as intake truth.

---

## 4. Recommended shaping fields

These fields are optional at intake, but expected once the card is clarified.

```yaml
normalized_intent: <normalized task statement>
task_type: feature|bugfix|refactor|review|research|process-design|release|mixed
execution_mode: explore|plan|build|review|debug|docs|mixed
target_repo: <repo/project>
target_area: <subsystem/path/area or unknown>
scope_in:
  - <item>
scope_out:
  - <item>
non_goals:
  - <item>
risk_level: low|medium|high|unknown
why_now: <optional urgency/context>
links:
  - <url or file ref>
open_questions:
  - <question>
acceptance_shape:
  - <acceptance statement>
recommended_next_step: <next action>
```

### Purpose of these fields

They let the orchestrator move the card from raw idea to execution-ready object without prematurely turning it into a full execution task.

---

## 5. Bridge fields

These fields connect the card to execution and process layers after shaping.

```yaml
linked_beads_ids:
  - <beads id>
linked_workstreams:
  - <workstream id>
required_artifacts:
  - <artifact name>
required_checks:
  - <check name>
linked_artifacts:
  - <artifact path or id>
blocking_reasons:
  - <reason>
waiting_on:
  - <human|dependency|review|qa|external>
```

### Important rule

These fields should appear only when they become real.
Do not fabricate them at raw intake time.

---

## 6. Canonical YAML example

```yaml
id: feature-openclaw-2026-03-22-001
project_id: openclaw
title: Unify reminder escalation rules
status: clarifying
raw_request: make reminders escalate more intelligently across projects and personal tasks
created_at: 2026-03-22T09:20:00Z
updated_at: 2026-03-22T09:27:00Z
normalized_intent: unify reminder escalation policy across reminder-producing flows
task_type: feature
execution_mode: mixed
target_repo: openclaw
target_area: reminders and notification policy
scope_in:
  - escalation levels
  - reminder severity mapping
  - cross-project consistency
scope_out:
  - full calendar redesign
  - unrelated notification channels
non_goals:
  - redesign all reminder UX
risk_level: medium
why_now: current reminder behavior is inconsistent and easy to ignore
links:
  - /docs/reminders/current-behavior.md
open_questions:
  - which channels should escalate beyond chat only?
  - should personal and project reminders share the same severity thresholds?
acceptance_shape:
  - reminders have explicit escalation levels
  - escalation behavior is consistent across supported flows
recommended_next_step: ask two human clarification questions, then mark ready
linked_beads_ids: []
linked_workstreams: []
required_artifacts: []
required_checks: []
linked_artifacts: []
blocking_reasons: []
waiting_on: []
```

---

## 7. Status semantics

### `inbox`
Raw intake. Human dropped the request. Minimal structure.

### `clarifying`
Orchestrator is shaping the card.
May still have ambiguity.

### `needs_input`
Blocked on human clarification or missing decision.

### `ready`
Sufficiently shaped for decomposition/execution handoff.

### `executing`
Execution tasks exist and active work is underway.

### `reviewing`
Execution is largely done; verification/review/QA still active.

### `blocked`
Cannot move due to dependency, external blocker, or execution issue.

### `parked`
Deliberately deferred.

### `done`
Accepted and completed.

### `needs_changes`
Optional state if the panel chooses to surface review bounce-back explicitly.
Can also be modeled as `reviewing` + blocking reason.

---

## 8. Ready gate

A card may move to `ready` only if all of the following are true:

1. `normalized_intent` exists
2. `task_type` is known
3. `target_repo` is known
4. at least rough `scope_in` exists
5. `risk_level` is set or explicitly `unknown`
6. there is a `recommended_next_step`

### Strongly recommended before `ready`
- at least one `acceptance_shape` item
- at least one `scope_out` or `non_goals` item

---

## 9. Bridge to SDP

The bridge to SDP begins immediately at intake.

At minimum, the card should gain an early SDP artifact such as:
- intake note
- intent brief
- task brief

As the card matures, the orchestrator or SDP layer can derive:
- `required_artifacts`
- `required_checks`
- minimum artifact bundle using task envelope mapping

### Principle
A `FeatureCard` should not require full SDP paperwork at intake time.
But it **should** start traceable SDP context at intake time.

---

## 10. Bridge to Beads

When a `FeatureCard` becomes sufficiently mature, the system may create or refine:
- a feature-level Beads issue
- child execution tasks later
- workstream records later if needed

### Principle
A `FeatureCard` should bridge into Beads, not be replaced by Beads.
Beads is the evolving execution graph, not necessarily the first raw-intake object.

The control panel still needs the card as the project-board summary object.

---

## 11. Recommended orchestrator actions on a FeatureCard

### `clarify`
Populate shaping fields from raw intake.

### `ask_human`
Move card to `needs_input` with explicit unanswered questions.

### `mark_ready`
Apply ready gate and move to `ready`.

### `park`
Move card to `parked` with reason.

### `spawn_execution`
Create feature-level execution tasks / links and move to `executing`.

### `reopen`
Move card back from `ready|blocked|reviewing` to `clarifying` if the shape is no longer sufficient.

---

## 12. Anti-patterns

### Anti-pattern 1
Using Beads issues as raw feature inbox cards.

Why bad:
- raw intake becomes too heavy
- execution system gets polluted with vague ideas

### Anti-pattern 2
Demanding full artifact bundles before clarification.

Why bad:
- kills usability
- discourages cheap capture

### Anti-pattern 3
Treating `ready` as "already decomposed into full execution graph".

Why bad:
- makes shaping too expensive
- slows down flow

### Anti-pattern 4
Losing the original raw request.

Why bad:
- normalized intent can drift from what the human actually asked for

---

## 13. Short formula

A `FeatureCard` is:
- cheap to create
- structured enough to clarify
- durable enough to survive the board lifecycle
- bridgeable into Beads and SDP

That is its job.
