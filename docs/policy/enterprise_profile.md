# Enterprise Profile Policy

> **Version:** 1.0
> **Last Updated:** 2026-03-01
> **Status:** Active

## Overview

The Enterprise Profile defines the operational constraints, compliance requirements, and governance policies for SDP deployments in enterprise environments.

## Activation

```bash
sdp up --profile enterprise
```

## Core Policies

### 1. Orchestrator Selection

| Orchestrator | Status | Notes |
|--------------|--------|-------|
| FSM V2 | **Required** | Default and only supported orchestrator |
| Ralph Loop | **Forbidden** | Deprecated, will fail fast |

**Enforcement**: The enterprise profile rejects any configuration that enables the Ralph loop. This is a hard constraint, not a warning.

### 2. Policy Checkpoints

All orchestration phases must pass through policy checkpoints:

| Checkpoint | Phase | Required Checks |
|------------|-------|-----------------|
| `validate` | Pending → Validated | Input schema, authz, rate limits |
| `assign` | Validated → Assigned | Resource availability, skill matching |
| `execute` | Assigned → Executed | Sandbox boundary, timeout, audit log |
| `review` | Executed → Reviewed | Evidence validation, quality gates |
| `finalize` | Reviewed → Completed | Artifact signing, retention policy |

### 3. Evidence Requirements

Enterprise deployments must produce evidence conforming to:

- **Format**: in-toto attestation with `coding-workflow/v1` predicate (per ADR-002)
- **Signing**: Sigstore keyless signing with OIDC identity
- **Storage**: Immutable storage with retention policy (default: 365 days)
- **Verification**: CI gates validate evidence before merge

### 4. Audit and Compliance

| Requirement | Implementation |
|-------------|----------------|
| Audit Trail | All state transitions logged with timestamp, actor, and decision |
| Retention | Logs retained per organizational policy (min 90 days) |
| Access Control | RBAC enforced on all orchestration actions |
| Encryption | At-rest and in-transit encryption required |

### 5. Error Handling

| Error Type | Action | Escalation |
|------------|--------|------------|
| Transient | Auto-retry with backoff (max 3 attempts) | Log only |
| Policy Violation | Fail immediately, emit audit event | Notify compliance team |
| Resource Exhaustion | Pause orchestration, queue for retry | Alert ops team |
| Security Event | Immediate halt, preserve evidence | Alert security team |

## Integration Points

### Observability Stack

Enterprise profile requires integration with:

- **Metrics**: Prometheus-compatible endpoint for orchestration metrics
- **Logs**: Structured JSON logging to centralized log aggregator
- **Traces**: OpenTelemetry traces for cross-service correlation

### Policy Engine

Integration with enterprise policy engine via `DecisionMaker` interface:

```go
type EnterprisePolicyHook struct {
    OPAEndpoint string
    Policies    []string // Policy package paths
}

func (h *EnterprisePolicyHook) Decide(event OrchestrationEvent) (RuntimeDecision, error) {
    // Query OPA with event context
    // Return decision based on policy evaluation
}
```

### Secret Management

Enterprise profile does NOT store secrets locally. Required integrations:

- **Vault**: HashiCorp Vault for secrets (recommended)
- **Cloud KMS**: AWS KMS, GCP KMS, or Azure Key Vault
- **SPIFFE/SPIRE**: Workload identity for service-to-service auth

## Migration from Ralph Loop

### Pre-Migration Checklist

- [ ] Audit existing Ralph loop runs for custom behaviors
- [ ] Map Ralph loop phases to FSM V2 states
- [ ] Implement `DecisionMaker` for enterprise policy hooks
- [ ] Test FSM V2 in staging with production-like workload
- [ ] Document rollback procedure

### Migration Command

```bash
sdp migrate --from ralph --to fsm-v2 --profile enterprise --dry-run
sdp migrate --from ralph --to fsm-v2 --profile enterprise
```

### Rollback (Cutover Window Only)

```bash
sdp up --profile enterprise --orchestrator=ralph --emergency-rollback
```

> **Note**: Rollback only available during cutover window (2026-04-01 to 2026-05-01). After sunset (2026-06-01), rollback is not supported.

## Compliance Frameworks

Enterprise profile supports compliance with:

| Framework | Coverage |
|-----------|----------|
| SOC 2 Type II | Audit logging, access control, encryption |
| ISO 27001 | Risk management, incident response |
| GDPR | Data minimization, retention, right to erasure |
| HIPAA | PHI handling (requires additional configuration) |

## Support

For enterprise support:
- Review ADR-003 for architectural decisions
- Consult `internal/orchestrate/fsm_v2.go` for implementation details
- Contact enterprise support team for migration assistance

## References

- [ADR-003: Ralph Loop Decommission](../decisions/ADR-003-ralph-decommission.md)
- [ADR-002: Standards Pivot](../decisions/ADR-002-standards-pivot.md)
- [OSS Combine Profile](../quickstart/OSS_COMBINE.md)
