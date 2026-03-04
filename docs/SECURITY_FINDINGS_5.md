# Security Findings — Fifth Comprehensive Review (2026-03-03)

**Scope:** `internal/` directory  
**Focus:** Security vulnerabilities after previous fixes — injection, path traversal, unsafe operations, credential handling, SSRF, TOCTOU

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
| `internal/ciloop/autofixer.go:151` | **HIGH** | `f.Command` from `.sdp/auto-fixers.yaml` is split via `SplitCommand` and passed to `exec.CommandContext(runCtx, parts[0], parts[1:]...)`. If config is writable by untrusted input (e.g., PR), command injection. Built-in fixers use safe commands; user-added fixers are arbitrary. |

**Recommendation:** Validate fixer config source. Ensure only trusted paths can define fixers. Consider allowlist of executables.

---

### MEDIUM: OPA policy paths from config

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/orchestrate/policy.go:88,102` | **MEDIUM** | `opaPath`, `policiesDir`, `inputFile`, `query` passed to `exec.Command(opaPath, "eval", ...)`. Paths from project root; query from Rego file. If policies dir is compromised, path/query injection possible. |

**Recommendation:** Validate paths are under project root; restrict query to fixed Rego paths.

---

### LOW: Beads issueID passed to `bd show`

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/beads/client.go:274` | **LOW** | `issueID` passed as arg to `exec.Command("bd", "show", issueID, "--json")`. Args are separate (no shell); issueID from internal data. Low risk unless bd mishandles args. |

---

## 2. Path Traversal

### HIGH: runID in LoadTraceEventsFromRunFile without validation

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/evidence/trace_validator.go:110` | **HIGH** | `path := filepath.Join(workDir, ".sdp", "runs", runID+".json")` — runID is passed in with no validation. If runID is `../../../etc/passwd`, path escapes `.sdp/runs` and `os.ReadFile(path)` reads outside intended directory. |

**Recommendation:** Validate runID (e.g., alphanumeric, hyphen, underscore only) before use. Reuse `ValidateSessionID`-style checks.

---

### MEDIUM: runID and EvidenceDir in discrepancy findAttestation

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/evidence/discrepancy.go:65-69,130-139` | **MEDIUM** | `findAttestation(opts.EvidenceDir, runID, prefix)` uses `filepath.Join(dir, prefix+runID+".json")` and `prefix+"*"+runID+"*.json"` for Glob. runID with `..` or glob metacharacters (`*`, `[`, `?`) can match unintended files or produce paths outside EvidenceDir. |

**Recommendation:** Validate runID and EvidenceDir. Restrict runID to safe chars; ensure EvidenceDir is under project root.

---

### MEDIUM: agent-constraints Path from YAML

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/orchestrate/constraints.go:141` | **MEDIUM** | `path := strings.ReplaceAll(c.Path, "{feature_id}", featureID)` then `fullPath := filepath.Join(projectRoot, path)`. c.Path from `.sdp/agent-constraints.yaml`. If Path is `../../../etc/passwd` or `{feature_id}/../../secret`, path traversal. |

**Recommendation:** Validate path after replacement; ensure fullPath is under projectRoot using `filepath.Rel` or similar.

---

### MEDIUM: Markdown link resolution can escape docs root

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/docsync/docsync.go:167-168` | **MEDIUM** | `resolved := filepath.Clean(filepath.Join(filepath.Dir(path), target))` — target from markdown link syntax (for example `../../../etc/passwd`) can push resolution outside `docsRoot`. Currently only `os.Stat(resolved)` is used (existence check), but pattern is unsafe if code later reads the file. |

**Recommendation:** Verify `resolved` is under `docsRoot` using `filepath.Rel(projectRoot, resolved)` and reject if result starts with `..`.

---

### LOW: entry.Name() in bridge findings

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/bridge/github_findings.go:267` | **LOW** | `path := filepath.Join(dir, entry.Name())` — entry from `os.ReadDir(dir)`. Filtered by `.json` suffix. On some systems filenames can contain `..`. |

**Recommendation:** Use `filepath.Clean` or validate `entry.Name()` before join.

---

### MITIGATED: Session/wisp/formula IDs

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
| `internal/evidence/operator_gate.go:38` | **LOW** | `status, _ := env["status"].(string)` — ignores ok; wrong type yields empty string. |
| `internal/evidence/inspect.go:146-147,171-172` | **LOW** | Same pattern with `comp["ok"].(bool)`, `src["path"].(string)` etc. |
| `internal/evidence/strict.go:56,120,124,128,146-148` | **LOW** | `prURL, _ := trace["pr_url"].(string)` etc. |
| `internal/bridge/github_findings.go:221` | **LOW** | `checkName, _ := source["check_name"].(string)` |

**Recommendation:** Use `v, ok := x.(T); if !ok { ... }` where type correctness matters.

---

## 5. Credential Exposure in Logs/Errors

### MEDIUM: Audit log may include error details

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/modelgateway/credentials.go:111` | **MEDIUM** | `cm.auditLog(ctx, tenantID, providerID, "get", "system", false, err.Error())` — if `err` ever contains credential data (e.g., from buggy store), it could be logged. |

