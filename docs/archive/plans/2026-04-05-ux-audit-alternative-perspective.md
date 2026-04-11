# UX Audit: Alternative Perspective & Recommendations

**Date:** 2026-04-05
**Source:** [UX Audit Results](2026-04-05-ux-audit-results.md), [Improvement Proposals](2026-04-05-ux-improvement-proposals.md), [Improvement Specs](2026-04-05-ux-improvement-specs.md)
**Status:** Alternative analysis — contrarian review with codebase-validated corrections

---

## How to Read This Document

This is a companion to the original audit. It does not replace the audit or proposals — it challenges assumptions, corrects factual errors, identifies strategic gaps, and proposes a reshuffled priority order. Read the originals first, then this document for the counterpoint.

---

## 1. Factual Corrections

The audit contains 58 findings. Three of the HIGH-severity findings do not match the actual codebase state.

### F6: «33 questions in @feature» — Overstated

**Audit claim:** `@feature` chains 4 sub-skills with up to 33 interactive questions. `--quick` and `--auto` flags are undocumented.

**Actual state:** The feature skill file (`sdp/prompts/skills/feature/SKILL.md`) specifies a "quick interview with 3-5 questions" in Step 1. The number 33 appears to be a theoretical maximum obtained by summing all possible questions across all 4 sub-skills (@discovery + @idea + @ux + @design). Flags `--quick` and `--auto` **already exist** in the skill file.

**Corrected severity:** The flags exist but are not surfaced in the CLAUDE.md decision tree. This is a documentation gap (MEDIUM), not a design flaw (HIGH).

### F52: «JSON-only output» — Partially Incorrect

**Audit claim:** `sdp-orchestrate --next-action` and `--status` output raw JSON with no human-readable mode.

**Actual state:** `--status` outputs **human-readable Markdown** with headings and a JSON code block only for the next-action portion. Only `--next-action` is JSON-only. The default run path prints plain text (`INVOKE: @build ...`, `INVOKE: git push && gh pr create`).

**Corrected severity:** `--next-action` JSON-only is a real gap (MEDIUM). `--status` human-readable already works. The audit conflates two different commands.

### F21: «install.sh overwrites settings.json» — Describes Worst Case, Not Default

**Audit claim:** install.sh overwrites existing `.claude/settings.json` with no merge strategy.

**Actual state:** The real installer (`scripts/install-project.sh`) uses merge behavior by default: `sync_file` copies, `sync_link` uses `ln -sfn`. Flag `--no-overwrite-config` preserves existing files. `--overwrite-config` is opt-in, not default.

**Corrected severity:** The merge logic could be more robust (backup before merge, explicit conflict resolution), but "overwrites by default" is incorrect. MEDIUM, not HIGH.

### Impact on Prioritization

F6 and F52 are both rated HIGH and both feed into RP1 (progressive disclosure) and RP5 (orchestrator UX). Their correction reduces the urgency of SPEC-01 and SPEC-04 relative to other work.

---

## 2. Root Problem Clusters: Reframed

The audit identifies 7 root problem clusters (RP1-RP7). These describe symptoms accurately but misframe causes. Below is a reframing into 4 causal themes.

### Original → Reframed

| Original Cluster | Reframed Theme | Why |
|---|---|---|
| RP1: No progressive disclosure | **Surface Area Overload** | The problem isn't lack of adaptive layers — it's too many concepts exposed before value is delivered |
| RP2: Claude-first, not protocol-first | **Product-Boundary Confusion** | Internal authoring model leaks into product; the axis should be "outcome-first", not "protocol-first" |
| RP3: No brownfield adoption path | **No Low-Risk Overlay Mode** | Users don't want to "adopt a protocol" — they want help without repo takeover |
| RP4: Silent failures | **Trust Model Failure** | Unclear writes, unclear state transitions, unclear recovery |
| RP5: Orchestrator UX is raw | Subset of Trust Model | Covered by write boundaries and failure transparency |
| RP6: Missing escape hatches | Subset of Trust Model | Recovery is necessary but not the product |
| RP7: Dead references and gaps | **Release Discipline Failure** | Symptom of missing CI gates for doc/code consistency, not a documentation problem |

