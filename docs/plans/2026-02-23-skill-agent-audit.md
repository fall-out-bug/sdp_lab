# Skill & Agent Audit: Necessity, Reliability, Consolidation

> **Status:** Research complete
> **Date:** 2026-02-23
> **Goal:** Audit all 10 skills + 26 operational skills + 30 agents for reliability, necessity, and simplification
> **Related:** [Agent Loop Reliability](2026-02-23-agent-loop-reliability.md) — root cause analysis

---

## Executive Summary

| Category | Total | Keep | Simplify | Merge | Delete |
|----------|-------|------|----------|-------|--------|
| Core workflow skills | 5 | 2 | 2 | — | — |
| Planning/design skills | 10 | 3 | 4 | 1 | 2 |
| Operational skills | 13 | 2 | 8 | — | 3 |
| Agents | 30 | 12 | — | 1 | 17 |
| **Total** | **58** | **19** | **14** | **2** | **22** |

**Net result:** 58 → ~34 components. ~4,700 lines removed. Zero functional loss.

---

## Critical Cross-Cutting Problems

### 1. Python/Go Mismatch (SEVERITY: CRITICAL)

Skills `bugfix`, `hotfix`, `tdd`, `test`, `init` reference Python tooling (`pytest`, `mypy`, `ruff`, `poetry`) — but this is a **Go project** using `go test`, `go vet`, `go build`. Agents following these skills run commands that don't apply.

**Affected:** bugfix, hotfix, tdd, test, init

### 2. Phantom CLI Commands (SEVERITY: HIGH)

Multiple skills reference `sdp` subcommands that don't exist:

| Phantom Command | Referenced By |
|-----------------|---------------|
| `sdp collision detect` | design |
| `sdp contract generate/lock` | design |
| `sdp memory search/stats` | discovery |
| `sdp resolve <id>` | bugfix |
| `sdp guard finding add/list/resolve/clear` | guard |
| `sdp parse ws <id>` | protocol-consistency |

### 3. Branch Model Mismatch

`bugfix` branches from `dev`, `hotfix` from `main` — but project uses `master` with `feature/` branches. No `dev` branch exists.

### 4. 17 Dead Agents (57% of total)

17 of 30 agents in `.opencode/agents/` are completely unreferenced by any skill, command, or workflow. They are dead code that confuses future agents reading the project.

### 5. Triple-Copy Drift

Skills exist in 3 places: `sdp/prompts/skills/`, `.opencode/skills/`, `.cursor/skills/`. Changes to one don't propagate to others.

---

## Part 1: Core Workflow Skills

| Skill | Lines | Loop Risk | Action | Target Lines |
|-------|-------|-----------|--------|-------------|
| **@oneshot** | 381 | CRITICAL (3 loops) | **REWRITE** — extract loops to CLI (F014-F016) | ~50 |
| **@build** | 319 | MED (TDD + auto-continue) | **SIMPLIFY** — remove auto-continue, strip evidence boilerplate | ~120 |
| **@review** | 277 | LOW | **KEEP** — add CLI verdict validation post-run | ~277 |
| **@deploy** | 258 | LOW | **KEEP** — reword "Next steps" to "Human actions required" | ~250 |
| **@feature** | 308 | LOW | **KEEP** — absorb @discovery Step 0, strip "Next steps" | ~280 |

### @oneshot — P0: Rewrite

The 381-line imperative script is unfixable by adding more rules. Every "NEVER STOP" is a patch on broken architecture. **All 3 loops must move to external CLI** (already planned as F014-F016). Prompt shrinks to ~50 lines: "read checkpoint, execute current phase, run sdp orchestrate --advance."

### @build — P1: Scope Surgery

Remove CRITICAL RULES 2 ("AUTO-CONTINUE") and 4 ("start next WS"). @build should build ONE workstream and STOP. Continuation is the orchestrator's job. Remove evidence lifecycle boilerplate (~100 lines) — make it a post-build CLI hook.

