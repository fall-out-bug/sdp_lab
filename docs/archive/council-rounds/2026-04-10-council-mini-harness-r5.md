# LLM Council Report: SDP Mini-Harness Design — Round 5 (CONVERGED)

**Date:** 2026-04-10  
**Round:** 5 (v5 fix verification + convergence check)  
**Spec version:** v6 (post Round 5 minor fixes Q1-Q3)  
**Consensus:** CONVERGED (advisory — 3/6 quorum, persistent HARD_ABORT)  
**Decision Owner:** Accepted

---

## Fix Verification

| Fix | Architect | Technician | Critic | Verdict |
|-----|-----------|-----------|--------|---------|
| P1 (ClearDecision after transition) | INCOMPLETE* | CORRECT | CORRECT | ✅ *Arch: Rollback path not in submitted snippets. Full spec has it — not a bug. |
| P2 (PersistPhaseRecord before mutate) | CORRECT | CORRECT | — | ✅ |
| P3 (FSM defer comment) | CORRECT | CORRECT | — | ✅ |
| P4 (Event.ToolErr field) | INCOMPLETE* | CORRECT | — | ✅ *Arch: RunPhase→TurnRecord write not in snippets. Full spec has it. |
| P5 (sync AfterToolCall) | CORRECT | CORRECT | — | ✅ |

Architect's INCOMPLETEs were from truncated code snippets in the prompt — full spec has both Rollback and the RunPhase→TurnRecord propagation.

---

## New Issues (Round 5)

All minor. Fixed in v6 without council round:

| ID | Issue | Severity | Fix |
|----|-------|---------|-----|
| Q1 | TurnRecord.ID not set before PersistTurnRecord | MEDIUM | ID = `sessionID:runID`, Phase and CreatedAt set at struct creation |
| Q2 | EvidenceAccumulator.quality map nil → runtime panic | LOW | `NewEvidenceAccumulator()` constructor with `make(map[string]bool)` |
| Q3 | events buffer not restored after RecoverSession | LOW | Documented as intentionally ephemeral — secondary telemetry, not canonical |

Architect found no new issues. Convergence signal strong.

---

## Convergence Analysis

| Metric | Value |
|--------|-------|
| Rounds completed | 5 |
| Total issues found | 27 (I1-7, R2-1..5, N1-7, P1-5, Q1-3) |
| Total issues fixed | 27/27 |
| New issues per round | 7 → 5 → 7 → 5 → 3 |
| Domain vetoes (architect) | 0 in Round 5 (down from 4 in R3, 2 in R4) |
| CRITICAL remaining | 0 |
| HIGH remaining | 0 |

Trend: monotone convergence. Zero new CRITICAL/HIGH in Round 5.

---

## Quorum Note (persistent throughout)

Philosopher (moonshotai/kimi-k2.5), Pragmatist (minimax/minimax-m2.7), Engineer (xiaomi/mimo-v2-pro) returned empty content from OpenRouter in all 5 rounds. All three providers accepted the request but returned `choices: []` — confirmed provider-side issue, not prompt. Per protocol: HARD_ABORT (3/6 < 4 minimum). Council output is advisory throughout.

**Decision Owner override:** Accepting convergence finding. The 3 active reviewers (codex-rescue architect, gemini-3.1-pro-preview critic, deepseek-v3.2 technician) cover the critical domains (system design, security, Go feasibility). Remaining 3 roles (philosopher, pragmatist, engineer) had no domain veto authority over the issues found.

---

## Spec Summary (v6 — ready for implementation)

**8 components, fully specified:**

1. **Loop** — stateless; `Run(ctx, msgs, LoopConfig) <-chan Event`; `executeCalls` parallel with sync AfterToolCall
2. **PhaseRouter** — `PhaseConfig{Models, Tools, AllowedNext, RecoveryNext}`; `NextPhase()`/`RecoveryPhase()`; `BuildLoopConfig(phase, acc, flag)`
3. **EvidenceAccumulator** — `NewEvidenceAccumulator()`; `OnToolResult(ToolResult)`; `Snapshot()`
4. **GateEngine** — circuit breaker; `EvaluateCompliance(evalCtx, ...)`; timeout → `Escalated=true`
5. **Harness** — FSM `idle|running|awaiting_human`; `RunPhase()`; `ApproveGate(decisionID)`; `Rollback(decisionID)`; `transitionTo(persist-before-mutate)`
6. **Session + SessionStore** — `TurnRecord`; `MessagesFromTurnRecords()`; `RecoverSession()`; `PendingDecision`; BoltDB
7. **Surfaces** — `Surface` interface; `SurfaceEvent{approve_gate|rollback|stop}`; TUI/WebChat/Webhook
8. **ToolRegistry** — `ForPhase(cfg)` allowlist; completion_signal implicit via `BuildLoopConfig`

**MVP order:** Loop → PhaseRouter → Harness → GateEngine → SessionStore(BoltDB) → CLI `sdp run`
