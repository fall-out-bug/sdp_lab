## Round 2 ARCHITECT Council — 2026-04-10

### Issue Review

#### I1 — Data Model Resolution
**VERDICT**: SUPPORT

**EVIDENCE**: The extractor spec now freezes three new types, `ModuleBoundary`, `APISurface`, and `LayerAssignment`, with exact fields in Section 9, but it still does not define `DependencyInfo` even though assembly step 2.b says "Dependencies: merge DependencyInfo slices" ([docs/plans/2026-04-10-ai-architect-impl-extractors.md:1534-1606](docs/plans/2026-04-10-ai-architect-impl-extractors.md), [docs/plans/2026-04-10-ai-architect-impl-extractors.md:1883-1900](docs/plans/2026-04-10-ai-architect-impl-extractors.md)). That leaves two unresolved dependency models in the same design: `ModuleBoundary.Dependencies []string` for module relationships and legacy `DependencyInfo` for profile assembly, with no mapping between them. The C4 spec also defines the graph semantically, not as exact Go structs or JSON schema; Sections 1 and 4.1 specify node/edge meaning and count invariants, but not the serialized contract that Phase 1 and Phase 2 exchange ([docs/plans/2026-04-10-ai-architect-impl-c4.md:22-85](docs/plans/2026-04-10-ai-architect-impl-c4.md), [docs/plans/2026-04-10-ai-architect-impl-c4.md:314-320](docs/plans/2026-04-10-ai-architect-impl-c4.md)).

**PROPOSALS**:
1. Replace the current dual-purpose dependency model with two exact types this week: package/manifest dependencies and module/container dependencies, then remove the unresolved `DependencyInfo` merge placeholder.
2. Add exact Go structs or a JSON schema for `DeterministicGraph`, `EnrichedGraph`, `Node`, and `Edge`, including stable IDs and dedup keys.
3. Add one assembly test that proves how `ModuleBoundary.Dependencies`, import edges, and SQL/infra signals become C4 `Uses`/`PersistsTo` edges.

**CONFIDENCE**: High — the ambiguity is explicit in the documents, not inferred from implementation details.

#### I2 — C4 Algorithm
**VERDICT**: SUPPORT

**EVIDENCE**: The C4 doc now gives a real two-phase algorithm: deterministic structure creation in Phase 1 and annotation-only enrichment in Phase 2 ([docs/plans/2026-04-10-ai-architect-impl-c4.md:12-18](docs/plans/2026-04-10-ai-architect-impl-c4.md)). Sections 2 and 3 define concrete rules for system, container, component, `Uses`, `PersistsTo`, `Implements`, and `Exposes` creation, which is the missing algorithmic core from Round 1 ([docs/plans/2026-04-10-ai-architect-impl-c4.md:88-303](docs/plans/2026-04-10-ai-architect-impl-c4.md)). The validation guard then rejects any LLM response that changes counts, IDs, or names, and the sequence diagram repeats that Phase 2 is enrichment only ([docs/plans/2026-04-10-ai-architect-impl-c4.md:381-405](docs/plans/2026-04-10-ai-architect-impl-c4.md), [docs/plans/2026-04-10-ai-architect-impl-c4.md:641-656](docs/plans/2026-04-10-ai-architect-impl-c4.md)).

**PROPOSALS**:
1. Implement `ReferenceModelBuilder` directly from Sections 1-4, with Phase 1 and Phase 2 as separate interfaces.
2. Add invariant tests for the six graph invariants plus the Phase 2 no-structural-change rule.
3. Use WS13 mock extractors and golden repos to lock one monorepo case, one DB-inference case, and one cross-container import case before broader coding begins.

**CONFIDENCE**: High — the spec is now concrete enough to implement without filling major algorithmic gaps.

#### I3 — Language Scope
**VERDICT**: CONDITIONAL

