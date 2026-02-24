# Phase 0 (F014–F021) Security Review — Round 3 Verdict

**Reviewer:** Security Expert (Troy Hunt principles)  
**Date:** 2026-02-23  
**Scope:** Agent Loop Reliability — fixer, checkpoint, runfile, exec usage, JSON parsing

---

## 1. Round-2 Fix Verification

### Fix 1: sanitizeFixDescs — DecisionLogger stdout ✅

**Location:** `internal/ciloop/fixer.go` lines 84–89, 175–187

- `DecisionLogger` receives only `sanitizeFixDescs(fixDescs)` — fix types (e.g. `go-test`, `go-build`), not raw log content.
- `sanitizeFixDescs` correctly extracts the prefix before `:` or truncates to 30 chars.
- **Verdict:** Correct and complete.

### Fix 2: Commit messages — fix type only ✅

**Location:** `internal/ciloop/fixer.go` lines 70–75

- Commit message uses `sanitizeFixDescs(fixDescs)` plus `FeatureID` (e.g. `F014`).
- No CI log content in the message.
- **Verdict:** Correct and complete.

---

## 2. New Vulnerabilities

### P1: Sensitive data in committed diagnostics file

**Location:** `internal/ciloop/fixer.go` `writeDiagnostics()` lines 95–116

**Issue:** The diagnostics file written to `.sdp/ci-fixes/fix-pr{N}-{timestamp}.md` contains:

- Raw `fixDescs` (e.g. `go-test: fix assertion: expected X got Y`, truncated to 60 chars)
- `truncate(log, 2000)` — up to 2000 chars of raw CI log

This file is staged via `git add .sdp/ci-fixes/` and committed. CI logs often include:

- Env vars, secrets, tokens
- Test output with sensitive data
- Build artifacts

**Impact:** Secrets and sensitive data can be committed and remain in git history.

**Recommendation:** Either:

- Sanitize diagnostics content (e.g. only fix types, no log excerpt), or
- Add `.sdp/ci-fixes/` to `.gitignore` and do not commit diagnostics, or
- Store diagnostics outside the repo (e.g. ephemeral artifact).

---

## 3. Input Validation (featureID path traversal)

### Checkpoint validation ✅

**Locations:** `internal/ciloop/checkpoint.go`, `internal/orchestrate/checkpoint.go`

- `validateFeatureID` rejects `/`, `\`, `.`, and empty strings.
- Used in `LoadCheckpoint` and `SaveCheckpoint`.
- **Verdict:** Adequate for checkpoint paths.

### AppendRunEvent — missing validation (P3)

**Location:** `internal/ciloop/runfile.go` lines 36–38, 72–78

- `AppendRunEvent` uses `featureID` in `findRunFile` as a prefix only; no path traversal in `filepath.Join`.
- No `validateFeatureID` call; defense in depth is missing.
- **Verdict:** P3 — add validation for consistency.

### EnsureRunFile — path construction (P3)

**Location:** `internal/orchestrate/cli.go` lines 32–35

- `runID := "oneshot-" + featureID + "-" + timestamp` → `filepath.Join(dir, runID+".json")`.
- If `featureID` contained `..`, path traversal would be possible.
- In practice, `EnsureRunFile` is only reached after `SaveCheckpoint`, which validates `featureID`.
- **Verdict:** P3 — add `validateFeatureID` for defense in depth.

---

## 4. exec.Command Injection Risks

**Locations:** `cmd/sdp-ci-loop/main.go`

| Call | Args | Risk |
|------|------|------|
| `bd create` | `--title`, `--labels` (includes `*feature`) | Args passed separately; no shell. Safe. |
| `git add` | Fixed path | Safe. |
| `git commit -m msg` | `msg` from sanitized fixDescs | Safe. |
| `git push` | No user input | Safe. |
| `gh run list/view` | `id` from gh CLI output | Safe (args, no shell). |
| `execRunner.Run` | `name`, `args...` | 30s timeout; args passed separately. Safe. |

**Verdict:** No injection risk. All uses of `exec.Command` / `exec.CommandContext` pass arguments as separate parameters.

---

## 5. JSON Parsing DoS

| File | Pattern | Limit |
|------|---------|-------|
| `internal/ciloop/checkpoint.go` | `io.LimitReader(..., maxJSONDecodeBytes)` | 10MB ✅ |
| `internal/ciloop/runfile.go` | Same | 10MB ✅ |
| `internal/orchestrate/checkpoint.go` | Same | 10MB ✅ |

**Verdict:** DoS protection in place for Phase 0 JSON decode paths.

---

## 6. Additional Notes

### Poller JSON (`internal/ciloop/poller.go` line 55)

- `json.Unmarshal(out, &raw)` on `gh pr checks` output — no `LimitReader`.
- Output size is bounded by GitHub API; low risk.
- **Verdict:** P3 — consider adding a limit for consistency.

### bd create labels

- `--labels "ci-finding,%s"` with `*feature` could allow label injection if `bd` treats commas specially.
- **Verdict:** P3 — consider sanitizing `feature` for labels (e.g. alphanumeric only).

---

## Verdict: **FAIL**

| Severity | Count | Findings |
|----------|-------|----------|
| P0 | 0 | — |
| P1 | 1 | Sensitive data in committed diagnostics file |
| P2 | 0 | — |
| P3 | 3 | AppendRunEvent/EnsureRunFile validation; poller JSON limit; bd labels |

**Rule:** PASS only if all findings are P2 or P3. One P1 finding → **FAIL**.

---

## Recommended Remediation

1. **P1 (required):** Sanitize or stop committing diagnostics content. Options:
   - Sanitize: only fix types in the file, no log excerpt.
   - Ignore: add `.sdp/ci-fixes/` to `.gitignore` and do not stage/commit it.
   - External: write diagnostics to an ephemeral artifact, not the repo.

2. **P3 (optional):** Add `validateFeatureID` to `AppendRunEvent` and `EnsureRunFile` for defense in depth.

3. **P3 (optional):** Add `LimitReader` for poller JSON and sanitize `feature` for bd labels.
