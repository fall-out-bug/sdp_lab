# LLM Council Report: SDP Mini-Harness Design — Round 11

**Date:** 2026-04-11  
**Round:** 11  
**Spec version:** v11 → v12 (post Round 11 fixes applied)  
**Consensus:** NOT_READY → 4 fixes → v12 ready for Round 12  
**Quorum:** 5/6 (architect + critic + technician + pragmatist + engineer — kimi abstained, provider Io Net error)

---

## Quorum

| Role | Model | Status | max_tokens | Notes |
|------|-------|--------|-----------|-------|
| Architect | codex-rescue | ✓ Active | — | separate session |
| Critic | google/gemini-3.1-pro-preview | ✓ Active (partial, truncated) | 2000 | truncated mid-response |
| Technician | deepseek/deepseek-v3.2 | ✓ Active | 2000 | |
| Philosopher | moonshotai/kimi-k2.5 | ✗ ABSTAIN | 12000 | provider "Io Net" returned finish_reason=None (not a token limit issue — different from prior rounds) |
| Pragmatist | minimax/minimax-m2.7 | ✓ Active | 8000 | |
| Engineer | xiaomi/mimo-v2-pro | ✓ Active | 10000 | |

**Note:** kimi-k2.5 abstained in Round 11 due to provider routing change (Round 10 used "Inceptron"/"Chutes", Round 11 used "Io Net"). `finish_reason=None` and `native_finish_reason=None` indicate provider-level failure, not a max_tokens issue.

---

## Fix Verification (W1, W2, W3)

| Fix | Technician | Critic | Pragmatist | Engineer | Verdict |
|-----|-----------|--------|-----------|---------|---------|
| W1 (RestoreHarness terminal stop check) | CORRECT | CORRECT | CORRECT | INCOMPLETE | ⚠️ |
| W2 (PhaseRouter.contextManager + BuildLoopConfig) | INCOMPLETE | CORRECT (partial) | CORRECT | CORRECT | ⚠️ |
| W3 (h.runID = len(turnRecords)) | CORRECT | — | CORRECT | CORRECT | ✅ |

**W1 INCOMPLETE (Engineer):** The guard `len(turnRecords) > 0` is wrong — it incorrectly fails to block a stopped session with no turn records. Correct guard: `len(session.History) > 0` (checks PhaseRecords, which always exist after any Stop() call).

**W2 INCOMPLETE (Technician):** `contextManager` field added to PhaseRouter and wired in BuildLoopConfig, but `NewPhaseRouter` constructor signature not shown — callers cannot set the field. Missing constructor update means the field stays nil.

---

## New Issues Found (Round 11)

### X1 HIGH — NewPhaseRouter constructor not updated (Technician — 1 vote)

**Severity:** HIGH  
**Location:** `PhaseRouter` initialization

`PhaseRouter` has a new `contextManager ContextManager` field (W2 fix), but the `NewPhaseRouter` constructor is not shown accepting it. All callers building a PhaseRouter will leave `contextManager = nil`, silently disabling context trimming.

**Fix in v12:**
```go
func NewPhaseRouter(
    phaseMap map[Role]PhaseConfig,
    registry *ToolRegistry,
    gateway ModelGateway,
    cm ContextManager, // Fix X1 (v12): explicit param, nil = passthrough
) *PhaseRouter {
    return &PhaseRouter{phaseMap: phaseMap, registry: registry, gateway: gateway, contextManager: cm}
}
```

---

### X2 CRITICAL — TurnRecord missing ToolCalls, breaking LLM API conversation replay (Engineer DOMAIN_VETO)

**Severity:** CRITICAL  
**Location:** `TurnRecord` struct / `Session.MessagesFromTurnRecords()` / `RunPhase` event loop

`TurnRecord` stored only `AssistantText` and `ToolResults`, not the tool calls themselves. `MessagesFromTurnRecords` reconstructed assistant messages without the `tool_calls` field. OpenAI/Anthropic APIs **reject** conversations where `tool_result` messages are not preceded by a matching `tool_calls` in the assistant message. This breaks multi-turn sessions entirely.

