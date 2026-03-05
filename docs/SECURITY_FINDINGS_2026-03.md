# Security Findings — Fourth Comprehensive Review (2026-03-02)

**Scope:** `internal/` directory  
**Focus:** Injection, path traversal, unsafe operations, credential handling, TOCTOU

---

## 1. Command Injection

### CRITICAL: Pipeline hooks execute arbitrary YAML command via `sh -c`

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/orchestrate/hooks.go:86` | **CRITICAL** | `h.Command` from `.sdp/pipeline-hooks.yaml` is passed to `exec.CommandContext(..., "sh", "-c", h.Command)`. Any user/attacker who can write this file (e.g., via PR, compromised dev machine) can execute arbitrary shell commands. |

**Recommendation:** Document that pipeline-hooks.yaml is a trusted config file. Consider restricting to allowlisted commands or removing `sh -c` and using explicit argv.

---

### HIGH: Escalation notify command executes config via `sh -c`

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/monitor/escalation.go:94` | **HIGH** | `eh.notifyCmd` from `EscalationConfig.NotifyCommand` is passed to `exec.CommandContext(..., "sh", "-c", eh.notifyCmd)`. Config comes from operator/deployment; if config source is compromised, arbitrary execution. |

**Recommendation:** Document trust boundary. Consider allowlist or structured command format.

---

### HIGH: Autofixer commands from YAML config

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/ciloop/autofixer.go:151` | **HIGH** | `f.Command` from fixer registry (YAML) is split and passed to `exec.CommandContext(runCtx, parts[0], parts[1:]...)`. If `.sdp/ci-fixers.yaml` or equivalent is writable by untrusted input, command injection via crafted command string. |

**Recommendation:** Validate fixer config source. Ensure only trusted paths can define fixers.

---

### MEDIUM: OPA policy paths from config

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/orchestrate/policy.go:88,102` | **MEDIUM** | `opaPath`, `policiesDir`, `inputFile`, `query` passed to `exec.Command(opaPath, "eval", "--data", policiesDir, "--input", inputFile, ..., query)`. If any come from user/config, path or query injection possible. |

**Recommendation:** Validate paths are under project root; query is fixed set of Rego paths.

---

### LOW: Beads escalation title passed to `bd create`

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/monitor/escalation.go:82` | **LOW** | `title` (contains `sessionID`) passed as last arg to `bd create`. `sessionID` is internal; if `bd` mishandles args, theoretical injection. Unlikely. |

---

## 2. Path Traversal

### MEDIUM: Formula name validated but search paths include `$HOME`

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/beads/formula_parser.go:123,179-191` | **MEDIUM** | `FindFormula(name)` validates `..` and `/` in `name`, but `searchPaths` includes `filepath.Join(os.Getenv("HOME"), ".beads/formulas")`. Malicious `HOME` (e.g., `HOME=/etc`) could change resolution. |

**Recommendation:** Validate `HOME` or restrict search paths to project-relative dirs.

---

### LOW: `entry.Name()` in bridge findings

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/bridge/github_findings.go:267` | **LOW** | `path := filepath.Join(dir, entry.Name())` — `entry` from `os.ReadDir(dir)`. Filesystem controls names; `entry.Name()` can contain `..` on some systems. |

**Recommendation:** Use `filepath.Clean` or validate `entry.Name()` before join.

---

### MITIGATED: Session/wisp IDs

| File:Line | Status |
|-----------|--------|
| `internal/session/paths.go:21-34` | `ValidateSessionID` rejects `..`, `/`, `\`, invalid chars |
| `internal/workstream/session_store.go:17-31` | `validateID` same checks |
| `internal/beads/formula_parser.go:170-175` | Formula name validated for `..` and path separators |

---

## 3. SQL / NoSQL Injection

### HIGH: ExecuteRawQuery accepts raw query string

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/beads/sql_client.go:261` | **HIGH** | `ExecuteRawQuery(query string, args ...interface{})` passes `query` directly to `db.Query(query, args...)`. If `query` comes from user/config/API, SQL injection. |

**Recommendation:** Restrict to read-only, document as admin-only, or remove. If kept, add strict allowlist of query patterns.

---

### MITIGATED: Other SQL usage

All other `db.Query`/`QueryRow` calls use parameterized queries with `?` placeholders. No string concatenation of user input into SQL.

---

## 4. Unsafe Type Assertions

### LOW: Comma-ok used but `_` discards failure

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/evidence/operator_gate.go:32,35` | **LOW** | `got, _ := env["role"].(string)` — ignores ok; wrong type yields empty string. Logic may not handle that. |
| `internal/evidence/inspect.go:104,107,124,154,157,160` | **LOW** | Same pattern with `id, _ := intent["issue_id"].(string)` etc. |
| `internal/evidence/strict.go:29,97,100,112,135` | **LOW** | `t, _ := raw["_type"].(string)` etc. |

**Recommendation:** Use `v, ok := x.(T); if !ok { ... }` where type correctness matters.

---

### MITIGATED: Proper comma-ok

| File:Line | Status |
|-----------|--------|
| `internal/orchestrate/merge_policy.go:275` | `policyStr, ok := policy.(string); ok` |
| `internal/orchestrate/fsm_v2.go:247` | `te, ok := transitionErr.(*TransitionError); ok` |
| `internal/modelgateway/provider.go:286` | `pErr, ok := err.(*ProviderError); ok` |

---

## 5. Credential Exposure in Logs/Errors

### MEDIUM: Audit log may include error details

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/modelgateway/credentials.go:307-320` | **MEDIUM** | `auditLog(..., errMsg)` — `errMsg` from callers can include `fmt.Sprintf("failed to mark old credential as rotating: %s", err)`. If `err` ever contains credential data (e.g., from buggy store), it could be logged. |

