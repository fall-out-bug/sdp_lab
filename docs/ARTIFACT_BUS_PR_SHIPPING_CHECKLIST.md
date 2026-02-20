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
