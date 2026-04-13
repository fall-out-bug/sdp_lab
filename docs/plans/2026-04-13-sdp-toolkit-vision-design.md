# SDP Toolkit Vision: From Unknown Codebase to AI-Native Development

> **For Claude:** This is a design document, not an implementation plan. It describes the full toolkit architecture. Individual implementation plans reference this document.

**Goal:** Define the complete SDP toolkit pipeline that takes an unknown codebase and prepares it for AI-native development by humans and agents.

**Status:** Design — approved by brainstorming session 2026-04-13

---

## North Star

When you arrive at a new codebase, people should understand what's happening, and agents should start working on it in AI-native mode. The toolkit produces this understanding through a pipeline of composable tools.

## Pipeline Overview

```
 Unknown                                                         AI-Native
 Codebase ──────────────────────────────────────────────────── Development
      │                                                            │
      ▼                                                            ▼
  ┌────────┐  ┌───────────┐  ┌─────────┐  ┌───────────┐  ┌──────────┐
  │ scout  │→ │ architect │→ │ metrics │→ │ bootstrap │→ │ activate │
  │        │  │           │  │         │  │           │  │          │
  │ What   │  │ How is it │  │ How     │  │ Set up    │  │ Start    │
  │ is it? │  │ built?    │  │ alive?  │  │ for agents│  │ working  │
  └────────┘  └───────────┘  └─────────┘  └───────────┘  └──────────┘
    30 sec      5-15 min       2-5 min      1-3 min        ongoing
```

Cross-cutting: **`sdp index`** builds and maintains persistent codebase memory used by all components.

## Components

### 1. @scout — Quick Reconnaissance (30 seconds)

**What:** Fast assessment — language, size, build system, README summary, "what is this".
**Status:** Design complete. See `2026-04-13-sdp-scout-design.md`.
**Output:** stdout summary, optionally `.sdp/scout.json`.
**CLI:** `sdp scout <repo>`

### 2. @architect — Architecture Analysis (v7.2.0, production)

**What:** Deep architectural analysis, C4 diagrams, patterns, risks, tech debt.
**Status:** Production. 21K LOC, 49 files, 9 extractors, HTML renderer.
**Output:** `.sdp/architecture/report.json`, markdown report, HTML report.
**CLI:** `sdp architect analyze <repo>`
**Skill:** `@architect` (v7.2.0)

### 3. @metrics — Process & Code Health

**What:** Git-derived metrics across 7 categories: commit hygiene, wasted work, git flow detection, release quality, release stabilization, knowledge risk, code decay.
**Status:** Design complete. See `2026-04-13-sdp-metrics-design.md`.
**Output:** `.sdp/metrics/report.json` (Go CLI), markdown report (skill).
**CLI:** `sdp metrics <repo>`
**Skill:** `@metrics` (new)

**Architecture:**
- Go CLI produces structured JSON from git history (4 git commands, 7 parallel analyzers)
- Skill interprets JSON into narrative markdown with traffic-light ratings
- Meta-skill combines with architect for full picture

**Data flow:**
```
git log --numstat (single pass)
git tag --sort=creatordate
git branch -r
git log --merges --first-parent main
        │
   7 parallel analyzers → MetricsReport JSON
        │
   Skill interprets → Markdown with 🟢🟡🔴 per category
```

### 4. @spec — Specification Recovery

**What:** Extract implicit specifications from code: API contracts, business rules, invariants.
**Status:** Design complete. See `2026-04-13-sdp-spec-design.md`.
**Output:** `.sdp/specs/` — recovered OpenAPI, business rules, invariants.
**CLI:** `sdp spec <repo>`

### 5. @index — Codebase Memory & Indexing

**What:** Persistent, queryable index of codebase knowledge for agent context.
**Status:** Design complete. See `2026-04-13-sdp-index-design.md`.
**Output:** `.sdp/index.db` (sqlite-vec + FTS5), `.sdp/manifest.md` (context primer).
**CLI:** `sdp index build|refresh|query|manifest|deps|find`
**Three levels:** Repo (MVP) → Multi-repo → Organization.

### 6. @bootstrap — Agent-Ready Setup

**What:** Generate CLAUDE.md, AGENTS.md, configure beads, hooks, policies from analysis data.
**Status:** Design complete. See `2026-04-13-sdp-bootstrap-design.md`.
**Output:** CLAUDE.md, hooks, beads init, `.sdp/policies/`.
**CLI:** `sdp bootstrap <repo>`

### 7. @landscape → folded into @understand intent

**What:** Was: meta-skill orchestrating architect + metrics + index.
**Status:** Folded into `@understand --depth standard|deep`. See `2026-04-13-sdp-skill-architecture-design.md`.
**Rationale:** Separate meta-skill adds cognitive load. @understand with depth parameter does the same job.

### 8. @plan → folded into @operate intent

**What:** Was: convert analysis insights into prioritized beads issues.
**Status:** Folded into `@operate --mode plan`. See `2026-04-13-sdp-skill-architecture-design.md`.
**Rationale:** Planning is an operational activity, not a standalone skill.

### 9. MCP Server — Universal Agent Interface

