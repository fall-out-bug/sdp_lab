# AI Architect Module — Design Document v3

**Date:** 2026-04-10
**Status:** Draft v3 (post council-of-models review, 7 independent critiques integrated)
**Codename:** AI Architect

---

## What AI Architect Is

An **Architecture Documentation & Hypothesis Assistant** — a component of SDP that helps teams **document, understand, and enforce** architecture of external target repositories.

**It is NOT:**
- An architecture detection engine (it produces **hypotheses**, not labels)
- A standalone product (embedded in the SDP pipeline as an augmentation pack)
- A 7th canonical agent (the 6-agent model stays intact)

**Language coverage:** MVP ships with support for the 5 most impactful ecosystems: **Go, Python, Java/Kotlin, TypeScript/JavaScript, and SQL**. Additional languages added by demand. Each language has documented accuracy tiers — no false promises.

**Core value proposition:** Helps architects document and enforce what they already know, while discovering what they might have missed.

---

## Problem Statement

No existing tool combines:
- Architecture hypothesis generation from code + infra analysis
- C4 diagram generation tied to runtime deployment topology
- Integration contract discovery with lifecycle management
- Conformance checking embedded in CI

Competitors each solve a slice: Structure101 (conformance), CodeScene (hotspots/coupling), ArchUnit (rules), Lattix (DSM), Backstage (catalog). The whitespace is an **AI-augmented workflow** connecting discovery → documentation → enforcement.

---

## Design Decisions Summary

| Aspect | Decision | Confidence | Source |
|--------|----------|-----------|--------|
| Analysis engine | Hybrid: structural extraction → CodebaseProfile → LLM interpretation | High | 7/7 models consensus |
| Code parsing | Tree-sitter default, regex fallback for unsupported langs | High | 7/7 models: regex alone is insufficient |
| Architecture classification | Scored hypothesis profile (8 types, not 15) | High | 6/7 models: hexagonal/clean/onion indistinguishable |
| Contract lifecycle | 3 states: observed → proposed → reference | High | 7/7 called brilliant |
| Contract discovery | Layer 1 (spec discovery) for MVP; inference deferred | Medium | 6/7: inference is 10x harder than estimated |
| Adoption model | probe → catalog → govern | High | Codex proposal, clearer than 4-level maturity |
| SDP integration | Augmentation pack, not new agent | High | 7/7 consensus |
| Artifact storage | Target repo `.sdp/architecture/`, distributed per directory for monorepos | High | Codex + Gemini 3.1 Pro |
| Security | Secret scrubbing + LLM opt-in + local LLM option | Critical | 7/7: non-negotiable |
| MVP scope | 5 ecosystems: Go, Python, Java/Kotlin, TS/JS, SQL | High | User requirement: cover popular languages |

---

## 1. Analysis Engine

### 1.1 Two-Phase Architecture

```
Target Repository
        |
        v
[Phase 1: Deterministic Extraction]  ← pure Go, no LLM
    |
    |-- FileTreeAnalyzer          directory structure, naming patterns
    |-- DependencyManifestParser  go.mod, package.json, Cargo.toml, pom.xml, etc.
    |-- ImportGraphExtractor      tree-sitter (default) or regex (fallback)
    |-- SpecInventoryScanner      OpenAPI, AsyncAPI, Proto, GraphQL, ADR, Docker, Terraform, CI
    |-- InfraExtractor            Dockerfile, docker-compose, k8s, Helm
    |-- MetricsCollector          LOC, test ratio, fan-out, file counts
    |-- GitHistoryAnalyzer        co-change coupling, ownership, change frequency
    |-- SecurityFilter            secret detection, PII scrubbing BEFORE any LLM call
    |-- GeneratedCodeDetector     file headers, known codegen patterns → exclude from analysis
        |
        v
   CodebaseProfile (JSON)
        |
        v
[Phase 2: LLM Interpretation]  ← on sanitized profile only
    |-- ArchitectureHypothesizer  → scored style hypothesis
    |-- PatternDetector           → patterns with evidence + confidence
    |-- RiskAssessor              → risks, debt indicators
        |
        v
   ArchitectureReport (JSON)
```

