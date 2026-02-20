# Parallel Lock Domain Intake

This intake captures the first-pass concurrency hazard catalog for the autonomy stream and maps each hazard to lock domains that the scheduler must serialize.

## Hazard Catalog

| Hazard key | Trigger examples | Required lock domain(s) |
| --- | --- | --- |
| `repo-tree-conflict` | edits under `internal/`, `cmd/`, `docs/`, `specs/`, `scripts/`, `deploy/` | `repo-tree`, `branch-ref` |
| `beads-state-race` | writes under `.beads/` | `beads-state` |
| `evidence-trace-interleave` | writes under `.sdp/evidence/` | `evidence-store` |
| `cluster-rollout-collision` | rollouts in `deploy/`, `scripts/apply_*`, `scripts/orchestrate_k8s_*` | `k8s-control-plane` |

## Implementation Baseline

- `internal/parallel/locks.go` provides deterministic hazard detection from touched paths.
- `BuildLockRequests` emits sorted lock requests suitable for scheduler preflight checks.
- Branch lock scope is set to the active branch name to prevent concurrent ref updates.

## Notes for Next Design Step

- This intake establishes lock domains and trigger rules only.
- Scheduler policy (ordering, starvation prevention, fairness) is intentionally deferred to `sdp_dev-2aq.19.2`.
