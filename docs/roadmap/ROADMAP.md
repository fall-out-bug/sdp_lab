# sdp_dev Roadmap — Standards-Based Trust Layer for AI Coding Agents

> **Updated:** 2026-02-24
> **Direction:** Standards-based enforcement (in-toto, OPA, Sigstore) + autonomous agent pipeline
> **Pivot:** [ADR-002 Standards Pivot](../decisions/ADR-002-standards-pivot.md) — why we moved from custom to standards
> **Research:** [Enforcement Hypotheses](../plans/2026-02-24-enforcement-hypotheses.md) — SLSA, in-toto, MI9, AgentSpec
> **Audit:** [Phase 0 Enforcement Audit](../plans/2026-02-24-phase0-enforcement-audit.md) — why Phase 0 tools didn't enforce
> **K8s Archive:** `archive/k8s-v0` branch — domain knowledge for future K8s rebuild

---

## Overview

Two horizons. Phases 1-7 build the trust layer as CLI tools with CI enforcement. Phases 8-9 extend the same standards into K8s for autonomous swarm execution. The dream (issue in, PR with proof out) doesn't change — the path becomes standards-based.

```mermaid
graph LR
    subgraph done [Done: Agent Loop Reliability]
        F014["F014 CI Loop CLI"]
        F015["F015 Stop Hook"]
        F016["F016 Outer Loop"]
        F018["F018 Dead Code Purge"]
        F019["F019 Skill Compression"]
        F023["F023 Scope Enforcement"]
        F027["F027 CI Auto-Fixers"]
    end
    subgraph p1 [Phase 1: Enforcement Foundation]
        EG["Evidence Gate CI"]
        SG["Scope Gate CI"]
        BP["Branch Protection"]
    end
    subgraph p2 [Phase 2: Archive and Focus]
        ARCH["Archive K8s code"]
        ADR["ADR-002"]
    end
    subgraph p3 [Phase 3: in-toto Migration]
        PRED["Predicate type"]
        ATTEST["Attestation format"]
    end
    subgraph p4 [Phase 4: Auto-Attestation]
        AUTO["CI auto-attestation"]
        SIGN["Sigstore signing"]
    end
    subgraph p5 [Phase 5: Policy-as-Code]
        OPA["OPA/Rego policies"]
        PGATE["Policy gate CI"]
    end
    subgraph p6 [Phase 6: Runtime Governance]
        FSM["FSM conformance"]
        DRIFT["Drift detection"]
    end
    subgraph p7 [Phase 7: Ecosystem Launch]
        CLI["sdp-evidence release"]
        BLOG["OSS launch"]
    end
    subgraph p89 [Phases 8-9: K8s Dream]
        K8S["K8s pipeline v2"]
        KYVERNO["Kyverno admission"]
        TEKTON["Tekton Chains"]
    end

    done --> p1
    p1 --> p2
    p2 --> p3
    p3 --> p4
    p4 --> p5
    p5 --> p6
    p6 --> p7
    p7 --> p89
```

---

## Done: Phase 0 (Agent Loop Reliability)

14 features completed. Established outer-loop/inner-loop architecture. See [full Phase 0 details](../plans/2026-02-23-agent-loop-reliability.md).

**Key outcomes kept:**
- `sdp-orchestrate` — outer loop state machine (build → review → PR → CI → done)
- `sdp-ci-loop` — deterministic CI polling + auto-fix
- `sdp-guard` — scope enforcement via git diff vs declared scope
- Stop hooks, context pre-hydration, prompt consolidation, skill compression

**Lesson learned:** Phase 0 built tools (7% enforcement, 43% cleanup, 50% potential). Tools work when used but nothing ensures they are used. This triggered the standards pivot.

---

## Phase 1: Enforcement Foundation (2-3 sessions)

**Goal:** Wire existing tools into the merge path. No code rewrite — just CI gates and branch protection.

