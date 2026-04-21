# sdp_lab Roadmap — Agent Platform + Trust Lane

> **Status: CANONICAL (active roadmap)** — this is the current source of truth. Other files in `docs/roadmap/` are supporting context; see [docs/roadmap/README.md](README.md).
> **Updated:** 2026-04-20
> **Repo naming:** GitHub repo is `sdp_lab`; historical workstreams and bead IDs may still use `sdp_dev` as a legacy label for the same codebase
> **Direction:** Reusable agent platform (`kernel`, `adapters`, `augmentation`, `evals`) with standards-based trust and evidence as a secondary lane

Supporting context (read on demand, not required for orientation):

- Platform Reset: [AGENT_PLATFORM_ROADMAP_2026-03-31.md](AGENT_PLATFORM_ROADMAP_2026-03-31.md) + [2026-03-31-platform-backlog-reset.md](../archive/plans/2026-03-31-platform-backlog-reset.md)
- Pivot rationale: [ADR-002 Standards Pivot](../decisions/ADR-002-standards-pivot.md)
- Research: [Enforcement Hypotheses](../archive/plans/2026-02-24-enforcement-hypotheses.md), [Phase 0 Enforcement Audit](../archive/plans/2026-02-24-phase0-enforcement-audit.md)
- K8s Archive: `archive/k8s-v0` branch (domain knowledge for future rebuild)
- Synergies: [ECOSYSTEM_SYNERGIES.md](../integrations/ECOSYSTEM_SYNERGIES.md)
- Unified Plan: [UNIFIED_VISION_ROADMAP_2026-03-03.md](UNIFIED_VISION_ROADMAP_2026-03-03.md)
- Market Loop: [MARKET_INTELLIGENCE_OPERATING_LOOP.md](MARKET_INTELLIGENCE_OPERATING_LOOP.md)
- Consistency Policy: [CONSISTENCY_MITIGATION_POLICY.md](CONSISTENCY_MITIGATION_POLICY.md)
- A* Alignment Stream: [STATE_ALIGNMENT_STREAM_ASTAR.md](STATE_ALIGNMENT_STREAM_ASTAR.md)

---

## Overview

Execution priority has changed.

Current execution priority is the remaining UX/runtime lane:

- `F100` — release discipline gates
- `F106` — real `agentloop` integration into the delivery path
- `F108` — architecture normalization and missing production gaps
- `F125` — finish the intent-routed UX migration and close the remaining cutover tail

Recently shipped on 2026-04-18:

- `F077` — CI to Local Bridge for Improvement Loop
- `F097` — Product Truth and Activation Loop
- `F098` — Simplified Progressive Disclosure
- `F099` — Brownfield Safe Overlay
- `F101` — Write Plan Emission and Confirmation
- `F105` — AI Architect Phase A

