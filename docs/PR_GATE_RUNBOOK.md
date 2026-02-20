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

`cmd/pr-publish` also dispatches a callback payload to required recipients (`issue-owner`, `orchestrator-audit`) on `pr-callbacks.v1` and appends a dispatch report note in Beads. The payload includes `trace.run_context_link` and `trace.evidence_context_link` for deterministic run/evidence navigation.

## Exit codes

- `0`: gate passed
- `2`: gate failed (evidence incomplete)
- `1`: operational/runtime error

## Live publish validation

For real validation of `cmd/pr-publish`:

1. create a feature branch with at least one commit
2. prepare evidence file for the target issue in `.sdp/evidence/<issue-id>.json`
3. run `cmd/pr-publish` and confirm it writes `trace.pr_url`, `trace.run_context_link`, and `trace.evidence_context_link`
4. confirm Beads issue notes include `PR callback dispatch report:`