### 1.2 CodebaseProfile: Information Bottleneck with Tiered Depth

The profile compresses structural signals. But compression is lossy — different questions need different depth.

**Tier 1 — System Overview (~2K tokens):** Always available. Containers, languages, external deps, spec inventory. Sufficient for L1/L2 C4 and style hypothesis.

**Tier 2 — Container Detail (~5-15K per container):** On-demand drill-down. Import graph, component list, API surfaces, test coverage. Used for L3 C4 and conformance.

**Tier 3 — Component Source (~10-30K per component):** On-demand. Actual code for specific files. Used for L4, pattern detection, contract inference.

Context assembly is **question-driven**: design review uses Tier 1 + selective Tier 2. Impact analysis starts from changed files (Tier 3) and traces up.

### 1.3 Architecture Style Hypothesis (8 types, not 15)

Reduced from 15 to 8 reliably detectable types. Others require explicit human input.

```go
type StyleHypothesis struct {
    Styles []StyleScore `json:"styles"`
    // HumanInputNeeded lists styles that cannot be determined from code alone
    HumanInputNeeded []string `json:"human_input_needed,omitempty"`
}

type ArchStyle string
const (
    // Reliably detectable (>60% confidence from code + infra)
    StyleLayered              ArchStyle = "layered"              // 60-85%
    StyleModular              ArchStyle = "modular"              // 60-80%
    StyleMicroservices        ArchStyle = "microservices"        // 60-95% (with infra signals)
    StyleEventDriven          ArchStyle = "event_driven"         // 50-70% (with message broker detection)
    StyleServerless           ArchStyle = "serverless"           // 75% (with infra configs)
    StyleMonorepoMultiService ArchStyle = "monorepo_multi_service" // 80-95%
    StyleLibrary              ArchStyle = "library"              // 85-95%
    StyleInfraRepo            ArchStyle = "infra_repo"           // 90-98%

    // Detected only as secondary characteristics, not primary labels
    // StylePortsAndAdapters — collapsed hexagonal/clean/onion, requires human confirmation
    // StyleCQRS — requires event store + command/query separation evidence
    // StylePipeAndFilter — confused with functional style
)
```

**Naming:** "Hypothesis" not "Detection." Output always includes confidence intervals and evidence. Low-confidence hypotheses are explicitly marked "requires human validation."

### 1.4 Security: Non-negotiable Pre-filter

Before ANY data leaves the local machine:

```go
type SecurityFilter struct {
    SecretPatterns  []regexp.Regexp  // AWS keys, API tokens, passwords
    PIIScrubber     func(string) string  // hash internal package prefixes, usernames
    AllowExternalLLM bool  // explicit opt-in required, default false
}
```

- **Secret detection:** Scan env vars, config files, connection strings. Hash or redact before LLM.
- **PII scrubbing:** Internal package names (`com.company.secret`) → `pkg.<hash>`. File paths with usernames → anonymized.
- **Local LLM default:** Self-hosted models (Llama, CodeLlama) as primary path. Cloud LLMs only with `--allow-external-llm`.
- **Audit log:** Every LLM call logged with sanitized input hash for compliance.

### 1.5 Language Support: 5 Ecosystems at MVP

Tree-sitter is the default parser for all languages. Each ecosystem gets a dedicated extractor that handles language-specific import resolution, framework conventions, and known edge cases.

#### Core 5 Ecosystems (ship at MVP)

