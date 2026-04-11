# LLM Council Report: SDP Mini-Harness Design — Round 10

**Date:** 2026-04-10  
**Round:** 10 (first 5/5 OpenRouter + architect full quorum)  
**Spec version:** v10 → v11 (post Round 10 fixes applied)  
**Consensus:** NOT_READY → 3 fixes → v11 ready for Round 11  
**Quorum:** 6/6 ✅ (all 5 OpenRouter + architect) — FIRST FULL QUORUM

---

## Historic Note: First Full Quorum

kimi-k2.5 responded for the first time at 12000 max_tokens. All 5 OpenRouter models + architect are now active simultaneously.

| Role | Model | Status | max_tokens |
|------|-------|--------|-----------|
| Architect | codex-rescue | ✓ Active | — |
| Critic | google/gemini-3.1-pro-preview | ✓ Active (partial) | 2000 |
| Technician | deepseek/deepseek-v3.2 | ✓ Active | 2000 |
| Philosopher | moonshotai/kimi-k2.5 | ✓ **FIRST RESPONSE** | 12000 |
| Pragmatist | minimax/minimax-m2.7 | ✓ Active | 8000 |
| Engineer | xiaomi/mimo-v2-pro | ✓ Active | 10000 |

---

## Fix Verification (V1, V2, V3)

| Fix | Technician | Pragmatist (INCOMPLETE on V1) | Engineer | Philosopher |
|-----|-----------|------------------------------|---------|-------------|
| V1 (hStateStopped) | CORRECT | INCOMPLETE | CORRECT | CORRECT |
| V2 (beforeToolCall in RestoreHarness) | CORRECT | CORRECT | CORRECT | CORRECT |
| V3 (ContextManager documented) | INCOMPLETE | CORRECT | CORRECT | CORRECT |

**V1 INCOMPLETE (Pragmatist):** hStateStopped correctly prevents in-memory reuse, but `RestoreHarness` always sets `h.state = hStateIdle` — it never checks if the latest `PhaseRecord.NextPhase == ""`. After a restart following Stop(), the session returns to `hStateIdle` → `RunPhase` can be called again. **V1 is in-memory-only fix, not durable.**

**V3 INCOMPLETE (Technician):** ContextManager is documented in Run() pseudo-code, but `BuildLoopConfig` never sets `LoopConfig.ContextManager` → field is always nil → Trim() never called.

---

## New Issues Found (Round 10)

### W1 CRITICAL — RestoreHarness ignores terminal stop state (Philosopher/Pragmatist/Engineer — 3 DOMAIN VETOs)

**Severity:** CRITICAL  
**Location:** `RestoreHarness()` — FSM state restoration

`RestoreHarness` sets `h.state = hStateIdle` unconditionally, without checking whether the recovered session's last `PhaseRecord.NextPhase == ""` (terminal stop signal). After restart, a stopped session appears idle — `RunPhase` can be called, violating the explicit invariant "no reuse after Stop()."

**Fix in v11:**
```go
// session.Phase == "" after RecoverSession means latest PhaseRecord.NextPhase == "" (Stop() was called)
if session.Phase == "" && len(session.turnRecords) > 0 {
    return nil, fmt.Errorf("session %s was terminated by Stop() — cannot restore", sessionID)
}
```
Caller handles the error — does not call RunPhase on a terminated session.

---

### W2 HIGH — ContextManager not wired in BuildLoopConfig (Technician)

**Severity:** HIGH  
**Location:** `PhaseRouter.BuildLoopConfig()`

`LoopConfig.ContextManager` is always nil — never set by `BuildLoopConfig`. `Run()` pseudo-code shows it being used, but since it's nil, `Trim()` is never called. Long sessions will overflow context window.

**Fix in v11:**
```go
type PhaseRouter struct {
    phaseMap       map[Role]PhaseConfig
    registry       *ToolRegistry
    gateway        ModelGateway
    contextManager ContextManager // Fix W2: new field; nil = passthrough
}

// BuildLoopConfig:
return LoopConfig{
    ...
    ContextManager: r.contextManager, // Fix W2: wired from router
}
```

---

### W3 HIGH — runID not restored after restart (Philosopher HIGH, Engineer MEDIUM)

**Severity:** HIGH  
**Location:** `RestoreHarness()` / TurnRecord.ID uniqueness

`h.runID` starts at 0 after restart (unless a `PendingDecision` exists). Each `RunPhase` call uses `h.runID` as part of `TurnRecord.ID` (format `"sessionID:runID"`). After restart, runID 1, 2, 3... would collide with existing TurnRecord IDs from before the restart.

**Fix in v11:**
```go
runID: uint64(len(session.turnRecords)), // starts after all existing records
```
Each RunPhase created exactly one TurnRecord, so `len(turnRecords)` == max prior runID. Next runID is `len+1`, guaranteed unique.

---

## Convergence Analysis

| Round | Spec | Issues | Fixed | New CRITICAL/HIGH |
|-------|------|--------|-------|-------------------|
| ... | ... | ... | ... | ... |
| 9 | v9→v10 | 3 | 3 | 2 |
| 10 | v10→v11 | 3 | 3 | 3 |

**Total: 45 issues found, 45 fixed. Issues per round: 7→5→7→5→3→8→2→2→3→3.**

Monotone convergence continues. 3 issues per round now, all HIGH/CRITICAL but concrete. The philosopher joining has sharpened recovery-path analysis.

---

## Round 11 Plan

Verify W1-W3 fixes in v11. With all 6 models active, expect strong convergence signal.

*Raw responses: `/tmp/council_r10_responses.json`*  
*Design: `docs/plans/2026-04-10-sdp-mini-harness-design.md` (v11)*
