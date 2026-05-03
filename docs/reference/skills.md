# SDP Skills Reference

Status: canonical reference

Canonical design reference:

- `docs/reference/canonical-happy-path.md`
- `docs/plans/2026-04-05-canonical-sdp-happy-path-consistency.md`
- `docs/plans/2026-04-13-sdp-skill-architecture-design.md` (F125 intent model)

This document defines the public SDP skill surface and the internal or conditional skills that support the canonical workflow.

Skills are the guided control surface over the canonical stage model.
They are not a second workflow separate from board state, Beads, CLI, PR, or `QA/UAT`.
Skills own executable workflows, not module-local runtime facts. Package contracts,
dependency rules, and local gates belong in module-local `AGENTS.md` files per
[agent-instruction-cascade.md](agent-instruction-cascade.md).

Mode note:

- `Local Mode` may use skills without a full shared queue
- full board-backed `Operator Mode` still depends on Beads-backed operational truth

## ⚡ Quick Start: The Five Intents (F125)

**New to SDP?** Start here. SDP uses **intent-based skills** instead of a flat skill list.

The five intents answer "what do you want to do?":

1. **@understand** — "What is this codebase?" — Discovery, architecture, health, documentation
2. **@build** — "Create something new" — Features, prototypes, designs with TDD
3. **@fix** — "Something is broken" — Bugs, errors, failures from hotfix to systematic
4. **@review** — "Is this good enough?" — Code, architecture, security, readiness
5. **@operate** — "Keep it running" — Deploy, triage, backlog planning

**That's it.** Five intents cover 90% of workflows. Each intent has modes for fine-tuned control.

See `docs/plans/2026-04-13-sdp-skill-architecture-design.md` for the complete intent model.

## ⚠️ Legacy Skill Mapping (Deprecated)

> **STATUS:** Legacy skill names are deprecated (2026-04-17 → 2026-06-01).
> **MIGRATION:** See `docs/reference/migration-guide.md` for complete migration guide.
> **ACTION:** Use the five intents above — all legacy skills have 1:1 equivalents.

Old skills have been absorbed into the five intents. During the deprecation period, legacy names still work but will show warnings.

| Old Skill | Routes To | Intent Mode | Deprecation Warning |
|-----------|-----------|-------------|---------------------|
| @scout | @understand | quick | Use `@understand --depth quick` |
| @architect | @understand | standard | Use `@understand --depth standard` |
| @metrics | @understand | standard | Use `@understand --depth standard` |
| @spec | @understand | deep | Use `@understand --depth deep` |
| @landscape | @understand | standard/deep | Use `@understand --depth standard` (or `--depth deep`) |
| @index query | @understand | deep | Use `@understand --depth deep` |
| @feature | @build | feature | Use `@build --mode feature` |
| @idea | @build | idea | Use `@build --mode idea` |
| @design | @build | idea | Use `@build --mode idea` |
| @ux | @build | idea | Use `@build --mode idea` |
| @vision | @build | idea | Use `@build --mode idea` |
| @oneshot | @build | prototype | Use `@build --mode prototype`. Note: Checkpoint/resume now via `@operate --mode plan` |
| @prototype | @build | prototype | Use `@build --mode prototype` |
| @hotfix | @fix | quick | Use `@fix --mode quick` |
| @bugfix | @fix | systematic | Use `@fix --mode systematic` |
| @issue | @fix | systematic | Use `@fix --mode systematic` |
| @debug | @fix | investigate | Use `@fix --mode investigate` |
| @reality-check | @review | reality | Use `@review --dimension reality` |
| @verify-workstream | @review | readiness | Use `@review --dimension readiness` |
| @deploy | @ship | N/A | Use `@ship` (renamed for clarity) |
| @ci-triage | @operate | triage | Use `@operate --mode triage` |
| @plan | @operate | plan | Use `@operate --mode plan` |

**Note:** `@review` is **not deprecated** — it's the primary intent. Only dimension-specific aliases (`@reality-check`, `@verify-workstream`) are deprecated.

For the complete deprecation mapping and implementation guidance, see:
- `docs/reference/migration-guide.md` — Full migration guide with examples
- `docs/reference/internal/deprecated-aliases.md` — Machine-readable alias mapping

## Practices vs Skills

Some things are **NOT skills** — they're practices or policies that apply across intents:

