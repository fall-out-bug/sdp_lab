# Error Handling Code Review Findings (Fifth Pass)

**Scope:** `internal/` directory  
**Date:** 2026-03-03  
**Purpose:** Identify ALL remaining error handling issues after previous fixes. Map to Beads for tracking.

---

## CRITICAL

| File:Line | Issue | Description |
|-----------|-------|-------------|
| `internal/modelgateway/router.go:69` | Error masking in ToJSON | `if err != nil { return "{}" }` — json.Marshal failure returns valid-looking JSON `"{}"`. Callers cannot distinguish success from failure; routing evidence may be silently corrupt. |
| `internal/verify/quorum.go:331` | Error masking in ToJSON | Same pattern in `VerifierResult.ToJSON()` — marshal failure yields `"{}"`; verification evidence can be lost. |
| `internal/bridge/beads_sink.go:52-61` | Silent error masking | `LoadExistingFindings`: (1) `cmd.Output()` error → `return nil` (no findings); (2) `json.Unmarshal` error → `return nil`. Comment says "Log parse error" but no logging. Corrupt Beads output or bd list failure is invisible; deduplication may fail. |
| `internal/orchestrate/policy.go:94-98,108-111` | OPA evaluation failures masked | `queryOPAString` returns `""` on cmd error; `queryOPAStringSet` returns `nil`. Policy evaluation failures (OPA crash, missing data) are indistinguishable from "no denials/warnings". Security decisions may be wrong. |
| `internal/evidence/auto_attest.go:178-224` | collectTestResults uses output on error | `cmd.Output()` error: code still iterates `string(out)` (may be nil/partial). `passed`/`failed` counts can be wrong; status set to "fail" but coverage/parsing may use corrupt data. |

---

## HIGH

| File:Line | Issue | Description |
|-----------|-------|-------------|
| `internal/evidence/discrepancy.go:133,140,151` | Ignored filepath.Glob errors | `matches, _ := filepath.Glob(pattern)` — Glob can fail (permission, I/O). Empty matches may mean "error" not "no files". Affects attestation discovery. |
| `internal/orchestrate/hydrate.go:53` | Ignored bdShow error | `out, _ := bdShow(projectRoot, beadsID)` — Dependency description wrong/empty on failure. |
| `internal/orchestrate/hydrate.go:60` | Ignored os.ReadFile | `agentsContent, _ := os.ReadFile(agentsPath)` — AGENTS.md read failure yields empty quality gates. |
| `internal/orchestrate/hydrate.go:62` | Ignored gitStatusPorcelain | `pkt.DriftStatus, _ = gitStatusPorcelain(projectRoot)` — Git status failure yields empty drift; repo state hidden. |
| `internal/evidence/auto_attest.go:157` | Ignored runGit error | `out, _ := runGit(repoRoot, "log", ...)` — Commit list wrong on failure; beads ID extraction may miss IDs. |
| `internal/evidence/auto_attest.go:197,210` | Type assertion ignores ok | `action, _ := evt["Action"].(string)` — Wrong type yields empty string; event classification wrong. |
| `internal/orchestrate/invoke_opencode.go:134` | Ignored WritePromptProvenance | `_ = WritePromptProvenance(...)` — Provenance write failure silent; audit trail broken. |
| `internal/orchestrate/loop.go:26` | Ignored SaveCheckpoint | `_ = SaveCheckpoint(cpPath, cp)` — On shutdown, checkpoint may not persist; resume could re-run last phase. |
| `internal/orchestrate/attest.go:259` | Ignored cmd2.Output | `out2, _ := cmd2.Output()` — Fallback git diff output ignored on error; changed files may be wrong. |
| `internal/modelgateway/router.go:147-151` | Ignored tenantStore.Get error | `tenant, err := r.tenantStore.Get(ctx, input.TenantID)` — If err != nil, tenant config not applied. Routing may use wrong provider/constraints. |
| `internal/monitor/stuck_detector.go:192-203` | Return nil on I/O error | `getLastEventTime`: Open fails → return `(sessionID, lastEvent, nil)`; Seek/Read fail → return `nil`. ModTime used as fallback but I/O errors are masked; stuck detection may use stale timestamps. |
| `internal/orchestrate/state_machine.go:140-141` | Redundant Glob, second ignores error | First Glob checks err; second `ents, _ := filepath.Glob(...)` ignores error. If second fails, valid project root can be skipped. |
| `internal/session/paths.go:96` | Panic in production | `panic("session: InitPaths not called")` in `MustGetPaths()` — Uninitialized use crashes process. Consider returning error or ensuring init at startup. |

---

## MEDIUM

