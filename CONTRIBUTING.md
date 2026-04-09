# Contributing to SDP Lab

## Quick Start (5 minutes)

### Prerequisites
- Go 1.26+
- git
- gh (GitHub CLI) — for PR workflows
- bd (beads) — for issue tracking (optional for code changes)
- golangci-lint — for linting (optional)

### Setup
```bash
git clone --recurse-submodules https://github.com/fall-out-bug/sdp_lab
cd sdp_lab
go build ./...
go test ./internal/... -short -count=1
```

### Verify Everything Works
```bash
make test              # Full test suite
make quality-go        # Build + test + vet
```

## Development Workflow

1. Branch from `main`: `git checkout -b feature/FXXX-short-name`
2. Make changes
3. Run tests: `go test ./... -count=1`
4. Run quality gates: `make quality-go`
5. Commit and push
6. Open draft PR early: `gh pr create --draft --base main`

## Project Structure

| Directory | Purpose |
|-----------|---------|
| `cmd/sdp/` | Main CLI (discover, dispatch, board, tower, doctor) |
| `cmd/sdp-*/` | Standalone binaries (orchestrate, evidence, ci-loop, etc.) |
| `internal/control/` | Feature card store, project management |
| `internal/orchestrate/` | FSM, checkpoints, state machine |
| `internal/dispatch/` | Routing, harness selection, profiles |
| `internal/discovery/` | LLM-driven discovery pipeline |
| `internal/executor/` | Execution loop, OMO client, bridge |
| `internal/evidence/` | Attestation, validation, provenance |
| `internal/kernel/` | Core runtime adapter interfaces |
| `internal/sdputil/` | Shared utilities |

## Testing

```bash
# Fast: internal packages only
make test-internal

# Single package
go test ./internal/discovery/... -v -count=1

# With coverage
make coverage

# Integration tests (skipped by default)
go test ./... -count=1  # integration tests use t.Skip() or -short
```

## Code Style

- Follow standard Go conventions
- Use `slog` for structured logging (not `log` or `fmt.Print`)
- Wrap errors with context: `fmt.Errorf("context: %w", err)`
- Table-driven tests preferred
- No `interface{}` — use `any` or concrete types

## Architecture

Two-repo structure:
- **sdp_lab** (this repo): Go code, K8s manifests, roadmap
- **sdp** (submodule at `sdp/`): Protocol schemas, prompts, hooks

Read `AGENTS.md` for the full multi-repo workflow.