| Change | What | Prevents |
|--------|------|----------|
| `evidence-gate` CI job | Validates `.sdp/evidence/*.json` on PRs | Merging with invalid evidence |
| `scope-gate` CI job | Runs sdp-guard on PR diff | Merging with scope violations |
| Branch protection | Required checks: build-test, evidence-gate | Bypassing CI entirely |
| Remove `--skip-guard` | No escape hatch in sdp-orchestrate | Bypassing scope enforcement |

**Status:** Done

---

## Phase 2: Archive & Focus (2-3 sessions)

**Goal:** Archive K8s/swarm code to `archive/k8s-v0` branch. Master becomes lean CLI-only (~5,500 LOC).

| Action | What |
|--------|------|
| Create `archive/k8s-v0` branch | Preserves all K8s/swarm code |
| Remove 27 binaries from master | swarm-worker, adapter-controller, etc. |
| Remove 33 internal packages | adapter, bus, swarm, policy, etc. |
| Keep 5 binaries + 8 packages | sdp-orchestrate, sdp-ci-loop, sdp-guard, sdp-evidence, sdp-eval |
| Write ADR-002 | Documents the pivot decision |

**Status:** Done

---

## Phase 3: in-toto Migration (3-5 sessions)

**Goal:** Replace custom evidence envelope with in-toto attestation format.

Our 9 sections become the **predicate** in an in-toto statement. The **envelope** (DSSE signing) comes from in-toto/Sigstore. Domain knowledge is preserved; the format becomes standard.

```
in-toto Envelope (DSSE)
  └── Statement
        ├── subject: [{ name: "PR #42", digest: { sha256: "..." } }]
        ├── predicateType: "https://sdp.dev/attestation/coding-workflow/v1"
        └── predicate:
              ├── intent      (issue, trigger, AC, risk)
              ├── plan        (workstreams, ordering)
              ├── execution   (branch, changed_files)
              ├── verification (tests, lint, coverage)
              ├── review      (findings, verdict)
              ├── risk_notes  (residual, excluded)
              ├── boundary    (declared vs observed scope)
              ├── provenance  (run_id, model, prompt_hash)
              └── trace       (beads_ids, commits, pr_url)
```

| Action | What |
|--------|------|
| Define predicate type | `https://sdp.dev/attestation/coding-workflow/v1` |
| Rewrite `internal/evidence/` | Use `in-toto-golang` library |
| Rewrite `sdp-evidence` | Validate/inspect in-toto attestations |
| Delete `internal/artifact/` | Custom hash chain replaced by DSSE signing |

**Go library:** `github.com/in-toto/in-toto-golang`

---

## Phase 4: Auto-Attestation (3-5 sessions)

**Goal:** CI generates attestations automatically from observed facts (Tekton Chains model).

| Action | What |
|--------|------|
| CI observer job | Collects git diff, test results, coverage, lint output |
| Auto-attestation | Generates in-toto attestation from observed facts |
| Sigstore signing | Signs with keyless OIDC (GitHub Actions) |
| Discrepancy detection | Agent attestation vs CI attestation → audit finding |

**Key insight:** Agent cannot "forget" evidence — CI creates it from facts.

---

## Phase 5: Policy-as-Code (3-5 sessions)

**Goal:** Replace markdown policies with executable OPA/Rego.

| Action | What |
|--------|------|
| `.sdp/policies/*.rego` | Declarative policies (evidence-required, scope, coverage, beads) |
| `policy-gate` CI job | `opa eval` against PR metadata |
| Embed OPA in sdp-orchestrate | Runtime policy checks |

**Go library:** `github.com/open-policy-agent/opa/rego`

---

## Phase 6: Runtime Governance (5-10 sessions, research-heavy)

**Goal:** Study MI9/AgentSpec, adopt what works for runtime agent control.

