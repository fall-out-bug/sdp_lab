# PR Gate Runbook (Private)

## Purpose

`cmd/pr-gate` enforces Strict Evidence before PR progression.

## Modes

- publish mode (default)
  - requires all 7 strict sections
  - requires `trace.pr_url`
- prepublish mode (`--prepublish`)
  - requires all 7 strict sections
  - allows empty `trace.pr_url` before PR is opened

## Usage

```bash
# Pre-PR creation check
go run ./cmd/pr-gate --issue <issue-id> --prepublish

# Publish/final check
go run ./cmd/pr-gate --issue <issue-id>
```

Publish PR and update evidence trace:

```bash
go run ./cmd/pr-publish --issue <issue-id> --title "..." --head <branch> --body-file <path>
```

## Exit codes

- `0`: gate passed
- `2`: gate failed (evidence incomplete)
- `1`: operational/runtime error
