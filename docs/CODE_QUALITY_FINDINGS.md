# Code Quality Findings — internal/ Directory

Fourth comprehensive code review. Findings mapped for Beads issue tracking.

---

## 1. Bugs / Logic Errors

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/evidence/discrepancy.go:146-155` | **MEDIUM** | **Bug:** `findAttestation` third branch (backwards-compat prefix glob) updates `matches` but never returns when matches exist. Function always falls through to `return ""`. Add `if len(matches) > 0 { sort.Sort(...); return matches[0] }` before final `return ""`. |
| `internal/modelgateway/router.go:372-374` | **MEDIUM** | **Bug:** `hashInput(input RoutingInput)` ignores `input` and returns `fmt.Sprintf("%x", time.Now().UnixNano())`. Hash is non-deterministic and does not reflect input; breaks caching/deduplication. |
| `internal/evidence/auto_attest.go:345` | **MEDIUM** | **Bug:** `defer f.Close()` inside loop in `collectDeclaredScopePrefixes`. All file handles deferred until function return; many files opened without timely close. Resource leak. |
| `internal/evidence/discrepancy.go:281-296` | **LOW** | **Indentation/scope:** `compareCoverage` severity block misaligned; `if diff > threshold` body and inner `if diff >= 10`/`if diff >= 20` are confusing. Fix indentation and structure. |

---

## 2. Functions Exceeding 50 Lines

| File:Line | Severity | Lines | Function |
|-----------|----------|-------|----------|
| `internal/orchestrate/fsm_v2.go:148-296` | MEDIUM | ~149 | `(f *FSMV2) Transition` |
| `internal/evidence/discrepancy.go:51-127` | LOW | ~77 | `CompareAttestations` |
| `internal/orchestrate/merge_policy.go:91-150` | LOW | ~60 | `(p *DefaultMergePolicy) Merge` |
| `internal/evidence/auto_attest.go:179-233` | LOW | ~55 | `collectTestResults` |
| `internal/workstream/session_store.go:231-268` | LOW | ~38 | `ListWisps` (nested logic) |
| `internal/workstream/session_store.go:269-312` | LOW | ~44 | `ExpireWisps` |
| `internal/beads/sql_client.go:69-138` | LOW | ~70 | `QueryIssues` |
| `internal/bridge/beads_sink.go:100-181` | LOW | ~82 | `syncProtocolFinding` + `syncDocsFinding` (duplicated structure) |

---

## 3. Deeply Nested Code (>3 Levels)

| File:Line | Severity | Context |
|-----------|----------|---------|
| `internal/orchestrate/merge_policy.go:263-292` | MEDIUM | `CheckConsistency`: for → if r.Result → if dataMap → if policy → if policyStr (5 levels) |
| `internal/modelgateway/router.go:189-199` | LOW | `defaultRoute`: for → if caps → for m → if string(m)==hint (4 levels) |
| `internal/evidence/discrepancy.go:242-261` | LOW | `compareTestResults`: for t → if fail → for at → if match (4 levels) |
| `internal/evidence/auto_attest.go:337-364` | LOW | `collectDeclaredScopePrefixes`: for e → for scanner.Scan → if inScopeSection → if prefix (4 levels) |
| `internal/guard/permission_bridge.go:252-256` | LOW | `matchPattern`: for rule → if rule.Pattern==pattern (searches all rules for compiled regex) |

---

## 4. Duplicate Code Blocks

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/session/paths.go:20-34`, `internal/session/writer.go:22-36`, `internal/workstream/session_store.go:16-29` | MEDIUM | Three nearly identical ID validation functions: `ValidateSessionID`, `validateSessionID`, `validateID`. Same regex `^[a-zA-Z0-9_-]+$`, same checks (empty, `..`, path separators). Extract to shared `sdputil.ValidateID(id string) error`. |
| `internal/bridge/beads_sink.go:100-139`, `140-181` | MEDIUM | `syncProtocolFinding` and `syncDocsFinding` share ~80% structure: severity check, key dedup, title, description, labels, priority, dry-run, create. Extract common `syncFinding` or template. |
| `internal/bridge/beads_sink.go:185-222`, `226-258` | LOW | `buildProtocolDescription` and `buildDocsDescription` share header format (Category, Severity, File, Message). Consider shared builder. |

---

## 5. Magic Numbers Without Constants