| Ecosystem | Parser | Import Accuracy | What works well | Known blind spots |
|-----------|--------|----------------|-----------------|-------------------|
| **Go** | `go/packages` (native) | **90-95%** | Static imports, module graph, interface detection | Build tags, CGo, dot imports, `go generate` output |
| **Python** | tree-sitter + requirements.txt/pyproject.toml | **60-70%** | Top-level imports, package structure, Flask/FastAPI/Django route detection | `importlib`, `sys.path` manipulation, conditional imports, notebook (.ipynb) imports |
| **Java/Kotlin** | tree-sitter + Maven/Gradle analysis | **70-80%** | Package imports, Spring Boot annotations (`@RestController`, `@Service`), Maven module graph | Reflection (`Class.forName`), runtime DI wiring, annotation processors, Kotlin DSL configs |
| **TypeScript/JS** | tree-sitter + tsconfig.json/package.json resolution | **65-75%** | ES module imports, React/Next.js/NestJS/Express patterns, workspace dependencies | Path aliases (`@/`), barrel re-exports, dynamic `import()`, CommonJS require with variables, Webpack module federation |
| **SQL** | Custom SQL parser (DDL/DML) | **80-90%** (schema), **50-60%** (queries) | Table/view definitions, foreign keys, migration files (Flyway, Alembic, Prisma), stored procedures | Dynamic SQL, ORM-generated queries, cross-database references, database-specific dialects |

#### SQL: Special Case — Data Architecture Analysis

SQL is not analyzed for import graphs. Instead, it provides **data architecture** signals:

```go
type SQLAnalysis struct {
    Tables       []TableDef       `json:"tables"`        // table name, columns, types
    ForeignKeys  []ForeignKey     `json:"foreign_keys"`  // inter-table relationships
    Views        []ViewDef        `json:"views"`         // materialized and regular
    Migrations   []Migration      `json:"migrations"`    // ordered schema evolution
    StoredProcs  []StoredProc     `json:"stored_procs"`  // business logic in DB
    Indexes      []IndexDef       `json:"indexes"`       // performance hints
    DataDomains  []DataDomain     `json:"data_domains"`  // inferred bounded contexts from table clusters
}
```

**SQL signals feed into:**
- **C4 L2:** Each database/schema = a Container
- **C4 edges:** Foreign keys and cross-schema queries = Relationships between containers
- **Contracts:** Table schemas = data contracts (observed state)
- **PII detection:** Column names matching PII patterns (`email`, `ssn`, `phone`, `address`, `birth_date`) flagged automatically
- **Migration path:** Migration file history shows schema evolution and breaking changes

**SQL sources detected:**
- Migration directories: `migrations/`, `db/migrate/`, `alembic/versions/`
- Schema files: `schema.sql`, `schema.prisma`, `*.dbml`
- ORM models: SQLAlchemy models, Django models, GORM structs, JPA entities, Prisma schema
- Stored procedures: `*.sql` in `procedures/`, `functions/`

#### JS/TS Framework Detection

The TypeScript/JS extractor recognizes major frameworks and extracts architecture-relevant signals:

| Framework | Detection Signal | Architectural Extraction |
|-----------|-----------------|--------------------------|
| **React/Next.js** | `next.config.*`, `pages/` or `app/` directory, `package.json` deps | Page routes as API surface, SSR/SSG classification |
| **NestJS** | `@Module`, `@Controller`, `@Injectable` decorators | Module graph, controller routes, dependency injection tree |
| **Express/Fastify** | `app.get/post/put`, `fastify.register` | Route table extraction, middleware chain |
| **Vue/Nuxt** | `nuxt.config.*`, `.vue` files | Page routes, composables as components |
| **Angular** | `angular.json`, `@NgModule` | Module dependency graph, service injection |

#### Additional Languages (by demand, Phase D+)

| Language | Parser | Expected Accuracy | Trigger to add |
|----------|--------|-------------------|---------------|
| Rust | tree-sitter-rust | 70-80% | Customer with Rust monorepo |
| C# | tree-sitter-c-sharp | 70-80% | Enterprise .NET customer |
| PHP | tree-sitter-php | 50-60% | Laravel/Symfony project |
| Ruby | tree-sitter-ruby | 50-60% | Rails project |
| C/C++ | tree-sitter-c/cpp | 40-50% | Embedded/systems project |
| Kotlin (standalone) | tree-sitter-kotlin | 70-80% | KMP/Compose project |

