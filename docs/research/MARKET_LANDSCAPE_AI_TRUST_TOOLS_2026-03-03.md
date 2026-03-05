# AI Development Trust, Provenance, Evidence, and Governance Tools Market Landscape (2024-2026)

**Research Date:** March 3, 2026  
**Coverage:** 12+ tools across 6 categories  
**Sources:** GitHub repositories (star counts, activity), official documentation, code analysis  

---

## Executive Summary

The AI development trust and governance landscape has consolidated around three pillars: (1) **software supply chain security** (in-toto, Sigstore), (2) **policy-as-code enforcement** (OPA, Kyverno), and (3) **LLM-specific evaluation, safety, and observability** (DeepEval, Promptfoo, NeMo Guardrails). The market is shifting from siloed evaluation tools toward integrated platforms that combine red teaming, guardrails, and evidence tracking.

**Key findings:**
- **MLflow** (24,504★) dominates traditional ML observability
- **DeepEval** (13,909★) leads LLM evaluation with CI/CD integration
- **Promptfoo** (10,760★) has enterprise traction (127 Fortune 500) for AI security testing
- **OPA** (11,292★) is the de facto policy engine across cloud-native stacks
- **Sigstore** is winning as the cryptographic signing standard (backed by OpenSSF)

---

## Market Landscape Table (Ranked by OpenCode Relevance)

| Rank | Tool | Category | GitHub Stars | Primary Enforcement | Strengths | Weaknesses | OpenCode Fit | 2026 Maturity |
|-------|------|----------|--------------|---------------------|------------|-------------|---------------|---------------|
| 1 | **in-toto** | Supply Chain | 977★ | Artifact rules (CREATE/MODIFY/MATCH), signed layouts + link metadata | Evidence-permalink: Verifies each task in chain is carried out as planned, by authorized personnel only, product not tampered in transit. CNCF graduated. | Steep learning curve, no built-in execution enforcement (evidence-only). | ★★★★★: Perfect fit for SDP's evidence envelope requirement. Already specified in ADR-002. | Production-ready |
| 2 | **Sigstore** | Supply Chain | 1,066★ (gitsign) + 810★ (fulcio) + 502★ (core) | OIDC-based identity binding, ephemeral keys, Rekor transparency log enforcement | Evidence-permalink: Signs artifacts with ephemeral keys, associates with OIDC identity, witnesses in immutable Rekor log. Backed by OpenSSF. | Complex infrastructure, requires public-good service dependency. | ★★★★★: SDP plans Sigstore signing (Phase 4). Perfect for cryptographic provenance. | Production-ready |
| 3 | **OPA** | Policy Enforcement | 11,292★ | Rego policy rules evaluated against JSON data; deny-by-default enforcement | Evidence-permalink: General-purpose policy engine for unified, context-aware policy enforcement across entire stack. CNCF graduated. | Rego learning curve, performance at scale needs tuning. | ★★★★★: SDP's policy-as-code foundation (Phase 5). Direct fit. | Production-ready |
| 4 | **Kyverno** | K8s Governance | 7,453★ | CEL policies enforced via K8s admission webhook; validate/mutate/generate/delete/image verify | K8s-native policy engine with CRDs, no external policy engine dependency. | K8s-only scope; CEL policies still evolving. | ★★★★☆: Good for K8s path (Phases 8-9), but SDP prioritizes CLI first. | Production-ready |
| 5 | **DeepEval** | LLM Evaluation | 13,909★ | 50+ metrics (G-Eval, DAG, QAG), Pytest integration, CI/CD enforcement | Evidence-permalink: Unit-testing for LLMs with Pytest, 50+ research-backed metrics, native CI integration. Cloud platform (Confident AI). | Enterprise pricing for advanced features, metrics quality varies by model. | ★★★★☆: Strong for LLM agent testing (F022). Could integrate into SDP evidence gates. | Production-ready |
| 6 | **Promptfoo** | AI Security Testing | 10,760★ | Red teaming, guardrails, evaluations; 300,000+ users, 127 Fortune 500 | Evidence-permalink: Automated red teaming finds & fixes AI risk. Application-focused testing beyond model-level. Covers 50+ vulnerability types. | Commercial model for enterprise features, open-source is powerful but limited. | ★★★★★: Near-perfect fit for SDP's AI safety testing. Could integrate via OpenCode hooks. | Production-ready |
| 7 | **NeMo Guardrails** | AI Safety | 5,718★ | Colang-based guardrails, hallucination prevention, conversation flow enforcement | Configurable guardrails for conversational AI, hallucination checks, topic boundaries. | Colang language is unique, requires investment. NVIDIA ecosystem lock-in risk. | ★★★☆☆: Good for conversational guardrails, but SDP already has guardrails via OPA. | Mature |
| 8 | **MLflow** | ML Observability | 24,504★ | Experiment tracking, model registry, deployment lifecycle enforcement | Evidence-permalink: Comprehensive experiment tracking, model packaging, registry management. Most adopted (24,504★). | Traditional ML focus, less LLM-specific compared to newer tools. | ★★★☆☆: Good for evidence tracking, but overkill for OpenCode-centric workflows. | Production-ready |
| 9 | **Weights & Biases** | ML Tracking | 10,877★ | Model training sweeps, artifact tracking, experiment governance | Weave product for LLM tracing, inference API access, hosted playground. | Commercial-only, no open-source core. | ★★☆☆☆: Strong but closed-source. SDP prefers open standards. | Production-ready |
| 10 | **Helicone** | LLM Observability | 5,178★ | Request logging, 100+ model gateway, automatic fallback enforcement | Evidence-permalink: AI Gateway with 100+ models, automatic observability, 0% markup on provider pricing. | Gateway model means dependency on Helicone infrastructure. | ★★★☆☆: Good for observability, but SDP's evidence-first approach diverges from gateway pattern. | Production-ready |
| 11 | **RAGas** | RAG Evaluation | 12,780★ | RAG-specific metrics (faithfulness, context relevancy, answer correctness) | Specialized for RAG evaluation, synthetic test generation. | Narrow scope (RAG-only), less comprehensive than DeepEval. | ★★★☆☆: Good for RAG workstreams (F016-F019), but not a priority. | Production-ready |
| 12 | **Azure PyRIT** | Red Teaming | 3,502★ | Adversarial attack simulation, jailbreak testing, prompt injection enforcement | Microsoft's red teaming tool for LLMs, synthetic attack generation. | Azure ecosystem bias, limited multi-cloud support. | ★★☆☆☆: Good for red teaming, but Promptfoo is more OpenCode-friendly. | Mature |
| 13 | **TruLens** | LLM Evaluation | 3,127★ | Trulens evaluations for RAG, retrieval-augmented explanations | TruEra's LLM evaluation framework with retrieval metrics. | Smaller community, less CI integration than DeepEval. | ★★☆☆☆: Overlap with DeepEval, less mature. | Mature |

