# UP-001 Work Plan: KubeOpenCode Retry + Terminal Reason

**Parent:** sdp_dev-4py (PR: submit upstream kubeopencode PR UP-001)  
**Epic:** sdp_dev-j2b (Rollout validation and upstream contribution)

## Task decomposition

| Beads ID | Title | Agent lane | Workstream | Deps |
|----------|-------|------------|------------|------|
| sdp_dev-9j3 | UP-001: Create kubeopencode fork and branch | explore | kubeopencode-upstream | — |
| sdp_dev-vwo | UP-001: Add Task spec retry fields (CRD) | commit | kubeopencode-upstream | sdp_dev-9j3 |
| sdp_dev-3gj | UP-001: Add Task status terminalReason + retryAttempt | commit | kubeopencode-upstream | sdp_dev-vwo |
| sdp_dev-dzf | UP-001: Implement controller retry loop | commit | kubeopencode-upstream | sdp_dev-vwo |
| sdp_dev-2j7 | UP-001: Add retry/terminalReason tests | commit | kubeopencode-upstream | sdp_dev-3gj, sdp_dev-dzf |
| sdp_dev-542 | UP-001: Fix adapter PhaseCompleted vs PhaseSucceeded | commit | kubeopencode-upstream | — |

## Agent assignment

- **UP-001-1** (fork/branch): Human or agent with gh/kubectl — requires GitHub fork.
- **UP-001-2..5**: K8s agents (autonomy-worker) — run in kubeopencode fork repo.
- **UP-001-6**: sdp_dev agent — change in `internal/adapter/lifecycle_reconciler.go`.

## Labels for autonomy

```
autonomy, strict-evidence, lane:commit, model:glm-5, workstream:kubeopencode-upstream, risk:medium
```

## Execution order

1. **UP-001-6** (adapter fix) — can run in sdp_dev now, no fork needed.
2. **UP-001-1** — fork kubeopencode, create `feat/retry-budget-terminal-reason`.
3. **UP-001-2** → **UP-001-3** → **UP-001-4** → **UP-001-5** — sequential in fork.
4. **sdp_dev-4py** — open PR after UP-001-5 passes.

## References

- [KUBEOPENCODE_UP_001_PR_REVIEW.md](archive/k8s/KUBEOPENCODE_UP_001_PR_REVIEW.md)
- [KUBEOPENCODE_UPSTREAM_PR_CANDIDATE_PLAN.md](archive/k8s/KUBEOPENCODE_UPSTREAM_PR_CANDIDATE_PLAN.md)
- [docs/upstream/UP-001-pr-body.md](upstream/UP-001-pr-body.md)
