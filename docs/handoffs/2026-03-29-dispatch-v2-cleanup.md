# Handoff: Dispatch v2 + Global Cleanup

**Date:** 2026-03-29
**Branch:** main (c96b92d)
**Status:** green — 45 packages pass, 0 open beads

## What was done

### 1. Dispatch v2 (7 issues, 3 dependency levels, all closed)

Multi-harness dispatch intelligence layer — routes tasks to optimal harness+model.

**Core (v1, from prior session):**
- `internal/dispatch/` — classify, route, limits, profiles, compare, invoker
- `internal/dispatch/harness/` — Claude, Codex, Cursor, OpenCode, z.ai adapters
- `cmd/sdp-dispatch/` — CLI with route, limits, profile subcommands
- Wired into orchestration via `internal/orchestrate/dispatch_integration.go`

**Gaps filled (v2, this session):**
- Cold start routing — 3 strategies: capability-heuristic, round-robin, fallback-chain
- Profile staleness — TTL-based decay (fresh/stale/expired), configurable thresholds
- DispatchDecision in WSStatus checkpoint — audit trail per workstream
- DispatchEvidence in in-toto attestation — provenance for dispatch decisions
- CLI bench/compare/status subcommands (bench is scaffolding, needs gastown)
- L1 Project Router (`internal/router/`) — intent to project+rig+phase
- Human gates (`internal/gate/`) — filesystem-backed decision points

**Architecture:**
```
L0: INTENT      OpenClaw / Kanban → Beads
L1: ROUTING     internal/router/ → project + rig + entry phase
L2: PLANNING    internal/planner/ → task DAG + scheduler (orphan, needs wiring)
L3: DISPATCH    internal/dispatch/ → best harness+model
L4: EXECUTION   Gastown → spawn, monitor, recover
L5: DATA        Beads → evidence, metrics, profiles
```

**Entry point:**
```go
// internal/orchestrate/dispatch_integration.go
inv := NewDispatchingInvoker(projectRoot)
```

### 2. Workflow discipline

- **Squash-only merge** — GitHub repo settings configured
- **OPA policies** — `.sdp/policies/main.rego` (P0 blocking, scope, beads linkage, evidence)
- **Git hooks** — `scripts/hooks/commit-msg` (conventional commits), `scripts/hooks/pre-push` (build+test gate)
- **Pattern doc** — `sdp/.claude/patterns/commit-discipline.md`
- **Branch protection** — requires GitHub Pro for private repo (hooks compensate)

**Flow:** `bd create → feature/<id> → work → squash merge main → bd close`

### 3. Code review + fixes

3 P0 + 4 P1 findings fixed:
- `route.go` — time.Now() consistency, epsilon float comparison
- `gate/beads.go` — path traversal prevention, atomic write
- `gate/wait.go` — context.Context support
- `dispatch_integration.go` — filepath.Join
- `cmd/sdp-dispatch/` — extracted shared helper

### 4. Global codebase cleanup

**Go modernization (18 files):** sort.Slice → slices.SortFunc everywhere
**Security (3 files):** timing-safe API key compare, HTTP server timeouts
**Error handling (6 files):** 8 HIGH discarded errors → slog logging
**Utilities:** `sdputil.AtomicWriteJSON`, `sdputil.UnmarshalJSON` (extracted, not yet migrated)
**Archived:** `internal/adapters/sdk/` → `archive/adapters-sdk/` (stub schemas)

### 5. Beads cleanup

- 78 OpenClaw test artifacts closed
- 14 wrong-scope issues (CW-*, F0xx) closed
- 7 dispatch issues created, worked, closed
- Final state: 447 total, 0 open

## Orphan packages — all intentional

9 packages with zero importers, all mapped to L0-L5 architecture:

| Package | Layer | Status |
|---------|-------|--------|
| `runtime/` | L4 | Backpressure controller — wire when dispatch has load |
| `authz/` | Enterprise | Multi-tenant RBAC (F074) |
| `modelgateway/` | L3 | Rich LLM provider abstraction (F073) — overlaps dispatch |
| `monitor/` | Ops | Stuck agent detector (F059) |
| `planner/` | L2 | Task DAG + scheduler — wire into orchestrate loop |
| `policy/` | L2-L4 | Evidence quality gate |
| `verify/` | Review | Multi-verifier quorum (F072) |
| `gate/` | L0 | Human decision points — wire into orchestrate loop |
| `router/` | L1 | Project Router — wire into orchestrate loop |

## Backlog (prioritized)

### Ready to migrate (Low effort, high value)
1. Migrate 12 call sites → `sdputil.AtomicWriteJSON`
2. Migrate 10 call sites → `sdputil.UnmarshalJSON`
3. `exec.Command` → `CommandContext` (12 locations)
4. `cmp.Or`, `strings.Cut`, `range int` (25 locations)

### Medium effort
5. `os.Setenv` → `t.Setenv` in tests (80+ locations)
6. Unexport ~40 internal-only symbols
7. Wire `router/`, `gate/`, `planner/` into orchestrate loop

### Large effort
8. Split 5 god files: `main.go` (1369), `control_tower_view.go` (1162), `update.go` (1122), `beads_sink.go` (820), `sdp-control/main.go` (708)
9. Wrap ~100 bare `return err` with `fmt.Errorf`
10. Consolidate `sdp/` submodule fork (6 duplicate packages)

## Key files

| File | Purpose |
|------|---------|
| `internal/dispatch/route.go` | Router with cold start + staleness |
| `internal/dispatch/invoker.go` | DispatchingInvoker |
| `internal/orchestrate/dispatch_integration.go` | Wiring: NewDispatchingInvoker + RecordDispatch |
| `internal/orchestrate/checkpoint.go` | WSStatus with WSDispatchInfo |
| `internal/orchestrate/attest.go` | DispatchEvidence in attestation |
| `internal/router/router.go` | L1 Project Router |
| `internal/gate/gate.go` | Human gate types |
| `.sdp/policies/main.rego` | OPA policy for CI |
| `scripts/hooks/commit-msg` | Conventional commit validation |
| `internal/sdputil/atomic.go` | Shared atomic write pattern |
