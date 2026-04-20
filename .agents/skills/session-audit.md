---
name: session-audit
description: Analyze Claude Code session logs — nudge rate, context-loss events, skill distribution, productivity ratio. Identifies sessions with poor autonomy.
version: 1.0.0
tags:
  - analytics
  - observability
  - process-health
requires_cli: []
compatibility:
  - claude-code
  - opencode
  - cursor
  - codex
---

# Session Audit

## Purpose

Analyze historical Claude Code sessions for process health signals.
Answers: where is the agent stopping autonomously, why review is re-run, which sessions had context loss, which were productive.

## When to Use

- After a sprint: understand where the agent needed manual intervention
- When tuning skills: baseline before/after comparing autonomy metrics
- Debugging a specific bad session: `@session-audit --session <id>`
- Regular process health check

## Invocation

```bash
# Build once
go build ./cmd/sdp-session-audit/...

# Summary of all sessions (aggregate + per-session)
./sdp-session-audit

# Limit to N largest sessions
./sdp-session-audit --top 10

# Sessions from last N days/hours/weeks
./sdp-session-audit --since 7d
./sdp-session-audit --since 24h

# Deep-dive single session (shows nudge message text)
./sdp-session-audit --session <id-prefix> --detail

# Machine-readable JSON
./sdp-session-audit --json
./sdp-session-audit --top 5 --json | jq '.aggregate'
```

## Metrics Explained

| Metric | What it means | Target |
|--------|---------------|--------|
| **Nudges** | Short user messages: "ок", "да", "продолжай" | < 5% of user msgs |
| **Context-loss** | "Continue from where you left off" | 0 per session |
| **Review iterations** | Times @review skill was called | ≥1 per feature session |
| **Productivity** | Issues closed per 100 user messages | > 1.0 |

## Interpreting Results

**High nudge rate (>10%):** agent stops and waits; check `build.md` Session Bootstrap, compaction recovery.

**High context-loss:** compaction is breaking flow; checkpoint.json not written or not read on resume.

**Review iterations = 0 in feature sessions:** @review not called; delivery-loop not being used.

**Productivity < 0.3:** session spent on exploration/planning, not delivery; or agent was stuck in loops.

**Top skills:** `superpowers:*` appearing means SDP skills aren't being used — should be `build`, `review`, `delivery-loop`.

## Output Location

Reads: `~/.claude/projects/-Users-fall-out-bug-projects-vibe-coding-sdp-lab/*.jsonl`
Override: `SESSION_AUDIT_DIR=<path> ./sdp-session-audit`
No files are written — read-only analysis.

## References

- `cmd/sdp-session-audit/main.go` — CLI entry point
- `internal/sessionaudit/audit.go` — parsing and aggregation logic
- `.agents/skills/delivery-loop.md` — the loop that eliminates manual nudges
- `.agents/skills/build.md` — Session Bootstrap (checkpoint recovery)
