# Feature: Quality Pipeline Extraction (FR-007)

Priority: P1
Effort: 1 day
No dependencies

## Problem

Quality pipeline (tests, evidence collection, provenance signing, PR gate, FSM transitions) is embedded in `internal/pipeline/executor.go` as part of Path A. Adapter's EvidenceProjector duplicates part of this logic with different field names. A shared package is needed.

## Scope

1. Create `internal/quality/` package
2. Extract from executor.go:
   - `RunTests(workDir) → bool`
   - `CollectEvidence(issueID, branch, model, role, changed) → Evidence`
   - `SignProvenance(runID, issueID, evidence) → SignedEnvelope`
   - `RunPRGate(issueID, workDir) → error`
   - `TransitionFSM(issueID, targetState) → error`
   - `CommitAndPublish(workDir, issueID, title, branch) → prURL`
3. adapter post-reconcile hook invokes quality pipeline
4. EvidenceProjector uses shared schema (no duplication)

## Acceptance Criteria

- `internal/quality/` package compiles
- adapter-controller uses quality pipeline on Succeeded reconcile
- No duplication of evidence schema between adapter and quality
