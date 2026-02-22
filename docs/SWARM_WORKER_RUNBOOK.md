# Swarm Worker Runbook (Private)

## Purpose

`cmd/swarm-worker` executes the workflow for a claimed autonomy task. It invokes autonomy-worker to claim, applies workstream-specific changes, runs tests, updates evidence, and publishes PRs.

## Modular layout

| File | Responsibility |
|------|----------------|
| `main.go` | Entry point, orchestration flow |
| `main_runner.go` | run, runComponent, parseClaim, loadIssue, hasLabel, discardBeadsSyncNoise, hasStagedChanges |
| `main_flow.go` | resolveWorkstream, applyWorkstreamFlow, commitBodyForWorkstream, writePRBody |
| `main_handlers.go` | applyEvaluatorRecommendationWorkstream, applySelfImprovementWorkstream, applyGenericWorkstream, appendHandoffValidationTimestamp |
| `main_patches.go` | patchSlugifyForTrim, addSlugifyRegressionTest, patchModelChainUnknownFallback, addModelChainRegressionTest, patchRiskK8sHigh, addRiskK8sRegressionTest |
| `main_ensure_*.go` | ensurePlannerEnvelopeFiles, ensureOneShotManifestFiles, ensureTelegramIntakeFiles |
| `main_verify.go` | updateEvidence |
| `main_verify_oneshot.go` | evaluateOneShotVerification, applyOneShotVerification |
| `main_observability.go` | emitWorkerObservability, extractLinkage |
| `main_util.go` | toStringSlice, hasPrefixAny, uniqueStrings |

## Workflow

1. Invokes `autonomy-worker` to claim next task
2. Parses claim (issue_id, title, model, branch)
3. Loads issue detail, resolves workstream from labels
4. Git checkout branch
5. Applies workstream flow (patches, ensure files, or handlers)
6. Runs `go test ./...`
7. Updates `.sdp/evidence/<issue-id>.json`
8. Runs beads-fsm, pr-gate
9. Git add, commit, push
10. Publishes PR via pr-publish

## Supported workstreams

- `policy-slugify-trim`, `model-chain-default-fallback`, `policy-k8s-risk-high`
- `telegram-ingress-intake`, `planner-boundary-decomposition`, `oneshot-swarm-orchestrator`
- `handoff-validation`, `generic`, `self-improvement`, `evaluator-recommendation`

## Usage

```bash
go run ./cmd/swarm-worker
```

Requires: autonomy-worker already claimed a task; `bd` in PATH; git repo; `GH_TOKEN` for pr-publish.

## Expected output

JSON with `issue`, `branch`, `status` (e.g. `review` or `blocked`).