### @review — P3: Keep + External Validation

Best-designed skill. Add CLI verdict validation: after @review runs, CLI checks `jq '.verdict' .sdp/review_verdict.json` exists and is valid.

### @deploy — P3: Reword

"Next steps" handoff is intentionally human-gated (UAT), which is correct. Reword to "Human actions required" to avoid training the handoff pattern.

### @feature — P3: Absorb + Strip

Absorb useful 20 lines from @discovery (roadmap overlap check). Remove "Next steps" section.

---

## Part 2: Planning & Design Skills

| Skill | Lines | Action | Target Lines |
|-------|-------|--------|-------------|
| **@think** | 244 | **SIMPLIFY** — cut expert table (LLMs know experts), cut Stage 3 template | ~80 |
| **@idea** | 206 | **KEEP** — remove "Next Steps" section | ~180 |
| **@design** | 160 | **KEEP** — remove phantom CLI commands, remove "Next Steps" | ~100 |
| **@discovery** | 192 | **DELETE** — merge 20 useful lines into @feature Step 0 | 0 |
| **@reality** | 160 | **KEEP** — no changes needed | 160 |
| **@reality-check** | 253 | **SIMPLIFY** — cut examples, keep core logic | ~60 |
| **@ux** | 111 | **KEEP** — clean, focused, no issues | 111 |
| **@vision** | 126 | **KEEP** — absorb @prd functionality | ~150 |
| **@prd** | 156 | **DELETE** — merge into @vision --update | 0 |
| **@verify-workstream** | 239 | **SIMPLIFY** — cut verbose examples | ~80 |

### Key Merges

- **@discovery → @feature:** Only the "check ROADMAP for overlap" step (~20 lines) is useful. The NOVEL track with Cagan risk framework will never execute reliably in an LLM.
- **@prd → @vision:** Two skills producing two PRD formats. Consolidate into one skill, one format.

### Phantom Command Cleanup

Remove from @design: `sdp collision detect`, `sdp contract generate/lock`
Remove from @discovery: `sdp memory search/stats`

---

## Part 3: Operational & Utility Skills

| Skill | Lines | Action | Target Lines |
|-------|-------|--------|-------------|
| **@debug** | 106 | **KEEP** — best-designed operational skill | 106 |
| **@ci-triage** | 77 | **KEEP** — real commands, clear output | 77 |
| **@bugfix** | ~200 | **SIMPLIFY** — rewrite for Go, fix branch model | ~60 |
| **@hotfix** | ~200 | **SIMPLIFY** — rewrite for Go, align with master | ~60 |
| **@tdd** | ~150 | **SIMPLIFY** — rewrite example for Go | ~50 |
| **@issue** | ~200 | **SIMPLIFY** — keep classification table + routing only | ~30 |
| **@guard** | 185 | **SIMPLIFY** — strip fictional `finding` commands | ~40 |
| **@beads** | 346 | **SIMPLIFY** — keep reference card + integration | ~80 |
| **@prototype** | 100 | **SIMPLIFY** — keep gate override table only | ~30 |
| **@protocol-consistency** | 76 | **SIMPLIFY** — remove phantom `sdp parse ws` | ~60 |
| **@test** | 324 | **DELETE** — contract approval workflow is unenforceable | 0 |
| **@help** | 118 | **DELETE** — redundant with native LLM skill-matching | 0 |
| **@init** | ~200 | **DELETE** — `sdp init` CLI exists, skill adds nothing | 0 |

### Model Skills: @debug and @ci-triage

These two skills are **how all skills should look**: concise (77-106 lines), real commands only, clear output format, no "NEVER/MUST" behavioral rules, no handoff lists. They should be the template for simplification.

---

## Part 4: Agents

### Keep (12 agents — actively used by skills)

