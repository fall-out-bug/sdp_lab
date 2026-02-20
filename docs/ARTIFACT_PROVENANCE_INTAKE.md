# Artifact Provenance Intake

This intake defines the initial artifact classes for the swarm artifact bus, retention windows, and provenance fields that must be captured across each execution phase.

## Artifact Classes

| Class ID | Purpose | Swarm phases | Retention window |
| --- | --- | --- | --- |
| `intent-brief` | Issue framing, acceptance gates, and risk posture | `intake`, `plan` | 365 days |
| `execution-plan` | Deterministic workstream ordering and dependencies | `plan` | 365 days |
| `code-diff` | Patch set for implementation changes | `execute` | 1095 days |
| `verification-report` | Test/lint/contract gate results | `verify` | 1095 days |
| `review-verdict` | Reviewer decision and risk signoff | `review`, `publish` | 1095 days |
| `trace-link` | Commit/PR evidence pointers | `verify`, `publish` | 1825 days |

Retention baseline rationale:

- planning artifacts (`intent-brief`, `execution-plan`) retain for one year to support auditability of scope decisions;
- implementation and review artifacts retain for at least three years to preserve incident investigation evidence;
- trace links retain for five years to keep immutable pointers to external systems and long-lived compliance reports.

## Provenance Capture Contract

All artifact classes must capture the base provenance contract:

- `run_id`
- `orchestrator`
- `runtime`
- `model`
- `gate_results`
- `phase`
- `role`
- `captured_at`
- `source_issue_id`
- `artifact_id`
- `hash`
- `hash_prev`

The `hash` + `hash_prev` pair establishes an append-only hash chain for artifact lineage.

## Phase-Specific Additions

| Phase | Required classes | Additional provenance keys |
| --- | --- | --- |
| `intake` | `intent-brief` | `trigger` |
| `plan` | `intent-brief`, `execution-plan` | `depends_on` |
| `execute` | `code-diff` | `paths_touched`, `test_targets` |
| `verify` | `verification-report`, `trace-link` | `gate_name`, `gate_status` |
| `review` | `review-verdict` | `reviewer_role`, `decision` |
| `publish` | `review-verdict`, `trace-link` | `decision`, `pr_url` |

## Gate Signal Registry

| Gate signal | Phase | Purpose |
| --- | --- | --- |
| `intake:issue-scoped` | `intake` | Confirms issue scope and acceptance criteria are captured before planning. |
| `intake:risk-assessed` | `intake` | Confirms risk posture and lane ownership are declared. |
| `plan:dependencies-resolved` | `plan` | Confirms dependencies and execution order are resolved. |
| `execute:diff-prepared` | `execute` | Confirms implementation diff exists with touched-path/test-target evidence. |
| `verify:tests-passed` | `verify` | Confirms required tests passed for the issue scope. |
| `verify:boundary-ok` | `verify` | Confirms boundary policy checks passed with no forbidden writes. |
| `verify:evidence-contract-pass` | `verify` | Confirms strict evidence contract validation passed. |
| `review:decision-recorded` | `review` | Confirms reviewer decision and rationale are captured. |
| `review:risk-signoff` | `review` | Confirms risk signoff is complete for publication eligibility. |
| `publish:pr-gate-pass` | `publish` | Confirms PR gate checks passed for publication transition. |
| `publish:callback-published` | `publish` | Confirms PR callback payload was published to callback stream. |

## Phase Transition Prerequisites

| Transition | Required gate signals | Required artifact classes | Required provenance keys |
| --- | --- | --- | --- |
| `intake` -> `plan` | `intake:issue-scoped`, `intake:risk-assessed` | `intent-brief` | `trigger` |
| `plan` -> `execute` | `plan:dependencies-resolved` | `execution-plan` | `depends_on` |
| `execute` -> `verify` | `execute:diff-prepared` | `code-diff` | `paths_touched`, `test_targets` |
| `verify` -> `review` | `verify:tests-passed`, `verify:boundary-ok`, `verify:evidence-contract-pass` | `verification-report`, `trace-link` | `gate_name`, `gate_status` |
| `review` -> `publish` | `review:decision-recorded`, `review:risk-signoff`, `publish:pr-gate-pass`, `publish:callback-published` | `review-verdict`, `trace-link` | `reviewer_role`, `decision`, `pr_url` |

## Gate Aggregation and Transition Policy Contract

`internal/artifact/gate_policy_contract.go` defines the strict adjudication contract that the transition controller must enforce.

- aggregation contract version: `gate-aggregation/v1`
- mode: `all-required-signals-pass`
- pass statuses: `pass`
- blocking statuses: `fail`, `missing`
- unknown status handling: `block`

The transition policy contract (`transition-policy/v1`) is derived from `TransitionPrerequisites()` and carries, per phase edge:

- required gate signals
- required artifact classes
- required provenance keys
- deterministic denial reason codes (`missing-gate-signal`, `gate-not-passed`, `missing-artifact`, `missing-provenance-key`)

This keeps design inputs and runtime enforcement aligned: intake defines prerequisites and the policy contract projects those prerequisites into a machine-enforceable transition rule set.

`internal/artifact/transition_controller.go` applies that contract at runtime. For a requested phase edge, it:

- adjudicates required gate signals with the strict aggregation policy (`pass` required; `fail`, `missing`, and unknown statuses block);
- verifies required artifact classes exist in the issue artifact stream;
- verifies required provenance keys are present in payloads for required artifact classes;
- emits deterministic denial reason codes in policy order (`missing-gate-signal`, `gate-not-passed`, `missing-artifact`, `missing-provenance-key`) plus per-signal gate decisions for traceability.

## Implementation Baseline

- `internal/artifact/intake.go` is the canonical intake map for class IDs, retention windows, and provenance requirements.
- `internal/artifact/intake_test.go` enforces unique class IDs, retention coverage, phase-to-class validity, gate signal integrity, and transition prerequisite consistency.
- `internal/artifact/gate_policy_contract.go` builds strict gate aggregation semantics and transition policy rules from intake prerequisites.
- `internal/artifact/gate_policy_contract_test.go` verifies strict aggregation defaults, prerequisite parity, and deterministic transition denial reason codes.
- `internal/artifact/transition_controller.go` enforces policy-backed transition gating against issue artifact intake evidence.
- `internal/artifact/transition_controller_test.go` verifies allowed transitions, denied transitions, and deterministic denial reason sequencing.
- Hash-chain and append-only storage contract is formalized in `docs/ARTIFACT_PROVENANCE_HASH_CHAIN_CONTRACT.md`.
- Artifact bus ingestion/retrieval lifecycle and provenance index APIs are implemented in `internal/artifact/bus_service.go`.
- Tamper/retention verification checks for audit gates are implemented in `internal/artifact/bus_verify.go`.
- PR shipping checklist, migration steps, and evidence access examples are documented in `docs/ARTIFACT_BUS_PR_SHIPPING_CHECKLIST.md`.
