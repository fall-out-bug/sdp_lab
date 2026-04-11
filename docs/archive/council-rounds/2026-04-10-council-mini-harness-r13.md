# LLM Council Report: SDP Mini-Harness Design — Round 13

**Date:** 2026-04-11  
**Round:** 13  
**Spec version:** v13 → v14 (post Round 13 fixes applied)  
**Consensus:** 4/5 READY + 1 truncated → 1 fix → v14 ready for Round 14  
**Quorum:** 5/6 (architect + critic (truncated) + technician + pragmatist + engineer + philosopher)

---

## Quorum

| Role | Model | Status | max_tokens | Notes |
|------|-------|--------|-----------|-------|
| Architect | codex-rescue | ✓ Active | — | separate session |
| Critic | google/gemini-3.1-pro-preview | ✓ Partial (truncated mid-issue) | 3000 | Y1 CORRECT but truncated before completing new issue |
| Technician | deepseek/deepseek-v3.2 | ✓ Active | 3000 | |
| Philosopher | moonshotai/kimi-k2.5 | ✓ Active | 16000 | |
| Pragmatist | minimax/minimax-m2.7 | ✓ Active | 10000 | |
| Engineer | xiaomi/mimo-v2-pro | ✓ Active | 12000 | |

---

## Fix Verification (Y1)

| Fix | Technician | Critic | Pragmatist | Engineer | Philosopher | Verdict |
|-----|-----------|--------|-----------|---------|-------------|---------|
| Y1 (Event.ToolID + RunPhase "tool_end" + Loop emission) | CORRECT | CORRECT | CORRECT | CORRECT | CORRECT | ✅ |

**Y1 unanimous CORRECT (5/5 including truncated critic who verified Y1 before truncation).**

Engineer confirmed the full correlation chain: `ToolCall.ID → executeCalls result.ID → Event.ToolID → TurnRecord.ToolResults[].ID → Message.ToolCallID`. Also verified edge cases: parallel tool ordering (API matches by ID not position — correct), completion_signal ToolID, BeforeToolCall rejection paths.

---

## New Issues Found (Round 13)

### Z1 HIGH — Tool error not propagated to LLM in MessagesFromTurnRecords (Philosopher)

**Severity:** HIGH  
**Location:** `Session.MessagesFromTurnRecords()` — tool_result Message construction

When a tool execution fails (`ToolResult.Err != nil`), `r.Output` is empty (e.g., for "tool not in phase allowlist" or `BeforeToolCall` rejections). `MessagesFromTurnRecords` passes this empty string as `Message.Content` in the `tool_result` message. The LLM receives an empty tool result with no indication of what went wrong — it cannot recover, adjust its plan, or explain the failure to the user.

**Fix in v14:**
```go
for _, r := range tr.ToolResults {
    // Fix Z1 (v14): propagate error to LLM — without this, failed tools are silent
    content := r.Output
    if content == "" && r.Err != nil {
        content = fmt.Sprintf("Error: %v", r.Err)
    }
    out = append(out, Message{
        Role:       "tool_result",
        Content:    content,
        ToolCallID: r.ID,
    })
}
```

**Note:** Philosopher declared CONVERGENCE: READY despite raising this issue — the architectural design is correct, this is a functional detail that can be addressed as a minor fix without structural changes.

---

## Dismissed Issues (Round 13)

**Critic's truncated issue:** Critic began raising "TurnRecord flattens multi-step loops, corrupting LLM API message sequence" but was truncated before completing. This refers to the fact that a single `RunPhase` call may include multiple internal LLM turns (Loop's `for {}`), but `TurnRecord` captures all tool calls from all LLM turns in a single flat slice. **Assessment:** This is a design decision, not a bug. `TurnRecord` captures the externally-visible state of one RunPhase call (one human prompt → one agent response). The Loop's internal multi-turn context is managed by the Loop itself (appending tool results to `msgs` between LLM calls). `MessagesFromTurnRecords` reconstructs inter-RunPhase context, not intra-RunPhase sub-turns. The design intention is to present the agent's high-level turn history to the next RunPhase, abstracting internal reasoning steps. Not raising as Z2.

**Pragmatist's LOW issues:**
- GateEngine timeout race: accepted as MVP limitation (<1ms window, no structural change needed)
- AssistantText accumulation across turns: accepted (APIs accept multiple assistant messages in sequence, inter-RunPhase context is correct)

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
| 13 | v13→v14 | 1 | 1 | 1 |

**Total: 51 issues found, 51 fixed. Issues per round: 7→5→7→5→3→8→2→2→3→3→4→1→1. Clear convergence: last two rounds single-issue.**

---

## Convergence Signals

| Model | Verdict | Notes |
|-------|---------|-------|
| Technician | READY | No new issues |
| Pragmatist | READY | 2 LOW non-blocking |
| Engineer | READY | Detailed edge-case analysis — all clean |
| Philosopher | READY | Z1 raised but READY declared |
| Critic | Unknown (truncated) | Y1 CORRECT confirmed |

Strong convergence: 4 explicit READY votes, no domain vetoes. Z1 applied in v14 as minimal fix.

---

## Round 14 Plan

Final verification of Z1 in v14. Expect full unanimous READY.

**Fix to verify:**
- Z1: `MessagesFromTurnRecords` propagates `r.Err` to LLM when `r.Output == ""`

*Raw responses: `/tmp/council_r13_responses.json`*  
*Design: `docs/plans/2026-04-10-sdp-mini-harness-design.md` (v14)*
