# Implementation Drift Audit (2026-03-03)

Status: active
Scope: drift between intended stabilization requirements and current implementation state

## 1. Audit basis

Compared against:

- `docs/roadmap/STATE_ALIGNMENT_STREAM_ASTAR.md`
- `docs/roadmap/CONSISTENCY_MITIGATION_POLICY.md`
- `.sdp/STATE_ALIGNMENT_AUDIT_2026-03-03.md`
- Latest stabilization commits: `13eb1d7`, `52d04c5`, `c6ec109`, `709c903`, `7a3146b`, `28637bd`

Runtime checks executed:

- `./scripts/run_consistency_checks.sh`
- `python3 scripts/check_repo_consistency.py --strict-ac --json`
- `./scripts/go_with_project_toolchain.sh build ./...`
- `./scripts/go_with_project_toolchain.sh run ./cmd/sdp-guard ...`

## 2. Current gap summary

- High: 5
- Medium: 4
- Low: 2
- Total drifts: 11

Interpretation: documentation/consistency plane is mostly aligned; executable governance/tooling plane still diverges materially from intended stable-state requirements.

## 3. Detailed drift findings

### DRIFT-01 (High) — Stable-state build gate not met

- **Expected:** A* stable state requires `go build ./...` to pass.
- **Actual:** build fails with compile errors.
- **Evidence:** `./scripts/go_with_project_toolchain.sh build ./...`.
- **Files:** `internal/orchestrate/contract_gate.go`, `internal/bridge/beads_sink.go`, `internal/modelgateway/provider.go`, `internal/beads/sql_client.go`.

### DRIFT-02 (High) — Harness guard commands not runnable end-to-end

- **Expected:** new `sdp-guard` contract/clarification commands should be operational.
- **Actual:** command compile path fails due upstream compile errors in orchestrate/modelgateway/bridge/beads packages.
- **Evidence:** `./scripts/go_with_project_toolchain.sh run ./cmd/sdp-guard --check-contract ...` fails.

### DRIFT-03 (High) — Contract gate compile mismatch (value vs pointer)

- **Expected:** `EnforceContractGate` returns/uses `*harness.ComplianceReport` consistently.
- **Actual:** `harness.EvaluateCompliance` value is passed where pointer expected.
- **Evidence:** compile errors at `internal/orchestrate/contract_gate.go:49-55`.

### DRIFT-04 (High) — CI policy input still hardcodes evidence validation pass

- **Expected:** policy input derived from real evidence validation outcome.
- **Actual:** workflow writes `"evidence_validation_passed": true` unconditionally.
- **File:** `.github/workflows/ci.yml:274`.
- **Impact:** evidence-validation-related deny paths can be bypassed.

### DRIFT-05 (High) — OPA failures degrade to empty denials

- **Expected:** governance-critical check should fail on evaluation failure.
- **Actual:** `opa eval ... || echo '[]'` turns eval failures into effective pass.
- **Files:** `.github/workflows/ci.yml:294-304`.

### DRIFT-06 (Medium) — Consistency gate not strict in CI

- **Expected:** once warning set is zero, CI should use strict AC mode.
- **Actual:** CI runs `python3 scripts/check_repo_consistency.py --json` without `--strict-ac`.
- **File:** `.github/workflows/ci.yml:181`.

### DRIFT-07 (Medium) — Constraint check returns success on config-load failure

- **Expected:** failed governance config load should not silently pass.
- **Actual:** exits `0` on config load error.
- **File:** `cmd/sdp-guard/main.go:86-88`.

### DRIFT-08 (Medium) — Schema/example inconsistency in required evidence enum

- **Expected:** examples should satisfy schema enum.
- **Actual:** example contains `gate_results`, missing in schema enum.
- **Files:** `specs/examples/feature-task-contract.example.json:33`, `specs/runtime/schemas/feature-task-contract.schema.json:85-93`.
- **Evidence:** `invalid_values ['gate_results']` from schema-example check.

### DRIFT-09 (Medium) — Stabilization traceability gap in commit-to-requirement linking

- **Expected:** stabilization deliveries are explicitly traceable to requirement IDs/workstreams.
- **Actual:** recent commit messages do not include Beads/workstream IDs, reducing governance traceability.
- **Scope:** commits `13eb1d7`..`28637bd`.

