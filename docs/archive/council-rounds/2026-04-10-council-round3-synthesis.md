# LLM Council Report: AI Architect Phase A — Round 3

**Date:** 2026-04-10
**Rounds:** 3 of 5
**Consensus:** PARTIAL — I1/I5 mostly resolved, Critic DOMAIN VETO persists on security model
**Subject:** Validation of Data Model + Security specs after Round 2 alignment

## Council Members

| Role | Model | Status | Response Size |
|------|-------|--------|---------------|
| **Critic** | Gemini 3.1 Pro | Complete | 3.2 KB |
| **Technician** | DeepSeek V3.2 | Complete | 5.8 KB |
| **Philosopher** | Kimi K2.5 | Partial (truncated) | 0.6 KB |
| **Pragmatist** | MiniMax M2.7 | Complete | 4.1 KB |
| **Engineer** | MiMo V2 Pro | Complete | 5.2 KB |
| **Architect** | GPT 5.4 (Codex) | Complete | 4.4 KB |

---

## Round 3 Synthesis

### I1 (Data Model) — Mostly Resolved, Critic DOMAIN VETO Active

| Model | Verdict | Key Concern |
|-------|---------|-------------|
| Critic | **OPPOSE** | DOMAIN VETO: ID scheme uses `:` delimiter (collides with Windows paths, Maven coords). Last-writer-wins race condition with async extractors. |
| Technician | CONDITIONAL | Need validation invariants, field-level merge (not fragment-level), canonical JSON for content hash. |
| Philosopher | CONDITIONAL | Premature concretization — "build compiles clean" ≠ correct architecture. |
| Pragmatist | **SUPPORT** | "Don't refactor further. Ship it." |
| Engineer | **SUPPORT** | All types defined, deterministic IDs, build compiles. |
| Architect | CONDITIONAL | Cross-spec contract drift: extractor/C4 specs still reference old types. DependencyCorrelation orphaned. |

**Tally:** 2 SUPPORT / 3 CONDITIONAL / 1 OPPOSE (DOMAIN VETO)
**Status:** 5/6 agree the split is correct in principle. Critic's DOMAIN VETO on data integrity (delimiter collision, race condition) is addressable with spec fixes.

### I5 (Security) — Mostly Resolved, Critic DOMAIN VETO Persists

| Model | Verdict | Key Concern |
|-------|---------|-------------|
| Critic | **OPPOSE** | DOMAIN VETO: Blocklist-based prompt stripping is fundamentally broken. Need markdown sanitizer (not just HTML escaping). 4 regexes insufficient for secrets. TOCTOU in path validation. |
| Technician | CONDITIONAL | Delimiter must regenerate per call. Regex must be anchored. JSON schema validation is vague. |
| Philosopher | (partial) | Security patterns are "brittle." |
| Pragmatist | **SUPPORT** | All veto items addressed. "Move forward." |
| Engineer | **SUPPORT** | All 4 veto items covered. Implementation checklist actionable. |
| Architect | CONDITIONAL | ValidatePath needs absolute normalization. Security not bound into C4 enrichment pipeline. |

**Tally:** 2 SUPPORT / 2 CONDITIONAL / 1 OPPOSE (DOMAIN VETO) / 1 partial
**Status:** The Critic's position is philosophical: regex-based blocklists cannot solve prompt injection. This is a valid security stance but fundamentally changes the approach.

### N1 (Secrets) — Critic DOMAIN VETO

| Model | Verdict |
|-------|---------|
| Critic | **OPPOSE** — 4 regexes insufficient. Must integrate gitleaks or Shannon entropy. |
| Technician | SUPPORT |
| Pragmatist | SUPPORT |
| Engineer | SUPPORT |

**Tally:** 3 SUPPORT / 1 OPPOSE (DOMAIN VETO)

### N2 (Path Traversal) — Resolved with Conditions

| Model | Verdict | Condition |
|-------|---------|-----------|
| Critic | CONDITIONAL | TOCTOU race: must open file first, then validate fd path |
| Others | SUPPORT | — |

