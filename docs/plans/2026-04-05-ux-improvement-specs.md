# UX Improvement Specifications

**Date:** 2026-04-05
**Source:** [UX Improvement Proposals](2026-04-05-ux-improvement-proposals.md)
**Status:** Draft feature specs (no workstreams yet)

Each spec below is a feature draft ready for `@design` to decompose into workstreams.

---

## SPEC-01: Progressive Disclosure Engine

### Meta

| Field | Value |
|---|---|
| Priority | P0 |
| Effort | Medium |
| Scope | sdp (public repo) |
| Root findings | F1, F4, F6, F11, F14 |
| Depends on | Nothing |
| Enables | SPEC-02 (tiered CLAUDE.md), SPEC-03 (tiered docs) |

### Intent

Users at every experience level get the right amount of information and interaction depth, not the maximum. First-time users see 50 lines instead of 440. Simple features skip unnecessary sub-skills. Small changes get proportional review.

### Scope

#### In scope

1. **Tiered CLAUDE.md restructure**
   - New CLAUDE.md: ~50 lines. Install, init, decision tree, 5 core commands, pointers to deeper docs.
   - Move full skill catalog, quality gates, CLI reference, evidence details, memory system to PROTOCOL.md and docs/reference/.
   - Decision tree becomes the first thing the user reads, not buried at line 52.

2. **Adaptive @feature skill**
   - Context detection at skill start:
     - If `docs/workstreams/backlog/` has files for this feature: skip @discovery
     - If @ux output exists (docs/drafts/ux-*): skip @ux
     - If @idea output exists (docs/drafts/idea-*): skip @idea
   - Flags documented in decision tree and skill frontmatter:
     - `--quick`: @design only (skip discovery, idea, ux). 0 interactive questions.
     - `--auto`: generate workstreams non-interactively from description.
     - `--full`: force all 4 sub-skills (current behavior, becomes opt-in).
   - Default (no flag): context-aware — run only sub-skills whose outputs don't exist yet.

3. **Scaled @review skill**
   - Diff size detection at skill start: count LOC changed in feature branch vs base.
   - Tier mapping:
     - <50 LOC: 2 reviewers (qa, tech-lead)
     - 50-200 LOC: 4 reviewers (+ security, documentation)
     - >200 LOC: 7 reviewers (all current)
   - `--full` flag: force all 7 regardless of size.
   - `--light` flag: force 2 regardless of size.

#### Out of scope

- Web UI or dashboard
- Dynamic skill recommendation engine (future)
- Changes to @build, @oneshot, @deploy skills
- Changes to orchestrator CLI

### Acceptance Criteria

- [ ] CLAUDE.md is <= 60 lines. Contains: install, init, decision tree, 5 commands, links.
- [ ] Decision tree is in the first 20 lines of CLAUDE.md.
- [ ] `@feature "description"` in a project with existing workstreams skips @discovery automatically.
- [ ] `@feature --quick "description"` completes with 0 interactive questions.
- [ ] `@review` on a branch with <50 LOC changed spawns exactly 2 reviewers (or 2 checklist sections if spawn unavailable).
- [ ] `@review --full` spawns all 7 regardless of diff size.
- [ ] All flags documented in skill frontmatter `examples:` field.

### Non-goals

- Removing any skill or capability. This is about layering access, not reducing features.
- Changing the protocol itself. Flow stays: vision -> reality -> feature -> oneshot -> review -> ship.

### Risks

| Risk | Mitigation |
|---|---|
| Context detection wrong (skips needed sub-skill) | Always allow `--full` override. Detect by output file existence, not heuristics. |
| Shorter CLAUDE.md loses critical info | Content moves to PROTOCOL.md, not deleted. Links are explicit. |
| Scaled review misses security issue on small diff | Security reviewer included from 50 LOC. `--full` always available. |

---

## SPEC-02: Harness Parity Contract

### Meta