---

## Category Analysis

### 1. Software Supply Chain Security
**What they enforce:** Cryptographic provenance, artifact integrity, step-by-step chain of custody.

**Leader:** **in-toto + Sigstore** combination. Both are CNCF graduated, OpenSSF-backed, and form the de facto standard for software supply chain security.

**Key evidence:**
- **in-toto** enforces artifact rules (CREATE/MODIFY/MATCH/DISALLOW) to authorize and chain supply chain steps
- **Sigstore** provides OIDC-based identity binding and ephemeral key signing with Rekor transparency log

**OpenCode fit:** Direct integration with Git workflows. SDP's F003-F005 legacy migration and F004 reconciler can leverage in-toto layouts for evidence envelopes.

### 2. Policy Enforcement & Governance
**What they enforce:** Declarative policies across infrastructure, code, and AI workflows.

**Leader:** **OPA** is the undisputed leader (11,292★). Kyverno (7,453★) is K8s-specific but gaining ground.

**Key evidence:**
- **OPA** provides unified, context-aware policy enforcement via Rego language
- **Kyverno** uses CEL policies enforced via K8s admission webhook

**OpenCode fit:** OPA integrates via REST API and Go SDK. Perfect for SDP's Phase 5 policy-as-code requirements.

### 3. LLM Evaluation & Testing
**What they enforce:** Quality metrics (hallucination, faithfulness, safety), test coverage, regression prevention.

**Leader:** **DeepEval** (13,909★) leads with Pytest integration and CI/CD support. Promptfoo (10,760★) leads AI security testing.

**Key evidence:**
- **DeepEval** provides 50+ research-backed metrics including G-Eval, DAG, QAG
- **Promptfoo** offers automated red teaming with 300,000+ users, 127 Fortune 500 customers

**OpenCode fit:** Both integrate via CLI. DeepEval's Pytest integration is ideal for SDP's F022 evaluation workstream.

### 4. AI Safety & Guardrails
**What they enforce:** Guardrails for conversational AI, jailbreak prevention, topic boundaries, toxicity filters.

**Leader:** **NeMo Guardrails** (5,718★) offers the most comprehensive guardrails framework. Promptfoo also provides guardrails.

