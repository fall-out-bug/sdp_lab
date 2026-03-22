# Control Store Skeleton

Status: initial implementation
Date: 2026-03-22

## What exists now

A first file-backed control-store skeleton has been added:

- `internal/control/`
  - file-backed `FeatureCard` write model
  - automatic intake artifact creation
  - project board snapshot derivation
  - portfolio snapshot derivation
- `cmd/sdp-control/`
  - `card-create`
  - `board-build`
  - `board-show`

## Current scope

This is intentionally small.
It proves the storage/projection model before deeper orchestration and before any dashboard implementation.

## Current behavior

### `sdp-control card-create`
Creates:
- YAML card in `.sdp/control/projects/<project>/cards/`
- Markdown intake artifact in `.sdp/control/projects/<project>/intake/`

### `sdp-control board-build`
Builds:
- project snapshot if `--project` is set
- portfolio snapshot otherwise

### `sdp-control board-show`
Currently rebuilds and prints the relevant snapshot.

## Next implementation steps

1. add card update/clarify actions
2. add ready-gate helper logic
3. attach Beads bridge operations
4. wire orchestrator actions onto the store
5. add richer status views / UI later
