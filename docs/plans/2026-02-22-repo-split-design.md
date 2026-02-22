# SDP Repo Split: Architecture, Naming, Migration

> **Status:** Research complete
> **Date:** 2026-02-22
> **Goal:** Split monolith into focused repos: protocol (public), evidence tooling (public), K8s orchestration (public), experimental lab (private), enterprise (private)

---

## Table of Contents

1. [Overview](#overview)
2. [Naming Strategy](#1-naming-strategy)
3. [Repo Map](#2-repo-map)
4. [SDP Protocol Surgery](#3-sdp-protocol-surgery)
5. [Go Module Architecture](#4-go-module-architecture)
6. [Enterprise vs OSS Boundary](#5-enterprise-vs-oss-boundary)
7. [Migration Plan](#6-migration-plan)
8. [Community Positioning](#7-community-positioning)
9. [Vision Alignment](#8-vision-alignment)

---

## Overview

### Goals

1. **Focus** — each repo has one identity and one audience
2. **Publish** — evidence tooling and K8s adapter as standalone OSS projects
3. **Adopt** — use community tools where we're not differentiated
4. **Enterprise** — clear boundary between OSS and paid features

### Key Decisions

| Aspect | Decision |
|--------|----------|
| Evidence repo name | **TraceForge** (backup: ProvenanceKit) |
| K8s repo name | **SwarmOps** or **sdp-operator** (Vasiliy stays as mascot) |
| Shared Go lib | New `sdp-go` repo for beads, policy, bus, observability |
| SDP protocol | Strip Go CLI out; keep prompts, schemas, hooks, docs |
| CLI fate | Separate `sdp-plugin` repo, installs via `go install` or brew |
| Migration order | Evidence first → Protocol cleanup → K8s extraction |
| Enterprise scope | SaaS, SSO/RBAC, compliance templates, SLA/support |

---

## 1. Naming Strategy

### Evidence/Trace Repo

| Candidate | Verdict | Reasoning |
|-----------|---------|-----------|
| **TraceForge** | **Winner** | "Trace" matches trace validation; "Forge" suggests building/hardening evidence. Professional, memorable. `traceforge` CLI works. `go get github.com/org/traceforge` is clean. |
| BlameChain | **Reject** | "Blame" is defensive/negative. Enterprise compliance teams won't adopt a tool with "blame" in the name. |
| CodeLedger | **Runner-up** | "Ledger" fits audit/immutability. "Code" is narrowing — system covers evidence beyond code. |
| ProvenanceKit | **Backup** | Clear, professional. Aligns with in-toto/Witness ecosystem terminology. |
| EvidenceChain | **Backup** | Direct, audit-friendly. May be too generic. |

### K8s Orchestration Repo

| Candidate | Verdict | Reasoning |
|-----------|---------|-----------|
| **SwarmOps** | **Winner** | Matches "swarm" terminology in docs, suggests operations/orchestration, short. |
| sdp-operator | **Runner-up** | Clearer SDP association; good for kubeopencode ecosystem. |
| Vasiliy-ai | **Mascot only** | Fun, meme value is high, but CNCF/enterprise won't accept it. Keep as mascot/codename. |
| AgentRun | **Reject** | Too generic; collides with the CRD name. |

### Full Ecosystem

| Repo | Name | Visibility | Identity |
|------|------|------------|----------|
| Protocol | **sdp** (existing) | Public | "The spec" — prompts, schemas, hooks |
| Evidence | **traceforge** | Public | "The proof" — evidence validation, provenance |
| K8s Orchestration | **swarmops** | Public | "The platform" — K8s adapter, CRDs, federation |
| Shared Go lib | **sdp-go** | Public | "The SDK" — beads, policy, bus, observability |
| CLI | **sdp-plugin** | Public | "The tool" — `sdp quality all`, `sdp init` |
| Experimental | **sdp_dev** | Private | "The lab" — experiments, prototypes |
| Enterprise | **traceforge-enterprise** | Private | "The business" — SaaS, SSO, compliance |

---

## 2. Repo Map

```
                         ┌──────────────────┐
                         │     sdp          │ PUBLIC
                         │  (protocol)      │
                         │  prompts/        │
                         │  schema/         │
                         │  docs/           │
                         │  hooks/          │
                         │  templates/      │
                         └────────┬─────────┘
                                  │ schemas define contracts
                    ┌─────────────┼─────────────┐
                    │             │             │
                    ▼             ▼             ▼
            ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
            │   sdp-go     │ │  sdp-plugin  │ │              │
            │ (shared lib) │ │  (CLI)       │ │              │
            │ pkg/beads    │ │ sdp init     │ │              │
            │ pkg/policy   │ │ sdp quality  │ │              │
            │ pkg/bus      │ │ sdp guard    │ │              │
            │ pkg/observe  │ │              │ │              │
            └──────┬───────┘ └──────────────┘ │              │
                   │                           │              │
           ┌───────┼───────┐                   │              │
           │               │                   │              │
           ▼               ▼                   │              │
    ┌──────────────┐ ┌──────────────┐          │              │
    │ traceforge   │ │  swarmops    │          │              │
    │ (evidence)   │ │ (K8s orch)  │          │              │
    │              │ │              │          │              │
    │ evidence/    │ │ adapter/     │          │              │
    │ artifact/    │ │ federation/  │          │              │
    │ quality/     │ │ orchestrator/│          │              │
    │ pr-gate      │ │ CRDs        │          │              │
    │ beads-fsm    │ │ deploy/     │          │              │
    │ telemetry    │ │ Helm chart  │          │              │
    └──────────────┘ └──────────────┘          │              │
                                               │              │
                         ┌─────────────────────┘              │
                         │                                    │
                         ▼                                    ▼
              ┌────────────────────┐            ┌────────────────────┐
              │     sdp_dev        │ PRIVATE    │ traceforge-ent.    │ PRIVATE
              │  (experimental)    │            │ (enterprise)       │
              │  prototypes        │            │ SaaS control plane │
              │  legacy code       │            │ SSO/RBAC           │
              │  integration tests │            │ Compliance         │
              └────────────────────┘            └────────────────────┘
```

---

## 3. SDP Protocol Surgery

### Current State (sdp/ submodule)

The SDP repo has **two souls** mixed together:
- **Soul 1: Protocol** (~2.1 MB) — prompts, schemas, docs, hooks, templates
- **Soul 2: CLI** (~4.9 MB, 658 Go files, 106K LOC) — sdp-plugin with 38 internal packages

### Surgery Plan

#### STAYS in SDP (pure protocol)

```
sdp/
├── prompts/
│   ├── skills/           # 28 skill definitions
│   └── agents/           # 30+ agent definitions
├── docs/
│   ├── PROTOCOL.md       # Core specification
│   ├── reference/        # Protocol reference
│   ├── design/           # Design decisions
│   └── vision/           # Vision docs
├── schema/
│   └── *.schema.json     # 6 JSON schemas (evidence, traceability, etc.)
├── templates/            # 10 workstream templates
├── hooks/                # Shell validators (pre-edit, session-quality)
├── .claude/              # Claude Code integration (symlinks to prompts)
├── .cursor/              # Cursor integration (symlinks + commands)
├── .opencode/            # OpenCode integration
├── scripts/
│   ├── install.sh        # Entry point (installs protocol + optionally CLI)
│   └── install-project.sh
├── AGENTS.md
├── README.md
└── LICENSE
```

**No Go code.** Hook validators stay as shell scripts. `session-quality-check.sh` calls `sdp` CLI if installed, degrades gracefully if not.

#### MOVES to sdp-plugin (separate repo)

```
sdp-plugin/
├── cmd/sdp/              # CLI commands (~16K LOC)
├── internal/             # 38 packages (~90K LOC)
│   ├── checkpoint/
│   ├── guard/
│   ├── memory/
│   └── ... (everything)
├── src/                  # Research/experimental Go
├── go.mod                # module github.com/org/sdp-plugin
└── README.md
```

**Dependencies:** sdp-plugin imports `github.com/org/sdp-go` for shared types. Evidence/quality logic inside sdp-plugin stays there (different implementation from traceforge — local CLI vs K8s).

#### TRANSITION

| Step | Action | Impact |
|------|--------|--------|
| 1 | Create `sdp-plugin` repo from `sdp/sdp-plugin/` | New repo |
| 2 | Remove `sdp-plugin/` and `src/` from SDP | SDP becomes protocol-only |
| 3 | Update `install.sh` to optionally `go install github.com/org/sdp-plugin/cmd/sdp@latest` | CLI is global tool |
| 4 | sdp_dev switches submodule to protocol-only SDP | Symlinks still work (prompts/) |
| 5 | sdp_dev uses `sdp` binary from PATH (not from submodule) | Decoupled |

---

## 4. Go Module Architecture

### Module Dependency Graph

```
github.com/org/sdp            (no Go — protocol only)

github.com/org/sdp-go         (shared library)
├── pkg/beads/                 (~400 LOC)  Beads CLI adapter
├── pkg/policy/                (~900 LOC)  Config, decisions, model chain
├── pkg/bus/                   (~1,200 LOC) NATS client, trace propagation
└── pkg/observability/         (~830 LOC)  Tracing, metrics, logging

github.com/org/traceforge     (evidence tooling)
├── require: sdp-go
├── internal/evidence/         Strict validation, trace validator
├── internal/artifact/         Provenance hash chain
├── internal/quality/          Quality pipeline
├── internal/adapter/          CRD reconciliation, evidence projector
├── cmd/pr-gate/               PR gate CLI
├── cmd/beads-fsm/             FSM validator
├── cmd/telemetry-analyzer/    Telemetry analysis
└── cmd/adapter-controller/    K8s controller (builds adapter binary)

github.com/org/swarmops        (K8s orchestration)
├── require: sdp-go
├── internal/orchestrator/     Scheduling, dispatch, aggregation
├── internal/federation/       Cross-project bridge
├── internal/registry/         Project registry
├── internal/swarm/            Swarm coordination
├── cmd/feature-orchestrator/  Primary orchestrator
├── cmd/swarm-worker/          Worker binary
├── cmd/intake-gateway/        HTTP intake
└── deploy/                    K8s manifests, Helm chart, CRDs
```

### Why adapter-controller is in traceforge (not swarmops)

The adapter-controller's primary job is **evidence projection** — it reconciles CRD status into evidence envelopes. Its core imports are `internal/evidence`, `internal/quality`, `internal/artifact`. It's an evidence bridge into K8s, not an orchestration component.

swarmops depends on traceforge only for the adapter-controller **Docker image**, not as a Go library import. The K8s deploy manifests reference the traceforge image.

### Cross-repo dependency rules

```
sdp-go      → nothing (leaf library)
traceforge  → sdp-go only
swarmops    → sdp-go only (adapter image from traceforge registry)
sdp-plugin  → sdp-go only (or sdp-go + traceforge for evidence validation)
sdp_dev     → all of the above (integration playground)
```

**No circular imports. No traceforge → swarmops. No swarmops → traceforge Go imports.**

---

## 5. Enterprise vs OSS Boundary

| Component | OSS | Enterprise | Reasoning |
|-----------|-----|------------|-----------|
| Evidence spec + validation | **traceforge** | — | Core auditability, must be OSS for trust |
| PR gate CLI | **traceforge** | — | CI/CD tool, universal |
| Beads FSM | **traceforge** | — | Protocol enforcement |
| Telemetry analyzer (base) | **traceforge** | — | Rule-based metrics |
| Telemetry analyzer (LLM) | — | **traceforge-ent** | Custom LLM prompts, proprietary models |
| Policy engine | **swarmops** | — | Model allowlist, risk gates |
| Multi-project federation | **swarmops** | — | Core platform feature |
| K8s adapter + CRDs | **swarmops** | — | Open platform |
| SaaS hosted orchestration | — | **traceforge-ent** | Managed control plane |
| Compliance templates (SOC2, HIPAA) | — | **traceforge-ent** | Enterprise policy rules in Rego |
| SSO/RBAC integration | — | **traceforge-ent** | LDAP, OIDC, per-org policies |
| SLA & support | — | **traceforge-ent** | Incident response, SLOs |
| Cross-project analytics dashboard | — | **traceforge-ent** | Cost attribution, trends |

**Rule:** If a solo dev can run it on their own cluster, it's OSS. Enterprise = managed services + compliance + support.

---

## 6. Migration Plan

### Phase-by-phase Timeline

```
Week 0    Week 1-2     Week 3-4      Week 5-6     Week 7-8     Week 9+
  │          │            │             │            │            │
  ▼          ▼            ▼             ▼            ▼            ▼
┌────┐   ┌────────┐  ┌──────────┐  ┌──────────┐  ┌────────┐  ┌────────┐
│Prep│   │Evidence │  │Protocol  │  │K8s Orch  │  │Cleanup │  │Launch  │
│    │   │Extract  │  │Surgery   │  │Extract   │  │sdp_dev │  │Commun. │
└────┘   └────────┘  └──────────┘  └──────────┘  └────────┘  └────────┘
```

### Phase 0: Preparation (Week 0)

- [ ] Reserve GitHub repo names: `traceforge`, `swarmops`, `sdp-go`, `sdp-plugin`
- [ ] Decide org name (fall-out-bug? new org like `sdp-platform`?)
- [ ] Add `.gitignore` for `*.out` files in sdp_dev
- [ ] Delete dead binaries in sdp_dev (brain-gateway, operator-gate, openclaw-agent)
- [ ] Update PRODUCT_VISION.md with repo split plan

### Phase 1: Evidence Extraction — TraceForge (Weeks 1-2)

**Minimum Viable Extraction:**

- [ ] Create `traceforge` repo with `go.mod`
- [ ] Move from sdp_dev:
  - `internal/evidence/` → `traceforge/internal/evidence/`
  - `internal/artifact/` → `traceforge/internal/artifact/`
  - `internal/quality/` → `traceforge/internal/quality/`
  - `internal/adapter/` → `traceforge/internal/adapter/`
  - `cmd/pr-gate/` → `traceforge/cmd/pr-gate/`
  - `cmd/beads-fsm/` → `traceforge/cmd/beads-fsm/`
  - `cmd/telemetry-analyzer/` → `traceforge/cmd/telemetry-analyzer/`
  - `cmd/adapter-controller/` → `traceforge/cmd/adapter-controller/`
  - `specs/strict-evidence-template.json` → `traceforge/specs/`
- [ ] Create `sdp-go` repo with shared packages:
  - `pkg/beads/` from `internal/beads/`
  - `pkg/policy/` from `internal/policy/`
  - `pkg/bus/` from `internal/bus/`
  - `pkg/observability/` from `internal/observability/`
- [ ] `traceforge` depends on `sdp-go`
- [ ] Write README with pr-gate usage and CI integration guide
- [ ] Tag v0.1.0
- [ ] sdp_dev uses `go mod replace` during transition

### Phase 2: Protocol Surgery (Weeks 3-4)

- [ ] Create `sdp-plugin` repo from `sdp/sdp-plugin/`
- [ ] Remove `sdp-plugin/` and `src/` from SDP repo
- [ ] Update SDP README: "This is the protocol spec. CLI: `go install .../sdp-plugin`"
- [ ] Update `install.sh` to install protocol + optionally install CLI binary
- [ ] sdp_dev updates submodule reference
- [ ] Verify symlinks still work (prompts/ → skills)
- [ ] Update `.cursor/skills`, `.claude/skills` paths

### Phase 3: K8s Orchestration — SwarmOps (Weeks 5-6)

- [ ] Create `swarmops` repo with `go.mod`
- [ ] Move from sdp_dev:
  - `internal/orchestrator/` → `swarmops/internal/orchestrator/`
  - `internal/federation/` → `swarmops/internal/federation/`
  - `internal/registry/` → `swarmops/internal/registry/`
  - `internal/swarm/` → `swarmops/internal/swarm/`
  - `cmd/feature-orchestrator/` → `swarmops/cmd/feature-orchestrator/`
  - `cmd/swarm-worker/` → `swarmops/cmd/swarm-worker/`
  - `cmd/intake-gateway/` → `swarmops/cmd/intake-gateway/`
  - `deploy/` → `swarmops/deploy/`
- [ ] Create Helm chart or consolidated Kustomize overlay
- [ ] `swarmops` depends on `sdp-go`
- [ ] Write README with architecture diagram and quickstart
- [ ] Tag v0.1.0

### Phase 4: Cleanup (Weeks 7-8)

- [ ] sdp_dev: remove all extracted code, keep only experiments
- [ ] sdp_dev: update go.mod to import traceforge, swarmops, sdp-go
- [ ] sdp_dev: remove `go mod replace` directives
- [ ] Delete remaining dead binaries
- [ ] Create `traceforge-enterprise` repo (private, skeleton)
- [ ] Update all docs: PRODUCT_VISION, ROADMAP, workstreams

### Phase 5: Community Launch (Week 9+)

- [ ] Submit traceforge to awesome-opencode
- [ ] Submit swarmops to kubeopencode ecosystem
- [ ] Publish `EVIDENCE_PROTOCOL_SPEC.md` in SDP
- [ ] Blog post / thread: "Evidence for AI Agent Runs"

---

## 7. Community Positioning

### Each Repo's Identity

| Repo | Tagline | awesome-opencode? | Target Audience |
|------|---------|-------------------|-----------------|
| **sdp** | "Structured Development Protocol — the spec for AI agent workflows" | Yes (spec/protocol) | Framework authors, tool builders |
| **traceforge** | "Audit-grade evidence for AI agent runs" | Yes (project) | Platform teams, compliance, CI/CD |
| **swarmops** | "K8s-native AI agent orchestration with strict evidence" | kubeopencode ecosystem | DevOps, platform engineers |
| **sdp-go** | "Go SDK for the Structured Development Protocol" | No (utility) | Go developers building SDP-compatible tools |
| **sdp-plugin** | "CLI for SDP quality gates and development workflow" | Yes (tool) | Individual developers |

### README Strategy

**traceforge README:**
```
# TraceForge

Audit-grade evidence for AI agent runs.

TraceForge validates strict evidence envelopes
(intent → plan → execution → verification → review → provenance)
with JSON Schema and hash-chain provenance.

## Quick Start (CI)
$ traceforge validate --evidence .sdp/evidence/run-123.json
$ traceforge pr-gate --issue sdp_dev-abc --mode strict

## Quick Start (K8s)
adapter-controller uses TraceForge for evidence projection.
```

**swarmops README:**
```
# SwarmOps

K8s-native AI agent orchestration with strict evidence.

Issue → AgentRun CRD → kubeopencode → multi-role agents → PR
with evidence validation at every step.

## Architecture
[intake] → NATS → [adapter-controller] → AgentRun → Task CRD → kubeopencode
```

---

## 8. Vision Alignment

### Does splitting weaken or strengthen the vision?

**Strengthens.**

The vision says: *"There is no production-ready platform that takes an issue from the backlog and returns a PR through a K8s-native multi-role agent swarm with strict evidence."*

Split repos make each piece independently adoptable:

| Vision Element | Repo | Standalone Value |
|----------------|------|------------------|
| "Strict evidence" | traceforge | Use without K8s — CI validation, PR gates |
| "K8s-native" | swarmops | Use without evidence — basic orchestration |
| "Protocol" | sdp | Use without either — structured dev workflow |
| "Issue→PR" | composition | All three together = full vision |

The "one platform" is the **composition** documented in swarmops:
```
helm install sdp swarmops/sdp-stack
# Installs: adapter-controller (from traceforge) + feature-orchestrator + CRDs + NATS
# Requires: kubeopencode operator (upstream)
```

### Risk: Fragmentation

**Mitigation:** swarmops provides a single Helm chart / Kustomize overlay that installs the full stack. User doesn't need to assemble pieces manually.

---

## Success Metrics

| Metric | Baseline | Target |
|--------|----------|--------|
| Repos | 2 (sdp, sdp_dev) | 7 (sdp, traceforge, swarmops, sdp-go, sdp-plugin, sdp_dev, traceforge-ent) |
| sdp_dev lines | ~31,000 | ~5,000 (experiments only) |
| SDP Go lines | 106,000 | 0 (protocol only) |
| traceforge v0.1.0 | — | Published with pr-gate + validation |
| awesome-opencode listings | 0 | 2 (traceforge + swarmops) |
| Independent CI per repo | 0 | All repos have own CI |
| External adoption | 0 | First non-us user of traceforge |
