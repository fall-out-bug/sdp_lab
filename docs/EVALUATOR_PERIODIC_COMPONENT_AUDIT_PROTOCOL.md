# Evaluator Periodic Component Audit Protocol

This document captures the repeatable audit protocol delivered in `sdp_dev-hx0.1.3`.

## Protocol Scope

- Establish a deterministic loop for periodic component audits.
- Normalize required audit inputs before persona scoring starts.
- Define explicit decision checkpoints and escalation behavior when a checkpoint fails.

## Cadence

- Cadence contract: `weekly-or-change-triggered`.
- Cadence is sourced from `DefaultDeepThinkingSwarmPlan()` so the audit loop and swarm orchestration stay aligned.

## Required Inputs

The protocol requires the following input bundle before a run can pass:

- `component-boundary-map`
- `component-change-log`
- `dependency-risk-delta`
- `incident-signal-digest`
- `outcome-metric-delta`
- `persona-score-report`

## Decision Checkpoints

| Checkpoint | Decision question | Pass condition | Escalation on failure |
| --- | --- | --- | --- |
| `entry-gate` | Are intake/trigger conditions complete for this audit run? | `all-intake-triggers-pass` | `systems-architect` |
| `evidence-gate` | Is evidence complete across reliability, security, DX, and outcomes? | `all-persona-evidence-present` | `dx-expert`, `security-reviewer`, `sre` |
| `decision-gate` | Is consensus or explicit dissent handling present for recommendations? | `consensus-threshold-met-or-dissent-published` | `product-strategist`, `systems-architect` |
| `handoff-gate` | Can accepted recommendations be converted into implementation tasks? | `backlog-items-created-with-trace-link` | `product-strategist` |

## Escalation Policy

- Escalation path: `checkpoint-owner -> issue-owner-with-dissent-log`.
- Multiple failed checkpoints are merged into a deduplicated escalation target set.
- Any missing required input or failed checkpoint marks the run as escalation-required.

## Implementation and Tests

- Contract helper: `internal/evaluator/audit_protocol.go`.
- Core API:
  - `DefaultPeriodicComponentAuditProtocol()`
  - `EvaluatePeriodicAuditDecision(...)`
- Regression tests: `internal/evaluator/audit_protocol_test.go`.

## Acceptance Mapping for `sdp_dev-hx0.1.3`

- Cadence is explicit and deterministic.
- Input bundle is fixed and test-validated.
- Decision checkpoints are explicit and test-validated.
- Escalation behavior and target derivation are deterministic and test-validated.
