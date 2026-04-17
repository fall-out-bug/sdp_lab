---
name: review-readiness
description: Extends @review with --mode readiness: verify-before-completion gate returning a structured JSON report.
version: 1.0.0
tags:
  - review
  - verification
  - quality
requires_cli: []
compatibility:
  - claude-code
  - opencode
  - cursor
  - codex
---

> **This is NOT a standalone skill.** It extends the `@review` intent with
> `--mode readiness`. Invoke as `@review --mode readiness` — do not call it
> directly as a standalone skill.

# @review --mode readiness

## Purpose

Intent-mode extension of `@review` that runs a verify-before-completion gate.
Produces a JSON pass/fail report across five dimensions before work is declared done.

## Integration with @review

When a harness processes `@review --mode readiness`, it should:
1. Load this skill file as the mode-specific logic for the review intent.
2. Execute the checklist below instead of the default review flow.
3. Return the JSON report as the review result.

In skill manifests or intent routing tables, register this under the `@review`
intent with mode `readiness` — not as a top-level callable skill.

## Checklist

| # | Check | Command / Method | Pass criteria |
|---|-------|-------------------|---------------|
| 1 | Tests | `go test ./...` (or project-appropriate runner) | exit 0 |
| 2 | Coverage | compare current coverage to baseline | no regression >2pp |
| 3 | Docs | `docsync.CheckConsistency(root, strict=true)` | 0 error-severity findings |
| 4 | Orphans | workstream protocol validation | no orphaned WS files |
| 5 | TODOs | scan changed files for TODO/FIXME/HACK | 0 new occurrences |

## JSON Report Format

```json
{
  "ready": true,
  "checks": [
    {"name": "tests",    "status": "pass", "detail": "42 tests passed"},
    {"name": "coverage", "status": "pass", "detail": "78.3% (baseline 77.0%, delta +1.3pp)"},
    {"name": "docs",     "status": "pass", "detail": "0 findings"},
    {"name": "orphans",  "status": "pass", "detail": "0 orphaned workstreams"},
    {"name": "todos",    "status": "pass", "detail": "0 new TODO/FIXME/HACK"}
  ],
  "summary": "All checks pass"
}
```

On failure `"ready": false` and `summary` lists failing checks.

## Integration

Call from `@review` intent when `--mode readiness` is specified.
Go implementation: `internal/readiness` package (`ReadinessChecker.Check()`).
