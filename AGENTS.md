# Agent Instructions

> **Sync:** Sync only genuinely shared agent conventions (placement, "продолжай", command tree) to `sdp/CLAUDE.md`. Repo topology, branch policy, beads workflow, and private-lab process stay local to `sdp_lab`. See [docs/plans/2026-02-25-agents-claude-sync-rules.md](docs/plans/2026-02-25-agents-claude-sync-rules.md).

## Start Here

Use these entrypoints before diving into older plans or runbooks:

1. [docs/reference/project-map.md](docs/reference/project-map.md) — project identity, source-of-truth split, read order
2. [docs/MULTI-REPO-WORKFLOW.md](docs/MULTI-REPO-WORKFLOW.md) — parent repo vs submodule workflow
3. [docs/architecture/REPO-BOUNDARY.md](docs/architecture/REPO-BOUNDARY.md) — what belongs in `sdp_lab` vs `sdp`
4. [docs/roadmap/ROADMAP.md](docs/roadmap/ROADMAP.md) — current platform-first direction

## Cold Start For Development Agents

If you enter this repo cold, answer these before doing real work:

1. Am I changing `sdp_lab`, or am I actually being asked to adopt or publish through `sdp/`?
2. Is the user asking for platform work, or for "use SDP in my project" onboarding?
3. Which single `feature`, `workstream`, or `beads issue` owns this task?
4. Which doc is canonical for this question, instead of a historical plan?

Minimum first pass:

1. `git status --short --branch`
2. read [docs/reference/project-map.md](docs/reference/project-map.md)
3. if this is execution work, run `scripts/beads_transport.sh fetch` and `bd ready --json`
4. if the request is about greenfield or brownfield SDP adoption, stop reading private-lab process docs and jump to [sdp/docs/QUICKSTART.md](sdp/docs/QUICKSTART.md)
5. if the path starts with `sdp/`, read [docs/MULTI-REPO-WORKFLOW.md](docs/MULTI-REPO-WORKFLOW.md) before editing anything

## Project Structure

This project has **two repos** with different roles:

| | `sdp_lab` (this repo) | `sdp` (submodule at `sdp/`) |
|---|---|---|
| **Remote** | `origin → fall-out-bug/sdp_lab` | `origin → fall-out-bug/sdp` |
| **Visibility** | Private | Public |
| **Contains** | Go code, K8s manifests, roadmap, research | Protocol: prompts, JSON schemas, hooks |
| **Changes** | Daily — all features built here | Rare — only when protocol spec changes |

**Rule:** All work happens in `sdp_lab`. The `sdp/` submodule is only touched when publishing protocol artifacts (schemas, prompts, hooks).
**Source of truth:** `sdp/` must track the public GitHub repo `https://github.com/fall-out-bug/sdp.git`. A local sibling clone such as `../sdp` is a convenience checkout, not a canonical submodule URL.
**Legacy naming:** Historical workstreams, plans, and beads IDs may still use `sdp_dev` or `sdp_dev-*` as a label for this same root repo. Treat that as legacy naming, not as a third repository.

**sdp vs sdp_lab (CI/secrets):** sdp = protocol, CLI, release workflow. Secrets (e.g. GLM_API_KEY) live in sdp. sdp_lab = lab, Go binaries. When debugging CI for a PR in sdp, check sdp workflows and `workflow_call` / `secrets: inherit` — do not assume the user forgot to add secrets.

### Multi-Repo: Repo from Path

**Path `sdp/*` = repo sdp (submodule).** Different git, CI, PR.

| Path prefix | Repo | Commit | CI | PR |
|-------------|------|--------|-----|-----|
| (root), `internal/`, `cmd/`, `docs/` | sdp_lab | `git add/commit/push` in root | `.github/workflows/ci.yml` | sdp_lab |
| `sdp/` | sdp | `cd sdp && git add/commit/push` | `sdp/.github/workflows/` | sdp; then `git add sdp` in sdp_lab |

**When editing sdp/:** 1) Commit in sdp first. 2) Push sdp. 3) `git add sdp && git commit` in sdp_lab. 4) Push sdp_lab.

