---
name: parallel-dispatch
description: Harness-neutral parallel subtask dispatch — decision tree, safety guards, and result aggregation.
version: 1.0.0
tags:
  - orchestration
  - efficiency
requires_cli: []
compatibility:
  - claude-code
  - opencode
  - cursor
  - codex
---

# Parallel Dispatch

## Purpose

Dispatch ≥2 independent tasks to concurrent subagents for faster completion.
Each subagent gets isolated context and a focused scope.

## Decision Tree

```
Multiple tasks?
  no  -> single agent
  yes -> are they independent (no shared mutable state, no edit collisions)?
    no  -> sequential (task B depends on task A output)
    yes -> parallel dispatch
```

**Use when:** ≥2 tasks with disjoint scope (different files, subsystems, or domains).
**Do not use when:** tasks share mutable state, edit overlapping files, or one depends on another's output.

## Rules

1. **Independence.** Each subagent owns a non-overlapping scope. No two agents edit the same file.
2. **Self-contained prompts.** Each prompt includes all context needed — no agent reads another's output.
3. **Clear deliverable.** Every prompt specifies what the agent must return (summary, diff, pass/fail).
4. **Constraint boundaries.** State explicit constraints: files to touch, files to avoid, behaviors to preserve.

## Harness Mapping

| Harness | Mechanism |
|---------|-----------|
| Claude Code | `Agent` tool / `TaskCreate` + `TaskUpdate` |
| OpenCode | `@agent <role>` dispatch |
| Cursor | Agent panel parallel sessions |
| Codex CLI | Subprocess or separate invocation |

## Safety Checks

- **Max concurrent:** 5 subagents. Queue beyond that.
- **Timeout:** each subagent gets a reasonable bound; fail fast on stall.
- **Error isolation:** one subagent failure does not cancel others. Collect partial results.
- **No shared writes:** verify no file-path overlap before dispatch.

## Result Aggregation

1. Collect all subagent outputs.
2. Check for conflicts (overlapping edits, contradictory conclusions).
3. Integrate non-conflicting results.
4. Run full validation (tests, lint) on the integrated state.
5. Report: per-agent status + overall pass/fail.

## Anti-Patterns

- Dispatching without verifying independence — causes merge conflicts.
- Vague prompts ("fix everything") — agent loses focus, wastes tokens.
- Ignoring partial failures — silent data loss or incomplete work.
- Over-dispatching trivial tasks — overhead exceeds sequential execution.
- Shared mutable state between agents — race conditions, corrupted output.