| Agent | Used By | SDP-Specific? | Action |
|-------|---------|---------------|--------|
| orchestrator | @oneshot | Heavy | Keep |
| implementer | @build | Heavy | Keep, trim 408→150 |
| spec-reviewer | @build | Heavy | Keep, trim 589→150 |
| reviewer | commands.json | Medium | Keep |
| planner | commands.json | Medium | Keep |
| deployer | commands.json | Medium | Keep |
| qa | @review | Low (Beads only) | Keep |
| security | @review | Low (Beads only) | Keep |
| devops | @review | Low (Beads only) | Keep |
| sre | @review | Low (Beads only) | Keep |
| tech-lead | @review | Low (Beads only) | Keep |
| architect | commands.json | Low | Keep |

### Merge (1)

| Agent | Into | Reason |
|-------|------|--------|
| builder | implementer | builder is a subset of implementer |

### Delete (17 agents — completely unreferenced)

analyst, developer, supervisor, business-analyst, ci-reviewer, code-analyzer, contract-synthesizer, contract-validator, debugger, fixer, product-manager, system-architect, systems-analyst, tester, visionary, technical-decomposition, workflow-auditor

Also replace 562-line README.md with 20-line index.

**Total lines removed from agents: ~2,847 out of ~4,500 (63%)**

---

## Implementation Priority

### Wave 1: Quick Wins (1-2 sessions)

- [ ] Delete 3 skills: test, help, init
- [ ] Delete 17 agents + builder merge
- [ ] Replace agents README (562 → 20 lines)
- [ ] Fix Python→Go in tdd, bugfix, hotfix (critical mismatch)
- [ ] Fix branch model: dev→master, main→master
- [ ] Remove all phantom CLI commands from skills
- [ ] Remove "Next Steps" sections from idea, design, feature, deploy

### Wave 2: Simplification (2-3 sessions)

- [ ] Compress beads (346→80), guard (185→40), issue (200→30)
- [ ] Compress reality-check (253→60), verify-workstream (239→80)
- [ ] Compress think (244→80), prototype (100→30)
- [ ] Merge discovery→feature, prd→vision
- [ ] Trim implementer (408→150), spec-reviewer (589→150)

### Wave 3: Architecture (F014-F016 from roadmap)

- [ ] @build scope surgery: remove auto-continue, strip evidence boilerplate
- [ ] @oneshot rewrite: 381→50 lines, all loops external
- [ ] CLI gates: sdp ci-loop, sdp orchestrate state machine
- [ ] Stop hooks for Cursor + Claude Code

---

## Line Count Summary

| Category | Before | After | Reduction |
|----------|--------|-------|-----------|
| Core skills (5) | 1,543 | ~977 | -37% |
| Planning skills (10) | 1,847 | ~921 | -50% |
| Operational skills (13) | ~2,382 | ~593 | -75% |
| Agents (30→13) | ~4,500 | ~1,653 | -63% |
| **Total** | **~10,272** | **~4,144** | **-60%** |

---

## Design Principles for Simplified Skills

Based on what works (@debug, @ci-triage) vs what fails (@oneshot, @test):

| Principle | Good Example | Bad Example |
|-----------|-------------|-------------|
| **Concise** (50-100 lines) | @debug (106) | @oneshot (381) |
| **Real commands only** | @ci-triage (`gh pr checks`) | @guard (`sdp guard finding add`) |
| **No "NEVER/MUST" walls** | @debug (0 behavioral rules) | @oneshot (8 CRITICAL RULES) |
| **No "Next Steps" handoff** | @review (writes file, done) | @deploy ("1. Wait for CI...") |
| **Single responsibility** | @ci-triage (classify CI failures) | @build (TDD + evidence + auto-continue) |
| **Positive framing** | "Output: verdict file" | "Do NOT output Next Steps" |
| **Correct toolchain** | `go test ./...` | `pytest --cov` |
| **CLI for enforcement** | `sdp ci-loop` exits 0/1 | "RULE: Do NOT hand off" |
