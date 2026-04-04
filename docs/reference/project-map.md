# Project Map

Status: canonical project entrypoint

This is the shortest accurate way to orient inside `sdp_lab`.

## What This Repo Is

`sdp_lab` is the private lab workspace for SDP.

- it owns Go code, orchestration, adapter work, evals, roadmap, and private planning
- it uses `sdp/` as a public submodule for protocol artifacts, prompts, schemas, hooks, and public CLI work
- it uses `main` as the live default branch
- historical docs and bead IDs may still say `sdp_dev`; treat that as a legacy label for this same repo

Start with these files:

1. [AGENTS.md](../../AGENTS.md)
2. [docs/MULTI-REPO-WORKFLOW.md](../MULTI-REPO-WORKFLOW.md)
3. [docs/architecture/REPO-BOUNDARY.md](../architecture/REPO-BOUNDARY.md)

## Current Product Direction

The active direction is the platform-first reset:

- `F091` backlog reset and canonical doc sync
- `F092` kernel contract surface
- `F093` adapter gateway layer
- `F094` augmentation engine
- `F095` behavioral eval system
- `F096` legacy drift cleanup support lane

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
| What is the canonical operator loop? | [docs/plans/2026-03-15-canonical-sdp-loop-and-agent-stack.md](../plans/2026-03-15-canonical-sdp-loop-and-agent-stack.md) |
| What agents and skills are on the happy path? | [agent-catalog.md](agent-catalog.md), [skills.md](skills.md) |
| What is ready or blocked right now? | `bd ready`, `bd show`, `docs/workstreams/INDEX.md` |
| What work exists in backlog form? | `docs/workstreams/backlog/`, `.beads-sdp-mapping.jsonl` (legacy coverage gaps still under triage) |
| How does an operator execute a feature? | [docs/SDP_OPERATOR_WORKFLOW.md](../SDP_OPERATOR_WORKFLOW.md), [docs/REAL_FEATURE_TO_PR_RUNBOOK.md](../REAL_FEATURE_TO_PR_RUNBOOK.md) |
| How do docs stay consistent? | `go run ./cmd/sdp-protocol-check --format json`, `go run ./cmd/sdp-doc-sync --mode check --strict` |

## Canonical Happy Path

Default workflow:

1. shape `vision` or `feature`
2. decompose into `workstream` and linked `beads issue`
3. branch from `main`
4. open an early draft PR
5. execute ready work
6. route findings back into beads
7. run `QA/UAT`
8. merge a clean PR

Canonical references:

- [docs/plans/2026-03-15-canonical-sdp-loop-and-agent-stack.md](../plans/2026-03-15-canonical-sdp-loop-and-agent-stack.md)
- [agent-catalog.md](agent-catalog.md)
- [skills.md](skills.md)

## Read Order

If you are new to this repo, read in this order:

1. [AGENTS.md](../../AGENTS.md)
2. [docs/reference/project-map.md](project-map.md)
3. [docs/roadmap/ROADMAP.md](../roadmap/ROADMAP.md)
4. [docs/workstreams/INDEX.md](../workstreams/INDEX.md)
5. [docs/SDP_OPERATOR_WORKFLOW.md](../SDP_OPERATOR_WORKFLOW.md)

If you are touching `sdp/`, read these before changing anything:

1. [docs/MULTI-REPO-WORKFLOW.md](../MULTI-REPO-WORKFLOW.md)
2. [docs/architecture/REPO-BOUNDARY.md](../architecture/REPO-BOUNDARY.md)

## Legacy And Optional Areas

These exist, but they are not the default entrypoint for current work:

- K8s and swarm platform docs
- older control-tower working models
- historical plans that still assume `sdp_dev`, `dev`, or `bd sync`

Use them as background or archive, not as the default operator guide, unless a workstream explicitly sends you there.
