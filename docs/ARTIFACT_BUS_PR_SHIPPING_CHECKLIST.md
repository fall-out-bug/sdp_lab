# Artifact Bus PR Shipping Checklist

This checklist documents the artifact bus implementation, migration path, and audit evidence retrieval patterns for `sdp_dev-2aq.16`.

## Architecture Snapshot

- `internal/artifact/bus_service.go`
  - append-only ingestion (`Ingest`) with deterministic provenance hash-chain generation
  - retrieval by issue/artifact ID, provenance hash, sequence, and latest record
  - provenance index projection and chain metadata (`genesis` + `head` anchors)
- `internal/artifact/bus_verify.go`
  - issue-level audit verification (`VerifyIssue`) for:
    - hash-chain append invariants
    - payload digest tamper checks
    - by-hash index consistency
    - provenance index consistency
    - retention window compliance by artifact class

## Migration Steps

1. Keep existing strict evidence generation in worker paths (`cmd/autonomy-worker`, `cmd/swarm-worker`).
2. Route each phase artifact into `BusService.Ingest` with class/phase/role metadata and evidence payload.
3. Persist bus snapshots in the chosen backend (in-memory now; durable backing can wrap the same API).
4. During verify/review gates, run `BusService.VerifyIssue(issueID, nowUTC)` and fail the gate on any finding.
5. Expose read-only evidence endpoints backed by bus retrieval helpers for audit and PR evidence linking.

## Evidence Access Examples

```go
meta, ok := bus.ChainMetadata(issueID)
if !ok {
	return errors.New("no artifact stream for issue")
}

latest, ok := bus.LatestByIssue(issueID)
if !ok {
	return errors.New("no latest artifact")
}

report := bus.VerifyIssue(issueID, time.Now().UTC())
if !report.OK() {
	return fmt.Errorf("artifact evidence verification failed: tamper=%v retention=%v", report.TamperFindings, report.RetentionFindings)
}
```

## PR Evidence Checklist

- [x] ingestion/retrieval lifecycle implemented and tested
- [x] provenance index includes deterministic hash-chain metadata
- [x] tamper + retention consistency verification implemented and tested
- [x] `gofmt` run on touched Go files
- [x] `go test ./...` passes
- [x] beads notes updated with implementation evidence

## Gate Transition Enforcement Rollout (sdp_dev-2aq.14)

### Transition Controller Scope

- `internal/artifact/transition_controller.go` enforces `transition-policy/v1` rules from `BuildTransitionPolicyContract()`.
- Transition adjudication requires all contract gate signals to pass, required artifact classes to exist, and required provenance keys to be present on required-class payloads.
- Denials emit deterministic reason codes in policy order with per-signal gate decision trace output for auditability.

### Migration Guardrails

1. Keep phase progression in monitor mode first: record `TransitionDecision` outputs before wiring hard-fail behavior in orchestrators.
2. Require parity checks between controller denials and existing gate outcomes for at least one full issue cycle per phase edge.
3. Fail closed only after parity is stable: block transition when `Allowed=false` and persist `ReasonCodes` plus `GateDecisions` in verification artifacts.
4. Treat unknown gate statuses as denied (`gate-not-passed`) and surface them as explicit operator action items.

### Rollback Strategy

1. Roll back to monitor mode by disabling hard-fail transition blocking while continuing to emit decision traces.
2. Keep policy contract and controller logic intact so evidence remains comparable before/after rollback.
3. Use deterministic reason codes to diff rollback-period denials against pre-rollback baseline and isolate noisy gates.