### The Missing Cluster: Activation Gap

None of the 7 clusters address the fundamental question: **how does a user go from "installed" to "I got value" in under 15 minutes?**

This is the single biggest adoption blocker. Benchmarks confirm it: gstack reaches first success in ~2 min, superpowers in ~1 min, SDP in ~30 min. The gap isn't documentation structure or harness parity — it's the absence of a guided, bounded first task that produces a visible outcome without requiring understanding of the full system.

---

## 3. Strategic Gaps in the Proposals

### 3.1 No Metrics or Measurement

All six SPECs propose changes without defining how to measure whether those changes work. There are no target metrics, no instrumentation plan, no feedback loop.

**What to add:**

| Metric | Purpose | Collection Method |
|---|---|---|
| Time to first value | Measure activation funnel | Local log, opt-in |
| Step abandon rate | Find where users give up | Local log, opt-in |
| `sdp reset` / `sdp uninstall` frequency | Detect trust failures | Local log |
| Fallback frequency by harness | Validate SPEC-02 necessity | Local log |
| Brownfield init completion rate | Validate SPEC-03 design | Local log |
| Second-use rate | Measure retention | Opt-in ping |

Implementation: local-first event log in `.sdp/log/events.jsonl`, opt-in anonymous export. No cloud dependency, no privacy risk by default.

### 3.2 No Migration Strategy

Every SPEC introduces new commands, flags, configs, or behavior changes. None describe what happens to existing SDP users during transition.

**What to add:**

| Change | Migration Requirement |
|---|---|
| `@deploy` → `@ship` | 2-version alias with deprecation warning |
| `.sdp/config.yml` (new) | Auto-create on first run, detect old layout |
| Adoption levels (Level 0-3) | Default to current behavior (full gates) |
| Language profiles | Auto-detect, don't require manual config |
| New @feature flags | Backward-compatible: no flag = current behavior |
| Capability manifests | New files, no existing state to migrate |
| `sdp uninstall` | Net-new command, no migration needed |

General principle: **every behavior change preserves existing behavior when possible, with explicit opt-in to new behavior.**

### 3.3 Persona Model Too Coarse

The audit defines 3 personas. The proposals implicitly target one: the motivated power user. Real-world adoption requires at least 4 distinct segments:

| Persona | Needs | Ceremonies | Primary friction |
|---|---|---|---|
| **Solo developer** | Fast local wins, zero setup | Near zero | Time investment before value |
| **Small team (3-5)** | Consistent outputs, shared context | Light | Onboarding multiple people |
| **Enterprise** | Auditability, admin controls, air-gapped | Moderate to heavy | Compliance and governance |
| **OSS maintainer / consultant** | Fast context-switch across many repos | Minimal per repo | Deep adoption per project |

SPEC-03 (brownfield) especially suffers from missing ICP: language profiles + tracker adapters are enterprise features sold as a solution for ">90% of projects."

### 3.4 Subtraction Strategy Missing

All 6 proposals are additive. They add engines, contracts, tiers, adapters, manifests, profiles, flags, sections. None propose removing anything.

This is a warning sign. UX improvements often come from **removing** surface area, not adding adaptive layers. Before building a Progressive Disclosure Engine, consider: which concepts can be hidden entirely from new users? Which commands can be removed or merged? Which flags are never used?

---

## 4. SPEC-by-SPEC Evaluation

### SPEC-01: Progressive Disclosure Engine — Overbuilt

**Right diagnosis, overengineered solution.**

The problem (too much information at once) is real. The proposed solution (tiered docs, adaptive @feature, scaled @review) adds significant machinery for marginal gain.

