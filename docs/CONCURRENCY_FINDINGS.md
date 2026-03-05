# Concurrency & Thread Safety Findings

**Review Date:** 2026-03-02  
**Scope:** `internal/` directory  
**Focus:** Race conditions, deadlocks, synchronization issues

---

## Summary

| Severity | Count |
|----------|-------|
| CRITICAL | 2 |
| HIGH     | 4 |
| MEDIUM   | 3 |
| LOW      | 2 |

---

## CRITICAL

### C1. StuckDetector.Stats() returns mutable map reference — data race

**File:** `internal/monitor/stuck_detector.go:266-274`

**Issue:** `Stats()` returns `Stats{StuckSessions: sd.stuckSessions}` — the actual internal map reference. Callers receive a mutable reference to the map. After the lock is released, concurrent modification by `check()` (in the goroutine) and read/write by the caller causes a data race.

```go
func (sd *StuckDetector) Stats() Stats {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	return Stats{
		StuckCount:    len(sd.stuckSessions),
		StuckSessions: sd.stuckSessions,  // BUG: returns live reference
		Timeout:       sd.timeout,
	}
}
```

**Fix:** Copy the map before returning:
```go
result := make(map[string]time.Time)
for k, v := range sd.stuckSessions {
	result[k] = v
}
return Stats{StuckCount: len(sd.stuckSessions), StuckSessions: result, Timeout: sd.timeout}
```

---

### C2. Scheduler.CancelPlan accesses plan.tasks without holding Plan mutex

**File:** `internal/planner/scheduler.go:153-169`

**Issue:** `CancelPlan` iterates over `plan.tasks` and modifies `task.Status` without holding `plan.mu`. The goroutine launched by `startTask` calls `plan.BlockTask`/`plan.CompleteTask`, which hold `plan.mu` and modify the same map. Concurrent map iteration and map write causes panic or data race.

```go
func (s *Scheduler) CancelPlan(ctx context.Context, planID string) error {
	plan, err := s.GetPlan(planID)  // releases lock
	// ...
	for _, task := range plan.tasks {  // BUG: no plan.mu held
		if task.Status == TaskStatusInProgress {
			// ...
		}
	}
}
```

**Fix:** Hold `plan.mu` during iteration:
```go
plan.mu.Lock()
defer plan.mu.Unlock()
for _, task := range plan.tasks {
	// ...
}
```

---

## HIGH

### H1. BeadsSink — map and stats without mutex protection

**File:** `internal/bridge/beads_sink.go`

**Issue:** `findings map[string]bool` and `stats SyncStats` are accessed from `SyncProtocolFindings`, `SyncDocsFindings`, `LoadExistingFindings`, `GetStats`, `PrintSummary`, `GenerateReport` without any mutex. If called concurrently (e.g. from different goroutines in a CI pipeline or polling mode), map concurrent write and non-atomic stats updates cause data races.

**Lines:** 30, 44-46, 49-66, 72-84, 86-96, 108-109, 136-137, 150-151, 179-180

**Fix:** Add `sync.RWMutex` to `BeadsSink` and protect all access to `findings` and `stats`.

---

### H2. ProviderRegistry — maps without mutex protection

**File:** `internal/modelgateway/provider.go:118-163`

**Issue:** `providers` and `factories` maps are accessed without synchronization. `Register`, `RegisterFactory`, `Get`, `List`, `CreateProvider`, `HealthCheck` can be called concurrently (e.g. during startup with multiple providers, or during request routing). Concurrent map read/write causes panic.

**Fix:** Add `sync.RWMutex` to `ProviderRegistry` and protect all map access.

---

### H3. MappingFile.entries — map without mutex (if used concurrently)

**File:** `internal/beads/client.go:184-227`

**Issue:** `MappingFile.entries` is a map with no mutex. `GetSDPID` reads it. If `LoadMapping` populates and other goroutines call `GetSDPID` during population, or if a future API allows concurrent read/write, race occurs. Currently usage appears single-threaded; document as potential issue if API is extended.

**Severity:** HIGH if used from multiple goroutines; otherwise MEDIUM.

---

### H4. FormulaParser.searchPaths — slice append without mutex

**File:** `internal/beads/formula_parser.go:129-131`

**Issue:** `AddSearchPath` appends to `p.searchPaths` without a mutex. `FindFormula` (and `ParseDir`) iterate over `searchPaths`. Concurrent `AddSearchPath` and `FindFormula` causes slice concurrent read/write.

**Fix:** Add `sync.RWMutex` to `FormulaParser` or document single-threaded usage.

---

## MEDIUM

### M1. PolicyRouter.decisionLog — slice append under lock but returned configs may be shared

**File:** `internal/modelgateway/router.go:159`

**Issue:** `decisionLog` is appended under `r.mu.Lock()`. The `Route` method holds the lock for the entire duration including policy evaluation. If policy evaluation is slow, this blocks all routing. Consider RWMutex with shorter critical sections.

**Severity:** MEDIUM — correct but may cause contention.

---

### M2. sync.Once not used for one-time initialization

**File:** Various

**Issue:** No `sync.Once` usage found in `internal/`. One-time init patterns (e.g. lazy provider creation, singleton caches) that may be called from multiple goroutines could benefit from `sync.Once` to avoid redundant work or races. No specific instance identified; consider for future lazy init patterns.

**Severity:** LOW (informational).

---

### M3. Writer.Sequence() — lock held for read of single int

**File:** `internal/session/writer.go:194-198`

**Issue:** `Sequence()` uses `Lock()` for reading an int. For high-contention reads, `RWMutex` with `RLock` would allow concurrent reads. Current usage (single writer) makes this minor.

**Severity:** LOW.

---

## LOW

### L1. Session writer test — unbuffered channel

**File:** `internal/session/writer_test.go:173`

**Issue:** `done := make(chan bool)` is unbuffered. Ten goroutines send, main receives ten times. No deadlock. OK.

**Status:** No action needed.

---

### L2. All channels in internal/ are buffered

**Files:** `parallel_executor.go`, `quorum.go`, `stuck_detector.go`

**Finding:** All `make(chan ...)` calls use buffered channels. No unbuffered channel deadlock risk identified.

---

## WaitGroup Usage — OK

**Files:** `internal/orchestrate/parallel_executor.go:130-176`, `internal/verify/quorum.go:176-194`

**Finding:** WaitGroup is used correctly: `Add(1)` before goroutine launch, `Done()` in defer. No misuse detected.

---

## Structs with Goroutines — Synchronization Summary

| Struct              | Goroutines                    | Mutex        | Status                          |
|---------------------|-------------------------------|--------------|---------------------------------|
| StuckDetector       | `run()` in Start()            | sync.Mutex   | BUG: Stats returns map ref (C1) |
| ParallelExecutor    | Execute() worker goroutines   | sync.RWMutex | OK                              |
| Quorum              | Execute() verifier goroutines  | sync.RWMutex | OK                              |
| Scheduler           | startTask() executor          | sync.RWMutex | BUG: CancelPlan race (C2)       |
| Session Writer      | None (caller may use async)   | sync.Mutex   | OK                              |

---

## Recommended Next Steps

1. **Immediate:** Fix C1 (StuckDetector.Stats) and C2 (Scheduler.CancelPlan).
2. **Short-term:** Add mutex to BeadsSink (H1) and ProviderRegistry (H2).
3. **Validation:** Run `go test ./internal/... -race` in CI.
4. **Beads mapping:** Create one Beads issue per finding for tracking.
