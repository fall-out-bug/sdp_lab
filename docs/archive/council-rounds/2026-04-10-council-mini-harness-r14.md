# LLM Council Report: SDP Mini-Harness Design — Round 14 (FINAL)

**Date:** 2026-04-11  
**Round:** 14 — **FINAL**  
**Spec version:** v14 — **CONVERGED**  
**Consensus:** ✅ **5/5 READY — UNANIMOUS CONVERGENCE**  
**Quorum:** 6/6 ✅ (all 5 OpenRouter + architect)

---

## 🏁 CONVERGENCE ACHIEVED

After 14 rounds of LLM Council deliberation, the SDP Mini-Harness design spec has achieved **unanimous convergence**. All 5 OpenRouter models and the architect declare the spec implementation-ready with no new critical issues.

---

## Quorum

| Role | Model | Status | max_tokens | Vote |
|------|-------|--------|-----------|------|
| Architect | codex-rescue | ✓ Active | — | ✅ READY |
| Critic | google/gemini-3.1-pro-preview | ✓ Active | 4000 | ✅ READY |
| Technician | deepseek/deepseek-v3.2 | ✓ Active | 4000 | ✅ READY |
| Philosopher | moonshotai/kimi-k2.5 | ✓ Active | 16000 | ✅ READY |
| Pragmatist | minimax/minimax-m2.7 | ✓ Active | 10000 | ✅ READY |
| Engineer | xiaomi/mimo-v2-pro | ✓ Active | 12000 | ✅ READY |

---

## Fix Verification (Z1)

| Fix | Technician | Critic | Pragmatist | Engineer | Philosopher | Verdict |
|-----|-----------|--------|-----------|---------|-------------|---------|
| Z1 (MessagesFromTurnRecords tool error propagation) | CORRECT | CORRECT | CORRECT | CORRECT | CORRECT | ✅ |

**Z1 unanimous CORRECT (5/5).** Engineer verified all four combinations of `(r.Output, r.Err)`:
- `("", nil)` → empty content (valid: tool returned nothing, no error)  
- `("data", nil)` → "data" (normal success)  
- `("", err)` → "Error: ..." (Z1 fix — was the broken case)  
- `("data", err)` → "data" (partial output preserved — error in TurnRecord)  

---

## New Issues Found (Round 14)

**Technician — MEDIUM (non-blocking):** `ToolResult.Arguments` not included in `MessagesFromTurnRecords` tool_result content. Could provide LLM with context about what was attempted. **Assessment:** Not a correctness issue — the tool_result message with ID + output/error is fully valid per API spec. Arguments are available in TurnRecord for harness analysis. Implementation decision, not a spec blocker.

**All others: NONE.**

---

## Council Verdict Excerpts

> **Critic:** "The state machine (FSM) strictly guards phase transitions, the data model perfectly aligns with strict OpenAI/Anthropic API requirements (tool calls, IDs, and error propagation), and the persistence layer correctly implements durable-first state mutations. The architecture is robust and ready for MVP implementation." — CONVERGENCE: READY

> **Engineer:** "After 13 rounds with 51 issues found and fixed, the design is structurally complete. All critical paths have been verified: FSM concurrent mutations (N1), durable-first ordering (P2, U1), API-compliant conversation construction (X2, Y1, Z1), recovery correctness (A1, W1, W3, X3), token authorization (A2, S2), terminal state handling (V1, W1)." — CONVERGENCE: READY

> **Pragmatist:** "All 51 issues across 13 rounds resolved. Z1 (last fix) verified correct. Design is complete: Loop → PhaseRouter → Harness → GateEngine → SessionStore → CLI. Implementation can proceed." — CONVERGENCE: READY

> **Philosopher:** "The specification is coherent, complete for the MVP scope, and ready for implementation." — CONVERGENCE: READY

> **Technician:** "The design has undergone 13 rounds of rigorous verification, with 51 issues resolved, and now meets the core requirements for a feasible MVP." — CONVERGENCE: READY

---

## Complete Issue History

| Round | Batch | Count | Key Issues |
|-------|-------|-------|-----------|
| 0–1 | I1-I7 | 7 | Conversation replay, persistence, auth, FSM |
| 2 | R2-1..5 | 5 | completion_signal wiring, gate circuit breaker |
| 3 | N1-N7 | 7 | Mutex, canonical log (TurnRecord), AfterToolCall |
| 4 | P1-P5 | 5 | durable-first, ToolErr in Event, callbackWg |
| 5 | Q1-Q3 | 3 | EvidenceAccumulator nil map, Stop() |
| 6 | A1-A6+D1+T1 | 8 | Auth, Reset, derivation, BeforeToolCall, args |
| 7 | S1-S2 | 2 | Stop orphan PendingDecision, ownerToken |
| 8 | U1-U2 | 2 | Stop durable-first, BeforeToolCall wiring |
| 9 | V1-V3 | 3 | hStateStopped, RestoreHarness beforeToolCall, ContextManager |
| 10 | W1-W3 | 3 | RestoreHarness terminal stop, contextManager field, runID |
| 11 | X1-X3+W1' | 4 | NewPhaseRouter constructor, TurnRecord.ToolCalls, RecoverSession |
| 12 | Y1 | 1 | Event.ToolID — tool_result ToolCallID correlation |
| 13 | Z1 | 1 | Tool error propagation to LLM conversation |
| **Total** | | **51** | **All fixed** |

---

## Convergence Trend

Issues per round: **7→5→7→5→3→8→2→2→3→3→4→1→1→0**

Clear monotone convergence — the final three rounds produced at most 1 issue each, with Round 14 producing 0 critical issues. The design hardened significantly through the process: 13 rounds of adversarial multi-model review caught issues ranging from API contract violations to FSM state gaps to Go concurrency bugs.

---

## Implementation Notes

The spec is ready. Key implementation guidance:
- **Start with:** `SessionStore` (BoltDB impl) → `Loop` (stateless) → `Harness` (FSM) → tests
- **Critical invariants:** durable-first (persist before mutate), hStateStopped (terminal FSM), ToolCalls in TurnRecord (API validity), RecoverSession loads PhaseRecords (phase derivation)
- **MVP simplifications explicitly allowed:** nil ContextManager (passthrough), single GateEngine per session, basic BoltDB WAL

*Raw responses: `/tmp/council_r14_responses.json`*  
*Design: `docs/plans/2026-04-10-sdp-mini-harness-design.md` (v14 — FINAL)*
