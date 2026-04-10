# LLM Council Report: AI Architect Implementation Specs

**Date:** 2026-04-10
**Rounds:** 1 of 5
**Consensus:** NOT REACHED (1 round completed, issues require fixes before Round 2)

## Council Members

| Role | Model | Status | Response Size |
|------|-------|--------|---------------|
| **Critic** | Gemini 3.1 Pro | Complete | 8.8 KB |
| **Technician** | DeepSeek V3.2 | Complete | 10.0 KB |
| **Philosopher** | Kimi K2.5 | Complete | 14.0 KB |
| **Pragmatist** | MiniMax M2.7 | Complete | 5.8 KB |
| **Engineer** | MiMo V2 Pro | Complete | 9.6 KB |
| **Architect** | GPT 5.4 (Codex) | In Progress | — |

---

## Round 1 Synthesis

### UNANIMOUS CONSENSUS (5/5)

| # | Finding | Action |
|---|---------|--------|
| 1 | **DO NOT START CODING** — specs are 40-60% ready | Complete blocking specs first |
| 2 | **Data Model is P0 blocker** — DependencyInfo dual-purpose, APISurface undefined | Fix before anything else |
| 3 | **Evaluation Harness is cross-cutting, NOT WS12** — must exist from day 1 | Extract to parallel workstream |
| 4 | **Tree-sitter must be Phase 1** — regex insufficient, contains security vulnerabilities (ReDoS) | Fix 5 query errors immediately |
| 5 | **C4 Algorithm absent** — "completely missing" per all reviewers | Specify deterministic graph construction |

### STRONG MAJORITY (4/5)

| # | Finding | Dissent |
|---|---------|---------|
| 6 | **SQL stays P2** — too complex for MVP, different parsing paradigm | Philosopher: evaluate by information entropy, not difficulty |
| 7 | **3-4 impl specs minimum before coding** (not 7) | Critic: 4 "core" specs; Pragmatist: 3 minimum |
| 8 | **Roles overlap** — 6 is too many, consolidate to 3-4 | Philosopher: use antagonistic pairs instead |

### SPLIT VOTES

| # | Issue | Position A | Position B |
|---|-------|-----------|-----------|
| 9 | **C4 approach** | Deterministic first, LLM enrichment second (Critic, Engineer, Pragmatist) | Define schema/ontology first, algorithm is secondary (Philosopher, Technician) |
| 10 | **How many languages for MVP** | Go + Python only for Phase 1 (Pragmatist) | All 4 non-SQL languages (Critic, Technician) |

---

## Per-Model Highlights

### Critic (Gemini 3.1 Pro) — Security Focus
- **Key insight:** Indirect prompt injection via source code — malicious PR comments can hijack LLM
- **New issue:** ReDoS vulnerability in regex fallback
- **C4 proposal:** Deterministic graph → LLM enrichment (LLM only adds descriptions, never nodes/edges)
- **Verdict:** Security Auditor and Data Steward roles should replace Philosopher/Pragmatist

### Technician (DeepSeek V3.2) — Feasibility
- **Key insight:** "Specs are 40% ready, workstreams 60% structured"
- **New issues:** Performance SLAs absent, error handling gap, no integration testing plan
- **Critical path:** DataModel → C4 Algorithm → Extractor Integration → Coding
- **Timeline:** 4 weeks minimum before coding can start

### Philosopher (Kimi K2.5) — Reframing
- **Key insight:** "Category error — conflation of grammar contracts with parser bindings"
- **Framework:** Dependency Inversion Principle — extractors should depend on abstract LanguageGrammar, not concrete tree-sitter
- **C4 reframe:** Define graph ontology (nodes + edges + invariants) before algorithm
- **Roles:** 3 antagonistic pairs > 6 harmonious roles

### Pragmatist (MiniMax M2.7) — Scope Cutting
- **Key insight:** "80% of content is elaboration, 20% is actionable"
- **Cut proposal:** Reduce to 2 languages (Go/Python), 3 impl specs, 5 workstreams
- **SQL:** P2 with explicit defer criteria
- **C4:** Text-first MVP (no Mermaid), 100-line spec, done in a day

### Engineer (MiMo V2 Pro) — Implementation
- **Key insight:** "12 engineer-days before coding can begin"
- **Concrete fixes:** Provided corrected tree-sitter queries, data model resolution, error handling pattern
- **C4 algorithm:** Provided working pseudocode (build_reference_model)
- **New issues:** Missing concurrency spec, framework detection priority, memory limits

---

## Open Issues for Round 2

After fixing the identified blockers, Round 2 should address:

1. **Data Model Resolution** — Apply Engineer's `ModuleDependency`/`ComponentDependency` split or Philosopher's "3 conceptual domains"?
2. **C4 Algorithm** — Deterministic-first (Critic/Engineer) or Schema-first (Philosopher/Technician)?
3. **Language scope for MVP** — Go+Python only (Pragmatist) or Go+Python+Java+TS (design spec)?
4. **Workstream consolidation** — 12 files → 5 (Pragmatist) or keep 12 with eval extracted?
5. **Security architecture** — Prompt injection defense, ReDoS mitigation, SecurityFilter pipeline integration
6. **Performance SLAs** — Max file parse time, memory limits, repo size handling

---

## Round 2 Prompt

Fix the 5 identified blockers, then re-run council on the fixed artifacts:

1. Fix tree-sitter queries (5 errors)
2. Resolve DependencyInfo dual-purpose
3. Write C4 generation algorithm spec
4. Extract evaluation harness from WS12
5. Define APISurface type
