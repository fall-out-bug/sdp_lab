# LLM Council Report: AI Architect Phase A — Round 2

**Date:** 2026-04-10
**Rounds:** 2 of 5
**Consensus:** PARTIAL — 1 P0 blocker (I1), 3 issues resolved, 2 need work
**Subject:** Updated specs after Round 1 blocker fixes

## Council Members

| Role | Model | Status | Response Size |
|------|-------|--------|---------------|
| **Critic** | Gemini 3.1 Pro | Complete | 6.7 KB |
| **Technician** | DeepSeek V3.2 | Complete | 6.4 KB |
| **Philosopher** | Kimi K2.5 | Complete | 8.8 KB |
| **Pragmatist** | MiniMax M2.7 | Complete | 3.2 KB |
| **Engineer** | MiMo V2 Pro | Complete | 7.8 KB |
| **Architect** | GPT 5.4 (Codex) | In Progress | — |

---

## Round 2 Synthesis

### RESOLVED — Consensus ≥80% (no vetoes)

| ID | Issue | Resolution | Confidence |
|----|-------|-----------|------------|
| I2 | C4 Algorithm | Deterministic-first spec (316 lines) accepted by all 5. LLM enrichment limited to descriptions only. Minor edge cases noted (circular deps, custom frameworks). | HIGH (4/5) / MEDIUM (Philosopher: schema ossification risk) |
| I6 | Performance SLAs | 5s/file, 1GB/10K, 30s extractor accepted. Enforcement mechanism needs testing. Memory cap per extractor suggested. | HIGH (4/5) |

### P0 BLOCKER — Domain Vetoes Active

| ID | Issue | Vetoes | Consensus |
|----|-------|--------|-----------|
| **I1** | **Data Model** | **Critic** (security), **Engineer** (implementation) | 5/5 AGREE: must resolve before ANY coding |

**Critic's position:** Split into `ASTDependencyRecord` (tree-sitter populated, read-only) and `LLMEnrichmentRecord` (descriptions only). Prevent structural edges from LLM-sourced data.

**Engineer's position:** Define concrete interfaces: `Module`, `ModuleDependency`, `Component` with deterministic IDs. Zero implementability without types.

**Technician's position:** `ContainerDependency` vs `ComponentDependency` split. 2-week schema design sprint.

**Pragmatist's position:** 3-page spec, draft TODAY. Reject any "iterate during implementation" proposals.

**Philosopher's position:** `DependencyInfo` is "schizophrenic concept." Needs `StructuralEdge` vs `TemporalContract` split (3-domain ontology).

### HIGH — Needs Work Before Implementation

| ID | Issue | Status | Action |
|----|-------|--------|--------|
| I5 | Security Architecture | **Critic DOMAIN VETO** — prompt injection not addressed | 1-page security spec: input sanitization, secrets filtering, LLM sandboxing, path traversal defense |
| I3 | Language Scope | SQL P1 accepted | Minor: add language support matrix |
| I4 | Workstreams | 13 files accepted with reservations | Pragmatist wants ≤7; Technician suggests WS14 baseline validation |

### NEW ISSUES Raised in Round 2

| ID | Title | Severity | Raised By |
|----|-------|----------|-----------|
| N1 | Secrets exfiltration via LLM API | CRITICAL | Critic |
| N2 | Path traversal / symlink exploits | HIGH | Critic |
| N3 | XSS via LLM-enriched Mermaid output | HIGH | Critic |
| N4 | Integration test framework absent | HIGH | Technician |
| N5 | DataModel query performance undefined | HIGH | Technician |
| N6 | CrossCuttingConcern node type missing from C4 | MEDIUM | Philosopher |
| N7 | Extractor spec overgrown (1400+ lines) | MEDIUM | Pragmatist |

---

## Critical Path to Implementation

**Agreed by 5/5 models:**

```
STEP 1 (TODAY): Data Model Spec
  → Split DependencyInfo into StructuralEdge + LLMEnrichment
  → Define Module, ModuleDependency, Component interfaces
  → 3 pages max (Pragmatist mandate)
  → BLOCKS: Everything

STEP 2 (TOMORROW): Security Spec
  → 1 page: prompt injection defense, secrets filtering, LLM sandboxing
  → Input sanitization with delimiter boundaries
  → filepath.EvalSymlinks for path traversal
  → BLOCKS: LLM enrichment (Phase 2 of C4)

STEP 3 (THIS WEEK): Start Coding
  → Data model types (internal/architect/model.go)
  → Extractor interface contracts
  → C4 deterministic graph builder (Phase 1, no LLM)
  → Performance SLA enforcement (ExtractorConfig)

STEP 4 (PARALLEL): Evaluation Harness (WS13)
  → Gold dataset: 3 sample repos
  → Cross-language integration tests
```

**Timeline estimate:**
- Technician: 6 weeks minimum to production readiness
- Pragmatist: start coding after Step 1+2 (this week)
- Engineer: 1 engineer-day for data model, 0.5 for language matrix

## Round 1 vs Round 2 Comparison

| Metric | Round 1 | Round 2 | Delta |
|--------|---------|---------|-------|
| Resolved issues | 0/6 | 2/6 (I2, I6) | +2 |
| P0 blockers | 5 | 1 (I1) | -4 |
| New issues raised | 6 | 7 | +1 |
| Models saying "don't code" | 5/5 | 1/5 (Engineer) | Major convergence |
| Models saying "code after I1" | 0/5 | 4/5 | Consensus shifted |

**Convergence: Round 1 → Round 2 moved from "DO NOT CODE" to "CODE AFTER DATA MODEL SPEC".**
