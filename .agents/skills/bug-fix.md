---
name: bug-fix
description: Systematic bug resolution — root cause first, minimal fix, reproducer test, full verification.
version: 1.0.0
tags: [debugging, quality, testing]
requires_cli: [go, bd]
compatibility: [claude-code, opencode, cursor, codex]
---

# Bug Fix

## Purpose

Given a bug report (beads issue, error log, failing test, or description), find the exact root cause,
apply a minimal targeted fix, and prove it's fixed with tests.

**Outcome:** root cause identified, reproducer test added, fix applied, all tests pass, beads closed.

## Use When

- A beads issue has `type: bug` or contains "QA:", "SRE:", "TechLead:" prefix
- A test is failing in CI with no obvious cause
- A user-visible behavior is wrong

**Do not use when:** the issue is a feature request, a performance improvement, or a docs update.

## MUST DO

1. **Read the bug report completely** — beads description, all linked issues, AC.
2. **Verify bug is reproducible** — grep for the named symbol/type/function. If code doesn't exist, report "Cannot reproduce: symbol not found" and stop.
3. **Locate code by searching** — grep for the symbol/function/type named in the report; read those files.
4. **Find the exact root cause** — one precise statement: "line X in file Y does Z when it should do W".
5. **Write a failing test first** — a test that reproduces the bug before any code change.
6. **Apply the minimal fix** — change only what's necessary to make the reproducer pass.
7. **Run all tests** — `go test ./...` must exit 0; integration tests use `testing.Short()`.
8. **Scan for related bugs** — grep for the same pattern in sibling files.
9. **Close beads** — `bd close <id> --reason="root cause: ... fix: ..."`.

## MUST NOT DO

- Fix symptoms (add nil-guard, return early) without addressing root cause.
- Create new code to "introduce" the bug — if the code doesn't exist, the bug is not reproducible.
- Skip the reproducer test — fixing without a test is not a fix, it's a guess.
- Change code outside the bug's scope.
- Claim done without running `go test ./...`.
- Suppress errors with `_ = err` or empty catch.

## Response Format

```
## Root Cause
[One paragraph: exact file, line, what the code does wrong, why]

## Reproducer Test
[Test name + what assertion it makes + before/after output]

## Fix
[File(s) changed, line range, before → after diff summary]

## Verification
go test ./... output — N passed, 0 failed

## Related Patterns
[Any sibling code with the same bug, or "none found"]
```

## References

- `docs/reference/go-patterns.md` — antipatterns section (especially panic, suppressed errors)
- `AGENTS.md` — quality gates, beads close workflow