**Recommendation:** Sanitize `errMsg` before audit; never log raw errors from credential operations.

---

### LOW: Test fixtures use placeholder keys

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/modelgateway/adapters/adapters_test.go:21,29,37,...` | **LOW** | `APIKey: "sk-test"` etc. in tests. Ensure these never reach production logs. |

---

### MITIGATED: Credential struct

`Credential.APIKey` has `omitempty` in JSON; `RevokeCredential` clears `cred.APIKey = ""`. No obvious direct logging of API keys in production paths.

---

## 6. Missing Input Validation

### MEDIUM: baseBranch from CLI in auto-attest

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/evidence/cmd/auto-attest/main.go:12,26` | **MEDIUM** | `baseBranch` from `-base-branch` flag passed to `runGit(..., "origin/"+baseBranch+"...HEAD")`. No validation; malicious ref names could confuse git. |

**Recommendation:** Validate `baseBranch` against `^[a-zA-Z0-9/_.-]+$` or similar.

---

### MEDIUM: Evidence dir and runID in discrepancy

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/evidence/discrepancy.go:50-52,130-132` | **MEDIUM** | `opts.EvidenceDir` and `runID` used in `findAttestation(dir, runID, prefix)` with `filepath.Join(dir, prefix+runID+".json")`. If `runID` contains `..` or path separators, path traversal. |

**Recommendation:** Validate `runID` and `EvidenceDir` (e.g., under project root).

---

### LOW: Template execution with user data

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/bridge/beads_sink.go:371-379` | **LOW** | `RenderIssueTemplate(tmpl string, data *IssueTemplateData)` — if `tmpl` is user-supplied, Go `text/template` can expose server info via `{{.}}`. `IssueTemplateData` fields (File, Message, etc.) are from findings — ensure no sensitive data. |
| `internal/workstream/template.go:198-205` | **LOW** | `substituteVars` uses `fmt.Sprintf("%v", value)` — formula-driven values could inject content into generated markdown. |

**Recommendation:** Use `text/template` with no `{{define}}`/`{{template}}` for user templates; sanitize finding data.

---

## 7. TOCTOU Race Conditions

### LOW: Write-then-rename pattern

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/orchestrate/hydrate.go:117-123` | **LOW** | `os.WriteFile(tmpPath, ...)` then `os.Rename(tmpPath, path)`. Brief window where partial write exists; rename is atomic on same filesystem. |
| `internal/orchestrate/invoke_opencode.go:95` | **LOW** | Same pattern for prompt provenance. |

**Recommendation:** Use `O_EXCL` or temp in same dir; already acceptable for same-fs rename.

---

### LOW: Session store read-modify-write

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/workstream/session_store.go:218-227` | **LOW** | `UpdateStatus` reads wisp, modifies, writes. No file locking; concurrent updates could overwrite. |

**Recommendation:** Add `flock` or use DB for concurrent access.

---

## Summary Table

| Category | CRITICAL | HIGH | MEDIUM | LOW |
|----------|----------|------|--------|-----|
| Command injection | 1 | 2 | 1 | 1 |
| Path traversal | 0 | 0 | 1 | 1 |
| SQL injection | 0 | 1 | 0 | 0 |
| Unsafe type assertions | 0 | 0 | 0 | 4+ |
| Credential exposure | 0 | 0 | 1 | 1 |
| Missing validation | 0 | 0 | 2 | 2 |
| TOCTOU | 0 | 0 | 0 | 2 |

---

## Recommended Beads Mapping

Each finding can be mapped to a Beads issue for tracking:

- **Command injection (hooks):** `bd create --title "SEC: Pipeline hooks command injection" --label security`
- **Command injection (escalation):** `bd create --title "SEC: Escalation notify command injection" --label security`
- **Command injection (autofixer):** `bd create --title "SEC: Autofixer command from config" --label security`
- **ExecuteRawQuery:** `bd create --title "SEC: ExecuteRawQuery SQL injection risk" --label security`
- **Path traversal (formula HOME):** `bd create --title "SEC: Formula parser HOME path" --label security`
- **Credential audit errMsg:** `bd create --title "SEC: Audit log error message sanitization" --label security`
- **baseBranch validation:** `bd create --title "SEC: Validate baseBranch in auto-attest" --label security`
- **Evidence dir/runID validation:** `bd create --title "SEC: Validate EvidenceDir and runID" --label security`

---

*Generated 2026-03-02. Review and prioritize per deployment context.*