| Aspect | Proposed | Simpler Alternative |
|---|---|---|
| Tiered CLAUDE.md | 3-layer progressive disclosure | 3 fixed paths: **Try**, **Adopt**, **Scale** — no engine, just separate entry points |
| Adaptive @feature | Context detection, auto-skip sub-skills | 3 explicit modes: `@feature`, `@feature --quick`, `@feature --auto` — already partially exists |
| Scaled @review | LOC-based reviewer count (2/4/7) | 2 fixed profiles: `@review` (full) and `@review --quick` (2 reviewers) — predictable, testable |

**Unintended consequence:** Adaptive behavior creates non-reproducible results. Same command, different context, different output. Users can't predict what will happen.

**Recommendation:** Replace engine with explicit paths. Keep SPEC-01 scope but remove the "automatic context detection" complexity. Make it a documentation restructure + 2 new flags, not a new system.

### SPEC-02: Harness Parity Contract — Wrong Priority

**Right area, wrong priority and scope.**

Full T1/T2/T3 parity across 4 harnesses is a significant engineering investment. At the current stage, it risks becoming an expensive substitute for choosing a narrower, better product.

| Aspect | Proposed | Simpler Alternative |
|---|---|---|
| 4-harness parity | Full capability manifests, fallback paths, hook ports | **Core contract in 2 harnesses** (Claude Code + one other), community support for rest |
| Capability manifests | Per-harness YAML files read by skills | Feature-detect at runtime: "can this harness spawn?" — simpler, no new config surface |
| Fallback sections | In every spawn-dependent skill | Single "fallback mode" document referenced from all skills |

**Unintended consequence:** Parity contract creates a permanent compatibility tax. Every new skill must maintain full + fallback paths across all harnesses, even for features used by 5% of users.

**Recommendation:** Keep SPEC-02 but move to P1. Focus on making **one core flow** (init → feature → build → review → ship) work reliably in 2 harnesses. Add `.cursorrules` and `.codex/AGENTS.md` as quick wins (no engine needed).

### SPEC-03: Brownfield Adoption Kit — Right Problem, Overambitious

**The most important proposal, but needs significant scope reduction.**

The core insight is correct: SDP is unusable for existing projects. But the proposed solution (language profiles, issue tracker adapters, graduation levels) assumes users want formal adoption. Most want help without commitment.

| Aspect | Proposed | Simpler Alternative |
|---|---|---|
| `sdp init --adopt` | Full scan + config + adoption plan | `sdp assess` — read-only, writes nothing to repo |
| Graduation levels (0-3) | Formal progression with config | Overlay mode: SDP works without changing repo structure |
| Language profiles | Auto-detect + per-language commands | Start with Go + Python; defer Node/Rust/Java |
| Issue tracker adapters | Beads/GitHub/Linear/Jira/none | `issue_tracker: none` (skip all beads steps) as only non-Beads option |
| `sdp uninstall` | Full removal with backup | Keep as proposed — this is the right scope |

**Unintended consequence:** Tracker adapters create a maintenance burden that grows with ecosystem. Each adapter needs testing, versioning, auth handling, error recovery.

**Recommendation:** Split SPEC-03 into two phases:
- **Phase 1 (P0):** `sdp assess` (read-only) + `sdp try "task"` (bounded execution) + `sdp uninstall`. No language profiles, no adapters, no graduation levels.
- **Phase 2 (P1):** Language profiles + `issue_tracker: github` adapter. Only after Phase 1 proves adoption works.

### SPEC-04: Resilient Orchestrator — Mostly Right, Two Concerns

**The strongest proposal overall, but two items need rethinking.**

Human-readable output, checkpoint resilience, resume guidance — all correct. Two issues:

1. **Evidence auto-commit** — commits to the repository without user confirmation. This violates the trust model. Users should see what will be committed and explicitly approve. Consider: `--auto-commit` flag (opt-in, not default), or show staged evidence and ask for confirmation.

