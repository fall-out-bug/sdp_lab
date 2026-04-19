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
# Summary of all sessions (aggregate + per-session)
python3 scripts/session_audit.py

# Limit to N largest sessions
python3 scripts/session_audit.py --top 10

# Sessions from last N days/hours/weeks
python3 scripts/session_audit.py --since 7d
python3 scripts/session_audit.py --since 24h

# Deep-dive single session (shows nudge message text)
python3 scripts/session_audit.py --session <id-prefix> --detail

# Machine-readable JSON
python3 scripts/session_audit.py --json
python3 scripts/session_audit.py --top 5 --json | jq '.aggregate'
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

Script reads: `~/.claude/projects/-Users-fall-out-bug-projects-vibe-coding-sdp-lab/*.jsonl`
No files are written — read-only analysis.

## References

- `scripts/session_audit.py` — implementation
- `.agents/skills/delivery-loop.md` — the loop that eliminates manual nudges
- `.agents/skills/build.md` — Session Bootstrap (checkpoint recovery)