**Ambiguous task?** Ask: "в sdp или в sdp_lab?" — See [docs/MULTI-REPO-WORKFLOW.md](docs/MULTI-REPO-WORKFLOW.md).

## Agent Interaction Rules

**Scope:** sdp_lab only — do not sync repo-specific rules to sdp/CLAUDE.md (sdp stays generic).

**Source:** [docs/plans/2026-02-26-agent-frustration-analysis.md](docs/plans/2026-02-26-agent-frustration-analysis.md)

| Rule | Do | Don't |
|------|-----|-------|
| **No blame first** | Check code, CI, workflow before suggesting user error | "Add secret X", "You forgot to configure" |
| **Read logs** | Open Actions, read failed run, find root cause | "Please share the log" |
| **Complete the task** | "Done" = push + CI green. If CI red → keep debugging | Stop at "I made changes" without push/verify |
| **Fix, not workaround** | Find and fix root cause | Skip, non-blocking, exclude — only if user explicitly asks |
| **Commit yourself** | After changes: commit + push | "Who will make commits?" |
| **Right repo** | sdp PR → sdp workflow; sdp_lab PR → sdp_lab workflow | Fix sdp_lab when the issue is in sdp |
| **Clarify, don't guess** | If the task is ambiguous — ask: scope? fix vs analyze? which repo? | Assume intent and proceed |
| **Push back on non-constructive insults** | If the user insults without adding useful info — you may respond firmly or bluntly | Take abuse silently |
| **ПишиСокращай / ЯсноПонятно** | Notes, comments, docs: no filler, active voice, short sentences, clear structure. Each text helps solve a problem. | Watery prose, "на данном этапе", jargon, long paragraphs |
| **Radical Candor** | Care personally + challenge directly. Disagree, argue, push back when you see a better way. Not a yes-man. | Sugarcoating, ruinous empathy, subservient "as you wish" |

**Ambiguous examples:** "разобраться" (analyze or fix until done?), "займись X" (just do it or push + CI green?), "исправить" (root cause or workaround OK?), "почини CI" (в sdp или в sdp_lab?). When in doubt — one short clarifying question.

## Issue Tracking (beads)

```bash
bd ready              # Find available work
bd ready --json       # Find available work (JSON output)
bd show <id>          # View issue details
bd update <id> --status in_progress  # Claim work
bd close <id> -r "reason"  # Complete work with reason
scripts/beads_transport.sh fetch   # Restore Beads state before work
scripts/beads_transport.sh export  # Publish Beads state after work
```

`bd sync` is removed in `bd 0.61.0`. This repo uses `scripts/beads_transport.sh`, which prefers `bd dolt pull/push` when a real Dolt remote is configured and otherwise publishes an archival `bd export` snapshot through `origin/beads-backup` with plain git worktrees. In git-backup mode, `fetch` is intentionally a no-op.

### Beads ↔ Workstream Sync

- **Mapping:** `.beads-sdp-mapping.jsonl` is a helper map from `00-XXX-YY` to one primary `sdplab-*` issue when automation needs a direct lookup.
- **WS files:** The canonical live issue links belong in each workstream file's `## Beads` section. The `Feature: FXXX (...)` line names the feature, not the Beads issue.
- **Coverage rule:** do not assume `.beads-sdp-mapping.jsonl` has 1:1 line-count parity with `docs/workstreams/backlog/*.md`. Historical backlog coverage is intentionally partial, and one workstream can accumulate more than one Beads issue over time.

## Feature Delivery Flow

**Base branch:** `main`. Feature branches branch from `main`; PRs target `main`. This repo does not use a living `dev` branch.

Canonical design reference: [docs/plans/2026-03-15-canonical-sdp-loop-and-agent-stack.md](docs/plans/2026-03-15-canonical-sdp-loop-and-agent-stack.md)

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
bd update <id> --status in_progress
```

Use `docs/roadmap/ROADMAP.md` and `docs/workstreams/INDEX.md` for planning priority, not as a substitute for the live Beads queue.

Each executable unit must link back to one `feature` and one `workstream`.

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

### Step 8: Protocol Changes (only F001, F002 if needed)

If the feature publishes artifacts to the `sdp` protocol repo:

```bash
# Copy artifact into submodule
cp schema/evidence-envelope.schema.json sdp/schema/

