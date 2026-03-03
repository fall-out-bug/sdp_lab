# Resource Management & Leak Findings (Fifth Review)

**Review Date:** 2026-03-03  
**Scope:** `internal/` directory  
**Focus:** Resource leaks, improper cleanup, resource exhaustion

---

## Summary

| Severity | Count |
|----------|-------|
| CRITICAL | 1 |
| HIGH     | 2 |
| MEDIUM   | 6 |
| LOW      | 3 |

---

## CRITICAL

### RL5-001: StuckDetector.Stop() double-close panic on stopCh

**File:** `/home/fall_out_bug/projects/vibe_coding/sdp_lab/internal/monitor/stuck_detector.go`  
**Lines:** 99-105

**Issue:** `Stop()` calls `close(sd.stopCh)` without guarding against multiple invocations. Calling `Stop()` twice (e.g., from shutdown handlers or tests) causes `panic: close of closed channel`.

**Code:**
```go
func (sd *StuckDetector) Stop() {
	close(sd.stopCh)  // Panics if already closed
	if sd.checkTicker != nil {
		sd.checkTicker.Stop()
	}
}
```

**Fix:** Use `sync.Once` to ensure close runs only once:
```go
var stopOnce sync.Once
func (sd *StuckDetector) Stop() {
	sd.stopOnce.Do(func() {
		close(sd.stopCh)
	})
	if sd.checkTicker != nil {
		sd.checkTicker.Stop()
	}
}
```
Add `stopOnce sync.Once` to the struct.

**Beads:** Map to `sdplab-*` for tracking.

---

## HIGH

### RL5-002: defer inside loop causes file handle retention until function return

**File:** `/home/fall_out_bug/projects/vibe_coding/sdp_lab/internal/evidence/auto_attest.go`  
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

### RL5-003: Context cancellation not respected in Scheduler background task

**File:** `/home/fall_out_bug/projects/vibe_coding/sdp_lab/internal/planner/scheduler.go`  
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

### RL5-004: autofixer context.WithTimeout cancel not deferred — panic leaks context

**File:** `/home/fall_out_bug/projects/vibe_coding/sdp_lab/internal/ciloop/autofixer.go`  
**Lines:** 145-160

**Issue:** `runCtx, cancel := context.WithTimeout(ctx, timeout)` is created in a loop. `cancel()` is called on all normal paths (continue/fall-through), but if `cmd.Run()` panics, `cancel()` is never invoked. The context and its timer goroutine leak until GC.

**Code:**
```go
runCtx, cancel := context.WithTimeout(ctx, timeout)
// ... if cmd.Run() panics, cancel never runs
if runErr := cmd.Run(); runErr != nil {
	cancel()
	continue
}
cancel()
```

**Fix:** Add `defer cancel()` immediately after `WithTimeout`:
```go
runCtx, cancel := context.WithTimeout(ctx, timeout)
defer cancel()
```

**Beads:** Map to `sdplab-*` for tracking.

---

### RL5-005: attest.go collectWorkstreamScopePrefixes — file close not deferred, panic leaks

**File:** `/home/fall_out_bug/projects/vibe_coding/sdp_lab/internal/orchestrate/attest.go`  
**Lines:** 192-220

**Issue:** `os.Open(wsPath)` in loop, `f.Close()` at end of iteration. If `scanner.Scan()` or any code in the loop panics, the file handle is never closed.

**Code:**
```go
for _, wsID := range wsIDs {
	f, err := os.Open(wsPath)
	if err != nil { continue }
	// ... scanner.Scan() loop ...
	_ = f.Close()  // Never reached on panic
}
```

**Fix:** Use `defer` with closure to capture loop variable:
```go
for _, wsID := range wsIDs {
	f, err := os.Open(wsPath)
	if err != nil { continue }
	defer func(f *os.File) { _ = f.Close() }(f)
	// ... use f ...
}
```
Note: This reintroduces defer-in-loop retention. Prefer closing in a helper or using explicit close at end of iteration with recover if panic handling is needed.

