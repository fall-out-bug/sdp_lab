# SDP Policy Runtime — v1 Contract

## Context Objects
- `Decision`: Primary output (Allowed bool, Explanation ExplainResult)
- `ExplainResult`: Human-readable rationale via ExplainMode (Basic/Detailed/Internal)

## Dependencies
- `internal/evidence`: CodingWorkflowStatement, DiscrepancyReport (wrapped in EvidenceInput)
- **Risk**: Evidence schema changes break policy API
- Stdlib only: context, encoding/json, fmt, strings

## Configuration
Discrepancy thresholds (GateConfig):
- Critical: 0, High: 0, Medium: 5, Low: 20

Attestation:
- RequireSignedAttestation: true (default)
- Verify via evidence.NewSigner().VerifyAttestation()

## Evaluation Contract
- **Ordering**: Severity descending (Critical→High→Medium→Low) for determinism
- **Short-circuit**: Stop on first deny
- **Override hooks**: Post-evaluation, can flip deny→allow with audit

## Runtime Assumptions
- Context propagation: No specific keys required
- Audit: AuditFields() returns status, reasons, severity counts, thresholds
- Errors: Return nil Decision + error; verification/threshold violations deny with message

## Versioning
- v1.0.0 stable for SDP runtime integration
- Evidence changes require policy contract review
