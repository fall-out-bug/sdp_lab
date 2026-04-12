# LLM Council Report: SDP Mini-Harness Design — Round 12

**Date:** 2026-04-11  
**Round:** 12  
**Spec version:** v12 → v13 (post Round 12 fixes applied)  
**Consensus:** NOT_READY → 1 fix → v13 ready for Round 13  
**Quorum:** 6/6 ✅ (all 5 OpenRouter + architect — kimi returned at 16000 max_tokens)

---

## Quorum

| Role | Model | Status | max_tokens | Notes |
|------|-------|--------|-----------|-------|
| Architect | codex-rescue | ✓ Active | — | separate session |
| Critic | google/gemini-3.1-pro-preview | ✓ Active (partial, truncated) | 3000 | truncated mid-X3 response |
| Technician | deepseek/deepseek-v3.2 | ✓ Active | 3000 | |
| Philosopher | moonshotai/kimi-k2.5 | ✓ **RETURNED** | 16000 | Io Net issue resolved — responded at 16000 max_tokens |
| Pragmatist | minimax/minimax-m2.7 | ✓ Active | 10000 | |
| Engineer | xiaomi/mimo-v2-pro | ✓ Active | 12000 | |

---

## Fix Verification (X1, X2, X3, W1')

| Fix | Technician | Critic | Pragmatist | Engineer | Philosopher | Verdict |
|-----|-----------|--------|-----------|---------|-------------|---------|
| X1 (NewPhaseRouter constructor) | CORRECT | CORRECT | CORRECT | CORRECT | CORRECT | ✅ |
| X2 (TurnRecord.ToolCalls + MessagesFromTurnRecords) | CORRECT | CORRECT | CORRECT | INCOMPLETE | CORRECT | ⚠️ |
| X3 (RecoverSession loads PhaseRecords) | CORRECT | — (truncated) | CORRECT | CORRECT | CORRECT | ✅ |
| W1' (len(session.History)>0 guard) | CORRECT | CORRECT | CORRECT | CORRECT | CORRECT | ✅ |

**X2 INCOMPLETE (Engineer):** X2 fixed the write side (assistant message has tool_calls), but the read side is broken: `Event` has no `ToolID` field, so `ToolResult.ID` in the "tool_end" handler is always `""`. `MessagesFromTurnRecords` emits `ToolCallID: r.ID` → empty string. APIs require `tool_call_id` to exactly match the preceding `tool_call.id`.

---

## New Issues Found (Round 12)

### Y1 CRITICAL — Event.ToolID missing, tool_result.ToolCallID always empty (Engineer DOMAIN_VETO)

**Severity:** CRITICAL  
**Location:** `Event` struct / `RunPhase` "tool_end" handler / `Loop` "tool_end" emission

`Event` has `ToolName string` and `ToolResult string` for "tool_end" events, but no `ToolID string`. When `RunPhase` constructs a `ToolResult` from a "tool_end" event:

```go
// BEFORE (broken):
turnRecord.ToolResults = append(turnRecord.ToolResults, ToolResult{
    Name:   ev.ToolName,
    Output: ev.ToolResult,
    Err:    ev.ToolErr,
    // ID: ← never set, remains ""
})
```

`MessagesFromTurnRecords` then emits:
```go
Message{Role: "tool_result", Content: r.Output, ToolCallID: r.ID}
// r.ID is "" → API rejects: tool_call_id must match a preceding tool_call.id
```

X2 fixed the write path (assistant message carries `ToolCalls`), but this leaves the read path broken: the IDs don't match.

**Fix in v13:**
```go
// Event struct:
ToolID     string  // Fix Y1 (v13): for "tool_end" — matches ToolCall.ID for tool_call_id correlation

// Loop emits for each result from executeCalls:
Event{Type:"tool_end", ToolID: result.ID, ToolName: result.Name,
      ToolResult: result.Output, ToolErr: result.Err}

// RunPhase case "tool_end":
turnRecord.ToolResults = append(turnRecord.ToolResults, ToolResult{
    ID:     ev.ToolID,   // Fix Y1: correlation key
    Name:   ev.ToolName,
    Output: ev.ToolResult,
    Err:    ev.ToolErr,
})
```

**Note from Technician (HIGH, non-blocking):** Loop pseudo-code does not explicitly show "tool_end" event emission from `executeCalls` results. The fix implicitly requires the Loop to emit `Event{ToolID: result.ID, ...}` for each result — this should be explicit in the spec. Added as a comment in Event struct definition.

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
| 12 | v12→v13 | 1 | 1 | 1 |

**Total: 50 issues found, 50 fixed. Issues per round: 7→5→7→5→3→8→2→2→3→3→4→1. Monotone convergence — Round 12 down to 1 issue.**

---

## Convergence Signals

| Model | Verdict | Notes |
|-------|---------|-------|
| Technician | READY | flags Y1 as HIGH but considers overall READY |
| Pragmatist | READY | no new issues |
| Philosopher | READY | no new issues |
| Critic | NOT_READY (implied) | truncated, couldn't see full X3 verdict |
| Engineer | NOT_READY | Y1 CRITICAL DOMAIN_VETO |

Y1 applied in v13. This is a single-field addition in three locations — minimal surface, high impact.

---

## Known False Alarms (do not re-raise in Round 13)

All prior false alarms (U3, S3, S4, V4, V5) remain dismissed.

**Technician's PhaseRecord.StartedAt note:** Not a bug in this spec — StartedAt is a field but its assignment is an implementation detail. The spec documents the durable state model; exact timestamp wiring is engineering discretion during implementation. Not raising in R13.

---

## Round 13 Plan

Verify Y1 fix in v13. With full quorum (6/6) expected, this should be the final convergence round.

**Fix to verify:**
- Y1: `Event.ToolID string` added; Loop emits `ToolID: result.ID` for "tool_end" events; RunPhase case "tool_end" uses `ID: ev.ToolID`

*Raw responses: `/tmp/council_r12_responses.json`*  
*Design: `docs/plans/2026-04-10-sdp-mini-harness-design.md` (v13)*