#### Cross-Language Dependency Detection

In polyglot repos, the tool detects cross-language boundaries:

| Pattern | Detection | Example |
|---------|-----------|---------|
| API client/server | OpenAPI spec shared between services | Go backend serves spec → TS frontend consumes |
| Protobuf/gRPC | `.proto` files referenced from multiple languages | Shared proto → Go server + Python client |
| Database schema | ORM models in app code ↔ migration SQL files | Django models ↔ Alembic migrations |
| Message contracts | AsyncAPI or event schema shared across services | Java producer → Python consumer via Kafka |
| Shared types | JSON Schema referenced from multiple codebases | Schema in `shared/` → validators in Go + TS |

### 1.6 Generated Code Detection

Exclude from analysis to avoid false dependencies:

```go
var generatedCodePatterns = []string{
    "// Code generated by",     // Go standard
    "// DO NOT EDIT",
    "@Generated",               // Java
    "# Generated by",           // Python
    "/* eslint-disable */",     // JS/TS auto-generated
    "__generated__/",           // GraphQL codegen
    "*.pb.go", "*.pb.ts",      // Protobuf
}
```

Files matching patterns are tagged `generated: true` in the profile and excluded from import graph / conformance analysis.

---

## 2. Universal C4 Generation

### 2.1 C4 by Runtime Reality

| C4 Level | Source | Confidence | Human Input Needed |
|----------|--------|-----------|-------------------|
| **L1 System Context** | Infra extractor (Docker, Terraform, CI, sanitized env refs) | High | External actor descriptions |
| **L2 Container** | Deploy units (docker-compose, k8s, workspace members) | High | Business purpose per container |
| **L3 Component** | Import graph clusters + naming heuristics | Medium | Business context, logical groupings, responsibility descriptions |
| **L4 Code** | AST on demand | Medium | Only generated when user requests specific component detail |

**L3 honesty:** The tool generates a **technical component diagram** based on code structure. It explicitly marks components needing human enrichment:

```yaml
components:
  - id: "internal/auth"
    auto_label: "auth"
    description: "[AUTO] Package with 12 exported types, 3 HTTP handlers"
    human_description: null  # TODO: what business capability does this provide?
    confidence: 0.7
```

### 2.2 Performance SLAs (honest)

| Repo Size | Extraction | LLM Interpretation | Total |
|-----------|-----------|-------------------|-------|
| Small (<1K files) | <10s | ~15s | **<30s** |
| Medium (<10K files) | <60s | ~15s | **<2 min** |
| Large (<50K files) | <5 min | ~30s (tiered) | **<6 min** |
| Very large (>50K files) | Incremental only | Tiered RAG | **Variable** |

### 2.3 Mermaid Output with Layout Hints

Auto-generated Mermaid with explicit layout direction and grouping:

```
{target-repo}/.sdp/architecture/c4/
  level-1-context.mmd
  level-2-containers.mmd
  level-3-{container}.mmd
```

Known limitation: auto-layout produces suboptimal results for >15 nodes. For complex systems, the tool outputs a graph data file alongside Mermaid, compatible with manual layout in Excalidraw/draw.io.

---

## 3. Contract System

### 3.1 Three Contract States (unchanged — council validated)

```
observed → proposed → reference
```

| State | Source | Gates code? | Confidence required |
|-------|--------|-------------|-------------------|
| `observed` | Auto-discovery from existing specs | No | Any |
| `proposed` | AI suggestion or human draft | No | >0.7 |
| `reference` | Human-approved | **Yes** | 1.0 (explicit approval) |

### 3.2 MVP: Spec Discovery Only (Layer 1)

