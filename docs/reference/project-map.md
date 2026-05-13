# Project Map

Status: canonical project entrypoint

This is the shortest accurate way to orient inside `sdp_lab`.

## What This Repo Is

`sdp_lab` is the primary public workspace for SDP.

- it owns Go code, orchestration, adapter work, evals, roadmap, and planning
- protocol artifacts (prompts, schemas, hooks, public CLI work) live at native tracked paths (`prompts/`, `schema/`, `templates/`, `.claude/hooks/`); distilled artifacts are published to the `sdp` repo via `scripts/sdp-publish.sh`. `sdp/` is an optional local checkout of that distribution repo, not a tracked component.
- it uses `main` as the live default branch
- Go module path is `github.com/fall-out-bug/sdp_lab`; historical docs and bead IDs may still say `sdp_dev` — treat as legacy naming

## Who Should Start Here

Use this doc if you need to understand the workspace itself:

- you want to know what this repo does and which components live here
- you are contributing to SDP platform code, evals, adapters, or planning
- you are a development agent entering cold and need the canonical read order

Do not use this doc as your main onboarding path if your real goal is:

- install SDP into another repo
- give SDP your IDE and keys
- start greenfield delivery or brownfield adoption

For that path, go straight to [../QUICKSTART.md](../QUICKSTART.md). Today that quickstart covers `Claude Code`, `OpenCode`, `Codex`, `Cursor`, and `Pi`.

## Main Components

| Area | What it owns |
|---|---|
| `cmd/`, `internal/` | Go binaries, kernel, adapters, orchestration, evals |
| `deploy/` | K8s runtime and observability manifests |
| `docs/roadmap/`, `docs/workstreams/`, `docs/plans/` | planning, execution queue, and design history |
| `docs/reference/` | stable reference docs for the current canonical loop |
| `sdp/` | optional local checkout of the distilled `sdp` repo (not tracked in `sdp_lab`); publish target for `scripts/sdp-publish.sh` |

Start with these files:

1. [AGENTS.md](../../AGENTS.md)
2. [docs/MULTI-REPO-WORKFLOW.md](../MULTI-REPO-WORKFLOW.md)
3. [docs/architecture/REPO-BOUNDARY.md](../architecture/REPO-BOUNDARY.md)

## Current Product Direction

SDP is organized into seven product layers. See [`product-surface.md`](product-surface.md) for the full inventory and [`../../docs/strategy/2026-04-27-sdp-product-layering-4d.md`](../../docs/strategy/2026-04-27-sdp-product-layering-4d.md) for the canonical layer taxonomy.

**What ships today (SDP Toolkit + Toolbox):**

- first-run `sdp` CLI surface: `scout`, `metrics`, `index build`, `spec`, `bootstrap --dry-run`
- install/support surface: `init`, `manifest`, `generate-adapters`, `doctor`
- after `index build`: `index query`, `index find`, `index deps`, and `index stats` reuse the local index cache
- multi-harness adapter install for Claude Code, OpenCode, Codex, Cursor, and Pi
- Operator Mode as the default Toolkit happy path (stateful orchestration). `architect` is useful here as second-run/operator analysis, not as first-run onboarding.

**Product direction (not yet shipped):**

- ChangePassport (`sdp-pr-gate`) — separate merge-readiness product surface
- Enterprise Delivery Governance — enterprise governed delivery control plane

**Research / lab-only (not in formula):**

- agentloop FSM runtime, model gateway, MicroFirst, telemetry daemon
- K8s/swarm/control tower, eval framework, benchmark tooling

Historical platform reset (F091-F096) and Toolkit lane (F120-F126) details remain in the archive plans below. The current active track is F150 (product layering and release readiness).

Canonical roadmap:

- [docs/roadmap/ROADMAP.md](../roadmap/ROADMAP.md)
- [docs/plans/2026-03-31-platform-backlog-reset.md](../plans/2026-03-31-platform-backlog-reset.md)

## Source Of Truth Split

Use one source per question.

