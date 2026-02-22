# Evaluator Intake Baseline

This intake defines the minimum happy-path operability gate and the initial evaluator scope for `sdp_dev-hx0.1` before continuous evaluator runs begin.

## Happy-Path Operability Gate

The gate is satisfied only when every prerequisite below is true:

| Check ID | Requirement |
| --- | --- |
| `issue-selected` | Target issue is selected and moved to `in_progress`. |
| `dependencies-clear` | Blocking dependencies for the selected issue are closed. |
| `scope-baseline-defined` | Baseline evaluator scope components are declared. |
| `gate-command-declared` | Validation command baseline is declared (`go test ./...`). |
| `callback-contract-available` | Publish callback contract is available for evaluator publish-phase evidence. |

The current implementation lives in `internal/evaluator/intake.go` and exposes `EvaluateHappyPathOperability(...)` for deterministic prerequisite evaluation.

## Baseline Evaluator Scope

| Scope ID | Description | Owner signal | Evidence class |
| --- | --- | --- | --- |
| `issue-intake` | Scope, dependencies, and acceptance gates are captured before launch. | `intake:issue-scoped` | `intent-brief` |
| `execution-flow` | Planner/execution path can produce deterministic patches. | `execute:diff-prepared` | `code-diff` |
| `verification-stack` | Verification gates are runnable for evaluator-selected scope. | `verify:tests-passed` | `verification-report` |
| `review-publish-link` | Review/publish handoff emits callback and trace evidence. | `publish:callback-published` | `trace-link` |

## Validation Baseline

- Formatting: `gofmt` over evaluator intake package.
- Regression gate: `go test ./...`.

## Notes for Follow-on Tasks

- `sdp_dev-hx0.1.1` can now consume this intake baseline to define persona roles and deep-thinking workflow details.
- Downstream protocol/rubric tasks should reuse these operability checks as hard prerequisites for trial runs.
