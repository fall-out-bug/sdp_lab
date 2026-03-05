# Resource Management & Leak Findings

**Review Date:** 2026-03-02  
**Scope:** `internal/` directory  
**Focus:** Resource leaks, improper cleanup, resource exhaustion

---

## Summary

| Severity | Count |
|----------|-------|
| CRITICAL | 1 |
| HIGH     | 2 |
| MEDIUM   | 5 |
| LOW      | 2 |

---

## CRITICAL

### RL-001: Ticker not stopped when StuckDetector run loop exits on context cancellation

**File:** `internal/monitor/stuck_detector.go`  
**Lines:** 109-118

**Issue:** When `run()` returns on `ctx.Done()` or `sd.stopCh`, it does not call `checkTicker.Stop()`. The `time.Ticker` keeps an internal goroutine running indefinitely. If callers cancel context without explicitly calling `Stop()`, the ticker goroutine leaks.

**Code:**
```go
case <-ctx.Done():
    return  // Ticker never stopped
case <-sd.stopCh:
    return  // Ticker never stopped
```

**Fix:** Stop the ticker before returning:
```go
case <-ctx.Done():
    sd.checkTicker.Stop()
    return
case <-sd.stopCh:
    sd.checkTicker.Stop()
    return
```

**Beads:** Map to `sdplab-*` for tracking.

---

## HIGH

### RL-002: defer inside loop causes file handle retention until function return

**File:** `internal/evidence/auto_attest.go`  
**Lines:** 337-366

**Issue:** `defer func(f *os.File) { _ = f.Close() }(f)` is inside a `for` loop. Each iteration opens a file and defers its close. All defers run only when the function returns. With N workstream files, N file handles stay open for the entire loop duration. Can exhaust process file descriptor limits.

**Code:**
```go
for _, e := range entries {
    f, err := os.Open(filepath.Join(backlogDir, e.Name()))
    if err != nil { continue }
    defer func(f *os.File) { _ = f.Close() }(f)  // BAD: defers accumulate
    // ... use f ...
}
```

**Fix:** Close immediately after use:
```go
for _, e := range entries {
    f, err := os.Open(filepath.Join(backlogDir, e.Name()))
    if err != nil { continue }
    // ... use f ...
    _ = f.Close()
}
```

**Beads:** Map to `sdplab-*` for tracking.

---

### RL-003: Context cancellation not respected in Scheduler background task

**File:** `internal/planner/scheduler.go`  
**Lines:** 139-148

**Issue:** `startTask` spawns a goroutine that uses `context.Background()` instead of the parent `ctx`. When the plan is cancelled via `CancelPlan`, the background task ignores cancellation and continues running. Goroutine cannot be stopped; may complete work that is no longer needed.

**Code:**
```go
go func() {
    err := s.executor.Execute(context.Background(), task)  // Ignores parent ctx
    // ...
}()
```

**Fix:** Pass parent context (or a child with cancellation) so the executor can respect cancellation:
```go
go func() {
    err := s.executor.Execute(ctx, task)
    // ...
}()
```
Ensure the executor implementation checks `ctx.Done()` during long-running work.

**Beads:** Map to `sdplab-*` for tracking.

---

## MEDIUM

### RL-004: PermissionBridge audit file requires explicit Close

**File:** `internal/guard/permission_bridge.go`  
**Lines:** 131-138

**Issue:** `NewPermissionBridge` opens an audit log file and stores it in `pb.auditFile`. There is a `Close()` method, but if callers never call it (e.g., discard the bridge), the file handle leaks.

**Mitigation:** Document that `Close()` must be called when done. Consider `defer pb.Close()` in typical usage or adding a finalizer (not recommended). Tests correctly use `defer pb.Close()`.

**Beads:** Map to `sdplab-*` for tracking.

---

### RL-005: Session Writer file requires explicit Close/Finalize

**File:** `internal/session/writer.go`  
**Lines:** 61-72

**Issue:** `NewWriter` opens a log file. The caller must call `Close()` or `Finalize()` when done. If the Writer is discarded without closing, the file handle leaks.

**Mitigation:** Document lifecycle. Callers should use `defer w.Close()` or `defer w.Finalize(...)` after creation.

**Beads:** Map to `sdplab-*` for tracking.

---

### RL-006: Temp file in policy.go — missing defer Close on error path

**File:** `internal/orchestrate/policy.go`  
**Lines:** 56-65