| Question | Source |
|---|---|
| What repo owns this file? | [docs/MULTI-REPO-WORKFLOW.md](../MULTI-REPO-WORKFLOW.md) |
| What belongs in `sdp_lab` vs `sdp`? | [docs/architecture/REPO-BOUNDARY.md](../architecture/REPO-BOUNDARY.md) |
| How do I adopt SDP in another repo? | [../QUICKSTART.md](../QUICKSTART.md) |
| What is the canonical happy path from intake to delivery? | [canonical-happy-path.md](canonical-happy-path.md) |
| What is the canonical operator loop? | [canonical-happy-path.md](canonical-happy-path.md), [docs/SDP_OPERATOR_WORKFLOW.md](../SDP_OPERATOR_WORKFLOW.md) |
| What agents and skills are on the happy path? | [agent-catalog.md](agent-catalog.md), [skills.md](skills.md) |
| How do root, module, skill, command, and harness instructions compose? | [agent-instruction-cascade.md](agent-instruction-cascade.md) |
| What is ready or blocked right now? | `bd ready` and `bd show` are authoritative; `docs/workstreams/INDEX.md` is only the planning summary |
| What work exists in backlog form? | `docs/workstreams/backlog/` is canonical backlog; `.beads-sdp-mapping.jsonl` is helper data, not full historical coverage |
| How does an operator execute a feature? | [docs/SDP_OPERATOR_WORKFLOW.md](../SDP_OPERATOR_WORKFLOW.md), [docs/REAL_FEATURE_TO_PR_RUNBOOK.md](../REAL_FEATURE_TO_PR_RUNBOOK.md) |
| How do docs stay consistent? | `go run ./cmd/sdp-protocol-check --format json`, `go run ./cmd/sdp-doc-sync --mode check --strict` |
| What is ready vs tooling vs experimental? | [product-surface.md](product-surface.md), [maturity-matrix.md](maturity-matrix.md) |

## Canonical Happy Path

The canonical system story is stage-first:

1. install and bootstrap
2. intake into a board-backed queue
3. clarification and shaping
4. execution setup with early draft PR
5. execution
6. findings loop
7. `QA/UAT`
8. delivery

That story has two explicit modes:

- `Local Mode` — adoption ramp inside one repo, where `Beads` may still be optional
- `Operator Mode` — full board-to-delivery path, where `Beads` is required and board views are projections over Beads-backed truth

Default operator workflow:

1. shape `vision` or `feature`
2. decompose into `workstream` and linked `beads issue`
3. branch from `main`
4. open an early draft PR
5. execute ready work
6. route findings back into beads
7. run `QA/UAT`
8. merge a clean PR

Canonical references:

- [canonical-happy-path.md](canonical-happy-path.md)
- [docs/SDP_OPERATOR_WORKFLOW.md](../SDP_OPERATOR_WORKFLOW.md)
- [agent-catalog.md](agent-catalog.md)
- [skills.md](skills.md)
- [docs/plans/2026-04-05-canonical-sdp-happy-path-consistency.md](../plans/2026-04-05-canonical-sdp-happy-path-consistency.md)

## Read Order

If you are new to this repo, read in this order:

1. [AGENTS.md](../../AGENTS.md)
2. [docs/reference/project-map.md](project-map.md)
3. [docs/reference/canonical-happy-path.md](canonical-happy-path.md)
4. [docs/reference/agent-instruction-cascade.md](agent-instruction-cascade.md)
5. [docs/roadmap/ROADMAP.md](../roadmap/ROADMAP.md)
6. [docs/workstreams/INDEX.md](../workstreams/INDEX.md)
7. [docs/SDP_OPERATOR_WORKFLOW.md](../SDP_OPERATOR_WORKFLOW.md)

If you are new to SDP but not to this repo, use this shorter decision:

1. "I want to use SDP in my own repo" -> [../QUICKSTART.md](../QUICKSTART.md)
2. "I want to work on SDP platform internals" -> keep reading this file
3. "I am a dev agent entering cold" -> [../../AGENTS.md](../../AGENTS.md), then this file

If you are touching protocol artifacts in `sdp/`, read these before changing anything:

1. [docs/MULTI-REPO-WORKFLOW.md](../MULTI-REPO-WORKFLOW.md) — publish workflow
2. [docs/architecture/REPO-BOUNDARY.md](../architecture/REPO-BOUNDARY.md) — what to publish

## Legacy And Optional Areas

These exist, but they are not the default entrypoint for current work:

- K8s and swarm platform docs
- older control-tower working models
- historical plans that still assume `sdp_dev`, `dev`, or `bd sync`

Use them as background or archive, not as the default operator guide, unless a workstream explicitly sends you there.
