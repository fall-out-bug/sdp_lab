# cmd — Agent Contract

## Scope

This subtree owns executable entrypoints for SDP CLIs, lab tools, smoke runners,
benchmarks, and operational utilities.

## Contract

Command packages should parse flags, validate inputs, call internal packages, and
return clear exit codes. Business logic belongs under `internal/`.

## Dependencies

Commands may depend on internal packages and stdlib. Avoid duplicating logic
between commands; extract shared behavior to an internal package when reuse is
real and current.

## Runtime Assumptions

Commands run from a developer workstation, CI, or harness dispatch. Any command
that writes files, calls external providers, executes subprocesses, or requires
credentials must make that behavior visible in help text and docs.

## Local Rules

- Keep `main.go` thin: flag parsing, validation, invocation, exit.
- Prefer explicit `--dry-run`, `--format json`, and evidence path flags for
  automation-facing commands.
- Do not make experimental or lab-only commands part of the stable release
  surface without updating release-surface docs.
- If command help and docs conflict, inspect code and update the canonical owner.
