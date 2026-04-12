# SDP Ecosystem Identity: From Gas Cloud to Star

> **Status:** Research complete
> **Date:** 2026-02-22
> **Goal:** Define SDP's identity, positioning, and ecosystem strategy for gas-cloud stage

---

## Overview

### Core Question

SDP is a gas cloud — forming, not formed. How should it position itself? As an isolated ecosystem of 7 repos with branded products? Or as one focused contribution to the opencode OSS community?

### Key Decisions

| Aspect | Decision |
|--------|----------|
| SDK (sdp-go) | **Kill it.** Premature abstraction with 0 external consumers |
| CLI (sdp-plugin) | **Keep in monorepo.** Not a separate repo until someone else needs it |
| Repo count | **2 repos** (sdp protocol-only + sdp_dev monorepo), not 7 |
| Naming | **sdp-* umbrella** when we do split, not branded chaos |
| Manifesto tone | **Formation narrative** — own the gas-cloud stage honestly |
| Ecosystem play | **Evidence layer + upstream contributions**, not parallel universe |
| Commodity binaries | **Kill 15+ binaries.** Use ecosystem tools instead |

---

## 1. SDK and CLI: Do We Need Them?

> **Experts:** Sam Newman, Martin Kleppmann, Martin Fowler

### The Verdict: Not Yet

**sdp-go is a premature abstraction.** It's 4 unrelated packages (beads, bus, policy, observability — 3,300 LOC) that exist as a "shared library" for two consumers... both written by the same person, from the same codebase, on the same schedule. The bus package even depends on internal/artifact, so the "leaf library" boundary isn't clean.

Martin Fowler's principle applies: "Monolith first. Extract when you feel the pain, not when you imagine the pain."

**sdp-plugin is a real product (91K LOC)** but doesn't need its own repo. Its internal packages (evidence, quality, guard) are *different implementations* from sdp_dev's — they don't share code. Putting them in the same Go workspace doesn't create coupling.

### What To Do

| Item | Action | Trigger to Reconsider |
|------|--------|----------------------|
| sdp-go | Don't create | An external Go project wants to import our types |
| sdp-plugin | Keep in sdp_dev as `cmd/sdp/` | Someone needs the CLI without the platform code |
| SDP protocol repo | Strip all Go code → pure spec | Already needed (the "two souls" problem) |

### On the "sdp" Name

The CLI should stay `sdp` — it's what the project is called. The packages are `internal/`, so naming doesn't matter to anyone but us. Drop the "sdp-go" and "sdp-plugin" repo names entirely. When (if) we extract, use `sdp-evidence` not `traceforge` (see naming section).

---

## 2. Naming: Is Chaos Normal?

> **Experts:** Sam Newman, Kelsey Hightower, Nir Eyal

### The Verdict: No, It's Premature Brand Architecture

Three naming conventions (sdp-*, traceforge, swarmops) for a 1-person project with 0 users is designing the conference booth before writing the README.

**The HashiCorp trap:** HashiCorp didn't name 5 products on day zero. They shipped Vagrant (2010), then Packer (2013), then Terraform (2014), then Vault (2015). Each earned its brand through sequential product-market fit. SDP has zero users and hasn't split the repo yet.

**The manifesto contradiction:** The manifesto says "Everything else is commodity. Evidence is the moat." But `swarmops` gives equal brand weight to the orchestration layer that the manifesto explicitly calls commodity.

### Recommendation

**If/when we split, use `sdp-*` umbrella prefix:**