Contract inference from code (Layer 2) is **deferred** — council rated it 10x harder than estimated.

**MVP scope:**
- Discover existing OpenAPI, AsyncAPI, Protobuf, GraphQL, JSON Schema
- Index in `contract-catalog.json` with provider/consumer mapping
- Validate specs against each other (cross-service compatibility)

**Deferred to Phase C+:**
- Infer contracts from HTTP handlers, message producers, ORM models
- Per-framework plugins (Express, FastAPI, Spring, Gin)

### 3.3 Contracts as C4 Edges (unchanged)

Every discovered contract = labeled edge in C4 diagrams.

---

## 4. Codebase Partitioning

### 4.1 Hierarchical with Semantic Invalidation

Partitions: Deploy Unit → Module → File (unchanged).

**Cache invalidation fix** (from council critique): Content hash alone is insufficient. Use **Merkle tree of import dependencies**:

```
If File B changes its exported API:
  → B's content hash changes
  → All partitions importing B have their dependency hash invalidated
  → Those partitions are re-analyzed even if their own content is unchanged
```

### 4.2 Monorepo Support

For large monorepos, `.sdp/architecture/` is **distributed per bounded context**:

```
monorepo/
  services/auth/.sdp/architecture/    # auth context
  services/orders/.sdp/architecture/  # orders context
  .sdp/architecture/                  # system-level (L1, cross-context)
```

This avoids merge conflicts on a single root manifest.

---

## 5. Greenfield Conversation

### 5.1 Iterative Pipeline (not strictly linear)

Based on council critique, the pipeline supports **back-tracking**:

```
[Complexity Triage]  ← 3 questions → Simple / Medium / Complex
        |
        v
[Elicit]  ← requirements, quality attributes, constraints
        |
        v  (can loop back to Elicit)
[Classify]  ← architecture candidates from decision matrix
        |
        v  (can loop back to Classify)
[Tradeoff]  ← 2-3 options with explicit tradeoffs and assumptions
        |
        v  (can loop back to any previous phase)
[Decide]  ← human selects, AI generates ADR documenting WHY
        |
        v
[Scaffold]  ← C4 model, contract stubs, conformance rules
```

Each phase saves state to `.sdp/architecture/conversation-state.json` so the user can resume, revise, or restart any phase.

### 5.2 Decision Matrix (data-driven, not cargo cult)

The matrix is a **starting point for discussion**, not a prescription. Stored in `packs/architect/architecture-decision-matrix.yaml`. The AI presents it as: "Based on your constraints, here are candidates — which tradeoffs matter most to you?"

No deterministic mapping like ">10 devs = microservices." Instead, weighted scoring across quality attributes.

---

## 6. Adoption Model: probe → catalog → govern

Replaces the previous 4-level maturity ladder (council critique: clearer, more actionable).

| Mode | What happens | Prerequisites |
|------|-------------|---------------|
| **probe** | One-shot analysis. Produces report + hypothesis + C4 draft. No persistent state. | None — works on any repo immediately |
| **catalog** | Persistent architecture model. `.sdp/architecture/` committed to repo. Contracts cataloged. Team refines model over time. | Human reviews and approves initial model |
| **govern** | Conformance rules enforced in CI. Only `reference` contracts/rules gate code. Findings tracked. | Evaluation harness shows <5% false positive rate on this repo |

**"AI Native" is removed** — council consensus: no trust/evaluation infrastructure exists to support autonomous architecture maintenance. This can be revisited when govern mode has 6+ months of data.

---

## 7. SDP Integration

### 7.1 Augmentation Pack (unchanged — council validated)

```
packs/architect/
  architect.pack
  architecture-decision-matrix.yaml
```

### 7.2 Invocation (unchanged)

