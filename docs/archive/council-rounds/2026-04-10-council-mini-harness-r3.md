# LLM Council Report: SDP Mini-Harness Design — Round 3

**Date:** 2026-04-10  
**Round:** 3 (v3 fix verification)  
**Spec version:** v3 (post Round 2 fixes)  
**Consensus:** NOT REACHED — HARD_ABORT (quorum failure)  
**Decision Owner:** PENDING (user sign-off required)

---

## ⚠️ HARD_ABORT: Quorum Failure

Per protocol: minimum floor = `ceil(2/3 * 6 configured models) = 4`. Only 3 models responded.

| Model | Role | Status |
|-------|------|--------|
| codex-rescue | Architect | ✓ responded |
| google/gemini-3.1-pro-preview | Critic | ✓ responded (truncated) |
| deepseek/deepseek-v3.2 | Technician | ✓ responded |
| moonshotai/kimi-k2.5 | Philosopher | ✗ ABSTAIN (empty API response) |
| minimax/minimax-m2.7 | Pragmatist | ✗ ABSTAIN (empty API response) |
| xiaomi/mimo-v2-pro | Engineer | ✗ ABSTAIN (empty API response) |

**Root cause:** kimi-k2.5, minimax-m2.7, mimo-v2-pro return `choices: []` from OpenRouter's API consistently. These models appear to be available (request accepted, model IDs resolve) but produce no content. This is an upstream provider issue, not a prompt problem.

**Advisory note:** Architect (codex-rescue) produced 4 domain vetoes on CRITICAL/HIGH issues. Technician confirmed all R2 fixes correct and found 4 new issues. Output is valuable but not council-resolved — presented as advisory findings pending Decision Owner choices.

---

## R2 Fix Verification (Advisory)

| Fix | Architect | Technician | Critic (partial) | Verdict |
|-----|-----------|-----------|-----------------|---------|
| R2-1 (mutex scope) | INTRODUCES_NEW_BUG | CORRECT | CORRECT | ⚡ SPLIT — specific deadlock fixed, but broader concurrency gap identified |
| R2-2 (completion flag) | INCOMPLETE | CORRECT | — | ⚡ SPLIT — closure mechanism correct, but duplicate tool issue |
| R2-3 (timeout pass) | CORRECT | CORRECT | — | ✅ CONFIRMED |
| R2-4 (NextPhase) | INCOMPLETE | CORRECT | — | ⚡ SPLIT — single-edge works, multi-edge phases (RoleReview) incomplete |
| R2-5 (RecoveryNext) | INTRODUCES_NEW_BUG | CORRECT | — | ⚡ SPLIT — validation correct, ApproveGate/Rollback unguarded |

**Architect/Technician split on R2-1, R2-4, R2-5:** Technician assessed the specific bug being fixed as correct. Architect found deeper structural gaps introduced alongside or revealed by the fix. Both are right in scope: fixes resolve the reported bug but open adjacent issues.

---

## New Issues Found (Advisory — Domain Vetoes from Architect)

### CRITICAL — N1: Phase Execution FSM Missing
**Raised by:** Architect (DOMAIN_VETO: YES)

`RunPhase`, `ApproveGate`, and `Rollback` can interleave on the same session. No `idle|running|awaiting_human` state. No `runID`. Concurrent calls can mix or erase evidence mid-run.

**Action required:**
```go
type harnessState int
const (
    hStateIdle           harnessState = iota
    hStateRunning
    hStateAwaitingHuman
)

type Harness struct {
    // ...
    state   harnessState
    runID   uint64        // monotone, increments per RunPhase call
}

// RunPhase: reject if state != hStateIdle
// ApproveGate/Rollback: reject if state != hStateAwaitingHuman
```

---

### CRITICAL — N2: Human Gate Decisions Not Durable
**Raised by:** Architect (DOMAIN_VETO: YES)

On escalation, only `Event{Type:"human_gate"}` is emitted. No `PendingDecision` is persisted. `ApproveGate()`/`Rollback()` operate without proof a decision is pending — any caller can trigger a phase transition at any time.

**Action required:**
```go
type PendingDecision struct {
    DecisionID       string
    RunID            uint64
    Phase            Role
    GateResult       GateResult
    AllowedActions   []string  // "approve" | "rollback" | "stop"
}

// SessionStore.PersistDecision(sessionID, d PendingDecision) error
// ApproveGate(ctx, decisionID string) — validates decisionID matches pending
// Rollback(ctx, decisionID string) — same
```

---

### HIGH — N3: Conversation State Not Canonical Source of Truth
**Raised by:** Architect (DOMAIN_VETO: YES)

`RunPhase` builds `msgs := append(h.session.Messages(), userPrompt)` locally, but never writes the new messages, model outputs, or tool outputs back into `Session` or `SessionStore`. Events are persisted but they're telemetry, not a canonical replay log. Next turn context can diverge from WAL.

