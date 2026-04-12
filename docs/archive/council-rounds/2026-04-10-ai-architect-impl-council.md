# AI Architect — Implementation Council Review

**Date:** 2026-04-10
**Reviewers:** 3 independent perspectives (Extractor Specialist, Workstream Planner, Gap Analyst)
**Artifacts reviewed:**
- `2026-04-10-ai-architect-impl-extractors.md` (1378 lines)
- `docs/workstreams/backlog/00-105-01.md` through `00-105-12.md`
- `2026-04-10-ai-architect-design.md` (v3)
- `2026-04-10-ai-architect-spec-ru.md` (v3.1)
- Existing code: types.go, assembler.go, security.go, extract/

---

## Executive Summary

**3 reviewers: 2x CONDITIONAL PASS, 1x GAP ANALYSIS**

The implementation artifacts are a significant improvement over the previous state (no specs at all). However, they are NOT yet implementation-ready. Key blockers:

1. Tree-sitter queries in the extractor spec contain **5 syntactic/semantic errors** that will fail at runtime
2. The C4 generation algorithm is **completely absent** — no spec describes how to build a ReferenceModel from a CodebaseProfile
3. LLM prompts are **under-specified** — no system prompts, no few-shot examples, no retry strategy
4. The workstream decomposition has **structural issues** — evaluation harness trapped as a late dependency, cross-language detection has no owner
5. **7 additional impl specs needed** before a developer can start work without making undocumented decisions

---

## Review 1: Extractor Implementation Spec

**Verdict: CONDITIONAL PASS**

### Critical Issues

**C1. Python tree-sitter queries (Section 1.2) are structurally wrong.**
- `import_from_statement` query uses nonexistent `relative_import` node type
- `import_prefix` is a field name, not a capture-able node
- Absolute and relative imports need separate query patterns
- **Fix:** Use field names from tree-sitter-python grammar: `module:`, `name:`

**C2. Java tree-sitter query (Section 1.3) uses nonexistent `static_import` node.**
- tree-sitter-java handles `static` via grammar rule, not a child node
- **Fix:** Use `(import_declaration (scoped_identifier) @import.path)` with separate static check

**C3. TypeScript queries (Section 1.4) use `#eq?` predicate unsupported by Go tree-sitter bindings.**
- `go-tree-sitter` does NOT support `#eq?` predicates
- **Fix:** Use structural matching + Go code filtering, or verify specific binding support

