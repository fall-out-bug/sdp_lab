# Code Quality Findings — Fifth Review (internal/)

Fifth comprehensive code review. Focus: code smells, anti-patterns, bugs, maintainability. Each finding mapped for Beads issue tracking.

---

## 1. Bugs / Logic Errors

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/modelgateway/router.go:377` | **MEDIUM** | **Bug:** `hashInput` references `input.SessionID`, `input.Prompt`, `input.Constraints` — none exist on `RoutingInput`. Struct has `TaskClass`, `Sensitivity`, `TenantID`, `ModelHint`, etc. Either compile error or wrong fields; hash is incorrect. Use `input.TaskClass`, `input.TenantID`, `input.ModelHint`, etc. |
| `internal/orchestrate/merge_policy.go:332-335` | **MEDIUM** | **Nil pointer risk:** When `p.inner.Merge` returns `(nil, err)`, `mergeResult` is nil. Line 334 accesses `mergeResult.Status` without nil check — panic. Add `if mergeResult == nil { return nil, err }` before block. |
| `internal/evidence/discrepancy.go:146-152` | **MEDIUM** | **Bug:** `findAttestation` third branch (prefix glob) updates `matches` but never returns when matches exist. Misplaced comment; logic falls through to `return ""`. Add `if len(matches) > 0 { return matches[0] }` before final return. |
| `internal/evidence/auto_attest.go:107` | **LOW** | `boundaryOK` assigned but never used (`_ = boundaryOK`). Remove or use. |

---

## 2. Functions Exceeding 50 Lines

| File:Line | Severity | Lines | Function |
|-----------|----------|-------|----------|
| `internal/orchestrate/fsm_v2.go` | MEDIUM | ~149 | `(f *FSMV2) Transition` |
| `internal/evidence/discrepancy.go:51-127` | LOW | ~77 | `CompareAttestations` |
| `internal/beads/sql_client.go:69-136` | LOW | ~68 | `QueryIssues` |
| `internal/bridge/beads_sink.go:101-181` | LOW | ~81 | `syncProtocolFinding` + `syncDocsFinding` (combined flow) |
| `internal/orchestrate/merge_policy.go:91-150` | LOW | ~60 | `(p *DefaultMergePolicy) Merge` |
| `internal/evidence/auto_attest.go:179-233` | LOW | ~55 | `collectTestResults` |
| `internal/monitor/stuck_detector.go:178-249` | LOW | ~72 | `getLastEventTime` |
| `internal/verify/quorum.go:162-226` | LOW | ~65 | `(q *Quorum) Execute` |

---

## 3. Deeply Nested Code (>3 Levels)

| File:Line | Severity | Context |
|-----------|----------|---------|
| `internal/orchestrate/merge_policy.go:269-278` | MEDIUM | `CheckConsistency`: for → if r.Result → if dataMap → if policy → if policyStr (5 levels) |
| `internal/modelgateway/router.go:191-203` | LOW | `defaultRoute`: for id → if caps → for m → if string(m)==hint (4 levels) |
| `internal/evidence/discrepancy.go:242-252` | LOW | `compareTestResults`: for t → if fail → for at → if match (4 levels) |
| `internal/modelgateway/router.go:212-221` | LOW | `applyTenantConfig`: if len>0 → for → if → if !allowed (4 levels) |
| `internal/planner/scheduler.go:95-111` | LOW | `Schedule`: for → if err → if status → if canStart → if startTask (5 levels) |

---

## 4. Duplicate Code Blocks

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/session/paths.go:20-34`, `internal/session/writer.go:22-36`, `internal/workstream/session_store.go:16-29`, `internal/beads/formula_parser.go:16-27` | MEDIUM | Four nearly identical ID/name validation functions. Same checks: empty, `..`, path separators, regex. Extract to `sdputil.ValidateSafeID(id, label string) error`. |
| `internal/bridge/beads_sink.go:185-225`, `227-261` | MEDIUM | `buildProtocolDescription` and `buildDocsDescription` share ~70% structure (Category, Severity, File, Message, Remediation). Extract shared builder. |
| `internal/bridge/beads_sink.go:101-139`, `143-181` | LOW | `syncProtocolFinding` and `syncDocsFinding` share flow: severity check, key dedup, build desc, labels, create. Extract `syncFinding` helper. |

---

## 5. Magic Numbers Without Constants

