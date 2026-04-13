# SDP Skill Architecture: Intent Routing & Tool Composition

> **For Claude:** This is a design document, not an implementation plan.

**Goal:** Replace 26+ flat skills with 5-7 intent-based skills that compose CLI tools, eliminating cognitive overload while scaling tool count freely.

**Architecture:** Intent routing (Option B) — agent picks intent, skill picks tools. CLI tools scale linearly, skills scale logarithmically.

**Parent design:** `2026-04-13-sdp-toolkit-vision-design.md`

---

## Problem Statement

Skills are overloaded. One word covers four fundamentally different things:

| Type | Example | Shape |
|------|---------|-------|
| **Tool wrapper** | @scout, @architect, @metrics | CLI → JSON → interpretation |
| **Workflow** | @feature, @brainstorming | Multi-step, human-in-the-loop |
| **Policy** | @review, @guard, @tdd | Constraints, quality gates |
| **Orchestrator** | @landscape, @bootstrap | Composes tools + workflows |

Result: 26 skills, growing. Agent doesn't know which to pick. User doesn't remember what's available. Every new capability = new skill = more confusion.

## Core Insight

**Tools scale linearly, skills should scale logarithmically.**

Adding `sdp metrics` as a CLI tool costs nothing — it's just another command. But adding `@metrics` as a skill competes for attention with 25 others. The fix: decouple tools from skills.

```
Before:  1 capability = 1 skill    → 26 skills, growing
After:   1 capability = 1 tool     → 6 tools via 5 skills
```

## The Five Intents

```
┌─────────────────────────────────────────────────────┐
│                  Agent / Human                       │
│                                                      │
│  "I want to..."                                      │
│                                                      │
│  understand ─── build ─── fix ─── review ─── operate │
│                                                      │
└──────┬──────────┬────────┬────────┬──────────┬──────┘
       │          │        │        │          │
       v          v        v        v          v
   ┌───────┐ ┌────────┐ ┌─────┐ ┌───────┐ ┌────────┐
   │scout  │ │feature │ │debug│ │policy │ │deploy  │
   │archi- │ │tdd     │ │fix  │ │review │ │ci-triage│
   │tect   │ │guard   │ │     │ │       │ │monitor │
   │metrics│ │        │ │     │ │       │ │        │
   │spec   │ │        │ │     │ │       │ │        │
   │index  │ │        │ │     │ │       │ │        │
   └───────┘ └────────┘ └─────┘ └───────┘ └────────┘
    6 tools   3 tools    2 tools  2 tools   3 tools
```

### 1. @understand — "What is this codebase?"

**Absorbs:** @scout, @architect, @metrics, @spec, @landscape, @index query

**Modes:**
- **quick** (30s): `sdp scout` only → project card
- **standard** (5-15 min): scout + architect + metrics → full picture
- **deep** (15-30 min): + spec + index build → complete knowledge base

**The skill decides which tools to call based on:**
- What's already available (scout.json exists? skip scout)
- What the user asked ("how healthy is this repo?" → scout + metrics, skip spec)
- Time budget ("quick look" → scout only)

**Replaces @landscape:** The "standard" and "deep" modes ARE landscape. No need for a separate meta-skill — @understand at standard/deep mode synthesizes everything @landscape would.

```
User: "@understand this repo"
Skill: → sdp scout . (30s)
       → reads scout.json, decides: Go project, 48K LOC, active
       → sdp architect analyze . (5 min)
       → sdp metrics . (3 min)
       → synthesizes: report with architecture + health + risks
       → writes .sdp/manifest.md (context primer for future sessions)
```

### 2. @build — "Create something new"

**Absorbs:** @feature, @idea, @design, @ux, @vision, @oneshot, @prototype

**Modes:**
- **idea** (brainstorm): problem → design doc
- **feature** (full cycle): idea → design → implement → test → PR
- **prototype** (fast): skip design, build quickly, mark as prototype

**The skill decides the scope based on:**
- What's being built ("add button" → prototype, "new auth system" → feature)
- Available context (has design doc? skip to implement)
- User preference ("just build it" → prototype mode)

### 3. @fix — "Something is broken"

**Absorbs:** @hotfix, @bugfix, @issue, @debug

**Modes:**
- **quick** (hotfix): known cause, minimal change, instant PR
- **investigate** (debug): unknown cause, needs diagnosis
- **systematic** (issue): known issue, needs planned fix

**Severity as parameter, not separate skill:**
```
@fix --severity critical "production 500 on /api/users"
@fix "flaky test in CI"
@fix --issue PROJ-123
```

### 4. @review — "Is this good enough?"

**Absorbs:** @review (all 6 reviewer roles), @reality-check, @verify-workstream

**Roles as parameters:**
```
@review                        # default: code review
@review --arch                 # architecture review
@review --security             # security review
@review --readiness            # release readiness
```

**Key change:** One skill, one entry point. The skill decides which review dimensions to apply based on the diff size, risk profile, and what changed.

