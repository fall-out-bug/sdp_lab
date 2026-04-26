# Evidence Export — SIEM & Compliance Bundle

## Overview

SDP exports normalized audit and evidence records to SIEM systems and produces
compliance-ready evidence bundles with integrity verification.

## Sink Adapters

### Syslog (Structured JSON)

Writes one JSON line per record to any `io.Writer` (stdout, file, syslog daemon).

```go
sink := export.NewSyslogSink(os.Stdout)
sink.Write(export.SIEMRecord{
    EventType: "evidence.gate.evaluated",
    Source:    "sdp-lab",
    Severity:  "info",
    Actor:     "operator-1",
    Action:    "gate.evaluate",
    Outcome:   "allowed",
})
```

### HTTP (REST/SIEM)

POSTs JSON records to a configurable endpoint with authorization and retry.

```go
sink := export.NewHTTPSink("https://siem.example.com/api/v1/ingest",
    "Bearer <token>",
    export.WithMaxRetries(3),
    export.WithRetryDelay(200 * time.Millisecond),
)
sink.Write(record)
```

## Backfill Mode

For historical run exports, `BatchSink` wraps any sink and sends records in
configurable batch sizes.

```go
batchSink := export.NewBatchSink(httpSink, 50)
batchSink.WriteBatch(historicalRecords)
```

## Integrity Verification

Each record carries a SHA-256 checksum over its canonical JSON representation
(excluding the checksum and signature fields themselves).

```go
record.Checksum = record.ComputeChecksum()
record.VerifyChecksum() // true
```

## Compliance Evidence Bundle

`ExportBundle` aggregates records with a bundle-level checksum for chain-of-custody.

```go
bundle := export.NewExportBundle("tenant-1", "F074", records)
bundle.Verify() // checks all record checksums + bundle checksum
```

## Compliance Control Mapping

| Control | Implementation |
|---------|---------------|
| AU-6 (Audit Review) | Syslog/HTTP sinks provide real-time audit stream |
| AU-10 (Non-repudiation) | SHA-256 checksums per record, bundle-level integrity |
| SC-8 (Transmission Integrity) | HTTPS enforced, checksum verified post-delivery |
| SI-4 (System Monitoring) | Structured JSON events with severity, actor, outcome |
| EU AI Act Art. 14 (Human Oversight) | Evidence gate outcomes exported with actor attribution |
| EU AI Act Art. 12 (Record-keeping) | ExportBundle provides tamper-evident evidence archives |
| Colorado AI Act (Risk Mgmt) | Tenant-scoped exports, discrepancy thresholds logged |