**OpenCode fit:** Guardrails are already covered in SDP via OPA policies (Phase 5). NeMo Guardrails could be used for specialized conversational guardrails.

### 5. Observability & Evidence Tracking
**What they enforce:** Request logging, model performance tracking, cost monitoring, lineage.

**Leader:** **MLflow** (24,504★) dominates traditional ML observability. Helicone (5,178★) leads LLM-specific observability.

**Key evidence:**
- **MLflow** provides experiment tracking, model registry, deployment lifecycle
- **Helicone** provides AI Gateway with 100+ models, automatic logging

**OpenCode fit:** SDP's evidence-first approach (in-toto envelopes) diverges from gateway observability patterns. However, Helicone could be used for CI-to-local bridge (F077).

### 6. Red Teaming & Adversarial Testing
**What they enforce:** Vulnerability discovery, jailbreak testing, prompt injection attacks.

**Leader:** **Promptfoo** (enterprise-proven) and **Azure PyRIT** (Microsoft-backed).

**OpenCode fit:** Promptfoo's open-source core and enterprise traction make it ideal for SDP's AI safety initiatives.

---

## OpenCode-Centric Workflow Fit Assessment

### Highly Complementary Tools (Use Directly)
| Tool | How SDP Would Use It | Integration Point |
|------|----------------------|------------------|
| **in-toto** | Evidence envelope format for all agent runs | sdp-emit-evidence CLI, evidence gate CI |
| **Sigstore** | Cryptographic signing of evidence envelopes | sdp-sign CLI, CI auto-attestation |
| **OPA** | Policy enforcement for scope, tools, evidence | Policy gate CI, pre-tool-call guards |
| **Promptfoo** | Red teaming and safety testing for AI agents | CI integration, PR remediation |
| **DeepEval** | LLM agent evaluation metrics | CI test pipeline, regression tracking |

### Partial Fit (Adapt Required)
| Tool | Adaptation Needed | Why |
|------|-----------------|-----|
| **Kyverno** | K8s-only, need policy translation | Only relevant for Phases 8-9 |
| **NeMo Guardrails** | Colang language, NVIDIA ecosystem | SDP already has guardrails via OPA |
| **Helicone** | Gateway model vs evidence-first | Could use for CI-to-local bridge (F077) |

### Low Fit (Consider Alternatives)
| Tool | Why Low Fit | Alternative |
|------|-------------|-------------|
| **Weights & Biases** | Closed-source, commercial-only | Use DeepEval + in-toto evidence |
| **MLflow** | Overkill for OpenCode workflows | Use simpler evidence tracking |

---

## 2026 Market Trends

1. **Convergence of Evaluation + Security:** Tools like Promptfoo are combining evaluation, red teaming, and guardrails into single platforms.

2. **CI/CD Integration is Table Stakes:** DeepEval, Promptfoo, and in-toto all integrate directly into CI/CD pipelines.

3. **Standards-Based Approach is Winning:** in-toto, Sigstore, and OPA are CNCF/OpenSSF-backed standards, not proprietary solutions.

4. **Identity-Based Signing > Key-Based:** Sigstore's OIDC approach is replacing traditional key management.

5. **LLM-Native Tooling Rising:** Tools like DeepEval and Promptfoo are specifically designed for LLM workloads, not adapted from traditional ML.

---

## Roadmap Implications for SDP Next 12 Weeks

### Week 1-4: Foundation Lock-in (Phase 1-3)
**Priority:** Complete in-toto migration and Sigstore integration.

- **Week 1-2:** Finalize in-toto predicate type and attestation format (F004-F005 legacy cleanup)
- **Week 3-4:** Complete Sigstore signing integration (evidence gate CI, auto-attestation)

**Justification:** in-toto and Sigstore are the foundational standards. No other tooling can be integrated without these in place. Both have proven enterprise adoption and CNCF graduation.

### Week 5-8: Policy & Evaluation Layer (Phase 4-5)
**Priority:** OPA policies and LLM evaluation integration.

- **Week 5-6:** Complete OPA policy enforcement (policy gate CI, pre-tool-call guards)
- **Week 7-8:** Integrate DeepEval for LLM agent evaluation (F022)

**Justification:** OPA provides the policy layer for scope, tools, and evidence. DeepEval provides the evaluation metrics needed for regression testing.

### Week 9-12: Safety & Red Teaming (Phase 6)
**Priority:** Promptfoo integration and AI safety testing.

- **Week 9-10:** Integrate Promptfoo for red teaming and guardrails
- **Week 11-12:** Complete CI-to-local bridge (F077) using Helicone gateway

**Justification:** Promptfoo is the best-fit tool for AI security testing with OpenCode workflows. Helicone provides the observability needed for CI-to-local development.