| Field | Value |
|---|---|
| Priority | P0 |
| Effort | Large |
| Scope | sdp (public repo), all 4 harness directories |
| Root findings | F31, F32, F33, F34, F35, F37, F38, F39, F41, F43, F46, F50 |
| Depends on | SPEC-01 (tiered CLAUDE.md for harness-specific docs) |
| Enables | Consistent multi-IDE experience |

### Intent

Every supported harness provides a predictable, documented experience. Skills adapt to harness capabilities instead of silently degrading. Users know exactly what their IDE supports.

### Scope

#### In scope

1. **Discipline tier model**

   | Tier | Provides | Harnesses |
   |---|---|---|
   | T1: Protocol | Skills, agents, CLI, evidence schema | All |
   | T2: Guardrails | Scope guard, git safety, workflow validation (via hooks) | Claude Code, OpenCode (after hook port) |
   | T3: Orchestration | Agent spawn, beads auto-sync, checkpoint management | Claude Code |

2. **Capability manifest per harness**
   - File: `{harness}/sdp-capabilities.yml`
   - Schema:
     ```yaml
     sdp_version: 0.10.0
     tier: T1|T2|T3
     capabilities:
       hooks: boolean
       spawn: boolean
       beads_auto_sync: boolean
       agent_teams: boolean
       patterns: boolean
     ```
   - Skills read this at invocation to choose full or fallback path.

3. **Skill fallback sections**
   - Every skill that uses `spawn` gets a new section:
     ```markdown
     ## Without Subagent Spawn
     If your harness does not support subagent spawning, execute the
     following checklist sequentially instead:
     1. [checklist item from subagent 1]
     2. [checklist item from subagent 2]
     ...
     ```
   - Skills affected: @review (7 subagents), @vision (7 experts), @reality (8 experts), @build (3 subagents), @feature (4 sub-skills)

4. **Cursor harness fix**
   - Create `.cursorrules` file with:
     - SDP role description
     - Decision tree (from CLAUDE.md Level 1)
     - Available skills list
     - Quality gates summary
     - Link to full docs
   - Remove CLAUDE.md reference from Cursor README

5. **Codex harness fix**
   - Create `.codex/AGENTS.md` with public-facing agent instructions
   - Not the private sdp_lab/AGENTS.md (contains lab-specific content)
   - Content: role, available skills, quality gates, landing-the-plane

6. **OpenCode hook port**
   - Port `sdp-omc-guard` hook from sdp_lab/.opencode/hooks/ to public sdp/.opencode/hooks/
   - Add pre-tool-use.json for edit/write scope enforcement
   - Elevates OpenCode from T1 to T2

7. **commands.json equivalent for non-Claude harnesses**
   - Create `prompts/commands.yml` as the canonical LLM-agnostic mapping
   - .claude/commands.json becomes a derived view
   - Other harnesses can read prompts/commands.yml natively

#### Out of scope

- Adding spawn capability to Cursor/OpenCode/Codex (platform limitation)
- Creating new harness integrations (e.g., Windsurf-specific, Zed)
- Changing skill behavior in Claude Code

### Acceptance Criteria

- [ ] `sdp-capabilities.yml` exists in all 4 harness directories.
- [ ] Every spawn-dependent skill has a "Without Subagent Spawn" section.
- [ ] `.cursorrules` exists and provides SDP context to Cursor agent.
- [ ] `.codex/AGENTS.md` exists with public-facing instructions.
- [ ] `.opencode/hooks/pre-tool-use.json` exists in public sdp repo.
- [ ] `prompts/commands.yml` exists as canonical source.
- [ ] Running `@review` in Cursor produces a manual checklist (7 sections) instead of silent degradation.

### Non-goals

- Making all harnesses equal. T1/T2/T3 tiers are intentional.
- Removing Claude Code advantages. Claude stays T3.

### Risks