# Commit inside the submodule (sdp: branch from its remote default branch)
SDP_BASE_BRANCH=$(git -C sdp symbolic-ref --short refs/remotes/origin/HEAD | sed 's@^origin/@@')
cd sdp
git checkout "$SDP_BASE_BRANCH" && git pull
git checkout -b schema/evidence-envelope
git add schema/
git commit -m "Add evidence envelope JSON Schema"
git push -u origin HEAD
gh pr create --base "$SDP_BASE_BRANCH" --title "Add evidence envelope JSON Schema"
cd ..

# After sdp PR is merged:
cd sdp && git checkout "$SDP_BASE_BRANCH" && git pull && cd ..
git add sdp
git commit -m "Update sdp submodule: evidence schema published"
git push
```

**When to do Step 8:** Only when the workstream file says "Publish to sdp repo" or the feature touches `sdp/` contents. Check the workstream's Scope Files section.

**Submodule recovery:** If `git submodule status` shows a missing path (`-<sha> sdp`) or a sha nobody can fetch, fix the source first:

```bash
git config -f .gitmodules submodule.sdp.url https://github.com/fall-out-bug/sdp.git
git submodule sync -- sdp
git submodule update --init --checkout sdp
```

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
| Protocol CLI (`sdp quality`, `sdp apply`, etc.) | sdp `sdp-plugin/` | Published to sdp repo |
| K8s manifests (`deploy/`) | sdp_lab only | F009 beads-bridge CronJob |
| Tests | sdp_lab only | F004 integration test |
| Roadmap, workstreams, plans | sdp_lab only | Any planning work |
| JSON Schema for evidence | sdp_lab (create) → sdp (publish) | F001 |
| Prompts, hooks | sdp_lab (develop) → sdp (publish) | Rare |
| README, Manifesto | sdp submodule directly | Rare |

**Boundary:** See [docs/architecture/REPO-BOUNDARY.md](docs/architecture/REPO-BOUNDARY.md) for component → repo → publish mapping.

**If unsure:** it goes in sdp_lab. The only things in `sdp/` are spec artifacts that external users need.

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
3. **Update issue status** — `bd close` finished work
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

**Status:** `go run ./cmd/sdp-orchestrate --feature F053 --status` (or `sdp status --feature F053`) — outputs pending workstreams, open beads count (`bd ready`), and next action. Use when checking "Проверь beads" or "Найди оставшиеся".

Example: `go run ./cmd/sdp-orchestrate --feature F053 --next-action`

### Command Decision Tree

| Need | Command |
|------|---------|
| Check status (pending WS, beads, next action) | `sdp-orchestrate --feature FXXX --status` |
| Execute one workstream | `/build 00-FFF-SS` |
| Execute all WS for feature | `@oneshot` or `sdp-orchestrate --feature FXXX` |
| Multi-agent quality review | `/review FXXX` |
| Create workstreams from findings | `@design phase4-remediation` |

## Key Files

| File | Purpose |
|---|---|
| `docs/architecture/REPO-BOUNDARY.md` | sdp vs sdp_lab boundary, component mapping |
| `docs/MULTI-REPO-WORKFLOW.md` | Multi-repo cheat sheet, commit workflow, submodule recovery |
| `docs/roadmap/ROADMAP.md` | Features F001-F013, phases, dependencies |
| `docs/workstreams/INDEX.md` | All workstreams with status |
| `docs/workstreams/backlog/00-XXX-YY.md` | Individual workstream: goal, scope, acceptance criteria |
| `docs/plans/2026-02-22-dream-swarm-design.md` | Architecture decisions for the dream swarm |
| `.beads-sdp-mapping.jsonl` | WS ID ↔ beads ID mapping |
| `docs/MANIFESTO.md` | What SDP is and where it fits |

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

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   scripts/beads_transport.sh export
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds

<!-- END BEADS INTEGRATION -->
