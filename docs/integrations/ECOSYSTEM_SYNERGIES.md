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

"Dependency-aware execution. Agents only see work that's actually ready. Formula templates for repeatable workflows."

---

## F062: vibe-kanban Integration

**Phase:** 8-9 (K8s Orchestration)
**LOC estimate:** ~1,000
**Repository:** sdp_lab (private)

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│           vibe-kanban (Task Orchestration)                  │
│  Kanban Board │ MCP Config │ Agent Router                  │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│           SDP Bridge Layer                                 │
│                                                             │
│  ┌─────────────────┐  ┌─────────────────┐                  │
│  │ sdp-kanban-bridge│  │ task-evidence   │                  │
│  │ (Beads sync)    │  │ (completion)    │                  │
│  └─────────────────┘  └─────────────────┘                  │
│                                                             │
│  Output: Multi-agent coordination with evidence            │
└─────────────────────────────────────────────────────────────┘
```

### Workstreams

| WS | Title | AC |
|----|-------|----|
| **00-062-01** | Analyze vibe-kanban architecture | Done |
| **00-062-02** | Design SDP ↔ vibe-kanban bridge | Architecture doc |
| **00-062-03** | Implement orchestration layer | K8s deployment |

### Synergies from vibe-kanban

| Pattern | Source | SDP Adoption |
|---------|--------|--------------|
| Kanban board | vibe-kanban | Visual task management for agents |
| MCP centralization | vibe-kanban | Unified agent configuration |
| Agent routing | vibe-kanban | Task → best agent matching |

### Enterprise Value

"Visual orchestration for 20+ agents. Kanban board shows real-time task status. MCP config centralized."

---

## F063: opencode-mem Memory Module

**Phase:** 5 (Policy-as-Code)
**LOC estimate:** ~400 (config + integration)
**Repository:** sdp (public adapter) + sdp_lab (private config)

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│           opencode-mem (Persistent Memory)                  │
│  SQLite + HNSW │ User Profiles │ Auto-Capture              │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│           SDP Integration Layer                             │
│                                                             │
│  ┌─────────────────┐  ┌─────────────────┐                  │
│  │ memory-context  │  │ session-sync   │                  │
│  │ (prompt inject) │  │ (lifecycle)    │                  │
│  └─────────────────┘  └─────────────────┘                  │
│                                                             │
│  Output: Session continuity + user preference learning     │
└─────────────────────────────────────────────────────────────┘
```

### Workstreams

| WS | Title | AC |
|----|-------|----|
| **00-063-01** | Analyze opencode-mem capabilities | Done (9/10 synergy) |
| **00-063-02** | Install opencode-mem plugin | Plugin in opencode.json |
| **00-063-03** | Configure memory for SDP sessions | Context injection working |
| **00-063-04** | Integrate with evidence flow | Memory events in evidence |

### Synergies from opencode-mem

| Pattern | Source | SDP Adoption |
|---------|--------|--------------|
| Session continuity | opencode-mem | Context preserved across sessions |
| User profile learning | opencode-mem | Preferences learned from behavior |
| Project-scoped memory | opencode-mem | Per-project context |
| Auto-capture on idle | opencode-mem | Learnings captured automatically |
| Post-compaction recovery | opencode-mem | Memory restored after context trim |

### Key Capabilities

- **SQLite + HNSW**: Local-first vector search, 50K vectors/shard
- **12+ embedding models**: Local (Xenova) or API (OpenAI, Cohere)
- **User profiles**: Preferences with confidence scoring
- **Project metadata**: Auto-includes gitRepoUrl, projectPath

### Enterprise Value

"Session continuity for AI agents. User preferences learned automatically. Project context preserved across sessions."

---

| Feature | Source | SDP Value | Phase | Priority |
|---------|--------|-----------|-------|----------|
| Pre-tool-call guard | OhMyOpenCode + Gas Town | Enforcement | 5 | **P0** |
| Session evidence | OhMyOpenCode | Provenance | 5 | **P0** |
| `bd ready` bridge | Beads | Work intake | 5 | **P1** |
| Persistent memory | opencode-mem | Session continuity | 5 | **P1** |
| User profile learning | opencode-mem | Personalization | 5 | **P1** |
| Stuck detection | Gas Town | Reliability | 6 | **P1** |
| Agent CV chain | Gas Town | Capability routing | 8 | **P2** |
| Convoy → WS | Gas Town | Batch tracking | 8 | **P2** |
| Kanban orchestration | vibe-kanban | Multi-agent coordination | 8 | **P2** |
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
- [vibe-kanban](https://github.com/BloopAI/vibe-kanban) — Kanban-style task orchestration for coding agents
- [opencode-mem](https://github.com/tickernelz/opencode-mem) — Persistent memory for AI coding agents
- [opencode-beads](https://github.com/joshuadavidthomas/opencode-beads) — Beads plugin for OpenCode

### SDP Documents

- [ROADMAP.md](../roadmap/ROADMAP.md) — Feature schedule
- [MANIFESTO.md](../MANIFESTO.md) — Vision
- [PRIVATE_BLUEPRINT.md](../PRIVATE_BLUEPRINT.md) — Enterprise architecture
- [ADR-002](../decisions/ADR-002-standards-pivot.md) — Why standards

---

*"AI agents can implement features, but without evidence it's just vibes. With ecosystem integration, it's governance."*
