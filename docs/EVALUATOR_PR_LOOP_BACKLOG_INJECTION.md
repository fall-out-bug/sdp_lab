# Evaluator Continuous-Improvement PR Loop and Backlog Injection

This document defines the automation increment delivered in `sdp_dev-hx0.1.6` for turning calibrated evaluator outcomes into deterministic PR-ready backlog injection plans.

## Objective

- Reuse evaluator runtime and rubric outputs as source artifacts for implementation planning.
- Keep improvement-to-PR flow deterministic so repeated runs generate stable injection plans.
- Enforce guardrails before suggesting backlog creation for continuous improvement loops.

## Source Artifacts Reused

The loop consumes existing evaluator artifacts directly:

- `TrialRunCalibrationReport` from `BuildTrialRunCalibrationReport(...)` (`internal/evaluator/calibration.go`)
- `SwarmScoreReport` from `AssembleSwarmScoreReport(...)` (`internal/evaluator/swarm_runtime.go`)
- ranked opportunities from `RankImprovementOpportunities(...)` (`internal/evaluator/rubric.go`)

## Contracts

- PR loop report contract: `deep-thinking-improvement-pr-loop/v1`
- Backlog injection plan contract: `deep-thinking-backlog-injection-plan/v1`

Implemented in `internal/evaluator/pr_loop.go` via:

- `DefaultPRLoopGuardrails()`
- `BuildContinuousImprovementPRLoopReport(...)`
- `BuildBacklogInjectionPlan(...)`

## Deterministic Backlog Injection Plan Format

`BacklogInjectionPlan` includes:

- `source_issue_id`
- `source_contract_versions[]` (calibration, runtime report, rubric)
- `eligible_opportunity_ids[]` (sorted)
- `injected_items[]` with deterministic issue fields:
  - `opportunity_id`
  - `normalized_score`
  - `target_issue_title`
  - `target_issue_type`
  - `target_issue_priority`
  - `target_issue_labels[]`
  - `source_recommendations[]`
  - `required_evidence_signals[]`
- `excluded_opportunities[]` with machine-readable reasons

Determinism rules:

- opportunities are sorted by normalized score descending, then opportunity ID ascending
- recommendation excerpts are sliced from already deterministic runtime priority order
- exclusion rows are sorted by opportunity ID + reason
- no wall-clock timestamps are embedded in plan payloads

## Guardrails

Default guardrails enforce:

- calibration aggregate gate pass
- runtime consensus reached
- no missing persona responses
- minimum opportunity score (`>= 80`)
- complete rubric dimensions (no missing or unknown dimensions)
- maximum injected item limit (`3`)

Common exclusion reasons:

- `score-below-minimum`
- `missing-rubric-dimensions`
- `unknown-rubric-dimensions`
- `max-injected-items-reached`

## Validation

- `internal/evaluator/pr_loop_test.go` verifies:
  - deterministic PR-ready report generation on default calibrated artifacts
  - issue ID validation behavior
  - calibration failure blocks PR readiness
  - deterministic guardrail exclusions and max-injection limits
