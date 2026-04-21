# UX Audit Results: SDP + SDP Lab

**Date:** 2026-04-05 (updated with council corrections)
**Scope:** End-to-end UX from task to PR across 3 personas, 3 happy paths, 4 harnesses, and the orchestrator
**Method:** Walk-through of actual user journeys against skill files, harness configs, CLI output, and documentation. Benchmarked against 9 reference projects (superpowers, gstack, Gas Town, paperclip, oh-my-openagent, sgr-agent-core, deer-flow, hyperpowers, openclaw).
**Council review:** Reviewed by Codex, Cursor, and OpenCode+OhMyOpenAgent sessions. Three factual corrections applied (F6, F21, F52). See [Council Synthesis](../council-rounds/2026-04-05-ux-council-synthesis.md) for full cross-review.

---

## 1. Audit Scope

### Personas

| Persona | Description |
|---|---|
| **New Adopter** | Installing SDP for the first time, running through first success |
| **Day-to-Day Operator** | Using SDP daily for feature delivery in an SDP-native project |
| **Platform Contributor** | Contributing to sdp_lab itself (multi-repo, Go, private planning) |

### Happy Paths

| Path | Scenario |
|---|---|
| **HP1** | Greenfield project from zero to first meaningful prototype |
| **HP2** | New feature in an existing SDP-native project |
| **HP3** | Brownfield / legacy project adopting AI SDLC |

### Harnesses

| Harness | Config Location |
|---|---|
| Claude Code | `.claude/` (settings.json, commands.json, hooks/, patterns/) |
| Cursor | `.cursor/` (worktrees.json, README.md) |
| OpenCode | `.opencode/` (opencode.json, README.md) |
| Codex | `.codex/` (INSTALL.md, skills/) |

### Orchestrator

`cmd/sdp-orchestrate/main.go` — outer loop state machine with checkpoint/resume.

---

## 2. Happy Path 1: Greenfield to First Prototype

**Expected steps:** install -> init -> vision -> feature -> oneshot -> review -> deploy

### Findings

| ID | Finding | Severity | Detail |
|---|---|---|---|
| F1 | 700+ lines required reading before first command | HIGH | CLAUDE.md (440 lines) + QUICKSTART (156 lines) + PROTOCOL.md. No progressive disclosure. |
| F2 | `@init` skill listed in Available Skills but does not exist | HIGH | Dead reference in CLAUDE.md. Skill file missing from prompts/skills/. |
| F3 | `sdp demo` undiscoverable | MEDIUM | QUICKSTART mentions it, CLAUDE.md does not. New user reading CLAUDE.md misses the demo path. |
| F4 | Mode choice (Local vs Operator) forced before user understands either | MEDIUM | QUICKSTART step 0 asks for mode decision with insufficient context. |
| F5 | @vision output has no "next step" guidance | MEDIUM | Produces VISION.md, PRD, ROADMAP. User doesn't know where files land or what to do next. |
| F6 | @feature flags not in decision tree | MEDIUM | discovery -> idea -> ux -> design. Skill file says 3-5 questions per sub-skill, not 33. `--quick` and `--auto` flags exist in skill file but not surfaced in CLAUDE.md decision tree. (Corrected: was HIGH, flags exist but undiscoverable.) |
| F7 | Feature ID format and location opaque after @feature | MEDIUM | User gets an ID but must know to look in `docs/workstreams/backlog/`. |
| F8 | install.sh silent on IDE detection | LOW | Auto-detects IDE but doesn't report which was chosen. Fallback to `.claude/` is silent. |
| F9 | Beads presented as "optional" but deeply integrated | HIGH | settings.json has beads.enabled=true by default. New user without Beads gets confusing noise. |
| F10 | No undo/restart path | MEDIUM | If @vision produces garbage, no documented way to reset and retry. |

### Benchmark

| System | First success time | Required reading |
|---|---|---|
| gstack | ~2 min (git clone + ./setup) | 0 lines (hooks auto-suggest) |
| Superpowers | ~1 min (first conversation) | 0 lines (using-superpowers teaches interactively) |
| **SDP** | ~30 min (read docs + install + init + first @build) | 700+ lines minimum |

