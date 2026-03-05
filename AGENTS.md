# Agent Instructions

> **Sync:** When updating shared conventions (placement, "продолжай", command tree), also update `sdp/CLAUDE.md`. See [docs/plans/2026-02-25-agents-claude-sync-rules.md](docs/plans/2026-02-25-agents-claude-sync-rules.md).

## Project Structure

This project has **two repos** with different roles:

| | `sdp_dev` (this repo) | `sdp` (submodule at `sdp/`) |
|---|---|---|
| **Remote** | `origin → sdp_private.git` | `origin → sdp.git` |
| **Visibility** | Private | Public |
| **Contains** | Go code, K8s manifests, roadmap, research | Protocol: prompts, JSON schemas, hooks |
| **Changes** | Daily — all features built here | Rare — only when protocol spec changes |

**Rule:** All work happens in `sdp_dev`. The `sdp/` submodule is only touched when publishing protocol artifacts (schemas, prompts, hooks).

**sdp vs sdp_dev (CI/secrets):** sdp = protocol, CLI, release workflow. Secrets (e.g. GLM_API_KEY) live in sdp. sdp_dev = lab, Go binaries. When debugging CI for a PR in sdp, check sdp workflows and `workflow_call` / `secrets: inherit` — do not assume the user forgot to add secrets.

### Multi-Repo: Repo from Path

**Path `sdp/*` = repo sdp (submodule).** Different git, CI, PR.

| Path prefix | Repo | Commit | CI | PR |
|-------------|------|--------|-----|-----|
| (root), `internal/`, `cmd/`, `docs/` | sdp_dev | `git add/commit/push` in root | `.github/workflows/ci.yml` | sdp_dev |
| `sdp/` | sdp | `cd sdp && git add/commit/push` | `sdp/.github/workflows/` | sdp; then `git add sdp` in sdp_dev |

**When editing sdp/:** 1) Commit in sdp first. 2) Push sdp. 3) `git add sdp && git commit` in sdp_dev. 4) Push sdp_dev.

**Ambiguous task?** Ask: "в sdp или в sdp_dev?" — See [docs/MULTI-REPO-WORKFLOW.md](docs/MULTI-REPO-WORKFLOW.md).

## Agent Interaction Rules

**Scope:** sdp_dev only — do not sync to sdp/CLAUDE.md (sdp stays generic).

**Source:** [docs/plans/2026-02-26-agent-frustration-analysis.md](docs/plans/2026-02-26-agent-frustration-analysis.md)

| Rule | Do | Don't |
|------|-----|-------|
| **No blame first** | Check code, CI, workflow before suggesting user error | "Add secret X", "You forgot to configure" |
| **Read logs** | Open Actions, read failed run, find root cause | "Please share the log" |
| **Complete the task** | "Done" = push + CI green. If CI red → keep debugging | Stop at "I made changes" without push/verify |
| **Fix, not workaround** | Find and fix root cause | Skip, non-blocking, exclude — only if user explicitly asks |
| **Commit yourself** | After changes: commit + push | "Who will make commits?" |
| **Right repo** | sdp PR → sdp workflow; sdp_dev PR → sdp_dev workflow | Fix sdp_dev when the issue is in sdp |
| **Clarify, don't guess** | If the task is ambiguous — ask: scope? fix vs analyze? which repo? | Assume intent and proceed |
| **Push back on non-constructive insults** | If the user insults without adding useful info — you may respond firmly or bluntly | Take abuse silently |
| **ПишиСокращай / ЯсноПонятно** | Notes, comments, docs: no filler, active voice, short sentences, clear structure. Each text helps solve a problem. | Watery prose, "на данном этапе", jargon, long paragraphs |
| **Radical Candor** | Care personally + challenge directly. Disagree, argue, push back when you see a better way. Not a yes-man. | Sugarcoating, ruinous empathy, subservient "as you wish" |

**Ambiguous examples:** "разобраться" (analyze or fix until done?), "займись X" (just do it or push + CI green?), "исправить" (root cause or workaround OK?), "почини CI" (в sdp или в sdp_dev?). When in doubt — one short clarifying question.

## Issue Tracking (beads)

```bash
bd ready              # Find available work
bd ready --json       # Find available work (JSON output)
bd show <id>          # View issue details
bd update <id> --status in_progress  # Claim work
bd close <id> -r "reason"  # Complete work with reason
bd sync               # Sync with git
```

### Beads ↔ Workstream Sync

