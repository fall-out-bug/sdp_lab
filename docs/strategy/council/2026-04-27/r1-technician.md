# R1 — Technician

Model: `moonshotai/kimi-k2.6`  
Fallback used: `False`  
Elapsed: 348.4s  
OK: `True`

---

**C1**
- **VERDICT:** OK
- **EVIDENCE:** Keeping Operator Mode inside the Toolkit monorepo avoids a separate release train and install surface, which aligns with F150's need to limit workstream proliferation. Because Operator Mode currently depends on Beads and workstreams—both internal primitives—surface coupling is acceptable at this stage. The extraction lifecycle is correctly deferred until buyer signal justifies the CI and packaging overhead of a standalone SKU.

**C2**
- **VERDICT:** WEAK
- **EVIDENCE:** Elevating five utilities to a first-class category with eventual repo extraction forces immediate interface stability and parallel CI investment before any standalone tool proves independent utility. Doc-tracer and arch-snap likely share AST parsing logic with Toolkit internals; without a monorepo-first package strategy, the team will fight dependency drift and substrate version skew. The "may eventually" clause creates an ambiguity that avoids explicit extraction criteria.
- **PROPOSAL:** Standalone Tools are a first-class product category but start as strict semver packages inside a single `sdp-tools` monorepo; extraction to separate repos requires two external consumers and a substrate stability review.

**C3**
- **VERDICT:** WEAK
- **EVIDENCE:** A 606-line root implies deep contextual coupling; a ≤60-line target demands dependency inversion and context isolation that does not yet exist. Without an automated CI gate validating line count, cross-module imports, and cold-start independence, the meta-rule will decay the first time a deadline looms. The migration cost competes directly with Schema v1 implementation and risks stalling the release readiness program.
- **PROPOSAL:** The cascade target is ≤60 lines enforced by a CI linting script; migration is an incremental deliverable of 00-150-03 with a 20% line-reduction sprint goal, not a blocking rewrite.

**C4**
- **VERDICT:** OK
- **EVIDENCE:** Sequencing free adoption via Homebrew before paid ChangePassport is mechanically sound and leverages existing workstream 00-150-08. The Schema v1 lock acts as a clean dependency gate, preventing paid-surface API breakage. This requires strict semver discipline on shared substrates so that Toolkit updates do not implicitly bump ChangePassport interfaces.

**C5**
- **VERDICT:** STRONG
- **EVIDENCE:** Explicit exclusion prevents scope creep into high-touch enterprise infrastructure such as SSO, air-gapped networking, and tenant isolation. The reserved slot acknowledges future layering without incurring CI complexity, Helm charts, or compliance documentation during F150. This preserves team capacity for the Schema v1 and Evidence Provider API implementations that actually gate commercial readiness.

**C6**
- **VERDICT:** OK
- **EVIDENCE:** Removing third-party sovereign adapter certification from F150 avoids blocking on unstable external APIs and export-control review. However, the local-model-router standalone tool must expose a strict adapter interface contract with mock providers in CI; otherwise F150 implicitly re-absorbs adapter integration scope through e2e test dependencies. Keeping the adapter track separate is valid only if the router tool is architected against an internal bridge, not a direct SDK dependency.

**C7**
- **VERDICT:** VETO
- **EVIDENCE:** Shipping CLI binaries, Homebrew formulas, package import paths, and GitHub App slugs under a working name that lacks a vendor namespace incurs irreversible migration debt. Rename after Schema v1 freeze forces a semver-breaking transition across all published artifacts. The listed rename criteria are commercial and legal, entirely downstream of F150 release engineering, yet F150 workstreams will bake the un-namespaced slug into infrastructure before the rename review even occurs.
- **PROPOSAL:** All technical artifacts—repos, packages, CLI commands, GitHub App slugs—are prefixed with a stable vendor namespace (e.g., `sdp-changepassport`) immediately; the display name remains `ChangePassport` until brand criteria are met.

