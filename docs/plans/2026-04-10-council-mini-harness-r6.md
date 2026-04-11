# LLM Council Report: SDP Mini-Harness Design — Round 6

**Date:** 2026-04-10  
**Round:** 6 (first full quorum — fresh blind review of v6)  
**Spec version:** v6 → v7 (post Round 6 fixes applied)  
**Consensus:** NOT_READY → fixes applied → v7 ready for Round 7 verification  
**Quorum:** 4/6 ✅ (first quorum achieved — 3/6 in all prior rounds)

---

## ⚡ First Quorum Achieved

Rounds 1–5 ran with a persistent HARD_ABORT (3/6 active models). Round 6 added moonshotai/kimi-k2.5 as the 4th active model, crossing the ≥4 minimum threshold.

| Model | Round 6 Status | Root cause of prior abstention |
|-------|---------------|-------------------------------|
| kimi-k2.5 | ✓ Active | max_tokens=1200+ sufficient (chain-of-thought ~500 tok) |
| minimax-m2.7 | ✗ ABSTAIN | max_tokens=1800 still insufficient — needs ~3000+ |
| mimo-v2-pro | ✗ ABSTAIN | max_tokens=1500 still insufficient — needs ~2500+ |

---

## Issues Found (Round 6)

| ID | Role | Severity | Domain Veto | Issue | Fixed in v7 |
|----|------|---------|-------------|-------|-------------|
| A1 | Architect | CRITICAL | YES | `LoadDecision()` absent from `SessionStore` → can't restore `awaiting_human` after restart | ✅ |
| A2 | Architect | CRITICAL | YES | `ownerToken` not validated in `RunPhase`/`ApproveGate`/`Rollback` → any caller can mutate | ✅ |
| A3 | Architect | HIGH | YES | `Harness.Stop()` absent despite `SurfaceEvent{stop}` in Surface spec | ✅ |
| A4 | Architect | HIGH | YES | `AfterToolCall` error silently dropped in `executeCalls` → silent evidence loss | ✅ |
| A5 | Technician | MEDIUM | NO | `BeforeToolCall` hook defined in `LoopConfig` but never called in `executeCalls` | ✅ |
| A6 | Philosopher | MEDIUM | NO | `EvidenceAccumulator.Reset()` called in `transitionTo` but never defined in spec | ✅ |
| D1 | Architect | MEDIUM | NO | `SessionStore.Persist()` usage undocumented — ambiguity about Session.Phase authority | ✅ |
| T1 | Technician | LOW | NO | `ToolResult` missing `Arguments` field — evidence accumulator loses call context | ✅ |

**Total: 8 issues (4 domain vetoes). All fixed in v7.**

---

## Fix Details

### A1 — LoadDecision + RestoreHarness
`SessionStore` gained `LoadDecision(sessionID string) (*PendingDecision, error)`.  
`RestoreHarness(sessionID, store, router, gate)` factory added:
```go
func RestoreHarness(sessionID string, store SessionStore, router *PhaseRouter, gate *GateEngine) (*Harness, error) {
    session, err := RecoverSession(sessionID, store)
    if err != nil { return nil, err }
    h := &Harness{session: session, store: store, router: router, gate: gate,
                  accumulator: NewEvidenceAccumulator(), state: hStateIdle}
    pending, err := store.LoadDecision(sessionID)
    if err != nil { return nil, fmt.Errorf("load decision: %w", err) }
    if pending != nil { h.state = hStateAwaitingHuman; h.runID = pending.RunID }
    return h, nil
}
```

### A2 — validateToken
```go
func (h *Harness) validateToken(token string) error {
    if h.ownerToken == "" { return nil }
    if token != h.ownerToken { return fmt.Errorf("unauthorized: invalid owner token") }
    return nil
}
// All mutating methods: RunPhase(ctx, prompt, token), ApproveGate(ctx, decisionID, token),
//                       Rollback(ctx, decisionID, token), Stop(ctx, token)
```

