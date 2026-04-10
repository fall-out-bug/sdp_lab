# Council Round 6 Synthesis — AI Architect Implementation Specs

**Date:** 2026-04-10
**Status:** NEAR CONVERGENCE — 2 residual vetoes (1 model), 4 new spec gaps

## Verdict Matrix

| Veto Item | Critic | Technician | Philosopher | Pragmatist | Engineer | Architect | Lifted? |
|-----------|--------|------------|-------------|------------|----------|-----------|---------|
| V1 (Output) | SUPPORT | CONDITIONAL | SUPPORT | SUPPORT | CONDITIONAL | SUPPORT | YES |
| V2 (RNG) | SUPPORT | SUPPORT | SUPPORT | SUPPORT | SUPPORT | SUPPORT | YES |
| V3 (:: delimiter) | SUPPORT | SUPPORT | SUPPORT | SUPPORT | CONDITIONAL+VETO | SUPPORT | NO (1 veto) |
| V4 (TOCTOU) | SUPPORT | SUPPORT | SUPPORT | SUPPORT | SUPPORT | SUPPORT | YES |
| V5 (FD Leak) | SUPPORT | SUPPORT | SUPPORT | SUPPORT | CONDITIONAL | CONDITIONAL | YES |
| V6 (Error Recovery) | SUPPORT | SUPPORT | SUPPORT | SUPPORT | OPPOSE+VETO | SUPPORT | NO (1 veto) |

## Residual DOMAIN VETOes (2, from Engineer/Grok only)

### V3: Null byte delimiter normalization (Engineer)
**Issue:** `%00` percent-encoding in segments + verbatim storage creates ambiguous normalization. SplitID/JoinID must be explicitly idempotent and canonical.
**Fix:** Add invariant: "A well-formed ID never contains literal `\x00` except as delimiter." Mandate JoinID/SplitID as sole canonical functions with normalization before comparison.

### V6: Partial recovery undefined (Engineer)
**Issue:** EnrichmentResult doesn't define recoverable vs non-recoverable errors, idempotency, or consistency guarantees for partial data.
**Fix:** Define explicit error taxonomy, idempotency keys, and partial record sentinel flag.

## New Issues (Architect, no veto but blocks implementation)

### N1: ID hash-fallback contradicts validation
3-segment validation rule breaks when hash fallback appends 4th segment.
**Fix:** Hash goes INTO 3rd segment, not as 4th. Module name becomes `name~abc12345`.

### N2: Component vs C4Component naming
Data model defines `Component` but merge/output uses `C4Component`.
**Fix:** Unify to single type name.

### N3: Persistence format mismatch
Security spec stores sanitized HTML; data model reads as plain text.
**Fix:** Define explicit contract: LLMEnrichment fields store sanitized HTML.

### N4: openat() dirfd anchor underspecified
No dirfd specified — developer could use AT_FDCWD.
**Fix:** Specify: open repoRoot dirfd first, use as anchor for all subsequent openat calls.

## Additional Minor Items (no veto)
- Technician: ScrubSecrets on output should be JSON-aware (operate on values only)
- Engineer: V5 interface ambiguity (ReadCloser vs ReadSeekCloser)
- Critic (Gemini): V4 wants O_TMPFILE/unlink for inode protection (CONDITIONAL, no veto)

## Round 7 Plan
Fix all 6 items above. All are small, targeted edits. Then run Round 7 for convergence confirmation.
