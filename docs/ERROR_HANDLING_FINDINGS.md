# Error Handling Code Review Findings

**Scope:** `internal/` directory  
**Date:** 2026-03-02  
**Purpose:** Map findings to Beads issues for tracking and resolution.

---

## CRITICAL

| File:Line | Issue | Description |
|-----------|-------|-------------|
| `internal/orchestrate/merge_policy.go:181` | Ignored json.Unmarshal error | `_ = json.Unmarshal(bytes, &dataMap)` — If unmarshal fails, `dataMap` stays empty and merge proceeds with wrong/corrupt data. Could cause data loss or incorrect merge results. |
| `internal/modelgateway/router.go:66` | Ignored json.Marshal in production | `bytes, _ := json.Marshal(e)` in `ToJSON()` — Marshal failure returns empty string; callers may treat as valid JSON. Affects routing evidence serialization. |
| `internal/verify/quorum.go:330` | Ignored json.Marshal in production | `bytes, _ := json.Marshal(v)` in `VerifierResult.ToJSON()` — Same pattern; marshal failure silently produces empty string. |
| `internal/beads/sql_client.go:121` | defer rows.Close() ignores error | `defer rows.Close()` — sql.Rows.Close() returns error (e.g. cancelled query, connection issues). Error is discarded. Same at 166, 244, 266. |
| `internal/beads/dependency.go:50,79,123,153,183,248,278` | defer rows.Close() ignores error | Same as above; 7 occurrences across dependency queries. |
| `internal/beads/client.go:124,165` | defer rows.Close() ignores error | Same pattern in Client methods. |
| `internal/modelgateway/adapters/selfhosted.go:101` | defer resp.Body.Close() ignores error | `defer resp.Body.Close()` — HTTP response body close can return errors. Same in `internal/evidence/sigstore_signer.go:345`. |
| `internal/monitor/stuck_detector.go:191` | defer f.Close() ignores error | `defer f.Close()` — File close can fail (e.g. NFS, sync errors). No error handling. |

---

## HIGH

| File:Line | Issue | Description |
|-----------|-------|-------------|
| `internal/evidence/discrepancy.go:133,140,151` | Ignored filepath.Glob errors | `matches, _ := filepath.Glob(pattern)` — Glob can fail (permission, I/O). Empty matches may be misinterpreted as "no files" vs "error". Affects attestation discovery. |
| `internal/orchestrate/hydrate.go:53` | Ignored bdShow error | `out, _ := bdShow(projectRoot, beadsID)` — Dependency description may be wrong/empty on failure. |
| `internal/orchestrate/hydrate.go:60` | Ignored os.ReadFile error | `agentsContent, _ := os.ReadFile(agentsPath)` — Comment says "best-effort" but AGENTS.md read failure yields empty quality gates. |
| `internal/orchestrate/hydrate.go:62` | Ignored gitStatusPorcelain error | `pkt.DriftStatus, _ = gitStatusPorcelain(projectRoot)` — Git status failure yields empty drift status; may hide repo state. |
| `internal/evidence/auto_attest.go:157` | Ignored runGit error | `out, _ := runGit(repoRoot, "log", ...)` — Commit list may be wrong on failure. |
| `internal/evidence/auto_attest.go:197,210` | Type assertion ignores ok | `action, _ := evt["Action"].(string)` — Wrong type yields empty string; could misclassify events. |
| `internal/orchestrate/invoke_opencode.go:134` | Ignored WritePromptProvenance error | `_ = WritePromptProvenance(...)` — Provenance write failure is silent; breaks audit trail. |
| `internal/orchestrate/loop.go:26` | Ignored SaveCheckpoint error | `_ = SaveCheckpoint(cpPath, cp)` — On shutdown, checkpoint may not persist; resume could re-run last phase. |
| `internal/orchestrate/migration_shim.go:321` | Ignored RecordEvent error | `_ = s.telemetry.RecordEvent(ctx, event)` — Telemetry loss; may hide dry-run issues. |
| `internal/orchestrate/fsm_v2.go:350` | Ignored EmitEventAsync error | `_ = f.eventProducer.EmitEventAsync(ctx, event)` — Event emission failure is silent. |
| `internal/modelgateway/credentials.go:323` | Ignored audit Log error | `_ = cm.audit.Log(ctx, entry)` — Audit log failure is silent (comment: "best-effort"). |
| `internal/authz/tenant_scope.go:224` | Ignored crossTenantLogger.Log error | `_ = a.crossTenantLogger.Log(entry)` — Cross-tenant audit loss. |
| `internal/orchestrate/attest.go:259` | Ignored cmd.Output error | `out2, _ := cmd2.Output()` — Second command output ignored; could affect attestation logic. |
| `internal/orchestrate/state_machine.go:141` | Redundant Glob, ignored error | `if ents, _ := filepath.Glob(...)` — Second Glob call ignores error; logic could be simplified. |

---

## MEDIUM