| Practice | Applies To | How It Works |
|----------|-----------|--------------|
| @tdd | @build, @fix | Embedded as default workflow (test-first) |
| @guard | All intents | Pre-commit quality gate (automatic via hooks) |
| @go-modern | All intents | Language style convention (CLAUDE.md) |
| @think | All intents | Prompt technique (use everywhere) |
| @beads | All intents | Issue tracker (`bd` commands), not a skill |

These are **not invoked** — they're embedded in how intents work.

> **Migration note:** If you were using `@tdd` as a standalone skill, just use `@build` or `@fix` — test-first is now the default behavior. See `docs/reference/migration-guide.md`.

## Canonical Public Surface

The public happy path through intents:

- `@understand` — first-time codebase exploration
- `@build` — feature creation (idea → feature → prototype modes)
- `@fix` — bug resolution (quick → investigate → systematic modes)
- `@review` — quality gates (code → architecture → security → readiness)
- `@ship` — deployment and release (creates PR to main or tags release)
- `@operate` — operations and maintenance (triage → plan modes)

## Stage Mapping (Intent-Based)

| SDP stage | Primary intent | Intent mode | Result |
|-----------|---------------|-------------|--------|
| `vision` | `@understand` | deep | updated project map, complete knowledge base |
| `feature` shaping | `@build` | idea | design document, requirements |
| `feature` execution | `@build` | feature | implemented feature with tests, PR |
| `workstream` + `beads issue` mapping | `@build` or `@operate` | feature or plan | executable graph with dependencies |
| early `draft PR` + graph execution | `@build` | feature | active `PR` and progressing execution |
| bug resolution | `@fix` | quick/investigate/systematic | fixed issue with regression tests |
| engineering review | `@review` | code/arch/security/readiness | pass or typed findings |
| release or deploy path | `@operate` | deploy | deployed system with verification |

## Intent Reference

### `@understand`

**Modes:** quick (30s), standard (5-15 min), deep (15-30 min)

**Use when:**
- First time working with a codebase
- Need to understand architecture or dependencies
- Checking codebase health or technical debt
- Before starting feature work or major refactors

**Absorbs:** @scout, @architect, @metrics, @spec, @landscape, @index query

**Must emit:**
- quick: project card (language, LOC, dependencies, health snapshot)
- standard: complete understanding (architecture, health, risks)
- deep: knowledge base in `.sdp/manifest.md` + documentation + index

**Tools composed:** `sdp scout`, `sdp architect analyze`, `sdp metrics`, `sdp spec generate`, `sdp index build`

### `@build`

**Modes:** idea (brainstorm), feature (full cycle), prototype (fast)

**Use when:**
- Implementing new features or components
- Creating designs or system architectures
- Prototyping quick solutions
- User-facing work with clear acceptance criteria

**Absorbs:** @feature, @idea, @design, @ux, @vision, @oneshot, @prototype

**Must emit:**
- idea: design document with requirements, approach, tradeoffs
- feature: complete implementation with TDD tests, documentation, PR
- prototype: working code marked as [PROTOTYPE], TODO list for productionization

**Embedded practices:**
- @tdd: test-driven development is DEFAULT behavior

### `@fix`

**Modes:** quick (hotfix), investigate (debug), systematic (issue)

**Use when:**
- Known bugs with clear reproduction steps
- Test failures or CI breaks
- Production incidents or errors
- Error logs or stack traces available

**Absorbs:** @hotfix, @bugfix, @issue, @debug

**Must emit:**
- quick: minimal fix with regression test, RCA comment
- investigate: root cause, reproduction steps, proposed fix
- systematic: complete fix with comprehensive tests, docs, RCA document

**Embedded practices:**
- @tdd: regression test BEFORE fix

### `@review`

**Dimensions:** code (default), architecture, security, performance, readiness, reality

**Use when:**
- PR ready for engineering review
- Before merging to main
- Architecture or security concerns
- Release readiness verification

**Absorbs:** @review (all roles), @reality-check, @verify-workstream

**Must emit:**
- Pass/fail verdict with specific findings by dimension
- Beads issues for blocking findings
- Re-review criteria for failures

**Severity levels:** critical (blocking), high (should block), medium (warn), low (nit)

### `@operate`

**Modes:** deploy, triage, plan

**Use when:**
- Deploying to production or staging
- CI failures need investigation
- System monitoring or alerts
- Converting insights into backlog

**Absorbs:** @deploy, @ci-triage, @plan

