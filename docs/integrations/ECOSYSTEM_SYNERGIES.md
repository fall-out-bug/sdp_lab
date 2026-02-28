# Ecosystem Synergies — SDP × OhMyOpenCode × Gas Town × Beads

> **Status:** Research Complete
> **Date:** 2026-03-01
> **Goal:** Define integration points that turn SDP into an evidence-gated runtime for ecosystem tools

---

## Executive Summary

SDP's core value is **evidence + enforcement**. Three ecosystem tools provide complementary capabilities:

| System | Has | SDP Adds | Integration Feature |
|--------|-----|----------|---------------------|
| **OhMyOpenCode** | Permission system, session management | Pre-tool-call guard, hash chain evidence | **F059** |
| **Gas Town** | Multi-agent orchestration, GUPP, worktree hooks | Evidence envelope, witness escalation | **F060** |
| **Beads** | Dependency graph, formulas, wisps | Workstream sync, ready queue | **F061** |

**Unique Value:** SDP becomes the *trust layer* that these orchestrators emit evidence through. "Issue in, PR with proof out" — SDP provides the "proof" part.

---

## F059: OhMyOpenCode Evidence Integration

**Phase:** 5 (Policy-as-Code)
**LOC estimate:** ~1,200
**Repository:** sdp (public adapter) + sdp_lab (private integration tests)

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│           OhMyOpenCode (Agent Runtime)                      │
│  Permission Engine │ Session Manager │ Tool Router         │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│           SDP Integration Layer                             │
│                                                             │
│  ┌─────────────────┐  ┌─────────────────┐                  │
│  │ sdp-omc-guard   │  │ sdp-omc-emitter │                  │
│  │ (pre-tool-call) │  │ (session events)│                  │
│  │                 │  │                 │                  │
│  │ • GUPP check    │  │ • hash chain    │                  │
│  │ • boundary val  │  │ • typed entries │                  │
│  └─────────────────┘  └─────────────────┘                  │
│                                                             │
│  Output: .sdp/evidence/session-<id>.json (in-toto format)  │
└─────────────────────────────────────────────────────────────┘
```

### Workstreams

| WS | Title | AC |
|----|-------|----|
| **00-059-01** | Pre-tool-call guard hook | Hook fires before edit/write/bash; checks scope via GUPP |
| **00-059-02** | Session evidence emitter | Every tool call emits typed event to .sdp/log/session.jsonl |
| **00-059-03** | Permission↔Guard bridge | OMO pattern ACL `ask` → SDP boundary gate escalation |
| **00-059-04** | Stuck detection via timestamps | Watch evidence timestamps; no event > 5m → stuck → escalation |

### Synergies from OhMyOpenCode

| Pattern | Source | SDP Adoption |
|---------|--------|--------------|
| Pattern-based ACL | OMO PermissionConfig | `sdp guard rules --pattern` |
| Ask/Allow/Deny | OMO PermissionAction | Boundary gate escalation flow |
| Session permission overrides | OMO Session.permission | Per-session scope files |
| `primary_tools` restriction | OMO experimental | Agent role → allowed tools |

### Enterprise Value

"Your AI agent runs in a controlled environment with cryptographic proof of what it did. Compliance-ready."

---

## F060: Gas Town Adapter

**Phase:** 8-9 (K8s Orchestration Research / Pipeline Rebuild)
**LOC estimate:** ~800
**Repository:** sdp_lab (private) → subset to sdp (public)

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│           Gas Town (Orchestration Layer)                    │
│  Mayor │ Witness │ Deacon │ Hooks (worktree)               │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│           SDP Adapter Layer                                 │
│                                                             │
│  ┌─────────────────┐  ┌─────────────────┐                  │
│  │ gt-convoy-to-ws │  │ witness-escalate│                  │
│  │ (convoy sync)   │  │ (stuck → esc)   │                  │
│  │                 │  │                 │                  │
│  │ • convoy → WS   │  │ • stuck detect  │                  │
│  │ • hook → guard  │  │ • escalation    │                  │
│  └─────────────────┘  └─────────────────┘                  │
│                                                             │
│  Output: SDP workstreams from Gas Town convoys             │
└─────────────────────────────────────────────────────────────┘
```