| File:Line | Issue | Description |
|-----------|-------|-------------|
| `internal/orchestrate/hydrate.go:74` | return err without wrapping | `return nil, err` — Loses context (WriteContextPacket failed). Caller cannot distinguish from other errors. |
| `internal/orchestrate/hydrate.go:86,90` | return err without wrapping | Same; multiple `return nil, err` / `return err` without `%w`. |
| `internal/orchestrate/hydrate.go:136` | return err without wrapping | LoadContextPacket error returned bare. |
| `internal/evidence/auto_attest.go:119,146,463` | return err without wrapping | Several bare returns. |
| `internal/planner/scheduler.go:157` | return err without wrapping | GetPlan error returned without context. |
| `internal/modelgateway/credentials.go:249,257` | return err without wrapping | Store Get/Set errors returned bare. |
| `internal/orchestrate/migration_shim.go:190,250-259,357,402` | return err without wrapping | Multiple bare returns in migration flow. |
| `internal/orchestrate/invoke_opencode.go:86,93,96,100` | return err without wrapping | MkdirAll, Marshal, WriteFile, Rename errors returned without context. |
| `internal/orchestrate/hooks.go:62,72` | return err without wrapping | LoadHookConfig and runHook errors. |
| `internal/bridge/beads_sink.go:134,176` | return err without wrapping | createBeadsIssue errors. |
| `internal/ciloop/runfile.go:51,58` | return err without wrapping | ValidateFeatureID, findRunFile. |
| `internal/ciloop/checkpoint.go:47` | return err without wrapping | ValidateFeatureID. |
| `internal/evidence/trace_validator.go:136,140,151` | return err without wrapping | ReadFile, Unmarshal, Marshal errors. |
| `internal/orchestrate/runfile.go:33` | return err without wrapping | ValidateFeatureID. |
| `internal/orchestrate/cli.go:104,108,123,132,138,144` | return err without wrapping | Multiple CLI error returns. |
| `internal/orchestrate/checkpoint.go:76` | return err without wrapping | ValidateFeatureID. |
| `internal/ciloop/cmdhelpers.go:60,92` | return err without wrapping | add.Run() errors. |
| `internal/ciloop/fixer.go:107,124,128` | return err without wrapping | MkdirAll, WriteFile, Rename. |
| `internal/ciloop/deterministic_fixer.go:37` | return err without wrapping | RunDeterministicFixers. |
| `internal/profile/oss_combine.go:218,229` | fmt.Printf for errors | `fmt.Printf("  ✗ Failed to remove...")` — Errors printed but not returned; caller cannot react. |
| `internal/evidence/discrepancy.go:148-150` | Misplaced comment | Comment block appears inside `if prefix != ""` block; may confuse readers. |

---

## LOW

| File:Line | Issue | Description |
|-----------|-------|-------------|
| `internal/verify/quorum_test.go:216` | _ = RegisterVerifier | Test code; acceptable. |
| `internal/planner/scheduler_test.go` | Multiple _, _ = | Test setup; acceptable. |
| `internal/modelgateway/adapters/adapters_test.go` | p, _ := New*Provider | Test code; acceptable. |
| `internal/authz/tenant_scope_test.go:223,224,340` | _ = RegisterTenant, got, _ | Test code; acceptable. |
| `internal/modelgateway/credentials_test.go` | Various _, _ = | Test code; acceptable. |
| `internal/beads/client.go:263` | fmt.Fprintf warning | `fmt.Fprintf(os.Stderr, "warning: get blockers...")` — Logs but continues; documented behavior. |
| `internal/adapters/sdk/examples/main.go:33,38,58` | fmt.Printf for errors | Example code; prints errors to user. |
| `internal/bridge/github_findings.go:221` | checkName, _ := source["check_name"] | Type assertion; empty string fallback for missing key is intentional. |
| `internal/evidence/strict.go`, `inspect.go`, `operator_gate.go` | Type assertions with _ | Map access with `_, _ := m["key"].(type)` — Intentional optional field handling. |
| `internal/orchestrate/hydrate.go:122` | _ = os.Remove(tmpPath) | Cleanup on Rename failure; best-effort. |
| `internal/orchestrate/invoke_opencode.go:99` | _ = os.Remove(tmpPath) | Same. |
| `internal/ciloop/runfile.go:85` | _ = os.Remove(tmpPath) | Same. |
| `internal/orchestrate/runfile.go:60` | _ = os.Remove(tmpPath) | Same. |
| `internal/orchestrate/checkpoint.go:89` | _ = os.Remove(tmpPath) | Same. |
| `internal/ciloop/checkpoint.go:60` | _ = os.Remove(tmpPath) | Same. |
| `internal/ciloop/fixer.go:127` | _ = os.Remove(tmpPath) | Same. |
| `internal/ciloop/cleanup.go:21` | _ = os.Remove | Cleanup loop; best-effort. |
| `internal/orchestrate/policy.go:60,62,65` | defer Close/Remove | Temp file cleanup; acceptable. |
| `internal/evidence/auto_attest.go:345` | defer f.Close() | Explicit `_ = f.Close()` in defer; documented. |
| `internal/orchestrate/attest.go:146` | defer f.Close() | Same. |

---

## Summary by Severity

| Severity | Count |
|----------|-------|
| CRITICAL | 9 |
| HIGH | 14 |
| MEDIUM | 25+ |
| LOW | 20+ (many acceptable) |

---

## Recommended Actions

1. **CRITICAL:** Fix `merge_policy.go:181` — check `json.Unmarshal` error; abort merge on failure.
2. **CRITICAL:** Add error handling for `json.Marshal` in `ToJSON()` methods (router.go, quorum.go).
3. **CRITICAL:** Consider checking `rows.Close()` and `resp.Body.Close()` errors (or document why ignored).
4. **HIGH:** Add context to best-effort operations (e.g. log when hydrate/attest fallbacks occur).
5. **HIGH:** Handle `WritePromptProvenance` and `SaveCheckpoint` failures explicitly (log + optional retry).
6. **MEDIUM:** Wrap propagated errors with `fmt.Errorf("...: %w", err)` for traceability.
7. **MEDIUM:** Fix `oss_combine.go` to return or propagate remove errors where appropriate.

---

## Beads Mapping

Each finding above can be mapped to a Beads issue using:

```bash
bd create --title "Error handling: [brief description]" --body "[file:line and details]"
```

Suggested labels: `error-handling`, `code-quality`, `internal`.