| Research | Source |
|----------|--------|
| FSM conformance engines | MI9 (arXiv 2508.03858) |
| Runtime constraint DSL | AgentSpec (arXiv 2503.18666, ICSE 2026) |
| Goal-conditioned drift detection | MI9 |
| Graduated containment | MI9 (warn → block → halt → escalate) |

---

## Phase 7: Ecosystem & Launch

**Goal:** Ship SDP as an open-source trust layer for AI coding agents.

- Publish predicate type spec in sdp protocol repo
- Release `sdp-evidence` binary (validates in-toto attestations for coding workflows)
- Blog: "What Stripe's Minions Proved — and What's Still Missing"
- awesome-opencode listing

---

## Phase 8: K8s Orchestration Research (5-10 sessions)

**Goal:** Design the K8s pipeline with enforcement built-in from day one.

| Study | Source |
|-------|--------|
| Stripe Minions | Blueprint graph, devbox isolation, Toolshed MCP |
| kubeopencode | Task CRD, AgentRun CRD, reconciler design |
| Tekton Chains | Auto-attestation in K8s (observe TaskRun → provenance) |
| Kyverno | Admission-level policy enforcement |

**Output:** Design doc for K8s pipeline v2 with no bypass paths.

---

## Phase 9: K8s Pipeline Rebuild (10-20 sessions)

**Goal:** The dream — issue in, PR with proof out, on K8s, fully autonomous.

- Minimal K8s components (adapter-controller v2, beads-bridge)
- Built on in-toto attestations (same format as CLI)
- Policies enforced by Kyverno admission controller
- Auto-attestation by K8s observer (Tekton Chains pattern)
- Signed with Sigstore (in-cluster keyless)
- Sequential pipeline: analyst → coder → reviewer
- Target: 10 consecutive runs with full attestation chain

---

## What SDP Does Better Than Anyone

| Others | What they cover | What SDP covers |
|--------|----------------|-----------------|
| SLSA / in-toto | Build artifacts | **Development process** |
| GLACIS / TrustPlane | LLM API calls | **Commits, tests, reviews, PRs** |
| MI9 / AgentSpec | Academic proposals | **Practical CLI tools** |
| Stripe Minions | Internal system | **Open source** |

---

## Key References

### Standards & Tools
- [in-toto attestation](https://github.com/in-toto/attestation) — envelope format
- [OPA/Rego](https://www.openpolicyagent.org/) — policy-as-code
- [Sigstore](https://docs.sigstore.dev/) — keyless signing
- [SLSA](https://slsa.dev/) — supply chain levels
- [Tekton Chains](https://tekton.dev/docs/chains/) — K8s auto-attestation
- [Kyverno](https://kyverno.io/) — K8s admission enforcement

### Research
- MI9 (runtime governance): arXiv 2508.03858
- AgentSpec (runtime constraints): arXiv 2503.18666
- PROV-AGENT (W3C PROV for agents): arXiv 2508.02866
- VET (verifiable execution traces): arXiv 2512.15892

### Ongoing
- **F053** — Phase 4 Beads Remediation (00-053-01 … 00-053-46). See [INDEX](../workstreams/INDEX.md).
- **F054** — Continuous Protocol Improvement (00-054-01 … 00-054-05). Skills, AGENTS.md, workflow. Part of future swarm. Source: [multifaceted analysis](../plans/2026-02-25-agent-protocol-multifaceted-analysis.md).

### Project History
- [Agent Loop Reliability](../plans/2026-02-23-agent-loop-reliability.md)
- [Stripe Minions Comparison](../plans/2026-02-23-stripe-minions-comparison.md)
- [Phase 0 Enforcement Audit](../plans/2026-02-24-phase0-enforcement-audit.md)
- [Enforcement Hypotheses](../plans/2026-02-24-enforcement-hypotheses.md)
- [ADR-002 Standards Pivot](../decisions/ADR-002-standards-pivot.md)
- K8s domain knowledge: `archive/k8s-v0` branch
