# Agent Handoff

Updated: 2026-03-11

## Scope First

- Continue day-to-day development in `sdp_lab` only.
- In this checkout, `sdp_lab` is the root repo at `/Users/fall_out_bug/projects/vibe_coding/sdp_dev`.
- Treat `sdp/` as a separate public protocol repo. Do not move feature work there by default.
- Touch `sdp/` only for protocol artifacts, public docs, prompts, hooks, or explicitly scoped protocol fixes.

## Current Branches

- `sdp_lab`: `codex/reality-specs` at `d7e78f7`
- `sdp`: `codex/beads-059-alignment` at `dab76bf`
- Both branches are pushed and clean.

## What Was Just Landed

- `beads` upgraded to `0.59.0` in `sdp_lab` runtime/build paths.
- `sdp_lab` active workflow no longer depends on `bd sync`; use:
  - `./scripts/beads_import_only.sh`
  - `./scripts/beads_export.sh`
- `sdp_lab` branch policy is explicit now:
  - private lab repo works from `dev`
  - public `sdp` repo works from `main`
- K8s defaults in `sdp_lab` now point at `sdp_lab.git` and default repo branch `dev`.
- Public `sdp` repo has a separate pushed branch with the beads 0.59 compatibility layer.

## Continue In sdp_lab

Default next work should stay in `sdp_lab`.

Priority follow-ups:
- `sdplab-vc0` — Sweep remaining beads 0.59 workflow drift
- `sdplab-a4w` — Audit `sdp-plugin` for Dolt-native beads semantics

Interpretation:
- `sdplab-vc0` is mostly docs / historical drift / cleanup.
- `sdplab-a4w` is the deeper technical follow-up around SQLite-era assumptions in the public plugin.
- If the task is not clearly about protocol/public surface, keep it in `sdp_lab`.

## When To Enter sdp/

Enter `sdp/` only if one of these is true:
- you are opening or finishing the PR for `codex/beads-059-alignment`
- you are publishing protocol-facing artifacts from `sdp_lab`
- the requested change is explicitly about public prompts / schemas / hooks / protocol docs

If not, stay in root.

## Startup Commands

```bash
cd /Users/fall_out_bug/projects/vibe_coding/sdp_dev
bd prime
./scripts/beads_import_only.sh
bd ready
bd show sdplab-vc0
bd show sdplab-a4w
git status --short --branch
```

## Session Rules

- Use Beads as source of truth for follow-up work.
- Keep root repo and submodule commits separate.
- If you edit `sdp/`:
  1. Commit and push inside `sdp/`
  2. Return to root
  3. Commit the updated submodule pointer in `sdp_lab`
- On session completion in `sdp_lab`:
  1. `git pull --rebase`
  2. `./scripts/beads_import_only.sh` if `.beads/issues.jsonl` changed after pull
  3. `./scripts/beads_export.sh` if Beads changed in this session
  4. run required quality gates
  5. `git push`

## Open PR Candidates

- `sdp`: open PR from `codex/beads-059-alignment` to `main`
- `sdp_lab`: open PR from `codex/reality-specs`

## Residual Warning

`bd doctor` in `sdp_lab` is green except for 2 legacy SQLite artifacts:
- `.beads/beads.db`
- `sdp/.beads/beads.db`

These are non-blocking local leftovers, not active storage.