- **Mapping:** `.beads-sdp-mapping.jsonl` maps `00-XXX-YY` (WS ID) → `sdp_dev-abc` (beads ID)
- **WS files:** Each `docs/workstreams/backlog/00-XXX-YY.md` has `Feature: FXXX (sdp_dev-abc)` with the beads ID
- **Validation:** `wc -l .beads-sdp-mapping.jsonl` must equal `ls docs/workstreams/backlog/*.md | wc -l`

## Feature Delivery Flow

**Base branch:** `dev`. Feature branches branch from `dev`; PRs target `dev`. `main`/`master` for releases only.

### Step 1: Pick Work

```bash
bd ready                    # or: look at docs/roadmap/ROADMAP.md
bd show <id>                # read acceptance criteria
bd update <id> --status in_progress
```

### Step 2: Branch & Build

```bash
git checkout dev
git pull
git checkout -b feature/FXXX-short-name   # e.g. feature/F004-sequential-reconciler
```

Write code. Run tests. Follow TDD if the workstream says so.

### Step 3: Push & PR (sdp_dev)

```bash
go test ./...
git add -A && git commit -m "F004: rewrite AgentRunReconciler to sequential phases"
git push -u origin HEAD
gh pr create --base dev --title "F004: sequential reconciler"
```

### Step 4: Merge & Close

```bash
gh pr merge                 # after review
bd close <id>
```

**For 90% of features (F003-F010, F013) — done. Stop here.**

### Step 5: Protocol Changes (only F001, F002 if needed)

If the feature publishes artifacts to the `sdp` protocol repo:

```bash
# Copy artifact into submodule
cp schema/evidence-envelope.schema.json sdp/schema/

# Commit inside the submodule (sdp: branch from dev)
cd sdp
git checkout dev && git pull
git checkout -b schema/evidence-envelope
git add schema/
git commit -m "Add evidence envelope JSON Schema"
git push -u origin HEAD
gh pr create --base dev --title "Add evidence envelope JSON Schema"
cd ..

# After sdp PR is merged:
cd sdp && git checkout dev && git pull && cd ..
git add sdp
git commit -m "Update sdp submodule: evidence schema published"
git push
```

**When to do Step 5:** Only when the workstream file says "Publish to sdp repo" or the feature touches `sdp/` contents. Check the workstream's Scope Files section.

## Branch Naming

```
feature/FXXX-short-name     # feature work (e.g. feature/F004-sequential-reconciler)
fix/FXXX-description        # bug fixes within a feature
docs/topic                  # documentation-only changes
```

## What Goes Where

| Change Type | Where | Example |
|---|---|---|
| Go code (`internal/`, `cmd/`) | sdp_dev only | F004 reconciler rewrite |
| Lab binaries (orchestrate, ci-loop, evidence, guard, eval) | sdp_dev `cmd/` | `make build-sdp-orchestrate` |
| Protocol CLI (`sdp quality`, `sdp apply`, etc.) | sdp `sdp-plugin/` | Published to sdp repo |
| K8s manifests (`deploy/`) | sdp_dev only | F009 beads-bridge CronJob |
| Tests | sdp_dev only | F004 integration test |
| Roadmap, workstreams, plans | sdp_dev only | Any planning work |
| JSON Schema for evidence | sdp_dev (create) → sdp (publish) | F001 |
| Prompts, hooks | sdp_dev (develop) → sdp (publish) | Rare |
| README, Manifesto | sdp submodule directly | Rare |

**Boundary:** See [docs/architecture/REPO-BOUNDARY.md](docs/architecture/REPO-BOUNDARY.md) for component → repo → publish mapping.

**If unsure:** it goes in sdp_dev. The only things in `sdp/` are spec artifacts that external users need.

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
   bd sync
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
| `docs/architecture/REPO-BOUNDARY.md` | sdp vs sdp_dev boundary, component mapping |
| `docs/MULTI-REPO-WORKFLOW.md` | Multi-repo cheat sheet, commit workflow |
| `docs/roadmap/ROADMAP.md` | Features F001-F013, phases, dependencies |
| `docs/workstreams/INDEX.md` | All workstreams with status |
| `docs/workstreams/backlog/00-XXX-YY.md` | Individual workstream: goal, scope, acceptance criteria |
| `docs/plans/2026-02-22-dream-swarm-design.md` | Architecture decisions for the dream swarm |
| `.beads-sdp-mapping.jsonl` | WS ID ↔ beads ID mapping |
| `docs/MANIFESTO.md` | What SDP is and where it fits |