### Workstreams

| WS | Title | AC |
|----|-------|----|
| **00-060-01** | Convoy → Workstream bridge | `gt convoy list` → generates WS files in backlog/ |
| **00-060-02** | Hook → Guard scope bridge | Gas Town worktree hook → SDP guard scope file |
| **00-060-03** | Witness → Escalation bridge | Stuck agent → Beads wisp (ephemeral escalation issue) |
| **00-060-04** | Agent CV → Provenance | Gas Town CV chain → SDP provenance.agent_cv field |

### Synergies from Gas Town

| Pattern | Source | SDP Adoption |
|---------|--------|--------------|
| **GUPP** (propulsion) | Gas Town | "If work on hook, must execute" → pre-tool-call enforcement |
| Git worktree hooks | Gas Town | Scope persistence via worktree `.sdp/guard-scope.json` |
| Witness monitoring | Gas Town Deacon | Stuck detection + escalation |
| Agent CV chain | Gas Town | Capability routing in provenance section |
| Mail vs Nudge | Gas Town | Persistent mail → evidence, nudge → ephemeral |
| MEOW workflow | Gas Town | Mayor → Convoy → Agent handoff pattern |

### Enterprise Value

"Scale to 20-30 agents with governance. Witness monitors for stuck agents. Agent CVs enable capability-based routing."

---

## F061: Beads Graph Integration

