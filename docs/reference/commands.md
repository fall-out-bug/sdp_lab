# SDP Commands Reference

Complete reference for all SDP CLI commands and skills.

---

## Table of Contents

- [Intent-Based Commands](#intent-based-commands)
- [Utility Commands](#utility-commands)
- [Internal Commands](#internal-commands)
- [Command Options](#command-options)
- [Deprecated Aliases](#deprecated-aliases)

---

## Intent-Based Commands

SDP v1.0+ is organized around 5 core intents. These are the primary commands you should use.

### @build

**Intent:** Construct and create - build features, prototypes, and designs

**Usage:**
```bash
@build "Feature description"
@build WS-{ID}
```

**What it does:**
1. Pre-build validation
2. TDD cycle (Red → Green → Refactor)
3. Quality gate checks
4. Git commit
5. Beads status update

**Quality Gates:**
- Coverage ≥80%
- mypy --strict
- ruff clean
- Files <200 LOC
- No bare exceptions

**Example:**
```bash
@build "Add user authentication"
@build WS-001-01
```

**See:** [.agents/skills/build.md](../../.agents/skills/build.md)

---

### @fix

**Intent:** Repair and resolve - fix bugs, issues, and problems

**Usage:**
```bash
@fix "Bug description"
@fix --hotfix "Critical issue"
@fix --issue "Investigate problem"
```

**What it does:**
1. Bug analysis and classification
2. Routes to appropriate fix strategy
3. TDD cycle for fixes
4. Quality verification

**Routes to:**
- Emergency fixes for P0 (critical) issues
- Quality fixes for P1/P2 issues
- Investigation for complex problems

**Example:**
```bash
@fix "Login fails on Firefox"
@fix --hotfix "Production database connection fails"
@fix --issue "Test fails unexpectedly"
```

**See:** [.agents/skills/fix.md](../../.agents/skills/fix.md)

---

### @operate

**Intent:** Run and maintain - deploy, plan, and manage operations

**Usage:**
```bash
@operate deploy F{ID}
@operate plan "Feature description"
@operate ci-triage
```

**What it does:**
1. Deployment orchestration
2. Feature planning
3. CI/CD triage and management

**Examples:**
```bash
@operate deploy F001
@operate plan "Add user authentication"
@operate ci-triage
```

**See:** [.agents/skills/operate.md](../../.agents/skills/operate.md)

---

### @understand

**Intent:** Analyze and explore - understand codebase, architecture, and metrics

**Usage:**
```bash
@understand landscape
@understand scout "query"
@understand architect "component"
@understand metrics
```

**What it does:**
1. Codebase analysis
2. Architecture exploration
3. Metrics analysis
4. Code reconnaissance

**Examples:**
```bash
@understand landscape
@understand scout "authentication flow"
@understand architect "user service"
@understand metrics
```

**See:** [.agents/skills/understand.md](../../.agents/skills/understand.md)

---

### @review

**Intent:** Evaluate and verify - quality checks, reality testing, and verification

**Usage:**
```bash
@review F{ID}
@review reality-check
@review verify-workstream WS-{ID}
```

**What it checks:**
- All workstreams completed
- Tests passing
- Coverage ≥80%
- Type hints complete
- No TODO markers
- Code-requirements alignment

**Examples:**
```bash
@review F001
@review reality-check
@review verify-workstream WS-001-01
```

**See:** [.agents/skills/review.md](../../.agents/skills/review.md)

---

## Utility Commands

### @git-worktree

**Purpose:** Create isolated git worktrees for feature work

**Usage:**
```bash
@git-worktree "feature-name"
```

**See:** [.agents/skills/git-worktree.md](../../.agents/skills/git-worktree.md)

---

### @parallel-dispatch

**Purpose:** Delegate work to parallel subagents

**Usage:**
```bash
@parallel-dispatch
```

**See:** [.agents/skills/parallel-dispatch.md](../../.agents/skills/parallel-dispatch.md)

---

### @review-readiness

**Purpose:** Check readiness for code review

**Usage:**
```bash
@review-readiness
```

**See:** [.agents/skills/review-readiness.md](../../.agents/skills/review-readiness.md)

---

### @llm-council

**Purpose:** Multi-LLM consensus and decision making

**Usage:**
```bash
@llm-council "decision topic"
```

**See:** [.agents/skills/llm-council.md](../../.agents/skills/llm-council.md)

---

### @strataudit

**Purpose:** Audit codebase structure and organization

**Usage:**
```bash
@strataudit
```

**See:** [.agents/skills/strataudit.md](../../.agents/skills/strataudit.md)

---

## Internal Commands

### @beads

**Purpose:** Task tracking

**Usage:**
```bash
@beads list
@beads show {ID}
@beads update {ID} --status in_progress
```

**See:** [.claude/skills/beads/SKILL.md](../../.claude/skills/beads/SKILL.md)

---

### @init

**Purpose:** Initialize SDP

**Usage:**
```bash
@init
```

**See:** [.claude/skills/init/SKILL.md](../../.claude/skills/init/SKILL.md)

---

### /tdd

**Purpose:** TDD cycle enforcement (internal)

**Usage:**
Automatic (called by @build)

**Process:**
1. Red - Write failing test
2. Green - Write minimal code
3. Refactor - Improve code

**See:** [.claude/skills/tdd/SKILL.md](../../.claude/skills/tdd/SKILL.md)

---

### @guard

**Purpose:** Scope enforcement (internal)

**Usage:**
Automatic (called by @build)

**See:** [.claude/skills/guard/SKILL.md](../../.claude/skills/guard/SKILL.md)

---

## Command Options

### Verbosity Levels

Most commands support verbosity:

```bash
@build WS-001-01 --verbose
@review F001 --quiet
```

### Background Execution

For long-running commands:

```bash
@build F001 --background
```

---

## Quick Reference

| Command | Purpose | Intent |
|---------|---------|--------|
| `@build` | Build features | build |
| `@fix` | Fix bugs | fix |
| `@operate` | Deploy & manage | operate |
| `@understand` | Analyze codebase | understand |
| `@review` | Quality checks | review |

---

## Command Flow

### Standard Feature Development

```
@understand → @build → @build → ... → @review → @operate
```

### Bug Fix Flow

```
@fix → @review → @operate
```

### Analysis Flow

```
@understand → @review
```

---

## Deprecated Aliases

The following legacy commands are deprecated but still work. They redirect to the appropriate intent-based command:

### Build Intent Aliases

| Legacy Command | Routes To | Notes |
|----------------|-----------|-------|
| `@feature` | `@build` | Feature planning now part of build intent |
| `@idea` | `@build` | Requirements gathering now part of build intent |
| `@design` | `@build` | System design now part of build intent |
| `@ux` | `@build` | UX design now part of build intent |
| `@vision` | `@build` | Product vision now part of build intent |
| `@oneshot` | `@build` | Autonomous execution now part of build intent |
| `@prototype` | `@build` | Prototyping now part of build intent |

### Fix Intent Aliases

| Legacy Command | Routes To | Notes |
|----------------|-----------|-------|
| `@hotfix` | `@fix` | Emergency fixes now part of fix intent |
| `@bugfix` | `@fix` | Quality fixes now part of fix intent |
| `@issue` | `@fix` | Bug analysis now part of fix intent |
| `@debug` | `@fix` | Debugging now part of fix intent |

### Operate Intent Aliases

| Legacy Command | Routes To | Notes |
|----------------|-----------|-------|
| `@deploy` | `@operate` | Deployment now part of operate intent |
| `@ci-triage` | `@operate` | CI/CD triage now part of operate intent |
| `@plan` | `@operate` | Planning now part of operate intent |

### Understand Intent Aliases

| Legacy Command | Routes To | Notes |
|----------------|-----------|-------|
| `@landscape` | `@understand` | Landscape analysis now part of understand intent |
| `@scout` | `@understand` | Code reconnaissance now part of understand intent |
| `@architect` | `@understand` | Architecture analysis now part of understand intent |
| `@metrics` | `@understand` | Metrics analysis now part of understand intent |

### Review Intent Aliases

| Legacy Command | Routes To | Notes |
|----------------|-----------|-------|
| `@reality-check` | `@review` | Reality checking now part of review intent |
| `@verify-workstream` | `@review` | Workstream verification now part of review intent |
| `@reality` | `@review` | Codebase analysis now part of review intent |

**Note:** These deprecated aliases are maintained for backward compatibility but may be removed in future versions. Use the intent-based commands instead.

---

## See Also

- [skills.md](skills.md) - Skill system details
- [quality-gates.md](quality-gates.md) - Quality standards
- [beginner/02-common-tasks.md](../beginner/02-common-tasks.md) - Common workflows
- [deprecated-aliases.md](../../.agents/skills/deprecated-aliases.md) - Complete alias mapping

---

**Version:** SDP v1.0.0
**Updated:** 2026-04-17