### DRIFT-10 (Low) — Runbook assumes globally installed `sdp-guard`

- **Expected:** local reproducible path should be explicit for fresh environments.
- **Actual:** runbook uses `sdp-guard` directly; no `go run`/wrapper fallback shown.
- **File:** `docs/runbooks/HARNESS_CONTRACT_GATES_RUNBOOK.md`.

### DRIFT-11 (Low) — Policy evaluator fallback still advisory on local OPA absence

- **Expected:** enforce clear behavior for local policy execution in stabilization mode.
- **Actual:** orchestrate policy evaluator returns advisory on missing OPA.
- **File:** `internal/orchestrate/policy.go:44-49`.

## 4. What is aligned (no drift)

- Working tree entropy issue WT-001 resolved via commit slicing and clean git state.
- State-alignment Beads stream completed (23/23 closed).
- Strict consistency checker currently returns `ok=true` with zero errors/warnings.

## 5. Delta from intended project state

At this checkpoint, project intent and implementation diverge mainly in executable reliability and enforcement strictness, not strategic direction:

- **Direction alignment:** high
- **Operational reliability alignment:** medium-low (blocked by compile/gate drifts)
- **Governance strictness alignment:** medium

## 6. Remediation status update (2026-03-04)

### 6.1 Fixed in this pass

- **DRIFT-01 fixed:** `go build ./...` now succeeds after compile remediation.
- **DRIFT-02 fixed:** `sdp-guard --check-contract` now runs end-to-end on example contract/snapshot and returns non-blocking pass report.
- **DRIFT-03 fixed:** pointer/value mismatch in `internal/orchestrate/contract_gate.go` resolved.
- **DRIFT-04 fixed:** CI policy input no longer hardcodes evidence validation; now derives from `needs.evidence-gate.result`.
- **DRIFT-05 fixed:** OPA policy evaluation removed `|| echo '[]'` soft-pass fallback and now fails closed on eval errors.
- **DRIFT-06 fixed:** consistency gate now runs `python3 scripts/check_repo_consistency.py --strict-ac --json`.
- **DRIFT-07 fixed:** `sdp-guard --check-constraints` now fails closed on constraint config load errors.
- **DRIFT-08 fixed:** `gate_results` added to `required_evidence` enum in `specs/runtime/schemas/feature-task-contract.schema.json`.
- **DRIFT-10 fixed:** runbook now includes `go run ./cmd/sdp-guard` local fallback.
- **DRIFT-11 fixed:** local policy evaluator now returns error when OPA is missing instead of silent advisory fallback.

### 6.2 Remaining drifts

- **DRIFT-09 (Medium):** commit-to-requirement traceability still inconsistent in commit messages.

### 6.3 Verification evidence (2026-03-04 pass)

- `python3 scripts/check_repo_consistency.py --strict-ac --json` => `ok=true`, `errors=0`, `warnings=0`.
- `./scripts/run_consistency_checks.sh` => fallback checks pass; `sdp-protocol-check`/`sdp-doc-sync` absent in PATH (reported by script).
- `./scripts/go_with_project_toolchain.sh build ./...` => pass.
- `./scripts/go_with_project_toolchain.sh run ./cmd/sdp-guard --check-contract --contract specs/examples/feature-task-contract.example.json --snapshot specs/examples/feature-task-snapshot.example.json` => pass report (`blocked=false`).
- `./scripts/go_with_project_toolchain.sh run ./cmd/sdp-guard --check-constraints --phase build --command "go test ./..."` => pass (`OK: no constraint violations`).

### 6.4 Known pre-existing quality failures (outside this remediation scope)

- Full `go test ./...` remains red due pre-existing failures/build issues outside files touched in this pass:
  - duplicate tests in `internal/orchestrate` (`TestIsolatedMergePolicy`, `TestPolicyConsistencyChecker`),
  - type/API mismatches in `internal/planner/scheduler_test.go` and `internal/session/writer_test.go`,
  - failing assertion in `internal/modelgateway/credentials_test.go` (`TestCredentialManagerRotate`),
  - timeout failures in `internal/verify/quorum_test.go`.
