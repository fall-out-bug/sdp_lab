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

## Start command (next active work)

```bash
bd update sdp_dev-2aq.6 --status in_progress --json
```