---

## 3. Happy Path 2: New Feature in SDP-Native Project

**Expected steps:** @feature -> @oneshot -> @review -> @deploy

### Findings

| ID | Finding | Severity | Detail |
|---|---|---|---|
| F11 | @feature always runs all 4 sub-skills | HIGH | Simple feature in known project: 12-27 questions (@idea) + 6 questions (@ux) = up to 33 questions before first line of code. |
| F12 | No path between @feature (heavy) and @build (one WS) | HIGH | No @design-only or @plan-quick. @prototype exists but doesn't create workstreams, so @oneshot can't use it. |
| F13 | Feature ID to Workstream ID mapping opaque | MEDIUM | @feature creates workstreams in backlog/. @oneshot reads checkpoint. If checkpoint not created or corrupted: silent failure. |
| F14 | @review always spawns 7 agents | MEDIUM | 10-line fix gets same 7-reviewer treatment as 500-line feature. No scaling. |
| F15 | Findings loop has no exit condition after max retries | MEDIUM | @review -> CHANGES_REQUESTED -> fix -> loop. Max 3 documented, but no guidance after 3 failures. |
| F16 | Evidence commit is manual step in automatic flow | HIGH | @build step 3b says "commit evidence". Neither @build nor @oneshot auto-commits .sdp/evidence/. |
| F17 | "Prodolzhay {feature-id}" is a text convention, not a command | LOW | Documented in CLAUDE.md as shortcut. Works only if agent reads CLAUDE.md. Not a CLI command. |
| F18 | No progress indication between phases | MEDIUM | Between @oneshot start and end: silence. sdp-orchestrate --status exists but requires manual invocation in separate terminal. |
| F19 | @deploy doesn't deploy | HIGH | Name implies production deployment. Actually does `gh pr merge`. QUICKSTART warns about this, but skill is named @deploy. |

### Benchmark

| System | Feature delivery friction |
|---|---|
| gstack | `/ship` — single command: tests + coverage + PR + readiness dashboard. Progress visible at each step. |
| Gas Town | Convoy-based: create convoy of tasks, workers claim one-by-one. Status via `gt status`. |
| **SDP** | Linear pipeline without visibility between steps. Trust the system or manually check checkpoint. |

---

## 4. Happy Path 3: Brownfield / Legacy Adoption

**Expected steps:** install in existing project -> init -> reality -> first SDP-managed feature

### Findings

| ID | Finding | Severity | Detail |
|---|---|---|---|
| F20 | No migration/adoption guide | CRITICAL | No document describes "I have an existing project, how to add SDP". QUICKSTART assumes greenfield. |
| F21 | install.sh merge logic could be more robust | MEDIUM | install-project.sh uses merge by default and has --no-overwrite-config flag. But no explicit backup, no preview of changes. (Corrected: was HIGH, merge exists but not robust enough.) |
| F22 | Quality gates incompatible with legacy code | CRITICAL | SDP requires: files < 200 LOC, coverage >= 80%, full type hints, TDD, clean architecture. Legacy project with 500-line files and 30% coverage fails every gate. No adoption mode. |
| F23 | Go-centric tooling in language-agnostic protocol | HIGH | @go-modern skill, `go test`, `golangci-lint` in quality gates, Go patterns in CLAUDE.md. Python/Node/Rust project sees noise. |
| F24 | Guard scope assumes workstreams exist | HIGH | @guard checks file scope against workstream. Brownfield has no workstreams. Guard blocks everything or passes everything. |
| F25 | @reality doesn't produce actionable output for adoption | MEDIUM | Generates reality report (description). Doesn't create migration plan, first workstreams, or gap assessment. |
| F26 | Hooks break existing CI/workflow | HIGH | PostToolUse runs PostToolUseWorkflowCheck.sh after every Edit. Without .sdp/checkpoints/ this errors or silently passes. PreToolUse blocks git reset --hard which may conflict with existing scripts. |
| F27 | Beads vs existing issue tracker | HIGH | SDP deeply integrated with Beads. Projects on GitHub Issues or Linear have no bridge. No adapter, no "use your own tracker" mode. |
| F28 | No partial adoption path | CRITICAL | Cannot take only evidence layer, or only planning skills, or only TDD enforcement. All-or-nothing install: 26 skills, hooks, guard, evidence, beads config. |
| F29 | Submodule vs install — unclear choice | MEDIUM | QUICKSTART offers 3 install methods. No decision matrix for when to use which. |
| F30 | No rollback / uninstall | MEDIUM | No `sdp uninstall`. Manual removal of .sdp/, hooks, settings, workstream files. |

