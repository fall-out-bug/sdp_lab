# Orchestrator V2 Cutover Runbook

> **Version:** 1.0.0
> **Last Updated:** 2026-03-02
> **Feature:** F071 (Ralph Decommission and Orchestrator V2)

## Overview

This runbook provides the controlled migration path from legacy Ralph loop orchestration to FSM V2. Follow the checklist carefully to ensure safe cutover.

## Architecture Comparison

| Aspect | Legacy (Ralph) | FSM V2 |
|--------|----------------|--------|
| State Management | Implicit | Explicit typed states |
| Policy Checks | None | Per-transition hooks |
| Error Handling | Basic | Typed errors with retry |
| Rollback | Manual | Built-in |
| Telemetry | Limited | Full event stream |

## Cutover Phases

```
Phase 0: Preparation     → Phase 1: Shadow Mode     → Phase 2: Canary     → Phase 3: Full Cutover
   │                         │                          │                       │
   └─ Preflight checks       └─ Run both, use legacy    └─ 10% V2 traffic      └─ 100% V2 traffic
      Telemetry setup           for results              Monitor errors          Monitor errors
      Team training             Compare results          Rollback if needed      Rollback if needed
```

## Preflight Checklist

### Environment Configuration

- [ ] `SDP_ENVIRONMENT` set correctly (`staging`, `production`, `enterprise`)
- [ ] Feature flag provider configured (if using `auto` backend selection)
- [ ] Migration telemetry enabled and verified
- [ ] Monitoring dashboards deployed

### FSM V2 Readiness

- [ ] `internal/orchestrate/fsm_v2.go` deployed
- [ ] `internal/orchestrate/transitions.go` deployed
- [ ] `internal/orchestrate/migration_shim.go` deployed
- [ ] All FSM V2 tests passing

### Team Preparation

- [ ] Team trained on FSM V2 concepts
- [ ] On-call engineer briefed on rollback procedure
- [ ] Incident response playbooks updated
- [ ] Stakeholder notification sent

## Phase 1: Shadow Mode

Run both orchestrators in parallel, using legacy for actual results.

### Steps

1. Configure migration shim for shadow mode:
   ```bash
   export SDP_ORCHESTRATOR_BACKEND=auto
   export SDP_SHADOW_MODE=true
   export SDP_FALLBACK_ENABLED=true
   ```

2. Deploy with shadow mode enabled:
   ```bash
   # Apply configuration
   sdp config set orchestrator.backend auto
   sdp config set orchestrator.shadow_mode true
   
   # Verify configuration
   sdp config show orchestrator
   ```

3. Monitor telemetry:
   ```bash
   # Watch migration events
   sdp telemetry watch --type migration
   
   # Compare results
   sdp telemetry compare --legacy --v2 --window 1h
   ```

4. Validate for 24-48 hours before proceeding.

### Success Criteria

- Shadow runs complete without errors
- Telemetry shows no discrepancies
- Team comfortable with FSM V2 behavior

## Phase 2: Canary Deployment

Route 10% of traffic to FSM V2.

### Steps

1. Enable canary mode:
   ```bash
   export SDP_ORCHESTRATOR_BACKEND=auto
   export SDP_CANARY_PERCENTAGE=10
   ```

2. Or via feature flag:
   ```bash
   # LaunchDarkly example
   ld-cli flag set orchestrator-v2 --rule "10% rollout"
   ```

3. Monitor canary traffic:
   ```bash
   # Watch V2 traffic
   sdp telemetry watch --backend v2 --canary
   
   # Check error rate
   sdp telemetry errors --backend v2 --rate
   ```

### Rollback Trigger Criteria

Rollback immediately if ANY of:
- Error rate > 5% on V2 traffic
- Any critical system outage
- Data loss or corruption detected
- Stakeholder escalation

### Rollback Procedure

1. Set backend to legacy:
   ```bash
   sdp migration set-backend legacy
   ```

2. Verify legacy working:
   ```bash
   sdp migration verify --backend legacy
   ```

3. Clear migration flags:
   ```bash
   sdp migration clear-flags
   ```

4. Notify team:
   ```bash
   # Post to incident channel
   incident notify "Rolled back to legacy orchestrator - investigating V2 issues"
   ```

### Success Criteria

- Canary runs for 24 hours without rollback
- Error rate < 1% on V2 traffic
- No customer-impacting incidents

## Phase 3: Full Cutover

