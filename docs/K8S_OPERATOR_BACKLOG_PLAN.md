# Kubernetes Operator Backlog Plan

This document captures the operator design direction and links it to Beads backlog items.

## Decision

Build the Kubernetes operator after orchestrator production hardening is complete.

## Why sequence this way

- Current orchestrator already proves worker/reviewer end-to-end behavior.
- Operator should codify a stable flow, not replace a moving target.
- Hardening first reduces controller complexity and rollback risk.

## Target architecture (Operator phase)

- **CRD:** `AgentRun`
  - `spec`: `issueId`, `repo`, `baseBranch`, `model`, `lane`, `workstream`, `timeoutSec`
  - `status`: `phase`, `conditions`, `workerJob`, `reviewerJob`, `prUrl`, `lastError`
- **Controller reconcile loop:**
  - validate `AgentRun`
  - create worker `Job`
  - wait and collect result/artifacts
  - create reviewer `Job`
  - publish terminal status (`Succeeded`/`Failed`/`Blocked`)
- **Platform assets:** RBAC, manifests/packaging, runbook, observability hooks.

## Backlog mapping (Beads)

- `sdp_dev-2aq.6` - O1-03: Productionize k8s orchestrator loop (must finish first)
- `sdp_dev-2aq.7` - F-O2: Kubernetes operator for agent spawning
  - `sdp_dev-2aq.7.3` - O2-01: Define AgentRun CRD and status conditions
  - `sdp_dev-2aq.7.1` - O2-02: Implement controller reconcile loop for worker/reviewer Jobs
  - `sdp_dev-2aq.7.2` - O2-03: Add RBAC, Helm/kustomize manifests, and runbook

## Dependency policy

- `sdp_dev-2aq.6` blocks `sdp_dev-2aq.7`.
- `sdp_dev-2aq.6` blocks `sdp_dev-2aq.7.3`.

## Adapter-to-Operator Handoff Checklist

Exit criteria (all required before switching the primary execution path to operator reconcile):

- k8s orchestrator loop demonstrates stable terminal outcomes (`closed|blocked|timeout`) across at least 10 consecutive production-like runs.
- Every run emits `.sdp/runs/orchestrate-*.json` with run ID, phase transitions, and terminal reason.
- Strict evidence for each issue remains valid (`cmd/pr-gate` passes and reviewer trace includes PR URL when applicable).
- Duplicate dispatch prevention is active for explicit issue orchestration (same issue cannot trigger concurrent explicit cycles).

Compatibility constraints (must remain true during migration):

- Beads/FSM transitions remain source of truth for task state (`open -> in_progress -> closed|blocked`).
- Existing worker/reviewer binaries and contract files remain reusable by operator jobs without changing strict-evidence schema.
- Runtime policy stack (`go-first`, `strict-evidence`, model allowlist) is unchanged by controller adoption.

Rollback triggers (immediate fallback to adapter path):

- Operator reconcile creates duplicate worker/reviewer jobs for a single run ID or issue.
- Operator cannot preserve PR/evidence trace linkage in issue notes and `.sdp/evidence` artifacts.
- Failure rate for terminal runs increases above baseline for two consecutive validation windows.
- Any regression that blocks `bd sync`/FSM transitions or violates strict evidence gates.

## Start command (next active work)

```bash
bd update sdp_dev-2aq.6 --status in_progress --json
```