**C8**
- **VERDICT:** STRONG
- **EVIDENCE:** This is the minimum viable release-engineering contract for a multi-surface architecture. Semver and deprecation policy prevent silent breaking changes across Toolkit, Standalone Tools, and ChangePassport, and they unblock the Schema v1 freeze by making substrate dependencies explicit. It directly enables the extraction lifecycle claimed in C2 and C9.

**C9**
- **VERDICT:** OK
- **EVIDENCE:** Deferring the split avoids submodule complexity and multiple release trains during rapid Schema v1 churn. The combined trigger of technical freeze plus external pilot carries deadlock risk if pilot feedback demands schema changes. Mitigating this requires enforcing package-level isolation inside the monorepo immediately, ensuring the future split is a mechanical `git filter-repo` operation rather than a cross-import rewrite.

**C10**
- **VERDICT:** WEAK
- **EVIDENCE:** The cascade AGENTS.md migration is structural refactoring requiring CI linting and dependency inversion, not documentation; bundling it into 00-150-09 obscures implementation cost. The Standalone Tools registry is a release-surface deliverable belonging in 00-150-02 or 00-150-08. This bundling risks hiding tooling requirements from the project tracker and overloading a docs workstream with engineering scope that should own its own CI gates.
- **PROPOSAL:** Cascade AGENTS.md migration becomes a bounded subtask of 00-150-03 (module path migration) with CI gate ownership; Standalone Tools registry becomes a deliverable of 00-150-02 (release surface).

**C11**
- **VERDICT:** WEAK
- **EVIDENCE:** "Useful decision rate," "reviewer time -20%," and "hallucination <5%" require baselines, human-labeling infrastructure, and automated eval harnesses that do not currently exist. None of the F150 workstreams explicitly build measurement CI jobs or telemetry schemas for these KPIs. Without pre-pilot instrumentation, these targets are unfalsifiable and will yield noisy signals that destabilize product judgment during the pilot.
- **PROPOSAL:** All ChangePassport metrics require an eval harness and telemetry schema defined in 00-150-07 before pilot launch; pilot-stage targets are ratified only after baseline measurement tooling passes CI.

**C12**
- **VERDICT:** OK
- **EVIDENCE:** Treating Operator Mode as a Toolkit happy path avoids premature SKU extraction, separate Helm charts, and subscription logic before revenue evidence exists. It keeps CI and install surfaces unified, reducing F150 scope. The explicit re-evaluation gate prevents technical deadlock while preserving optionality to fork the surface if telemetry justifies a standalone offering later.

---

**MINORITY REPORT**
- **C3** (cascade AGENTS.md ≤60 lines). I predict the majority will vote STRONG or OK, treating this as elegant architectural hygiene. From a technician's perspective, a 606-line root without existing dependency inversion or CI enforcement means this is a rewrite masquerading as cleanup; the majority will underestimate migration cost and overestimate team discipline under delivery pressure.

**DOMAIN VETOES**
- **C7.** [DOMAIN VETO: Published technical asset names (packages, CLI slugs, GitHub App IDs) are immutable release-engineering identifiers; a working name without a stable vendor namespace creates irreversible semver breakage and import-path migration debt upon any future brand rename.]

**THREE BIGGEST RISKS the document fails to address**

1. **CI Matrix and Artifact Registry Proliferation.** The document envisions Toolkit (Homebrew), Standalone Tools (Homebrew + possible separate repos), and ChangePassport (GitHub App) without specifying a unified CI platform, reusable workflow strategy, or internal artifact registry. F150 release engineering will collapse under parallel tracks without a shared build matrix.

2. **Schema Freeze Collision with Module Path Migration.** Schema v1 freeze gates ChangePassport commercialization, yet F150 workstream 00-150-03 (module path migration) will restructure shared substrates. If module migration causes breaking changes in sdp-evidence-core or sdp-policy-core after the freeze, the semver contracts in C8 and the commercial wedge logic in C4 are violated.

3. **Statefulness and Evidence Storage Architecture.** Operator Mode and ChangePassport both generate evidence artifacts (Beads, findings loop, passports), but the document never specifies where this state lives—git LFS, object storage, MCP server, or local SQLite—nor its retention, privacy, or backup policy. This omission risks a last-minute architectural pivot during pilot onboarding.