| File:Line | Severity | Value | Suggested Constant |
|-----------|----------|-------|--------------------|
| `internal/evidence/discrepancy.go:282,285` | MEDIUM | `10`, `20` | `CoverageDiffMediumThreshold`, `CoverageDiffHighThreshold` |
| `internal/evidence/auto_attest.go:89` | MEDIUM | `80` | `DefaultCoverageThreshold` |
| `internal/workstream/session_store.go:104,148,157,227` | LOW | `0755`, `0644` | `FileModeDir`, `FileModeFile` (or use `os.ModePerm` / `0o644`) |
| `internal/bridge/beads_sink.go:115,157` | LOW | `60` | `MaxTitleLength` or `TruncateTitleLen` |
| `internal/beads/sql_client.go:72` | LOW | `50` | `DefaultQueryLimit` |
| `internal/beads/dependency.go:237` | LOW | `10` | `MaxTransitiveDepth` |
| `internal/orchestrate/parallel_executor.go:78-79` | LOW | `10`, `30*time.Minute` | `DefaultMaxBranches`, `DefaultTimeout` |
| `internal/workstream/protocol_validate.go:326,337` | LOW | `60` | `LegacyFeatureIDThreshold` |

---

## 6. Unused Variables / Imports

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/evidence/auto_attest.go:107` | MEDIUM | `boundaryOK` assigned but never used: `_ = boundaryOK`. Either use or remove. |
| `internal/guard/permission_bridge.go:252` | LOW | `matchPattern` iterates over `pb.config.Rules` to find compiled regex by pattern; `pattern` param could match multiple rules. Logic is O(n) per call; consider caching pattern→regex map. |

---

## 7. Inconsistent Naming Conventions

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/session/paths.go` vs `internal/session/writer.go` | LOW | `ValidateSessionID` (exported) vs `validateSessionID` (unexported) — same logic, different visibility. |
| `internal/verify/quorum.go` | LOW | `dissentingToEvidence` (unexported) vs `ToEvidence` (exported). Consistent. |
| `internal/evidence/` | LOW | `compareFileScope`, `compareTestResults`, etc. are unexported; `CompareAttestations` exported. Fine. |

---

## 8. Missing Documentation for Exported Items

| File:Line | Severity | Item |
|-----------|----------|------|
| `internal/modelgateway/router.go` | LOW | `TaskClass`, `SensitivityLevel`, `RoutingInput`, `RoutingDecision`, `PolicyRouter`, `PolicyEvaluator`, `TenantConfigStore` — no `//` doc comments. |
| `internal/verify/quorum.go` | LOW | `VerifierID`, `Verdict`, `VerifierRole`, `VerifierResult`, `QuorumPolicy`, `QuorumVerdict`, `Verifier`, `Quorum`, `PromotionGate` — types lack doc comments. |
| `internal/orchestrate/merge_policy.go` | LOW | `ConflictResolution`, `MergeResult`, `MergeStatus`, `Conflict`, `MergePolicy`, `DefaultMergePolicy`, `ScopeChecker` — no doc comments. |

---

## 9. Hardcoded Values That Should Be Configurable

| File:Line | Severity | Value | Suggestion |
|-----------|----------|-------|------------|
| `internal/evidence/discrepancy.go:53` | LOW | `".sdp/evidence"` | Default in `CompareOptions`; allow override. |
| `internal/evidence/auto_attest.go:114,141` | LOW | `"master"` | Base branch default; allow config (e.g. `main`). |
| `internal/modelgateway/router.go:184,206,331` | LOW | `"selfhosted"`, `"openai"` | Provider IDs hardcoded; use registry/config. |
| `internal/evidence/auto_attest.go:328` | LOW | `"docs/workstreams/backlog"` | Backlog path; allow project layout override. |
| `internal/evidence/auto_attest.go:267` | LOW | `"--timeout=120s"` | golangci-lint timeout; make configurable. |

---

## Summary by Severity

| Severity | Count |
|----------|-------|
| MEDIUM   | 12    |
| LOW      | 28    |

---

## Recommended Next Steps

1. **Create Beads issues** for each MEDIUM finding; batch LOW findings by category.
2. **Fix bugs first** (findAttestation, hashInput, defer-in-loop).
3. **Extract shared validation** (`sdputil.ValidateID`) to reduce duplication.
4. **Introduce constants** for magic numbers in discrepancy.go and auto_attest.go.
5. **Refactor long functions** (FSMV2.Transition, CompareAttestations) by extracting helpers.
6. **Add doc comments** for exported types in modelgateway, verify, orchestrate.
