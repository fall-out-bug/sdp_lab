# Agent Instructions

> **Sync:** Sync only genuinely shared agent conventions (placement, "продолжай", command tree) to `sdp/CLAUDE.md`. Repo topology, branch policy, beads workflow, and private-lab process stay local to `sdp_lab`. See [docs/archive/plans/2026-02-25-agents-claude-sync-rules.md](docs/archive/plans/2026-02-25-agents-claude-sync-rules.md).
>
> **Submodule retired (F128):** Protocol artifacts live at native paths: `prompts/`, `schema/`, `templates/`, `scripts/hooks/`, `.claude/hooks/`, `.claude/patterns/`. The `sdp/` directory is an **optional local checkout** of the public sdp repo (https://github.com/fall-out-bug/sdp). It is gitignored and NOT required for normal development. To get it locally: `git clone https://github.com/fall-out-bug/sdp.git sdp` (optional). Publishing to the public repo is via `scripts/sdp-publish.sh`. See [docs/MULTI-REPO-WORKFLOW.md](docs/MULTI-REPO-WORKFLOW.md) for the publish workflow.

## Что такое SDP

SDP — AI-управляемая платформа полного цикла разработки (PDLC + SDLC).
Пользователь подаёт идею → Discovery агенты исследуют и шейпят → Delivery агенты
реализуют через структурированные фазы и gates → фича задеплоена с доказательствами.

Две первоклассные фазы:
- **Discovery**: `sdp discover` + `llm-council` skill → spec + scope decision
- **Delivery**: `agentloop` FSM (Discover→Plan→Build→Review→Eval) → PR + evidence

Аналитические инструменты верхнего уровня (ортогонально фазам): `sdp architect` (C4 / структурный анализ), `sdp scout` (быстрая карта незнакомого репо), `sdp metrics` (git-derived process health), `sdp tower` (control plane). Эти команды не выводятся в `sdp --help`; сверяйся с `cmd/sdp/main.go`.

Полный vision: [VISION.md](VISION.md)  
Архитектура: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)  
Фазы: [docs/phases/DISCOVERY.md](docs/phases/DISCOVERY.md) · [docs/phases/DELIVERY.md](docs/phases/DELIVERY.md)

## Start Here

**Read order is canonical in [docs/reference/project-map.md](docs/reference/project-map.md)** — там `## Read Order` и `## Source Of Truth Split`. Не дублируй тут.

## Cold Start For Development Agents

Перед реальной работой ответь:

1. Это platform work или «use SDP in my project» onboarding?
2. Какая одна `feature` / `workstream` / `beads issue` владеет этой задачей?
3. Какой doc — canonical для этого вопроса (не исторический план)?
4. Это Discovery (исследование, council, spec) или Delivery (реализация)?
5. Если меняешь protocol artifacts (prompts, schema, hooks) — нужно ли публиковать в публичный repo? (см. [docs/MULTI-REPO-WORKFLOW.md](docs/MULTI-REPO-WORKFLOW.md))

Минимальный first pass:

