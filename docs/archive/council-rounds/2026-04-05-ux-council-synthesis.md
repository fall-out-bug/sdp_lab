# UX Council Synthesis: Final Recommendations

**Date:** 2026-04-05
**Status:** Final synthesis — supersedes individual proposals for prioritization and scope
**Inputs:**
- [UX Audit Results](2026-04-05-ux-audit-results.md) — original audit (Claude Code session)
- [UX Improvement Proposals](2026-04-05-ux-improvement-proposals.md) — original proposals
- [UX Improvement Specifications](2026-04-05-ux-improvement-specs.md) — original specs
- [Alternative Recommendations](2026-04-05-ux-alternative-recommendations.md) — Codex review
- [Improvement Addendum](2026-04-05-ux-improvement-addendum.md) — Cursor review
- [Alternative Perspective](2026-04-05-ux-audit-alternative-perspective.md) — OpenCode + Oh My OpenAgent review

---

## 1. What the Council Agreed On

Four independent reviews (original audit + three harness critiques) converge on these points:

1. **SDP's biggest UX problem is not surface friction — it's product identity.** Users cannot tell which product they're using: local CLI assistant, prompt bundle, queue-backed operator system, or private lab stack. Every doc tells a slightly different story.

2. **Time to first value is the single biggest adoption blocker.** SDP takes ~30 minutes to first success vs 1-2 minutes for benchmarks. This is not a documentation problem — it's a product design problem.

3. **Brownfield is the real market but the specs are overbuilt for it.** The right first move is a minimal safe overlay, not language profiles and tracker adapters.

4. **Measurement is absent.** All proposals define changes without defining how to tell if they worked.

5. **Harness parity is expensive and premature.** Fix the core flow in 1-2 harnesses before chasing 4-way parity.

6. **Trust before value.** Users need to predict what SDP will change before it changes anything. Terraform plan/apply pattern is the proven model.

---

## 2. Factual Corrections to Original Audit

Three findings were corrected by the OpenCode review after codebase verification:

| Finding | Original Claim | Correction | Revised Severity |
|---|---|---|---|
| **F6** | @feature chains 33 interactive questions. --quick/--auto undocumented. | Feature skill specifies "3-5 questions" in Step 1. 33 is theoretical max across all sub-skills. --quick and --auto exist in skill file but not in CLAUDE.md decision tree. | HIGH → **MEDIUM** (documentation gap, not design flaw) |
| **F52** | --status and --next-action both JSON-only. | --status outputs human-readable Markdown. Only --next-action is JSON-only. | HIGH → **MEDIUM** (partial, not systemic) |
| **F21** | install.sh overwrites settings.json with no merge. | install-project.sh uses merge by default. --no-overwrite-config flag exists. | HIGH → **MEDIUM** (worst-case description, not default behavior) |

**Impact on priorities:** These corrections reduce urgency of SPEC-01 (progressive disclosure) and SPEC-04 (orchestrator) relative to product identity and activation problems.

**Revised finding totals:** CRITICAL: 6, HIGH: 17 (was 20), MEDIUM: 25 (was 22), LOW: 5.

---

## 3. Revised Root Problem Model

The original audit identified 7 root problem clusters. The council reframes them into 5 causal themes:

| # | Theme | Original Clusters | Core Issue |
|---|---|---|---|
| **T1** | Product Identity Crisis | RP1 (disclosure), RP2 (Claude-first) | 4 different stories told by 4 different doc surfaces. No single honest default path. |
| **T2** | Activation Gap | New (all three reviews) | No guided path from "installed" to "I shipped something" in under 15 minutes. |
| **T3** | Trust Model Failure | RP4 (silent failures), RP5 (raw orchestrator), RP6 (no escape hatches) | Users cannot predict what SDP will do, cannot see what it did, cannot undo what went wrong. |
| **T4** | Brownfield Exclusion | RP3 (no adoption path) | SDP requires conformance from day one. Existing projects are blocked. |
| **T5** | Release Discipline Gap | RP7 (dead references) | Doc-code drift is a CI problem, not a documentation problem. Will recur after any cleanup. |

---

## 4. Revised Spec Portfolio

### New specs (from council)

