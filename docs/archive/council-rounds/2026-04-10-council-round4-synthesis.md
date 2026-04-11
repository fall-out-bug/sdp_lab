# LLM Council Report: AI Architect Phase A — Round 4

**Date:** 2026-04-10
**Rounds:** 4 of 5
**Consensus:** NOT REACHED — 2 DOMAIN VETOes active, critical pipeline bug found
**Subject:** Validation of Round 3 fixes

## Council Members

| Role | Model | Status | Verdict |
|------|-------|--------|---------|
| **Critic** | Gemini 3.1 Pro | Complete | OPPOSE + DOMAIN VETO |
| **Technician** | DeepSeek V3.2 | Complete | CONDITIONAL |
| **Philosopher** | Kimi K2.5 | Partial (truncated) | CONDITIONAL |
| **Pragmatist** | MiniMax M2.7 | Complete | SUPPORT |
| **Engineer** | MiMo V2 Pro | Complete | SUPPORT |
| **Architect** | GPT 5.4 (Codex) | Complete | OPPOSE + DOMAIN VETO |

**Tally:** 2 SUPPORT / 2 CONDITIONAL / 2 OPPOSE (both with DOMAIN VETO)

---

## Critical Bugs Found

### BUG 1: ScrubSecrets pipeline order is CATASTROPHICALLY WRONG

**Flagged by:** Critic, Technician, Architect

The SecureEnricher pipeline is specified as:
```
SanitizeForLLM → WrapForLLM → API call → JSON validate → ScrubSecrets → SanitizeOutput
```

`ScrubSecrets` runs **AFTER** the API call. This means secrets are sent to the LLM provider before being scrubbed — the exact data exfiltration the spec was written to prevent.

**Fix:** Move ScrubSecrets before WrapForLLM:
```
ScrubSecrets → SanitizeForLLM → WrapForLLM → API call → JSON validate → SanitizeOutput
```

### BUG 2: hex8 delimiters = only 32-bit entropy (brute-forceable)

**Flagged by:** Critic

8 hex characters = 16^8 = ~4 billion combinations. An attacker triggering multiple evaluations can brute-force the delimiter boundary.

**Fix:** Change from hex8 to hex32 (128-bit, crypto/rand).

### BUG 3: bluemonday is HTML sanitizer, NOT Markdown sanitizer

**Flagged by:** Critic

`bluemonday.UGCPolicy()` is an HTML sanitizer. Raw Markdown like `[click](javascript:alert(1))` passes through because it's Markdown syntax, not HTML. When later rendered to HTML by the client, XSS executes.

**Fix:** Either (a) convert Markdown to HTML first, then apply bluemonday, or (b) use a Markdown-native AST sanitizer.

### BUG 4: NPM scoped packages collide with `/` delimiter

**Flagged by:** Critic

`@types/node` uses `/`, which collides with the ID delimiter. Parsing `typescript/@types/node/module` by splitting on `/` misaligns.

**Fix:** Use `::` as delimiter, or URL-encode components before joining with `/`.

### BUG 5: "Alphabetical" ordering contradicts explicit list

**Flagged by:** Critic

Spec says "Go > Python > Java > TypeScript > SQL (alphabetical)". Alphabetical would be G, J, P, S, T. The explicit list is G, P, J, T, S. This is not alphabetical.

**Fix:** Remove "(alphabetical)", use explicit index array.

---

## Secondary Concerns

| Issue | Raised By | Severity |
|-------|-----------|----------|
| Windows TOCTOU not implemented (needs `GetFinalPathNameByHandle`) | Critic, Technician | MEDIUM |
| Shannon entropy false positives (UUIDs, hashes, base64) | Critic, Technician | MEDIUM |
| gitleaks subprocess per-enrichment = DoS via process table | Critic | MEDIUM |
| Mermaid `postMessage` bypass in sandboxed iframe | Technician | MEDIUM |
| Spec-code drift: specs updated, code still uses old types | Architect | HIGH |
| Extractor spec defines different `ModuleBoundary`/`LayerAssignment` shapes | Architect | HIGH |

---

## Convergence

| Metric | Round 2 | Round 3 | Round 4 |
|--------|---------|---------|---------|
| SUPPORT (both I1+I5) | 0 | 2 | 2 |
| DOMAIN VETO active | 2 | 3 | 2 (Critic + Architect) |
| Critical bugs | 0 | 0 | **5** |
| Models saying "ship" | 0 | 2 | 2 |

**The council regressed.** Round 3's fixes introduced new bugs (pipeline order, entropy, Markdown sanitizer mismatch). These are real engineering errors, not philosophical disagreements.

---

## Required Fixes (Round 5 prerequisite)

| # | Fix | Effort |
|---|-----|--------|
| R1 | Reorder pipeline: ScrubSecrets before API call | 1 line spec change |
| R2 | Delimiter: hex8 → hex32 (128-bit) | 1 line spec change |
| R3 | Markdown sanitization: render MD→HTML first, then bluemonday | ~3 lines |
| R4 | ID delimiter: `/` → `::` or URL-encoded components | ~5 lines |
| R5 | Remove "alphabetical", use explicit index array | 1 line |
| R6 | Document Windows unsupported for TOCTOU-safe paths | 2 lines |
| R7 | gitleaks: use compiled rules (not subprocess) + allowlist for hashes/UUIDs | ~3 lines |
