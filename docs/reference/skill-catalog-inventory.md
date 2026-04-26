# Skill Catalog Inventory

**Feature:** F138 (Skill Catalog Normalization)
**Status:** Active Inventory (2026-04-26)
**Scope:** All skill directories across harnesses

## Purpose

This document provides a complete inventory of all skills across all harness directories and the canonical merge map for cleanup. It preserves the F125 intent model while identifying duplicates, legacy aliases, and consolidation targets.

## Directory Structure

```
.agents/skills/       - Canonical skill definitions (45 files)
prompts/skills/       - Legacy prompts/ skills (30 directories with SKILL.md)
.codex/skills/        - Codex harness skills (30 files)
.opencode/skill/      - OpenCode harness skills (30 files)
.claude/skills/       - Empty (no skills found)
```

## F125 Intent Model (PRESERVE)

The following **5 intent-based skills** are the canonical active surface and must be preserved:

| Skill | Description | Status |
|-------|-------------|--------|
| `@understand` | Discovery, architecture, health, documentation | **ACTIVE - KEEP** |
| `@build` | Feature creation (idea/feature/prototype modes) | **ACTIVE - KEEP** |
| `@fix` | Bug resolution (quick/investigate/systematic modes) | **ACTIVE - KEEP** |
| `@review` | Quality gates (code/architecture/security/readiness) | **ACTIVE - KEEP** |
| `@operate` | Operations (deploy/triage/plan modes) | **ACTIVE - KEEP** |
| `@ship` | Deployment and release command | **ACTIVE - KEEP** |

## Skill Inventory and Merge Map

### Legend

- **KEEP** - Active canonical skill, preserve as-is
- **MERGE-INTO** - Duplicate that should redirect to target
- **DEPRECATE** - Legacy alias with explicit deprecation behavior
- **REMOVE** - Obsolete, safe to remove
- **SPECIALIZED** - Outside intent model, keep as-is

### .agents/skills/ Inventory (45 files)

| File | Action | Target | Notes |
|------|--------|--------|-------|
| `understand.md` | **KEEP** | - | F125 intent: discovery |
| `build.md` | **KEEP** | - | F125 intent: feature creation |
| `fix.md` | **KEEP** | - | F125 intent: bug resolution |
| `review.md` | **KEEP** | - | F125 intent: quality gates |
| `operate.md` | **KEEP** | - | F125 intent: operations |
| `ship.md` | **KEEP** | - | Deployment command |
| `strataudit.md` | **SPECIALIZED** | - | Strategy traceability audit |
| `git-worktree.md` | **SPECIALIZED** | - | Git worktree setup |
| `parallel-dispatch.md` | **SPECIALIZED** | - | Parallel subagent dispatch |
| `llm-council.md` | **SPECIALIZED** | - | Multi-model synthesis |
| `spec-interrogate.md` | **SPECIALIZED** | - | Spec interrogation capability |
| `scout.md` | **DEPRECATE** | `@understand --depth quick` | Legacy: quick discovery |
| `architect.md` | **DEPRECATE** | `@understand --depth standard` | Legacy: architecture analysis |
| `metrics.md` | **DEPRECATE** | `@understand --depth standard` | Legacy: metrics collection |
| `landscape.md` | **DEPRECATE** | `@understand --depth standard` | Legacy: codebase landscape |
| `feature.md` | **DEPRECATE** | `@build --mode feature` | Legacy: feature implementation |
| `idea.md` | **DEPRECATE** | `@build --mode idea` | Legacy: brainstorming |
| `design.md` | **DEPRECATE** | `@build --mode idea` | Legacy: design work |
| `ux.md` | **DEPRECATE** | `@build --mode idea` | Legacy: UX design |
| `vision.md` | **DEPRECATE** | `@build --mode idea` | Legacy: vision work |
| `prototype.md` | **DEPRECATE** | `@build --mode prototype` | Legacy: prototyping |
| `oneshot.md` | **DEPRECATE** | `@build --mode prototype` | Legacy: one-shot implementation |
| `hotfix.md` | **DEPRECATE** | `@fix --mode quick` | Legacy: hotfix |
| `bugfix.md` | **DEPRECATE** | `@fix --mode systematic` | Legacy: bugfix |
| `issue.md` | **DEPRECATE** | `@fix --mode systematic` | Legacy: issue tracking |
| `debug.md` | **DEPRECATE** | `@fix --mode investigate` | Legacy: debugging |
| `reality-check.md` | **DEPRECATE** | `@review --dimension reality` | Legacy: reality checking |
| `verify-workstream.md` | **DEPRECATE** | `@review --dimension readiness` | Legacy: workstream verification |
| `deploy.md` | **DEPRECATE** | `@ship` | Legacy: deployment |
| `ci-triage.md` | **DEPRECATE** | `@operate --mode triage` | Legacy: CI triage |
| `plan.md` | **DEPRECATE** | `@operate --mode plan` | Legacy: planning |
| `delivery-loop.md` | **SPECIALIZED** | - | Autonomous delivery cycle |
| `README.md` | **KEEP** | - | Documentation |

### Consolidation Summary

**Total Active Skills: 11**

### Active Canonical Skills (11 total)

**F125 Intents (6):**
- `@understand` - Discovery
- `@build` - Feature creation
- `@fix` - Bug resolution
- `@review` - Quality gates
- `@operate` - Operations
- `@ship` - Deployment

**Specialized Skills (5):**
- `@strataudit` - Strategy traceability audit
- `@git-worktree` - Git worktree setup
- `@parallel-dispatch` - Parallel subagent dispatch
- `@llm-council` - Multi-model synthesis
- `@spec-interrogate` - Spec interrogation

**CLI Tools / Practices (4 - not skills):**
- `@beads` - Issue tracker (CLI tool)
- `@init` - SDP initialization
- `@delivery-loop` - Autonomous delivery cycle
- `@protocol-consistency` - Protocol consistency audit

## Implementation Notes

- Legacy skill files in `.agents/skills/` should be updated with deprecation notices
- `prompts/skills/`, `.codex/skills/`, and `.opencode/skill/` should mirror the canonical catalog
- Internal implementation files (phases, gates) should be moved to internal directories
- Documentation should reference only the canonical 11 skills

## References

- `docs/reference/skills.md` - Canonical skills reference
- `docs/reference/migration-guide.md` - F125 migration guide
- `docs/plans/2026-04-13-sdp-skill-architecture-design.md` - F125 intent model design
- `.agents/skills/index.json` - Machine-readable catalog
