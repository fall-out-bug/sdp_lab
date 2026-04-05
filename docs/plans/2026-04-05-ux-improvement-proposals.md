# UX Improvement Proposals

**Date:** 2026-04-05
**Source:** [UX Audit Results](2026-04-05-ux-audit-results.md)
**Status:** Draft proposals for review

---

## How to Read This Document

Each proposal addresses one root problem cluster from the audit. Proposals are ordered by priority (P0 first). Each contains: the problem statement, the proposed approach, what changes, and the expected outcome. Detailed specifications with acceptance criteria are in [the specs document](2026-04-05-ux-improvement-specs.md).

---

## Proposal 1: Progressive Disclosure Engine

**Priority:** P0
**Root Problem:** RP1 — Users get everything at once (700 lines of docs, 33 questions in @feature, 7 reviewers on 2-file changes)
**Addresses:** F1, F4, F6, F11, F14

### Problem

SDP treats all users identically regardless of context. A first-time user reads the same 440-line CLAUDE.md as a daily operator. A 10-line bugfix triggers the same 7-reviewer @review as a 500-line feature. A simple feature in a well-known project goes through the same 33-question @feature pipeline as a novel product exploration.

This is the single largest friction source for daily use.

### Approach

Three changes:

**A. Tiered documentation.** Split CLAUDE.md into 3 layers:
- Level 1 (~50 lines): install, init, decision tree, 5 core commands, "want more? read PROTOCOL.md"
- Level 2 (PROTOCOL.md): full flow, all skills, quality gates
- Level 3 (docs/reference/): deep topic-specific docs

**B. Adaptive @feature.** The skill auto-selects depth based on context:
- Has workstreams already? Skip @discovery
- @ux already ran for this feature? Skip
- `--quick` flag: only @design, skip discovery/idea/ux
- `--auto` flag: generate workstreams non-interactively
- Make these flags visible in the CLAUDE.md decision tree

**C. Scaled @review.** Reviewer count proportional to change size:
- <50 LOC changed: 2 reviewers (QA + TechLead)
- 50-200 LOC: 4 reviewers (+Security, +Docs)
- >200 LOC: all 7
- `--full` flag for forced full review

### What Changes

| Component | Before | After |
|---|---|---|
| CLAUDE.md | 440 lines, flat | ~50 lines, links to deeper docs |
| @feature skill | Always 4 sub-skills | Context-aware, 1-4 sub-skills |
| @review skill | Always 7 agents | 2-7 agents based on diff size |
| Decision tree | Buried in middle of CLAUDE.md | First thing user sees |

### Expected Outcome

- New user reaches first @build in <5 minutes of reading
- @feature --quick completes in <5 interactive questions
- @review on a 10-line fix spawns 2 agents, not 7

---

## Proposal 2: Harness Parity Contract

**Priority:** P0
**Root Problem:** RP2 — 75% of SDP capabilities are Claude Code-only, other harnesses get degraded experience without warning
**Addresses:** F31, F32, F33, F34, F35, F37, F38, F39, F41, F43, F46, F50

### Problem

SDP advertises 4 IDE integrations (Claude Code, Cursor, OpenCode, Codex). In reality:
- Claude Code: full enforcement (hooks, guard, beads, spawn, patterns)
- Cursor: stub (README pointing to CLAUDE.md, no .cursorrules, no hooks)
- OpenCode: partial (agent definitions but no hooks)
- Codex: stub (INSTALL.md only)

Skills that require spawn (7 of 26) silently degrade to single-agent execution in non-Claude harnesses. Users don't know they're getting a degraded experience.

This is worse than not supporting an IDE — it creates false expectations.

### Approach

**A. Formalize discipline tiers.** Declare what each harness level provides:
- T1 (Protocol): skills, agents, CLI commands, evidence schema. All harnesses.
- T2 (Guardrails): scope guard, git safety, workflow validation. Where hooks are supported.
- T3 (Orchestration): agent spawn, beads auto-sync, checkpoint management. Only spawn-capable harnesses.

**B. Per-harness capability manifest.** Each harness declares its capabilities:
```yaml
# .cursor/sdp-capabilities.yml
tier: T1
hooks: false
spawn: false
beads_auto_sync: false
```
Skills read this and adapt behavior.

**C. Skill fallback paths.** Every skill with spawn gets a "If spawn unavailable" section with manual checklist equivalent.

**D. Fix the stubs.**
- Cursor: create proper `.cursorrules` with SDP context and decision tree
- Codex: create public-facing `AGENTS.md` (not the private lab version)
- OpenCode: port hooks from sdp_lab to public sdp repo

### What Changes

| Component | Before | After |
|---|---|---|
| Skill files (spawn-dependent) | Assume spawn available | Fallback section for single-agent |
| .cursor/ | README + worktrees.json | + .cursorrules + capabilities manifest |
| .codex/ | INSTALL.md + symlinks | + AGENTS.md + capabilities manifest |
| .opencode/ | opencode.json | + hooks from sdp_lab + capabilities manifest |
| Documentation | "Supports 4 IDEs" | Explicit tier per IDE with capabilities |

