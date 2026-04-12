# LLM Council Report: SDP Mini-Harness Design

**Date:** 2026-04-10  
**Rounds:** 1 of 5 (early termination — convergence ≥80% on all issues)  
**Consensus:** REACHED  
**Convergence:** 7/7 resolved, 3 new issues (2 accepted, 1 deferred)  
**Decision Owner:** PENDING (user sign-off required)

## Council Members

| Role | Model | Status |
|------|-------|--------|
| Architect | anthropic/claude-opus-4-5 | ✓ responded |
| Critic | openai/gpt-4.1 | ✓ responded |
| Technician | deepseek/deepseek-r1 | ✓ responded |
| Philosopher | meta-llama/llama-4-maverick | ✓ responded |
| Pragmatist | qwen/qwen3-235b-a22b | ✓ responded |
| Engineer | qwen/qwen-2.5-coder-32b | ✗ ABSTAIN (model error) |

---

## Vote Tally

| ID | Severity | SUPPORT | OPPOSE | ABSTAIN | Domain Veto | Result |
|----|----------|---------|--------|---------|-------------|--------|
| I1 | CRITICAL | 5/5 | 0 | 0 | — | RESOLVED |
| I2 | HIGH | 5/5 | 0 | 0 | Architect ✓, Technician ✓ | RESOLVED (mandatory action) |
| I3 | HIGH | 5/5 | 0 | 0 | — | RESOLVED |
| I4 | MEDIUM | 5/5 | 0 | 0 | — | RESOLVED |
| I5 | MEDIUM | 5/5 | 0 | 0 | — | RESOLVED |
| I6 | MEDIUM | 5/5 | 0 | 0 | — | RESOLVED |
| I7 | LOW | 4/5 | 0 | 1 | — | RESOLVED |

> Note: Philosopher's I2 verdict was OPPOSE with evidence/proposal consistent with SUPPORT
> (semantics confusion — treated as SUPPORT by orchestrator).
> Pragmatist domain vetoes on all issues are invalid per protocol (role has no domain veto authority).

---

## Recommendations (for Decision Owner)

### RESOLVED — action required before implementation

---

**[I1: Evidence accumulation gap]** — CRITICAL — 5/5 support

The design shows `GateCheck(PhaseSnapshot)` but no code path from agent output to
`PhaseSnapshot.Evidence[]` or `Claims[]`. The loop emits events but nothing consumes
them into structured evidence. Gate cannot pass without evidence → deadlock.

**Action required:**
Add `EvidenceAccumulator` interface called in `afterToolCall` hook and at each
`turn_end`. Tool results feed the accumulator; accumulator builds `PhaseSnapshot`.
Make explicit in loop.go — not an implementation detail.

```go
type EvidenceAccumulator interface {
    OnToolResult(toolName string, result string) error
    OnTurnEnd(message Message) error
    Snapshot() PhaseSnapshot
}
```

---

**[I2: TransitionTo caller]** — HIGH — 5/5 support — 2 domain vetoes (Architect + Technician)

If the agent LLM calls `TransitionTo` as a tool, structural discipline collapses.
This is the highest-risk design ambiguity. Domain vetoes from both system design
and feasibility domains mean this CANNOT proceed without explicit resolution.

**Action required (mandatory per domain veto):**
`TransitionTo` is harness-internal ONLY. Never expose as agent tool. Add comment
in code: `// NOT A TOOL — infrastructure only`. Criteria for harness to call
transition: (a) GateCheck passes, (b) harness detects task completion signal
(defined completion heuristic or explicit `done` tool that signals intent but
does not transition directly).

```go
// completionTool signals intent, does NOT transition
var doneTool = Tool{Name: "signal_done", Execute: func(...) { 
    l.completionSignaled = true; return "noted", nil
}}
// harness checks completionSignaled + gate pass → calls TransitionTo
```

---

**[I3: Session persistence]** — HIGH — 5/5 support

`Session` struct has no `Save()`/`Load()`, no storage backend reference.
Process restart = lost session = broken SDLC continuity.

**Action required:**
Add `SessionStore` interface. Inject at construction. Serialize after each turn
(WAL pattern for crash safety).

```go
type SessionStore interface {
    Persist(s Session) error
    Recover(id string) (Session, error)
}
// Implementations: BoltDB (embedded, zero-config), SQLite, or file-based JSON
```

Recommended for MVP: BoltDB (single file, no server, ACID).

---