**Alternative (preferred):** Extract file read into a helper that opens, uses, and closes in one scope:
```go
func readScopePrefixes(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil { return nil, err }
	defer f.Close()
	// ... scan and return
}
```

**Beads:** Map to `sdplab-*` for tracking.

---

### RL5-006: policy.go tmpInput — missing defer Close on panic path

**File:** `/home/fall_out_bug/projects/vibe_coding/sdp_lab/internal/orchestrate/policy.go`  
**Lines:** 56-65

**Issue:** `tmpInput` is created with `os.CreateTemp`. On write error, `tmpInput.Close()` is called explicitly. On success, `tmpInput.Close()` at line 65. If `queryOPAString` or `queryOPAStringSet` panics before line 65, the file handle leaks. `defer os.Remove` runs but `Close` is not deferred.

**Fix:** Add `defer tmpInput.Close()` immediately after successful `CreateTemp`:
```go
tmpInput, err := os.CreateTemp("", "sdp-policy-input-*.json")
if err != nil { ... }
defer tmpInput.Close()
defer func() { _ = os.Remove(tmpInput.Name()) }()
```

**Beads:** Map to `sdplab-*` for tracking.

---

### RL5-007: PermissionBridge audit file requires explicit Close

**File:** `/home/fall_out_bug/projects/vibe_coding/sdp_lab/internal/guard/permission_bridge.go`  
**Lines:** 131-138

**Issue:** `NewPermissionBridge` opens an audit log file and stores it in `pb.auditFile`. There is a `Close()` method, but if callers never call it (e.g., discard the bridge), the file handle leaks.

**Mitigation:** Document that `Close()` must be called when done. Consider `defer pb.Close()` in typical usage. Tests correctly use `defer pb.Close()`.

**Beads:** Map to `sdplab-*` for tracking.

---

### RL5-008: Session Writer file requires explicit Close/Finalize

**File:** `/home/fall_out_bug/projects/vibe_coding/sdp_lab/internal/session/writer.go`  
**Lines:** 61-72

**Issue:** `NewWriter` opens a log file. The caller must call `Close()` or `Finalize()` when done. If the Writer is discarded without closing, the file handle leaks.

**Mitigation:** Document lifecycle. Callers should use `defer w.Close()` or `defer w.Finalize(...)` after creation.

**Beads:** Map to `sdplab-*` for tracking.

---

### RL5-009: SQLClient has no Close — relies on parent Client

**File:** `/home/fall_out_bug/projects/vibe_coding/sdp_lab/internal/beads/sql_client.go`  
**Lines:** 16-18

**Issue:** `SQLClient` wraps `*Client.db` and has no `Close()` method. Lifecycle depends on the parent `Client.Close()`. If `SQLClient` is used without the parent `Client` being closed, or if references are held after `Client` is closed, behavior is undefined.

**Mitigation:** Document that `SQLClient` shares the DB with `Client`; closing `Client` invalidates `SQLClient`. Consider documenting ownership clearly.

**Beads:** Map to `sdplab-*` for tracking.

---

## LOW

### RL5-010: exec.Command (non-Context) used in several places

**Files:**  
- `/home/fall_out_bug/projects/vibe_coding/sdp_lab/internal/evidence/auto_attest.go` (180, 254, 266, 390)  
- `/home/fall_out_bug/projects/vibe_coding/sdp_lab/internal/orchestrate/attest.go` (252, 257, 266)  
- `/home/fall_out_bug/projects/vibe_coding/sdp_lab/internal/orchestrate/policy.go` (88, 102)  
- `/home/fall_out_bug/projects/vibe_coding/sdp_lab/internal/orchestrate/advance.go` (40)  
- `/home/fall_out_bug/projects/vibe_coding/sdp_lab/internal/guard/scope_check.go` (72)  
- `/home/fall_out_bug/projects/vibe_coding/sdp_lab/internal/orchestrate/hydrate_sources.go` (12, 28, 38)

