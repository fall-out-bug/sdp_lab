# Dispatch Bridge Specification

Status: P0 — critical gap
Date: 2026-03-23
Scope: transport layer connecting control tower (ExecutionPacket) to OmO executor (opencode)
Depends on: `SDP_SPEC_DRIVEN_PIPELINE_CANON.md`
Owner: OmO (implementation)

## Problem

SDP has two disconnected execution paths:

1. **Control tower** builds `ExecutionPacket` and writes it to disk. Nobody reads it.
2. **Orchestrate loop** invokes `opencode` subprocess directly via `InvokeOpenCode`, bypassing `ExecutionPacket` entirely.

This means the control tower can plan, route, and packet-ize work — but cannot actually execute it. The orchestrator can execute — but is disconnected from the control tower's lifecycle, routing, and card state.

**This is the single critical gap.** Without it, the control tower is a planning shell, not an execution engine.

---

## Goal

Build a thin bridge that:
1. Reads `ExecutionPacket` produced by `DispatchCard`
2. Translates it into hydration context and prompt for opencode
3. Launches opencode subprocess via existing `LLMInvoker` interface
4. Writes heartbeat state to `FeatureCard` during execution
5. Translates opencode output into `ExecutorResultPacket`
6. Places result in `executor-results/` for auto-ingest by `OrchestrateOnce`

**Constraint**: reuse existing code. Do not redesign the orchestrate loop or replace opencode.

---

## Design

### Component: `internal/executor/bridge.go`

```go
// ExecutorBridge connects control tower ExecutionPackets to OmO opencode execution.
type ExecutorBridge struct {
    Store       *control.Store
    Invoker    orchestrate.LLMInvoker  // testable: DefaultLLMInvoker = opencode
    ProjectRoot string
}

// DispatchAndRun reads the execution packet for a dispatched card, launches opencode,
// writes heartbeat, and places the result for auto-ingest.
func (b *ExecutorBridge) DispatchAndRun(ctx context.Context, projectID, cardID string) (*ExecutorResultPacket, error)
```

### Flow

```
1. Load ExecutionPacket from dispatch dir
2. Load FeatureCard
3. Determine agent role from packet.ExecutorRole
   - "omo-implementation" → "implementer"
   - "review" → "reviewer"
   - Other roles → routing decision (TBD, P0 only handles implementation)
4. Build ContextPacket from ExecutionPacket fields
   - Workstream content: from packet.Objective + packet.ScopeIn
   - AcceptanceCriteria: from card.AcceptanceShape (or packet.ScopeOut)
   - QualityGates: from AGENTS.md (existing logic)
   - DriftStatus: git status (existing logic)
5. Build prompt from ContextPacket (existing buildPromptWithContext)
6. Write prompt provenance (existing WritePromptProvenance)
7. Update FeatureCard: executor_runtime_state = "running", executor_session_id, executor_started_at
8. Invoke opencode via LLMInvoker.Invoke(ctx, dir, agent, prompt)
9. Parse result:
   - exit code 0 + commit hash → success
   - exit code non-zero → failed
   - Extract artifacts (commit hash, changed files)
10. Build ExecutorResultPacket
11. Write result JSON to executor-results/
12. Update FeatureCard: executor_runtime_state = "completed"
```

### Heartbeat

For P0, heartbeat is minimal:
- Set `executor_runtime_state = running` before invoke
- Set `executor_runtime_state = completed` (or `failed`) after invoke
- No periodic heartbeat goroutine (P0 is synchronous/blocking)

Full heartbeat with periodic updates is P1 (see `EXECUTION_HEARTBEAT_RUNTIME_RECONCILIATION_SPEC.md`).

### Role Mapping (P0)

| ExecutionPacket.ExecutorRole | opencode --agent | Behavior |
|------------------------------|-------------------|----------|
| omo-implementation | implementer | Full build phase |
| review | reviewer | Review phase |
| _other_ | implementer | Default fallback |

### Result Packet

```go
func translateResult(packet *control.ExecutionPacket, output string, exitCode int) *control.ExecutorResultPacket
```

Mapping:
- exitCode == 0 → `status: success`, extract commit hash as artifact
- exitCode != 0 → `status: failed`, output as summary
- Always include `parent_feature_id` from packet

Result placed in `.sdp/control/executor-results/<card-id>-<timestamp>.json` for auto-ingest.

---

## Integration Points

### Wire into DispatchCard

Option A (conservative): separate command
```bash
sdp dispatch next --execute   # dispatch + bridge + wait for result
```

Option B (integrated): modify `DispatchCard` to accept `--run` flag
```bash
sdp-control card-execute --project <id> --id <card-id> --run
```

**Recommendation**: Option A. Keeps DispatchCard as pure packet builder (testable, no side effects). New command wraps dispatch + bridge + ingest.

### Auto-ingest

After bridge writes result to `executor-results/`, the next `OrchestrateOnce` call will automatically:
1. Detect the new result file
2. Call `ingestExecutorResultFile`
3. Update FeatureCard status (executing → done/reviewing/blocked/needs_input)
4. Rebuild snapshots

This means `sdp dispatch next --execute` can be followed by `sdp orchestrate once` to close the loop.

---

## What This Is NOT

- Not a daemon or scheduler
- Not an A2A server (that's P3)
- Not a replacement for the orchestrate loop
- Not a new agent framework
- Not async (P0 is synchronous with timeout)

---

## Exit Criteria

- [ ] `internal/executor/bridge.go` exists with `DispatchAndRun`
- [ ] `sdp dispatch next --execute` dispatches and runs a ready card end-to-end
- [ ] FeatureCard is updated with executor session metadata
- [ ] Result is auto-ingested by `OrchestrateOnce`
- [ ] Prompt provenance is written before each invoke
- [ ] Tests cover happy path and failure path
- [ ] Existing `DispatchCard` tests still pass (no regression)

---

## Future (Post-P0)

- P1: Periodic heartbeat goroutine + stale detection
- P2: Async dispatch (launch + poll + collect)
- P3: A2A HTTP endpoint wrapping this bridge
- P4: Multi-repo parallel execution

---

## Related

- `SDP_SPEC_DRIVEN_PIPELINE_CANON.md` — pipeline architecture
- `internal/orchestrate/invoke_opencode.go` — existing LLM invocation
- `internal/orchestrate/hydrate.go` — existing hydration logic
- `internal/control/routing.go` — ExecutionPacket contract
- `internal/control/update.go` — DispatchCard implementation
- `internal/control/orchestrate_once.go` — auto-ingest logic
- `EXECUTION_HEARTBEAT_RUNTIME_RECONCILIATION_SPEC.md` — heartbeat spec
