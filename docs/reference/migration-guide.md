# SDP Intent Model Migration Guide

**Status:** Active Migration
**Feature:** F125 (Toolkit UX — intent-routed skills over composable tools)
**Effective:** 2026-04-17

## What Changed?

SDP migrated from **26+ flat skills** to **5 intent-based skills** with modes. The new model reduces cognitive overhead while enabling unlimited tool composition behind each intent.

### Before: Flat Skill List

```
@scout, @architect, @metrics, @spec, @landscape, @index
@feature, @idea, @design, @ux, @vision, @oneshot, @prototype
@hotfix, @bugfix, @issue, @debug
@review, @reality-check, @verify-workstream
@deploy, @ci-triage, @plan
```

**Problem:** Too many choices, unclear boundaries, every new capability = new skill.

### After: Intent-Based Skills

```
@understand (quick|standard|deep)
@build (idea|feature|prototype)
@fix (quick|investigate|systematic)
@review (code|architecture|security|readiness|reality)
@operate (deploy|triage|plan)
```

**Benefits:**
- 5 memorable intents that map to "what you want to do"
- Modes fine-tune scope and depth
- Skills compose CLI tools automatically
- New tools don't require new skills

## Migration Mapping

All legacy skills have **exact equivalents** in the new intent model. No functionality is lost.

### Understand Intent Migrations

| Legacy Skill | New Intent | Mode | What Changes |
|--------------|------------|------|--------------|
| `@scout` | `@understand` | quick | Direct replacement. Use `@understand` or `@understand --depth quick` |
| `@architect` | `@understand` | standard | Use `@understand --depth standard` |
| `@metrics` | `@understand` | standard | Use `@understand --depth standard` |
| `@spec` | `@understand` | deep | Use `@understand --depth deep` |
| `@landscape` | `@understand` | standard/deep | Use `@understand` without mode (auto-detects) |
| `@index query` | `@understand` | deep | Use `@understand --depth deep` |

**Example migrations:**
- `@scout this repo` → `@understand this repo` (auto-selects quick if scout.json exists)
- `@architect analyze` → `@understand --depth standard`
- `@landscape full analysis` → `@understand --depth deep`

### Build Intent Migrations

| Legacy Skill | New Intent | Mode | What Changes |
|--------------|------------|------|--------------|
| `@feature` | `@build` | feature | Direct replacement. Use `@build` or `@build --mode feature` |
| `@idea` | `@build` | idea | Use `@build --mode idea` |
| `@design` | `@build` | idea | Use `@build --mode idea` |
| `@ux` | `@build` | idea | Use `@build --mode idea` |
| `@vision` | `@build` | idea | Use `@build --mode idea` |
| `@oneshot` | `@build` | prototype | Use `@build --mode prototype`. Note: Checkpoint/resume behavior now available through `@operate --mode plan` |
| `@prototype` | `@build` | prototype | Use `@build --mode prototype` |

**Example migrations:**
- `@feature add authentication` → `@build add authentication`
- `@design payment flow` → `@build --mode idea payment flow`
- `@oneshot F001` → `@build --mode prototype F001` (for fast implementation)
- `@prototype quick mock` → `@build --mode prototype quick mock`

**Note on @oneshot checkpoint/resume:** The legacy @oneshot skill supported checkpoint save/restore and background execution. In the new intent model:
- Fast implementation: Use `@build --mode prototype` (skip design, mark experimental)
- Session management & resume: Use `@operate --mode plan` (for session planning and checkpoint management)

### Fix Intent Migrations

| Legacy Skill | New Intent | Mode | What Changes |
|--------------|------------|------|--------------|
| `@hotfix` | `@fix` | quick | Use `@fix` or `@fix --mode quick` |
| `@bugfix` | `@fix` | systematic | Use `@fix --mode systematic` |
| `@issue` | `@fix` | systematic | Use `@fix --mode systematic` |
| `@debug` | `@fix` | investigate | Use `@fix --mode investigate` |

**Example migrations:**
- `@hotfix CI is broken` → `@fix CI is broken` (auto-selects quick for clear bugs)
- `@debug investigate auth failure` → `@fix --mode investigate auth failure`
- `@issue #123` → `@fix --mode systematic #123`

### Review Intent Migrations

| Legacy Skill | New Intent | Dimension | What Changes |
|--------------|------------|-----------|--------------|
| `@review` | `@review` | code | Direct replacement. Default dimension |
| `@reality-check` | `@review` | reality | Use `@review --dimension reality` |
| `@verify-workstream` | `@review` | readiness | Use `@review --dimension readiness` |

**Example migrations:**
- `@review PR #42` → `@review PR #42` (no change needed)
- `@reality-check architecture` → `@review --dimension reality architecture`
- `@verify-workstream WS-123` → `@review --dimension readiness WS-123`

### Ship Command

| Legacy Skill | New Command | What Changes |
|--------------|-------------|--------------|
| `@deploy` | `@ship` | Renamed for clarity - same functionality |

**Example migrations:**
- `@deploy to production` → `@ship to production`
- `@deploy --release` → `@ship --release`

### Operate Intent Migrations