| Planned Name | Better Name | Why |
|--------------|-------------|-----|
| traceforge | sdp-evidence | Discoverable under the umbrella |
| swarmops | sdp-operator | What it actually is |
| sdp-go | (don't create) | — |
| sdp-plugin | (keep in monorepo) | — |

**Upgrade path:** If `sdp-evidence` gets traction (100+ stars, external users), *then* rebrand it to TraceForge. GitHub redirects handle the transition. Earn the brand, don't pre-assign it.

**But the real recommendation: don't split yet.** The naming question evaporates if we keep a monorepo with good internal structure.

---

## 3. Honesty: The Manifesto Overclaims

> **Experts:** Kelsey Hightower, Troy Hunt, Martin Kleppmann

### The Problem

The manifesto claims:
- "SDP is the only system that produces..." — competitive positioning against a market that can't evaluate the claim because the code isn't public
- An ecosystem table with 5 repos — presenting a planned split as existing architecture
- Three user personas — all of whom are the same developer
- "10+ consecutive successful runs" — there are 5 of 10

Troy Hunt: "Every unverifiable claim is a liability. The moment someone checks and finds the 5 repos don't exist, every other claim becomes suspect."

### The Fix: Formation Narrative

**Separate tense:**

1. **What IS** (present tense, verifiable): The protocol spec. The evidence schema. 240 commits. One developer using it daily.

2. **What WILL BE** (future tense, explicitly aspirational): The repo split. Multi-user adoption. K8s hardening.

3. **What MATTERS** (timeless, the thesis): The evidence gap in agent tooling. The audit trail nobody is building.

**Concrete changes to the manifesto:**

| Current | Better |
|---------|--------|
| "SDP is the only system that produces..." | "We believe agent tooling needs structured evidence, and nobody is building it. Here's our attempt." |
| Ecosystem table with 5 repos | Status matrix with Implemented / In Development / Planned |
| "The Honest Part" at the bottom | Status section at the top |
| Three user personas | "Today: one developer's daily workflow. Tomorrow: ..." |

**Replace the ecosystem table with:**

```
| Component    | Status         | Where             |
|--------------|----------------|-------------------|
| Protocol     | Implemented    | sdp/ (submodule)  |
| Evidence     | Implemented    | internal/evidence |
| PR Gate      | Implemented    | cmd/pr-gate       |
| K8s Platform | In Development | cmd/adapter-*     |
| CLI          | In Development | cmd/sdp (planned) |
| SDK          | Planned        | —                 |
```

### Why This Is Stronger

The problem statement ("Show me proof that this PR was planned, implemented, tested, reviewed") is genuinely the best part of the manifesto. It doesn't need embellishment. Honest framing of what exists *increases* trust. Pre-1.0 projects that overclaim attract tire-kickers who leave disappointed. Projects that are honest attract builders who stay.

---

## 4. Ecosystem Integration: Not a Parallel Universe

> **Experts:** Thorsten Ball, Harrison Chase, Sam Newman, Martin Kleppmann

### The Overlap Problem

SDP maintains 27 binaries. Most rebuild what the ecosystem already ships:

| SDP Component | Ecosystem Equivalent | SDP's Actual Contribution |
|---|---|---|
| swarm-orchestrator | Vibe Kanban (21K stars), Swarm Tools | **Zero** — commodity |
| intake-gateway, telegram | Various webhook plugins | **Zero** — commodity |
| policy engine | Cupcake (OPA/Rego, Wasm) | **Partial** — evidence-aware gating |
| beads integration | beads upstream, Swarm Tools | **Zero** — consumer |
| adapter-controller | kubeopencode upstream | **Novel** — evidence projection bridge |
| evidence validation | **Nothing in ecosystem** | **Entirely novel** |
| provenance hash chains | **Nothing** | **Entirely novel** |
| PR gate + evidence checks | **Nothing** | **Entirely novel** |
| beads-fsm | **Nothing** | **Entirely novel** |

### The Strategy: Evidence Layer + Upstream Contributions

**SDP's pitch to the ecosystem:**

> "You already have orchestration (Vibe Kanban). You already have policy (Cupcake). You already have K8s (kubeopencode). You don't have proof. SDP is the evidence layer."

**What we ship (public):**

| # | Deliverable | Form | Ecosystem Position |
|---|---|---|---|
| 1 | Evidence JSON Schema | Published spec in `sdp` | Protocol reference for any tool |
| 2 | `traceforge validate` / `sdp-evidence` CLI | Go binary | CI tool, awesome-opencode listing |
| 3 | opencode plugin for evidence collection | TS plugin (shells out to CLI) | Reaches opencode users directly |
| 4 | UP-001, UP-003 kubeopencode PRs | Upstream contributions | K8s native evidence |

**What we kill:**

| Binary | Reason | Replacement |
|---|---|---|
| swarm-orchestrator | Vibe Kanban does this better | Vibe Kanban / Swarm Tools |
| feature-orchestrator | Commodity | Vibe Kanban |
| intake-gateway | Generic webhook | Direct NATS / API |
| intake-telegram | Niche | Keep in private lab if needed |
| brain-gateway | LLM routing | OpenRouter API directly |
| openclaw-agent | Alternative runtime experiment | Private lab |
| operator-gate | Redundant | Fold into evidence CLI |
| autonomy-worker | Commodity | Swarm Tools |

**What survives as differentiated code:**

| Binary | Why Differentiated |
|---|---|
| pr-gate | Evidence-gated PR validation — no equivalent |
| beads-fsm | Evidence-linked state machine — no equivalent |
| telemetry-analyzer | Evidence-based run analysis — no equivalent |
| adapter-controller | K8s evidence projection bridge — novel |
| sdp CLI | Developer workflow + quality gates |

### The Give-Back Loop

```
SDP                                    Ecosystem
──────                                 ──────────
Evidence spec   ──────publish───────►  Any tool can produce/validate
                                       
UP-001 retry    ──────PR─────────────►  kubeopencode (upstream)
UP-003 traces   ──────PR─────────────►  kubeopencode (upstream)

opencode plugin ──────npm──────────►  108K opencode users

                ◄─────consume────────  beads (issue tracking)
                ◄─────consume────────  kubeopencode (agent exec)
                ◄─────consume────────  Vibe Kanban (orchestration)
                ◄─────consume────────  Cupcake (policy, if needed)
```

---

## 5. Repo Count: 2, Not 7

> **Experts:** Martin Fowler, all above

### The Math

7 repos for 1 developer = 7x CI pipelines, 7x READMEs, 7x go.mods, 7x issue trackers, 7x release processes. That's overhead, not architecture.

### The Recommendation

**Phase 1 (Now): 2 repos**

| Repo | Contents | Visibility |
|------|----------|------------|
| `sdp` | Protocol spec: prompts, schemas, hooks, docs. No Go code. | Public |
| `sdp_dev` | All Go code: evidence, adapter, CLI, experiments | Private |

**Phase 2 (When triggered): 3 repos**

| Trigger | Action |
|---------|--------|
| External user wants `traceforge validate` in their CI | Extract `sdp-evidence` from sdp_dev |
| External Go project wants to import evidence types | Extract `sdp-go` (just the evidence types, not everything) |
| kubeopencode rejects our upstream PRs | Keep adapter-controller in sdp_dev |
| kubeopencode accepts UP-003 | Delete adapter-controller, it's upstream now |

**Write the triggers down. Don't extract until a trigger fires.**

---

## 6. Adoption Path for Outsiders

### How A Stranger Discovers SDP

**Today:** They can't. It's private.

**After Phase 1:**

1. **awesome-opencode listing:** "sdp — Structured evidence for AI agent runs. Protocol spec + validation CLI."
2. They read the protocol README → understand the evidence envelope concept
3. They `go install .../sdp_dev/cmd/pr-gate@latest` or download a binary release
4. They add `traceforge validate` to their CI → immediate value
5. They want local evidence → install the opencode plugin
6. They want K8s integration → that's the "come talk to us" moment

### The One-Liner That Matters

> "Add `traceforge validate` to your CI. Get audit-grade proof of what your AI agents actually did."

That's the adoption path. Not "deploy our K8s platform." Not "adopt our entire protocol." One CLI command in CI.

---

## Summary: What Changes

### Mental Model Shift

| From | To |
|------|-----|
| SDP is a platform | SDP is an evidence layer |
| 7 repos, branded ecosystem | 2 repos, focused on what's novel |
| Build everything ourselves | Use ecosystem, contribute evidence |
| "The only system" (claim) | "Nobody else is building this" (observation) |
| Ready ecosystem (fiction) | Forming from gas cloud (truth) |

### Immediate Actions

1. **Rewrite manifesto** as formation narrative with status matrix
2. **Strip Go from SDP protocol repo** → pure spec (the "two souls" fix)
3. **Don't create** sdp-go, sdp-plugin, traceforge, swarmops repos yet
4. **Identify binaries to kill** vs keep in sdp_dev
5. **Publish evidence JSON Schema** in SDP protocol repo
6. **Ship one CLI command:** `traceforge validate` (or `sdp evidence validate`) as a binary release from sdp_dev

### What We're NOT Doing

- Not splitting into 7 repos
- Not creating branded products (TraceForge, SwarmOps) before we have users
- Not building orchestration (ecosystem has it)
- Not claiming things that don't exist yet
- Not designing the conference booth before writing the README

---

*"AI agents can implement features, but without evidence it's just vibes."*

*And right now, we're building the evidence. Honestly.*