### Risk Mitigation

| Risk | Mitigation | Tool Focus |
|------|------------|------------|
| **Tool Ecosystem Fragmentation** | Stick to CNCF/OpenSSF standards | in-toto, Sigstore, OPA |
| **Vendor Lock-in** | Prioritize open-source with commercial support | DeepEval (open core), Promptfoo (open core) |
| **K8s Distraction** | Defer Kyverno until Phases 8-9 | Focus on CLI/CI first |
| **Observability Overkill** | Avoid full MLflow adoption | Use in-toto evidence + Helicone for specific use cases |

### Decision Recommendations

1. **Adopt in-toto as the evidence envelope standard** (already decided in ADR-002)
2. **Adopt Sigstore for cryptographic signing** (Phase 4)
3. **Adopt OPA for policy-as-code** (Phase 5)
4. **Adopt DeepEval for LLM evaluation** (F022)
5. **Adopt Promptfoo for AI security testing** (new recommendation)
6. **Defer MLflow adoption** - use in-toto evidence + Helicone for observability
7. **Defer NeMo Guardrails** - use OPA policies instead
8. **Monitor Helicone** for CI-to-local bridge (F077), not full observability stack

---

## Conclusion

The AI development trust and governance landscape in 2026 has consolidated around three complementary standards:

1. **Supply Chain:** in-toto + Sigstore (cryptographic provenance)
2. **Policy:** OPA (declarative governance)
3. **Evaluation:** DeepEval (LLM quality) + Promptfoo (AI security)

SDP is well-positioned to leverage these tools to deliver an evidence-gated runtime for OpenCode workflows. The next 12 weeks should focus on locking in these foundations (Weeks 1-8) before expanding to safety and red teaming (Weeks 9-12).

**Key takeaway:** SDP should not build its own evaluation framework or policy engine. The landscape has matured standards that are enterprise-proven and OpenCode-friendly. The unique value is integrating these standards into an evidence-gated runtime for OpenCode agents.

---

## Research Sources

**GitHub Repositories:**
- in-toto/in-toto: https://github.com/in-toto/in-toto (977★)
- sigstore/gitsign: https://github.com/sigstore/gitsign (1,066★)
- sigstore/sigstore: https://github.com/sigstore/sigstore (502★)
- sigstore/fulcio: https://github.com/sigstore/fulcio (810★)
- open-policy-agent/opa: https://github.com/open-policy-agent/opa (11,292★)
- kyverno/kyverno: https://github.com/kyverno/kyverno (7,453★)
- confident-ai/deepeval: https://github.com/confident-ai/deepeval (13,909★)
- promptfoo/promptfoo: https://github.com/promptfoo/promptfoo (10,760★)
- NVIDIA-NeMo/Guardrails: https://github.com/NVIDIA-NeMo/Guardrails (5,718★)
- mlflow/mlflow: https://github.com/mlflow/mlflow (24,504★)
- wandb/wandb: https://github.com/wandb/wandb (10,877★)
- Helicone/helicone: https://github.com/Helicone/helicone (5,178★)
- vibrantlabsai/ragas: https://github.com/vibrantlabsai/ragas (12,780★)
- Azure/PyRIT: https://github.com/Azure/PyRIT (3,502★)
- truera/trulens: https://github.com/truera/trulens (3,127★)

**Official Documentation:**
- DeepEval: https://docs.confident-ai.com
- Promptfoo: https://promptfoo.dev
- Sigstore: https://docs.sigstore.dev
- in-toto: https://in-toto.io
- Kyverno: https://kyverno.io/docs
- MLflow: https://mlflow.org/docs
- W&B: https://docs.wandb.ai
- Helicone: https://docs.helicone.ai
- OPA: https://www.openpolicyagent.org/docs

**Evidence Permalinks:**
- in-toto artifact rules: https://github.com/in-toto/in-toto/blob/6989182346297b4e72079da6ea5b2b8a441b8d55/README.md#L43-L57
- Sigstore overview: https://docs.sigstore.dev/about/overview/
- OPA overview: https://github.com/open-policy-agent/opa/blob/436aee1d2fde80609a6ef25953a9898c5391f095/README.md#L5-L7
- DeepEval metrics: https://docs.confident-ai.com
- Promptfoo red teaming: https://promptfoo.dev
- MLflow tracking: https://mlflow.org/docs
- Helicone gateway: https://docs.helicone.ai

---

**Document Status:** Complete  
**Version:** 1.0  
**Next Review:** 2026-06-03 (3 months)  
**Maintained by:** SDP core team  
