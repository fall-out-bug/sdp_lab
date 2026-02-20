# Multi-Role Operator Orchestration (SDP)

Status: draft design for O1-03 and O2-01

## Objective

Run multiple agent roles in Kubernetes via kubeopencode, while preserving SDP semantics:

- deterministic run lifecycle
- traceable role outputs
- reviewer gate before final PR publication

## Roles

- `analyst`: requirement decomposition and risk notes
- `coder`: implementation proposal and patch plan
- `reviewer`: synthesis, quality/risk verdict, next-action recommendation

## Orchestration model

1. Orchestrator creates a run ID (`run-<timestamp>`).
2. Spawn `analyst` and `coder` Tasks in parallel.
3. Wait for both to reach terminal phase (`Completed|Failed`).
4. Collect both outputs from task logs.
5. Spawn `reviewer` Task with aggregated context.
6. Persist run trace (`run-id`, task names, phases, outputs, verdict).

This keeps role execution in operator resources, while orchestration and policy stay in SDP.

## Inter-agent communication contract

Communication is mediator-based (orchestrator hub), not direct pod-to-pod chat.

Envelope format (embedded in task logs):

```json
{
  "run_id": "run-20260219-2300",
  "role": "analyst",
  "status": "ok",
  "summary": "...",
  "artifacts": [
    {"type": "risk_note", "content": "..."},
    {"type": "task_breakdown", "content": "..."}
  ]
}
```

Rules:

- each role emits one final envelope
- reviewer receives analyst/coder envelopes as input context
- orchestrator stores final trace for audit and PR linkage

## Why mediator pattern

- deterministic and auditable flow
- no direct network permissions between role pods
- easier policy enforcement at a single control point
- portable to upstream operator with minimal assumptions

## Mapping to SDP

- beads/FSM remains source of truth for issue progression
- kubeopencode Task/Agent CRDs become execution substrate
- strict evidence includes role envelopes and run trace links

## Immediate prototype scope

- install kubeopencode in `kubeopencode-system`
- create 3 Agents (`analyst`, `coder`, `reviewer`)
- run one probe with parallel analyst/coder + sequential reviewer
- capture logs and emit run summary

## Verification and recovery increment

The `internal/oneshot` package now includes deterministic verifier/recovery helpers for hands-off orchestration:

- `VerifyRoleEvidence(manifest, evidence)` checks per-task evidence completeness, role/status validity, and reviewer dependency artifact references.
- `PlanFailureRecovery(manifest, failedTaskIDs)` computes deterministic requeue scope (failed tasks plus transitive downstream dependents).

These helpers support injected failure drills and make automatic retry planning auditable before reviewer and PR stages.

## OneShot operational SLOs

Targets are evaluated on rolling 7-day windows for runs labeled `workstream:oneshot-swarm-orchestrator`.

- verification coverage SLO: >= 99% of runs emit both `verification.oneshot.report` and `oneshot_verification`
- recovery determinism SLO: 100% of failed runs emit `oneshot_recovery.requeue_task_ids` with at least one task id
- gate health SLO: >= 95% of runs pass `go test ./...` (`verification.go_test_passed=true`) before reviewer approval
- delivery latency SLO: p95 from `in_progress` to `review` <= 20 minutes per issue

## Escalation policy (OneShot)

- severity 1 (immediate human intervention):
  - two consecutive runs missing any of `verification.oneshot.*`, `oneshot_verification`, or issue-note `kind=oneshot_verify`
  - any run where verification fails and no `oneshot_recovery` plan is emitted
- severity 2 (same-day escalation):
  - gate health SLO falls below 95% over 24h
  - delivery latency p95 exceeds 20 minutes for 3 consecutive runs
- required actions on escalation:
  - set impacted Beads task to `blocked`
  - append issue note with machine-readable incident payload (`kind=oneshot_escalation`)
  - link failing run/evidence artifact paths and recovery decisions

## PR-shipping evidence checklist (OneShot)

For task `sdp_dev-2aq.15.4`, include these artifacts/links in PR description and issue notes:

- run artifact: `.sdp/runs/<issue-id>.json` with `oneshot_verification` and, on failures, `oneshot_recovery`
- strict evidence artifact: `.sdp/evidence/<issue-id>.json` with
  - `verification.go_test_passed`
  - `verification.oneshot.evidence_ok`
  - `verification.oneshot.report`
  - `verification.oneshot.role_evidence`
  - `verification.oneshot.recovery_plan` (when applicable)
- issue-note linkage: machine-readable note payload with `kind=oneshot_verify`
- quality gates: test command output (`go test ./cmd/swarm-worker`, `go test ./internal/oneshot`, `go test ./...`)
- publish linkage: `pr-gate` outputs and final `trace.pr_url` value in evidence + Beads notes
