# SDP CLI Stage Navigation Spec

> **Status:** Proposed
> **Date:** 2026-04-05
> **Related issue:** `sdplab-8jb`
> **Scope repo:** `sdp/`

## Goal

Align runtime CLI help and navigation surfaces in `sdp/sdp-plugin` with the canonical happy path.

This spec exists because help strings and navigation behavior are code, not just docs.
They should be changed as an implementation slice in `sdp`, not inside the `sdp_lab` docs sync.

## Problem

Current runtime help is materially better than stale docs, but it still underspecifies the stage model:

- root help lists commands, but does not clearly frame Local Mode vs Operator Mode
- command help does not consistently explain stage purpose, artifact boundaries, or adjacent next step
- `status` and `next` are useful, but they are not presented as the canonical navigation center
- operators still need too much repo knowledge to infer which command belongs to which stage

## Decision

`sdp` CLI help must become a stage-navigation surface, not a flat command catalog.

### Required help contract

Each major command help must answer, in this order:

1. what stage this command belongs to
2. when to use it
3. what state or artifact it reads or writes
4. what usually comes before it
5. what usually comes after it

### Root help contract

Root help must:

- state the Local Mode vs Operator Mode split explicitly
- explain that CLI and skills are two control surfaces over one stage model
- group command families by stage or operator job, not by historical implementation accident
- surface `status` and `next` as the primary navigation aids

### `status` and `next` contract

`status` and `next` must be treated as first-class onboarding/runtime guidance surfaces.

They should be able to orient a user without requiring them to open stale reference docs.

## In Scope

- root `sdp --help`
- major command help for:
  - `init`
  - `doctor`
  - `status`
  - `next`
  - `plan`
  - `apply`
  - `build`
  - `verify`
  - `deploy`
- grouping and examples in `sdp/docs/CLI_REFERENCE.md`

## Out of Scope

- public README and quickstart rewrite
- board UI implementation
- Beads state model changes
- deploy semantics beyond copy/help and stage framing

## Acceptance Criteria

1. Root help explains the mode split and points users to `status` and `next` for navigation.
2. Major command help follows one consistent stage-oriented template.
3. `sdp/docs/CLI_REFERENCE.md` matches runtime help and no longer describes a competing primary workflow.
4. A user can infer when to use `plan`, `apply`, `build`, `verify`, and `deploy` without reading historical plans.
5. Help-copy examples do not contradict the canonical board-to-deploy path.

## Implementation Notes

Likely touchpoints:

- `sdp/sdp-plugin/cmd/sdp/*.go`
- `sdp/docs/CLI_REFERENCE.md`

This work should be implemented in `sdp/` and then synced back into `sdp_lab` through the submodule update flow.