| Risk | Mitigation |
|---|---|
| Capability manifest not read by LLM | Include capability summary in harness-specific system prompt (.cursorrules, AGENTS.md) |
| Fallback checklists become stale vs spawn behavior | Generate fallback sections from same source as spawn definitions |
| .cursorrules format changes in Cursor updates | Keep content minimal. Monitor Cursor changelog. |

---

## SPEC-03: Brownfield Adoption Kit

### Meta

| Field | Value |
|---|---|
| Priority | P0 |
| Effort | Large |
| Scope | sdp (public repo: install.sh, CLI, config), sdp_lab (CLI implementation) |
| Root findings | F20, F21, F22, F23, F24, F26, F27, F28, F29, F30 |
| Depends on | SPEC-01 partially (language profiles reference tiered docs) |
| Enables | SDP adoption in existing projects (>90% of target audience) |

### Intent

An existing project can adopt SDP incrementally without being blocked by quality gates, losing existing config, or committing to all-or-nothing. Graduation from zero enforcement to full SDP happens at the project's pace.

### Scope

#### In scope

1. **`sdp init --adopt` mode**
   - Project scan phase:
     - Detect primary language (Go, Python, Node, Rust, Java, mixed)
     - Detect test framework (go test, pytest, jest, cargo test)
     - Detect CI system (GitHub Actions, GitLab CI, none)
     - Detect issue tracker (Beads, GitHub Issues, Linear, Jira, none)
     - Detect existing IDE config (.claude/, .cursor/, .vscode/)
     - Measure current coverage (if possible)
     - Count files exceeding 200 LOC
   - Output:
     - `.sdp/config.yml` with adoption_mode: true and disabled quality gates
     - `docs/ADOPTION_PLAN.md` with gap analysis: current state vs SDP full compliance
     - Merged (not overwritten) IDE config

2. **Graduation levels**

   | Level | Command | What's enabled | What's not |
   |---|---|---|---|
   | 0 | `sdp init --adopt` | Evidence logging, CLI commands, basic skills | No quality gates, no guard, no TDD requirement |
   | 1 | `sdp adopt --planning` | + planning skills (@feature, @design), workstreams | No scope guard, no coverage gate |
   | 2 | `sdp adopt --guard` | + scope enforcement, guard active | No coverage gate, no file size gate |
   | 3 | `sdp adopt --full` | Full quality gates, all enforcement | Everything active |

   - Each level persisted in `.sdp/config.yml` as `adoption_level: 0|1|2|3`
   - Quality gates read adoption_level and skip checks above current level
   - Graduation is one-way (can go up, not down without explicit `sdp adopt --level N`)

3. **Language profiles**
   - New file: `.sdp/language-profile.yml`
     ```yaml
     primary: python
     test_command: pytest --cov
     lint_command: ruff check
     coverage_tool: pytest-cov
     file_patterns:
       source: ["**/*.py"]
       test: ["**/test_*.py", "**/*_test.py"]
     ```
   - Auto-detected by `sdp init --adopt`, editable by user
   - Quality gates use language profile for commands and patterns
   - @go-modern skill: only loaded when language profile is Go

4. **Issue tracker adapters**
   - Interface contract:
     ```
     create(title, description, labels) -> issue_id
     update_status(issue_id, status) -> ok
     close(issue_id, reason) -> ok
     list_ready() -> []issue
     get(issue_id) -> issue
     ```
   - Implementations:
     - `beads` (existing, default when bd is installed)
     - `github` (via gh CLI)
     - `none` (no tracking, skills skip beads steps)
   - Configured in `.sdp/config.yml`:
     ```yaml
     issue_tracker:
       type: github
       project: my-org/my-repo
     ```