**Fix in v12:**
```go
// Event struct
type Event struct {
    Type       string
    ToolCalls  []ToolCall // Fix X2: "tool_call" event carries parallel tool calls
    // ...
}

// TurnRecord struct
type TurnRecord struct {
    ID            string
    Phase         Role
    UserMsg       Message
    AssistantText string
    ToolCalls     []ToolCall   // Fix X2: required for LLM API conversation replay
    ToolResults   []ToolResult
    CreatedAt     time.Time
}

// RunPhase event loop — new case
case "tool_call":
    turnRecord.ToolCalls = append(turnRecord.ToolCalls, ev.ToolCalls...)

// MessagesFromTurnRecords — assistant message now includes ToolCalls
out = append(out, Message{
    Role:      "assistant",
    Content:   tr.AssistantText,
    ToolCalls: tr.ToolCalls, // Fix X2: included for API correctness
})
```

---

### X3 HIGH — RecoverSession never loads PhaseRecords (implied by W1 analysis)

**Severity:** HIGH  
**Location:** `RecoverSession()` / `SessionStore` interface

`RecoverSession` called `store.Recover(sessionID)` but never called `store.LoadPhaseRecords()`. As a result, `session.History` was always empty and `session.Phase` was always whatever value was stored in the raw session record (which was set by `NewSession` to `RoleDiscover`, not derived from history). This made the W1 fix inoperative: `session.Phase` would never be `""` after restart even for stopped sessions.

**Fix in v12:**
```go
// SessionStore interface — new method
LoadPhaseRecords(sessionID string) ([]PhaseRecord, error)

// RecoverSession
func RecoverSession(sessionID string, store SessionStore) (*Session, error) {
    s, err := store.Recover(sessionID)
    if err != nil { return nil, err }
    phases, err := store.LoadPhaseRecords(sessionID)
    if err != nil { return nil, fmt.Errorf("load phase records: %w", err) }
    s.History = phases
    if len(phases) > 0 {
        s.Phase = phases[len(phases)-1].NextPhase // derives current phase
    }
    turns, err := store.LoadTurnRecords(sessionID)
    if err != nil { return nil, fmt.Errorf("load turn records: %w", err) }
    s.turnRecords = turns
    return s, nil
}
```

---

### W1' — RestoreHarness stop guard condition incorrect (Engineer)

**Severity:** MEDIUM (refinement of W1 fix)  
**Location:** `RestoreHarness()` stop detection condition

```go
// v11 (wrong):
if session.Phase == "" && len(session.turnRecords) > 0 {

// v12 (correct):
if len(session.History) > 0 && session.Phase == "" {
```

`len(turnRecords) > 0` is the wrong guard — it incorrectly allows Stop() to be bypassed when no turns were recorded. `len(session.History) > 0` checks PhaseRecords (which always exist after any phase transition or Stop() call), which is the correct invariant.

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
| 10 | v10→v11 | 3 | 3 | 3 |
| 11 | v11→v12 | 4 | 4 | 3 |

**Total: 49 issues found, 49 fixed. Issues per round: 7→5→7→5→3→8→2→2→3→3→4.**

---

## Convergence Signals

| Model | Verdict | Notes |
|-------|---------|-------|
| Technician | NOT_READY | W2 constructor gap = HIGH |
| Critic | NOT_READY (implied) | truncated, W1 CORRECT |
| Pragmatist | READY | no new issues from pragmatist's view |
| Engineer | NOT_READY | X2 CRITICAL DOMAIN_VETO |
| Philosopher | ABSTAIN | provider error |

X2 CRITICAL (engineer DOMAIN_VETO) and X1 HIGH applied in v12. The `MessagesFromTurnRecords` omission was a fundamental correctness bug for any session with tool use.

---

## Known False Alarms (do not re-raise in Round 12)

All prior false alarms (U3, S3, S4, V4, V5) remain dismissed.

---

## Round 12 Plan

Verify X1-X3 and W1' fixes in v12. Attempt to recover kimi quorum (try different provider routing or bump max_tokens). Full 6/6 quorum last achieved in R10 — target is unanimous CONVERGENCE: READY.

**Fixes to verify:**
- X1: `NewPhaseRouter` constructor accepts `cm ContextManager` 
- X2: `TurnRecord.ToolCalls` + `Event.ToolCalls` + `MessagesFromTurnRecords` correctness
- X3: `RecoverSession` loads `PhaseRecords` + derives `session.Phase`
- W1': `len(session.History) > 0` guard in RestoreHarness

*Raw responses: `/tmp/council_r11_responses.json`*  
*Design: `docs/plans/2026-04-10-sdp-mini-harness-design.md` (v12)*
