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

## Implementation Baseline

- `internal/artifact/intake.go` is the canonical intake map for class IDs, retention windows, and provenance requirements.
- `internal/artifact/intake_test.go` enforces unique class IDs, retention coverage, and phase-to-class validity.
- Hash-chain and append-only storage contract is formalized in `docs/ARTIFACT_PROVENANCE_HASH_CHAIN_CONTRACT.md`.
- Artifact bus ingestion/retrieval lifecycle and provenance index APIs are implemented in `internal/artifact/bus_service.go`.
- Tamper/retention verification checks for audit gates are implemented in `internal/artifact/bus_verify.go`.
- PR shipping checklist, migration steps, and evidence access examples are documented in `docs/ARTIFACT_BUS_PR_SHIPPING_CHECKLIST.md`.
