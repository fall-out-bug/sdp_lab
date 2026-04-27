# R1 — Architect

Model: `google/gemini-3.1-pro-preview`  
Fallback used: `False`  
Elapsed: 35.5s  
OK: `True`

---

**C1: Operator Mode is NOT a separate product layer; it is an advanced Toolkit feature surfaced when a team enables Beads + workstreams + evidence collection.**
*   **VERDICT:** WEAK
*   **EVIDENCE:** Operator Mode introduces stateful orchestration (workstreams, evidence collection, findings loop) which violates the stateless, lightweight nature of a pure Toolkit layer. Bundling heavy orchestration into the Toolkit creates dependency bloat and blurs layer boundaries, making the Toolkit harder to adopt for simple use cases.
*   **PROPOSAL:** Operator Mode MUST be a distinct orchestration layer sitting above the Toolkit, consuming Toolkit primitives but maintaining separate state and dependency graphs.

**C2: Standalone Tools is a first-class new product category — single-purpose extractable utilities (doc-tracer, arch-snap, tok-economy, local-model-router, doc-analyzer) that may eventually live in their own repos.**
*   **VERDICT:** OK
*   **EVIDENCE:** Extracting single-purpose utilities enforces strict dependency boundaries and prevents monolithic entanglement. However, deferring the repo split ("eventually") risks accumulating hidden cross-dependencies; they must be built as isolated packages (e.g., via strict monorepo workspaces) immediately to guarantee extractability.

**C3: The architectural meta-rule is the cascade AGENTS.md ≤60 lines model: every separable surface MUST be cold-startable from root AGENTS.md (≤60) + module AGENTS.md (≤60). Current 606-line root is the migration target.**
*   **VERDICT:** STRONG
*   **EVIDENCE:** Enforcing a ≤60-line limit on AGENTS.md files forces strict bounded contexts and prevents context-window exhaustion for LLM agents. The root + module cold-start requirement guarantees that dependency sub-graphs remain shallow and independently comprehensible, directly enabling the repo topology splits proposed in C2 and C9.

**C4: Two parallel commercial wedges: Wedge A = free dev adoption via Toolkit + selected Standalone Tools on Homebrew; Wedge B = first paid via ChangePassport GitHub PR Gate Loop v1 after Schema v1 lock.**
*   **VERDICT:** OK
*   **EVIDENCE:** This aligns deployment topologies with commercial goals. Wedge A relies on local execution and package managers, while Wedge B requires a distributed, stateful architecture (GitHub App, API gateways). As long as Wedge B strictly consumes Wedge A's shared substrates without circular dependencies, the parallel track is architecturally sound.

**C5: Enterprise Perimeter Control Plane is out of F150 scope; only a reserved slot in the layer model.**
*   **VERDICT:** STRONG
*   **EVIDENCE:** Deferring the Enterprise Perimeter prevents scope creep in F150 while forcing the definition of clear interface boundaries upfront. Reserving the slot ensures the Shared Substrates (C8) are designed to support future on-prem/VPC deployments without requiring a rewrite.

**C6: Russian sovereign model adapters (GigaChat, YandexGPT, MWS, vLLM/NIM/Ollama) are a separate F-track of multiple epics, not part of F150.**
*   **VERDICT:** OK
*   **EVIDENCE:** Isolating sovereign model adapters into a separate track protects F150 from external API volatility and vendor-specific dependency hell. However, F150 must still deliver a strict, provider-agnostic `sdp-modelgw-core` interface to ensure these future adapters can plug in without breaking core layer boundaries.

**C7: ChangePassport is a working name; rename criteria are: domain available, no trademark collision, ICP recognizes the name, council/buyer language test passes. Until then keep `ChangePassport`.**
*   **VERDICT:** WEAK
*   **EVIDENCE:** Deferring the final name creates massive downstream refactoring debt for namespaces, database schemas, API routes, and package registries. Renaming a core domain entity after Schema v1 lock (C9) violates immutable contract principles and risks breaking external integrations.
*   **PROPOSAL:** The internal architectural namespace MUST be locked immediately (e.g., `sdp-pr-gate`) to decouple code/schema topology from future marketing/product name changes.

**C8: Shared Substrates must be explicitly versioned packages (e.g., sdp-evidence-core, sdp-policy-core, sdp-modelgw-core) with semver contracts and deprecation policy — not vague "technical assets".**
*   **VERDICT:** STRONG
*   **EVIDENCE:** Explicitly versioned packages with semver contracts are the only way to safely support parallel wedges (C4) and standalone tools (C2). This prevents the "big ball of mud" anti-pattern and allows independent release cycles for core domain logic versus user-facing surfaces.

**C9: ChangePassport repo split is a downstream event triggered by Schema v1 + Evidence Provider API v1 + Decision Record v1 freeze AND first external pilot landing — not an upfront F150 commitment.**
*   **VERDICT:** OPPOSE
*   **EVIDENCE:** Delaying the repo split until after an external pilot guarantees that rapid prototyping will create undocumented, entangled dependencies between ChangePassport and the core SDP monorepo. The physical repo boundary is the only reliable forcing function