### Expected Outcome

- Every skill works in every harness (full or fallback mode)
- User knows their harness tier and what's available
- Cursor user gets SDP context automatically via .cursorrules
- No silent degradation

---

## Proposal 3: Brownfield Adoption Kit

**Priority:** P0
**Root Problem:** RP3 — No path for existing projects. Quality gates block everything. >90% of real projects are brownfield.
**Addresses:** F20, F21, F22, F23, F24, F26, F27, F28, F29, F30

### Problem

SDP assumes every project starts clean. Quality gates (files < 200 LOC, coverage >= 80%, full type hints, TDD) are non-negotiable from day one. A legacy project with 500-line files and 30% coverage fails every gate. The installer overwrites existing config. There's no partial adoption, no graduation path, no uninstall.

This means SDP is effectively unusable for existing projects.

### Approach

**A. `sdp init --adopt`** — new initialization mode:
- Scans project: language, CI, test framework, coverage, issue tracker
- Generates .sdp/config.yml with adaptive thresholds (gates disabled until baseline)
- Creates ADOPTION_PLAN.md with gap analysis
- Merges with existing .claude/settings.json (doesn't overwrite)

**B. Graduation path** — 4 levels from zero enforcement to full SDP:
```
Level 0: sdp init --adopt     -> evidence only, no gates
Level 1: sdp adopt --planning -> + skills, workstreams
Level 2: sdp adopt --guard    -> + scope enforcement
Level 3: sdp adopt --full     -> full quality gates
```

**C. Language profiles** — remove Go-centric from protocol:
- @go-modern moves to optional language pack
- Quality gates become language-aware (pytest, jest, go test per project)
- Guard rules use language-specific file patterns

**D. Issue tracker adapters** — interface for non-Beads:
- Minimal contract: create, update_status, close, list_ready
- Implementations: github, linear, jira, beads
- Configured in .sdp/config.yml

**E. Safe install** — never overwrite, always merge:
- Existing .claude/settings.json: merge hooks, don't replace
- Backup existing files to .sdp/backup/
- `sdp uninstall` to cleanly remove

### What Changes

| Component | Before | After |
|---|---|---|
| install.sh | Overwrites config | Merges, backs up existing |
| sdp init | --auto or --guided | + --adopt for brownfield |
| Quality gates | Fixed thresholds | Adaptive, per-level |
| Language support | Go-centric | Language profiles |
| Issue tracker | Beads-only | Pluggable adapters |
| Uninstall | Doesn't exist | `sdp uninstall` + `--purge` |

### Expected Outcome

- `sdp init --adopt` works on Python, Node, Go, Rust projects
- Legacy project with 30% coverage can start using SDP at Level 0
- Existing .claude/settings.json preserved
- `sdp uninstall` returns project to pre-SDP state

---

## Proposal 4: Resilient Orchestrator

**Priority:** P1
**Root Problem:** RP4 + RP5 — Silent failures, JSON-only output, no progress visibility
**Addresses:** F13, F16, F17, F18, F19, F39, F41, F52, F53, F54, F55, F56, F57, F58

### Problem

The orchestrator is the core loop of SDP execution, but its UX is raw:
- All output is JSON (no human-readable mode)
- No progress indication during multi-WS execution
- Checkpoint corruption causes silent failure or stale state
- Evidence isn't auto-committed despite being mandatory
- @deploy name misleads users about what it does

### Approach

**A. Human-readable output by default.**
```
$ sdp-orchestrate --feature F042 --status

Feature F042: Add OAuth2 Login
==============================
Progress: 3/7 workstreams complete

  done  00-042-01  Auth schema migration
  done  00-042-02  OAuth2 provider config
  done  00-042-03  Login endpoint
  >     00-042-04  Token refresh logic      [IN PROGRESS]
  o     00-042-05  Session management       [READY]
  x     00-042-06  Admin OAuth settings     [BLOCKED by 00-042-04]
  o     00-042-07  E2E auth tests           [READY]

Next action: complete 00-042-04, then 00-042-05
```
JSON via `--json` flag (invert current behavior).

**B. Checkpoint resilience.**
- Validate JSON schema + hash integrity on load
- `sdp-orchestrate --repair` to recover from git history or evidence log
- Atomic writes via temp file + rename

**C. Evidence auto-commit.**
After @build evidence generation: automatic `git add .sdp/evidence/ .sdp/checkpoints/ && git commit -m "evidence: WS {id}"`. Remove manual step from automatic pipeline.

**D. Inline progress for @oneshot.**
After each WS completion: `[3/7] done 00-042-03 Login endpoint (2m 14s)`.
On block: `[4/7] blocked 00-042-06 blocked by 00-042-04`.

**E. Rename @deploy to @ship.**
- `@ship` — clearer name for "create/merge PR"
- Deprecation: both names work for 2 versions, warning on @deploy usage

### What Changes

| Component | Before | After |
|---|---|---|
| --status output | JSON | Human-readable (--json for machine) |
| Checkpoint load | No validation | Schema + hash validation + --repair |
| Evidence commit | Manual | Auto after each @build |
| @oneshot progress | Silent | Per-WS progress line |
| @deploy name | Misleading | @ship (with deprecation period) |

### Expected Outcome

- Operator reads status at a glance without jq
- Checkpoint corruption detected and recoverable
- Evidence always committed (no forgotten manual step)
- @oneshot shows progress per-workstream

---

## Proposal 5: Escape Hatches and Recovery

**Priority:** P1
**Root Problem:** RP6 — No undo, uninstall, skip, restart, or post-failure guidance
**Addresses:** F10, F12, F15, F29, F30, F42, F55

### Problem

When things go wrong or the user wants a different path, SDP offers no escape:
- No way to restart a botched @vision
- No shortcut between full @feature and single @build
- No guidance after 3 failed @review iterations
- No uninstall
- No resume guidance after interruption

Users feel trapped.

### Approach

**A. `sdp uninstall`** — clean removal:
- Removes .sdp/, SDP hooks, SDP symlinks
- Preserves user data (workstreams, evidence) by default
- `--purge` for complete removal

**B. `@feature --design-only "description"`** — direct path to workstreams:
- Skips discovery, idea, ux
- Only @design: workstream generation
- For experienced users who know what to build

**C. Post-max-retry guidance in @review:**
```
3 review iterations exhausted. Options:
  @review --override    Force approve with justification
  @review --partial     Approve passing checks, file issues for failures
  @review --escalate    Create issue for human review
```

**D. `sdp reset --feature F042`** — restart execution:
- Clears checkpoint
- Preserves workstream definitions
- Allows @oneshot to start fresh

**E. "If this fails" section in every skill.**

### What Changes

| Component | Before | After |
|---|---|---|
| Uninstall | Doesn't exist | `sdp uninstall` + `--purge` |
| @feature paths | All-or-nothing 4-skill pipeline | + `--design-only` shortcut |
| @review max retries | Silent stop | Guided options (override/partial/escalate) |
| Feature restart | Manual checkpoint surgery | `sdp reset --feature` |
| Skill files | No failure guidance | "If this fails" section in each |

### Expected Outcome

- User can always escape, restart, or escalate
- No dead ends in any workflow
- `sdp uninstall` returns project to clean state

---

## Proposal 6: Reference Hygiene

**Priority:** P2
**Root Problem:** RP7 — Dead references, Go-specific noise in agnostic context, missing files
**Addresses:** F2, F3, F5, F7, F8, F44, F45, F47, F49, F50, F51

### Problem

Accumulated inconsistencies: @init skill listed but missing, sdp demo not in CLAUDE.md, Go-specific patterns in language-agnostic docs, fragile symlinks, IDE detection silent.

These are individually minor but collectively erode trust in the system.

### Approach

1. Create `@init` skill file or remove from Available Skills table in CLAUDE.md
2. Add `sdp demo` to CLAUDE.md decision tree: "First time? -> sdp demo"
3. @vision and @feature: after completion, print "Files created: [list]. Next: [command]"
4. Feature ID documentation: after @feature, show ID, location, how to reference
5. install.sh: echo detected IDE ("Detected: Claude Code. Creating .claude/ integration.")
6. Move Go-specific content from protocol-level docs to language profile
7. Fix Codex nested symlink path
8. Replace Cursor/Codex README references from CLAUDE.md to harness-specific docs

### Expected Outcome

- Zero dead references in CLAUDE.md
- install.sh reports detected IDE
- @vision and @feature show "what was created, what to do next"
- No Go-specific content in language-agnostic protocol docs

---

## Priority Roadmap

```
NOW (blocks adoption):
|-- SPEC-01: Progressive Disclosure Engine      [P0, Medium effort]
|-- SPEC-02: Harness Parity Contract            [P0, Large effort]
+-- SPEC-03: Brownfield Adoption Kit            [P0, Large effort]

NEXT (blocks daily UX):
|-- SPEC-04: Resilient Orchestrator             [P1, Medium effort]
+-- SPEC-05: Escape Hatches & Recovery          [P1, Small effort]

LATER (polish):
+-- SPEC-06: Reference Hygiene                  [P2, Small effort]
```

### Dependency Graph

```
SPEC-06 (hygiene) -- no dependencies, can start immediately
SPEC-01 (progressive disclosure) -- no dependencies, can start immediately
SPEC-05 (escape hatches) -- no dependencies, can start immediately
SPEC-03 (brownfield) -- partially depends on SPEC-01 (tiered docs)
SPEC-02 (harness parity) -- partially depends on SPEC-01 (tiered CLAUDE.md)
SPEC-04 (orchestrator) -- independent, can start immediately
```

Recommended parallel execution:
- Stream A: SPEC-01 + SPEC-03 (docs + brownfield)
- Stream B: SPEC-04 + SPEC-05 (orchestrator + escape hatches)
- Stream C: SPEC-02 (harness parity, can start with stub fixes immediately)
- Stream D: SPEC-06 (hygiene, quick wins anytime)
