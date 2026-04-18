---
name: smoke-test
description: Run CLI smoke scenarios and produce a structured pass/fail report.
version: 1.0.0
tags: [testing, smoke, cli, integration]
requires_cli: [go]
compatibility: [claude-code, opencode, cursor, codex]
---

# Smoke Test

## Purpose

Verify that SDP CLI binaries behave correctly from the outside — exit codes, JSON shape,
error handling. Catches regressions before they reach a real user or PR gate.

**Outcome:** structured report of pass/fail per scenario; non-zero exit if any fail.

## Use When

- After a PR that touches a `cmd/` binary or `internal/` package it depends on
- Before a release tag
- "Do the CLI tools still work?" sanity check

**Do not use when:** you need unit/package-level coverage (use `test-coverage`).

## MUST DO

1. **Test only observable behavior** — exit codes, stdout shape, stderr presence.
2. **Use fast checks** — avoid checks that compile the full project (`go-build`).
   Prefer `git-clean`, `--lint-skills`, `--format=json`, `--help` style invocations.
3. **Allow known non-zero exit codes** — `sdp-protocol-check` exits 1 (warnings) or
   2 (errors); both are valid. Only exit -1 (process killed) is always a bug.
4. **Assert JSON structure** — for `--format=json` flags, unmarshal and check field presence.
5. **Run before reporting done** — `go test -tags=smoke ./test/smoke/... -v`.

## MUST NOT DO

- Run `--check=go-build` in smoke tests (can take > 30s, hit context timeout).
- Assert on exact output text — only structure and exit codes.
- Leave binaries outside `t.TempDir()` — they must be cleaned up automatically.

## Workflow

```
1. go test -tags=smoke ./test/smoke/... -v      # run all scenarios
2. go test -tags=smoke ./test/smoke/... -run X  # run single scenario
3. scripts/run_smoke_tests.sh --json            # machine-readable JSON report
```

## Adding a New Scenario

Add a `TestBinaryName_BehaviorDescription` function to `test/smoke/smoke_test.go`.
Call `buildBinary(t, root, "cmd-name")` once, then `run(t, bin, args...)`.
Check exit code and optionally unmarshal JSON.

## Response Format

```
## Smoke Results
| Scenario | Status | Notes |
|----------|--------|-------|

## Failures (if any)
[scenario name]: [what failed]

## Next Steps
```

## References

- `test/smoke/smoke_test.go` — scenario implementations
- `scripts/run_smoke_tests.sh` — JSON report wrapper
- `docs/reference/go-patterns.md` — subprocess patterns
