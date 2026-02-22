# FR-015: Automated Feature Orchestrator

Priority: P0
Effort: 5d
Dependencies: FR-002 (AgentRun CRD), FR-011 (cross-project registry), FR-005 (NATS bridge)

## Problem

Orchestration is currently manual — AgentRun must be created manually or a script must be run. An automated orchestrator is needed that:
- Monitors Beads backlog for ready tasks
- Prioritizes by dependencies and criticality
- Automatically creates AgentRun CRD for each task
- Balances load across available worker pods
- Tracks lifecycle until completion

## Existing Code

| Component | Status | What it does |
|-----------|--------|-------------|
| `swarm-orchestrator` | ⚠️ Path A | Polls Beads, dispatches via kubectl exec |
| `internal/federation/bridge.go` | ✅ Active | Polls ready issues, publishes to NATS |
| `internal/federation/aggregator.go` | ✅ Active | Aggregates across projects |
| `internal/swarm/coordinator.go` | ✅ Active | Claim deduplication |
| `internal/orchestrator/` | ✅ Active | Run tracker, scheduler |
| `autonomy-worker` | ✅ Active | Picks next task from Beads |

## Design

### Architecture (replaces swarm-orchestrator)

```
                    ┌──────────────────────────────────┐
                    │     feature-orchestrator          │
                    │     (K8s Deployment)              │
                    │                                    │
Beads ready ──────►│  1. Poll ready issues (all projects)│
NATS intake ──────►│  2. Priority queue (P0 > P1 > P2)  │
                    │  3. Dependency check               │
                    │  4. Worker capacity check          │
                    │  5. Create AgentRun CRD            │
                    │  6. Monitor until terminal         │
                    │  7. Close Beads / escalate         │
                    └──────────────┬───────────────────┘
                                   │
                                   ▼
                    ┌──────────────────────────────────┐
                    │     AgentRun CRD                  │
                    │     (via adapter-controller)      │
                    └──────────────────────────────────┘
```

### Orchestration Loop

```go
func (o *FeatureOrchestrator) Run(ctx context.Context) error {
    for {
        // 1. Gather ready tasks from all projects
        tasks := o.aggregator.ReadyTasks()
        
        // 2. Filter by dependencies (all deps closed)
        eligible := o.filterByDeps(tasks)
        
        // 3. Sort by priority, then age
        sort.Sort(ByPriorityThenAge(eligible))
        
        // 4. Check worker capacity
        capacity := o.checkCapacity()
        
        // 5. Dispatch top N tasks
        for i := 0; i < min(len(eligible), capacity); i++ {
            task := eligible[i]
            
            // Acquire lock (prevent duplicate dispatch)
            if !o.lockManager.TryAcquire(task.IssueID) {
                continue
            }
            
            // Create AgentRun CRD
            run := o.buildAgentRun(task)
            if err := o.k8s.Create(ctx, run); err != nil {
                o.lockManager.Release(task.IssueID)
                continue
            }
            
            o.trace.Emit("dispatched", task.IssueID, run.Name)
        }
        
        // 6. Monitor active runs
        o.monitorActiveRuns(ctx)
        
        // 7. Sleep before next cycle
        time.Sleep(o.pollInterval)
    }
}
```

### Capacity Management

```go
type CapacityManager interface {
    // How many more AgentRuns can we start?
    AvailableSlots() int
    // Resource-aware: check node resources
    CanSchedule(requirements ResourceRequirements) bool
}
```

Default: `maxConcurrentRuns` from ConfigMap (start with 1, scale up).

### Integration with FR-014

Telemetry analyzer creates issues → orchestrator automatically picks up ready issues → AgentRun → agents → PR.

**Full autonomous cycle:**
```
Telemetry Analyzer → Beads Issue → Orchestrator → AgentRun → Agents → PR → Review → Merge → Telemetry
                                                                                              ↓
                                                                              (loop back to Analyzer)
```

## Acceptance Criteria

- [x] Runs as K8s Deployment in sdp-control (WS-015-01)
- [x] Polls all registered projects via federation/aggregator
- [x] Creates AgentRun CRD (not kubectl exec)
- [x] Priority queue: P0 → P1 → P2 → P3, then by age (aggregator)
- [ ] Dependency gating: all deps must be closed (deferred: beads dep API)
- [x] Worker capacity management (configurable max concurrent)
- [x] RunLockManager prevents duplicate dispatch (LeaseLockManager in WS-015-01)
- [ ] Monitors active runs, escalates on timeout
- [ ] Closes Beads issues on successful completion
- [ ] NATS events: dispatched, completed, failed, escalated
- [x] ConfigMap: poll interval, max concurrent, project filter
- [x] Replaces swarm-orchestrator's dispatch loop
