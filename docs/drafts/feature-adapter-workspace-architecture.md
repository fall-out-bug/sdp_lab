# FR-018: Adapter Workspace Architecture

## Problem Statement

adapter-controller (sdp-adapter namespace) uses `emptyDir` for `/workspaces`, causing:
- **Beads:** `bd` CLI calls fail — no `.beads/issues.jsonl` in empty volume
- **Evidence:** `.sdp/evidence/` lost on pod restart
- **Traces:** `.sdp/runs/` lost on pod restart
- **Multi-project:** no access to per-project workspaces (`/workspaces/<project_id>/`)

Meanwhile, swarm-orchestrator and feature-orchestrator in sdp-control share `swarm-workspaces` PVC.

## Design Reference

Full analysis: `docs/plans/2026-02-22-adapter-workspace-design.md`

## Decisions

| Aspect | Decision | Rationale |
|--------|----------|-----------|
| Storage | Shared swarm-workspaces PVC | Single source of truth, no sync |
| Namespace | adapter → sdp-control | Control-plane component, same PVC access |
| Per-project | WorkspaceResolver(projectID) → workDir | Per-call BeadsAdapter/EvidenceProjector |
| Graceful degradation | Startup health check, BeadsAdapter=nil | Works without bd/workspace |
| Beads CLI | Add bd + beads-fsm to image | Short-term; Go API long-term |

## Beads

- **Epic:** sdp_dev-1es — FR-018: Adapter workspace architecture (P1)

## Workstreams

| WS | Bead | Title | Priority | Effort | Dependencies |
|----|------|-------|----------|--------|--------------|
| WS-018-01 | sdp_dev-lab | Graceful degradation: workspace health check | P1 | 0.5d | — |
| WS-018-02 | sdp_dev-2tm | Migrate adapter to sdp-control + PVC | P1 | 1d | WS-018-01 |
| WS-018-03 | sdp_dev-lft | Per-project workspace routing | P2 | 0.5d | WS-018-02 |
| WS-018-04 | sdp_dev-oiw | Dockerfile: add bd + beads-fsm to adapter image | P1 | 0.5d | WS-018-02 |
| WS-018-05 | sdp_dev-dfz | Kustomize overlays (dev/staging/prod) | P3 | 0.5d | WS-018-02 |

## Success Criteria

- adapter-controller starts cleanly with and without workspace (graceful degradation)
- Beads operations work when PVC mounted with initialized workspace
- Evidence survives pod restart
- Multi-project: correct workDir per AgentRun/Task labels
- E2E on minikube passes with and without beads
