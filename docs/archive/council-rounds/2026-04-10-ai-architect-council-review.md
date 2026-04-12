# AI Architect — Council of Models Review Synthesis

**Date:** 2026-04-10
**Reviewers:** 7 models (GPT-5.4/Codex, Gemini 2.5 Flash, Gemini 3.1 Pro, DeepSeek V3.2, Kimi K2.5, MiniMax M2.7, MiMo V2 Pro)
**Document reviewed:** `docs/plans/2026-04-10-ai-architect-design.md` v2

---

## Overall Verdicts

| Model | Verdict | One-line |
|-------|---------|----------|
| **GPT-5.4 (Codex)** | REVISE | "Over-scoped, under-governed, too confident about cross-language fidelity" |
| **Gemini 2.5 Flash** | REVISE | "Highly ambitious, conceptually beautiful, operationally naive" |
| **Gemini 3.1 Pro** | PIVOT EXECUTION | "Conceptually beautiful, operationally naive. Swap regex for tree-sitter." |
| **DeepSeek V3.2** | REBUILD | "20% brilliant, 80% naive" |
| **Kimi K2.5** | PIVOT SCOPE | "Naive polyglot optimism, LLM maximalism. Build documentation assistant, not detection engine." |
| **MiniMax M2.7** | SHIP+PAUSE | "Ship Phase A immediately. Pause Phase B. Cancel Phase D indefinitely." |
| **MiMo V2 Pro** | REVISE | "Brilliant core insights, over-ambition in language-agnosticism and timeline" |

---

## Consensus Matrix

### UNANIMOUS (7/7 agree)

| Finding | Impact | Action |
|---------|--------|--------|
| **Regex import at ~80% is false** | Critical | Make tree-sitter default, regex fallback only |
| **Timelines are 5-10x underestimated** | Critical | Phase A = 6-8 weeks minimum, not 2 |
| **Security/privacy is catastrophically missing** | Critical | Add PII scrubbing, secret detection, LLM opt-in, local LLM option |
| **Testing strategy is completely absent** | Critical | Golden repos, precision/recall metrics, fuzz testing |
| **Error handling is missing** | Major | Graceful degradation per extractor, LLM hallucination containment |
| **CodebaseProfile bottleneck is brilliant** | Keep | Core innovation, do not change |
| **Contract lifecycle (observed→proposed→reference) is brilliant** | Keep | Best part of the design |
| **Augmentation pack (not agent) is correct** | Keep | Respects 6-agent model |
| **Runtime reality over source structure is correct** | Keep | C4 containers = deploy units |

### STRONG CONSENSUS (5-6/7 agree)

| Finding | Who agrees | Action |
|---------|-----------|--------|
| **Hexagonal/Clean/Onion indistinguishable** (~5-15% confidence) | 6/7 | Collapse into single "ports_and_adapters" |
| **Cache invalidation needs graph-aware invalidation** | 5/7 | Merkle tree of imports, not just content hash |
| **C4 L3 requires human assistance** | 6/7 | Generate technical diagram, prompt user for business context |
| **Contract inference from code is 10x harder** | 6/7 | Scope to spec discovery (Layer 1) only for MVP |
| **Generated code detection is missing** | 5/7 | Add `.sdpignore` + generated file header detection |
| **Rate limiting/cost controls absent** | 5/7 | Token bucket, per-org limits, cost tracking |
| **Greenfield pipeline too linear** | 5/7 | Add iterative refinement, not just forward flow |
| **"Constant LLM cost" claim is misleading** | 5/7 | Needs tiered approach or RAG for cross-partition reasoning |

### UNIQUE INSIGHTS (only 1-2 models raised)

