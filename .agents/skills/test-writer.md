---
name: test-writer
description: Generate Go unit tests for uncovered functions from coverage reports.
version: 1.0.0
tags: [testing, coverage, generation]
requires_cli: [go]
compatibility: [claude-code, opencode, cursor, codex]
---

# Test Writer

## Purpose

Automatically generate Go unit tests for functions with insufficient test coverage.
Takes a coverage gap report (from `go tool cover -func` or `sdp coverage-scan`),
identifies uncovered functions, reads their source code, and produces table-driven
test stubs that pass on first run.

**Outcome:** new `_test.go` files with table-driven tests + coverage delta report.

## Use When

- "Write tests for uncovered functions in package X"
- "Generate tests from this coverage report"
- "Fill coverage gaps automatically"
- "Create test stubs for these functions"

**Do not use when:**
- Writing smoke/integration tests for CLI binaries (use `smoke-test`).
- Manually crafting tests for complex business logic (use `test-coverage`).
- The target is not Go code.

## MUST DO

1. **Accept coverage input** — run `go test -coverprofile=/tmp/cov.out ./target/...` and then `go tool cover -func=/tmp/cov.out`, OR accept pre-generated output.
2. **Parse gaps** — use `internal/testwriter.ParseCoverGaps(output, threshold)` with threshold 80.0 (default) to identify functions below the bar.
3. **Read source** — for each gap, use `internal/testwriter.ReadFuncSource(file, line)` to extract the function body. Understand what it does.
4. **Generate tests** — for each uncovered function, produce table-driven test cases:
   - Use `t.TempDir()` for any filesystem operations.
   - Use `testing.Short()` guards for external dependencies.
   - Test **behavior**, not implementation details.
   - Follow Go naming: `TestFuncName` with `t.Run` subtests.
5. **Write test file** — use `internal/testwriter.FormatTestFile(pkg, imports, tests)` to produce one `_test.go` per source file.
6. **Run and verify** — execute `go test ./target/... -short` to confirm all generated tests pass.
7. **Report delta** — re-run coverage scan and show before/after comparison.

## MUST NOT DO

- Import packages not already in `go.mod`.
- Delete or modify existing tests.
- Generate tests that only pass in CI (different PATH, binaries).
- Mock internal packages that are not at system boundaries.
- Skip the verification step (`go test` after generation).

## Workflow

```
1. go test -coverprofile=/tmp/cov.out ./target/...
2. go tool cover -func=/tmp/cov.out
3. Parse output → identify functions < threshold (default 80%)
4. For each uncovered function:
   a. Read source code
   b. Analyze parameters, return values, side effects
   c. Generate table-driven test cases
   d. Write to _test.go file
5. go test ./target/... -short  # verify all pass
6. Re-run coverage scan → show delta
```

## Helper Package

The Go helper package `internal/testwriter/` provides:

| Function | Purpose |
|----------|---------|
| `ParseCoverGaps(output, threshold)` | Parse `go tool cover -func` output, return gaps below threshold |
| `GenerateTestStub(funcName, pkg, cases)` | Generate table-driven test function code |
| `ReadFuncSource(file, line)` | Extract function body from source file |
| `FormatTestFile(pkg, imports, tests)` | Assemble complete `_test.go` file |

Types:
- `CoverageGap{File, Function, Coverage, Line}` — a function with insufficient coverage
- `TestCase{Name, Input, Expected}` — a single row in a table-driven test

## Response Format

```
## Tests Generated

| File | Function | Test Name | Cases |
|------|----------|-----------|-------|
| pkg/foo.go | Foo | TestFoo | 3 |
| pkg/bar.go | Bar | TestBar | 5 |

## Coverage Delta

| Function | Before | After |
|----------|--------|-------|
| Foo | 0.0% | 85.0% |
| Bar | 40.0% | 90.0% |

## Verification

- `go test ./target/... -short` — PASS (all tests)
- Coverage: 45.0% → 72.0% (+27.0pp)

## Remaining Gaps (if any)
- pkg/baz.go:Qux — complex external dependency, needs manual test
```

## References

- `.agents/skills/test-coverage.md` — manual coverage scanning workflow
- `internal/testwriter/` — Go helper package with parsing and generation functions
- `internal/coveragegate/` — coverage enforcement gate for CI