| File:Line | Severity | Value | Suggested Constant |
|-----------|----------|-------|--------------------|
| `internal/evidence/discrepancy.go:282,285` | MEDIUM | `10`, `20` | `CoverageDiffMediumThreshold`, `CoverageDiffHighThreshold` |
| `internal/evidence/auto_attest.go:89` | MEDIUM | `80` | `DefaultCoverageThreshold` |
| `internal/monitor/stuck_detector.go:199` | LOW | `4096` | `ReadBufferSize` |
| `internal/bridge/beads_sink.go:116,158` | LOW | `60` | `MaxTitleLength` |
| `internal/beads/sql_client.go:72` | LOW | `50` | `DefaultQueryLimit` |
| `internal/beads/dependency.go:237` | LOW | `10` | `MaxTransitiveDepth` |
| `internal/orchestrate/parallel_executor.go:78-79` | LOW | `10`, `30*time.Minute` | `DefaultMaxBranches`, `DefaultTimeout` |
| `internal/workstream/protocol_validate.go:326,337` | LOW | `60` | `LegacyFeatureIDThreshold` |
| `internal/workstream/session_store.go:355` | LOW | `8` | `WispIDHashLen` |

---

## 6. Unused Variables / Imports

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/evidence/auto_attest.go:107` | MEDIUM | `boundaryOK` assigned but never used. |
| `internal/workstream/protocol_validate.go:334-335` | LOW | `prefix`, `seq` assigned in Sscanf but discarded with `_`. Intentional; consider comment. |

---

## 7. Inconsistent Naming Conventions

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/session/paths.go` vs `internal/session/writer.go` | LOW | `ValidateSessionID` (exported) vs `validateSessionID` (unexported) — same logic. |
| `internal/verify/quorum.go` | LOW | Exported types `VerifierID`, `Verdict`, `Quorum` lack doc comments; `dissentingToEvidence` is unexported. |

---

## 8. Missing Documentation for Exported Items

| File:Line | Severity | Item |
|-----------|----------|------|
| `internal/verify/quorum.go` | LOW | `VerifierID`, `Verdict`, `VerifierRole`, `VerifierResult`, `QuorumPolicy`, `QuorumVerdict`, `Verifier`, `Quorum`, `PromotionGate`, `WithPolicy`, `WithPromoter`, `NewQuorum`, `DefaultQAPolicy`, `DefaultSecurityPolicy`, `DefaultReleasePolicy` — no doc comments. |
| `internal/modelgateway/router.go` | LOW | `TaskClass`, `SensitivityLevel`, `RoutingInput`, `PolicyRouter`, `PolicyEvaluator`, `TenantConfigStore` — no doc comments. |
| `internal/orchestrate/merge_policy.go` | LOW | `ConflictResolution`, `MergeResult`, `MergeStatus`, `MergePolicy`, `DefaultMergePolicy`, `ScopeChecker` — no doc comments. |

---

## 9. Hardcoded Values That Should Be Configurable

| File:Line | Severity | Value | Suggestion |
|-----------|----------|-------|------------|
| `internal/evidence/discrepancy.go:53` | LOW | `".sdp/evidence"` | Default in `CompareOptions`; allow override. |
| `internal/evidence/auto_attest.go:114,141` | LOW | `"master"` | Base branch default; allow config. |
| `internal/modelgateway/router.go:184,206,331` | LOW | `"selfhosted"`, `"openai"` | Provider IDs hardcoded. |
| `internal/orchestrate/parallel_executor.go:78-79` | LOW | `maxBranches: 10`, `timeout: 30*time.Minute` | Make configurable via env or config. |

---

## 10. Nil Pointer Dereference Risks

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/orchestrate/merge_policy.go:332-335` | MEDIUM | `mergeResult` can be nil when `p.inner.Merge` returns error; `mergeResult.Status` accessed without check. |
| `internal/modelgateway/router.go:242` | LOW | `provider` from `r.registry.Get` — ok check guards use; if registry returns (nil, true), `provider.Chat` would panic. Unlikely but worth defensive check. |
| `internal/modelgateway/credentials.go:368` | LOW | `InMemoryTenantStore.Set(config)` — no nil check on `config`; `config.TenantID` would panic if nil. |

---

## 11. Off-by-One / Boundary Errors

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/monitor/stuck_detector.go:181` | LOW | `sessionID = sessionID[len("session-"):]` — assumes filename starts with "session-"; if `filepath.Base(file)` is shorter, could panic. Add length check. |
| `internal/evidence/auto_attest.go:96` | LOW | `headSHA[:minLen(len(headSHA), 8)]` — safe; `minLen` guards. No issue. |

---

## Summary by Severity

| Severity | Count |
|----------|-------|
| MEDIUM   | 10    |
| LOW      | 32    |

---

## Recommended Next Steps

1. **Create Beads issues** for each MEDIUM finding.
2. **Fix bugs first:** hashInput fields, IsolatedMergePolicy nil check, findAttestation return.
3. **Extract shared validation** to `sdputil.ValidateSafeID`.
4. **Introduce constants** for magic numbers (discrepancy.go, auto_attest.go, stuck_detector.go).
5. **Refactor long functions** (FSMV2.Transition, CompareAttestations, getLastEventTime).
6. **Add nil checks** in merge_policy.go and credentials.go where pointers are used.
