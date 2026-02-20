# Evaluator Outcome Scoring Rubric

This document defines the weighted scoring rubric delivered in `sdp_dev-hx0.1.5` for ranking framework improvement opportunities.

## Objective

- Score evaluator findings against consistent outcome dimensions.
- Normalize weighted totals into a deterministic `0..100` rank signal.
- Preserve auditability by reporting missing and unknown scoring dimensions.

## Dimension Weights

| Dimension ID | Weight | Intent |
| --- | --- | --- |
| `reliability` | 40 | Prioritize stability, incident containment, and rollback confidence. |
| `security` | 25 | Prioritize exploit resistance and policy-compliant handling of sensitive paths. |
| `delivery` | 20 | Prioritize lead-time and throughput improvements that keep PR flow healthy. |
| `developer-experience` | 15 | Prioritize maintainability ergonomics and predictable operator execution. |

Total weight: `100`.

## Normalization and Ranking Rules

- Input scores are clamped into `0..100` before weighting.
- Weighted score is the sum of `clamped_score * dimension_weight`.
- Normalized score is `weighted_score / 100` (integer division).
- Missing rubric dimensions are recorded per opportunity and implicitly contribute `0`.
- Unknown dimension IDs are reported for evidence hygiene.
- Final rank ordering sorts by normalized score descending, then opportunity ID ascending.

## Implementation and Tests

- Contract helper: `internal/evaluator/rubric.go`.
- Core APIs:
  - `DefaultOutcomeScoringRubric()`
  - `RankImprovementOpportunities(...)`
- Regression tests: `internal/evaluator/rubric_test.go`.

Trial-run calibration evidence now composes this rubric via:

- `BuildTrialRunCalibrationReport(...)` in `internal/evaluator/calibration.go`
- deterministic simulation fixtures from `DefaultTrialRunFixtures()`
- methodology and evidence schema in `docs/EVALUATOR_TRIAL_RUN_CALIBRATION.md`

Continuous-improvement PR loop automation now reuses ranked rubric outputs via:

- `BuildContinuousImprovementPRLoopReport(...)` in `internal/evaluator/pr_loop.go`
- deterministic backlog injection payloads from `BuildBacklogInjectionPlan(...)`
- guardrail and plan-format contract in `docs/EVALUATOR_PR_LOOP_BACKLOG_INJECTION.md`

## PR Shipping Checklist Increment

Rubric runs should attach these artifacts before PR publication:

- [x] weighted dimension table (with total weight = 100)
- [x] ranked opportunity list with normalized scores
- [x] missing/unknown dimension report for each ranked opportunity
- [x] `gofmt` run on touched Go files
- [x] `go test ./...` pass evidence