Route 100% of traffic to FSM V2.

### Steps

1. Increase canary to 100%:
   ```bash
   export SDP_ORCHESTRATOR_BACKEND=v2
   # OR
   ld-cli flag set orchestrator-v2 --rule "100% rollout"
   ```

2. Remove legacy fallback (after 1 week of stable V2):
   ```bash
   export SDP_FALLBACK_ENABLED=false
   ```

3. Monitor for 1 week before deprecating legacy.

### Final Validation

- [ ] All workstreams completing on V2
- [ ] Error rate within acceptable bounds
- [ ] Telemetry showing stable patterns
- [ ] Team comfortable with V2 operation

## Telemetry Reference

### Event Types

| Event Type | Description |
|------------|-------------|
| `backend_selection_failed` | Failed to select backend |
| `orchestration_started` | Orchestration began |
| `orchestration_completed` | Orchestration succeeded |
| `orchestration_failed` | Orchestration failed |
| `migration_started` | Migration between backends started |
| `migration_completed` | Migration completed |
| `migration_failed` | Migration failed |
| `rollback_started` | Rollback initiated |
| `rollback_completed` | Rollback completed |
| `rollback_failed` | Rollback failed |
| `dry_run_completed` | Dry run validation passed |
| `dry_run_transition_failed` | Dry run validation failed |

### Failure Classes

| Class | Description | Action |
|-------|-------------|--------|
| `validation` | Input validation failed | Fix inputs |
| `policy` | Policy check denied | Review policy |
| `timeout` | Operation timed out | Increase timeout or investigate |
| `resource` | Resource unavailable | Check system resources |
| `internal` | Internal error | Investigate logs |
| `unknown` | Unclassified error | Investigate |

### Query Examples

```bash
# Get all migration events for a workstream
sdp telemetry query --workstream 00-071-03

# Get all V2 failures in last hour
sdp telemetry query --backend v2 --failure-only --window 1h

# Get migration summary
sdp telemetry summary --type migration --period 24h
```

## Configuration Reference

### Environment Variables

| Variable | Values | Default | Description |
|----------|--------|---------|-------------|
| `SDP_ORCHESTRATOR_BACKEND` | `legacy`, `v2`, `auto` | `auto` | Backend selection mode |
| `SDP_ENVIRONMENT` | `development`, `staging`, `production`, `enterprise` | `development` | Environment type |
| `SDP_FALLBACK_ENABLED` | `true`, `false` | `true` | Enable fallback to legacy |
| `SDP_SHADOW_MODE` | `true`, `false` | `false` | Run both backends |
| `SDP_CANARY_PERCENTAGE` | 0-100 | 0 | Percentage of V2 traffic |
| `SDP_FEATURE_FLAG_KEY` | string | `orchestrator-v2` | Feature flag key |

### Migration Shim Configuration

```yaml
orchestrator:
  backend: auto
  environment: production
  feature_flag_key: orchestrator-v2
  fallback_enabled: true
  dry_run: false
```

## Troubleshooting

### Common Issues

#### Backend Selection Stuck on Legacy

**Symptoms:** FSM V2 never selected in auto mode

**Diagnosis:**
```bash
sdp migration debug --backend-selection
```

**Resolution:**
1. Check feature flag configuration
2. Verify environment variable
3. Check telemetry for selection events

#### High Error Rate on V2

**Symptoms:** >5% error rate after cutover

**Diagnosis:**
```bash
sdp telemetry errors --backend v2 --classify
```

**Resolution:**
1. Rollback to legacy immediately
2. Analyze failure classes
3. Fix root cause
4. Retry canary phase

#### Migration Events Not Recorded

**Symptoms:** No telemetry events visible

**Diagnosis:**
```bash
sdp telemetry verify --migration
```

**Resolution:**
1. Verify telemetry backend is running
2. Check telemetry configuration
3. Verify network connectivity

## Emergency Contacts

| Role | Contact |
|------|---------|
| On-Call Engineer | PagerDuty: SDP Platform |
| Platform Lead | Slack: @platform-lead |
| Incident Commander | Slack: #incident-response |

## References

- [ADR-003: Ralph Decommission](../decisions/ADR-003-ralph-decommission.md)
- [Enterprise Profile Policy](../policy/enterprise_profile.md)
- [FSM V2 Implementation](../../internal/orchestrate/fsm_v2.go)
- [Migration Shim](../../internal/orchestrate/migration_shim.go)
