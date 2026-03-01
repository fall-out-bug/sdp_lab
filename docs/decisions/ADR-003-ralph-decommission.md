# ADR-003: Ralph Loop Decommission for Enterprise Profile

> **Status:** Accepted
> **Date:** 2026-03-01
> **Context:** F071 Ralph Decommission and Orchestrator V2

## Decision

Deprecate and remove the primitive Ralph loop from the enterprise orchestration profile. Replace with typed FSM (Finite State Machine) orchestration with explicit transitions and policy checkpoints.

The Ralph loop remains available for OSS profile with appropriate compatibility notes.

## Context

The current Ralph loop is a primitive orchestration pattern that:
- Runs continuously without explicit state boundaries
- Lacks typed state transitions
- Has no policy checkpoints between phases
- Provides limited observability into orchestration state
- Cannot be safely interrupted and resumed

Enterprise deployments require:
- Predictable, auditable orchestration flow
- Policy enforcement at phase boundaries
- Explicit error handling and recovery
- Integration with existing enterprise governance tooling
- Safe interruption/resume capabilities

The Adapter SDK (F068) and Contract Tests (F068-03) establish typed contracts for orchestration events. This ADR aligns the orchestrator implementation with those contracts.

## Replacement Architecture

### FSM Orchestrator V2

```
[Pending] → (validate) → [Validated] → (assign) → [Assigned]
     ↑                                                      ↓
     ↑                                                 (execute)
     ↑                                                      ↓
[Completed] ← (finalize) ← [Reviewed] ← (review) ← [Executed]
```

### Key Components

1. **Typed States**: `Pending`, `Validated`, `Assigned`, `Executed`, `Reviewed`, `Completed`, `Failed`
2. **Explicit Transitions**: Each transition is a typed function with pre/post conditions
3. **Policy Checkpoints**: `validate`, `assign`, `execute`, `review`, `finalize` each invoke policy hooks
4. **Error Recovery**: `Failed` state with typed error events and recovery paths
5. **Observability**: State transitions emit `OrchestrationEvent` (per F068 SDK)

### Contract Alignment

The FSM emits `OrchestrationEvent` structs and receives `RuntimeDecision` responses, matching the Adapter SDK contracts:

```go
type OrchestrationEvent struct {
    Type      string          // "state_transition", "error", "checkpoint"
    State     string          // Current FSM state
    Timestamp time.Time       // Event time
    Payload   json.RawMessage // Event-specific data
}

type RuntimeDecision struct {
    Action    string // "continue", "pause", "rollback", "escalate"
    Reason    string
    PolicyRef string // Reference to policy that triggered decision
}
```

## Migration Timeline

| Phase | Date | Milestone |
|-------|------|-----------|
| RFC Published | 2026-03-01 | This ADR accepted |
| FSM V2 Implementation | 2026-03-15 | 00-071-02 complete |
| Migration Tooling | 2026-03-22 | 00-071-03 complete |
| Enterprise Cutover | 2026-04-01 | Enterprise profile default to FSM V2 |
| Ralph Sunset | 2026-06-01 | Ralph loop removed from codebase |

## Risk Analysis

### High Risk
- **State corruption during migration**: Mitigate with rollback support and staged migration
- **Policy gaps in FSM**: Mitigate with comprehensive policy checkpoint testing

### Medium Risk
- **OSS compatibility breakage**: Mitigate with explicit compatibility notes and migration guide
- **Performance regression**: Mitigate with benchmark suite and optimization phase

### Low Risk
- **Learning curve**: Mitigate with documentation and examples
- **Tooling integration**: FSM events are JSON, compatible with existing observability

## Rollback Policy

1. **Pre-cutover**: Both Ralph and FSM V2 available; enterprise can opt-in to FSM
2. **Cutover window (2026-04-01 to 2026-05-01)**: `--orchestrator=ralph` flag available for emergency rollback
3. **Post-sunset (2026-06-01)**: No rollback; Ralph code removed

Rollback trigger criteria:
- >5% of enterprise runs fail due to FSM bugs
- Critical security vulnerability in FSM implementation
- Data loss or corruption events

## Compatibility Notes for OSS Profile

The OSS profile (`sdp up --profile oss-combine`) may continue using Ralph loop:

| Feature | Enterprise | OSS |
|---------|------------|-----|
| Ralph Loop | Forbidden | Allowed |
| FSM V2 | Required | Optional |
| Policy Checkpoints | Required | Best-effort |
| Migration Support | Full | Self-service |

OSS users migrating to FSM V2 should:
1. Review `internal/orchestrate/fsm_v2.go` for state definitions
2. Implement `DecisionMaker` interface for custom policy hooks
3. Test with `--dry-run` before production use

## Enforcement

- Enterprise profile (`internal/profile/enterprise.go`) MUST fail fast if Ralph loop is invoked
- Lint rule: `no-ralph-enterprise` blocks code that imports Ralph in enterprise context
- CI gate: `sdp-protocol-check --strict` enforces profile compliance

## References

- [F068 Adapter SDK](../workstreams/backlog/00-068-02.md)
- [F068-03 Contract Tests](../workstreams/backlog/00-068-03.md)
- [00-071-02 Typed FSM orchestration](../workstreams/backlog/00-071-02.md)
- [00-071-03 Migration tooling](../workstreams/backlog/00-071-03.md)
- [Enterprise Profile Policy](../policy/enterprise_profile.md)
