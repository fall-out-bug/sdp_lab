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

## Docs Placement Policy

`docs/` has subfolders by topic. **New `.md` files must go in a subfolder**, not in `docs/` root. Rationale: `docs/` root already has ~100 flat files; adding more makes discovery impossible.

Choose by topic:

| Topic | Folder |
|---|---|
| Architecture / system design | `docs/architecture/` |
| Decisions (ADRs) | `docs/decisions/` |
| Phase contracts (Discovery, Delivery) | `docs/phases/` |
| Plans with date in name | `docs/plans/` |
| Runbooks (ops, incident response) | `docs/runbooks/` |
| Reference (CLI, catalogs, SOT docs) | `docs/reference/` |
| Research notes | `docs/research/` |
| Reviews / audits | `docs/reviews/` |
| Workstream files (`00-FFF-SS.md`) | `docs/workstreams/backlog/` |
| Superseded / historical | `docs/archive/` |
| Roadmap (canonical + supporting) | `docs/roadmap/` |
| Specs (contracts, schemas) | `docs/specs/` |
| SRE | `docs/sre/` |
| Security | `docs/security/` |
| Integrations | `docs/integrations/` |

Exceptions: `docs/CHANGELOG.md`, `docs/ARCHITECTURE.md`, `docs/MULTI-REPO-WORKFLOW.md`, `docs/MANIFESTO.md` — stable high-level entry points, remain at root.

If a new file doesn't fit any subfolder, create a new one or discuss in the PR. Do not drop into root as default.

## Top-Level Docs

Files in the repo root serve distinct purposes; don't merge or duplicate:

- `README.md` — human onboarding, CLI inventory, clone instructions
- `AGENTS.md` — agent operational rules, workflow, harness-agnostic
- `CLAUDE.md` — Claude Code project context (short header + RTK block managed by `rtk init`)
- `RTK.md` — universal RTK reference (used by Codex and as `@RTK.md` include in AGENTS.md)
- `VISION.md` — product vision (what is SDP, philosophy)
- `CONTRIBUTING.md` — this file

`sdp-doc-sync --mode check` validates links in `docs/**/*.md` and all root-level `.md` files.
