# F082: Compliance Control Mapping — Design Document

**Status**: Design
**Created**: 2026-04-26
**Feature**: F082 (Compliance Control Mapping)
**Beads**: sdplab-jpj5

## Overview

This document outlines the design for a compliance control mapping system that connects audit framework controls (SOC 2, ISO 27001, NIST 800-53, EU AI Act) to SDP evidence artifacts, verification methods, and residual risk assessment.

## Problem Statement

SDP currently generates evidence artifacts (via F078/F079) but lacks:

1. A formal control framework mapping layer
2. Clear traceability from control → evidence → frequency → verifier
3. Residual risk calculation methodology
4. Gate-time API for compliance verification

## Open Questions (Resolved)

### 1. What audit framework is the source of truth?

**Decision**: Multi-framework support with pluggable registry

- Primary frameworks: SOC 2, ISO 27001, NIST 800-53, EU AI Act
- Storage: `compliance/controls/` directory with YAML registry files
- Schema: `compliance-control.schema.json`

### 2. How does the control chain represent in code?

**Decision**: YAML registry + Go types + OPA policies

```
compliance/
├── controls/
│   ├── soc2.yaml
│   ├── iso-27001.yaml
│   ├── nist-800-53.yaml
│   └── eu-ai-act.yaml
├── mappings/
│   ├── sdp-evidence-to-controls.yaml
│   └── control-to-verifier.yaml
└── schema/
    └── compliance-control.schema.json
```

Go types: `internal/compliance/control.go`
OPA bundle: `compliance/policies/`

### 3. What is the gate-time API?

**Decision**: Extend `sdp doctor` with compliance mode

```bash
sdp doctor controls --framework soc2
sdp doctor controls --control "SOC2-CC6.1"
sdp doctor controls --risk-threshold medium
```

### 4. Does F083 (Policy Engine Enforcement) consume this registry?

**Decision**: Yes, shared schema

- F082 defines the control registry and mapping structure
- F083 provides OPA policy enforcement using this registry
- Both reference `compliance-control.schema.json`

## Architecture

### Components

1. **Control Registry**: YAML files defining control metadata
2. **Evidence Mapper**: Links SDP evidence types to controls
3. **Verifier Registry**: Defines how each control is verified
4. **Risk Calculator**: Computes residual risk from control gaps
5. **Doctor Command**: CLI interface for queries

### Data Flow

```
Control Framework (YAML)
    ↓
Evidence Mapper (YAML)
    ↓
SDP Evidence Events (JSON)
    ↓
Verifier Registry (YAML)
    ↓
Residual Risk Assessment
    ↓
OPA Policy Enforcement (F083)
```

## Schema Design

### Control Schema

```yaml
id: "SOC2-CC6.1"
title: "Logical and Physical Access Controls"
framework: "SOC2"
category: "Security"
evidence_requirements:
  - type: "evidence-envelope"
    frequency: "per-commit"
    verifier: "strataudit"
    source: "F078"
risk_impact: "high"
deprecation_timeline: null
```

### Evidence Mapping Schema

```yaml
control_id: "SOC2-CC6.1"
evidence_type: "handoff-coder"
mapping_rules:
  - field: "auth_context"
    requirement: "must_exist"
    frequency: "per-commit"
```

### Verifier Schema

```yaml
control_id: "SOC2-CC6.1"
verifier_type: "strataudit"
verification_method: "automated"
pass_threshold: 0.95
remediation_path: "docs/guides/access-control-remediation.md"
```

## Implementation Phases

### Phase 1: Control Schema & Registry (00-082-01)
- Create `compliance-control.schema.json`
- Define Go types
- Bootstrap SOC 2 control registry
- Implement `sdp doctor controls` basic query

### Phase 2: Evidence Mapper (00-082-02)
- Create evidence mapping schema
- Implement mapper logic
- Map existing SDP evidence types to SOC 2 controls
- Add `--evidence-coverage` flag to doctor

### Phase 3: Risk Assessment (00-082-03)
- Implement risk calculation algorithm
- Add residual risk reporting
- Create OPA policies for risk-based gating
- Add `--risk-threshold` flag to doctor

## Dependencies

- **F134-03** (Gate evidence enforcement): Closed - provides `evidence.json` shape
- **F078** (Trust Surface Consistency): Active - provides evidence envelopes
- **F079** (Enterprise Trust Pack): Active - provides maturity matrix
- **F083** (Policy Engine Enforcement): Design-pending - consumes this registry

## Success Criteria

1. All SOC 2 controls are mapped to at least one SDP evidence type
2. `sdp doctor controls` returns zero control gaps for SOC 2
3. Residual risk can be calculated per framework
4. OPA policies can consume the control registry
5. EU AI Act controls are represented (before 2026-08 deadline)

## Alternatives Considered

1. **JSON-only registry**: Rejected - YAML is more maintainable for humans
2. **Hardcoded Go types**: Rejected - Not flexible enough for multi-framework
3. **External control database**: Rejected - Adds external dependency, slower

## Next Steps

1. Review and approve this design document
2. Implement Phase 1 (00-082-01)
3. Update workstream status from `design-pending` to `backlog`
4. Break down into 3 workstreams (00-082-01, 00-082-02, 00-082-03)

## References

- [EU AI Act Compliance Lane](../workstreams/backlog/00-082-01.md)
- [F078 Trust Surface Consistency](../workstreams/backlog/00-078-01.md)
- [F079 Enterprise Trust Pack](../workstreams/backlog/00-079-01.md)
- [F134-03 Gate Evidence Enforcement](../workstreams/backlog/00-134-03.md)
