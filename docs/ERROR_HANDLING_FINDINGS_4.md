# Error Handling Code Review Findings (Fourth Pass)

**Scope:** `internal/` directory  
**Date:** 2026-03-02  
**Purpose:** Map findings to Beads issues for tracking and resolution.

---

## CRITICAL

| File:Line | Issue | Description |
|-----------|-------|-------------|
| `internal/bridge/beads_sink.go:59-60` | Ignored json.Unmarshal error | `return nil // Ignore parse errors` — `bd list` JSON parse failures are silently ignored. Findings can be wrong or duplicated; caller cannot distinguish parse failure from empty list. |
| `internal/session/paths.go:96` | Panic in production | `panic("session: InitPaths not called")` in `MustGetPaths()` — uninitialized use can crash the process. |
| `internal/evidence/auto_attest.go:182-188` | Use of output when err != nil | `collectTestResults` uses `string(out)` in the loop even when `cmd.Output()` returns an error. `out` can be nil or partial; parsing can mislead or panic. |
| `internal/evidence/discrepancy.go:148-151` | Misplaced comment | Comment block appears inside `if prefix != ""` but describes the wrong block; `matches, _ = filepath.Glob(pattern)` is outside the block. Logic is confusing. |

---

## HIGH

| File:Line | Issue | Description |
|-----------|-------|-------------|
| `internal/evidence/discrepancy.go:133,140,151` | Ignored filepath.Glob errors | `matches, _ := filepath.Glob(pattern)` — Glob can fail (permissions, I/O). Empty matches may mean "no files" or "error". |
| `internal/orchestrate/hydrate.go:53` | Ignored bdShow error | `out, _ := bdShow(projectRoot, beadsID)` — Dependency description may be wrong/empty on failure. |
| `internal/orchestrate/hydrate.go:60` | Ignored os.ReadFile error | `agentsContent, _ := os.ReadFile(agentsPath)` — Best-effort; AGENTS.md read failure yields empty quality gates. |
| `internal/orchestrate/hydrate.go:62` | Ignored gitStatusPorcelain error | `pkt.DriftStatus, _ = gitStatusPorcelain(projectRoot)` — Git status failure yields empty drift status. |
| `internal/evidence/auto_attest.go:157` | Ignored runGit error | `out, _ := runGit(repoRoot, "log", ...)` — Commit list may be wrong on failure. |
| `internal/evidence/auto_attest.go:197,210` | Type assertion ignores ok | `action, _ := evt["Action"].(string)` — Wrong type yields empty string; can misclassify events. |
| `internal/orchestrate/attest.go:259` | Ignored cmd.Output error | `out2, _ := cmd2.Output()` — Fallback `git diff HEAD` output ignored; attestation logic may be wrong. |
| `internal/orchestrate/state_machine.go:141` | Redundant Glob, ignored error | Second `filepath.Glob` ignores error; logic could be simplified. |
| `internal/orchestrate/invoke_opencode.go:134` | Ignored WritePromptProvenance error | `_ = WritePromptProvenance(...)` — Provenance write failure is silent; breaks audit trail. |
| `internal/orchestrate/loop.go:26` | Ignored SaveCheckpoint error | `_ = SaveCheckpoint(cpPath, cp)` — On shutdown, checkpoint may not persist; resume could re-run last phase. |
| `internal/orchestrate/migration_shim.go:321` | Ignored RecordEvent error | `_ = s.telemetry.RecordEvent(ctx, event)` — Telemetry loss; may hide dry-run issues. |
| `internal/orchestrate/fsm_v2.go:350` | Ignored EmitEventAsync error | `_ = f.eventProducer.EmitEventAsync(ctx, event)` — Event emission failure is silent. |
| `internal/modelgateway/credentials.go:323` | Ignored audit Log error | `_ = cm.audit.Log(ctx, entry)` — Audit log failure is silent (comment: best-effort). |
| `internal/authz/tenant_scope.go:224` | Ignored crossTenantLogger.Log error | `_ = a.crossTenantLogger.Log(entry)` — Cross-tenant audit loss. |
| `internal/verify/quorum.go:330` | ToJSON returns "{}" on marshal failure | Callers may treat `"{}"` as valid JSON; marshal failure is indistinguishable from empty object. |
| `internal/modelgateway/router.go:66` | ToJSON returns "{}" on marshal failure | Same pattern. |

---

## MEDIUM