| File:Line | Issue | Description |
|-----------|-------|-------------|
| `internal/orchestrate/attest.go:156-166` | Missing scanner.Err() | `bufio.Scanner` loop never checks `scanner.Err()`. I/O errors during read are silently ignored; mapping data may be partial. |
| `internal/evidence/auto_attest.go:347-365` | Missing scanner.Err() | Same; scope prefix extraction from workstream files. |
| `internal/docsync/docsync.go:196-203` | Missing scanner.Err() | `splitNonEmpty` uses Scanner without Err check. |
| `internal/eval/framework.go:77-105` | Missing scanner.Err() | `extractAgentOutput` uses Scanner without Err check; transcript parsing may be incomplete. |
| `internal/orchestrate/migration_shim.go:321` | Ignored RecordEvent | `_ = s.telemetry.RecordEvent(ctx, event)` — Telemetry loss; dry-run issues may be hidden. |
| `internal/orchestrate/fsm_v2.go:350` | Ignored EmitEventAsync | `_ = f.eventProducer.EmitEventAsync(ctx, event)` — Event emission failure silent. |
| `internal/modelgateway/credentials.go:323` | Ignored audit Log | `_ = cm.audit.Log(ctx, entry)` — Audit log failure silent (documented best-effort). |
| `internal/authz/tenant_scope.go:224` | Ignored crossTenantLogger.Log | `_ = a.crossTenantLogger.Log(entry)` — Cross-tenant audit loss. |
| `internal/orchestrate/hydrate.go:74,86,90,136` | return err without wrapping | Multiple bare `return nil, err` / `return err`; caller loses context. |
| `internal/evidence/auto_attest.go:119,146,463` | return err without wrapping | Bare returns; no `%w` for traceability. |
| `internal/planner/scheduler.go:157` | return err without wrapping | GetPlan error returned without context. |
| `internal/modelgateway/credentials.go:249,257` | return err without wrapping | Store Get/Set errors returned bare. |
| `internal/orchestrate/migration_shim.go:190,250-259,357,402` | return err without wrapping | Migration flow errors. |
| `internal/orchestrate/invoke_opencode.go:86,93,96,100` | return err without wrapping | MkdirAll, Marshal, WriteFile, Rename. |
| `internal/orchestrate/hooks.go:62,72` | return err without wrapping | LoadHookConfig, runHook. |
| `internal/bridge/beads_sink.go:134,176` | return err without wrapping | createBeadsIssue. |
| `internal/evidence/trace_validator.go:136,140,151` | return err without wrapping | ReadFile, Unmarshal, Marshal. |
| `internal/evidence/discrepancy.go:148-150` | Misplaced comment | Comment block inside `if prefix != ""`; confusing. |

---

## LOW

| File:Line | Issue | Description |
|-----------|-------|-------------|
| `internal/beads/sql_client.go:121` | defer rows.Close() ignores error | `defer rows.Close()` — sql.Rows.Close() returns error. Same at 166, 244, 266. Common pattern; low impact. |
| `internal/beads/dependency.go` | defer rows.Close() ignores error | 7 occurrences. |
| `internal/beads/client.go:124,165` | defer rows.Close() ignores error | Same pattern. |
| `internal/modelgateway/adapters/selfhosted.go` | defer resp.Body.Close() ignores error | HTTP body close. |
| `internal/monitor/stuck_detector.go:196` | defer f.Close() ignores error | File close. |
| `internal/orchestrate/attest.go:146,219` | defer f.Close() ignores error | File close in defer. |
| `internal/orchestrate/policy.go:60,62,65` | defer Close/Remove | Temp file cleanup; best-effort. |
| `internal/ciloop/cleanup.go:21` | _ = os.Remove | Orphan tmp cleanup; best-effort. |
| `internal/ciloop/autofixer.go:182` | _ = opts.DecisionLogger | Best-effort logging. |
| `internal/ciloop/fixer.go:92,127` | _ = DecisionLogger, os.Remove | Best-effort. |
| Test files | _ = in setup | Acceptable in tests; not production. |

---

## Summary

| Severity | Count |
|----------|-------|
| CRITICAL | 5 |
| HIGH | 13 |
| MEDIUM | 18 |
| LOW | 10+ |

---

## Recommended Next Steps

1. **CRITICAL:** Fix ToJSON methods to return error or propagate; add logging in LoadExistingFindings.
2. **CRITICAL:** Policy query: return error from OPA evaluation; do not mask with empty string/nil.
3. **HIGH:** Add error wrapping (`%w`) to all propagated errors; fix scanner.Err() checks.
4. **HIGH:** Audit all ignored errors (Glob, bdShow, gitStatusPorcelain, etc.) — add logging or propagate where appropriate.
5. **MEDIUM:** Replace bare `return err` with `return fmt.Errorf("context: %w", err)`.
6. Map each finding to Beads issue via `bd create` for tracking.
