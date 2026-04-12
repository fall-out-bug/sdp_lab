# UP-001 PR Review: Retry Budget and Terminal Reason

**Status:** PR not yet submitted. Fork `sdp-contrib/kubeopencode` does not exist.  
**Review date:** 2026-02-21  
**Reviewer:** Code review against plan, kubeopencode main, and sdp_dev adapter contract.

---

## Executive Summary

| Aspect | Status | Notes |
|--------|--------|-------|
| **PR exists** | ❌ No | Fork and branch not created |
| **PR body** | ❌ Missing | `docs/upstream/UP-001-pr-body.md` referenced but not present |
| **Spec clarity** | ✅ Good | Plan and patch outline are clear |
| **API design** | ⚠️ Needs validation | Terminal reason taxonomy vs Conditions overlap |
| **Controller impact** | ⚠️ Non-trivial | Retry logic requires Pod recreation, attempt tracking |
| **Tests** | ❌ None | No retry/terminal tests in current kubeopencode |
| **Adapter readiness** | ⚠️ Partial | Adapter expects `failureReason`; kubeopencode does not expose it today |

---

## 1. Gaps and Missing Artifacts

### 1.1 PR Body Missing

The submission command references `docs/upstream/UP-001-pr-body.md`:

```bash
gh pr create ... --body-file docs/upstream/UP-001-pr-body.md
```

**This file does not exist.** PR cannot be submitted without it.

**Action:** Create `docs/upstream/UP-001-pr-body.md` with:
- Problem statement
- Proposed API changes (spec + status)
- Backward compatibility notes
- Upgrade/migration notes
- Test coverage summary

### 1.2 Fork and Branch Not Created

- `sdp-contrib/kubeopencode` — 404 (repo not found)
- Branch `feat/retry-budget-terminal-reason` — cannot be fetched

**Action:** Fork kubeopencode, create branch, implement changes, push, then open PR.

---

## 2. API Design Review

### 2.1 Proposed Spec Additions

```yaml
# Task.spec (optional)
retry:
  maxAttempts: 7
  profile: linear | exponential
```

**Concerns:**
- `profile` values (`linear`, `exponential`) — need concrete delay schedules. Plan mentions "deterministic delays" but no schema.
- `maxAttempts` — does it include the first attempt? (Convention: usually yes; document explicitly.)
- No `retriableReasons` — how does controller know which failures to retry? Today all `PodFailed` are terminal. Need either:
  - Explicit list of retriable exit codes / reasons, or
  - Default: retry all infrastructure failures (Pod OOMKilled, Evicted, etc.), not agent exit codes.

### 2.2 Proposed Status Additions

```yaml
# Task.status
terminalReason:
  code: string    # Normalized taxonomy
  message: string # Human-readable
```

**Concerns:**
- Overlap with `status.conditions[].reason` — kubeopencode already uses `ReasonPodCreationError`, `ReasonAgentError`, etc. Risk of duplication and drift.
- **Recommendation:** Either:
  - Extend `terminalReason` to be the canonical field and document mapping from Conditions, or
  - Add `terminalReason` only when Phase=Failed/Completed and derive from the most relevant Condition.

### 2.3 Phase Naming Mismatch

- **kubeopencode:** `TaskPhaseCompleted` (not Succeeded)
- **sdp_dev adapter:** `PhaseSucceeded` in `lifecycle_reconciler.go`

Adapter contract assumes `status.phase=Succeeded`. kubeopencode uses `Completed`. Ensure adapter maps `Completed` → Succeeded or update contract.

---

## 3. Controller Logic Review

### 3.1 Current Failure Handling

In `updateTaskStatusFromPod`:

```go
case corev1.PodFailed:
  task.Status.Phase = kubeopenv1alpha1.TaskPhaseFailed
  task.Status.CompletionTime = &now
  // No retry, no terminalReason, no attempt tracking
  return r.Status().Update(ctx, task)
```

Today: one-shot failure, no retries.

### 3.2 Retry Implementation Challenges

**Retry = Pod recreation.** To retry:
1. Delete the failed Pod (or let it stay for debugging?)
2. Reset Task status to allow re-initialization (e.g. Phase → "" or Pending?)
3. Increment attempt counter (new field: `status.attempt` or similar)
4. Re-run `initializeTask` to create a new Pod

**Risks:**
- Requeue loops: ensure `attempt` is persisted before creating new Pod.
- Idempotency: if reconcile runs twice, avoid creating duplicate Pods.
- Cleanup: delete old Pod on retry, or keep for forensics? (Recommend: delete to avoid clutter.)

### 3.3 Terminal Reason Population

When to set `terminalReason`:
- **Completed:** Optional. Could use `code: "success"`, `message: "Task completed"`.
- **Failed:** Required. Source of truth:
  - Pod container status: `state.terminated.reason`, `state.terminated.message`, `state.terminated.exitCode`
  - Or Condition reason/message

