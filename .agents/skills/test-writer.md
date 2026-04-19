---
name: test-writer
description: Generate Go unit tests for uncovered functions from coverage reports.
version: 1.1.0
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

1. **Collect coverage** — run `go test -coverprofile=/tmp/cov.out ./target/...` to produce a binary coverage profile.
2. **Convert to function-level report** — run `go tool cover -func=/tmp/cov.out` to get per-function coverage percentages. This is the format `ParseCoverGaps` expects.
3. **Parse gaps** — replicate the logic of `internal/testwriter.ParseCoverGaps(output, threshold)` with threshold 80.0 (default) to identify functions below the bar.
4. **Read source** — for each gap, read the function body from the source file. Use AST-based extraction (see `ReadFuncSource`) to correctly handle braces inside string literals.
5. **Generate tests** — for each uncovered function, produce table-driven test cases:
   - Use `t.TempDir()` for any filesystem operations.
   - Use `testing.Short()` guards for external dependencies.
   - Test **behavior**, not implementation details.
   - Follow Go naming: `TestFuncName` with `t.Run` subtests.
6. **Write test file** — assemble a complete `_test.go` file per source file, including the `TestCase` struct definition, package declaration, and imports.
7. **Run and verify** — execute `go test ./target/... -short` to confirm all generated tests pass.
8. **Report delta** — re-run coverage scan and show before/after comparison.

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
   a. Read source code (AST-based extraction for accurate boundaries)
   b. Analyze parameters, return values, side effects
   c. Generate table-driven test cases
   d. Write to _test.go file
5. go test ./target/... -short  # verify all pass
6. Re-run coverage scan → show delta
```

## Helper Package (Reference Implementation)

The Go helper package `internal/testwriter/` provides reference implementations of the
core operations. Since it is an `internal/` package, it cannot be imported by external
code. Agents should **replicate these APIs** in the target package or use them as a
behavioral reference when generating test code.

| Function | Purpose |
|----------|---------|
| `ParseCoverGaps(output, threshold)` | Parse `go tool cover -func` output, return gaps below threshold |
| `GenerateTestStub(funcName, pkg, cases)` | Generate table-driven test function code (includes target package comment) |
| `ReadFuncSource(file, line)` | Extract function body from source file using go/ast (handles braces in strings correctly) |
| `FormatTestFile(pkg, imports, tests)` | Assemble complete `_test.go` file with package declaration, imports, and test functions |

Types:
- `CoverageGap{File, Function, Coverage, Line}` — a function with insufficient coverage. Note: `Line` is populated only when the `go tool cover -func` output includes line numbers (`file:line:` format); in the basic format it is 0.
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
- `internal/testwriter/` — Go reference implementation with parsing and generation functions (internal-only; replicate in target package)
- `internal/coveragegate/` — coverage enforcement gate for CI