**What:** Single MCP server exposing all SDP tools, resources, and intent prompts to any AI agent.
**Status:** Design complete. See `2026-04-13-sdp-mcp-design.md`.
**Output:** `sdp-mcp` binary (stdio transport).
**Config:** One JSON per harness (Claude Code, Cursor, VS Code, OpenCode).

## Skills Rationalization

See `2026-04-13-sdp-skill-architecture-design.md` for full design.

**Before (26 skills):**
```
Discovery:  discovery, idea, ux, design, vision          (5)
Analysis:   architect, reality, reality-check             (3)
Execution:  build, oneshot, feature                       (3)
Fixes:      hotfix, bugfix, issue                         (3)
Ops:        deploy, ci-triage                             (2)
Quality:    review, debug                                 (2)
Internal:   beads, tdd, guard, think, go-modern,          (8)
            protocol-consistency, verify-workstream,
            prototype
```

**After (intent routing — Option B):**
```
5 INTENTS (agent-facing skills):
  @understand — uses: scout, architect, metrics, spec, index (modes: quick|standard|deep)
  @build      — uses: feature, tdd, guard (modes: idea|feature|prototype)
  @fix        — uses: debug, hotfix (modes: quick|investigate|systematic)
  @review     — uses: policy checks (modes: code|arch|security|readiness)
  @operate    — uses: deploy, ci-triage, plan (modes: deploy|triage|plan)

6+ CLI TOOLS (scale freely, not skills):
  sdp scout, sdp architect, sdp metrics, sdp index, sdp spec, sdp bootstrap

PRACTICES (embedded, not skills):
  tdd (in @build/@fix), guard (in hooks), go-modern (in CLAUDE.md)

MCP SERVER (universal interface):
  sdp-mcp — exposes tools + resources + prompts to any AI agent
```

**From 26 skills to 5 intents.** Tools scale freely via MCP.

## Three-Layer Memory Model

```
┌─────────────────────────────────────────────────────┐
│ Layer 1: MANIFEST (always in context, ≤2K tokens)   │
│ .sdp/manifest.md — auto-generated                   │
│ What: name, language, size, arch style, build,       │
│       modules list, git flow, team size              │
│ Updates: post-commit hook or sdp index refresh       │
│ Loaded: automatically via CLAUDE.md include          │
├─────────────────────────────────────────────────────┤
│ Layer 2: INDEX (queried on demand, ≤50K tokens)     │
│ .sdp/index.db — structured, sqlite-vec + FTS5       │
│ What: module→purpose→files→owner,                    │
│       API endpoints, key abstractions,               │
│       conventions, "where to find X"                 │
│ Updates: sdp index refresh / post-merge hook         │
│ Access: sdp index query "how does executor work"     │
├─────────────────────────────────────────────────────┤
│ Layer 3: FULL REPORTS (deep context)                │
│ .sdp/architecture/ — architect output                │
│ .sdp/metrics/      — metrics output                  │
│ .sdp/specs/        — recovered specs                 │
│ Updates: on demand (sdp architect / metrics / spec)  │
│ Access: agent reads when depth needed                │
└─────────────────────────────────────────────────────┘
```

## Implementation Priority

| # | Component | Strategy | Effort | Dependencies |
|---|-----------|----------|--------|-------------|
| 1 | **@metrics** | Build from scratch (Go CLI + skill) | Large | None |
| 2 | **@index** | Build from scratch (Go CLI) | Large | None |
| 3 | **@scout** | Extract FRAME from `sdp discover` | Small | None |
| 4 | **@spec** | Extend `@reality` skill | Medium | @index (optional) |
| 5 | **@bootstrap** | Extend `sdp-up` | Medium | @architect, @metrics, @index |
| 6 | **@landscape** | New meta-skill | Medium | @architect, @metrics |
| 7 | **@plan** | Extend planner stub | Medium | @landscape, beads |

@metrics and @index can be built in parallel — no dependencies between them.

## JSON Contracts

Each component communicates through well-defined JSON contracts stored in `.sdp/`:

- `scout.json` — quick facts (language, size, build system)
- `architecture/report.json` — full architect output (exists)
- `metrics/report.json` — 7-category process health metrics
- `index.db` — sqlite database with vec + FTS5
- `manifest.md` — human/agent-readable context primer
- `specs/` — recovered specifications

All JSON schemas live in `sdp/schemas/` (public protocol repo).

## Relationship to Existing Systems

- **beads** — issue tracking. @plan creates beads from insights. @bootstrap initializes beads.
- **superpowers** — development workflows. Used AFTER bootstrap for actual development.
- **sdp discover** — existing discovery pipeline. @scout extracts its FRAME phase.
- **sdp dispatch** — task routing. Uses @index for context when routing tasks.
- **sdp orchestrate** — phase orchestration. @landscape feeds its decision-making.

## Design Principles

1. **Each tool works standalone.** No forced ordering. @metrics works without @architect.
2. **JSON between layers.** Go produces data, skills interpret, meta-skill synthesizes.
3. **Incremental enrichment.** Each additional tool enriches the picture but isn't required.
4. **Offline-first.** Everything works without cloud APIs. Cloud is optional enhancement.
5. **One file per index.** `.sdp/index.db` — no external databases, no running servers.