2. **Rename @deploy → @ship** — migration cost may exceed benefit. The confusion is real (F19), but it's a one-time learning moment, not an ongoing friction. If renaming, ensure both names work for 3+ versions (not 2), because existing documentation, tutorials, and muscle memory persist much longer than code.

**Recommendation:** Keep SPEC-04 largely as-is. Change evidence auto-commit to opt-in. Extend deprecation window for @deploy.

### SPEC-05: Escape Hatches and Recovery — Necessary but Not the Product

**All proposals are correct. Risk: treating recovery as a feature instead of fixing fragility.**

If `sdp uninstall`, `sdp reset`, and post-max-retry menus become hero features, the core flow is still fragile. Recovery is a safety net, not a value proposition.

**Additional recommendation:** Every stateful command should support a consistent set of flags:
- `--plan` / `--dry-run`: show what will happen without executing
- `--no-write`: execute logic but don't persist changes
- `--abort`: cancel mid-execution cleanly
- `--resume`: continue from last known good state

This is more systematic than individual escape hatches per skill.

### SPEC-06: Reference Hygiene — Underestimated

**Positioned as P2 "polish" but actually a release discipline problem.**

Dead references (@init missing, Go-specific content in agnostic docs, broken symlinks) indicate that SDP lacks automated checks for doc-code consistency. This will recur after any cleanup unless turned into a CI gate.

**Recommendation:** Upgrade to P0 with reduced scope:
- CI gate: every skill referenced in CLAUDE.md, commands.json, and harness READMEs must exist on disk
- CI gate: every CLI command referenced in docs must have a corresponding implementation
- CI gate: all symlinks must resolve
- One-time cleanup of existing violations

---

## 5. Missing Proposals

### SPEC-00: Activation & Trust Loop (New P0)

**Problem:** SDP has no guided path from "installed" to "I shipped something." The 30-minute time-to-first-value (vs 1-2 minutes for benchmarks) is not a documentation problem — it's a product design problem.

**Approach:**

```
sdp assess                    # Read-only: scans repo, shows recommendations, writes nothing
sdp try "add error handling"  # Bounded task: one skill execution, minimal residue
sdp continue                  # If value confirmed: transition to formal SDP flow
```

Key properties:
- `sdp assess` never writes to the repository. Output goes to stdout only.
- `sdp try` creates a temporary branch, executes one bounded task, shows results. User merges or discards.
- `sdp continue` creates `.sdp/` structure and transitions to the current init flow.
- At every step, the user can walk away with zero cleanup needed.

**Acceptance Criteria:**

- [ ] `sdp assess` on a Python project with 0% coverage outputs recommendations without creating any files
- [ ] `sdp try "description"` produces a patch or PR on a temporary branch within 10 minutes
- [ ] After `sdp try`, repository is unchanged if user discards the branch
- [ ] `sdp continue` creates `.sdp/` structure equivalent to current `sdp init`

### SPEC-07: Write Boundaries (New P0)

**Problem:** Users cannot predict what SDP will change in their repository. Every skill that writes files should declare its write scope before executing.

**Approach:**

Every stateful skill emits a write plan before execution:

```
@build 00-042-03 will:
  CREATE: internal/auth/token.go
  CREATE: internal/auth/token_test.go
  MODIFY: internal/auth/handler.go (add refresh method)
  CREATE: .sdp/evidence/run-00-042-03.json

Proceed? [y/N]
```

This is inspired by `terraform plan` before `terraform apply`. It gives users:
- Predictability: know what will change
- Trust: opportunity to review before execution
- Recovery: clear record of what was changed

**Acceptance Criteria:**

- [ ] Every skill that creates or modifies files emits a write plan
- [ ] `--dry-run` flag shows plan without executing
- [ ] `--yes` flag skips confirmation (for CI/automation)
- [ ] Write plan is logged to `.sdp/log/` for recovery