**EVIDENCE**: SQL is now clearly in first-wave scope: it has a dedicated language section, a dedicated enhancement section with accuracy targets and tests, and it is present in `DefaultExtractors()` alongside Go, Python, Java, and TypeScript ([docs/plans/2026-04-10-ai-architect-impl-extractors.md:266-268](docs/plans/2026-04-10-ai-architect-impl-extractors.md), [docs/plans/2026-04-10-ai-architect-impl-extractors.md:2317-2383](docs/plans/2026-04-10-ai-architect-impl-extractors.md), [docs/plans/2026-04-10-ai-architect-impl-extractors.md:2447-2460](docs/plans/2026-04-10-ai-architect-impl-extractors.md)). WS13 also adds SQL fixtures and exit criteria to the eval harness, which is the right operational signal for P1 scope ([docs/workstreams/backlog/00-105-13.md:39-46](docs/workstreams/backlog/00-105-13.md)). The limit is explicit though: SQL is not treated as an import language and only contributes data-architecture signals, so the proposal is adopted for schema/database understanding, not for parity with code-language import analysis ([docs/plans/2026-04-10-ai-architect-impl-extractors.md:266-268](docs/plans/2026-04-10-ai-architect-impl-extractors.md)).

**PROPOSALS**:
1. State explicitly in the C4 builder contract that SQL contributes `Database` containers, ORM correlation, and `PersistsTo` edges, but not import graphs.
2. Add one mixed-language golden fixture where SQL migrations back a Go, Python, or TypeScript service so the ORM-to-table path is exercised end to end.
3. Gate SQL only on schema/query metrics, not on import or layer metrics that the spec explicitly excludes.

**CONFIDENCE**: High — the docs are clear on both the upgrade and the remaining scope boundary.

#### I4 — Workstream Consolidation
**VERDICT**: CONDITIONAL

**EVIDENCE**: Extracting evaluation into WS13 is a real improvement: the workstream is standalone, marked `P0`, and says evaluation starts in week 2 alongside extractors rather than at the end ([docs/workstreams/backlog/00-105-13.md:1-17](docs/workstreams/backlog/00-105-13.md), [docs/workstreams/backlog/00-105-13.md:55-64](docs/workstreams/backlog/00-105-13.md)). The extractor spec also moves evaluation earlier in implementation order by placing the evaluation harness ahead of the language extractor enhancements ([docs/plans/2026-04-10-ai-architect-impl-extractors.md:2482-2499](docs/plans/2026-04-10-ai-architect-impl-extractors.md)). The remaining problem is ownership drift: Section 16 of the extractor spec still contains a full harness design, while WS13 separately owns scope files and acceptance criteria, so the same capability now has two normative-looking homes ([docs/plans/2026-04-10-ai-architect-impl-extractors.md:1924-2013](docs/plans/2026-04-10-ai-architect-impl-extractors.md), [docs/workstreams/backlog/00-105-13.md:22-47](docs/workstreams/backlog/00-105-13.md)).

**PROPOSALS**:
1. Make WS13 the single owner of harness scope and acceptance; convert extractor Section 16 into a referenced implementation appendix, not a second source of truth.
2. Split WS13 internally into harness core, fixtures, and metrics so dependency ordering is obvious even if the backlog item stays single.
3. Add a rule this week that any extractor or C4 workstream must update a golden fixture in the same PR once WS13 exists.

**CONFIDENCE**: Medium — the improvement is real, but the remaining ownership split is still materially risky.

#### I5 — Security Architecture
**VERDICT**: SUPPORT

**EVIDENCE**: The extractor spec's security work is narrow and useful, but it is still about secret redaction and path scrubbing, not LLM threat modeling ([docs/plans/2026-04-10-ai-architect-impl-extractors.md:1611-1678](docs/plans/2026-04-10-ai-architect-impl-extractors.md)). The C4 flow says `SecurityFilter` sanitizes the graph before the Phase 2 prompt, but the prompt itself embeds raw `GraphJSON` and the validation guard only checks structural preservation; there is no spec for prompt-injection neutralization, field-length limits, instruction stripping, or semantic output validation beyond count/name checks ([docs/plans/2026-04-10-ai-architect-impl-c4.md:335-360](docs/plans/2026-04-10-ai-architect-impl-c4.md), [docs/plans/2026-04-10-ai-architect-impl-c4.md:381-405](docs/plans/2026-04-10-ai-architect-impl-c4.md), [docs/plans/2026-04-10-ai-architect-impl-c4.md:652-655](docs/plans/2026-04-10-ai-architect-impl-c4.md)). That means the current design protects secrets better than it protects the enrichment channel.

