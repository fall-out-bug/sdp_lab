# Artifact Provenance Hash Chain Contract

This document defines the design contract for provenance hash chaining and append-only evidence storage on the artifact bus.

## Objectives

- Make every provenance entry tamper-evident through deterministic hashing.
- Enforce append-only semantics per issue stream.
- Keep validation simple enough to run in worker paths and tests.

## Record Schema (v1)

Each provenance entry must carry:

- `contract_version` (`artifact-provenance/v1`)
- `hash_algorithm` (`sha256`)
- `issue_id`
- `artifact_id`
- `artifact_class`
- `phase`
- `role`
- `captured_at` (RFC3339 UTC)
- `sequence` (monotonic integer per issue stream)
- `hash_prev` (empty for genesis, otherwise previous record hash)
- `payload_digest` (sha256 over canonical payload JSON)
- `hash` (sha256 over canonical schema fields excluding `hash`)

Deterministic field order for hashing:

1. `contract_version`
2. `hash_algorithm`
3. `issue_id`
4. `artifact_id`
5. `artifact_class`
6. `phase`
7. `role`
8. `captured_at`
9. `sequence`
10. `hash_prev`
11. `payload_digest`
12. `hash`

## Append-Only Store Contract

Logical keying:

- Partition key: `issue_id`
- Ordering key: `sequence`

Write constraints:

- `(issue_id, sequence)` is immutable once written.
- Genesis record must have `sequence=0` and empty `hash_prev`.
- Non-genesis record must have `hash_prev == previous.hash`.
- Sequence must increase by exactly one for each append.
- `hash`, `hash_prev`, and `payload_digest` are immutable evidence anchors.

## Baseline Implementation

- `internal/artifact/provenance_contract.go`
  - deterministic canonical JSON helper for map/slice payloads
  - digest helper (`DigestHex`)
  - record builder (`BuildProvenanceRecord`)
  - append validator (`ValidateAppend`)
- `internal/artifact/provenance_contract_test.go`
  - deterministic hashing regression tests
  - append-only monotonic sequence and hash-link tests

## Strict Evidence Alignment

`provenance` sections in strict evidence now include baseline contract fields:

- `contract_version`
- `hash_algorithm`
- `sequence`
- `payload_digest`

This keeps the strict evidence envelope aligned with append-only/hash-chain semantics while allowing incremental rollout of full hashing in worker runtime.