5. **Safe install (merge, not overwrite)**
   - install.sh behavior change:
     - If `.claude/settings.json` exists: read existing, merge SDP hooks (append to hooks arrays), preserve existing keys
     - If `.cursorrules` exists: append SDP section at end
     - Backup all modified files to `.sdp/backup/` before modification
   - `sdp uninstall`:
     - Remove .sdp/ directory
     - Remove SDP hooks from settings.json (restore from backup)
     - Remove SDP symlinks
     - Preserve user data by default (workstreams/, evidence/)
     - `--purge` flag: remove everything including user data

6. **Adoption guide document**
   - `docs/ADOPTION.md` in sdp repo
   - Content: "You have an existing project. Here's how to add SDP."
   - Covers: install method choice, init --adopt, what to expect, graduation path, FAQ

#### Out of scope

- Jira adapter (complex auth, defer to community)
- Automatic remediation of legacy code (SDP doesn't rewrite your files)
- Migration of existing issues from other trackers to Beads
- Custom quality gate thresholds (adopt uses level-based, not custom per-gate)

### Acceptance Criteria

- [ ] `sdp init --adopt` successfully initializes in a Python project with 0% coverage and 500-line files.
- [ ] Existing `.claude/settings.json` preserved and merged (not overwritten) after install.
- [ ] At Level 0: `@build` executes without quality gate failures on legacy code.
- [ ] At Level 0: evidence events logged to `.sdp/log/events.jsonl`.
- [ ] `sdp adopt --planning` elevates to Level 1, enables @feature skill.
- [ ] `sdp adopt --full` enables all quality gates.
- [ ] Language profile auto-detected for Go, Python, Node.
- [ ] `issue_tracker: github` routes @build beads steps to `gh issue` commands.
- [ ] `issue_tracker: none` skips all beads steps silently.
- [ ] `sdp uninstall` returns project to pre-SDP state (verified by git diff).
- [ ] `docs/ADOPTION.md` exists with step-by-step brownfield guide.

### Non-goals

- Lowering quality standards permanently. Graduation path leads to full compliance.
- Supporting every language and tracker from day one. Start with Go/Python/Node + Beads/GitHub/none.

### Risks

| Risk | Mitigation |
|---|---|
| Users stay at Level 0 forever | sdp doctor shows current level and suggests next graduation step |
| Language detection wrong | User can edit .sdp/language-profile.yml manually. --guided mode asks. |
| Merge logic for settings.json breaks on edge cases | Backup to .sdp/backup/ before every modification. Manual restore path documented. |
| GitHub adapter rate-limited | Use gh CLI which handles auth and rate limiting natively |

---

## SPEC-04: Resilient Orchestrator

### Meta

| Field | Value |
|---|---|
| Priority | P1 |
| Effort | Medium |
| Scope | sdp_lab (cmd/sdp-orchestrate, internal/orchestrate) |
| Root findings | F13, F16, F17, F18, F19, F39, F41, F52, F53, F54, F55, F56, F57, F58 |
| Depends on | Nothing |
| Enables | Trustworthy daily execution |

### Intent

The orchestrator becomes reliable and human-friendly: readable output by default, progress visibility during execution, resilient checkpoints, automatic evidence commits, and honest naming.

### Scope

#### In scope

1. **Human-readable output by default**
   - `sdp-orchestrate --feature F042 --status` outputs formatted text:
     ```
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
   - `--json` flag for machine-readable output (inverts current default)
   - `--next-action` also human-readable by default, JSON via `--json`

2. **Checkpoint resilience**
   - On load: validate JSON schema, verify hash chain integrity
   - On corruption detected: error message with repair suggestion
   - `sdp-orchestrate --repair --feature F042`: reconstruct checkpoint from:
     1. Git history (most recent valid checkpoint commit)
     2. Evidence log (.sdp/log/events.jsonl)
     3. Filesystem state (workstream files + their status markers)
   - Atomic writes: write to temp file, fsync, rename. Prevents partial writes.
   - Address ERROR_HANDLING_FINDINGS_4.md checkpoint-related findings

3. **Evidence auto-commit**
   - After @build generates evidence: orchestrator runs:
     ```
     git add .sdp/evidence/ .sdp/checkpoints/
     git commit -m "evidence: WS {ws-id} for {feature-id}"
     ```
   - Only when running inside @oneshot loop (not standalone @build)
   - Configurable: `.sdp/config.yml` key `evidence.auto_commit: true` (default)

4. **Inline progress for @oneshot**
   - After each WS completion, emit to stdout:
     ```
     [3/7] done 00-042-03 Login endpoint (2m 14s)
     ```
   - On block:
     ```
     [4/7] blocked 00-042-06 waiting for 00-042-04
     ```
   - On start:
     ```
     [4/7] starting 00-042-04 Token refresh logic
     ```

5. **Rename @deploy to @ship**
   - Rename skill directory: `prompts/skills/deploy/` -> `prompts/skills/ship/`
   - Create `prompts/skills/deploy/SKILL.md` as deprecation wrapper:
     ```markdown
     > @deploy is renamed to @ship. This alias will be removed in v1.1.
     > Run @ship instead.
     ```
   - Update CLAUDE.md, commands.json, all references
   - Deprecation period: 2 minor versions

6. **Resume guidance**
   - After interruption, `sdp-orchestrate --feature F042 --status` includes:
     ```
     Session interrupted. To resume:
       sdp-orchestrate --feature F042 --resume
     ```
   - @oneshot skill adds "If interrupted" section:
     ```markdown
     ## If interrupted
     Run `sdp-orchestrate --feature {id} --resume` to continue from last checkpoint.
     Do NOT restart @oneshot from scratch — it will skip completed workstreams automatically.
     ```

#### Out of scope

- Parallel WS execution (future, requires worker pool)
- Web dashboard for orchestrator status
- Changes to @build skill internals
- Beads integration improvements (separate spec)

### Acceptance Criteria

- [ ] `--status` outputs human-readable text by default (no jq needed).
- [ ] `--status --json` outputs JSON.
- [ ] Corrupted checkpoint file triggers explicit error: "Checkpoint corrupted. Run --repair to recover."
- [ ] `--repair` successfully recovers from git history.
- [ ] Atomic checkpoint writes: kill -9 during write does not corrupt checkpoint.
- [ ] @oneshot auto-commits evidence after each WS (verifiable via git log).
- [ ] @oneshot prints progress line per-WS.
- [ ] @ship skill exists and works identically to former @deploy.
- [ ] @deploy shows deprecation warning.
- [ ] `--status` after interruption shows resume instruction.

### Risks

| Risk | Mitigation |
|---|---|
| Evidence auto-commit creates noisy git history | Use single evidence commit per WS (not per file). Squash option in @ship. |
| --repair reconstructs wrong state | Require user confirmation before applying repair. Show diff from current. |
| @deploy rename breaks existing scripts | 2-version deprecation. Both names work. Warning only, not error. |

---

## SPEC-05: Escape Hatches and Recovery

### Meta

| Field | Value |
|---|---|
| Priority | P1 |
| Effort | Small |
| Scope | sdp (public repo: skills, CLI) |
| Root findings | F10, F12, F15, F29, F30, F42, F55 |
| Depends on | Nothing |
| Enables | User confidence (no dead ends) |

### Intent

Every SDP workflow has an escape hatch. Users can undo, restart, skip, or escalate at any point. No dead ends.

### Scope

#### In scope

1. **`sdp uninstall`**
   - Default: remove .sdp/, SDP hooks from settings.json, SDP symlinks
   - Preserve: docs/workstreams/, .sdp/evidence/, .sdp/log/ (user data)
   - `--purge`: remove everything including user data
   - `--dry-run`: show what would be removed
   - Restore settings.json from .sdp/backup/ if available

2. **`@feature --design-only "description"`**
   - New flag on @feature skill
   - Skips: @discovery, @idea, @ux
   - Runs: @design only
   - Output: workstream files in docs/workstreams/backlog/
   - For experienced users who know what to build and don't need interactive discovery

3. **@review post-max-retry options**
   - After 3 CHANGES_REQUESTED iterations, skill outputs:
     ```
     3 review iterations exhausted. Choose an option:

     1. @review --override "justification"
        Force approve. Requires written justification. Logged to evidence.

     2. @review --partial
        Approve checks that passed. Create issues for failed checks.

     3. @review --escalate
        Create issue for human review. Park the PR.
     ```
   - Each option is a valid next step, not a dead end

4. **`sdp reset --feature F042`**
   - Clears checkpoint for feature F042
   - Preserves workstream definitions (no data loss)
   - Next @oneshot starts execution from scratch (skips already-merged WS by checking git)
   - Confirmation prompt: "This will reset execution state for F042. Workstream files preserved. Continue? [y/N]"

5. **"If this fails" section in every skill**
   - Template addition to all 26 skill files:
     ```markdown
     ## Recovery

     | Symptom | Fix |
     |---|---|
     | Checkpoint error | `sdp-orchestrate --repair --feature <id>` |
     | Hook timeout | Check `.sdp/agent-constraints.yaml`, retry |
     | Spawn unavailable | Follow "Without Subagent Spawn" section above |
     | Unexpected output | `sdp reset --feature <id>` and retry |
     ```
   - Customized per skill (not all symptoms apply to all skills)

6. **Install method decision matrix in QUICKSTART.md**
   ```markdown
   | Method | When to use |
   |---|---|
   | `curl install.sh` | Default. New project or adding SDP to existing project. |
   | `--binary-only` | You want CLI only, managing prompts/hooks yourself. |
   | `git submodule add` | You want vendored control, pinned versions, CI reproducibility. |
   ```

#### Out of scope

- Undo for individual @build results (git revert handles this)
- Time-travel for checkpoints (too complex, use git history)
- Automatic retry after failure (user should decide)

### Acceptance Criteria

- [ ] `sdp uninstall` removes SDP artifacts, preserves user data, returns clean state.
- [ ] `sdp uninstall --purge` removes everything.
- [ ] `sdp uninstall --dry-run` shows plan without executing.
- [ ] `@feature --design-only "desc"` produces workstreams with 0 interactive questions.
- [ ] @review after 3 failures shows 3 explicit options (override/partial/escalate).
- [ ] `sdp reset --feature F042` clears checkpoint, preserves workstreams.
- [ ] Every skill file has a "Recovery" section.
- [ ] QUICKSTART.md has install method decision matrix.

### Risks

| Risk | Mitigation |
|---|---|
| sdp uninstall deletes user work | Default preserves user data. --purge requires explicit flag. Dry-run available. |
| --override used to bypass real issues | Justification required and logged to evidence. Visible in PR. |
| reset causes re-execution of completed WS | Check git for already-merged commits. Skip completed WS. |

---

## SPEC-06: Reference Hygiene

### Meta

| Field | Value |
|---|---|
| Priority | P2 |
| Effort | Small |
| Scope | sdp (public repo: CLAUDE.md, install.sh, skill files) |
| Root findings | F2, F3, F5, F7, F8, F44, F45, F47, F49, F50, F51 |
| Depends on | Nothing |
| Enables | Trust in documentation accuracy |

### Intent

Eliminate all dead references, fix all inconsistencies, ensure every documented feature actually exists.

### Scope

#### In scope

1. **@init skill: create or remove reference**
   - Option A: Create `prompts/skills/init/SKILL.md` wrapping `sdp init` with guided setup
   - Option B: Remove @init from Available Skills table in CLAUDE.md
   - Decision: prefer Option A (init is a natural entry point)

2. **sdp demo in CLAUDE.md**
   - Add to decision tree: "First time using SDP? -> `sdp demo`"
   - Position: first branch in the tree, before "New project?"

3. **Skill completion messages**
   - @vision: after completion, print:
     ```
     Created:
       docs/vision/VISION.md
       docs/vision/PRD.md
       docs/vision/ROADMAP.md
     Next: @feature "first feature name" to plan your first feature.
     ```
   - @feature: after completion, print:
     ```
     Created:
       docs/workstreams/backlog/00-001-01.md (Title)
       docs/workstreams/backlog/00-001-02.md (Title)
       docs/workstreams/backlog/00-001-03.md (Title)
     Feature ID: F001
     Next: @oneshot F001 to execute, or @build 00-001-01 for one workstream.
     ```

4. **install.sh IDE detection feedback**
   - After IDE detection: `echo "Detected: Claude Code. Creating .claude/ integration."`
   - On fallback: `echo "No IDE detected. Defaulting to .claude/ (Claude Code)."`
   - On override: `echo "Using SDP_IDE=$SDP_IDE override."`

5. **Fix Go-specific content in protocol docs**
   - CLAUDE.md Quality Gates: remove `go test`, `golangci-lint` specifics
   - Replace with language-agnostic: "Run project test suite", "Run project linter"
   - @go-modern: add note "This skill is Go-specific. For other languages, use your standard style guide."

6. **Fix Codex nested symlink**
   - Change `.codex/skills/sdp` from `../../prompts/skills` to `../prompts/skills`
   - Match pattern used by other harnesses

7. **Fix Cursor/Codex README references**
   - Cursor README: replace "See CLAUDE.md" with Cursor-specific content (or link to PROTOCOL.md)
   - Codex INSTALL.md: replace CLAUDE.md reference with Codex-specific content (or link to PROTOCOL.md)

8. **worktrees.json language-agnostic**
   - Remove Go-specific `cd sdp-plugin && go mod download`
   - Replace with generic: `"setup_commands_unix": []` (or detect from language profile)

#### Out of scope

- Rewriting entire documentation structure (that's SPEC-01)
- Creating new skills beyond @init
- Changing skill behavior

### Acceptance Criteria

- [ ] @init skill exists as file or is removed from all Available Skills references.
- [ ] `sdp demo` appears in CLAUDE.md decision tree.
- [ ] @vision prints created files and next step after completion.
- [ ] @feature prints created workstreams, feature ID, and next step after completion.
- [ ] install.sh outputs detected IDE name.
- [ ] CLAUDE.md Quality Gates section contains zero Go-specific commands.
- [ ] Codex skills symlink uses single `../` path.
- [ ] Cursor README does not reference CLAUDE.md.

### Risks

| Risk | Mitigation |
|---|---|
| Creating @init skill adds maintenance burden | Keep it thin: wrapper around `sdp init` with interactive guidance. <30 lines. |
| Removing Go from quality gates confuses Go users | @go-modern skill still available. Language profile supplies Go-specific commands. |

---

## Cross-Spec Dependency Map

```
SPEC-06 (hygiene)            -- independent, start anytime
SPEC-05 (escape hatches)     -- independent, start anytime
SPEC-04 (orchestrator)       -- independent, start anytime
SPEC-01 (progressive disclosure) -- independent, start anytime
  |
  +-- SPEC-02 (harness parity) -- needs tiered CLAUDE.md from SPEC-01
  +-- SPEC-03 (brownfield)     -- needs language profiles concept from SPEC-01
```

## Execution Recommendation

| Stream | Specs | Rationale |
|---|---|---|
| A (docs + UX) | SPEC-01 then SPEC-03 | Tiered docs unblock brownfield adoption guide |
| B (orchestrator) | SPEC-04 + SPEC-05 in parallel | Independent reliability + escape hatches |
| C (harness parity) | SPEC-02 after SPEC-01 | Needs tiered CLAUDE.md as input |
| D (hygiene) | SPEC-06 anytime | Quick wins, no dependencies |
