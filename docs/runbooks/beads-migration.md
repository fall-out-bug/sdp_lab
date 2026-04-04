# Beads Migration Runbook

Status: historical context, not the current default workflow

The old `bd sync` and `sdp beads migrate` flow is no longer the live operating model in this repo.

## Current Rule

- create or locate the Beads issue directly with `bd create`, `bd show`, or `bd ready`
- record canonical issue links in the workstream file under `## Beads`
- use `.beads-sdp-mapping.jsonl` only as helper lookup data when automation needs one primary WS → Beads mapping
- use `./scripts/beads_transport.sh fetch` and `./scripts/beads_transport.sh export` for transport

## What Changed

- `bd 0.61.0` removed `bd sync`
- this repo does not treat `.beads-sdp-mapping.jsonl` as a full 1:1 mirror of every backlog file
- one workstream can accumulate more than one Beads issue over time, so the workstream file is the canonical live trace

## If You Need To Reconcile Workstream And Beads State

1. Inspect the workstream file and the `## Beads` section.
2. Check the live issue state with `bd show <id>` or `bd ready`.
3. Update the workstream file if ownership changed or the current Beads issue is missing.
4. Update `.beads-sdp-mapping.jsonl` only if a primary lookup needs to move.

## Related Docs

- [../beads-transport.md](beads-transport.md)
- [../../AGENTS.md](../../AGENTS.md)
- [../MULTI-REPO-WORKFLOW.md](../MULTI-REPO-WORKFLOW.md)
