# Decision Explainability (F070-02)

## Overview

The SDP policy system provides explainable decisions for all allow/ask/deny outcomes. This ensures transparency and auditability for policy enforcement.

## Decision Types

### Allow
The operation is permitted to proceed. Example reasons:
- All checks passed
- Boundary compliance verified
- Attestation valid and signed

### Ask
The operation requires human review. Example reasons:
- Test failures detected
- Discrepancies found between agent and CI evidence
- Boundary violations detected

### Deny
The operation is blocked. Example reasons:
- Critical discrepancies exceeded threshold
- Attestation missing or invalid
- Boundary violation outside declared scope

## Explanation Format

### Basic Mode
```
Decision: allow
Reason: all checks passed
```

### Detailed Mode
```
Decision: deny
Reason: critical or high discrepancies: Discrepancies found: 1 critical, 2 high
Rule: discrepancy-threshold

Evidence:
  1. boundary_violation: CI detected boundary violation not reported by agent (compliant vs [...])
  2. test_result_mismatch: CI reports test failure 'TestBranchProtection' not reported by agent (pass vs fail)
  3. commit_mismatch: Head commit mismatch between agent and CI attestations

Context:
  run_id: run-abc123
  agent_attestation: .sdp/evidence/run-run-abc123.json
  ci_attestation: .sdp/evidence/ci-auto-run-abc123.json
```

## API Usage

### Explaining a Decision

```go
import "sdp_dev/internal/policy"

// Get explanation from evidence gate result
result := evaluateEvidenceGate(config, attestation, report, verify)
explanation := policy.ExplainDecision(result, policy.ExplainModeDetailed)

// Format for display
fmt.Println(policy.FormatExplanation(explanation))

// Or output as JSON
json, err := policy.ExplainToJSON(explanation)
```

### Explaining an Attestation

```go
import "sdp_dev/internal/policy"
import "sdp_dev/internal/evidence"

stmt, err := evidence.ReadAttestation(path)
if err != nil {
    return err
}

explanation := policy.ExplainAttestation(stmt, policy.ExplainModeDetailed)
fmt.Println(policy.FormatExplanation(explanation))
```

### Explaining a Discrepancy Report

```go
import "sdp_dev/internal/policy"
import "sdp_dev/internal/evidence"

report, err := evidence.ReadDiscrepancyReport(path)
if err != nil {
    return err
}

explanation := policy.ExplainDiscrepancy(report, policy.ExplainModeDetailed)
fmt.Println(policy.FormatExplanation(explanation))
```

## CLI Usage

### Using sdp-watch with Explanation
```bash
# Stream events with explanations
sdp-watch

# Filter by run ID
sdp-watch -run-id run-abc123

# Show only errors
sdp-watch -severity error
```

## Evidence References

Each explanation includes references to supporting evidence:

- **Attestation**: Full attestation file path
- **Discrepancy Report**: Report file path and summary
- **Boundary Violations**: Specific files outside declared scope
- **Test Failures**: Specific failing test names and outcomes

## Sanitization

By default, user-facing explanations exclude internal implementation details:
- Stack traces are excluded
- Internal debug information is hidden
- Only policy-relevant information is shown

To include internal details, use `ExplainModeInternal`.

## Testing

### Regression Tests
```bash
# Run explainability tests
go test ./internal/policy/... -run TestExplain
```

### Deterministic Output
Explanation text is deterministic for the same input:
- Same decision always produces the same explanation
- Evidence references are in a stable order
- Timestamps are formatted consistently

## Integration with CI Gates

The explanation system integrates with CI gates:

1. **Evidence Gate**: Explains why evidence passed/failed
2. **Policy Gate**: Uses OPA rules with explainable results
3. **Scope Gate**: Explains boundary violations

## Future Enhancements

- [ ] Multi-language explanation support
- [ ] Customizable explanation templates
- [ ] Integration with documentation links
- [ ] Explanation caching for performance

## References

- Workstream: [00-070-02](../workstreams/backlog/00-070-02.md)
- Policy Package: `internal/policy/explain.go`
- Evidence Package: `internal/evidence/`