**[I4: Model unavailability]** — MEDIUM — 5/5 support

`PhaseConfig.Model string` is a single point of failure. No retry, no fallback.

**Action required:**
Change to `Models []string` ordered by preference. Loop tries first available
via modelgateway health check. Add circuit breaker in modelgateway.

```go
type PhaseConfig struct {
    Models      []string  // ["anthropic/claude-opus-4-5", "openai/gpt-4.1"]
    // ...
}
```

---

**[I5: LLM Council placement]** — MEDIUM — 5/5 support

"Model council / совет моделей" is a stated core requirement but absent from the
architecture diagram. It needs an explicit place in the system.

**Action required:**
Add `CouncilGate` component. When a phase transition requires council deliberation
(configured per `PhaseConfig.CouncilRequired bool`), `CouncilGate` runs the
llm-council protocol before `TransitionTo` is called.

```
PhaseRouter.TransitionTo → CouncilGate → EvaluateCompliance → GateCheck → transition
```

Council is governance above the harness, not a separate phase.

---

**[I6: Context window management]** — MEDIUM — 5/5 support

`transformContext` is mentioned as "preserved from pi-mono" but unspecified.
Multi-phase sessions will overflow context windows without strategy.

**Action required:**
Define `ContextManager` with sliding window strategy:
- Always pin: SystemPrompt + active PhaseRecord
- Recent N messages (configurable per model's context size)
- Summarized history for older messages (one-shot summary, not re-summarized)
- Total budget: 80% of model context window

```go
type ContextManager interface {
    Transform(messages []Message, model string) ([]Message, error)
}
```

---

**[I7: Surface failure isolation]** — LOW — 4/5 support (philosopher abstained)

Session durability under surface crash depends on I3 (persistence). With WAL
pattern from I3, sessions survive crashes. Surfaces reconnect via Session ID.

**Action required:** Resolved as dependency on I3 implementation. Add Session ID
to Surface.Connect(sessionID string) signature for reconnection flow.

---

### New Issues Accepted

**[I8: Tool registration lifecycle]** (raised by Architect) — MEDIUM

Tools per phase need availability validation before phase entry. No mechanism
to detect missing tool or handle gracefully.

Decision needed: static registry (validate at startup) vs dynamic (validate at phase entry).

---

**[N1: Tool failure cascade]** (raised by Critic) — MEDIUM

Tool execution errors can cascade to session failure. No explicit error boundary
between tool failure and loop state.

Decision needed: tool error → emit event → agent retries, or → block phase transition.

---

### New Issues Deferred to Decision Owner

**[N4: Audit log for compliance]** (raised by Critic)  
No event logging/auditing for SDLC compliance or debugging. Deferred: implement
after MVP loop works, using existing `bd remember` + structured log.

**[N5: Security boundaries]** (raised by Critic)  
Agent/tool/router privilege boundaries undefined. Deferred: implementation phase.

---

## Minority Reports

None. Council converged on all issues unanimously or near-unanimously.

---

## Round Convergence

| Round | Resolved | New | Confidence Avg | Budget |
|-------|----------|-----|----------------|--------|
| 0 | 0 | 7 | — | — |
| 1 | 7 | 3 | HIGH | ~0.1% |

**Convergence score:** 0.95 (verdict_agreement 1.0 × 0.7 + confidence 0.85 × 0.3)
Early termination triggered: convergence ≥ 0.80, no blocking domain vetoes.

---

## Decision Owner Action Required

Per council output, the following require your sign-off:

- [ ] **I1** Accept: add `EvidenceAccumulator` to design
- [ ] **I2** Accept (domain veto): `TransitionTo` harness-internal + `signal_done` tool pattern
- [ ] **I3** Accept: add `SessionStore` with BoltDB for MVP
- [ ] **I4** Accept: `Models []string` fallback chain
- [ ] **I5** Accept: `CouncilGate` as governance layer before phase transitions
- [ ] **I6** Accept: `ContextManager` with sliding window
- [ ] **I7** Accept: resolved via I3 + Surface.Connect(sessionID)
- [ ] **I8** Decide: static vs dynamic tool registry
- [ ] **N1** Decide: tool error handling strategy
- [ ] **N4** Defer: audit log post-MVP
- [ ] **N5** Defer: security boundaries to implementation

---

*Audit: docs/plans/2026-04-10-council-mini-harness-raw.md*  
*Design: docs/plans/2026-04-10-sdp-mini-harness-design.md*