**Issue:** `exec.Command` (without `CommandContext`) does not support context cancellation. Long-running or stuck subprocesses cannot be cancelled. Prefer `exec.CommandContext(ctx, ...)` where cancellation is desired.

**Beads:** Map to `sdplab-*` for tracking.

---

### RL5-011: HTTP clients created per provider — no shared connection pool

**Files:**  
- `/home/fall_out_bug/projects/vibe_coding/sdp_lab/internal/modelgateway/adapters/selfhosted.go` (45)  
- `/home/fall_out_bug/projects/vibe_coding/sdp_lab/internal/bridge/github_findings.go` (124)  
- `/home/fall_out_bug/projects/vibe_coding/sdp_lab/internal/evidence/sigstore_signer.go` (55)

**Issue:** Each provider/bridge creates its own `*http.Client`. For high-throughput scenarios, a shared client with tuned `Transport` and connection pooling may be more efficient. Not a leak, but can contribute to connection churn.

**Beads:** Map to `sdplab-*` for tracking.

---

### RL5-012: executeWithTimeout — verify ExecuteBranch respects context

**File:** `/home/fall_out_bug/projects/vibe_coding/sdp_lab/internal/orchestrate/parallel_executor.go`  
**Lines:** 181-205

**Issue:** When `timeoutCtx.Done()` fires, the function returns. The goroutine running `ExecuteBranch` may still be running. With buffered channels (cap 1), the send typically succeeds and the goroutine exits. If `ExecuteBranch` does not respect context cancellation and runs for a long time, the goroutine will eventually block on send. Lower risk due to buffering but worth verifying that `ExecuteBranch` honors `timeoutCtx`.

**Mitigation:** Ensure the branch executor checks `ctx.Done()` and returns promptly when cancelled.

**Beads:** Map to `sdplab-*` for tracking.

---

## Verified OK (no change from prior review)

| Pattern | Location | Notes |
|---------|----------|-------|
| Ticker stopped on run exit | `internal/monitor/stuck_detector.go:107-112` | `defer` in `run()` stops ticker before return |
| HTTP response body closed | `internal/modelgateway/adapters/selfhosted.go:101` | `defer resp.Body.Close()` after err check |
| HTTP response body closed | `internal/evidence/sigstore_signer.go:345` | `defer resp.Body.Close()` after err check |
| File closed in loop | `internal/orchestrate/attest.go:146` | `defer f.Close()` in `lookupBeadsIDsForFeature` |
| File closed | `internal/monitor/stuck_detector.go:196` | `defer f.Close()` in `getLastEventTime` |
| sql.DB Close | `internal/beads/client.go:74-76` | `Close()` method exists |
| Context WithTimeout cancel | `internal/orchestrate/parallel_executor.go:182-183` | `defer cancel()` |
| Context WithTimeout cancel | `internal/verify/quorum.go:170-171` | `defer cancel()` |
| Rows.Close | `internal/beads/*.go` | `defer rows.Close()` on all Query results |
| Quorum goroutines on timeout | `internal/verify/quorum.go` | Buffered channels; verifiers complete and exit |

---

## Changes from Prior Review (RESOURCE_LEAK_FINDINGS.md)

- **RL-001 (Ticker not stopped):** Verified FIXED — `run()` has `defer` that stops ticker before return.
- **RL5-001 (NEW):** StuckDetector double-close panic — not in prior review.
- **RL5-004 (NEW):** autofixer context cancel on panic — not in prior review.
- **RL5-005 (NEW):** attest.go panic path file leak — not in prior review.

---

## Next Steps

1. Create Beads issues for each finding (RL5-001 through RL5-012).
2. Prioritize CRITICAL (RL5-001) and HIGH (RL5-002, RL5-003) for immediate fix.
3. Add `defer`/`Close` patterns to code review checklist.
4. Consider static analysis (e.g., `staticcheck`, `govet`) for resource leaks in CI.
