# Project Map

Status: canonical project entrypoint

This is the shortest accurate way to orient inside `sdp_lab`.

## What This Repo Is

`sdp_lab` is the private lab workspace for SDP.

- it owns Go code, orchestration, adapter work, evals, roadmap, and private planning
- it uses `sdp/` as a public submodule for protocol artifacts, prompts, schemas, hooks, and public CLI work
- it uses `main` as the live default branch
- historical docs and bead IDs may still say `sdp_dev`; treat that as a legacy label for this same repo

## Who Should Start Here

Use this doc if you need to understand the private workspace itself:

- you want to know what this repo does and which components live here
- you are contributing to SDP platform code, evals, adapters, or planning
- you are a development agent entering cold and need the canonical read order

Do not use this doc as your main onboarding path if your real goal is:

- install SDP into another repo
- give SDP your IDE and keys
- start greenfield delivery or brownfield adoption

For that path, go straight to [../../sdp/docs/QUICKSTART.md](../../sdp/docs/QUICKSTART.md). Today that quickstart covers `Claude Code`, `Cursor`, `OpenCode`, and `Codex`. Use [../../sdp/.codex/INSTALL.md](../../sdp/.codex/INSTALL.md) for Codex-specific notes after install.

## Main Components

| Area | What it owns |
|---|---|
| `cmd/`, `internal/` | Go binaries, kernel, adapters, orchestration, evals |
| `deploy/` | K8s runtime and observability manifests |
| `docs/roadmap/`, `docs/workstreams/`, `docs/plans/` | planning, execution queue, and design history |
| `docs/reference/` | stable reference docs for the current canonical loop |
| `sdp/` | public submodule for prompts, hooks, schemas, and OSS CLI work |

Start with these files:

1. [AGENTS.md](../../AGENTS.md)
2. [docs/MULTI-REPO-WORKFLOW.md](../MULTI-REPO-WORKFLOW.md)
3. [docs/architecture/REPO-BOUNDARY.md](../architecture/REPO-BOUNDARY.md)

## Current Product Direction

The active direction has two lanes:

**Platform reset (core):**

- `F091` backlog reset and canonical doc sync
- `F092` kernel contract surface
- `F093` adapter gateway layer
- `F094` augmentation engine
- `F095` behavioral eval system
- `F096` legacy drift cleanup support lane

**Toolkit lane (unknown repo → AI-native adoption):**

- `F120` Toolkit Scout — instant repo card
- `F121` Toolkit Metrics — git-derived process health (Done)
- `F122` Toolkit Index — persistent codebase memory
- `F123` Toolkit Spec Recovery — recover implicit contracts
- `F124` Toolkit Bootstrap — brownfield-safe agent setup
- `F125` Toolkit UX — intent-routed skills
- `F126` Toolkit MCP — universal agent interface (Done)

Trust, evidence, and governance still matter, but they are the secondary lane, not the whole story.

Canonical roadmap:

- [docs/roadmap/ROADMAP.md](../roadmap/ROADMAP.md)
- [docs/plans/2026-03-31-platform-backlog-reset.md](../plans/2026-03-31-platform-backlog-reset.md)

## Source Of Truth Split

Use one source per question.

| Question | Source |
|---|---|
| What repo owns this file? | [docs/MULTI-REPO-WORKFLOW.md](../MULTI-REPO-WORKFLOW.md) |
| What belongs in `sdp_lab` vs `sdp`? | [docs/architecture/REPO-BOUNDARY.md](../architecture/REPO-BOUNDARY.md) |
| How do I adopt SDP in another repo? | [../../sdp/docs/QUICKSTART.md](../../sdp/docs/QUICKSTART.md) |
| What is the canonical happy path from intake to delivery? | [canonical-happy-path.md](canonical-happy-path.md) |
| What is the canonical operator loop? | [canonical-happy-path.md](canonical-happy-path.md), [docs/SDP_OPERATOR_WORKFLOW.md](../SDP_OPERATOR_WORKFLOW.md) |
| What agents and skills are on the happy path? | [agent-catalog.md](agent-catalog.md), [skills.md](skills.md) |
| What is ready or blocked right now? | `bd ready` and `bd show` are authoritative; `docs/workstreams/INDEX.md` is only the planning summary |
| What work exists in backlog form? | `docs/workstreams/backlog/` is canonical backlog; `.beads-sdp-mapping.jsonl` is helper data, not full historical coverage |
| How does an operator execute a feature? | [docs/SDP_OPERATOR_WORKFLOW.md](../SDP_OPERATOR_WORKFLOW.md), [docs/REAL_FEATURE_TO_PR_RUNBOOK.md](../REAL_FEATURE_TO_PR_RUNBOOK.md) |
| How do docs stay consistent? | `go run ./cmd/sdp-protocol-check --format json`, `go run ./cmd/sdp-doc-sync --mode check --strict` |

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
4. [docs/roadmap/ROADMAP.md](../roadmap/ROADMAP.md)
5. [docs/workstreams/INDEX.md](../workstreams/INDEX.md)
6. [docs/SDP_OPERATOR_WORKFLOW.md](../SDP_OPERATOR_WORKFLOW.md)

If you are new to SDP but not to this repo, use this shorter decision:

1. "I want to use SDP in my own repo" -> [../../sdp/docs/QUICKSTART.md](../../sdp/docs/QUICKSTART.md)
2. "I want to work on SDP platform internals" -> keep reading this file
3. "I am a dev agent entering cold" -> [../../AGENTS.md](../../AGENTS.md), then this file

If you are touching `sdp/`, read these before changing anything:

1. [docs/MULTI-REPO-WORKFLOW.md](../MULTI-REPO-WORKFLOW.md)
2. [docs/architecture/REPO-BOUNDARY.md](../architecture/REPO-BOUNDARY.md)

## Legacy And Optional Areas

These exist, but they are not the default entrypoint for current work:

- K8s and swarm platform docs
- older control-tower working models
- historical plans that still assume `sdp_dev`, `dev`, or `bd sync`

Use them as background or archive, not as the default operator guide, unless a workstream explicitly sends you there.