**PROPOSALS**:
1. Add a Phase 2 security addendum that defines how source-derived strings are bounded, normalized, or hashed before entering `GraphJSON`.
2. Add schema validation for enrichment output with max lengths and allowed field sets for `description`, `technologyTags`, `businessPurpose`, and `dataFlow`.
3. Add an adversarial mock-graph test in WS13 that includes instruction-like strings and proves the sanitization and validation pipeline rejects or neutralizes them.

**CONFIDENCE**: High — the missing control surface is visible in the prompt and validation sections.

#### I6 — Performance SLAs
**VERDICT**: CONDITIONAL

**EVIDENCE**: The extractor spec now has real SLAs and an enforcement mechanism in Section 7, including explicit limits for file parse time, extractor timeouts, concurrency, and total runtime ([docs/plans/2026-04-10-ai-architect-impl-extractors.md:1464-1514](docs/plans/2026-04-10-ai-architect-impl-extractors.md)). The problem is that the document is internally inconsistent: Section 7.2 allows five minutes for a 10K-file repo, while Section 22.2 requires under 60 seconds for a sub-10K repo and Section 24 repeats the 60-second acceptance gate ([docs/plans/2026-04-10-ai-architect-impl-extractors.md:1475-1481](docs/plans/2026-04-10-ai-architect-impl-extractors.md), [docs/plans/2026-04-10-ai-architect-impl-extractors.md:2423-2440](docs/plans/2026-04-10-ai-architect-impl-extractors.md), [docs/plans/2026-04-10-ai-architect-impl-extractors.md:2504-2512](docs/plans/2026-04-10-ai-architect-impl-extractors.md)). WS13 further says the full evaluation suite must stay under 60 seconds, so the council should not treat this as fully resolved yet ([docs/workstreams/backlog/00-105-13.md:45-47](docs/workstreams/backlog/00-105-13.md)).

**PROPOSALS**:
1. Collapse Sections 7, 22.2, and 24 into one canonical SLA table this week and delete the conflicting numbers.
2. Bind `ExtractorConfig` defaults and benchmark assertions to that single table so developers cannot implement against different budgets.
3. Separate extraction-only and extraction-plus-LLM budgets consistently, because the current text mixes them.

**CONFIDENCE**: High — the existence of SLAs is clear, and the contradiction is explicit.

### Round 1 Proposal Adoption Status

1. **Freeze 4 implementation contracts before coding begins — PARTIALLY ADOPTED.** The extractor spec now freezes exact structs for `ModuleBoundary`, `APISurface`, and `LayerAssignment`, and the C4 spec freezes graph semantics and Phase 2 count invariants ([docs/plans/2026-04-10-ai-architect-impl-extractors.md:1534-1606](docs/plans/2026-04-10-ai-architect-impl-extractors.md), [docs/plans/2026-04-10-ai-architect-impl-c4.md:22-85](docs/plans/2026-04-10-ai-architect-impl-c4.md), [docs/plans/2026-04-10-ai-architect-impl-c4.md:314-320](docs/plans/2026-04-10-ai-architect-impl-c4.md)). It is not fully adopted because `DependencyInfo` still lacks an exact contract and assembly still merges it by name only ([docs/plans/2026-04-10-ai-architect-impl-extractors.md:1883-1900](docs/plans/2026-04-10-ai-architect-impl-extractors.md)).