### A3 — Harness.Stop
```go
func (h *Harness) Stop(ctx context.Context, token string) error {
    if err := h.validateToken(token); err != nil { return err }
    h.mu.Lock(); defer h.mu.Unlock()
    if h.state == hStateRunning { return fmt.Errorf("phase in progress; cancel ctx first") }
    rec := PhaseRecord{SessionID: h.session.ID, Phase: h.session.Phase,
                       NextPhase: "", CompletedAt: time.Now()}
    if err := h.store.PersistPhaseRecord(rec); err != nil { return err }
    h.state = hStateIdle
    return nil
}
```

### A4 + A5 — executeCalls with BeforeToolCall + AfterToolCall error capture
```go
go func(i int, call ToolCall) {
    defer wg.Done()
    if cfg.BeforeToolCall != nil {
        if err := cfg.BeforeToolCall(call.Name, call.Arguments); err != nil {
            results[i] = ToolResult{ID: call.ID, Name: call.Name,
                                    Arguments: call.Arguments,
                                    Err: fmt.Errorf("before hook rejected: %w", err)}
            // AfterToolCall still fires on rejection
            if cfg.AfterToolCall != nil {
                if cbErr := cfg.AfterToolCall(results[i]); cbErr != nil {
                    results[i].Err = fmt.Errorf("%w; callback: %v", results[i].Err, cbErr)
                }
            }
            return
        }
    }
    // ... execute tool ...
    if cfg.AfterToolCall != nil {
        if cbErr := cfg.AfterToolCall(results[i]); cbErr != nil {
            results[i].Err = fmt.Errorf("callback: %w", cbErr)
        }
    }
}(i, call)
```

### A6 — EvidenceAccumulator.Reset
```go
func (ea *EvidenceAccumulator) Reset() {
    ea.mu.Lock(); defer ea.mu.Unlock()
    ea.evidence = ea.evidence[:0]
    ea.claims = ea.claims[:0]
    for k := range ea.quality { delete(ea.quality, k) }
}
```

### D1 — Session.Phase derivation documented
`Session.Phase` is NOT independently persisted per transition. On recovery, `RecoverSession` derives current phase from the latest `PhaseRecord.NextPhase` in the store. `SessionStore.Persist` is used only for initial session creation. `RestoreHarness` documents this explicitly.

### T1 — ToolResult.Arguments
```go
type ToolResult struct {
    ID        string
    Name      string
    Arguments json.RawMessage  // added — from ToolCall.Arguments
    Output    string
    Err       error
}
```

---

## Convergence Analysis

| Round | Spec | Issues In | Issues Fixed | New CRITICAL/HIGH |
|-------|------|-----------|-------------|-------------------|
| 0–1 | v1→v2 | 7 (I1-I7) | 7 | 5 |
| 2 | v2→v3 | 5 (R2-1..5) | 5 | 2 |
| 3 | v3→v4 | 7 (N1-N7) | 7 | 5 |
| 4 | v4→v5 | 5 (P1-P5) | 5 | 3 |
| 5 | v5→v6 | 3 (Q1-Q3) | 3 | 0 |
| 6 | v6→v7 | 8 (A1-A6, D1, T1) | 8 | 4 (A1-A2 CRITICAL, A3-A4 HIGH) |

**Total issues across all rounds: 35 (I1-I7 + R2-1..5 + N1-N7 + P1-P5 + Q1-Q3 + A1-A6 + D1 + T1). All 35 fixed.**

Round 6 found critical auth and recovery gaps (A1 CRITICAL: LoadDecision, A2 CRITICAL: token auth) that were genuinely missing from prior reviews. First quorum (4/6) was required to surface them — confirms value of full quorum.

---

## Quorum Note

Pragmatist (minimax-m2.7) and Engineer (mimo-v2-pro) still abstain due to insufficient `max_tokens`. These models consume ~2000–3000 tokens on internal chain-of-thought before generating content. Round 7 will use minimax=3500, mimo=2500.

---

## Next Step: Round 7

**Goal:** Verify A1-T1 fixes in v7. With minimax and mimo at correct token limits, attempt 6/6 full quorum.

**Round 7 script:** `/tmp/council_r7.py` — verify fixes + convergence check.

---

*Raw responses: docs/plans/2026-04-10-council-mini-harness-r6-raw.md*  
*Design: docs/plans/2026-04-10-sdp-mini-harness-design.md (v7)*