| File:Line | Issue | Description |
|-----------|-------|-------------|
| `internal/evidence/trace_validator.go:121-122` | Returns nil on parse error | `LoadTraceEventsFromRunFile` returns nil on `json.Unmarshal` error; caller cannot distinguish parse failure from empty events. |
| `internal/orchestrate/policy.go:108-110,117-118` | Returns nil on cmd/parse errors | `queryOPAString` and `queryOPAStringSet` return "" or nil on error; caller loses context. |
| `internal/modelgateway/adapters/selfhosted.go:101` | defer resp.Body.Close() ignores error | HTTP body close can return errors. |
| `internal/evidence/sigstore_signer.go:345` | defer resp.Body.Close() ignores error | Same as above. |
| `internal/beads/sql_client.go:121,166,244,266` | defer rows.Close() ignores error | `sql.Rows.Close()` can return errors. |
| `internal/beads/dependency.go:50,79,123,153,183,248,278` | defer rows.Close() ignores error | Same pattern. |
| `internal/beads/client.go:124,165` | defer rows.Close() ignores error | Same pattern. |
| `internal/monitor/stuck_detector.go:191` | defer f.Close() ignores error | File close can fail. |
| `internal/profile/oss_combine.go:217-229` | Rollback prints errors, returns nil | `fmt.Printf` for remove errors; caller gets `nil` even when removal failed. |
| `internal/orchestrate/hydrate.go:74,86,90,136` | return err without wrapping | Errors returned without `%w`; caller loses context. |
| `internal/evidence/auto_attest.go:463` | return err without wrapping | Same. |
| `internal/planner/scheduler.go:157` | return err without wrapping | Same. |
| `internal/modelgateway/credentials.go:249,257` | return err without wrapping | Same. |
| `internal/orchestrate/migration_shim.go:190,250-259,357,402` | return err without wrapping | Multiple bare returns. |
| `internal/orchestrate/invoke_opencode.go:86,93,96,100` | return err without wrapping | MkdirAll, Marshal, WriteFile, Rename. |
| `internal/orchestrate/hooks.go:62,72` | return err without wrapping | LoadHookConfig and runHook. |
| `internal/bridge/beads_sink.go:134,176` | return err without wrapping | createBeadsIssue errors. |
| `internal/ciloop/runfile.go:51,58` | return err without wrapping | ValidateFeatureID, findRunFile. |
| `internal/ciloop/checkpoint.go:47` | return err without wrapping | ValidateFeatureID. |
| `internal/evidence/trace_validator.go:136,140,151` | return err without wrapping | ReadFile, Unmarshal, Marshal. |
| `internal/orchestrate/runfile.go:33` | return err without wrapping | ValidateFeatureID. |
| `internal/orchestrate/cli.go:104,108,123,132,138,144` | return err without wrapping | Multiple CLI error returns. |
| `internal/orchestrate/checkpoint.go:76` | return err without wrapping | ValidateFeatureID. |
| `internal/ciloop/cmdhelpers.go:60,92` | return err without wrapping | add.Run() errors. |
| `internal/ciloop/fixer.go:107,124,128` | return err without wrapping | MkdirAll, WriteFile, Rename. |
| `internal/ciloop/deterministic_fixer.go:37` | return err without wrapping | RunDeterministicFixers. |

---

## LOW

| File:Line | Issue | Description |
|-----------|-------|-------------|
| `internal/workstream/session_store.go:352-353` | h.Write ignores error | `hash.Hash.Write` never fails for this use; acceptable. |
| `internal/workstream/template.go:213-216` | h.Write ignores error | Same. |
| `internal/evidence/strict.go`, `inspect.go`, `operator_gate.go` | Type assertions with _ | Map access with `_, _ := m["key"].(type)` — Intentional optional field handling. |
| `internal/bridge/github_findings.go:221` | checkName, _ := source["check_name"] | Type assertion; empty string fallback for missing key is intentional. |
| `internal/evidence/discrepancy_test.go:16-17,48-49` | os.WriteFile error ignored | Test setup; can cause flaky tests. |
| `internal/workstream/session_store_test.go:158` | os.WriteFile error ignored | Test setup. |
| `internal/ciloop/autofixer.go:55-59` | ParseAutoFixersYAML error ignored | Silent parse failure; config load best-effort. |

---

## Summary by Severity

| Severity | Count |
|----------|-------|
| CRITICAL | 4 |
| HIGH | 16 |
| MEDIUM | 25+ |
| LOW | 7 |

---

## Previously Fixed (per ERROR_HANDLING_FINDINGS.md)

- `internal/orchestrate/merge_policy.go:181` — `json.Unmarshal` error is now checked (lines 181–182).

---

## Beads Mapping

Each finding above can be mapped to a Beads issue using:

```bash
bd create --title "Error handling: [brief description]" --body "[file:line and details]"
```

Suggested labels: `error-handling`, `code-quality`, `internal`.