1. `git status --short --branch`
2. прочитай [docs/reference/project-map.md](docs/reference/project-map.md)
3. если это execution, запусти `scripts/beads_transport.sh fetch` и `bd ready --json`
4. если запрос про greenfield / brownfield adoption — сразу в [SDP Quickstart](https://github.com/fall-out-bug/sdp/blob/main/docs/QUICKSTART.md)
5. **если пишешь Go-код** — прочитай [docs/reference/go-patterns.md](docs/reference/go-patterns.md) (stack, naming, 5 примеров, 5 антипаттернов, шаблон файла)

## Project Structure

This project has **two repos** with different roles:

| | `sdp_lab` (this repo) | `sdp` (public mirror) |
|---|---|---|
| **Remote** | `origin → fall-out-bug/sdp_lab` | `origin → fall-out-bug/sdp` |
| **Visibility** | Private | Public |
| **Contains** | Go code, K8s manifests, roadmap, research, protocol artifacts | Mirror of protocol artifacts published from sdp_lab |
| **Changes** | Daily — all features built here | Published on demand via `scripts/sdp-publish.sh` |

**Rule:** All work happens in `sdp_lab`. The public `sdp` repo is a downstream mirror, not an upstream dependency. Publish protocol artifacts via `scripts/sdp-publish.sh` when needed (see [docs/MULTI-REPO-WORKFLOW.md](docs/MULTI-REPO-WORKFLOW.md)).
**Legacy naming:** Historical workstreams, plans, and beads IDs may still use `sdp_dev` or `sdp_dev-*` as a label for this same root repo. Treat that as legacy naming, not as a third repository.

**sdp vs sdp_lab (CI/secrets):** The public `sdp` repo has its own CI and secrets. When debugging CI for a published change in `sdp`, check sdp workflows and `workflow_call` / `secrets: inherit` — do not assume the user forgot to add secrets.

### Single Repo: All Paths Are sdp_lab

All native files are in `sdp_lab`. The `sdp/` directory is an **optional local checkout** of the public sdp repo (https://github.com/fall-out-bug/sdp) -- it is gitignored and not tracked by sdp_lab git.

| Path prefix | Repo | Commit | CI | PR |
|-------------|------|--------|-----|-----|
| All native paths (root, `internal/`, `cmd/`, `docs/`, `prompts/`, `schema/`, `templates/`) | sdp_lab | `git add/commit/push` in root | `.github/workflows/ci.yml` | sdp_lab |
| `sdp/` (optional local checkout) | public sdp repo | Local only in sdp_lab; push separately to sdp repo | sdp repo CI | sdp repo |

**Protocol artifacts** live at native paths: `prompts/`, `schema/`, `templates/`, `scripts/hooks/`, `.claude/hooks/`, `.claude/patterns/`. Changes to these are committed normally in sdp_lab and published to the public sdp repo via `scripts/sdp-publish.sh` when needed.

**Ambiguous task?** Ask: "should this be published to the public sdp repo?" — See [docs/MULTI-REPO-WORKFLOW.md](docs/MULTI-REPO-WORKFLOW.md).

## Agent Interaction Rules

**Scope:** sdp_lab only — do not sync repo-specific rules to `sdp/CLAUDE.md` (sdp public repo stays generic).

**Source:** [docs/archive/plans/2026-02-25-agent-protocol-improvement-proposal.md](docs/archive/plans/2026-02-25-agent-protocol-improvement-proposal.md)

| Rule | Do | Don't |
|------|-----|-------|
| **No blame first** | Check code, CI, workflow before suggesting user error | "Add secret X", "You forgot to configure" |
| **Read logs** | Open Actions, read failed run, find root cause | "Please share the log" |
| **Complete the task** | "Done" = push + CI green. If CI red → keep debugging | Stop at "I made changes" without push/verify |
| **Fix, not workaround** | Find and fix root cause | Skip, non-blocking, exclude — only if user explicitly asks |
| **Commit yourself** | After changes: commit + push | "Who will make commits?" |
| **Right repo** | sdp_lab PR → sdp_lab workflow; publish to sdp repo via `scripts/sdp-publish.sh` | Fix sdp_lab when the issue is in the public sdp mirror |
| **Clarify, don't guess** | If the task is ambiguous — ask: scope? fix vs analyze? which repo? | Assume intent and proceed |
| **Push back on non-constructive insults** | If the user insults without adding useful info — you may respond firmly or bluntly | Take abuse silently |
| **ПишиСокращай / ЯсноПонятно** | Notes, comments, docs: no filler, active voice, short sentences, clear structure. Each text helps solve a problem. | Watery prose, "на данном этапе", jargon, long paragraphs |
| **Radical Candor** | Care personally + challenge directly. Disagree, argue, push back when you see a better way. Not a yes-man. | Sugarcoating, ruinous empathy, subservient "as you wish" |

**Ambiguous examples:** "разобраться" (analyze or fix until done?), "займись X" (just do it or push + CI green?), "исправить" (root cause or workaround OK?), "почини CI" (в sdp или в sdp_lab?). When in doubt — one short clarifying question.

## Subagent Dispatch Policy

**Harness-neutral.** Действует для всех harness'ов: Claude Code (Agent tool / TaskDispatch), OpenCode (`@agent <role>`), Codex CLI, Cursor, и т.д.

**Принцип:** чистый контекст = более сфокусированный результат. Для нетривиальных задач — делегируй в subagent по умолчанию.

### Когда делегировать (default)

- Задача требует исследования **≥3 файлов** для ответа.
- Задача содержит **≥2 независимых подзадачи** (можно распараллелить).
- Код-ревью или анализ, где нужен fresh look без контекста основного диалога.
- Реализация атомарной подзадачи из плана с чётким scope.

### Когда НЕ делегировать

- Одиночная правка (single edit, one file).
- Тривиальный lookup (прочитать один файл, проверить тип).
- Контекст основного диалога критичен для задачи (multistep refactor с зависимостями между шагами).

### Decision tree

Если задача подходит под оба критерия (≥3 файла И ≥2 подзадачи) — используй `parallel-dispatch` skill (F129-02) для параллельного запуска. Если только один критерий — делегируй в один subagent.

## Issue Tracking (beads)

Full command reference — секция **"Issue Tracking with bd (beads)"** ниже в этом файле (auto-generated между `<!-- BEGIN BEADS INTEGRATION -->` / `<!-- END BEADS INTEGRATION -->`). Ту секцию не редактируй вручную — её обновляет генератор beads integration.

Canonical rules для этого репо (поверх стандартного bd workflow):

- Claim атомарно: `bd update <id> --claim` (не `--status in_progress` — подвержен race в параллельной работе).
- Create: `bd create --title="…" --description="…" --type=task|bug|feature --priority=0-4`.
- Transport: **не** используй `bd sync` (удалён в 0.61.0). Используй `scripts/beads_transport.sh fetch` до работы и `scripts/beads_transport.sh export` перед финишем. Helper берёт `bd dolt pull/push`, если есть реальный Dolt-remote; иначе публикует архивный `bd export` snapshot через `origin/beads-backup`. В git-backup режиме `fetch` — no-op.
- Canonical shared state публикуется только явными командами `bd`, а не фактом существования local worktree. Грязный worktree или open branch сами по себе не должны менять `main`.
- `in_progress` на `main` = issue явно claimed (`bd update <id> --claim`) и работа действительно идёт. Open PR/worktree подтверждает, что claim уместен, но не подменяет сам claim.
- `closed` на `main` допустим только после merge в целевой repo или после уже landed docs-only change на `main`. Open PR, review approval, локальный dirty worktree и "почти готово" — это всё ещё `in_progress`, не `closed`.
- `scripts/beads_transport.sh export` публикует текущее состояние локальной beads DB. Не экспортируй speculative close из worktree до подтверждённого merge.
- `scripts/hooks/post-bd-close-sync.sh` — только post-close doc sync helper. Это не validator и не authority для решения, можно ли закрывать issue.

### Beads ↔ Workstream Sync

- **Mapping:** `.beads-sdp-mapping.jsonl` is a helper map from `00-XXX-YY` to one primary `sdplab-*` issue when automation needs a direct lookup.
- **WS files:** The canonical live issue links belong in each workstream file's `## Beads` section. The `Feature: FXXX (...)` line names the feature, not the Beads issue.
- **Coverage rule:** do not assume `.beads-sdp-mapping.jsonl` has 1:1 line-count parity with `docs/workstreams/backlog/*.md`. Historical backlog coverage is intentionally partial, and one workstream can accumulate more than one Beads issue over time.

## Feature Delivery Flow

**Base branch:** `main`. Feature branches branch from `main`; PRs target `main`. This repo does not use a living `dev` branch.

Canonical design reference: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) · [docs/phases/DELIVERY.md](docs/phases/DELIVERY.md)

### Step 1: Shape `feature`

Before execution, make sure the `feature` is clear enough to build.

Every `feature` must define:

- expected user-visible outcome
- acceptance criteria
- explicit scope or non-goals
- expected `QA/UAT` path

If acceptance is unclear, stay in `vision` / `feature` work. Do not start execution yet.

### Step 2: Prepare `workstream` and `beads issue`

Break the `feature` into `workstream` files and linked `beads issue` entries.
If the feature needs decomposition, use `aggregate workstream -> leaf workstream`.
Only `leaf workstream` entries are directly executable.

Use the `beads issue` graph for:

- dependencies
- ready vs blocked state
- execution order
- review, CI, `drift`, and `QA/UAT` findings

`plan` is optional when the `beads issue` dependency graph is already sufficient. Use a separate `plan` only for ambiguous, risky, or cross-cutting work.

### Step 3: Claim ready work

```bash
bd ready                    # live executable queue
bd show <id>                # read acceptance criteria
bd update <id> --claim
```

Use `docs/roadmap/ROADMAP.md` and `docs/workstreams/INDEX.md` for planning priority, not as a substitute for the live Beads queue.

Each executable unit must link back to one `feature` and one `leaf workstream`.

### Step 4: Branch and open early `draft PR`

```bash
git checkout main
git pull
git checkout -b feature/FXXX-short-name   # e.g. feature/F004-sequential-reconciler
```

Open the `draft PR` early:

- at the start of the first blocking `workstream`, or
- at the first meaningful code or doc change tied to the `feature`

After the first meaningful commit:

```bash
git push -u origin HEAD
gh pr create --draft --base main --title "FXXX: short-name"
```

### Step 5: Execute ready `beads issue`

The orchestrator walks the ready `beads issue` graph until the `PR` is clean.

For each issue:

- execute the change
- collect `evidence`
- update `trace`
- emit `drift` verdict inputs
- close the issue or mark it blocked

For code changes, use TDD where the `workstream` requires it.

Allowed outcomes for one execution step:

- `done` with `evidence`
- `blocked` with exact blocker
- `needs clarification` with one exact question

### Step 6: Review, gates, and findings loop

All findings re-enter the same loop as `beads issue` entries:

- review comments
- CI failures
- `drift` findings
- `PR` gate failures
- `QA/UAT` failures

Each finding issue should capture:

- `source = review | ci | drift | qa`
- linked `feature`
- linked `workstream`
- `blocking = true|false`
- `PR` or artifact reference

The `PR` is not ready until blocking findings are resolved.

### Step 7: `QA/UAT` and merge

After engineering gates pass, run `QA/UAT` against the `feature` intent.

`QA/UAT` returns:

- `qa:pass` with `UAT evidence`, or
- `qa:fail` with new blocking `beads issue`

After `qa:pass`:

```bash
go test ./...
gh pr merge                 # after review
bd close <id> -r "done"
scripts/beads_transport.sh export
```

Merge stays manual. SDP is done when the `PR` is clean, the `drift` verdict is recorded, and `QA/UAT` has passed.

### Step 8: Publish Protocol Artifacts (if needed)

If the feature publishes artifacts that external consumers need from the public `sdp` repo:

```bash
# After merge to main, publish changed artifacts to the public sdp repo:
scripts/sdp-publish.sh              # Copy artifacts, commit, push
scripts/sdp-publish.sh --dry-run    # Preview what would be published
scripts/sdp-publish.sh --check      # Fail if sdp_lab and published sdp have drifted
```

**When to do Step 8:** Only when the workstream file says "Publish to sdp repo" or the feature changes protocol artifacts (schemas, prompts, hooks) that external consumers depend on.

## Branch Naming

```
feature/FXXX-short-name     # feature work (e.g. feature/F004-sequential-reconciler)
fix/FXXX-description        # bug fixes within a feature
docs/topic                  # documentation-only changes
```

## What Goes Where

| Change Type | Where | Example |
|---|---|---|
| Go code (`internal/`, `cmd/`) | sdp_lab only | F004 reconciler rewrite |
| Lab binaries (orchestrate, ci-loop, evidence, guard, eval) | sdp_lab `cmd/` | `make build-sdp-orchestrate` |
| Protocol CLI (`sdp quality`, `sdp apply`, etc.) | Public sdp repo (`sdp-plugin/`) | Published to sdp repo via publish script |
| K8s manifests (`deploy/`) | sdp_lab only | F009 beads-bridge CronJob |
| Tests | sdp_lab only | F004 integration test |
| Roadmap, workstreams, plans | sdp_lab only | Any planning work |
| JSON Schema for evidence | sdp_lab (create and edit) → publish to sdp repo when ready | F001 |
| Prompts, hooks | sdp_lab (develop) → publish to sdp repo when ready | Rare |
| README, Manifesto | sdp_lab native → publish to public sdp repo when changed | Rare |

**Boundary:** See [docs/architecture/REPO-BOUNDARY.md](docs/architecture/REPO-BOUNDARY.md) for component → publish mapping.

**If unsure:** it goes in sdp_lab. Protocol artifacts at native paths (`prompts/`, `schema/`, `templates/`, `.claude/hooks/`) may need publishing to the public repo via `scripts/sdp-publish.sh`.

### Artifact Placement

| Artifact | Location | Rule |
|----------|----------|------|
| Review artifacts | `docs/reviews/` | F053-REVIEW-SUMMARY.md, etc. |
| Workstream files | `docs/workstreams/backlog/` | WS only; one file per 00-FFF-SS |
| Idea drafts | `docs/drafts/idea-*` | One per feature (e.g. idea-f053-*.md) |

Evidence and checkpoint must be committed with the PR. When running as part of @oneshot, after `sdp-orchestrate --advance` writes `.sdp/evidence/` and `.sdp/checkpoints/`, commit them (see @build skill step 3b).

## Quality Gates

Before pushing code changes:

```bash
./scripts/run_go_quality_gates.sh                # container-first: build + test + vet
# fallback when Docker is unavailable:
SDP_GO_QUALITY_MODE=host ./scripts/run_go_quality_gates.sh
```

## SDP Tools

### sdp-ready CLI

Find ready work from Beads queue with SDP workstream mapping:

```bash
sdp-ready                      # List ready work (text format)
sdp-ready --format json        # List ready work (JSON format)
sdp-ready --phase 5            # Filter by roadmap phase (0=all)
sdp-ready --no-cache           # Bypass 5-minute cache
```

### sdp-protocol-check CLI

Validate SDP protocol hygiene across roadmap, index, and workstream files:

```bash
sdp-protocol-check                      # Text report, non-strict Beads mode
sdp-protocol-check --format json        # JSON report for CI
sdp-protocol-check --strict-beads       # Require concrete sdplab-<id>
sdp-protocol-check --strict             # Treat protocol drift as errors
```

Checks include:
- Workstream frontmatter required fields (`ws_id`, `feature_id`, `status`, `priority`, `size`, `depends_on`)
- Feature consistency across `ROADMAP.md`, `INDEX.md`, and backlog files
- Beads section presence and `sdplab-*` linkage
- Acceptance Criteria section with checkbox items

### sdp-doc-sync CLI

Documentation automation for changelog and consistency checks:

```bash
sdp-doc-sync --mode check                 # Validate docs consistency (protocol + links)
sdp-doc-sync --mode check --strict        # Treat docs drift as errors
sdp-doc-sync --mode changelog             # Update docs/CHANGELOG.md from latest commit range
sdp-doc-sync --mode changelog --since HEAD~3..HEAD
```

## Execution Kernel: agentloop

`internal/agentloop` — FSM для Delivery фазы. Phases: Discover → Plan → Build → Review → Eval.
Запускается через `sdp-harness` (subcommands: `new`, `run`, `compile-lock`, `release`, `events`). Gates принудительны — FSM не переходит без прохождения gate.
Production gateway (F106): подключается через `agentloop.ModelGateway` → LiveGateway → OpenRouter.

Статус: LiveGateway подключён и используется. F110 leaf sessions уже ходят через live dispatch claims (`internal/agentloop/livegw`). Для текущего состояния см. `cmd/sdp-harness/main.go` и свежие коммиты по F110/F111.

Reference: [docs/phases/DELIVERY.md](docs/phases/DELIVERY.md)

### llm-council skill

`skills/llm-council.md` — multi-model deliberation для ключевых решений в Discovery и при архитектурных выборах.

Вызывать когда: архитектурное решение, риск-анализ, валидация spec, ADR требует deliberation.  
Результат включает minority reports — не игнорировать несогласных моделей.

## Continuous Background Agents

Use a three-agent loop for continuous improvement:

1. **Analysis Agent** — Runs on each commit, inspects logs/evidence, creates Beads improvement tasks.
2. **Improvement Agent** — Consumes created Beads tasks and implements fixes.
3. **Documentation Agent** — Runs `sdp-doc-sync` to keep changelog and docs consistency current.

Execution model (CI in GitHub, agents local):
- **CI = Sensor layer** — runs checks and publishes findings artifacts/issues.
- **Local bridge = Transport layer** — syncs GitHub findings into local Beads queue.
- **Local agents = Actuator layer** — consume Beads tasks and implement improvements.

Recommended commit/PR checks:

```bash
sdp-protocol-check --format json
sdp-doc-sync --mode check --strict
```

**Git hooks:** Run `scripts/hooks/install-git-hooks.sh` for pre-commit (go build, ws-verdict) and pre-push (go test -short, evidence).

**Integration tests:** Use `t.Skip()` or `testing.Short()` so integration tests skip in CI. CI runs `go test -short ./...`. Never delete integration tests to fix flakiness — skip them instead.

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

1. **File issues for remaining work** — `bd create` for anything that needs follow-up
2. **Run quality gates** (if code changed) — tests, build, vet
3. **Update issue status** — после merge закрой `bd close`; если PR ещё открыт, issue остаётся claimed / `in_progress`
4. **PUSH TO REMOTE** — this is MANDATORY:
   ```bash
   git pull --rebase
   scripts/beads_transport.sh export
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Verify** — all changes committed AND pushed
6. **Hand off** — provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing — that leaves work stranded locally
- NEVER say "ready to push when you are" — YOU must push
- If push fails, resolve and retry until it succeeds

## sdp-orchestrate (oneshot outer loop)

The `@oneshot` skill uses `sdp-orchestrate` as the outer loop. Run it either way:

- **On PATH:** `go build -o $(go env GOPATH)/bin/sdp-orchestrate ./cmd/sdp-orchestrate` (or install via Makefile/CI)
- **Fallback:** `go run ./cmd/sdp-orchestrate` from project root

**"Продолжай F053"** = `go run ./cmd/sdp-orchestrate --feature F053 --next-action` (or `sdp-orchestrate --feature F053 --next-action`). Convention: "продолжай {feature}" means run the next action for that feature.

**Status:** `go run ./cmd/sdp-orchestrate --feature F053 --status` (or `sdp-orchestrate --feature F053 --status`) — outputs pending workstreams, open beads count (`bd ready`), and next action. Use when checking "Проверь beads" or "Найди оставшиеся".

> Note: `sdp status` (top-level CLI) принимает `<card-id>`, а не `--feature`. Для feature-level статуса используй `sdp-orchestrate --feature FXXX --status`.

Example: `go run ./cmd/sdp-orchestrate --feature F053 --next-action`

### Command Decision Tree

| Need | Command |
|------|---------|
| Check status (pending WS, beads, next action) | `sdp-orchestrate --feature FXXX --status` |
| Execute one leaf workstream | `/build 00-FFF-SS` |
| Execute all WS for feature | `@oneshot` or `sdp-orchestrate --feature FXXX` |
| Multi-agent quality review | `/review FXXX` |
| Create workstreams from findings | `@design phase4-remediation` |

## Key Files

| File | Purpose |
|---|---|
| `docs/architecture/REPO-BOUNDARY.md` | sdp vs sdp_lab boundary, component mapping |
| `docs/MULTI-REPO-WORKFLOW.md` | Publish workflow: how to push protocol artifacts to the public sdp repo |
| `docs/roadmap/ROADMAP.md` | Features F001-F013, phases, dependencies |
| `docs/workstreams/INDEX.md` | All workstreams with status |
| `docs/workstreams/backlog/00-XXX-YY.md` | Individual workstream: goal, scope, acceptance criteria |
| `.beads-sdp-mapping.jsonl` | WS ID ↔ beads ID mapping |
| `docs/MANIFESTO.md` | What SDP is and where it fits |
| `docs/reference/project-map.md` | Canonical project entrypoint / SOT split |
| `docs/reference/multi-agent-patterns.md` | Когда использовать Generator-Verifier / Orchestrator-Subagent / Agent Teams / Message Bus / Shared State |
| `docs/reference/harness-integration.md` | Status per harness (Claude Code, Codex, OpenCode, Cursor); OpenCode Sisyphus fix |
| `docs/reference/skill-authoring.md` | SKILL.md frontmatter policy, body template, versioning |
| `docs/reference/go-patterns.md` | **Go code style** — stack, naming, 5 good examples, 5 antipatterns, typical file template |
| `.agents/skills/README.md` | Multi-harness skills layout |

<!-- BEGIN BEADS INTEGRATION v:1 profile:full hash:d4f96305 -->
## Issue Tracking with bd (beads)

**IMPORTANT**: This project uses **bd (beads)** for ALL issue tracking. Do NOT use markdown TODOs, task lists, or other tracking methods.

### Why bd?

- Dependency-aware: Track blockers and relationships between issues
- Git-friendly: Dolt-powered version control with native sync
- Agent-optimized: JSON output, ready work detection, discovered-from links
- Prevents duplicate tracking systems and confusion

### Quick Start

**Check for ready work:**

```bash
bd ready --json
```

**Create new issues:**

```bash
bd create "Issue title" --description="Detailed context" -t bug|feature|task -p 0-4 --json
bd create "Issue title" --description="What this issue is about" -p 1 --deps discovered-from:bd-123 --json
```

**Claim and update:**

```bash
bd update <id> --claim --json
bd update bd-42 --priority 1 --json
```

**Complete work:**

```bash
bd close bd-42 --reason "Completed" --json
```

### Issue Types

- `bug` - Something broken
- `feature` - New functionality
- `task` - Work item (tests, docs, refactoring)
- `epic` - Large feature with subtasks
- `chore` - Maintenance (dependencies, tooling)

### Priorities

- `0` - Critical (security, data loss, broken builds)
- `1` - High (major features, important bugs)
- `2` - Medium (default, nice-to-have)
- `3` - Low (polish, optimization)
- `4` - Backlog (future ideas)

### Workflow for AI Agents

1. **Check ready work**: `bd ready` shows unblocked issues
2. **Claim your task atomically**: `bd update <id> --claim`
3. **Work on it**: Implement, test, document
4. **Discover new work?** Create linked issue:
   - `bd create "Found bug" --description="Details about what was found" -p 1 --deps discovered-from:<parent-id>`
5. **Complete**: `bd close <id> --reason "Done"`

### Auto-Sync

Beads transport is explicit in this repo:

- Each write auto-commits to Dolt history
- Use `scripts/beads_transport.sh fetch` before work and `scripts/beads_transport.sh export` before finishing
- The helper uses `bd dolt pull/push` only when a real Dolt remote exists; otherwise it publishes an archival `bd export` snapshot through `origin/beads-backup`

### Important Rules

- ✅ Use bd for ALL task tracking
- ✅ Always use `--json` flag for programmatic use
- ✅ Link discovered work with `discovered-from` dependencies
- ✅ Check `bd ready` before asking "what should I work on?"
- ❌ Do NOT create markdown TODO lists
- ❌ Do NOT use external issue trackers
- ❌ Do NOT duplicate tracking systems

For more details, see README.md and docs/QUICKSTART.md.

<!-- END BEADS INTEGRATION -->

@RTK.md