### Benchmark

| System | Brownfield approach |
|---|---|
| Paperclip | `npx paperclipai onboard --yes` — analyzes existing project, generates adapted config. Zero overwrites. |
| oh-my-openagent | `/init-deep` generates AGENTS.md based on real project structure. Adapts to what exists. |
| gstack | Installs to `~/.claude/skills/gstack/` (user-level). Zero project footprint. |
| **SDP** | Assumes project will conform to SDP standards from day one. Brownfield adoption effectively blocked by quality gates. |

---

## 5. Cross-Harness Consistency

### Capability Matrix

| Capability | Claude Code | Cursor | OpenCode | Codex |
|---|:---:|:---:|:---:|:---:|
| Skills (26) | symlink | symlink | symlink | symlink |
| Agents (13) | symlink | symlink | symlink | symlink |
| Hook enforcement | 3 hooks | none | none | none |
| Beads auto-sync | yes | no | no | manual |
| Agent teams / spawn | experimental | no | no | no |
| Command-CLI mapping | commands.json | no | no | no |
| Pattern library | 6 patterns | no | no | no |
| Git safety guards | PreToolUse.sh | no | no | no |
| Workflow validation | PostToolUse.sh | no | no | no |

### Findings

| ID | Finding | Severity | Detail |
|---|---|---|---|
| F31 | Dramatic enforcement gap between harnesses | CRITICAL | Same project behaves fundamentally differently by IDE. Cursor user can git reset --hard, bypass guard, skip beads — zero warnings. |
| F32 | Skills assume spawn but only Claude has it | CRITICAL | @review spawns 7 subagents. @vision spawns 7 experts. @build spawns 3. In Cursor/Codex: silent degradation to single-agent. User unaware. |
| F33 | No fallback documentation for non-Claude harness | HIGH | Skill files say "Spawn subagent: security-reviewer". No "if spawn unavailable" path. |
| F34 | commands.json is Claude-only despite "LLM-agnostic" label | HIGH | Master skill-to-CLI mapping with execution modes and subagent routing. Other harnesses have no equivalent. |
| F35 | Cursor is the thinnest harness | MEDIUM | Only worktrees.json (Go-specific) and README pointing to CLAUDE.md. For a growing IDE — critical gap. |
| F36 | OpenCode has agent definitions but no discipline | MEDIUM | opencode.json defines 9 agents with tool permissions. No hooks, no beads, no patterns. Agents know WHAT but no guardrails for HOW. |
| F37 | No unified discipline contract | HIGH | SDP discipline (TDD, evidence, guard, findings loop) enforced only through Claude hooks. Others: honor system. |
| F38 | Beads config in settings.json is Claude-only extension | MEDIUM | Other IDEs don't read these keys. Beads integration works only in Claude Code. |

### Discipline Enforcement Diagram

```
Claude Code:  [hooks] -> [guard] -> [beads sync] -> [workflow check] -> [stop gate]
              Full enforcement. Hard to skip steps.

Cursor:       [skill text] -> ... -> (nothing)
              Honor system. Easy to skip everything.

OpenCode:     [skill text] -> [agent permissions] -> ... -> (nothing)
              Agent knows its role, no guardrails.

Codex:        [skill text] -> ... -> (nothing)
              Honor system. Same as Cursor.
```

### Distraction Surface

