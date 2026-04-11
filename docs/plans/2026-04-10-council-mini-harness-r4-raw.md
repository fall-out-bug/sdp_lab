# Council Round 4 — Raw Responses (v4 fix verification)

**Date:** 2026-04-10  
**Spec:** v4 (post Round 3 fixes N1-N7)  
**Quorum status:** 3/6 responded (HARD_ABORT threshold: 4/6)

---

## Architect (codex-rescue)

FIX_N1: CORRECT  
REASON: RunPhase serializes on h.mu, requires state==idle, assigns runID. Defer only resets to idle if state==hStateRunning (not awaiting_human). ApproveGate/Rollback reject unless state==hStateAwaitingHuman.

FIX_N2: INCOMPLETE  
REASON: PendingDecision.RunID is populated, but ApproveGate/Rollback only call ValidateDecision(decisionID) — no explicit RunID==currentRunID check. Stale-run protection depends entirely on decisionID string uniqueness.

FIX_N3: INCOMPLETE  
REASON: Session.MessagesFromTurnRecords() rebuilds from session.turnRecords, but Recover() doesn't show how it repopulates the unexported field. Restart/recovery can silently lose conversation history.

FIX_N4: CORRECT  
REASON: Goroutine does select{case ch <- report; case <-evalCtx.Done()} — exits cleanly on timeout.

FIX_N5: INTRODUCES_NEW_BUG  
REASON: executeCalls launches AfterToolCall goroutines against a local callbackWg (returned from executeCalls), but RunPhase calls EvidenceAccumulator.WaitCallbacks() which waits on accumulator.callbackWg. Two disconnected WaitGroups. Real callback goroutines never increment accumulator.callbackWg → WaitCallbacks() returns immediately → race still exists.

FIX_N6: CORRECT  
FIX_N7: CORRECT

NEW_ISSUE: ClearDecision before transitionTo (non-atomic)
SEVERITY: CRITICAL
LOCATION: Harness.ApproveGate / Harness.Rollback
DESCRIPTION: ApproveGate sets state=idle, calls ClearDecision(), THEN calls transitionTo(). If transitionTo fails, state is idle and pending decision is gone — operator cannot safely retry approval/rollback.
PROPOSAL: Keep state=awaiting_human and keep pending decision until transitionTo durably succeeds. Only then clear decision and flip to idle, ideally in one store transaction.
DOMAIN_VETO: YES

NEW_ISSUE: transitionTo not durable-atomic
SEVERITY: HIGH
LOCATION: Harness.transitionTo
DESCRIPTION: transitionTo updates session.Phase and resets accumulator BEFORE persisting PhaseRecord. If persist fails, in-memory state has moved but durable state has not.
PROPOSAL: PersistPhaseRecord first. Mutate session.Phase and accumulator.Reset() only after successful persist.
DOMAIN_VETO: YES

---

## Critic (google/gemini-3.1-pro-preview) — TRUNCATED

N5: INTRODUCES_NEW_BUG — "The wiring is broken. executeCalls uses a local callbackWg, but RunPhase waits on h.accumulator.WaitCallbacks()" [response truncated by API]

---

## Technician (deepseek/deepseek-v3.2)

FIX_N1: INCOMPLETE  
REASON: defer resets state to idle if state==hStateRunning AND runID matches. After escalation, state is hStateAwaitingHuman — so defer correctly does NOT reset. However spec is hard to reason about with two nested lock/unlock patterns (section 4 has its own defer h.mu.Unlock()). Risk of misread causing future bugs.

FIX_N2: CORRECT  
REASON: PendingDecision persisted. ApproveGate/Rollback validate decisionID and FSM state. Atomic ClearDecision with transition.

FIX_N3: INCOMPLETE  
REASON: TurnRecord.ToolResults appended from ev.ToolResult (string) — ToolResult.Err is not captured via "tool_end" event. Tool failures lost from canonical log.

FIX_N4: CORRECT  
FIX_N5: INCOMPLETE — callbackWg wiring gap (same as architect)  
FIX_N6: CORRECT  
FIX_N7: CORRECT

NEW_ISSUE: FSM defer ambiguity
SEVERITY: HIGH
LOCATION: Harness.RunPhase defer block
DESCRIPTION: Dual-lock pattern (section 4 h.mu.Lock() + defer h.mu.Unlock() + outer FSM reset defer) is hard to reason about. While technically correct, it is a maintenance hazard.
PROPOSAL: Simplify: remove dual-lock pattern. Handle state transitions explicitly at each return point. Or use a single cleanup helper.

NEW_ISSUE: ToolResult Err missing from TurnRecord
SEVERITY: MEDIUM
LOCATION: Harness.RunPhase "tool_end" event processing
DESCRIPTION: Event{Type:"tool_end"} only carries ToolResult string. TurnRecord.ToolResults loses Err field → canonical log incomplete for tool failures.
PROPOSAL: Add ToolErr field to Event struct for "tool_end". Populate TurnRecord accordingly.

NEW_ISSUE: callbackWg wiring gap
SEVERITY: HIGH
LOCATION: Loop.executeCalls ↔ EvidenceAccumulator
DESCRIPTION: executeCalls returns a local *sync.WaitGroup. Run() doesn't expose it to RunPhase. RunPhase calls WaitCallbacks() on accumulator.callbackWg which is never incremented. N5 race still present.
PROPOSAL: Simplest fix — make AfterToolCall synchronous in executeCalls. EvidenceAccumulator operations are fast (mutex+append). Remove callbackWg complexity entirely.

---

## Philosopher, Pragmatist, Engineer — ABSTAIN (empty API responses, same provider issue)
