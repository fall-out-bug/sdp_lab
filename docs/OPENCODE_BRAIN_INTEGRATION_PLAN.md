# OpenCode Brain Integration Plan (Stage A)

Status: active
Priority: highest
Goal: deliver autonomous `feature -> PR` flow in OpenCode with Strict Evidence and private policy enforcement.

## 1. Scope

In scope:

- private brain integration into OpenCode runtime
- beads-driven task claiming and state transitions
- feature branch execution only
- strict evidence generation and PR publication
- model restriction enforcement (`glm-5`, `glm-4.7`)

Out of scope (Stage A):

- repo split execution
- OpenClaw runtime parity
- enterprise policy packs rollout

## 2. Architecture slice

Core components:

1. `Brain Gateway`
   - receives execution intent
   - applies risk/policy classification
   - returns execution decision
2. `Backlog Adapter (Beads)`
   - claim, update status, read dependencies
3. `Execution Driver`
   - branch creation and task run orchestration
4. `Verification Driver`
   - test/lint/contract/coverage gates
5. `Evidence Builder`
   - emits 7-section strict bundle
6. `PR Publisher`
   - opens PR and links trace back to beads

## 3. Stage A milestones

### A1. Brain decision API inside OpenCode

- define internal request/response schema
- include risk class, lane, model choice, branch target

### A2. Beads lifecycle wiring

- `open -> in_progress -> review -> verified -> done`
- enforce blocker checks before claim
- enforce transition gates from `BEADS_AUTONOMY_SPEC.md`

### A3. Branch and git safety enforcement

- branch pattern: `feat/<issue-id>-<slug>`
- deny protected branch pushes
- deny execution when branch target is ambiguous

### A4. Strict verification bundle

- build and validate all sections:
  - intent, plan, execution, verification, review, risk_notes, trace
- block PR if any section missing

### A5. Model policy enforcement

- allowlist: `glm-5`, `glm-4.7`
- fallback chain: `glm-5 -> glm-4.7 -> escalated`

### A6. End-to-end scenario run

- run scenario `user feature -> PR`
- produce one full traceable example and operator runbook

## 4. Acceptance criteria

- AC1: A feature request can be transformed into beads tasks and executed to PR without manual intermediate edits.
- AC2: All transitions obey Beads FSM and are rejected when invalid.
- AC3: Every published PR includes complete strict evidence.
- AC4: Only allowed models are ever selected; violations escalate.
- AC5: Human merge remains mandatory and explicit in PR template.

## 5. Deliverables

- OpenCode integration spec + code changes
- one operator runbook for daily use
- one incident runbook for escalation cases
- one reproducible demonstration trace