**Phase:** 5 (Policy-as-Code)
**LOC estimate:** ~600
**Repository:** sdp (public)

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│           Beads (Issue Tracking Layer)                      │
│  Dolt DB │ Dependency Graph │ Formulas │ Molecules         │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│           SDP Bridge Layer (ALREADY PARTIALLY EXISTS)       │
│                                                             │
│  ┌─────────────────┐  ┌─────────────────┐                  │
│  │ beads-bridge    │  │ dep-graph-sync  │                  │
│  │ (CronJob)       │  │ (SQL → WS)      │                  │
│  │                 │  │                 │                  │
│  │ • bd ready → WS │  │ • blocks → deps │                  │
│  │ • status sync   │  │ • parent-child  │                  │
│  └─────────────────┘  └─────────────────┘                  │
│                                                             │
│  Output: SDP workstreams synced with Beads graph           │
└─────────────────────────────────────────────────────────────┘
```

### Workstreams

| WS | Title | AC |
|----|-------|----|
| **00-061-01** | SQL dependency query | `SELECT ready FROM issues WHERE no blockers` → `sdp ready` |
| **00-061-02** | `bd ready` → `sdp ready` bridge | Command wrapper that calls Beads SQL and formats WS output |
| **00-061-03** | Formula → Workstream template | Beads formula TOML → WS frontmatter generator |
| **00-061-04** | Wisps → Ephemeral work items | Beads wisps → SDP session-only items (not committed) |

### Synergies from Beads

| Pattern | Source | SDP Adoption |
|---------|--------|--------------|
| 4 dependency types | Beads | blocks, parent-child, discovered-from, related |
| `bd ready` queue | Beads | Transitive blocking detection for workstreams |
| Formulas | Beads | Declarative WS templates with variable substitution |
| Wisps (ephemeral) | Beads | Session-only work items that auto-expire |
| Molecules | Beads | Multi-step WS with phase transitions |
| Content hashing | Beads | ComputeContentHash for dedup |
| Audit log | Beads `interactions.jsonl` | Append-only typed entries |

### Enterprise Value

"Dependency-aware execution. Agents only see work that's actually ready. Formula templates for repeatable workflows."

---

## Integration Priority Matrix

| Feature | Source | SDP Value | Phase | Priority |
|---------|--------|-----------|-------|----------|
| Pre-tool-call guard | OhMyOpenCode + Gas Town | Enforcement | 5 | **P0** |
| Session evidence | OhMyOpenCode | Provenance | 5 | **P0** |
| `bd ready` bridge | Beads | Work intake | 5 | **P1** |
| Stuck detection | Gas Town | Reliability | 6 | **P1** |
| Agent CV chain | Gas Town | Capability routing | 8 | **P2** |
| Convoy → WS | Gas Town | Batch tracking | 8 | **P2** |
| Formula templates | Beads | Reusability | 6 | **P2** |
| Wisps | Beads | Session isolation | 7 | **P3** |

---

## Enterprise Tiers

### Tier 1: Evidence Compliance (SOC2/DORA)

```
┌─────────────────────────────────────────────────────────────┐
│               SDP Evidence Compliance Pack                  │
│                                                             │
│  • in-toto attestations for every PR                       │
│  • Sigstore signing (keyless OIDC)                         │
│  • OPA/Rego policies for enforcement                       │
│  • Audit trail export (JSONL → SIEM)                       │
│  • Evidence retention policy                               │
│                                                             │
│  Value: "Pass your audit with AI agent evidence"           │
│  Price: Per-seat licensing                                 │
└─────────────────────────────────────────────────────────────┘
```

### Tier 2: Multi-Agent Governance

```
┌─────────────────────────────────────────────────────────────┐
│               SDP Multi-Agent Governance Pack               │
│                                                             │
│  • Gas Town adapter (20-30 agents)                         │
│  • Witness monitoring + stuck detection                    │
│  • Agent CV + capability routing                           │
│  • Cross-rig convoy tracking                               │
│  • Budget + model policy enforcement                       │
│                                                             │
│  Value: "Scale AI agents with governance"                  │
│  Price: Infrastructure licensing                           │
└─────────────────────────────────────────────────────────────┘
```

### Tier 3: Autonomous Swarm (K8s)

```
┌─────────────────────────────────────────────────────────────┐
│               SDP Autonomous Swarm Pack                     │
│                                                             │
│  • K8s pipeline (analyst → coder → reviewer)               │
│  • Kyverno admission enforcement                           │
│  • Tekton Chains auto-attestation                          │
│  • Beads-bridge for issue intake                           │
│  • Self-improvement loop (private)                         │
│                                                             │
│  Value: "Issue in, PR with proof out — fully autonomous"   │
│  Price: Enterprise license + support                       │
└─────────────────────────────────────────────────────────────┘
```

---

## What Goes Where (OSS vs Private)

### OSS (sdp public repo)

| Component | Why |
|-----------|-----|
| `sdp-evidence` | Core protocol |
| `sdp-guard` | Core protocol |
| `sdp-omc-adapter` | Ecosystem integration |
| `sdp-beads-bridge` | Ecosystem integration |
| Predicate type spec | Standard |
| OPA policies | Best practices |

### Private (sdp_lab)

| Component | Why |
|-----------|-----|
| `gt-adapter` | Competitive moat |
| `witness-monitor` | Enterprise feature |
| `self-improve/` | Competitive advantage |
| `brain-gateway` | Customer-specific |
| Enterprise policy packs | Compliance liability |
| Commercial automation | Business logic |

---

## Key References

### External Projects

- [OhMyOpenCode](https://github.com/oh-my-opencode) — Permission-gated agent runtime
- [Gas Town](https://github.com/steveyegge/gastown) — Multi-agent orchestration with GUPP
- [Beads](https://github.com/steveyegge/beads) — Git-backed issue tracker with dependency graph

### SDP Documents

- [ROADMAP.md](../roadmap/ROADMAP.md) — Feature schedule
- [MANIFESTO.md](../MANIFESTO.md) — Vision
- [PRIVATE_BLUEPRINT.md](../PRIVATE_BLUEPRINT.md) — Enterprise architecture
- [ADR-002](../decisions/ADR-002-standards-pivot.md) — Why standards

---

*"AI agents can implement features, but without evidence it's just vibes. With ecosystem integration, it's governance."*
