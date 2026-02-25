# Agent Instructions

## Project Structure

This project has **two repos** with different roles:

| | `sdp_dev` (this repo) | `sdp` (submodule at `sdp/`) |
|---|---|---|
| **Remote** | `origin → sdp_private.git` | `origin → sdp.git` |
| **Visibility** | Private | Public |
| **Contains** | Go code, K8s manifests, roadmap, research | Protocol: prompts, JSON schemas, hooks |
| **Changes** | Daily — all features built here | Rare — only when protocol spec changes |

**Rule:** All work happens in `sdp_dev`. The `sdp/` submodule is only touched when publishing protocol artifacts (schemas, prompts, hooks).

## Issue Tracking (beads)

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --status in_progress  # Claim work
bd close <id>         # Complete work
bd sync               # Sync with git
```

### Beads ↔ Workstream Sync

- **Mapping:** `.beads-sdp-mapping.jsonl` maps `00-XXX-YY` (WS ID) → `sdp_dev-abc` (beads ID)
- **WS files:** Each `docs/workstreams/backlog/00-XXX-YY.md` has `Feature: FXXX (sdp_dev-abc)` with the beads ID
- **Validation:** `wc -l .beads-sdp-mapping.jsonl` must equal `ls docs/workstreams/backlog/*.md | wc -l`

## Feature Delivery Flow

### Step 1: Pick Work

```bash
bd ready                    # or: look at docs/roadmap/ROADMAP.md
bd show <id>                # read acceptance criteria
bd update <id> --status in_progress
```

### Step 2: Branch & Build

```bash
git checkout master
git pull
git checkout -b feature/FXXX-short-name   # e.g. feature/F004-sequential-reconciler
```

Write code. Run tests. Follow TDD if the workstream says so.

### Step 3: Push & PR (sdp_dev)

```bash
go test ./...
git add -A && git commit -m "F004: rewrite AgentRunReconciler to sequential phases"
git push -u origin HEAD
gh pr create --base master --title "F004: sequential reconciler"
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

# Commit inside the submodule
cd sdp
git checkout -b schema/evidence-envelope
git add schema/
git commit -m "Add evidence envelope JSON Schema"
git push -u origin HEAD
gh pr create --base main --title "Add evidence envelope JSON Schema"
cd ..

# After sdp PR is merged:
cd sdp && git checkout main && git pull && cd ..
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

## Quality Gates

Before pushing code changes:

```bash
go build ./...              # must succeed
go test ./...               # must pass
go vet ./...                # no issues
```

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

Example: `go run ./cmd/sdp-orchestrate --feature F053 --next-action`

## Key Files

| File | Purpose |
|---|---|
| `docs/architecture/REPO-BOUNDARY.md` | sdp vs sdp_dev boundary, component mapping |
| `docs/roadmap/ROADMAP.md` | Features F001-F013, phases, dependencies |
| `docs/workstreams/INDEX.md` | All workstreams with status |
| `docs/workstreams/backlog/00-XXX-YY.md` | Individual workstream: goal, scope, acceptance criteria |
| `docs/plans/2026-02-22-dream-swarm-design.md` | Architecture decisions for the dream swarm |
| `.beads-sdp-mapping.jsonl` | WS ID ↔ beads ID mapping |
| `docs/MANIFESTO.md` | What SDP is and where it fits |