| Legacy Skill | New Intent | Mode | What Changes |
|--------------|------------|------|--------------|
| `@ci-triage` | `@operate` | triage | Use `@operate --mode triage` |
| `@plan` | `@operate` | plan | Use `@operate --mode plan` |

**Example migrations:**
- `@ci-triage failing tests` → `@operate --mode triage failing tests`
- `@plan backlog` → `@operate --mode plan backlog`

## Practices That Are Not Skills

Some workflows were previously called "skills" but are now **embedded practices** — they don't need explicit invocation:

| Practice | Status | How It Works Now |
|----------|--------|------------------|
| `@tdd` | **Embedded** | Default workflow in @build and @fix — test-first is automatic |
| `@guard` | **Automatic** | Pre-commit quality gate via hooks (not a skill you call) |
| `@go-modern` | **Convention** | Language style documented in CLAUDE.md (applied automatically) |
| `@think` | **Technique** | Prompt pattern used throughout intents (not a separate skill) |
| `@beads` | **CLI tool** | Use `bd` commands directly (issue tracker, not an AI skill) |

**You don't invoke these — they're part of how intents work.**

## Deprecation Timeline

| Phase | Date | Status |
|-------|------|--------|
| **Soft Launch** | 2026-04-17 | New intent skills available, legacy skills still work |
| **Warning Period** | 2026-04-17 → 2026-06-01 | Legacy skill usage triggers deprecation warnings |
| **Hard Cutover** | 2026-06-01 | Legacy skill names removed, intent skills only |

**Note:** Deprecation warning mechanism is documented but not yet implemented in CLI routers. This PR establishes the migration path and intent surface; runtime warnings require follow-up work.

## How to Migrate Your Workflows

### 1. Quick Start (5 min)

Replace direct skill names with intents:

```bash
# Old pattern
@scout .
@feature add user auth
@hotfix CI broken
@review PR #42
@deploy to staging

# New pattern
@understand .
@build add user auth
@fix CI broken
@review PR #42
@ship to staging
```

### 2. Learn Mode Auto-Detection

The intent skills **auto-detect** the right mode from your request:

- "Quick look" or "what is this?" → `quick` mode
- "Analyze" or "how does this work?" → `standard` mode
- "Full analysis" or "deep dive" → `deep` mode
- "Design..." or "how should we..." → `idea` mode
- "Implement..." or "build..." → `feature` mode
- "Prototype..." or "quick mock" → `prototype` mode

**You rarely need to specify modes explicitly.**

### 3. Update Documentation

Search your docs for legacy skill invocations:

```bash
# Find legacy skill usage
grep -r "@scout\|@feature\|@hotfix\|@deploy" docs/

# Replace with new intents
# @scout → @understand
# @feature → @build
# @hotfix → @fix
# @deploy → @ship
```

### 4. Update Scripts and Automation

If you have scripts that invoke skills directly:

```bash
# Old script
sdp skill @feature "add OAuth support"

# New script
sdp skill @build "add OAuth support"
```

## Common Migration Questions

### Q: Do I need to specify modes every time?

**A:** No. Intent skills auto-detect the right mode from context:
- What you're asking for ("design" → idea mode, "implement" → feature mode)
- What's already available (design doc exists? skip to implementation)
- Time budget ("quick look" → quick mode)

Explicit mode flags (`--mode`) are optional overrides.

### Q: What happened to TDD?

**A:** TDD is now the **default** workflow in @build and @fix. You don't invoke `@tdd` separately — test-first is embedded. Write failing test → implement → verify → refactor.

### Q: Can I still use old skill names?

**A:** During the warning period (until 2026-06-01), yes — they'll work but show deprecation warnings. After that, only intent names will work.

### Q: What if I forget the new name?

**A:** The deprecation warning tells you exactly what to use. Or check `docs/reference/skills.md` — the intent reference is now the primary documentation.

### Q: Are there any breaking changes?

**A:** No functional changes. All legacy skills have 1:1 equivalents. The new model is **purely a reorganization** for better discoverability and scalability.

### Q: What about specialized skills like @strataudit?

**A:** Specialized skills remain unchanged. Only the 26 core skills were consolidated into 5 intents. See `docs/reference/skills.md` for the full skill list.

## Getting Help

- **Full intent reference:** `docs/reference/skills.md`
- **Design rationale:** `docs/plans/2026-04-13-sdp-skill-architecture-design.md`
- **Legacy → New mapping:** See tables above

## Checklist

Use this checklist to migrate your workflow:

- [ ] Replace `@scout`, `@architect`, `@metrics`, `@spec` with `@understand`
- [ ] Replace `@feature`, `@idea`, `@design`, `@prototype` with `@build`
- [ ] Replace `@hotfix`, `@bugfix`, `@debug` with `@fix`
- [ ] Replace `@review` variants with `@review` (default) or `@review --dimension X`
- [ ] Replace `@deploy`, `@ci-triage`, `@plan` with `@operate --mode X`
- [ ] Remove explicit `@tdd` invocations (now embedded)
- [ ] Update any scripts or automation that use legacy skill names
- [ ] Update team documentation to reference intents first

**Migration complete!** You're now using the intent model. 🎉