**C4. Express route regex (Section 2.5) contains backticks in Go raw string literal.**
- `["'`]` character class inside backtick-delimited string = syntax error
- **Fix:** Use `\x60` or `regexp.MustCompile("(?:app|router|server)\\.(get|...)\\s*\\(\\s*[\"'\\x60]...")`

**C5. Code references mismatch actual source.**
- `parseModulesFromPom()` doesn't exist (actual: `parseModules`)
- Line numbers drift from actual code

### Important Issues

- Flask route patterns will false-positive on any `@Object.get()` decorator (no Blueprint tracking algorithm provided)
- Layer detection scores `api/` as "presentation" — wrong for Go projects where `api/` holds interface definitions
- gRPC/GraphQL/Message extraction (Section 4.3) has no specific patterns — just descriptions
- Tree-sitter queries are Phase 2 but presented as Phase 1 deliverables (Section 6.3 contradicts Section 1)

### Strengths

- Framework detection tables are thorough with confidence values and detection signals
- Layer detection weighted scoring system is well-designed
- Module boundary detection covers all major build systems
- Known blind spots honestly documented for every language

---

## Review 2: Workstream Files

**Verdict: CONDITIONAL PASS**

### Phase A Coverage Matrix

| Phase A Item | Workstream(s) | Status |
|---|---|---|
| 1. SecurityFilter | 00-105-01 | FULL |
| 2. FileTree + DepManifest + SpecInventory | 00-105-03 | FULL |
| 3. InfraExtractor | 00-105-04 | FULL |
| 4. GeneratedCodeDetector | 00-105-03 | FULL |
| 5. CodebaseProfile assembly | 00-105-02 | FULL |
| 6. Evaluation harness framework | 00-105-12 | PARTIAL — trapped as late dependency |
| 7. Go extractor | 00-105-05 | FULL |
| 8. Python extractor | 00-105-06 | FULL |
| 9. Java/Kotlin extractor | 00-105-07 | FULL |
| 10. TypeScript extractor | 00-105-08 | FULL |
| 11. SQL analysis | 00-105-09 | FULL |
| 12. LLM hypothesis generation | 00-105-11 | FULL |
| 13. C4 L1/L2 | 00-105-10 | FULL |
| 14. C4 L3 | 00-105-10 | FULL |
| 15. CLI | 00-105-12 | FULL |
| 16. Cross-language dependencies | 00-105-03/12 | PARTIAL — no clear owner |
| 17. Golden repo suite | 00-105-12 | PARTIAL — late dependency |
| 18. Precision/recall measurement | 00-105-12 | PARTIAL — late dependency |
| 19. Known limitations docs | 00-105-12 | PARTIAL — late dependency |

**Coverage: 16/19 FULL, 3/19 PARTIAL, 0/19 MISSING**

### Critical Issues

**C1. Evaluation harness (item 6) buried in 00-105-12 as late dependency.**
- Design spec puts eval framework in "weeks 1-3" (infrastructure)
- Every extractor needs eval tests during development
- **Recommendation:** Extract into separate workstream, depends only on 00-105-02

**C2. Cross-language dependency detection (item 16) has no owner.**
- 5 patterns defined in design spec (API client/server, Protobuf, DB schema, message contracts, shared types)
- Neither 00-105-03 nor 00-105-12 has acceptance criteria testing these patterns
- **Recommendation:** Add explicit criteria to 00-105-10 or create dedicated workstream

**C3. 00-105-09 (SQL) is P2 but SQL is core MVP per design spec exit criteria.**
- Exit criteria require ">80% schema extraction accuracy" for SQL
- P2 priority means SQL ships last, potentially missing Phase A exit
- **Recommendation:** Upgrade to P1

### Important Issues

- 00-105-03 bundles 4 extractors into one M-sized workstream (aggressive)
- 00-105-04 doesn't depend on 00-105-01 but its acceptance criteria require SecurityFilter
- 00-105-10 doesn't depend on 00-105-09 (SQL) but SQL feeds C4 L2
- No workstream covers MetricsCollector or GitHistoryAnalyzer (may be intentional)

### Recommended Reordering

Split 00-105-12 into:
- **00-105-12a** CLI + Pipeline Integration (M, depends on 10, 11)
- **00-105-12b** Evaluation Harness + Golden Repos (L, depends on 02 only — starts early)

---

## Review 3: Gap Analysis

### Critical Gaps (blocks implementation)

**G1. C4 Generation Algorithm — completely missing.**
No document specifies HOW to build a ReferenceModel from a CodebaseProfile. The `c4/renderer.go` only renders an existing model to Mermaid — the model construction step is entirely absent.
→ Need: `2026-04-10-ai-architect-impl-c4-generation.md`

**G2. LLM prompt specifications are ad-hoc and under-specified.**
Prompts embedded in Go code, not in specs. No system prompts, no few-shot examples, no retry strategy, no token budget management.
→ Need: `2026-04-10-ai-architect-impl-llm.md`

**G3. Data model mismatch between ProfileFragment and CodebaseProfile.**
`ProfileFragment.Dependencies` is `[]DependencyInfo` (slice) but `CodebaseProfile.Dependencies` is `DependencyInfo` (single). Same type serves dual purpose. `APISurface`, `LayerAssignment`, `ModuleBoundary` types don't exist yet but are referenced in impl spec.
→ Need: `2026-04-10-ai-architect-impl-datamodel.md`

**G4. Contract catalog schema and discovery algorithm undefined.**
Design says "Layer 1 only" but no algorithm for mapping SpecArtifact → provider container → consumer → gap detection.
→ Need: `2026-04-10-ai-architect-impl-contracts.md`

**G5. No impl spec for CLI integration.**
How `sdp architect analyze` wires through assembler → hypothesizer → C4 → report writer.
→ Need: `2026-04-10-ai-architect-impl-cli.md`

**G6. No impl spec for augmentation pack integration.**
`architect.pack` format, skill registration, auto-invocation triggers — all undocumented.
→ Need: `2026-04-10-ai-architect-impl-augmentation.md`

**G7. ArchitectureReport construction undefined.**
How to assemble HypothesisResult + CodebaseProfile + ReferenceModel + ConfidenceSummary into final report.
→ Part of datamodel impl spec

### Important Gaps

- `DependencyInfo` overloaded (per-manifest AND aggregate)
- Tier system is destructive (Tier1 → Tier2 requires re-extraction)
- No incremental analysis / cache invalidation (Merkle tree mentioned but not implemented)
- Security filter never called in pipeline — purely advisory
- SQLAnalysis types don't match Russian spec examples
- DetectedPattern categories are free-form strings with no validation
- No error taxonomy for partial extractor failures

### Contradictions Found

1. **DependencyInfo structure:** Same type used as both per-manifest entry and aggregate
2. **Performance SLAs:** Table shows ~constant LLM time but text says "tiered RAG" for large repos
3. **Impl spec references types that don't exist:** APISurface, LayerAssignment, ModuleBoundary
4. **Tree-sitter phases:** Section 1 provides queries as Phase 1, Section 6.3 says Phase 1 is regex-only

---

## Recommended Additional Impl Specs

| Priority | Document | What it covers |
|----------|----------|---------------|
| P0 | `impl-datamodel.md` | Type definitions, field semantics, merge strategy, Profile→Report mapping |
| P0 | `impl-c4-generation.md` | ReferenceModel construction algorithm from CodebaseProfile |
| P0 | `impl-llm.md` | Prompt templates, retry strategy, token budgets, output validation |
| P1 | `impl-cli.md` | CLI commands, flags, output formats, augmentation pack wiring |
| P1 | `impl-contracts.md` | Contract discovery algorithm, catalog schema, gap detection |
| P2 | `impl-conformance.md` | Rule engine, YAML schema, evaluation harness framework |
| P2 | `impl-security.md` | Pipeline integration, enforcement mechanism, audit log schema |

---

## Action Items

### Before any implementation starts:
1. Fix tree-sitter query errors in extractor spec (C1-C3 from Review 1)
2. Fix regex syntax errors (C4 from Review 1)
3. Write `impl-datamodel.md` (G3 — types must be defined before extractors can emit them)
4. Write `impl-c4-generation.md` (G1 — C4 is the primary output artifact)

### Before extractor workstream (00-105-03..09) starts:
5. Write `impl-llm.md` (G2 — extractors need to know what data the LLM phase consumes)
6. Fix workstream dependencies (C1-C3 from Review 2)
7. Upgrade SQL priority to P1

### Before C4 workstream (00-105-10) starts:
8. Write `impl-cli.md` (G5)
9. Verify all new types compile and align with extractor outputs

### Deferred to Phase B:
- Contract catalog (G4) — but schema should be defined now
- Conformance rules (P2 spec)
- Augmentation pack (G6)