---

## 6. Revised Priority Roadmap

### P0: Blocks Adoption and Trust

| Spec | Scope | Effort | Rationale |
|---|---|---|---|
| SPEC-00: Activation & Trust Loop | New | Medium | Without this, unknown if users reach other improvements |
| SPEC-06: Reference Hygiene as CI Gate | Enhanced | Small | Dead references erode trust immediately |
| SPEC-03 Phase 1: Brownfield Overlay | Reduced | Medium | `sdp assess` + `sdp try` + `sdp uninstall` |
| SPEC-07: Write Boundaries | New | Small | Trust model fix — predictability before value |
| SPEC-04 subset: Failure Transparency | Reduced | Medium | Checkpoint resilience + resume + human-readable status |

### P1: Blocks Daily UX After Activation

| Spec | Scope | Effort | Rationale |
|---|---|---|---|
| SPEC-01: Simplified Docs Restructure | Reduced | Small | 3 fixed paths (Try/Adopt/Scale) instead of engine |
| SPEC-04 remainder: Orchestrator Polish | Remaining | Small | Evidence commit (opt-in), inline progress |
| SPEC-05: Escape Hatches | As proposed | Small | Recovery semantics across all stateful commands |
| SPEC-02 Phase 1: Core Harness Fix | Reduced | Medium | `.cursorrules` + `.codex/AGENTS.md` + basic parity for core flow |

### P2: Polish After Core Works

| Spec | Scope | Effort | Rationale |
|---|---|---|---|
| SPEC-02 Full: Harness Parity Contract | Full | Large | Wait until core journey validated |
| SPEC-03 Phase 2: Language Profiles + Adapters | Remaining | Large | Wait until brownfield overlay proves adoption |
| SPEC-01 Adaptive: Context-Aware Skills | Remaining | Medium | Wait until metrics justify complexity |
| @deploy → @ship rename | As proposed | Small | 3-version deprecation window |

### Dependency Graph (Revised)

```
SPEC-06 (CI gate)            -- independent, start immediately
SPEC-00 (activation loop)    -- independent, start immediately
SPEC-07 (write boundaries)   -- independent, start immediately
SPEC-03 Phase 1 (overlay)    -- depends on SPEC-07 (write boundaries enable overlay)
  |
  +-- SPEC-03 Phase 2 (profiles, adapters) -- P2, after Phase 1 validates
SPEC-04 (orchestrator)       -- independent, start immediately
SPEC-01 (docs restructure)   -- independent, start immediately
  |
  +-- SPEC-02 Phase 1 (harness fix) -- needs restructured docs
       |
       +-- SPEC-02 Full (parity contract) -- P2, after Phase 1 validates
SPEC-05 (escape hatches)     -- independent, start immediately
```

### Recommended Parallel Execution

| Stream | Specs | Can Start |
|---|---|---|
| A (activation) | SPEC-00 + SPEC-07 + SPEC-03 Phase 1 | Immediately |
| B (trust) | SPEC-04 + SPEC-05 + SPEC-06 | Immediately |
| C (docs) | SPEC-01 + SPEC-02 Phase 1 | Immediately (SPEC-02 Phase 1 after SPEC-01) |

---

## 7. Benchmark Gaps

The original audit benchmarks against 9 agent/tooling systems. This is useful for feature comparison but misses critical product-strategy references.

### Missing Benchmark Classes

| Class | Examples | What They Would Reveal |
|---|---|---|
| Boring successful CLIs | `gh`, `docker`, `terraform`, `vercel`, `fly` | Onboarding, safe defaults, plan/apply pattern, progressive commitment |
| Brownfield adoption tools | ESLint, Prettier, Black, Renovate, OpenRewrite | Incremental adoption, autofix trust, minimal-residue rollout |
| Protocol ecosystems | OpenTelemetry, LSP, Kubernetes API | Versioning cost, capability negotiation, compatibility guarantees |
| Team workflow products | Linear, GitHub Projects, Backstage | Role separation, governance, migration UX, collaboration |
| Trust-model tools | Git, Terraform plan/apply, Ansible check mode | Inspectability, reversibility, explicit write boundaries |

