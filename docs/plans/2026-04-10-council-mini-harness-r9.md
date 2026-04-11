# LLM Council Report: SDP Mini-Harness Design — Round 9

**Date:** 2026-04-10  
**Round:** 9 (v9 verification)  
**Spec version:** v9 → v10 (post Round 9 fixes applied)  
**Consensus:** NOT_READY → 3 fixes → v10 ready for Round 10  
**Quorum:** 5/6 ✅ (architect + critic + technician + pragmatist + engineer)

---

## Quorum

| Role | Model | Status | max_tokens |
|------|-------|--------|-----------|
| Architect | codex-rescue | ✓ Active | — |
| Critic | google/gemini-3.1-pro-preview | ✓ Active (partial) | 1500 |
| Technician | deepseek/deepseek-v3.2 | ✓ Active | 1500 |
| Philosopher | moonshotai/kimi-k2.5 | ✗ ABSTAIN (finish=length) | 5000 (needs 10000+) |
| Pragmatist | minimax/minimax-m2.7 | ✓ Active | 4000 |
| Engineer | xiaomi/mimo-v2-pro | ✓ Active (partial) | 4500 |

---

## Fix Verification (U1, U2)

| Fix | Technician | Pragmatist | Engineer | Verdict |
|-----|-----------|-----------|---------|---------|
| U1 (Stop durable-first) | CORRECT | CORRECT | CORRECT | ✅ |
| U2 (BuildLoopConfig before param) | CORRECT | CORRECT | CORRECT | ✅ |

---

## New Issues Found (Round 9)

### V1 HIGH — hStateStopped missing (Critic)

**Severity:** HIGH  
**Location:** `harnessState` constants / `Harness.Stop()`

After `Stop()`, `h.state = hStateIdle` — the harness can be called again with `RunPhase()`. No terminal guard exists. Any caller with a valid token can restart execution on a stopped session.

**Fix in v10:**
```go
const (
    hStateIdle          harnessState = iota
    hStateRunning
    hStateAwaitingHuman
    hStateStopped       // Fix V1 (v10): terminal — prevents reuse after Stop()
)

// Stop() sets: h.state = hStateStopped
// RunPhase guard (state != hStateIdle) automatically rejects hStateStopped
```

---

### V2 MEDIUM — beforeToolCall not restored in RestoreHarness (Technician)

**Severity:** MEDIUM  
**Location:** `RestoreHarness()` signature

`RestoreHarness` restored `ownerToken` (S2) but not `beforeToolCall`. After restart, the pre-execution hook is silently lost.

**Fix in v10:**
```go
func RestoreHarness(
    sessionID, ownerToken string,
    store SessionStore,
    router *PhaseRouter,
    gate *GateEngine,
    beforeToolCall func(name string, args json.RawMessage) error,
) (*Harness, error) {
    // ...
    h := &Harness{
        // ...
        ownerToken:     ownerToken,
        beforeToolCall: beforeToolCall, // Fix V2: restored
    }
```

---

### V3 HIGH — ContextManager.Trim() never called in Run() (Engineer)

**Severity:** HIGH  
**Location:** `Loop.Run()` / `ContextManager` interface

`ContextManager` is defined and referenced in `LoopConfig` (I6), but `Run()`'s implementation never calls `cfg.ContextManager.Trim()`. For long sessions (many TurnRecords rebuilt into messages), the LLM call will exceed context window limits.

**Fix in v10:** Documented in `Run()` pseudo-code:
```go
// Before each LLM call in the loop:
if cfg.ContextManager != nil {
    msgs, err = cfg.ContextManager.Trim(msgs, cfg.Model, cfg.MaxTokens)
    if err != nil { /* emit error event, close channel */ return }
}
// Then: resp, err := llm.Call(ctx, msgs, cfg)
```
`cfg.ContextManager == nil` → messages passed through unchanged (MVP passthrough).

---

### V4 MEDIUM — Terminal PhaseRecord loses PendingDecision context (Pragmatist) — DOCUMENTATION

**Assessment:** Pragmatist raised that the terminal PhaseRecord snapshot doesn't include the PendingDecision reference. On recovery, if ClearDecision failed, RestoreHarness sees both NextPhase="" and a pending decision. Resolution: documented that NextPhase="" is the canonical stop signal and takes precedence. PendingDecision is cleared at next Stop() call (idempotent). **Not a bug — documentation clarification only.** Convergence: READY.

---

### V5 LOW — PersistEvent errors silently ignored (Pragmatist)

**Assessment:** Events are secondary telemetry; TurnRecord is the canonical log. `PersistEvent` failure does not affect FSM correctness, phase transitions, or durable state. For MVP, log-and-continue is acceptable. **Not a bug. Acceptable for MVP.** 

---

## Convergence Analysis

| Round | Spec | Issues | Fixed | New CRITICAL/HIGH |
|-------|------|--------|-------|-------------------|
| 0–1 | v1→v2 | 7 | 7 | 5 |
| 2 | v2→v3 | 5 | 5 | 2 |
| 3 | v3→v4 | 7 | 7 | 5 |
| 4 | v4→v5 | 5 | 5 | 3 |
| 5 | v5→v6 | 3 | 3 | 0 |
| 6 | v6→v7 | 8 | 8 | 4 |
| 7 | v7→v8 | 2 | 2 | 1 |
| 8 | v8→v9 | 2 | 2 | 1 |
| 9 | v9→v10 | 3 | 3 | 2 |

**Total: 42 issues found, 42 fixed. Issues per round: 7→5→7→5→3→8→2→2→3.**

---

## Convergence Signals

| Model | Verdict | Notes |
|-------|---------|-------|
| Technician | READY | V2 minor, doesn't block |
| Pragmatist | READY | V4/V5 minor doc issues |
| Engineer | NOT_READY | V3 HIGH ContextManager |
| Critic | NOT_READY | V1 HIGH hStateStopped |

All V1-V3 fixes applied in v10. Round 10 should produce full convergence.

---

## Round 10 Plan

Verify V1-V3 fixes. User requests max_tokens=50000 — will use 10000+ for kimi and mimo to achieve 6/6 quorum. 

*Raw responses: `/tmp/council_r9_responses.json`*  
*Design: `docs/plans/2026-04-10-sdp-mini-harness-design.md` (v10)*
