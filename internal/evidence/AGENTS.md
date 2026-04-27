# SDP Runtime Assumptions for sdp-evidence-core

## Context Objects
**CodingWorkflowStatement**: Primary context for all evidence operations
Wraps in-toto StatementHeader with CodingWorkflowPredicate:
- Intent (issue_id, trigger, risk_class)
- Plan (workstreams), Execution (branch, changed_files)
- Verification (tests, lint, coverage)
- Boundary (declared vs observed paths)
- Provenance (run_id, orchestrator, runtime)
- Trace (commits, PR URL)

All operations (attestation, validation, signing, verification) consume/produce CodingWorkflowStatement.

## Environment Variables
**Signing** (optional):
- SIGSTORE_ID_TOKEN, COSIGN_KEY
- GitHub Actions: GITHUB_ACTIONS, ACTIONS_ID_TOKEN_REQUEST_TOKEN, ACTIONS_ID_TOKEN_REQUEST_URL
- SDP_SIGSTORE_STRICT_VERIFY=1 enforces verification

**Validation**: No env vars required (pure Go logic on evidence files)

## Configuration
- CoverageThreshold: compareOptions.CoverageThreshold (default 5.0%)
- FileScopeThreshold: fileScopeHighSeverityOver (default 3 files)
- BoundaryValidation: Per-workstream in docs/workstreams/backlog/{workstream}.md

## Dependencies
**Signing** (optional): in-toto-golang, sigstore-go, cosign CLI
**Validation**: Stdlib encoding/json, crypto/sha256, in-toto-golang (StatementHeader only)
**Trace**: internal/kernel.TraceEvent

## Coupling Risk
Imports internal/kernel.TraceEvent (deliberate: evidence validates same trace shape kernel emits)
No other internal dependencies; package is self-contained