| Insight | Model | Value |
|---------|-------|-------|
| **Architecture Knowledge Model** (Component, API, Resource, Owner, Decision, Boundary) as source of truth | Codex | High — generate C4 FROM the model, not the other way around |
| **3 adoption modes: probe → catalog → govern** | Codex | High — clearer than greenfield/brownfield/assisted/native |
| **"Automated secret-leaking machine"** — extracting API keys and env vars into LLM | Gemini 3.1 Pro | Critical fix needed |
| **Tiered RAG instead of flat compression** | Gemini 3.1 Pro | High — LLM queries graph DB, not compressed summary |
| **"Architecture Hypothesis" not "Architecture Detection"** | Kimi | High — reframe confidence semantics |
| **Go-only MVP first**, expand per customer demand | DeepSeek | Pragmatic — proves value before generalization |
| **Cancel Phase D (AI Native) indefinitely** | MiniMax | Pragmatic — no trust/eval infrastructure for autonomous mode |
| **C4 diagram readability/layout is a hard problem** | MiMo | Medium — auto-layout produces spaghetti |
| **Monorepo: single .sdp = merge-conflict nightmare** | Gemini 3.1 Pro | High — need distributed manifests per directory |
| **Backstage catalog-info.yaml compatibility** | 4/7 | High — don't invent new format, adopt standard |
| **Governance UX missing** (waivers, expiry, audit trail, owner assignment) | Codex | Critical for enterprise adoption |
| **Evaluation harness with golden repos** before any CI gate | Codex, Kimi | Critical — need precision/recall thresholds |

---

## Regex Import Accuracy — Cross-Model Consensus

| Language | Gemini Flash | DeepSeek | Gemini Pro | Kimi | MiMo | MiniMax | **Consensus** |
|----------|-------------|---------|------------|------|------|---------|---------------|
| **Go** | 60-70% | 95% | 95% | 85-90% | 85% | 95% | **85-95%** |
| **Java** | 70-80% | 70% | 85% | 40-50% | 70% | 70% | **60-80%** |
| **TypeScript** | 30-40% | 60% | 20% | 50-60% | 60% | 50% | **30-60%** |
| **Python** | 40-50% | 40% | 40% | 30-40% | 50% | 45% | **35-50%** |
| **Rust** | 50-60% | 50% | 60% | 60-70% | 65% | 65% | **50-65%** |
| **C#** | 70-80% | 65% | — | — | 75% | 60% | **65-75%** |
| **PHP/Ruby** | — | 30% | — | 25-35% | — | 35% | **25-35%** |

**Verdict:** Regex is viable ONLY for Go (85%+). For everything else, tree-sitter or language-native tooling is required.

---

## Architecture Type Detection — Honest Confidence Matrix

| Type | Avg Confidence | Detectable? | Notes |
|------|---------------|-------------|-------|
| `infra_repo` | **93%** | Yes | File type distribution alone |
| `library` | **90%** | Yes | No main, export-focused |
| `monorepo_multi_service` | **88%** | Yes | Multiple build manifests |
| `monolith_layered` | **60%** | Partially | Import hierarchy + naming |
| `serverless` | **55%** | Partially | Needs infra configs |
| `plugin_extension` | **55%** | Partially | Dynamic loading patterns |
| `pipe_and_filter` | **45%** | Low | Confused with functional style |
| `microservices` | **45%** | Low | Could be monolith with sidecars |
| `monolith_modular` | **40%** | Low | "Modular" is intent, not structure |
| `big_ball_of_mud` | **40%** | Needs graph math | Cyclic dependency detection |
| `cqrs_event_sourcing` | **25%** | No | Needs event store + patterns |
| `microservices_event_driven` | **22%** | No | Needs runtime traces |
| `hexagonal` | **12%** | No | Intent-based, not structural |
| `clean_architecture` | **10%** | No | Indistinguishable from hexagonal |
| `onion` | **10%** | No | Same as above |

**Action:** Reduce to 8 reliably detectable types. Others require human confirmation. Rename from "Detection" to "Hypothesis."

---

## Missing Capabilities — Prioritized by Council Votes

| Capability | Votes | Priority |
|------------|-------|----------|
| **Security/secret scrubbing before LLM** | 7/7 | P0 — non-negotiable |
| **Data flow / PII tracking** | 7/7 | P1 |
| **Blast radius estimation** | 6/7 | P1 |
| **Conway's law / team topology** | 6/7 | P1 |
| **Technical debt quantification** | 6/7 | P2 |
| **Migration path analysis** | 5/7 | P2 |
| **API versioning strategy** | 5/7 | P2 |
| **Observability architecture** | 5/7 | P2 |
| **Governance UX (waivers, audit, ownership)** | 3/7 | P1 — critical for enterprise |
| **Evaluation harness / golden repos** | 3/7 | P0 — needed before any gate |
| **Generated code detection** | 5/7 | P1 |
| **Backstage compatibility** | 4/7 | P2 |