**Issue:** `tmpInput` is created with `os.CreateTemp`. On write error (line 61-63), `tmpInput.Close()` is called explicitly. On success, `tmpInput.Close()` is called at line 65. If a panic occurs between 61 and 65, or if `queryOPAString`/`queryOPAStringSet` panics before line 65, the file handle could leak. The `defer os.Remove` runs but `Close` is not deferred.

**Fix:** Add `defer tmpInput.Close()` immediately after successful `CreateTemp`:
```go
tmpInput, err := os.CreateTemp("", "sdp-policy-input-*.json")
if err != nil { ... }
defer tmpInput.Close()
defer func() { _ = os.Remove(tmpInput.Name()) }()
```

**Beads:** Map to `sdplab-*` for tracking.

---

### RL-007: SQLClient has no Close — relies on parent Client

**File:** `internal/beads/sql_client.go`  
**Lines:** 16-18

**Issue:** `SQLClient` wraps `*Client.db` and has no `Close()` method. Lifecycle depends on the parent `Client.Close()`. If `SQLClient` is used without the parent `Client` being closed, or if references are held after `Client` is closed, behavior is undefined.

**Mitigation:** Document that `SQLClient` shares the DB with `Client`; closing `Client` invalidates `SQLClient`. Consider embedding or documenting ownership clearly.

**Beads:** Map to `sdplab-*` for tracking.

---

### RL-008: executeWithTimeout goroutine — potential block on slow ExecuteBranch

**File:** `internal/orchestrate/parallel_executor.go`  
**Lines:** 182-205

**Issue:** When `timeoutCtx.Done()` fires, the function returns immediately. The goroutine running `ExecuteBranch` may still be running. With buffered channels (cap 1), the send typically succeeds and the goroutine exits. If `ExecuteBranch` does not respect context cancellation and runs for a long time, the goroutine will eventually block on send (if the caller has stopped receiving). Lower risk due to buffering but worth verifying that `ExecuteBranch` honors `timeoutCtx`.

**Mitigation:** Ensure the branch executor checks `ctx.Done()` and returns promptly when cancelled.

**Beads:** Map to `sdplab-*` for tracking.

---

## LOW

### RL-009: exec.Command (non-Context) used in several places

**Files:** `internal/evidence/auto_attest.go` (179, 254, 266), `internal/orchestrate/attest.go` (252, 257, 266), `internal/orchestrate/policy.go` (88, 102), `internal/orchestrate/advance.go` (40), `internal/guard/scope_check.go` (72), others

**Issue:** `exec.Command` (without `CommandContext`) does not support context cancellation. Long-running or stuck subprocesses cannot be cancelled. Prefer `exec.CommandContext(ctx, ...)` where cancellation is desired.

**Beads:** Map to `sdplab-*` for tracking.

---

### RL-010: HTTP clients created per provider — no shared connection pool

**Files:** `internal/modelgateway/adapters/selfhosted.go` (45), `internal/bridge/github_findings.go` (124), `internal/evidence/sigstore_signer.go` (55)

**Issue:** Each provider/bridge creates its own `*http.Client`. For high-throughput scenarios, a shared client with tuned `Transport` and connection pooling may be more efficient. Not a leak, but can contribute to connection churn.

**Beads:** Map to `sdplab-*` for tracking.

---

## Verified OK

| Pattern | Location | Notes |
|---------|----------|-------|
| HTTP response body closed | `internal/modelgateway/adapters/selfhosted.go:101` | `defer resp.Body.Close()` |
| HTTP response body closed | `internal/evidence/sigstore_signer.go:345` | `defer resp.Body.Close()` |
| File closed in loop | `internal/orchestrate/attest.go:219` | Explicit `f.Close()` in `collectWorkstreamScopePrefixes` |
| File closed with defer | `internal/orchestrate/attest.go:147` | `defer f.Close()` in `lookupBeadsIDsForFeature` |
| File closed | `internal/monitor/stuck_detector.go:191` | `defer f.Close()` in `getLastEventTime` |
| sql.DB Close | `internal/beads/client.go:74-76` | `Close()` method exists |
| Context WithTimeout cancel | `internal/orchestrate/parallel_executor.go:182-183` | `defer cancel()` |
| Context WithTimeout cancel | `internal/verify/quorum.go:170-171` | `defer cancel()` |
| Rows.Close | `internal/beads/*.go` | `defer rows.Close()` on all Query results |

---

## Next Steps

1. Create Beads issues for each finding (RL-001 through RL-010).
2. Prioritize CRITICAL and HIGH for immediate fix.
3. Add `defer`/`Close` patterns to code review checklist.
4. Consider static analysis (e.g., `staticcheck`, `govet`) for resource leaks in CI.