| Spec | Source | What It Does |
|---|---|---|
| **SPEC-00: Product Truth + Activation Loop** | Codex + OpenCode (merged) | One honest default path + `sdp assess` / `sdp try` guided first experience |
| **SPEC-07: Write Boundaries** | OpenCode | Every stateful skill shows write plan before executing (terraform plan/apply pattern) |

### Modified specs (from council feedback)

| Spec | Change | Source |
|---|---|---|
| **SPEC-01** | Simplify: 3 fixed paths instead of adaptive engine. No auto-detection. | OpenCode |
| **SPEC-02** | Split: Phase 1 (quick wins: .cursorrules, AGENTS.md) at P1. Full parity at P2. | OpenCode + Codex |
| **SPEC-03** | Split: Phase 1 MVP (assess + try + uninstall) at P0. Phase 2 (profiles + adapters) at P2. | All three reviews |
| **SPEC-04** | Evidence auto-commit → opt-in, not default. @deploy deprecation → 3 versions. | OpenCode |
| **SPEC-06** | Upgrade to P0: CI gate for doc-code consistency, not one-time cleanup. | OpenCode |

### Added cross-cutting concerns

| Concern | Source | Resolution |
|---|---|---|
| Measurement plan | All three reviews | Added to SPEC-00 as metrics baseline |
| Risk-based @review | Codex + Cursor | Added to SPEC-01 alongside LOC tiers |
| Honest harness support policy | Codex | Added to SPEC-00 as support level table |
| Migration strategy for behavior changes | Cursor + OpenCode | Added to each spec as migration section |
| CI/headless story | Cursor | Added to SPEC-04 as `--no-commit` and exit code contract |
| Governance for @review --override | Cursor | Added to SPEC-05 |

---

## 5. Final Spec Definitions

### SPEC-00: Product Truth and Activation Loop

**Priority:** P0
**Theme:** T1 (Product Identity) + T2 (Activation Gap)
**Effort:** Medium
**Scope:** sdp (public repo)

#### Intent

Users can answer "What is the default way to use SDP?" in one minute. A new user reaches first shipped value in under 15 minutes without understanding the full system.

#### Deliverables

**A. Canonical Product Contract**

One source-of-truth document (`docs/PRODUCT_CONTRACT.md`) defining:

1. **Default public path:** Local Mode. One repo, one IDE, `sdp` CLI. Beads optional.
2. **Advanced path:** Operator Mode. Queue-backed, Beads required, multi-session.
3. **Stage model:** bootstrap → intake → shaping → execution → findings → delivery.
4. **Control surfaces:** CLI (primary), Skills (companion), Board (visibility in Operator Mode only).
5. **Harness support policy:**

   | Harness | Level | Meaning |
   |---|---|---|
   | Claude Code | Recommended | Full enforcement, hooks, spawn, beads sync |
   | OpenCode | Supported | Hooks (after port), agents, CLI. No spawn. |
   | Cursor | Compatible | Skills via .cursorrules, CLI. No hooks, no spawn. |
   | Codex | Compatible | Skills via AGENTS.md, CLI. No hooks, no spawn. |

6. **One default-path diagram** reused in QUICKSTART, PROTOCOL, harness READMEs.

All existing docs (QUICKSTART, PROTOCOL, harness READMEs, CLAUDE.md) updated to reference this contract. No contradictions.

**B. Activation Loop**

Three new CLI commands for zero-commitment first experience:

```
sdp assess                     # Read-only. Scans repo, shows recommendations, writes nothing.
sdp try "add error handling"   # Bounded: temp branch, one task, show results. User merges or discards.
sdp adopt                      # If value confirmed: create .sdp/, transition to full SDP flow.
```

Properties:
- `sdp assess` never writes to the repository. Stdout only.
- `sdp try` creates a temporary branch, executes one bounded skill (@build or @prototype), shows results. On discard: branch deleted, zero residue.
- `sdp adopt` equivalent to current `sdp init` but with merge behavior and adoption level support.
- At every step user can walk away with zero cleanup.

**C. Metrics Baseline**

Define and instrument 6 metrics in `.sdp/log/events.jsonl`:

| Metric | Purpose | Collection |
|---|---|---|
| Time to first value | Activation funnel | Local timestamp log |
| Step abandon rate | Where users give up | Local event log |
| `sdp reset` / `sdp uninstall` frequency | Trust failures | Local event log |
| Brownfield init completion rate | SPEC-03 validation | Local event log |
| Recovery success rate | Trust model health | Local event log |
| Second-session return rate | Retention | Opt-in only |

