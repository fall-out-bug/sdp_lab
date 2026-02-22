# Feature: Removing Path A (FR-004)

Priority: P0
Effort: 1 day
Depends on: FR-001 working

## Problem

Path A (swarm-orchestrator + k8s_dispatch) — unplanned operator bypass. Creates a parallel execution path with a different consistency model. Every bugfix in Path A increases switching cost.

## Scope

### Remove

- `internal/pipeline/k8s_dispatch.go` + `k8s_dispatch_test.go` — kubectl exec bypass
- `SDP_DISPATCH_MODE=k8s` check from `internal/pipeline/executor.go`
- `deploy/k8s/workers/swarm-orchestrator-rbac.yaml` — RBAC for exec
- `SDP_DISPATCH_MODE: k8s` from `deploy/k8s/control/swarm-orchestrator.yaml`

### Extract and preserve

- Quality pipeline from `executor.go` (tests, evidence, provenance, PR gate) → `internal/quality/`
- `resolveRole()`, `resolveModel()` → shared helpers

### Mark as deprecated

- `cmd/swarm-orchestrator/` — dispatch via pipeline.ExecuteTask
- `scripts/run_local_swarm.sh` — go run on host

### Do not touch

- `cmd/swarm-worker/`, `cmd/swarm-reviewer/` — binaries are used inside agent pods
- `internal/federation/bridge.go` — intake via NATS, required for FR-005

## Acceptance Criteria

- `go build ./...` passes after removal
- `go test ./...` passes
- No references to dispatchK8s in codebase
- executor.go contains only local dispatch (test-only)