2. **Fix workstream dependency ordering — PARTIALLY ADOPTED.** WS13 now exists as an early P0 workstream and the extractor implementation order places evaluation ahead of the major language upgrades ([docs/workstreams/backlog/00-105-13.md:14-17](docs/workstreams/backlog/00-105-13.md), [docs/plans/2026-04-10-ai-architect-impl-extractors.md:2482-2499](docs/plans/2026-04-10-ai-architect-impl-extractors.md)). It is still only partial because the harness design is duplicated across WS13 and extractor Section 16, which weakens the dependency story even if the order improved ([docs/plans/2026-04-10-ai-architect-impl-extractors.md:1924-2013](docs/plans/2026-04-10-ai-architect-impl-extractors.md), [docs/workstreams/backlog/00-105-13.md:22-47](docs/workstreams/backlog/00-105-13.md)).

3. **Upgrade SQL to P1 language — ADOPTED.** SQL now has dedicated scope, dedicated enhancement work, dedicated tests, and presence in the default extractor registry, while WS13 includes SQL golden repos and thresholds ([docs/plans/2026-04-10-ai-architect-impl-extractors.md:266-268](docs/plans/2026-04-10-ai-architect-impl-extractors.md), [docs/plans/2026-04-10-ai-architect-impl-extractors.md:2317-2383](docs/plans/2026-04-10-ai-architect-impl-extractors.md), [docs/plans/2026-04-10-ai-architect-impl-extractors.md:2447-2460](docs/plans/2026-04-10-ai-architect-impl-extractors.md), [docs/workstreams/backlog/00-105-13.md:39-46](docs/workstreams/backlog/00-105-13.md)). The caveat is explicit in the spec: SQL is P1 for data architecture, not for import extraction.

4. **Build deterministic ReferenceModelBuilder (no LLM in structural pass) — ADOPTED.** The C4 spec now enforces deterministic Phase 1 structure creation, Phase 2 enrichment-only behavior, and post-enrichment structural validation ([docs/plans/2026-04-10-ai-architect-impl-c4.md:12-18](docs/plans/2026-04-10-ai-architect-impl-c4.md), [docs/plans/2026-04-10-ai-architect-impl-c4.md:88-303](docs/plans/2026-04-10-ai-architect-impl-c4.md), [docs/plans/2026-04-10-ai-architect-impl-c4.md:381-405](docs/plans/2026-04-10-ai-architect-impl-c4.md)). The sequence appendix reinforces the same split in executable flow terms ([docs/plans/2026-04-10-ai-architect-impl-c4.md:641-656](docs/plans/2026-04-10-ai-architect-impl-c4.md)).

### Remaining Critical Path

1. **Exact dependency and graph contract patch.** Deliver an exact type contract for `DependencyInfo` replacement/splitting plus serialized `DeterministicGraph`/`EnrichedGraph` structs and merge keys before implementation starts; current docs still mix `ModuleBoundary.Dependencies` with undefined `DependencyInfo` merging ([docs/plans/2026-04-10-ai-architect-impl-extractors.md:1539-1549](docs/plans/2026-04-10-ai-architect-impl-extractors.md), [docs/plans/2026-04-10-ai-architect-impl-extractors.md:1883-1900](docs/plans/2026-04-10-ai-architect-impl-extractors.md), [docs/plans/2026-04-10-ai-architect-impl-c4.md:47-75](docs/plans/2026-04-10-ai-architect-impl-c4.md)).

2. **Phase 2 security addendum and tests.** Ship a short spec plus tests for prompt-input sanitization, enrichment-output schema validation, and rejection behavior; secret scrubbing alone is not enough for a safe enrichment pass ([docs/plans/2026-04-10-ai-architect-impl-extractors.md:1611-1678](docs/plans/2026-04-10-ai-architect-impl-extractors.md), [docs/plans/2026-04-10-ai-architect-impl-c4.md:335-360](docs/plans/2026-04-10-ai-architect-impl-c4.md), [docs/plans/2026-04-10-ai-architect-impl-c4.md:381-405](docs/plans/2026-04-10-ai-architect-impl-c4.md)).

3. **WS13 metric contract.** Define exact precision/recall/F1 formulas, false-positive-budget semantics, and golden-file schema in the eval harness before any CI gate is wired; WS13 asks for these metrics, but Section 16 only defines generic field scoring ([docs/workstreams/backlog/00-105-13.md:36-47](docs/workstreams/backlog/00-105-13.md), [docs/workstreams/backlog/00-105-13.md:63-64](docs/workstreams/backlog/00-105-13.md), [docs/plans/2026-04-10-ai-architect-impl-extractors.md:1950-1997](docs/plans/2026-04-10-ai-architect-impl-extractors.md)).