All local-first, no cloud dependency. Opt-in anonymous export for aggregate analysis.

#### Acceptance Criteria

- [ ] `docs/PRODUCT_CONTRACT.md` exists. States one default path, one advanced path, one stage model.
- [ ] QUICKSTART, PROTOCOL, all 4 harness READMEs link to and do not contradict the contract.
- [ ] Harness support levels (recommended/supported/compatible) documented and visible before install.
- [ ] `sdp assess` on a Python project with 0% coverage outputs recommendations without creating any files.
- [ ] `sdp try "description"` produces a working change on a temp branch within 10 minutes.
- [ ] After `sdp try`, if user discards: repository identical to pre-try state (verified by `git diff`).
- [ ] `sdp adopt` creates `.sdp/` structure equivalent to current `sdp init`.
- [ ] Metrics events logged for all 6 defined metrics.
- [ ] A first-time user can answer "What is the default way to use SDP?" after reading <50 lines.

#### Migration

Net-new commands and document. No breaking changes. Existing `sdp init` continues to work.

---

### SPEC-01: Simplified Progressive Disclosure

**Priority:** P0 (reduced scope from original)
**Theme:** T1 (Product Identity)
**Effort:** Small (was Medium)
**Scope:** sdp (public repo)

#### Changes from Original

- **Removed:** Adaptive engine, automatic context detection in @feature.
- **Replaced with:** 3 fixed entry paths (Try / Adopt / Scale) and explicit flags.
- **Added:** Risk-based @review triggers alongside LOC tiers.

#### Deliverables

**A. Three-Path Documentation**

CLAUDE.md restructured into ~50 lines with 3 paths:

| Path | Audience | Entry | Commands |
|---|---|---|---|
| **Try** | First-time user | `sdp assess` → `sdp try` | Zero commitment |
| **Adopt** | Adopting user | `sdp adopt` → `@feature --quick` → `@build` | Light commitment |
| **Scale** | Power user / operator | `@vision` → `@feature` → `@oneshot` → `@review` → `@ship` | Full protocol |

Decision tree in first 15 lines. All detail moved to PROTOCOL.md and docs/reference/.

**B. Explicit @feature Modes (no auto-detection)**

| Flag | Behavior | Questions |
|---|---|---|
| `@feature "desc"` (no flag) | Full pipeline: discovery → idea → ux → design | 3-5 per sub-skill |
| `@feature --quick "desc"` | @design only: generate workstreams | 0 |
| `@feature --auto "desc"` | Non-interactive full pipeline | 0 |

No automatic skipping based on existing files. Explicit flags only. Predictable, reproducible.

**C. Risk-Aware @review**

Two dimensions: change size AND risk profile.

| Dimension | Trigger | Effect |
|---|---|---|
| LOC < 50 | Small change | 2 reviewers (qa, tech-lead) |
| LOC 50-200 | Medium change | 4 reviewers (+ security, docs) |
| LOC > 200 | Large change | 7 reviewers (all) |
| auth/crypto/secrets touched | Risk signal | +security reviewer regardless of LOC |
| CI/deployment files touched | Risk signal | +devops reviewer regardless of LOC |
| DB migrations touched | Risk signal | +sre reviewer regardless of LOC |
| `--full` | Override | 7 reviewers regardless |
| `--quick` | Override | 2 reviewers regardless |

Risk detection: file path pattern matching (configurable in `.sdp/config.yml`).

#### Acceptance Criteria

- [ ] CLAUDE.md <= 60 lines. Contains 3 paths, decision tree in first 15 lines.
- [ ] `@feature --quick` completes with 0 interactive questions.
- [ ] `@review` on a 10-line non-auth change spawns 2 reviewers.
- [ ] `@review` on a 10-line auth change spawns 3 reviewers (2 + security).
- [ ] `@review --full` spawns all 7 regardless.
- [ ] Same command with same input always produces same reviewer set (no hidden context).

#### Migration

No behavior change for existing users unless they use new flags. `@feature` without flag = current behavior.

---

### SPEC-02: Harness Quick Fixes (Phase 1) + Parity Contract (Phase 2)