| Trigger | Who | What |
|---------|-----|------|
| `@architect analyze <path>` | user | Full analysis (probe mode) |
| `@architect conform` | user/CI | Conformance check (govern mode) |
| `@architect greenfield` | user | Guided conversation |
| `@feature` on structural work | feature agent | Auto-invoke if >3 modules touched |
| `@review` on boundary changes | reviewer agent | Conformance re-check |

### 7.3 New Schemas (registered in `sdp/schema/index.json`)

| Schema | Purpose |
|--------|---------|
| `architecture-model.schema.json` | Reference model validation |
| `architecture-analysis.schema.json` | Analysis report (hypothesis + patterns + risks) |
| `contract-catalog.schema.json` | Contract registry |
| `architecture-conformance.schema.json` | Conformance verdict |
| `architecture-findings.schema.json` | Findings (compatible with protocol-findings) |

### 7.4 Backstage Compatibility

The architecture model supports export to Backstage `catalog-info.yaml` format:

```bash
sdp architect export --format backstage <repo-path>
```

This generates `catalog-info.yaml` per component, making SDP-analyzed repos discoverable in Backstage developer portals.

---

## 8. Architecture Conformance

### 8.1 Rule-Based (data-driven)

Rules in `.sdp/architecture/conformance-rules.yaml` (unchanged).

**New: governance UX** (from Codex critique):
- Rules have `owner`, `expires_at`, `waiver` fields
- `sdp architect waiver add <rule-id> --reason "migration in progress" --expires 2026-06-01`
- Audit log of all rule changes, waivers, and overrides

### 8.2 Limitations (explicitly documented)

Rules catch **structural** violations only:
- Dependency direction violations
- Forbidden imports across boundaries
- Naming convention breaks

Rules **cannot** catch:
- Wrong use of a pattern (e.g., leaky repository abstraction)
- Incorrect error handling strategy
- Missing retry/idempotency logic
- Performance anti-patterns (N+1, sync chokepoints)
- Security anti-patterns beyond dependency boundaries

These require LLM semantic check (Phase C+) or integration with SAST tools.

### 8.3 Evaluation Harness (required before any CI gate)

Before `govern` mode is enabled:
- Run conformance on **50+ golden repos** with known architecture
- Measure precision/recall: must achieve **>95% precision** (low false positives)
- False positive budget: **<5%** — a single false block erodes all trust
- Track metrics over time: `conformance_precision`, `conformance_recall`, `false_positive_rate`

---

## 9. Future Capabilities (not in MVP)

Documented as future directions, **not commitments**:

| Capability | When | Depends on |
|------------|------|-----------|
| Data flow / PII tracking | Phase C+ | Taint analysis via tree-sitter |
| Blast radius estimation | Phase C+ | Contract catalog + dependency graph |
| Conway's law / team topology | Phase B+ | CODEOWNERS + git ownership analysis |
| Technical debt quantification | Phase C+ | Integration with CodeScene/SonarQube APIs |
| Migration path analysis | Phase D+ | Current vs target architecture diff |
| API versioning strategy | Phase B+ | OpenAPI spec diffing |
| Observability architecture | Phase C+ | Library import detection |
| LLM semantic conformance | Phase C+ | ADR corpus of 5+, evaluation harness |

---

## 10. Implementation Priority

### Phase A: Core Engine + 5 Ecosystems (8-10 weeks)

**Goal:** `sdp architect analyze <repo>` produces useful output for Go, Python, Java, TS/JS, and SQL projects.

**Infrastructure (weeks 1-3):**
1. SecurityFilter (secret detection, PII scrubbing, local LLM default)
2. FileTree + DependencyManifest + SpecInventory extractors (language-agnostic)
3. InfraExtractor (Docker, k8s, Terraform, CI)
4. GeneratedCodeDetector
5. CodebaseProfile assembly (Tier 1 + Tier 2)
6. Evaluation harness framework