| Harness | Distraction risk | Reason |
|---|---|---|
| Claude Code | LOW | Hooks block off-scope edits, PostToolUse checks WS alignment, Stop gate requires landing |
| Cursor | HIGH | No restrictions. Agent edits any files, skips TDD, forgets evidence |
| OpenCode | MEDIUM | Agent permissions limit tools (reviewer can't write), but no scope enforcement |
| Codex | HIGH | No restrictions. Same as Cursor |

---

## 6. Individual Harness Audit

### 6.1 Claude Code (Completeness: 9/10)

The most feature-complete harness. Only one with full enforcement chain.

| Aspect | Grade | Notes |
|---|---|---|
| Onboarding | B | CLAUDE.md thorough but overloaded (440 lines). Decision tree good but buried mid-file. |
| Skill invocation | A | @skill syntax, slash commands, commands.json routing. Works. |
| Hook enforcement | A | PreToolUse (git safety), PostToolUse (workflow check), Stop (landing gate). |
| Agent teams | B- | EXPERIMENTAL flag. Works but if spawn fails, no fallback. Silent degradation. |
| Beads integration | A | Auto-sync, mapping file, enabled by default. |
| Evidence flow | C | Described but auto-commit absent. Manual step in automatic pipeline. |
| Error recovery | D | No guidance for checkpoint corruption, hook failure, spawn timeout. |

**Claude-specific issues:**

| ID | Finding | Severity |
|---|---|---|
| F39 | settings.json contains non-standard keys (`skills`, `beads`) that Claude Code runtime ignores. They work only when skills/hooks read them manually. | HIGH |
| F40 | Stop hook runs on every session end, not just after @oneshot. Simple Q&A gets "landing the plane" checklist. | MEDIUM |
| F41 | commands.json is not consumed by Claude Code runtime. It's a reference doc for the LLM. Works only because LLM reads it from CLAUDE.md. | HIGH |
| F42 | EXPERIMENTAL_AGENT_TEAMS=1 as production dependency. If Anthropic removes flag, spawn breaks. No graceful degradation. | MEDIUM |

### 6.2 Cursor (Completeness: 3/10)

Minimal harness. Effectively a stub.

| Aspect | Grade | Notes |
|---|---|---|
| Onboarding | F | README 28 lines, says "See CLAUDE.md". Cursor user reads Claude-specific guide. |
| Skill invocation | C | @skill syntax documented but no .cursorrules. Cursor doesn't auto-discover skills. |
| Hook enforcement | F | None. |
| Agent teams | F | Not supported. |
| Beads integration | F | None. |
| Evidence flow | F | Fully manual. No reminders. |

**Cursor-specific issues:**

| ID | Finding | Severity |
|---|---|---|
| F43 | No `.cursorrules` file. Cursor uses this as system prompt. SDP doesn't create one. Agent gets no SDP context automatically. | CRITICAL |
| F44 | worktrees.json is Go-specific (`cd sdp-plugin && go mod download`). Useless or harmful for non-Go projects. | MEDIUM |
| F45 | README links to CLAUDE.md — Cursor user sees Claude Code instructions and Claude-specific hooks. | HIGH |

### 6.3 OpenCode (Completeness: 5/10)

Best non-Claude harness, but without enforcement.

| Aspect | Grade | Notes |
|---|---|---|
| Onboarding | B- | README + opencode.json. Describes agent cards and skill symlinks correctly. |
| Skill invocation | B+ | @skill syntax, 9 primary agents with tool permissions in opencode.json. |
| Agent definitions | A- | 9 agents with explicit tool restrictions. Reviewer can't write. Good design. |
| Hook enforcement | F | sdp_lab has .opencode/hooks/ with sdp-omc-guard. Public sdp repo does not include hooks. |
| Beads integration | F | None. |

**OpenCode-specific issues:**

| ID | Finding | Severity |
|---|---|---|
| F46 | sdp_lab has OpenCode hooks but public sdp repo doesn't. Two enforcement levels for same system. | HIGH |
| F47 | opencode.json prompts are references ("See prompts/skills/build/SKILL.md"), not inline content. OpenCode may not resolve these. | MEDIUM |
| F48 | mem-config.json exists only in sdp_lab. Public users don't get memory persistence. | LOW |

### 6.4 Codex (Completeness: 2/10)

Thinnest harness.

| Aspect | Grade | Notes |
|---|---|---|
| Onboarding | C | INSTALL.md 48 lines, correct but minimal. |
| Skill invocation | D | No codex-specific config beyond symlinks. Skills may not be auto-discovered. |
| Agent definitions | F | Symlink to agents/ but no codex agent config. |
| Hook enforcement | F | None. |
| Beads integration | D | "Optional" in INSTALL.md. Manual only. |

**Codex-specific issues:**

| ID | Finding | Severity |
|---|---|---|
| F49 | Skill symlink nested (`../../prompts/skills`). Fragile double-parent path. Other harnesses use `../prompts/skills`. | MEDIUM |
| F50 | No Codex-specific AGENTS.md. Codex convention uses AGENTS.md as primary system prompt. Root AGENTS.md contains private lab instructions. | HIGH |
| F51 | INSTALL.md references `@build` syntax. No confirmation Codex supports `@` prefix for skills. | MEDIUM |

---

## 7. Orchestrator Audit

**Component:** `cmd/sdp-orchestrate/main.go`
**Completeness:** 6/10 — core works, UX surface is raw.

### Findings

| ID | Finding | Severity | Detail |
|---|---|---|---|
| F52 | --next-action is JSON-only | MEDIUM | `--next-action` outputs JSON with no human-readable mode. `--status` already outputs human-readable Markdown. (Corrected: was HIGH, conflated two commands. Only --next-action is JSON-only.) |
| F53 | No progress indication | HIGH | For 10-WS feature: no "3/10 done, next: 00-042-04, blocked: 00-042-07". Only raw checkpoint state. |
| F54 | Checkpoint corruption = silent failure | CRITICAL | Corrupted .sdp/checkpoints/ (partial write, disk full) causes crash or stale state. No validation on load, no repair command. Confirmed by ERROR_HANDLING_FINDINGS_4.md (checkpoint save errors ignored). |
| F55 | No resume guidance | MEDIUM | `--resume` exists but no "when to use it". After @oneshot interruption: retry @oneshot? Run --resume? Run --next-action? Three paths, unclear which is correct. |
| F56 | Hydrate output is machine-only | MEDIUM | `--hydrate` generates context-packet.json for agent. No `--hydrate --human` for operator who wants to see what agent will know. |
| F57 | Feature discovery via filesystem glob | MEDIUM | Orchestrator finds workstreams by scanning docs/workstreams/backlog/. If file renamed or moved: silent skip. |
| F58 | No dry-run for advance | LOW | `--advance` immediately mutates checkpoint. No `--advance --dry-run` to preview. |

### Benchmark

| Aspect | SDP orchestrate | gstack parallel sprint | Gas Town convoy |
|---|---|---|---|
| Progress visibility | JSON only | Real-time per-agent | `gt status` dashboard |
| Error recovery | Silent corruption | Agent restart with context | Escalation levels |
| Parallel execution | Sequential by default | 10-15 parallel agents | Work-stealing scheduler |
| Human-readable output | No | Yes | Yes |
| Resume after interruption | --resume flag, no guidance | Automatic from worktree | Context from git hooks |

---

## 8. Summary Statistics

| Metric | Value |
|---|---|
| Total findings | 58 |
| CRITICAL | 6 |
| HIGH | 17 (was 20, 3 corrected to MEDIUM) |
| MEDIUM | 25 (was 22, 3 promoted from HIGH) |
| LOW | 5 |
| Unique to Claude Code | 4 |
| Unique to Cursor | 3 |
| Unique to OpenCode | 3 |
| Unique to Codex | 3 |
| Cross-harness | 8 |
| Orchestrator | 7 |
| Happy Path specific | 30 |

### Root Problem Clusters

| Root Problem | Finding Count | Severity |
|---|---|---|
| RP1: No progressive disclosure | 5 | HIGH |
| RP2: Claude-first, not protocol-first | 12 | CRITICAL |
| RP3: No brownfield adoption path | 8 | CRITICAL |
| RP4: Silent failures | 6 | CRITICAL |
| RP5: Orchestrator UX is raw | 8 | HIGH |
| RP6: Missing escape hatches | 7 | HIGH |
| RP7: Dead references and gaps | 11 | MEDIUM |