**Priority:** Phase 1 at P1, Phase 2 at P2
**Theme:** T1 (Product Identity)
**Effort:** Phase 1 Small, Phase 2 Large
**Scope:** sdp (public repo)

#### Phase 1 (P1): Quick Wins

Minimum viable harness support. No engine, no manifests.

| Harness | Fix | Effort |
|---|---|---|
| **Cursor** | Create `.cursorrules` with: SDP role, decision tree, available skills, quality gates, link to PROTOCOL.md. Remove CLAUDE.md reference from README. | Small |
| **Codex** | Create `.codex/AGENTS.md` with public-facing instructions. Fix nested symlink (`../../` → `../`). | Small |
| **OpenCode** | Port `sdp-omc-guard` hook from sdp_lab to public sdp. Add pre-tool-use.json. | Medium |
| **All** | Add "Without Subagent Spawn" section to 5 spawn-dependent skills (@review, @vision, @reality, @build, @feature). Single shared reference doc, not per-skill duplication. | Medium |
| **All** | Move `commands.json` content to `prompts/commands.yml` as canonical source. | Small |

#### Phase 2 (P2): Full Parity Contract

Only after Phase 1 validates and core flow works in 2+ harnesses:
- Capability manifests (`sdp-capabilities.yml`)
- Runtime capability detection
- Per-harness testing matrix
- Formal T1/T2/T3 tier contract

#### Acceptance Criteria (Phase 1)

- [ ] `.cursorrules` exists. Cursor agent receives SDP context automatically.
- [ ] `.codex/AGENTS.md` exists with public instructions (no lab content).
- [ ] `.opencode/hooks/pre-tool-use.json` exists in public sdp repo.
- [ ] One shared `docs/reference/FALLBACK_MODE.md` covers all spawn-dependent skills.
- [ ] `prompts/commands.yml` exists as canonical command mapping.
- [ ] Running `@review` in Cursor produces 7-section manual checklist.

#### Migration

Net-new files. No changes to existing behavior.

---

### SPEC-03: Brownfield Adoption (Phase 1 MVP + Phase 2 Full)

**Priority:** Phase 1 at P0, Phase 2 at P2
**Theme:** T4 (Brownfield Exclusion)
**Effort:** Phase 1 Medium, Phase 2 Large
**Scope:** sdp (public repo: install.sh, CLI), sdp_lab (CLI implementation)

#### Phase 1 (P0): Safe Overlay

Minimum viable brownfield adoption. No profiles, no adapters, no graduation levels.

**Deliverables:**

