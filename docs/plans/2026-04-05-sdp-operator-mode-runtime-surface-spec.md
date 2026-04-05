# SDP Operator Mode Runtime Surface Spec

> **Status:** Proposed
> **Date:** 2026-04-05
> **Related issue:** `sdplab-20w`
> **Scope repo:** `sdp/`

## Goal

Define the runtime changes needed for SDP to make the board-backed Operator Mode promise credible.

This is not a docs cleanup task.
It is a runtime surface spec for the CLI and supporting state exposure in `sdp/`.

## Problem

The canonical product story is:

1. task enters board-backed queue
2. SDP shapes it
3. agents execute it
4. findings loop back into the same queue
5. the operator sees status and next action
6. the item reaches delivery with proof

Today the runtime surfaces do not fully expose that story:

- board semantics are mostly implicit
- `status` and `next` do not consistently expose queue truth, next actor, and delivery gating
- PR, findings, `QA/UAT`, and delivery states are not presented as one visible chain
- the operator still reconstructs too much manually from Beads, docs, and artifacts

## Decision

Operator Mode runtime surfaces must expose the same chain that the design docs now describe.

The CLI does not need to become a board UI.
It does need to expose enough state to make the board-backed system legible.

## Required Runtime Questions

In Operator Mode, SDP should be able to answer these questions from runtime surfaces:

1. what state is this item in right now?
2. why is it in that state?
3. who or what is expected next?
4. what is blocking progress?
5. what is the linked execution graph item?
6. what is the linked PR state?
7. what findings are still open?
8. has `QA/UAT` passed?
9. what human approval, if any, is still required?

## Scope

### In scope

- `status` output contract
- `next` recommendation contract
- visibility of queue-backed state
- visibility of findings loop state
- visibility of PR / `QA/UAT` / delivery gating
- explicit Operator Mode framing in runtime text and JSON surfaces where relevant

### Out of scope

- full control-tower UI
- replacing Beads
- changing Beads as source of truth
- autonomous production deployment

## Truth Model Constraints

- `Beads` remains the operational source of truth
- board or status views are projections
- PR remains the integration surface
- `evidence`, `trace`, `drift`, and `QA/UAT` remain proof layers

The runtime must reflect that instead of inventing a second queue model.

## Candidate Surfaces

- `sdp status`
- `sdp next`
- any future CLI board or queue summary commands
- exported JSON status contracts consumed by agents or other tooling

## Acceptance Criteria

1. Operator-facing runtime surfaces expose queue-backed current state and next actor.
2. Findings loop state is visible without manual artifact archaeology.
3. PR, `QA/UAT`, and delivery gates are represented as part of the same runtime story.
4. The runtime does not imply that board views or status views are the source of truth.
5. A colleague can inspect one item and understand where it is between intake and delivery without opening historical plans.

## Implementation Notes

Likely code areas:

- `sdp/sdp-plugin/cmd/sdp/status*`
- `sdp/sdp-plugin/cmd/sdp/next*`
- any shared state/view builders behind those commands

This spec should be decomposed into one or more `sdp/` implementation tasks after the owning issue is created.