**Recommendation:** Sanitize `errMsg` before audit; never log raw errors from credential operations.

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
| `internal/evidence/discrepancy.go:50-52,130-132` | **MEDIUM** | `opts.EvidenceDir` and `runID` used in `findAttestation` without validation. Path traversal and glob injection possible. |

**Recommendation:** Validate `runID` and `EvidenceDir` (e.g., under project root).

---

### LOW: Template execution with user data

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/bridge/beads_sink.go:371-379` | **LOW** | `RenderIssueTemplate(tmpl string, data *IssueTemplateData)` — if `tmpl` is user-supplied, Go `text/template` can expose info via `{{.}}`. `IssueTemplateData` fields from findings — ensure no sensitive data. |

---

## 7. TOCTOU Race Conditions

### LOW: Write-then-rename pattern

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/orchestrate/hydrate.go:117-123` | **LOW** | `os.WriteFile(tmpPath, ...)` then `os.Rename(tmpPath, path)`. Brief window where partial write exists; rename is atomic on same filesystem. |
| `internal/orchestrate/invoke_opencode.go:95` | **LOW** | Same pattern for prompt provenance. |
| `internal/ciloop/autofixer.go:122-127` | **LOW** | Same pattern for diagnostics file. |

**Recommendation:** Use `O_EXCL` or temp in same dir; already acceptable for same-fs rename.

---

### LOW: Session store read-modify-write

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/workstream/session_store.go:218-227` | **LOW** | `UpdateWispStatus` reads wisp, modifies, writes. No file locking; concurrent updates could overwrite. |

**Recommendation:** Add `flock` or use DB for concurrent access.

---

## 8. SSRF Vulnerabilities

### MEDIUM: GitHub Actions OIDC URL from environment

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/evidence/sigstore_signer.go:311,325,334` | **MEDIUM** | `requestURL := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_URL")` passed to `fetchGitHubOIDCToken` and used in `http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)`. GitHub Actions sets this; a malicious workflow author could set it to an internal URL (e.g., `http://169.254.169.254/` or internal service) to probe or exfiltrate. |

**Recommendation:** Validate that URL host is `*.github.com` or `*.githubusercontent.com` before making the request. Reject non-HTTPS or unexpected hosts.

---

### LOW: Self-hosted provider BaseURL from config

| File:Line | Severity | Description |
|-----------|----------|-------------|
| `internal/modelgateway/adapters/selfhosted.go:92` | **LOW** | `p.config.BaseURL+"/health"` used in HTTP request. BaseURL from tenant/config. If config is user-controlled, SSRF possible. Typically admin-controlled. |

---

## 9. Unsafe Deserialization

### LOW: JSON/YAML unmarshal without schema validation

| File:Line | Severity | Description |
|-----------|----------|-------------|
| Multiple | **LOW** | `json.Unmarshal` and `yaml.Unmarshal` used throughout. Go's decoder does not execute code. Risk is type confusion or excessive memory if input is maliciously large. Most inputs are from trusted sources (config, CI output). |

**Recommendation:** For untrusted input, use `json.Decoder` with `UseNumber()` or limit input size; validate structure before use.

---

## Summary Table

| Category | CRITICAL | HIGH | MEDIUM | LOW |
|----------|----------|------|--------|-----|
| Command injection | 1 | 2 | 1 | 1 |
| Path traversal | 0 | 1 | 4 | 1 |
| SQL injection | 0 | 1 | 0 | 0 |
| Unsafe type assertions | 0 | 0 | 0 | 4+ |
| Credential exposure | 0 | 0 | 1 | 0 |
| Missing validation | 0 | 0 | 2 | 1 |
| TOCTOU | 0 | 0 | 0 | 2 |
| SSRF | 0 | 0 | 1 | 1 |
| Unsafe deserialization | 0 | 0 | 0 | 1 |

---

## Recommended Beads Mapping

Each finding can be mapped to a Beads issue for tracking:

- **Command injection (hooks):** `bd create --title "SEC: Pipeline hooks command injection" --label security --type task`
- **Command injection (escalation):** `bd create --title "SEC: Escalation notify command injection" --label security --type task`
- **Command injection (autofixer):** `bd create --title "SEC: Autofixer command from config" --label security --type task`
- **Path traversal (trace_validator runID):** `bd create --title "SEC: Validate runID in LoadTraceEventsFromRunFile" --label security --type task`
- **Path traversal (discrepancy runID):** `bd create --title "SEC: Validate EvidenceDir and runID in discrepancy" --label security --type task`
- **Path traversal (constraints Path):** `bd create --title "SEC: Validate agent-constraints Path" --label security --type task`
- **Path traversal (docsync link):** `bd create --title "SEC: Validate resolved path under docsRoot" --label security --type task`
- **ExecuteRawQuery:** `bd create --title "SEC: ExecuteRawQuery SQL injection risk" --label security --type task`
- **Credential audit errMsg:** `bd create --title "SEC: Audit log error message sanitization" --label security --type task`
- **baseBranch validation:** `bd create --title "SEC: Validate baseBranch in auto-attest" --label security --type task`
- **SSRF (OIDC URL):** `bd create --title "SEC: Validate GitHub OIDC URL host" --label security --type task`

---

*Generated 2026-03-03. Fifth comprehensive review. Review and prioritize per deployment context.*
