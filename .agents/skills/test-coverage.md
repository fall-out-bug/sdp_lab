---
name: test-coverage
description: Find uncovered code paths and write unit tests that pass on first run.
version: 1.0.0
tags: [testing, coverage, quality]
requires_cli: [go]
compatibility: [claude-code, opencode, cursor, codex]
---

# Test Coverage

## Purpose

Find untested code paths in the codebase and write Go tests that close those gaps.
Tests must pass on first run without manual intervention.

**Outcome:** coverage delta report + new test functions committed.

## Use When

- "Increase coverage for package X"
- "Find uncovered paths"
- "Write tests for these functions"

**Do not use when:** writing smoke/integration tests for CLI binaries (use `smoke-test`).

## MUST DO

1. **Scan before writing** — run `go test -coverprofile=/tmp/cov.out ./pkg/...` first.
2. **Read the function** — understand what each uncovered branch does before writing a test.
3. **Test the behavior, not the implementation** — assert on outputs and side-effects.
4. **Guard integration tests** — any test that calls an external binary or network must start with `if testing.Short() { t.Skip(...) }`.
5. **Use `t.TempDir()`** for temporary files — never hardcode paths.
6. **Run the new tests before reporting done** — `go test ./pkg/... -run TestNewFunc`.
7. **Report the delta** — show coverage before and after.

## MUST NOT DO

- Mock internal packages that are not at system boundaries.
- Write tests that pass only in CI (different PATH, different binaries).
- Delete or modify existing tests to make new ones pass.
- Import packages not already in go.mod.

## Workflow

```
1. go test -coverprofile=/tmp/cov.out ./target/...
2. go tool cover -func=/tmp/cov.out | sort -t% -k1 -n
3. Identify functions < 80% or == 0%
4. Read function source (file:line from cover output)
5. Write test cases for uncovered branches
6. go test ./target/... -run TestNew
7. Re-run cover scan → show delta
```

## Response Format

```
## Coverage Delta
| Function | Before | After |
|----------|--------|-------|

## Tests Added
- `TestFuncName` in `pkg/file_test.go:line` — what it covers

## Remaining Gaps (if any)
```

## References

- `docs/reference/go-patterns.md` — test stubs, testing.Short() convention