### N3 (XSS via Mermaid) — Resolved with Conditions

| Model | Verdict | Condition |
|-------|---------|-----------|
| Critic | CONDITIONAL | Mermaid must render in sandboxed iframe |
| Others | SUPPORT | — |

### N4-N7 — Deferred

| ID | Title | Status | Recommendation |
|----|-------|--------|----------------|
| N4 | Integration tests | INSUFFICIENT_EVIDENCE | Post-MVP per Pragmatist |
| N5 | Query performance | INSUFFICIENT_EVIDENCE | Revisit with real workload data |
| N6 | CrossCuttingConcern | INSUFFICIENT_EVIDENCE | Emergent during implementation |
| N7 | Spec overgrown | CONDITIONAL | Freeze specs, redirect to code |

### NEW Issues Raised in Round 3

| ID | Title | Severity | Raised By |
|----|-------|----------|-----------|
| N8 | ID delimiter collision (`:` in paths/Maven) | HIGH | Critic |
| N9 | Race condition in last-writer-wins merge | HIGH | Critic |
| N10 | Security regex blocklist is false sense of security | HIGH | Critic |
| N11 | Markdown injection bypasses HTML escaping | HIGH | Critic |
| N12 | TOCTOU in path traversal validation | HIGH | Critic |
| N13 | ReDoS via unbounded regex on security patterns | MEDIUM | Critic |
| N14 | Cross-spec contract drift (old types still referenced) | HIGH | Architect |
| N15 | DependencyCorrelation orphaned from ProfileFragment | MEDIUM | Architect |
| N16 | Non-deterministic merge order in extractor tier 5 | MEDIUM | Architect |

---

## Convergence Analysis

| Metric | Round 2 | Round 3 | Delta |
|--------|---------|---------|-------|
| Resolved issues | 2/6 | 4/9 (I2, I6, N2✓, N3✓) | +2 |
| P0 blockers | 1 (I1) | 1 (I5 — Critic DOMAIN VETO) | Same |
| DOMAIN VETO active | 2 (I1, I5) | 3 (I1, I5, N1) | +1 (Critic expanded) |
| Models saying "ship it" | 0 | 2 (Pragmatist, Engineer) | +2 |
| Models saying "fix first" | 5 | 3 (Critic, Technician, Architect) | -2 |

**Key insight:** Convergence is happening but the Critic is *hardening* rather than yielding. The Critic's position shifted from "fix these specific things" to "the entire security model is wrong" — this is a fundamental philosophical disagreement, not a fixable spec issue.

---

## Decision Required

**The council is split into two camps:**

**Camp A — "Ship Now, Iterate" (Pragmatist + Engineer):**
> I1 is correct. I5 addresses all veto items. Start coding. Security model is practical for MVP. Critic's concerns are theoretical for a local-only tool.

**Camp B — "Fix First" (Critic + Technician + Architect):**
> Security model has real gaps (blocklists don't work, TOCTOU, insufficient secret coverage). Data model has real bugs (delimiter collision, race condition). Fix these specific items (not a rewrite) before coding.

**Actionable fix list from Camp B:**

1. **I1 fixes (addressable):**
   - Change ID delimiter from `:` to `/` or use structured composite key
   - Change merge from "last-writer-wins" to "highest-precedence-wins"
   - Add DependencyCorrelation to ProfileFragment or mark internal-only
   - Fix extractor/C4 spec references to old types

2. **I5 fixes (addressable):**
   - Remove instruction-line stripping (rely only on delimiters + schema validation)
   - Add `bluemonday` sanitizer for Markdown output (not just HTML escaping)
   - Add Shannon entropy check or gitleaks integration for secrets
   - Fix TOCTOU: open file first, validate fd path
   - Bind security functions into C4 enrichment pipeline
   - Add iframe sandbox for Mermaid rendering

---

## Recommendation

Fix the 10 specific items from Camp B (estimated 0.5 engineer-day), then proceed to implementation. The Critic's DOMAIN VETO can be overridden by Decision Owner with explicit risk acceptance — the tool is local-only and does not process untrusted repos in a multi-tenant environment.