**Language extractors (weeks 3-6, parallel):**
7. Go: `go/packages` import graph + module analysis
8. Python: tree-sitter + requirements/pyproject + Flask/FastAPI/Django pattern detection
9. Java/Kotlin: tree-sitter + Maven/Gradle + Spring Boot annotation extraction
10. TypeScript/JS: tree-sitter + tsconfig resolution + React/NestJS/Express patterns
11. SQL: DDL parser + migration file analysis + ORM model correlation + PII column detection

**LLM + Output (weeks 6-8):**
12. LLM hypothesis generation (style profile + patterns + risks)
13. Mermaid C4 L1/L2 from infra signals
14. C4 L3 from import graph clusters (all 5 languages)
15. `sdp architect analyze <path>` CLI
16. Cross-language dependency detection

**Evaluation (weeks 8-10):**
17. Golden repo suite: 10 repos per ecosystem (50 total)
18. Precision/recall measurement per language
19. Documentation of known limitations per ecosystem

**Exit criteria:**
- Go: >90% import accuracy, >85% style hypothesis precision
- Python/Java: >65% import accuracy, >75% style hypothesis precision
- TS/JS: >65% import accuracy, >70% style hypothesis precision
- SQL: >80% schema extraction accuracy
- Useful C4 L1/L2 for any repo with Docker/k8s

### Phase B: Contracts + Conway (6-8 weeks)

20. Contract discovery (existing OpenAPI/Proto/GraphQL/AsyncAPI)
21. Contract catalog with `observed` state
22. SQL schema as data contracts (tables → contract-catalog.json)
23. Conway's law mapping (CODEOWNERS + git shortlog → team topology)
24. `sdp architect contracts <path>` CLI
25. Backstage catalog-info.yaml export
26. Evaluation harness expanded to 80 golden repos

### Phase C: Conformance + Greenfield (8-12 weeks)

27. Conformance rule engine with governance UX (waivers, audit, ownership)
28. Greenfield conversation flow (iterative pipeline)
29. Integration with SDP dispatch/augmentation
30. `@architect` skill
31. Cross-language conformance rules (e.g., "Python service must not call Java service directly")
32. Evaluation harness: precision >95% before any CI gate

### Phase D: Depth (ongoing, demand-driven)

33. Contract inference from code (per-framework plugins — Express, FastAPI, Spring, Gin)
34. Data flow / PII tracking (build on SQL PII detection)
35. Blast radius estimation
36. LLM semantic conformance check
37. Technical debt quantification (integrate CodeScene/SonarQube APIs)
38. Additional languages (Rust, C#, PHP, Ruby) by customer demand

---

## 11. Design Principles

1. **Hypothesize, don't label.** Architecture styles are hypotheses with confidence intervals and evidence. Say "I don't know" when confidence is low.

2. **Security first.** No code leaves the machine without secret scrubbing and explicit LLM opt-in. Local LLM is the default path.

3. **Cover what matters, expand by demand.** MVP ships with 5 ecosystems (Go, Python, Java/Kotlin, TS/JS, SQL) — covers ~85% of enterprise projects. Each additional language is a deliberate investment with documented accuracy.

4. **Extract, don't declare.** Everything derivable from code is extracted. Humans curate only what machines cannot infer.

5. **observed → proposed → reference.** No premature enforcement. Earn trust through accuracy before gating code.

6. **probe → catalog → govern.** Adoption is incremental. Each mode has explicit prerequisites.

7. **Artifacts live in target repo.** `.sdp/architecture/` is the home. Distributed per bounded context in monorepos.

8. **Six agents, no exceptions.** AI Architect is an augmentation pack.

9. **Runtime reality over source structure.** C4 containers are deploy units, not directories.

10. **Honest about limitations.** Document what can't be detected. Mark low-confidence results. Never guess authoritatively.

11. **Evaluation before enforcement.** No CI gate ships without golden repo benchmarks and a false positive budget <5%.

12. **Steal shamelessly.** Adopt Backstage entity model, CodeScene coupling algorithms, ArchUnit rule patterns. Don't reinvent what exists.
