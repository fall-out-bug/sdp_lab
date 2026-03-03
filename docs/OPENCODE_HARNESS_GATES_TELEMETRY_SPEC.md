# OpenCode Harness Gates + Telemetry Spec

Status: proposal v1
Scope: harness-level control for feature execution, requirement drift prevention, and quality gates

## 1. Problem

Agent workflows can start strong during discovery and then degrade acceptance criteria, metrics, or evidence quality during implementation.

Harness-level controls must prevent this by enforcing a deterministic protocol and hard gates.

## 2. Target Outcome

The harness must guarantee:

- Requirement integrity (no silent scope/metric degradation)
- Evidence-backed claims (no unsupported output)
- Deterministic state transitions
- Mandatory quality checks before completion
- Drift visibility via telemetry and machine-readable reports

## 3. Protocol State Machine

Canonical phase order:

1. `discover`
2. `plan`
3. `implement`
4. `validate`
5. `report`
6. `done`

Rules:

- Out-of-order transitions are rejected.
- Every phase emits `phase_start` and `phase_end` telemetry.
- Every phase_end carries `status` in: `success`, `failed`, `blocked`, `retrying`.
- `done` is allowed only if all required gates passed.

## 4. Task Contract (Immutable Baseline)

At `/feature` start, harness creates immutable contract artifact:

- Path: `.sdp/contracts/feature-<run_id>.json`
- Schema: `specs/runtime/schemas/feature-task-contract.schema.json`

Required contract fields:

- `objective`: feature goal
- `acceptance_criteria[]`: mandatory AC list
- `required_metrics[]`: baseline metric set that cannot shrink without approval
- `required_evidence[]`: required artifact categories (tests, logs, diffs, traces)
- `quality_gates`: required checks (build/test/lint/typecheck)
- `constraints`: policy/security/performance constraints
- `version` and `created_at`

## 5. Hard Gates

### 5.1 Requirement Integrity Gate

Fails if:

- an acceptance criterion is removed
- a required metric is removed or downgraded
- definition-of-done is weakened

Bypass path:

- only via explicit change request with reason and approver metadata

### 5.2 Evidence Gate

Fails if any required evidence type is missing, or claims lack evidence references.

Enforcement policy: `no_evidence_no_claim=true`.

### 5.3 Metric Parity Gate

Fails if final metric set cardinality is lower than baseline without approved exception.

### 5.4 Quality Gate

Fails if any required validation command fails.

Minimum set:

- build
- tests
- lint
- typecheck

### 5.5 Process Gate

Fails if final report does not include:

- contract coverage summary (`planned vs delivered`)
- gate results
- evidence index
- drift decisions log

## 6. Telemetry Envelope

Every transition emits structured event:

- `run_id`
- `phase`
- `status`
- `gate_id` (if gate event)
- `contract_hash`
- `requirements_hash`
- `metrics_hash`
- `evidence_refs[]`
- `decision_id` and `decision_reason`
- `drift_detected` (bool)
- `drift_type` (`ac_drop`, `metric_drop`, `scope_weaken`, `unsupported_claim`)

Recommended sink:

- `.sdp/observability/intake.jsonl`

## 7. Drift Detector

Detector compares current snapshots against immutable contract.

Triggers:

- AC cardinality decrease
- Required metric set decrease
- Required evidence categories decrease
- New completion claim without evidence ref

Actions:

1. Block current phase transition
2. Emit `drift_blocked` event
3. Force return to `plan` state
4. Require explicit change request to continue

## 8. CI Enforcement

Add CI job: `protocol-compliance`.

Job runs:

1. Contract schema validation
2. Gate result verification
3. Drift history check (no unapproved critical drift)
4. Evidence completeness check

PR must show:

- `AC coverage: x/y`
- `Metric parity: pass/fail`
- `Evidence completeness: pass/fail`
- `Gates summary`

## 9. Rollout Plan

1. Shadow mode: emit telemetry, do not block (1 week)
2. Soft-fail mode: block only critical drift (1 week)
3. Enforced mode: all hard gates active

Deployment note:

- Enforcement is repository-local. A target repository must install the guard/job to be controlled.

## 10. Non-Goals

- Replacing existing repo-specific test pipelines
- Agent prompt-level behavior tuning as primary control

Primary control is harness protocol + gate enforcement.

## 11. Current implementation

Implemented in this repository:

- Gate engine: `internal/harness/evaluate.go`
- Clarification classification + apply flow: `internal/harness/clarification.go`
- Contract/snapshot IO: `internal/harness/io.go`
- Contract guard CLI mode: `sdp-guard --check-contract`
- Clarification CLI modes:
  - `sdp-guard --classify-clarification`
  - `sdp-guard --apply-clarification`

Operator guide:

- `docs/runbooks/HARNESS_CONTRACT_GATES_RUNBOOK.md`
