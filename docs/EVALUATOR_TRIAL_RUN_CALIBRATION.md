# Evaluator Trial-Run Calibration

This document captures the trial-run simulation and calibration evidence format delivered in `sdp_dev-hx0.1.7`.

## Trial-Run Methodology

- Run deterministic fixture set from `DefaultTrialRunFixtures()`.
- For each fixture:
  - build persona execution packet with `BuildPersonaExecutionPacket(...)`
  - assemble persona score report with `AssembleSwarmScoreReport(...)`
  - rank opportunities with `RankImprovementOpportunities(...)`
  - evaluate recommendation quality gates from calibration thresholds
- Aggregate all run outcomes into `BuildTrialRunCalibrationReport(...)`.

## Default Calibration Thresholds

| Threshold | Default | Intent |
| --- | --- | --- |
| `min_consensus_rate_percent` | `75` | Require consensus in most trial runs. |
| `min_average_persona_score` | `74` | Avoid recommending low-confidence runs. |
| `min_top_opportunity_score` | `80` | Ensure top recommendation quality bar. |
| `max_missing_persona_responses` | `1` | Keep coverage gaps bounded. |
| `min_run_quality_pass_rate_percent` | `75` | Require recommendation gate pass in most runs. |

## Calibration Evidence Format

Calibration output uses `TrialRunCalibrationReport`:

- `run_results[]`: per-run consensus, average persona score, missing persona count, top recommendation score, and failed checks.
- `consensus_rate_percent`: aggregate percent of runs with swarm consensus.
- `run_quality_pass_rate_percent`: aggregate percent of runs passing recommendation quality gates.
- `threshold_evidence[]`: aggregate metric rows with `observed`, `threshold`, and `passed` fields.
- `overall_gate_passed`: final calibration verdict from aggregate threshold evidence.

## Implementation and Tests

- Calibration helper: `internal/evaluator/calibration.go`.
- Regression tests: `internal/evaluator/calibration_test.go`.

Calibration aggregate verdict (`overall_gate_passed`) is consumed by continuous-improvement PR loop guardrails in `internal/evaluator/pr_loop.go`.