**Must emit:**
- deploy: deployed system with verification (smoke tests, monitoring)
- triage: categorized failures, assigned issues, CI health report
- plan: structured backlog (Beads issues, dependencies, priorities)

**Embedded practices:**
- @guard: pre-deployment quality gate (automatic via hooks)

## Specialized Skills (Non-Intent)

These skills remain outside the five-intent model for specialized use cases:

### `@strataudit`

Purpose: run a document-backed strategy traceability audit as a reusable discovery capability with explicit trust boundaries

Use when:
- the user needs evidence-backed alignment analysis across strategy, architecture, design, or execution documents
- the user needs one of these modes: `corpus-audit`, `traceability-audit`, `coverage-audit`, `evidence-pack`, `report-redraft`
- the harness can inject a native runtime, or the repo has a configured compatible network runtime, or reusable `.strataudit/` artifacts already exist

Must emit:
- `.strataudit/report.json`
- `.strataudit/report.html`
- explicit runtime choice or artifact-only path
- key trust caveats and what is not claimed

References:
- `docs/STRATAUDIT.md`
- `docs/reference/strataudit-evidence-policy.md`
- `docs/reference/strataudit-runtime-policy.md`
- `docs/reference/strataudit-output-modes.md`

### `@review-readiness`

Purpose: verify workstream completion against acceptance criteria before PR submission

Use when:
- a workstream is nearly complete and needs final verification
- pre-PR quality gate verification is needed

Invocation: `@review --dimension readiness` (NOT as standalone `@review-readiness`)

Must emit:
- readiness verdict with any blocking items
- list of acceptance criteria met vs. unmet

Reference: `docs/plans/2026-04-13-f125-review-readiness-skill.md`

### `@llm-council`

Purpose: multi-model synthesis for complex decisions requiring diverse AI perspectives

Use when:
- complex architectural decisions need diverse AI perspectives
- risk assessment requires multiple model viewpoints
- controversial or ambiguous technical tradeoffs need resolution

Must emit:
- synthesized recommendation with confidence levels
- dissenting opinions and their rationale
- risk assessment with mitigations

### `@git-worktree`

Purpose: safety-first git worktree setup for parallel feature work

Use when:
- starting feature work that needs isolation from current workspace
- executing implementation plans with independent tasks in the current session

Must emit:
- isolated worktree with safety checks (gitignore, clean working tree, baseline tests)
- cleanup cadence recommendations

Reference: `docs/plans/2026-04-13-f129-git-worktree-skill.md`

### `@parallel-dispatch`

Purpose: delegate independent tasks to parallel subagent sessions

Use when:
- facing 2+ independent tasks that can be worked on without shared state or sequential dependencies
- tasks are truly independent (no shared state, no sequential dependencies)

Must emit:
- completed parallel work with consolidated results
- explicit dependency verification before dispatch

Reference: `docs/plans/2026-04-13-f129-parallel-dispatch-skill.md`

## Intent Detection and Routing

How does the system pick the right intent?

**Keyword signals:**
- "what is" / "analyze" / "understand" / "how does" → @understand
- "build" / "add" / "create" / "implement" → @build
- "fix" / "bug" / "broken" / "error" / "failing" → @fix
- "review" / "check" / "approve" / "ready" → @review
- "deploy" / "release" / "ci" / "plan" / "backlog" → @operate

**Context signals:**
- No `.sdp/scout.json` exists → @understand (first)
- PR open, changes staged → @review
- CI red → @operate (triage)
- Beads issue assigned → @fix or @build (from type)

**Explicit override:**
- `@understand --depth deep` → force specific intent + mode
- `@build --mode prototype` → force specific mode
- `@review --dimension security` → force specific dimension
- `@fix --mode systematic` → force specific mode
- `@operate --mode deploy` → force specific mode

## Skill Quality Rule

Every intent must answer clearly:

1. **When** it is the right entry point
2. **What** tools or workflows it composes
3. **What** artifact or verdict it emits

If a skill cannot answer those three, it should not exist.

## Operator Rule (Intent-Based)

When in doubt, follow the intent flow:

1. **First time with codebase?** → `@understand`
2. **Creating something new?** → `@build`
3. **Something broken?** → `@fix`
4. **Need to verify quality?** → `@review`
5. **Deploying or triaging?** → `@operate`

Everything else is a specialized skill or embedded practice.
