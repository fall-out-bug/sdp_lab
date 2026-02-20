# Parallel Scheduler Policy and Merge Queue Semantics

This document defines the policy contract produced from lock-domain intake for `sdp_dev-2aq.19.2`.

## Lock Hierarchy

To avoid deadlocks, lock requests are canonicalized and acquired in this global order:

1. `beads-state`
2. `evidence-store`
3. `repo-tree`
4. `branch-ref`
5. `k8s-control-plane`

If multiple requests target the same domain, the canonicalizer keeps one entry with deterministic rules:

- `global` scope dominates branch/file scoped requests for that domain.
- Empty scope is normalized to `global`.
- Reason is normalized and deterministic for stable evidence output.

## Merge Queue Classes

Scheduler plans map lock sets into queue classes:

- `concurrent-safe`: no lock requirements.
- `serial-branch`: only branch-scoped `branch-ref` lock(s).
- `serial-global`: any non-branch lock or global branch lock.

This makes branch-only updates run independently while protecting shared resources.

## Starvation Prevention

Priority aging is applied in bounded steps:

- Inputs use Beads priority scale `P0..P4` as `0..4` (lower is higher urgency).
- Every 3 wait cycles raises urgency by one level.
- Aging is capped at `P0` so priorities do not invert beyond the highest class.

This policy guarantees eventual progress for long-waiting tasks without removing urgency bands.

## Implementation Contract

- `internal/parallel/locks.go` remains the hazard intake source.
- `internal/parallel/policy.go` defines canonical lock hierarchy, queue classing, and aging.
- `BuildSchedulerPlan(...)` composes intake + policy into a deterministic artifact for scheduler admission.
- `internal/parallel/manager.go` provides deterministic in-memory lock manager primitives and scheduler admission helpers (`DecideBuildAdmission`, `InMemoryBuildScheduler`) for infra-free tests.

## Verification Methodology

Verification for lock-manager and scheduler behavior uses deterministic in-memory simulations in `internal/parallel/manager_test.go`:

- **Deadlock prevention checks**: conflicting multi-lock requests are rejected atomically; failed acquires must not retain non-conflicting locks (no partial ownership).
- **Race wave checks**: concurrent admission attempts for serial-global work admit exactly one owner per wave while others receive deterministic denials (`global-queue-busy` or `lock-conflict`).
- **Fair progression checks**: denied contenders are retried after release and must each admit in bounded retries, demonstrating queue progression without starvation in retry-driven control loops.

Expected result profile for these simulations:

- Initial contention wave: one admitted contender for a serial-global queue.
- Post-release retries: each previously denied contender eventually admits when retried.
- No leaked ownership: lock state fully clears via `Release(...)` and subsequent acquisitions proceed.

## Operational Safeguards (Incident Playbook)

Use this playbook when runtime behavior around parallel admission deviates from expected outcomes.

### Trigger Signals

- Sustained denials with `global-queue-busy` for unrelated runs over multiple retry waves.
- Repeated `lock-conflict` denials after expected lock release points.
- Serial-branch work denied with `branch-queue-busy` while branch owner is no longer active.
- Missing branch identity causing `serial-branch-requires-branch` denials.

### Immediate Containment

1. Pause new parallel submissions for the affected lane.
2. Capture admission attempts with: queue class, denial reason, branch, and run id.
3. Confirm each admitted run executes `Release(id)` on completion/failure paths.
4. Re-run deterministic checks in `internal/parallel/manager_test.go` to verify lock and queue invariants.

### Diagnostic Path

1. **Queue pressure diagnosis**
   - Map denials to queue class (`concurrent-safe`, `serial-branch`, `serial-global`).
   - If most denials are `global-queue-busy`, validate whether hazard inputs are over-classifying work into global locks.
2. **Lock conflict diagnosis**
   - Inspect requested lock set from `BuildSchedulerPlan(...)` and canonicalized locks from `CanonicalizeLockRequests(...)`.
   - Confirm lock scope normalization did not escalate branch-scoped requests to `global` unexpectedly.
3. **Fairness diagnosis**
   - Validate retry loops provide bounded retries and that denied contenders are retried after current owner release.
   - Use race simulation expectations from `TestInMemoryBuildSchedulerRaceSimulationProgressAndFairRetries` as baseline.

### Recovery Actions

- Reduce contention window by splitting high-conflict work into branch-isolated batches where possible.
- Correct missing/incorrect branch metadata before scheduler admission.
- Ensure completion/failure handlers always invoke scheduler and lock release.
- If required, temporarily serialize a problematic lane while collecting evidence for policy updates.

## Tuning Guidance

Tune only with evidence from admission outcomes and test-backed invariants.

- **Lock scope tuning**: keep lock requests branch-scoped when possible; avoid unnecessary `global` scope because it forces `serial-global` queueing.
- **Hazard mapping tuning**: refine touched-path inputs so `BuildLockRequests(...)` captures real shared-resource hazards without over-locking.
- **Queue-class tuning**: verify lock sets intended for branch isolation remain `branch-ref` only to stay in `serial-branch`.
- **Priority aging tuning**: use `EffectivePriority(basePriority, waitCycles)` as the only urgency adjustment path; increase wait cycles in retries rather than bypassing queue semantics.
- **Retry policy tuning**: retries should happen after observed release events and remain bounded; repeated immediate retries inflate contention without progress.

### Required Test Anchors After Tuning

- `TestInMemoryLockManagerNoPartialAcquireOnConflict`
- `TestInMemoryBuildSchedulerRaceSimulationProgressAndFairRetries`
- `TestInMemoryBuildSchedulerQueueAndLockIntegration`
- `TestAdmitBuildPlanUsesBuildSchedulerPlanAndMergeQueue`

## PR Publication Evidence Checklist (Parallel Controls)

Before opening/updating PRs for parallel-control changes:

1. Documentation updated for lock hierarchy, queue semantics, incident playbook, and tuning guidance.
2. Tests cover lock atomicity, race-wave admission behavior, and retry fairness (`internal/parallel/manager_test.go`).
3. Validation commands recorded with results:
   - `go test ./...`
   - additional targeted checks when scheduler behavior changes (for example `go test -race ./internal/parallel`)
4. Beads issue notes include:
   - deliverables (docs/tests/code touchpoints)
   - validation outputs
   - explicit statement of PR readiness (open/update)
5. Strict evidence is complete for PR gate flow (`docs/PR_GATE_RUNBOOK.md`) before publish.