### 5. @operate — "Keep it running"

**Absorbs:** @deploy, @ci-triage, @plan

**Modes:**
- **deploy**: release preparation and execution
- **triage**: CI failure diagnosis
- **plan**: convert insights into backlog (replaces standalone @plan)

## Intent Detection Algorithm

How does the system pick the right intent?

```
Input: user message + conversation context + .sdp/ state
       │
       ├─ Keyword signals:
       │   "what is" / "analyze" / "understand" / "how does" → @understand
       │   "build" / "add" / "create" / "implement"         → @build
       │   "fix" / "bug" / "broken" / "error" / "failing"   → @fix
       │   "review" / "check" / "approve" / "ready"         → @review
       │   "deploy" / "release" / "ci" / "plan" / "backlog" → @operate
       │
       ├─ Context signals:
       │   No .sdp/scout.json exists   → @understand (first)
       │   PR open, changes staged     → @review
       │   CI red                      → @operate (triage)
       │   Beads issue assigned        → @fix or @build (from type)
       │
       └─ Explicit override:
           "@understand --deep"  → force specific intent + mode
```

## What Stays as Standalone

Some things are NOT skills — they're **policies** or **practices** that apply across all intents:

| Current Skill | Becomes | Why |
|---------------|---------|-----|
| @tdd | Practice: applies within @build and @fix | Every feature/fix follows TDD |
| @guard | Policy: pre-commit check | Runs automatically via hooks |
| @go-modern | Convention: embedded in CLAUDE.md | Language style, not a workflow |
| @think | Prompt technique: use everywhere | "Think step by step" isn't a skill |
| @beads | Tool: `bd` commands | Issue tracker, not a skill |
| @protocol-consistency | Policy: CI check | Automated gate, not invoked |

This eliminates 6+ "skills" that aren't really skills.

## Skill Count: Before and After

```
BEFORE (26 skills):
  Discovery:  discovery, idea, ux, design, vision                  (5)
  Analysis:   architect, reality, reality-check                    (3)
  Execution:  build, oneshot, feature                              (3)
  Fixes:      hotfix, bugfix, issue                                (3)
  Ops:        deploy, ci-triage                                    (2)
  Quality:    review, debug                                        (2)
  Internal:   beads, tdd, guard, think, go-modern,                 (8)
              protocol-consistency, verify-workstream, prototype

AFTER (5 intents + practices):
  Intents:    @understand, @build, @fix, @review, @operate         (5)
  Practices:  tdd (embedded), guard (hook), go-modern (CLAUDE.md)  (0 skills)
  Tools:      scout, architect, metrics, index, spec, bootstrap    (CLI, not skills)

  Total skills: 5
  Total CLI tools: 6+
```

**From 26 to 5.** Tools scale freely. Skills stay fixed.

## Interaction with MCP

Intent routing maps cleanly to MCP primitives:

```
MCP Server: sdp
  ├── Tools (CLI wrappers):
  │   sdp_scout, sdp_architect, sdp_metrics,
  │   sdp_index_query, sdp_spec, sdp_bootstrap
  │
  ├── Resources (data files):
  │   manifest.md, scout.json, report.json,
  │   metrics.json, index.db (via query)
  │
  └── Prompts (intents):
      understand, build, fix, review, operate
```

- **Tools** = the CLI commands, callable by any MCP client
- **Resources** = .sdp/ directory contents, readable by any client
- **Prompts** = the 5 intents, with their routing logic and mode selection

See `2026-04-13-sdp-mcp-design.md` for full MCP server design.

## Migration Path

How to get from 26 skills to 5 intents:

### Phase 1: Tool Extraction
- Extract CLI tools from skill logic (scout, metrics, spec already designed)
- Skills become thin wrappers: call CLI → interpret JSON → format output
- No user-facing change yet

### Phase 2: Intent Consolidation
- Merge related skills under intents (@hotfix + @bugfix + @issue → @fix)
- Old skill names become aliases that route to new intent
- Deprecation warnings for direct old-skill invocation

### Phase 3: MCP Server
- Expose tools as MCP tools
- Expose intents as MCP prompts
- Skills become the prompt definitions in MCP
- Any MCP client gets full SDP capability

### Phase 4: Retire Flat Skills
- Remove old skill files
- All interaction through intents or direct tool calls
- Skills directory: 5 files instead of 26

## Design Decisions

1. **5 intents, not 3, not 10.** Five covers the development lifecycle without gaps or overlaps: understand → build → fix → review → operate.

2. **Tools are CLI, not skills.** CLI tools have clear contracts (input → JSON output), are testable, and work outside any AI context. Skills add interpretation.

3. **Practices, not skills.** TDD is how you build, not what you invoke. It's embedded in @build and @fix behavior, not a standalone thing.

4. **Modes, not variants.** `@fix --severity critical` not `@hotfix`. One entry point, parameterized behavior.

5. **Progressive depth.** @understand quick/standard/deep — user controls time investment, skill controls tool selection.