1. **Safe install** — merge, not overwrite:
   - Backup existing IDE config to `.sdp/backup/` before any modification.
   - Merge SDP hooks into existing settings.json (append to arrays, don't replace).
   - If `.cursorrules` exists: append SDP section.
   - `--preview` flag: show what will change before changing it.

2. **`sdp assess`** (from SPEC-00) — read-only scan:
   - Detect: language, test framework, CI, coverage estimate, file size distribution.
   - Output: recommendations, gap analysis, suggested adoption path.
   - Writes: nothing. Stdout only.

3. **`sdp try "task"`** (from SPEC-00) — bounded trial:
   - Temp branch, one @build or @prototype execution.
   - User reviews result, merges or discards.
   - Zero residue on discard.

4. **Disabled gates on first adoption:**
   - `.sdp/config.yml` with `adoption_mode: true`.
   - Quality gates (file size, coverage, TDD) disabled.
   - Evidence logging enabled (lightweight, non-blocking).
   - `sdp adopt --full` to enable all gates when ready.

5. **`sdp uninstall`:**
   - Remove .sdp/, SDP hooks, SDP symlinks.
   - Preserve user data (workstreams, evidence) by default.
   - `--purge` for complete removal.
   - `--dry-run` to preview.
   - Restore from `.sdp/backup/` where available.

6. **`docs/ADOPTION.md`** — step-by-step brownfield guide:
   - When to use each install method.
   - What sdp assess shows.
   - How sdp try works.
   - How to transition to full SDP.
   - How to uninstall.

**Not in Phase 1:** Language profiles, issue tracker adapters, graduation levels (0-3), multi-language quality commands.

#### Phase 2 (P2): Full Adoption Kit

Only after Phase 1 proves brownfield adoption works:

1. **Language profiles:** Auto-detected, editable. Go + Python + Node initially.
2. **Issue tracker adapters:** `beads` (default), `github` (via gh CLI), `none` (skip all tracker steps).
3. **Graduation levels:** Level 0 → 1 → 2 → 3 with explicit `sdp adopt --level N`.
4. **@go-modern** moved to optional language pack.

#### Acceptance Criteria (Phase 1)

- [ ] `sdp assess` on a Python project with 500-line files and 0% coverage outputs recommendations without creating files.
- [ ] Existing `.claude/settings.json` preserved after install (verified by diff of non-SDP keys).
- [ ] `sdp try "add input validation"` produces change on temp branch. Discard leaves repo clean.
- [ ] At adoption_mode: true, `@build` executes without quality gate failures on legacy code.
- [ ] `sdp uninstall` returns project to pre-SDP state.
- [ ] `sdp uninstall --dry-run` shows plan without executing.
- [ ] `docs/ADOPTION.md` exists.

#### Migration

`sdp init` continues to work as before (assumes greenfield, full gates). `sdp adopt` is the new brownfield entry. No behavior change for existing users.

---

### SPEC-04: Resilient Orchestrator

**Priority:** P1
**Theme:** T3 (Trust Model)
**Effort:** Medium
**Scope:** sdp_lab (cmd/sdp-orchestrate, internal/orchestrate)

#### Changes from Original

- Evidence auto-commit → **opt-in** (`--auto-commit` flag or `evidence.auto_commit: true` in config), not default.
- @deploy deprecation window → **3 versions** (was 2).
- Added: `--no-commit` mode for CI/headless.
- Added: exit code contract for CI integration.

#### Deliverables

1. **Human-readable --status by default** (JSON via `--json`).
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

2. **Checkpoint resilience:**
   - Validate JSON schema + hash integrity on load.
   - On corruption: explicit error message with repair suggestion.
   - `sdp-orchestrate --repair --feature F042`: recover from git history, evidence log, or filesystem state. Requires user confirmation before applying.
   - Atomic writes: temp file → fsync → rename.

3. **Evidence commit (opt-in):**
   - `sdp-orchestrate --advance --auto-commit`: commits `.sdp/evidence/` and `.sdp/checkpoints/` after advancing.
   - Default: no auto-commit. User commits manually or via hook.
   - Configurable: `.sdp/config.yml` key `evidence.auto_commit: false` (default).

4. **Inline progress for @oneshot:**
   ```
   [3/7] done 00-042-03 Login endpoint (2m 14s)
   [4/7] starting 00-042-04 Token refresh logic
   ```

5. **Rename @deploy → @ship:**
   - New skill: `prompts/skills/ship/SKILL.md`.
   - Deprecation wrapper at `prompts/skills/deploy/SKILL.md`: "@deploy is renamed to @ship. This alias will be removed in v1.2."
   - Both names work for 3 minor versions.
   - All docs updated to prefer @ship.

6. **Resume guidance:**
   - After interruption, `--status` shows: `Session interrupted. To resume: sdp-orchestrate --feature F042 --resume`
   - @oneshot skill: "If interrupted" section.

7. **CI/headless contract:**
   - `--no-commit` flag: execute but don't git commit anything.
   - Exit codes: 0 = success, 1 = failure, 2 = needs human input, 3 = checkpoint corrupted.
   - Artifact output directory: `--output-dir` for CI to collect evidence without .sdp/ writes.

#### Acceptance Criteria

- [ ] `--status` outputs human-readable text by default.
- [ ] `--status --json` outputs JSON.
- [ ] Corrupted checkpoint triggers: "Checkpoint corrupted. Run --repair to recover."
- [ ] `--repair` recovers from git history with user confirmation.
- [ ] Atomic writes: kill -9 during write does not corrupt checkpoint.
- [ ] `--auto-commit` flag commits evidence. Default: no auto-commit.
- [ ] @oneshot prints progress per-WS.
- [ ] @ship works. @deploy shows deprecation warning.
- [ ] Exit codes documented and stable for CI.

#### Migration

- `--status` output format changes from JSON to text. Scripts using `--status | jq` must add `--json`. Documented in release notes. 1-version overlap where both formats mentioned.
- @deploy → @ship: 3-version deprecation. Both work. Warning only.

---

### SPEC-05: Escape Hatches and Recovery

**Priority:** P1
**Theme:** T3 (Trust Model)
**Effort:** Small
**Scope:** sdp (public repo)

#### Changes from Original

- Added: governance rules for `@review --override` (from Cursor review).
- Added: consistent flag contract (`--dry-run`, `--abort`, `--resume`) across all stateful commands (from OpenCode review).

#### Deliverables

1. **`sdp uninstall`** — as defined in SPEC-03 Phase 1 (single owner, referenced here).

2. **`@feature --design-only "description"`** — direct path to workstreams, 0 questions.

3. **@review post-max-retry options:**
   ```
   3 review iterations exhausted. Options:
     @review --override "justification"    Force approve. Logged to evidence. Visible in PR.
     @review --partial                     Approve passing, file issues for failing.
     @review --escalate                    Create issue for human review.
   ```

   **Governance (from Cursor review):**
   - `--override` requires written justification (non-empty string).
   - Justification logged to evidence envelope and PR description.
   - Branch protection (if enabled) still requires human approval after override.
   - `--override` usage surfaced in `sdp doctor` output.

4. **`sdp reset --feature F042`** — clear checkpoint, preserve workstreams, confirmation prompt.

5. **"Recovery" section in every skill** — per-skill symptom→fix table.

6. **Consistent flag contract for all stateful commands:**

   | Flag | Behavior |
   |---|---|
   | `--dry-run` | Show what will happen, don't execute |
   | `--yes` | Skip confirmation prompts (for CI) |
   | `--abort` | Cancel mid-execution cleanly |
   | `--resume` | Continue from last known good state |

   Not every command needs all flags, but where applicable they use the same names and semantics.

7. **Install method decision matrix in QUICKSTART.md.**

#### Acceptance Criteria

- [ ] `@feature --design-only` produces workstreams with 0 questions.
- [ ] @review after 3 failures shows 3 options with clear next steps.
- [ ] `@review --override ""` (empty justification) is rejected.
- [ ] `@review --override "reason"` logged to evidence.
- [ ] `sdp reset --feature F042` clears checkpoint, preserves workstreams, requires confirmation.
- [ ] Every skill has "Recovery" section.
- [ ] `--dry-run` available on: `sdp adopt`, `sdp uninstall`, `sdp reset`, `sdp-orchestrate --advance`.

#### Migration

Net-new flags and commands. No behavior change for existing users.

---

### SPEC-06: Release Discipline Gates

**Priority:** P0 (upgraded from P2)
**Theme:** T5 (Release Discipline)
**Effort:** Small
**Scope:** sdp (public repo: CI, docs)

#### Changes from Original

- **Upgraded from P2 to P0** (per OpenCode review: "dead refs are release discipline, not polish").
- **Reframed:** From one-time cleanup to CI gate that prevents recurrence.
- **Added:** Automated checks, not just manual fixes.

#### Deliverables

1. **CI gate: reference integrity**
   - Every skill listed in CLAUDE.md must have a corresponding `prompts/skills/{name}/SKILL.md` file.
   - Every command in `commands.json` / `commands.yml` must have a skill file.
   - Every skill in harness READMEs must exist.
   - All symlinks must resolve.
   - Script: `scripts/check-references.sh` (exit 1 on violation).
   - Run in: GitHub Actions on every PR.

2. **One-time cleanup (blocked by CI gate):**
   - Create `@init` skill (wrapper around `sdp init`).
   - Add `sdp demo` to CLAUDE.md decision tree.
   - @vision and @feature: print "Created: [files]. Next: [command]." after completion.
   - install.sh: echo detected IDE.
   - Remove Go-specific from CLAUDE.md Quality Gates (move to language profile / @go-modern).
   - Fix Codex nested symlink.
   - Replace CLAUDE.md references in Cursor/Codex READMEs with PROTOCOL.md.
   - Make worktrees.json language-agnostic.

3. **Ongoing enforcement:**
   - CI gate runs on every PR.
   - New skills must be added to CLAUDE.md and commands.yml or CI fails.
   - Skill removal must remove all references or CI fails.

#### Acceptance Criteria

- [ ] `scripts/check-references.sh` exists and catches all reference integrity violations.
- [ ] CI runs check-references.sh on every PR.
- [ ] Zero reference violations in current codebase after cleanup.
- [ ] @init skill exists.
- [ ] `sdp demo` in CLAUDE.md decision tree.
- [ ] install.sh outputs detected IDE name.

#### Migration

No behavior changes. Additive CI gate + file fixes.

---

### SPEC-07: Write Boundaries

**Priority:** P0
**Theme:** T3 (Trust Model)
**Effort:** Small
**Scope:** sdp (public repo: skill files, CLI)

#### Intent

Users can predict what SDP will change before it changes anything. Inspired by terraform plan / apply separation.

#### Deliverables

1. **Write plan emission for stateful skills:**
   Every skill that creates or modifies files shows a plan before executing:
   ```
   @build 00-042-03 will:
     CREATE  internal/auth/token.go
     CREATE  internal/auth/token_test.go
     MODIFY  internal/auth/handler.go
     CREATE  .sdp/evidence/run-00-042-03.json

   Proceed? [y/N]
   ```

2. **`--dry-run` flag for all stateful skills:**
   Shows write plan without executing. Returns plan as structured output.

3. **`--yes` flag for CI/automation:**
   Skips confirmation prompt. Logs intent to `.sdp/log/events.jsonl`.

4. **Write plan logged:**
   Every execution logs its write plan to `.sdp/log/events.jsonl` for recovery and audit.

5. **Skills affected:**
   - @build, @oneshot, @feature, @design, @vision, @reality, @deploy/@ship
   - @review (writes review artifacts)
   - @hotfix, @bugfix
   - Not: @debug (read-only analysis), @think (scratchpad), @help

6. **Scope of write plan:**
   - Files created or modified (path + action: CREATE/MODIFY/DELETE)
   - Git operations (branch create, commit, push)
   - External operations (PR create, beads issue create)
   - NOT: internal .sdp/ state (checkpoints, logs) unless user-facing

#### Acceptance Criteria

- [ ] `@build --dry-run 00-042-03` shows write plan without executing.
- [ ] `@build 00-042-03` shows write plan and asks for confirmation before executing.
- [ ] `@build --yes 00-042-03` executes without confirmation (for CI).
- [ ] Write plan logged to `.sdp/log/events.jsonl`.
- [ ] All stateful skills (listed above) emit write plan.

#### Migration

New behavior: confirmation prompt before execution. Existing users who want current behavior use `--yes`. First version: opt-in via `.sdp/config.yml` key `write_boundaries: true`. Second version: default on.

---

## 6. Revised Priority Roadmap

```
P0 — Blocks Adoption and Trust (do now):
|
|-- SPEC-00: Product Truth + Activation Loop         [Medium]
|     One honest story. sdp assess / sdp try.
|     Metrics baseline.
|
|-- SPEC-07: Write Boundaries                         [Small]
|     Trust before value. Predict before execute.
|
|-- SPEC-03 Phase 1: Brownfield Safe Overlay          [Medium]
|     Safe install. sdp try. sdp uninstall.
|     Disabled gates. Adoption doc.
|
|-- SPEC-06: Release Discipline Gates                 [Small]
|     CI gate for references. One-time cleanup.
|     @init skill. sdp demo in decision tree.
|
|-- SPEC-01: Simplified Progressive Disclosure        [Small]
|     3 paths (Try/Adopt/Scale). Explicit flags.
|     Risk-aware @review.
|
P1 — Blocks Daily UX After Adoption (do next):
|
|-- SPEC-04: Resilient Orchestrator                   [Medium]
|     Human-readable output. Checkpoint resilience.
|     Inline progress. @ship rename. CI contract.
|
|-- SPEC-05: Escape Hatches & Recovery                [Small]
|     @feature --design-only. @review post-retry.
|     sdp reset. Recovery sections. Flag contract.
|
|-- SPEC-02 Phase 1: Harness Quick Fixes              [Small]
|     .cursorrules. AGENTS.md. OpenCode hooks.
|     Fallback doc. commands.yml.
|
P2 — Scale After Core Works (do later):
|
|-- SPEC-02 Phase 2: Full Parity Contract             [Large]
|     Capability manifests. Runtime detection.
|     Per-harness testing matrix.
|
|-- SPEC-03 Phase 2: Full Adoption Kit                [Large]
|     Language profiles. Tracker adapters.
|     Graduation levels.
```

### Dependency Graph

```
SPEC-06 (CI gates)           -- independent, start immediately
SPEC-07 (write boundaries)   -- independent, start immediately
SPEC-00 (product truth)      -- independent, start immediately
  |
  +-- SPEC-01 (disclosure)   -- needs SPEC-00 product contract as input
  |     |
  |     +-- SPEC-02 Ph1      -- needs restructured docs from SPEC-01
  |
  +-- SPEC-03 Ph1 (brownfield) -- shares sdp assess/try with SPEC-00
        |
        +-- SPEC-03 Ph2      -- P2, after Phase 1 validates

SPEC-04 (orchestrator)       -- independent, start immediately
SPEC-05 (escape hatches)     -- independent, start immediately
```

### Recommended Parallel Streams

| Stream | Specs | Lead Harness |
|---|---|---|
| **A: Product Identity** | SPEC-00 → SPEC-01 → SPEC-02 Ph1 | Any (doc-heavy) |
| **B: Trust & Reliability** | SPEC-07 + SPEC-04 + SPEC-05 | sdp_lab (Go code) |
| **C: Adoption & Discipline** | SPEC-06 + SPEC-03 Ph1 | sdp (install, CLI) |

Three streams can run in parallel. Each produces independently shippable value.

---

## 7. Delivery Phases (from Codex recommendation, revised)

### Phase 1: Make the product honest (SPEC-00, SPEC-06, SPEC-01)

Ship: aligned QUICKSTART, PROTOCOL, harness READMEs, product contract, support level table, CI reference gates.

**Success condition:** A new user can answer "What is the default way to use SDP?" in one minute.

### Phase 2: Make adoption safe (SPEC-07, SPEC-03 Ph1)

Ship: write boundaries, safe install, sdp assess, sdp try, sdp uninstall, adoption doc.

**Success condition:** A brownfield repo can try SDP without fear of config loss or gate failures.

### Phase 3: Make recovery obvious (SPEC-04, SPEC-05)

Ship: human-readable orchestrator, checkpoint repair, resume guidance, escape hatches, @ship rename.

**Success condition:** Recovery no longer depends on reading historical docs. After any failure, SDP provides one explicit next command.

### Phase 4: Expand support honestly (SPEC-02 Ph1)

Ship: .cursorrules, Codex AGENTS.md, OpenCode hooks, fallback mode doc, commands.yml.

**Success condition:** Each harness has a clear promise and a reliable minimum experience.

### Phase 5: Scale (SPEC-02 Ph2, SPEC-03 Ph2)

Ship: capability manifests, language profiles, tracker adapters, graduation levels.

**Success condition:** Daily operators across multiple languages and IDEs see less ceremony without losing control.

---

## 8. What Was Rejected from Council Feedback

| Proposal | Source | Reason for Rejection |
|---|---|---|
| Full subtraction strategy (remove skills/commands) | OpenCode | Valid principle but needs usage data first. Add metrics (SPEC-00), then subtract based on data. |
| Artifact lifecycle management | Codex | Real problem but not blocking adoption. Defer to future spec. |
| Team/org personas (team lead, enterprise) | Cursor + OpenCode | SDP is currently single-developer-first. Team scenarios are future. |
| Telemetry as separate spec | Codex + Cursor | Folded into SPEC-00 as metrics baseline. Separate spec is overhead. |
| Formal migration guide per-version | Cursor | Each spec now has a "Migration" section. Central guide after v1.0, not before. |

---

## 9. Open Questions for Product Decision

These require human input before workstream creation:

1. **Should `sdp try` use @build or @prototype?** @build requires workstream file. @prototype has relaxed gates but no workstream. Need a "trial" mode that produces a real change without workstream ceremony.

2. **Should adoption_mode be a boolean or a level?** Phase 1 proposes boolean (on/off). Phase 2 proposes levels (0-3). Need to decide if Phase 1 should leave room for levels or commit to boolean.

3. **Should write boundaries be opt-in or default?** SPEC-07 proposes opt-in first, default second version. Council feedback (OpenCode) suggests default from start. Product decision.

4. **Which is the second "Supported" harness?** Currently proposed: OpenCode (with hook port). Alternative: Cursor (larger market share but less infrastructure). Product decision.

5. **Is @ship the right name?** Alternatives: @merge (literal), @land (aviation metaphor matching "landing the plane"), @deliver. Naming is product, not engineering.