`F098` is merged, but post-merge docs/security/SRE follow-up remains tracked in beads. `F100` (#96) is the remaining open P0 UX implementation PR. Shipped status in this roadmap means merged implementation, not zero open findings.

The platform reset lane established the base rather than remaining the active queue:

- `F091`, `F093`, `F094`, `F095` are done
- `F092` remains the only active platform-reset feature
- `F096` remains a blocked support lane, not a primary execution target

Trust, evidence, and enterprise governance remain in the roadmap.
They are now the `trust lane`, not the whole product story.

Analyst-facing evidence products also remain in scope.
`F109` formalizes StratAudit as a trust-sensitive product lane: the report surface is
not cosmetic HTML work, but the final slice on top of verified evidence, provenance,
and multilingual source preservation.

This lane is not an exclusive ready queue. `bd ready` remains the live source of executable work, and older ecosystem tasks can still coexist in Beads until they are explicitly triaged or deprioritized.

Toolkit foundations for brownfield adoption are now shipped, not just planned:

- `F120` scout is merged
- `F121` metrics is merged
- `F122` index is merged
- `F123` spec recovery is merged
- `F124` bootstrap is merged
- `F126` MCP server is merged

The remaining toolkit work is therefore smaller and more concrete:

- `F125` — finish the intent-routed UX migration and close the review-readiness/doc-sweep tail

This is the current next lane after the P0 UX/runtime work and remains the canonical plan for "unknown repo -> AI-native workspace".

A new repo-native collaboration-memory pilot is now shaped and queued:

- `F136` — Peer Memory Foundation (H1). Actor-aware episodic memory for humans + agents with repo-scoped local storage, FTS5 retrieval, CLI writes, agentloop auto-capture, and MCP read resources. H2/H3 stay parked in beads until the 2026-07-06 gate.

A separate follow-on lane now exists on top of that shipped toolkit foundation:

- `F137` — normalize the CLI surface into one machine-readable contract for downstream automation
- `F138` — normalize the skill catalog and harness-facing docs without reopening the shipped `F125` intent model
- `F139` — normalize MCP parity and discovery on top of the shipped `F126` server

This is a delta lane, not a retroactive rewrite of `F125` or `F126`.

Backlog triage 2026-04-18 (active backlog features are tracked in beads for consistency — search with `bd search FNNN`):

- **Closed (historical placeholders):** `F032`, `F034`, `F036`–`F049`, `F052` — closed with supersede reasons pointing to shipped tracks (F064–F067 auto-attestation, F134 Phase FSM, F108 drift).
- **Superseded by active epics:** `F062` → F125/F134, `F063` → F124, `F068` → F091/F092, `F072` → F094/F095 + F134, `F073` → F093. Tracked as closed in beads with `bd supersede` reasons.
- **Compliance lane (EU AI Act 2026-08 / Colorado AI Act 2026-06):** `F074`, `F078`, `F079`, `F080`, `F082` — open P2, all linked to `F134-03` (evidence enforcement) and/or `F134-04` (AI-vs-human attribution).
- **Critical path:** `F081` (30-min production pilot, gating layer for F135 deploy gate). Shipped prerequisite: `F077` (CI→local bridge, merged 2026-04-18) now unblocks F129/F135 self-healing.
- **Reframe pending F134:** `F060` (gastown), `F070` (observability) — open with dependency on F134 evidence schema.
- **Keep long-horizon:** `F075`, `F083`, `F084`, `F085` — open P3, gated on trust-lane completion.

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

14 features completed. Established outer-loop/inner-loop architecture. See [full Phase 0 details](../archive/plans/2026-02-23-agent-loop-reliability.md).

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

## Phase Toolkit: Unknown Codebase -> AI-Native Adoption

**Goal:** make SDP useful the moment it enters an unknown brownfield repo, without
forcing a full operator workflow or a slow architecture pass as the first step.

This lane intentionally overrides two stale assumptions inside the vision doc:

- first shipped value is `sdp scout`, not `metrics` or `index`
- `@landscape` and `@plan` are absorbed into the intent-routing lane, not planned as standalone capabilities

`sdp architect` remains an existing dependency surface. Toolkit planning consumes
it; it does not create a second architect backlog on top of `F105`.

| Feature | Priority | Outcome | Depends On | Status |
|--------|----------|---------|------------|--------|
| `F120` Toolkit Scout | P1 | 30-second repo card with stable `scout.json` and shared exclusions | - | Done |
| `F121` Toolkit Metrics | P1 | git-derived health report for hygiene, flow, risk, and decay | `F120` | Done |
| `F122` Toolkit Index | P1 | persistent `.sdp/index.db` + `.sdp/manifest.md` | `F120` | Done |
| `F123` Toolkit Spec Recovery | P2 | recovered contracts, rules, invariants, and SLA signals | `F120` | Done |
| `F124` Toolkit Bootstrap | P1 | brownfield-safe context docs, policies, hooks, and beads setup | `F120`, `F121`, `F122` | Done |
| `F125` Toolkit UX | P1 | five intent-based skills over composable toolkit tools | `F120`, `F121`, `F122`, `F123`, `F124` | In Progress |
| `F126` Toolkit MCP | P2 | one MCP server exposing toolkit tools, resources, and prompts | `F120`..`F125` | Done |

**Execution slices:**

- Slice A: `F120` + `F121` -> shipped first-look repo understanding
- Slice B: `F122` + `F123` -> shipped persistent context and recovered contracts
- Slice C: `F124` -> shipped brownfield-safe setup; `F125` remains the unfinished UX migration
- Slice D: `F126` -> shipped MCP server (tools, resources, prompts, harness configs, security hardening)

**Status note:** feature merges landed for `F120` (#75), `F121` (#76), `F122` (#77), `F123` (#78), `F124` (#79), core `F125` intent migration (#83), and `F126` (#81). `00-125-05` remains partial via #80, so the toolkit lane is effectively complete except for the final review-readiness/doc-sweep follow-up. `F121` still has one promptops evidence follow-up in beads, and `F122` still has index-hardening follow-ups in beads; merged status here means shipped implementation, not zero open findings.

**Canonical plan:** [2026-04-13-sdp-toolkit-implementation-plan.md](../plans/2026-04-13-sdp-toolkit-implementation-plan.md)
**Detailed child plans:** [F120 Scout](../plans/2026-04-13-sdp-scout-implementation-plan.md), [F121 Metrics](../plans/2026-04-13-sdp-metrics-implementation-plan.md), [F122 Index](../plans/2026-04-13-sdp-index-implementation-plan.md), [F123 Spec](../plans/2026-04-13-sdp-spec-implementation-plan.md), [F124 Bootstrap](../plans/2026-04-13-sdp-bootstrap-implementation-plan.md), [F126 MCP](../plans/2026-04-13-sdp-mcp-implementation-plan.md)

---

## Phase Surface Contract Normalization

**Goal:** normalize the post-toolkit surface so CLI discovery, skill catalog truth, and MCP exposure stop drifting from one another.

This lane is explicitly a delta lane over shipped toolkit foundations:

- `F125` stays shipped and remains the owner of the intent model
- `F126` stays shipped and remains the owner of the initial MCP server
- `F137` is the hard dependency for mini-harness and sweep because they need a stable CLI contract
- `F139` depends on `F137` and `F138`

| Feature | Priority | Outcome | Depends On | Status |
|--------|----------|---------|------------|--------|
| `F137` CLI Surface Normalization | P1 | unified `sdp` entrypoint, registry/discovery contract, shim and deprecation policy | - | Backlog |
| `F138` Skill Catalog Normalization | P1 | canonical skill catalog, deprecation map, and harness-facing docs parity | `F125`, `F137` | Backlog |
| `F139` MCP Contract Parity | P1 | registry/catalog-driven MCP tool, prompt, and resource parity with handshake validation | `F137`, `F138`, `F126` | Backlog |

**Source design:** [2026-04-20-sdp-framework-normalization-design.md](../plans/2026-04-20-sdp-framework-normalization-design.md)
**Execution note:** these features extend shipped surfaces; they do not reopen them.

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

### Feature ID continuity note

Reserved (intentionally unused) feature IDs: `F050`, `F051`, `F057`, `F058`.

These IDs are kept for historical continuity and are not active roadmap items.

### Feature coverage registry

This roadmap focuses active strategy phases. Full feature coverage is maintained in `docs/workstreams/INDEX.md` and includes:

- Archived pre-pivot features: `F001`..`F013`
- Phase 0 completed bootstrap features: `F014`..`F029`
- Remaining Phase 7 bootstrap backlog: `F030`
- Auto-generated planning ranges: `F031`..`F052` (historical placeholders; **closed 2026-04-18** with supersede reasons — see beads)
- Historical protocol and workflow hardening tracks: `F053`..`F056`
- Active strategy and ecosystem ranges: `F059`..`F085`
- Parked long-horizon ideas: `F086`..`F089`
- Canonical alignment and platform reset lanes: `F090`..`F096`
- Product truth, integration, architect, analyst, toolkit, and autonomy backlog: `F097`..`F129`

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

### Ecosystem Synergies

**Source:** [ECOSYSTEM_SYNERGIES.md](../integrations/ECOSYSTEM_SYNERGIES.md)

| System | What SDP Adopts | Feature |
|--------|------------------|--------|
| **OhMyOpenCode** | Permission → Guard bridge, Session evidence | F059 |
| **Beads** | Graph deps, Ready queue, Wisps | F061 |
| **Gas Town** | GUPP pattern, Witness monitoring, Agent CV | F060 |
| **vibe-kanban** | Kanban orchestration, MCP config centralization | F062 |
| **opencode-mem** | Persistent memory, Session continuity, User profiles | F063 |
| **opencode-beads** | Beads plugin for OpenCode | F061 |

### Historical Support Lanes

- **F053** — sdp Repository Comprehensive Audit and Remediation (00-053-01 … 00-053-46). Historical protocol and orchestrator hardening lane. **Status: DONE**.
- **F054** — Continuous Protocol Improvement (00-054-01 … 00-054-06). Historical workflow and agent contract sync lane. **Status: DONE**.
- **F055** — Evidence Enforcement Reality (00-055-01 … 00-055-03). Evidence commit flow, CI gate blocking, and branch protection validation. **Status: DONE**.
- **F056** — Local Git Hook Enforcement (00-056-01 … 00-056-03). Pre-commit, pre-push, and install/docs lane. **Status: DONE**.
- **F096** — Legacy Drift Cleanup (00-096-01 ... 00-096-03). Support lane for roadmap/index/backlog hygiene. Buckets A-D are complete; the remaining Dolt cutover item is blocked on external remote/secrets, so this lane is parked rather than actively executing.

### Ongoing
- **F059** — OhMyOpenCode Evidence Integration (00-059-01 … 00-059-04). Pre-tool-call guard, session evidence emitter, permission↔guard bridge, stuck detection. Phase 5. **Status: DONE** (sdp-omc-guard, sdp-ready implemented).
- **F060** — Gas Town Adapter (00-060-01 … 00-060-04). Convoy→WS bridge, hook→guard, witness escalation, Agent CV→provenance. Phase 8-9 target. Deferred unless explicitly revived. Source: [ECOSYSTEM_SYNERGIES.md](../integrations/ECOSYSTEM_SYNERGIES.md).
**F061** — Beads Graph Integration (00-061-01 … 00-061-04). SQL dep query, `sdp ready` bridge, formula templates, wisps. Phase 5 target. **Status: DONE** (sdp-ready, beads client implemented, opencode-beads plugin installed).
- **F062** — vibe-kanban Integration. **Superseded 2026-04-18 by F125 (5 intents) + F134 (Phase FSM orchestration).** Closed in beads.
- **F063** — opencode-mem Integration. **Superseded 2026-04-18 by F124 (toolkit memory, shipped).** Closed in beads.

### Phase 4: Auto-Attestation

- **F064** — CI Observer Job (00-064-01). Collects git diff, test results, coverage, lint output. **Status: DONE**.
- **F065** — Auto-Attestation Generation (00-065-01). Generates in-toto attestation from CI facts. **Status: DONE**.
- **F066** — Sigstore Signing (00-066-01). Keyless signing with Fulcio+Rekor. **Status: DONE**.
- **F067** — Discrepancy Detection (00-067-01). Agent vs CI attestation comparison. **Status: DONE**.

### Phase 6-8: Dual-Surface Productization (Planned)

- **F068** — Unified Integration Contracts (00-068-01 ... 00-068-05). **Split 2026-04-18 across F091 (backlog reset) + F092 (kernel contract); scope resolved.** Closed in beads.
- **F069** — Control Tower Pack + Spec-Driven Pipeline (00-069-01 ... 00-069-15). ~~One-command setup for OhMyOpenCode + Beads + Gas Town + SDP demo flow~~ **Extended to full spec-driven pipeline**: Control Store MVP, Beads Bridge, Orchestrator Loop, Human/Admin Surface, Dispatch Bridge (OmO transport), Auto-Contract Generation, Provenance Unification, A2A HTTP Interface, Constitution Layer, Advanced Execution (daemon + findings routing). **Status: DONE.** See `docs/SDP_SPEC_DRIVEN_PIPELINE_CANON.md`.
- **F070** — OSS Observability and Explainability (00-070-01 ... 00-070-03). Live event stream, allow/deny explanations, minimal audit export.
- **F071** — Ralph Decommission and Orchestrator V2 (00-071-01 ... 00-071-03). Remove primitive Ralph loop from enterprise profile and migrate to typed FSM orchestration. **Status: DONE.** Runtime `agentloop` wiring remains tracked separately in `F106`.
- **F072** — Advanced Agent Architecture for AI SDLC (00-072-01 ... 00-072-06). **Split 2026-04-18 across F094 (augmentation engine) + F095 (behavioral eval); planning slices merged into F134.** Closed in beads.
- **F073** — BYOM Model Gateway (00-073-01 ... 00-073-03). **Merged 2026-04-18 into F093 (adapter gateway layer); routing policy depends on F130.** Closed in beads.
- **F074** — Enterprise Governance Pack (00-074-01 ... 00-074-03). Multi-tenant RBAC, signed evidence gates, SIEM/compliance exports.
- **F075** — Enterprise K8s Runtime Pack (00-075-01 ... 00-075-03). HA deployment, queue control, canary rollout for agent workflows.
- **F076** — Documentation Agent Automation (00-076-01). Automatic changelog updates and documentation consistency checks on each commit.
- **F077** — CI to Local Bridge for Improvement Loop (00-077-01 ... 00-077-04). **Status: DONE (merged in PR #88 on 2026-04-18).** GitHub CI findings are synchronized into the local Beads queue and now serve as a prerequisite for F129 autonomous operations + F135-03 self-testing loop.

### Phase 8C: Trust Surface and Enterprise Readiness (Compliance lane, coupled with F134)

> **Compliance deadlines:** EU AI Act (2026-08), Colorado AI Act (2026-06). F074, F078, F079, F080, F082 form the trust lane; all linked to F134-03 (evidence enforcement) and/or F134-04 (AI-vs-human attribution) in beads.

- **F078** — Trust Surface Consistency (00-078-01 ... 00-078-03). Enforce version/link/metadata consistency and release-surface checks. **P2, compliance lane.**
- **F079** — Enterprise Trust Pack (00-079-01 ... 00-079-03). Public maturity matrix, canonical guarantees/non-guarantees, CI gates map with local reproduce path. **P2, compliance lane.**
- **F080** — Contract Governance Policy (00-080-01 ... 00-080-03). Schema semver, compatibility gates, conformance tests. **P2, compliance lane; depends on F091/F092 (F068 split).**
- **F081** — 30-Min Production Pilot (00-081-01 ... 00-081-03). CI-gate-only onboarding, contracted-runtime pilot, rollback/disable playbook. **Promoted to P1 2026-04-18**: gating layer for F135-06 deploy gate (ETH/DORA 2025: 60%+ AI errors surface post-deploy).

### Phase 8D: Medium-Priority Trust Expansion (Backlog)

- **F082** — Compliance Control Mapping. Audit-grade mapping: control -> evidence field -> frequency -> verifier -> residual risk.
- **F083** — Policy Engine Enforcement Pack. OPA/Rego policy bundles for evidence completeness and allow/deny decisions.
- **F084** — Enterprise Runtime Hardening. Incremental hardening package around identity, signatures, and operational guardrails.
- **F085** — Platform Productization Kit. Reusable org-level integration templates and operating model.

### Parking Lot: Long-Horizon Ideas

- **F086** — Cross-Project Evidence Federation.
- **F087** — Adversarial Reviewer Quorum.
- **F088** — Autonomous Backlog Synthesis from findings telemetry.
- **F089** — Adaptive Gate Tuning based on historical signal quality.

### Phase 6B: Canonical SDP Workflow (Done)

- **F090** — Canonical SDP Workflow and Orchestrator Alignment. Tighten SDP around one canonical loop (`feature -> workstream -> beads issue -> early draft PR -> review findings -> QA/UAT -> clean PR`) and align orchestrator runtime, verdict artifacts, and operator surfaces to that loop. **Status: DONE.**

### Phase Analyst: Evidence-Backed Strategy Audit (Planned)

- **F109** — StratAudit v2 — evidence-backed report redesign (00-109-01 ... 00-109-08). Turn StratAudit into a verifiable strategy-audit product over messy multilingual corpora: extraction trust gate, source-preserving language policy, document/section/quote provenance, trace evidence contract, grouped findings, `report.v2.json`, and only then final HTML reformatting.
- **F111** — StratAudit portability — provider-neutral engine and skill surface (00-111-01 ... 00-111-04). Split StratAudit into a provider-neutral engine, config-driven runtime resolution, and a reusable skill surface so host-native harness models stay first-class and OpenRouter acts as an enhancer instead of the only runtime path. The active follow-up hardens the skill contract itself: evidence policy, runtime policy, output modes, and fail-closed rules.
- **F117** — StratAudit claim-centric trace explorer and tabbed analyst report (00-117-01 ... 00-117-05). Turn the evidence-first report into a usable analyst surface: claim-based trace graph, first-class trace gaps, document correspondence, tabbed views for summary/documents/trace/gaps/diagnostics, and no compare-first default flow. Current status: `00-117-01..05` done.

### Phase Runtime Contract Normalization (In Progress)

- **F110** — Work Atomicity Normalization — strict leaf execution contract (00-110-01 ... 00-110-05). First make `leaf workstream` the only executable workstream shape in runtime-facing surfaces: normalized frontmatter and `## Beads` roles, compiled `workgraph.lock.json`, stale-lock rejection in the mini-harness, and canonical skill/agent wording that no longer treats every workstream as directly runnable. Then wire live Beads query, claim, revalidation, and explicit claim release into bound leaf sessions. Finalize the runtime contract with counters and structured dispatch diagnostics instead of opaque wrapped CLI errors. Repair the broken `sdp` repo boundary so public protocol publishing is possible again through a real submodule checkout, then publish the public `sdp` prompt/skill/agent wording to match the implemented runtime contract.

### Phase Harness Config Provisioning (In Progress)

- **F130** — AI Harness Config Auto-Provisioning (00-F130-01 ... 00-F130-05). SDP analyzes any repo at any lifecycle stage and generates harness-ready config files (CLAUDE.md sections, .cursorrules, AGENTS.md language rules, codex.yaml) from actual codebase patterns. Pipeline: Scout/Index/SpecRecovery → pattern extractor → rules generator → harness adapter → bootstrap integration. go-patterns.md (2026-04-18) is the reference output format. Depends on F120, F122, F123, F124, F127.
- **F131** — Workflow Skill Provisioning (00-F131-01 ... 00-F131-03). SDP generates workflow skill files (bug-fix, research, feature-delivery) tailored to a repo's tech stack during bootstrap. Extends F130 harness adapter with per-language skill templates. Manual examples shipped 2026-04-18. Depends on F130, F127.

### Layer Rollout Matrix (Vision Alignment)

Reference: `docs/vision/SDP_LAYERED_VISION.md`

| Layer | Roadmap focus | Delivery signal |
|-------|---------------|-----------------|
| L1 Protocol | F068, F080 + protocol contracts and compatibility policy | Stable input/output + self-check contracts across runtimes |
| L2 Runtime Governance | F059, F064-F067, F078 + protocol-compliance gates | Drift blocked before merge, no unsupported claims |
| L3 Orchestration Fabric | F071, F072, F077 + ecosystem bridges | Deterministic phase transitions with gate-controlled advancement |
| L4 Enterprise Trust | F074, F079, F081, F082, F083 | Verifiable claims, pilot-ready governance, and compliance mapping |
| L5 OSS Harness Runtime | F075, F084, F085 + K8s phases 8-9 | Portable operator/harness execution with trust controls |

### KPI Baseline for Layer Completion

- Evidence validity rate (`valid_evidence_envelope`) >= 99%
- Claim evidence linkage (`claims_with_evidence_refs`) = 100%
- Drift block effectiveness (`blocked_drift_before_done`) = 100% for critical drift
- Protocol compliance (`protocol_compliant_runs`) >= 95% before L3 graduation
- Gate reliability (`false_positive_gate_blocks`) <= 2% in rolling 30-day window


### Project History
- [Agent Loop Reliability](../archive/plans/2026-02-23-agent-loop-reliability.md)
- [Stripe Minions Comparison](../archive/plans/2026-02-23-stripe-minions-comparison.md)
- [Phase 0 Enforcement Audit](../archive/plans/2026-02-24-phase0-enforcement-audit.md)
- [Enforcement Hypotheses](../archive/plans/2026-02-24-enforcement-hypotheses.md)
- [ADR-002 Standards Pivot](../decisions/ADR-002-standards-pivot.md)
- K8s domain knowledge: `archive/k8s-v0` branch