---

## Top Implementation Risks — Ranked by Difficulty

| # | Risk | Multiplier | Who flagged |
|---|------|-----------|-------------|
| 1 | **Import graph resolution for dynamic languages** | 10x | All 7 |
| 2 | **Contract inference from code** | 10x | 6/7 |
| 3 | **C4 L3 component clustering** (NP-hard graph partitioning) | 8-10x | 4/7 |
| 4 | **Cross-language conformance at CI speed** | 5x | 4/7 |
| 5 | **Stable identity** (mapping noisy nodes to persistent objects) | 5x | Codex |
| 6 | **Greenfield conversation quality** | 4-5x | 4/7 |
| 7 | **LLM hallucination containment** | 4x | 5/7 |
| 8 | **C4 diagram readability/layout** | 4x | MiMo |

---

## What to Steal — Consolidated

| Tool | Steal What | Votes |
|------|-----------|-------|
| **Backstage** | catalog-info.yaml entity model (Component/API/Resource/Owner) | 6/7 |
| **CodeScene** | Temporal coupling from git, hotspot analysis, code health trends | 6/7 |
| **Structure101** | DSM visualization, current-vs-intended architecture model | 5/7 |
| **ArchUnit** | Fluent rule DSL, test-as-architecture-constraint | 5/7 |
| **Sourcegraph** | LSIF/SCIP code intelligence, batch changes | 4/7 |
| **Snyk** | Dependency risk scoring, license compliance | 4/7 |
| **Lattix** | Multi-domain traceability, DSM delta | 3/7 |
| **NDepend** | Historical comparison, trend charts, baseline diffing | 2/7 |

---

## Strongest Parts (Keep Exactly As-Is)

1. **CodebaseProfile as information bottleneck** — 7/7 called brilliant
2. **observed → proposed → reference contract states** — 7/7 called brilliant
3. **Augmentation pack, not new agent** — 7/7 called correct
4. **Runtime reality over source structure** — 7/7 called correct
5. **"Extract, don't declare" principle** — 5/7 praised

## Weakest Parts (Fix Immediately)

1. **Regex import extraction** — 7/7 called delusional for non-Go
2. **2-week phase timelines** — 7/7 called fantasy (real: 6-8 weeks each)
3. **80% accuracy claim** — 7/7 called false
4. **15 architecture types** — 6/7 said reduce to 8
5. **No security/privacy** — 7/7 called catastrophic
6. **No testing strategy** — 7/7 flagged
7. **AI Native mode** — 4/7 called premature/science fiction
8. **Greenfield decision matrix** (>10 devs = microservices) — 2/7 called "cargo cult"

---

## Recommended Actions

### Immediate (before any code)
1. Add security middleware (secret detection, PII scrubbing, LLM opt-in)
2. Replace regex with tree-sitter as default extraction
3. Reduce architecture types from 15 to 8, rename "Detection" → "Hypothesis"
4. Create evaluation harness with 50+ golden repos
5. Add governance UX (waivers, audit, ownership)
6. Revise timelines: Phase A = 6-8 weeks, Phase B = 8-12 weeks

### Scope Pivot (from the council)
- **From:** "Architecture Detection Engine" for any language
- **To:** "Architecture Documentation Assistant" — helps architects document and enforce what they already know
- **Go-only MVP first**, prove value, then expand by customer demand
- **probe → catalog → govern** adoption model (not greenfield/brownfield/assisted/native)

### Architecture Changes
- Adopt Backstage catalog-info.yaml entity model
- Add Architecture Knowledge Model as source of truth (C4 generated FROM it)
- Implement tiered RAG instead of flat CodebaseProfile compression
- Add Merkle tree import graph for cache invalidation
- Distributed `.sdp/` manifests per directory for monorepos
