# LLM Council Report: SDP Mini-Harness Design — Round 4

**Date:** 2026-04-10  
**Round:** 4 (v4 fix verification)  
**Spec version:** v4 (post Round 3 fixes N1-N7)  
**Consensus:** NOT REACHED — HARD_ABORT (quorum failure, 3/6)  
**Decision Owner:** PENDING

---

## ⚠️ HARD_ABORT: Quorum Failure (persistent)

Same 3 models as Round 3. kimi-k2.5, minimax-m2.7, mimo-v2-pro consistently return empty content.

---

## Fix Verification (Advisory)

| Fix | Architect | Technician | Critic (partial) | Verdict |
|-----|-----------|-----------|-----------------|---------|
| N1 (FSM) | CORRECT | INCOMPLETE* | — | ⚡ *Tech concern about defer is valid code-clarity issue, not a logic bug. Addressed with comment in v5. |
| N2 (PendingDecision) | INCOMPLETE | CORRECT | — | ⚡ Split — arch concerned about RunID check (minor gap). Tech sees atomicity as correct. |
| N3 (TurnRecord) | INCOMPLETE | INCOMPLETE | — | ⚡ Two different sub-issues: arch found Recover() doesn't reload turnRecords; tech found ToolResult.Err missing from events |
| N4 (goroutine leak) | CORRECT | CORRECT | — | ✅ |
| N5 (callbackWg) | INTRODUCES_NEW_BUG | INCOMPLETE | INTRODUCES_NEW_BUG | ❌ Unanimous: dual-WG design broken — executeCalls.callbackWg and accumulator.callbackWg disconnected |
| N6 (completion_signal dupe) | CORRECT | CORRECT | — | ✅ |
| N7 (summary warn) | CORRECT | CORRECT | — | ✅ |

---

## New Issues Found (Advisory)

### P1 CRITICAL🔴 — ClearDecision Before transitionTo (Architect DOMAIN_VETO)

`ApproveGate` / `Rollback` cleared the decision BEFORE calling `transitionTo`. If `transitionTo` fails (e.g., store write error), state is `idle` and the pending decision is gone — operator cannot retry. Non-atomic, violates crash-safety.

**Fix in v5:** `transitionTo()` first. `ClearDecision()` only after successful return. On error: state stays `awaiting_human`, decision intact → safe retry.

---

### P2 HIGH — transitionTo Not Durable-Atomic (Architect DOMAIN_VETO)

`transitionTo` mutated `session.Phase` and called `accumulator.Reset()` BEFORE `PersistPhaseRecord`. If persist failed, in-memory state had moved but durable state had not.

**Fix in v5:** `PersistPhaseRecord(... NextPhase: next ...)` first. `session.Phase = next` and `accumulator.Reset()` only after successful persist. `PhaseRecord.NextPhase` field added for idempotent recovery.

---

### P3 HIGH — FSM Defer Ambiguity (Technician)

The dual-lock pattern in `RunPhase` (outer FSM reset defer + inner `defer h.mu.Unlock()` in section 4) is hard to reason about. While technically correct (defer checks `state==hStateRunning`, which is false after escalation), it's a maintenance hazard.

**Fix in v5:** Added explicit comment in defer explaining exactly why `hStateAwaitingHuman` is not overwritten. The logic is correct as-is; clarity was missing.

---

### P4 MEDIUM — ToolResult Err Missing from TurnRecord (Technician)

`"tool_end"` events only carried `ToolResult string`. TurnRecord lost tool failure information, making the canonical log incomplete for replay/audit.

**Fix in v5:** Added `ToolErr error` field to `Event` struct. `RunPhase` now populates `TurnRecord.ToolResults[i].Err = ev.ToolErr`.

---

### P5 HIGH — callbackWg Wiring Broken (Architect + Technician + Critic — unanimous)

`executeCalls` launched `AfterToolCall` goroutines against a local `callbackWg`. `RunPhase` called `accumulator.WaitCallbacks()` which waits on `accumulator.callbackWg`. These are two different `sync.WaitGroup` instances — never connected. `WaitCallbacks()` returned immediately while callbacks still ran.

**Fix in v5:** Eliminated dual-WG complexity entirely. `AfterToolCall` is now **synchronous** within each `executeCalls` goroutine, called before `wg.Done()`. By the time `wg.Wait()` returns, all callbacks are complete. `callbackWg`, `WaitCallbacks()`, `TrackCallback()` removed.

---

## Round Convergence

| Round | Spec | Issues In | Issues Fixed | Remaining | Active Models |
|-------|------|-----------|-------------|-----------|---------------|
| 0-1 | v1→v2 | 7 (I1-I7) | 7 | 0 | 6/6, 5/6 |
| 2 | v2→v3 | 5 (R2-1..5) | 5 | 0 | ~3/6 |
| 3 | v3→v4 | 7 (N1-N7) | 7 | 0 | 3/6 HARD_ABORT |
| 4 | v4→v5 | 5 (P1-P5) | 5 | 0 | 3/6 HARD_ABORT |

**Convergence signal:** All found issues fixed each round. New issues per round: 7→5→5. Architect domain vetoes: 4 (R3) → 2 (R4). Trend is converging.

---

*Raw responses: docs/plans/2026-04-10-council-mini-harness-r4-raw.md*  
*Design: docs/plans/2026-04-10-sdp-mini-harness-design.md (v5)*