**Action required:** Persist a `TurnRecord{UserMsg, AssistantMsgs, ToolResults}` atomically per turn. `Session.Messages()` derives from TurnRecords, not an in-memory buffer.

---

### HIGH — N4: EvaluateCompliance Not Context-Aware → Goroutine Leak
**Raised by:** Architect (DOMAIN_VETO: YES) + Technician

`GateEngine.Evaluate` creates `evalCtx` but calls `harness.EvaluateCompliance(g.contract, snap.toHarness())` without passing the context. On timeout, the compliance goroutine hangs indefinitely.

**Action required:**
```go
go func() {
    select {
    case ch <- harness.EvaluateCompliance(evalCtx, g.contract, snap.toHarness()):
    case <-evalCtx.Done():
        // context cancelled — goroutine exits
    }
}()
```
`harness.EvaluateCompliance` must accept and respect `ctx context.Context`.

---

### HIGH — N5: AfterToolCall Signature Mismatch + Race with Snapshot
**Raised by:** Architect (DOMAIN_VETO: YES) + Technician

Two separate issues:
1. `LoopConfig.AfterToolCall` is `func(name, result string) error` but `EvidenceAccumulator.OnToolResult` is `func(toolName, result string, err error)`. Signature mismatch means tool errors cannot reach the accumulator.
2. `AfterToolCall` callbacks from parallel goroutines may still be running when `RunPhase` calls `acc.Snapshot()` after events drain.

**Action required:**
```go
// Unified hook type
type AfterToolCallFn func(result ToolResult) error

// In executeCalls, track callbacks with WaitGroup
// RunPhase calls toolWg.Wait() before acc.Snapshot()
```

---

### MEDIUM — N6: completion_signal Duplicate in ToolRegistry
**Raised by:** Technician + Architect (INCOMPLETE on R2-2)

`PhaseConfig.Tools` lists `completion_signal`, AND `BuildLoopConfig` appends it dynamically. If both paths execute, the tool appears twice — undefined behavior for LLM tool selection.

**Action required:** Remove `completion_signal` from all `PhaseConfig.Tools` allowlists. `BuildLoopConfig` is the only place that adds it. Document this as implicit/special tool.

---

### MEDIUM — N7: completionFlag Summary Not Validated
**Raised by:** Technician

If `json.Unmarshal` fails in `completion_signal.Execute`, `flag.summary` stays empty. Harness reads `flag.signaled` but doesn't validate `flag.summary`. This is a logging gap, not a correctness issue (summary is not used in gate logic).

**Action required:** After reading `signaled`, validate `summary` non-empty; log warning if empty. Don't block gate on missing summary.

---

## Decision Owner Actions Required

### Immediate (before proceeding to implementation)

- [ ] **N1 Accept:** Add `harnessState` FSM to Harness — reject concurrent calls based on state
- [ ] **N2 Accept:** Persist `PendingDecision`; bind `ApproveGate`/`Rollback` to `decisionID`
- [ ] **N3 Accept:** Persist canonical `TurnRecord`; derive `Session.Messages()` from records
- [ ] **N4 Accept:** Make `EvaluateCompliance` context-aware; fix goroutine leak
- [ ] **N5 Accept:** Unify `AfterToolCall` signature; add `WaitGroup` sync before `Snapshot`
- [ ] **N6 Accept:** Remove `completion_signal` from `PhaseConfig.Tools` allowlists
- [ ] **N7 Defer:** Summary validation is a log hygiene issue — acceptable for MVP

### Quorum Resolution Required

- [ ] **Choose one:**
  - A) **Override quorum** — proceed with advisory findings; accept architect's 4 domain vetoes as blocking; fix N1-N5 before implementation
  - B) **Replace unavailable models** — find working API IDs for philosopher/pragmatist/engineer roles and run Round 4
  - C) **Halt council** — accept current findings, proceed to implementation

**Recommendation:** Option A. The architect's domain vetoes are structural (FSM, durable decisions, canonical state, context leak, signature mismatch). These are correctness issues that should be fixed regardless of whether philosopher/pragmatist/engineer weigh in.

---

## Round Convergence

| Round | Spec | Resolved | New | Active Models |
|-------|------|----------|-----|---------------|
| 0 | v1 | 0 | 7 issues | 6/6 |
| 1 | v1 | 7 | 3 | 5/6 |
| 2 | v2 | 5 (fixes) | 5 new | ~3/6 |
| 3 | v3 | 0 (HARD_ABORT) | 7 advisory | 3/6 |

---

*Raw responses: docs/plans/2026-04-10-council-mini-harness-r3-raw.md*  
*Design: docs/plans/2026-04-10-sdp-mini-harness-design.md (v3)*
