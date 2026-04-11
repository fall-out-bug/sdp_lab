# LLM Council Report: SDP Mini-Harness Design — Round 8

**Date:** 2026-04-10  
**Round:** 8 (v8 verification)  
**Spec version:** v8 → v9 (post Round 8 fixes applied)  
**Consensus:** NOT_READY → 2 fixes → v9 ready for Round 9  
**Quorum:** 4/6 ✅ (architect + critic + technician + pragmatist)

---

## Quorum

| Role | Model | Status | max_tokens |
|------|-------|--------|-----------|
| Architect | codex-rescue | ✓ Active | — |
| Critic | google/gemini-3.1-pro-preview | ✓ Active (partial) | 1500 |
| Technician | deepseek/deepseek-v3.2 | ✓ Active | 1500 |
| Philosopher | moonshotai/kimi-k2.5 | ✗ ABSTAIN (finish=length) | 3500 (needs 5000+) |
| Pragmatist | minimax/minimax-m2.7 | ✓ Active | 3500 |
| Engineer | xiaomi/mimo-v2-pro | ✗ ABSTAIN (finish=length) | 3000 (needs 4500+) |

*Note: kimi and mimo continue to consume unusually large chain-of-thought budgets. kimi returned empty content even at 3500 tokens.*

---

## Fix Verification (S1, S2)

| Fix | Technician | Pragmatist | Verdict |
|-----|-----------|-----------|---------|
| S1 (Stop + ClearDecision) | CORRECT | CORRECT | ✅ |
| S2 (RestoreHarness ownerToken) | CORRECT | CORRECT | ✅ |

Critic assessed S1 as **INCOMPLETE** — specific concern: `PersistPhaseRecord` error was being ignored, AND the decision was cleared BEFORE the terminal record was persisted (violating durable-first). See U1 below.

---

## New Issues Found (Round 8)

### U1 HIGH — Stop() violates durable-first (Critic)

**Severity:** HIGH  
**Location:** `Harness.Stop()`

The v8 `Stop()` cleared `PendingDecision` **before** calling `PersistPhaseRecord`, and discarded the error from `PersistPhaseRecord`. Two problems:
1. **Wrong ordering:** If `PersistPhaseRecord` fails after decision was cleared, we have no terminal record AND no pending decision — restart sees ambiguous state.
2. **Error swallowed:** Caller can't know the terminal persist failed.

**Fix in v9 (durable-first, mirrors P2 pattern):**
```go
// 1. Persist terminal record FIRST — if fail, return error, nothing changed
if err := h.store.PersistPhaseRecord(h.session.ID, PhaseRecord{...NextPhase: ""...}); err != nil {
    return fmt.Errorf("persist terminal record: %w", err)
}
// 2. Clear decision AFTER durable record exists
if h.state == hStateAwaitingHuman {
    pending, _ := h.store.LoadDecision(h.session.ID)
    if pending != nil {
        h.store.ClearDecision(h.session.ID, pending.DecisionID)
    }
}
h.state = hStateIdle
```
*If ClearDecision fails after PersistPhaseRecord succeeded: terminal record exists; Stop() is idempotent — caller retries, sees NextPhase="" and returns (or skips ClearDecision if decision already cleared).*

---

### U2 MEDIUM — BeforeToolCall not wired into BuildLoopConfig (Technician)

**Severity:** MEDIUM  
**Location:** `PhaseRouter.BuildLoopConfig()` / `Harness.RunPhase()`

`BuildLoopConfig` always produced `cfg.BeforeToolCall = nil` — the field was never populated. `executeCalls` checks `if cfg.BeforeToolCall != nil` before calling it, so A5's fix was dead code — the hook never fired.

**Fix in v9:**
```go
// BuildLoopConfig gains explicit before parameter
func (r *PhaseRouter) BuildLoopConfig(phase Role, acc *EvidenceAccumulator, flag *completionFlag,
    before func(name string, args json.RawMessage) error) (LoopConfig, error) {
    ...
    return LoopConfig{
        ...
        BeforeToolCall: before,   // wired
        AfterToolCall:  acc.OnToolResult,
    }, nil
}

// Harness struct gains field
type Harness struct {
    ...
    beforeToolCall func(name string, args json.RawMessage) error // nil = no-op for MVP
}

// RunPhase passes it
cfg, err := h.router.BuildLoopConfig(phase, h.accumulator, flag, h.beforeToolCall)
```

---

### U3 LOW — Duplicate PendingDecision on double escalation (Technician) — FALSE ALARM

Technician raised concern that if a gate escalates twice, two `PendingDecision` entries could be persisted without checking for an existing one.

**Assessment:** FSM guards prevent double-escalation. `RunPhase` sets `state=hStateRunning` on entry. `handleGateResult` sets `state=hStateAwaitingHuman` on escalation. Any subsequent `RunPhase` call returns early (`state != hStateIdle`). The race doesn't exist. **Not a bug.**

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

**Total: 39 issues found, 39 fixed. Trend: clearly converging (2 per round now, both small).**

---

## Convergence Signals

| Model | Verdict | New issues |
|-------|---------|-----------|
| Pragmatist | READY | NONE |
| Technician | READY | U2 + U3(false alarm) |
| Critic | NOT_READY | U1 (fixed in v9) |
| Architect | NOT_READY → v9 fixes applied | — |

---

## Round 9 Plan

Verify U1-U2 fixes in v9. Attempt 6/6 quorum: kimi at 5000 tokens, mimo at 4500.

*Raw responses: `/tmp/council_r8_responses.json`*  
*Design: `docs/plans/2026-04-10-sdp-mini-harness-design.md` (v9)*