### Key Insight from Missing Benchmarks

Great adoption comes from **default paths**, not feature depth. Users trust tools that separate **inspect** from **apply**. Capability contracts are expensive and should be as small as possible. Brownfield success depends on **partial adoption**, not full-system buy-in.

---

## 8. Architecture Concerns

### New Coupling Introduced by Proposals

| Coupling Axis | Created By | Risk |
|---|---|---|
| Tiers × Harnesses × Capabilities × Docs × Tests | SPEC-01 + SPEC-02 | Matrix explosion; every new feature touches multiple dimensions |
| Languages × Profiles × Tracker Adapters × Brownfield States | SPEC-03 | Each new language or tracker adds a combinatorial test surface |
| Orchestrator × Git × Checkpoints × Evidence | SPEC-04 + SPEC-05 | State spread across 4 stores; recovery requires syncing all 4 |

### New Failure Modes

1. **Hidden degradation:** Fallback paths produce "success" without users realizing guarantees changed. A user runs @review in Cursor, gets a single-agent checklist, doesn't know they missed 6 reviewer perspectives.

2. **Non-reproducible behavior:** Adaptive review or feature depth changes output based on context. Same command today and tomorrow may produce different results. CI and team workflows depend on reproducibility.

3. **Split-brain state:** Repository state, checkpoint state, evidence state, and external tracker state can diverge. No single recovery command can reconcile all four.

### Maintenance Burden

| Proposal | Ongoing Cost | Acknowledged? |
|---|---|---|
| Capability manifests | Versioning + verification per release | No |
| Language profiles | Per-language testing + updates as ecosystems evolve | Partially (deferred to community) |
| Tracker adapters | Auth, rate limiting, API changes per service | No |
| Fallback sections in skills | Must stay synchronized with spawn behavior | Partially |
| Docs tiering | Cross-referencing between 3 layers | No |

---

## 9. Summary of Recommendations

### Keep (with minor changes)

- **SPEC-04:** Resilient Orchestrator — strongest proposal. Change evidence auto-commit to opt-in.
- **SPEC-05:** Escape Hatches — correct and necessary. Add consistent `--plan/--dry-run/--abort/--resume` flags.
- **SPEC-06:** Reference Hygiene — upgrade from "polish" to CI gate. Make it release discipline, not one-time cleanup.

### Simplify

- **SPEC-01:** Replace adaptive engine with 3 fixed paths. Remove context-detection complexity.
- **SPEC-02:** Reduce to core contract in 2 harnesses. Defer full parity to P2.
- **SPEC-03:** Split into Phase 1 (overlay mode, P0) and Phase 2 (profiles + adapters, P2). Remove graduation levels from Phase 1.

### Add

- **SPEC-00:** Activation & Trust Loop — guided path from install to first value in under 15 minutes.
- **SPEC-07:** Write Boundaries — every stateful skill declares its write scope before executing.
- **Measurement layer:** Local event log, opt-in telemetry, target metrics per SPEC.
- **Migration strategy:** Compatibility guarantees for every behavior change.

### Deprioritize

- Full T1/T2/T3 harness parity (SPEC-02 full scope) to P2
- Adaptive behavior / context detection (SPEC-01 engine) to P2
- Issue tracker adapters beyond `none` (SPEC-03 adapters) to P2
- Command renames (@deploy → @ship) to P2 with 3-version window

### The Core Shift

The original proposals add mechanisms to manage complexity. The alternative approach reduces exposed surface area and proves value before asking for commitment. The question isn't "how do we make all 26 skills work in 4 IDEs with adaptive behavior" — it's "how do we get a new user to ship their first PR in under 15 minutes, safely, reversibly."
