# Council Round 5 Synthesis — AI Architect Implementation Specs

**Date:** 2026-04-10
**Status:** NOT CONVERGED — 6 active DOMAIN VETOes, 3 new critical issues

## Verdict Matrix

| Item | Critic | Technician | Philosopher | Pragmatist | Engineer | Consensus |
|------|--------|------------|-------------|------------|----------|-----------|
| BUG 1 (Pipeline) | OPPOSE+VETO | CONDITIONAL | CONDITIONAL | SUPPORT | SUPPORT | NO |
| BUG 2 (Entropy) | CONDITIONAL+VETO | SUPPORT | CONDITIONAL | SUPPORT | SUPPORT | NO |
| BUG 3 (Markdown) | OPPOSE | CONDITIONAL | CONDITIONAL | CONDITIONAL | CONDITIONAL | NO |
| BUG 4 (:: delimiter) | — | OPPOSE+VETO | OPPOSE+VETO | SUPPORT | CONDITIONAL | NO |
| BUG 5 (Precedence) | — | SUPPORT | CONDITIONAL | SUPPORT | SUPPORT | YES |
| I1 (Data Model) | — | CONDITIONAL | CONDITIONAL | CONDITIONAL | CONDITIONAL | PARTIAL |
| I5 (Security) | — | OPPOSE+VETO | CONDITIONAL | CONDITIONAL | CONDITIONAL | NO |

## Active DOMAIN VETOes (6)

### V1: BUG 1 — Output Pipeline Corruption (Critic)
**Issue:** `SanitizeOutput` applies MD→HTML→bluemonday to raw JSON API responses, destroying JSON structure. No output-side secret scrubbing.
**Required Fix:** Output pipeline: `API call → ScrubSecrets(output) → JSON validate → field-level SanitizeOutput`

### V2: BUG 2 — RNG Not Specified (Critic)
**Issue:** Spec says "hex32" but doesn't mandate `crypto/rand`. `math/rand` would make delimiters predictable.
**Required Fix:** Explicitly mandate `crypto/rand` for delimiter generation.

### V3: BUG 4 — `::` Delimiter Collisions (Technician + Philosopher)
**Issue:** `::` appears in C++ (`std::vector`), Rust (`std::io::Error`), Java (`Class::method`), Maven Gradle DSL. URL-encoding `%2F` has double-encoding ambiguity.
**Required Fix:** Replace delimiter scheme entirely.

### V4: I5 — TOCTOU Illusory Fix (Technician)
**Issue:** `ValidatePath` returning `*os.File` with empty `Name()` doesn't prevent TOCTOU on Unix. No `openat()`-style protection.
**Required Fix:** Use `openat()` + `O_NOFOLLOW` on file descriptors, not paths.

### V5: ValidatePath FD Leak (Engineer)
**Issue:** No cleanup mechanism for abandoned file descriptors. Possible FD exhaustion.
**Required Fix:** Return wrapped `io.ReadCloser` with guaranteed cleanup, not raw `*os.File`.

### V6: Pipeline Error Recovery (Engineer)
**Issue:** No retry logic, circuit breaker, or partial result recovery on API failure.
**Required Fix:** Add retry with exponential backoff and circuit breaker.

## Critical Findings Not Vetoed

- **Concurrency race:** ScrubSecrets may leak via shared caches if goroutines process same artifact
- **LLM output injection:** Malicious LLM could return `<script>` in JSON string values
- **Entropy allowlist over-permissive:** "package lock lines" allowance too broad (Engineer OPPOSE+VETO)
- **bluemonday policy singleton risk:** UGCPolicy global state may interfere across instances
- **WrapForLLM delimiter injection:** User-crafted input containing delimiter could manipulate parsing

## Round 6 Fix Plan

### FIX-1: Replace `::` delimiter with length-prefixed framing
**Rationale:** Philosopher and Technician both converge on abandoning delimiters. Length-prefixed (4-byte big-endian + content) eliminates injection domain entirely. UUID delimiters add 36 chars per boundary and still have theoretical collision risk.

**Spec changes:**
- `datamodel.md`: ID format becomes `hex(language_code):hex(path):hex(module)` with length prefixes
- Alternative: keep human-readable with `\x00` (null byte) delimiter + base64 content encoding
- **Chosen approach:** Use `\x00` null byte delimiter. Null bytes are invalid in POSIX paths, NPM names, Maven coordinates, and all target ecosystems. Content segments are NOT encoded (remain human-readable for debugging). IDs are stored as Go strings (Go allows null bytes in strings).

### FIX-2: Fix output pipeline
**Spec changes in security.md:**
- Output pipeline: `API response → ScrubSecrets(output) → JSON Validate → Field-Level SanitizeOutput`
- `SanitizeOutput` operates on parsed JSON struct's individual string fields, not raw JSON
- Add `SanitizeField(value string) string` that applies MD→HTML→bluemonday per field

### FIX-3: Mandate crypto/rand
**Spec change in security.md:**
- Explicit: "Delimiter bytes MUST be generated using `crypto/rand.Read()`. Use of `math/rand` is a security violation."

### FIX-4: TOCTOU with openat + FD cleanup
**Spec changes in security.md:**
- `ValidatePath` uses `unix.Openat()` with `O_NOFOLLOW | O_RDONLY`
- Returns `io.ReadCloser` wrapper with `Close()` guaranteed via `runtime.SetFinalizer` + explicit defer
- Document: no path-dependent operations after validation

### FIX-5: Pipeline resilience
**Spec changes in security.md:**
- Add `RetryConfig` with exponential backoff (max 3 retries, 1s/2s/4s)
- Add circuit breaker: 5 consecutive failures → 30s cooldown
- Partial results: return `Result{Completed: false, Partial: ..., Error: ...}` on failure

### FIX-6: Tighten entropy allowlist
Replace "package lock lines" with exact patterns:
- UUID: `^[0-9a-f]{8}-[0-9a-f]{4}-...$`
- Integrity hashes: `^sha(256|512)-[A-Za-z0-9+/=]+$`
- SHA256 hex: `^[0-9a-f]{64}$`

## Consensus Status
- **5/5 bugs:** 1 resolved (BUG 5), 4 need fixes
- **2/2 gaps (I1, I5):** Both still conditional
- **Round 6 needed:** Yes