4. **Canonical performance contract plus benchmark assertions.** Reconcile the conflicting 10K-repo budgets and encode the chosen SLA in both extractor config defaults and harness tests; otherwise implementation teams will target different numbers ([docs/plans/2026-04-10-ai-architect-impl-extractors.md:1475-1514](docs/plans/2026-04-10-ai-architect-impl-extractors.md), [docs/plans/2026-04-10-ai-architect-impl-extractors.md:2423-2440](docs/plans/2026-04-10-ai-architect-impl-extractors.md), [docs/plans/2026-04-10-ai-architect-impl-extractors.md:2504-2512](docs/plans/2026-04-10-ai-architect-impl-extractors.md), [docs/workstreams/backlog/00-105-13.md:45-47](docs/workstreams/backlog/00-105-13.md)).

5. **Source-of-truth cleanup for evaluation.** Decide whether WS13 or extractor Section 16 is normative, then cross-link and trim the other. Right now the design is materially better than Round 1, but still too easy to drift before coding begins ([docs/plans/2026-04-10-ai-architect-impl-extractors.md:1924-2013](docs/plans/2026-04-10-ai-architect-impl-extractors.md), [docs/plans/2026-04-10-ai-architect-impl-extractors.md:2482-2499](docs/plans/2026-04-10-ai-architect-impl-extractors.md), [docs/workstreams/backlog/00-105-13.md:22-47](docs/workstreams/backlog/00-105-13.md)).

### NEW_ISSUES

| ID | Title | Severity | Rationale |
|---|---|---|---|
| N1 | SLA Drift Between Sections 7, 22, and WS13 | HIGH | Section 7.2 allows **5 minutes** for a 10K-file repo, Section 22.2 requires **<60s** extraction for a sub-10K repo, Section 24 repeats the **under-60s** acceptance gate, and WS13 requires the full eval suite to stay under **60s** ([docs/plans/2026-04-10-ai-architect-impl-extractors.md:1475-1481](docs/plans/2026-04-10-ai-architect-impl-extractors.md), [docs/plans/2026-04-10-ai-architect-impl-extractors.md:2423-2430](docs/plans/2026-04-10-ai-architect-impl-extractors.md), [docs/plans/2026-04-10-ai-architect-impl-extractors.md:2508-2512](docs/plans/2026-04-10-ai-architect-impl-extractors.md), [docs/workstreams/backlog/00-105-13.md:45-47](docs/workstreams/backlog/00-105-13.md)). This is an implementability bug, not an editorial nit, because a benchmark can pass and fail at the same time depending on which section the engineer follows. |
| N2 | Mermaid Overflow Policy Conflict | MEDIUM | Section 5.3 says L3 diagrams allow up to **20 nodes** before splitting, but Section 5.4 says any diagram over **15 nodes** triggers large-diagram fallback artifacts ([docs/plans/2026-04-10-ai-architect-impl-c4.md:457-484](docs/plans/2026-04-10-ai-architect-impl-c4.md)). That changes emitted artifacts and test expectations, so the threshold must be normalized before implementation. |
| N3 | Eval Metric Formula Gap | HIGH | WS13 requires precision, recall, and F1 per ecosystem plus a **>95% precision** guard before CI integration, but extractor Section 16 only defines `FieldAccuracy` and `OverallScore` with no formula or weighting ([docs/workstreams/backlog/00-105-13.md:36-47](docs/workstreams/backlog/00-105-13.md), [docs/workstreams/backlog/00-105-13.md:63-64](docs/workstreams/backlog/00-105-13.md), [docs/plans/2026-04-10-ai-architect-impl-extractors.md:1950-1997](docs/plans/2026-04-10-ai-architect-impl-extractors.md)). Without a metric contract, teams can build incompatible harnesses that all claim to satisfy the same gate. |