Need a **normalized taxonomy** (e.g. `InfrastructureError`, `AgentExitNonZero`, `Timeout`, `UserStopped`). Plan says "terminal reason taxonomy" but does not define codes.

---

## 4. Test Coverage

### 4.1 Current State

- `task_controller_test.go`: integration tests for create, agent ref, contexts, capacity, quota, cleanup.
- **No tests for:** retry exhaustion, terminal reason serialization, retriable vs non-retriable failure handling.

### 4.2 Required Tests (from plan)

1. **Retry exhaustion:** Task with `retry.maxAttempts: 2`, force Pod to fail twice → Phase=Failed, `terminalReason` set.
2. **Terminal reason publication:** Task fails → `status.terminalReason.code` and `message` populated.
3. **Backward compatibility:** Task without `retry` spec → behavior unchanged (no retries, single attempt).
4. **Default passthrough:** Omitted `retry` → same as today (no retries).

---

## 5. Potential "Костыли" (Hacks) and Smells

### 5.1 Retry State Machine

If retry logic is bolted onto existing flow without a clear state machine, expect:
- Phase transitions: `Running` → `Failed` (Pod failed) → `???` (retry) → `Running` (new Pod).
- Using `Phase == ""` to mean "re-initialize" is fragile. Consider `PhaseRetrying` or an explicit `status.retryAttempt` that drives behavior.

### 5.2 Condition vs terminalReason Duplication

Setting both `Conditions` and `terminalReason` for the same failure risks:
- Divergent messages
- Extra update round-trips

**Recommendation:** Set `terminalReason` from the same source as the primary Failure condition, in one status update.

### 5.3 Profile Implementation

"Linear" and "exponential" need concrete definitions. Plan references SDP's `standard-15m`: `5s, 15s, 30s, 60s, 120s, 240s, 420s`. That is neither pure linear nor exponential. Clarify:
- `linear`: fixed delay (e.g. 30s) between attempts?
- `exponential`: 2^n * base (e.g. 5s, 10s, 20s, 40s)?

---

## 6. Adapter Integration

### 6.1 Adapter Expectation

`ReconcilePhase` receives `failureReason string`:

```go
func (r *LifecycleReconciler) ReconcilePhase(currentFSM FSMState, crdPhase CRDPhase, failureReason string) (FSMState, string, error)
```

Today, kubeopencode does not expose a structured `failureReason`. Adapter would need to read:
- `status.conditions[].message`, or
- `status.terminalReason.message` (after UP-001)

**Action:** After UP-001, adapter should prefer `status.terminalReason` when present.

### 6.2 Contract Rule

Adapter contract: `"source":"status.phase=Failed","target":"fsm.blocked_or_escalated","rule":"retry budget with terminal reason taxonomy"`

UP-001 provides the "terminal reason taxonomy" in Task status. Adapter can then map `terminalReason.code` to `blocked` vs `escalated`.

---

## 7. Recommendations

### Before Submitting PR

1. **Create `docs/upstream/UP-001-pr-body.md`** with full PR description.
2. **Define terminal reason taxonomy** (code enum + mapping from Pod/Condition reasons).
3. **Define retry profiles** (linear, exponential) with concrete delay formulas.
4. **Clarify retriable vs non-retriable** — document which failures trigger retry.
5. **Implement and test** in a fork, then open PR.

### Implementation Order

1. Add CRD fields (spec + status) with validation.
2. Add `terminalReason` population on failure (no retry yet) — validates status contract.
3. Add retry loop with attempt tracking and Pod recreation.
4. Add profile-based backoff.
5. Add integration tests for retry exhaustion and terminal reason.

### Risks to Watch

- **Controller complexity:** Retry logic increases cyclomatic complexity. Consider extracting `shouldRetry(task, pod) (bool, error)` and `nextRetryDelay(attempt, profile) time.Duration`.
- **Upgrade path:** Existing Tasks without `retry` must behave identically. Default `maxAttempts: 1` or omit and treat as 1.

---

## 8. Checklist Before PR Submit

- [ ] Fork `kubeopencode` → `sdp-contrib/kubeopencode` (or org of choice)
- [ ] Create branch `feat/retry-budget-terminal-reason`
- [ ] Implement spec fields: `retry.maxAttempts`, `retry.profile`
- [ ] Implement status fields: `terminalReason.code`, `terminalReason.message`
- [ ] Implement retry loop in controller
- [ ] Add unit/integration tests for retry and terminal reason
- [ ] Create `docs/upstream/UP-001-pr-body.md`
- [ ] Run `make test` and `make lint` in kubeopencode
- [ ] Verify no SDP-private logic (model allowlist, risk classes, tenant metadata)
- [ ] Open PR with `--body-file docs/upstream/UP-001-pr-body.md`
