# Control Tower Executive Surface — Working Model

Status: working model
Date: 2026-03-23
Scope: single operator entry surface for SDP control tower

## Purpose

SDP now has multiple useful surfaces:
- doctor
- board
- card detail
- attention
- review/delivery trace

But an operator still needs one main place to land.

The executive surface is that landing zone.
It should answer, in one screen:
- what needs attention now?
- what is blocked on humans?
- what is blocked in process/execution/review/delivery?
- what is moving?
- where is friction accumulating?
- what is the next best action?

---

## Product rule

A colleague should be able to open one surface and understand the state of the tower without stitching together doctor + board + card detail manually.

If the system requires hunting across multiple commands just to know what matters, the operator UX is still incomplete.

---

## Executive surface goals

### 1. Situation first
Show the state of the tower in priority order.

### 2. Action second
Make the next move obvious.

### 3. Friction visible
Do not hide review/delivery/rollback trouble behind “done” cards.

### 4. Compact by default
This should be one-screen useful.
Deep detail belongs to card/project surfaces.

---

## Required sections

### A. Attention now
Highest-priority items across the tower.

Examples:
- waiting on human/admin
- blocked cards
- review failures
- failed deliveries
- rollbacks

### B. Moving now
What is actively executing / under review / recently delivered.

### C. Ready to move
What is ready for the next operator/orchestrator action.

### D. Friction hotspots
Top cards/projects where friction is accumulating.

Examples:
- repeated clarification loops
- repeated blocked cycles
- repeated review failures
- rollback presence

### E. Delivery trouble
Explicitly call out:
- failed delivery
- rolled back cards
- follow-up linked cards

### F. Next best action
One clear recommended move with target.

---

## Information hierarchy

Order of importance:
1. human attention required
2. blocked/risk state
3. delivery/review trouble
4. ready work
5. active work
6. friction summary
7. recommendation

---

## Relationship to existing surfaces

### doctor control
Doctor remains the hygiene and debt checker.
Executive surface should consume its spirit, not replace it.

### board show
Board remains the control-flow/project surface.
Executive surface is the portfolio/operator home.

### card show
Card detail remains the deep drill-down surface.
Executive surface should point to cards, not duplicate full detail.

### attention
Executive surface may subsume or sharpen `attention`.
A separate `attention` command is fine if it remains a compact alias or filtered view.

---

## First thin implementation options

### Option A — improve `sdp attention`
Turn `attention` into the executive operator home.

### Option B — add explicit home command
Add something like:
- `sdp tower`
- `sdp home`
- `sdp status`

For the first slice, reuse existing command surface if that keeps the slice thin.

Recommended first move: improve `sdp attention` into the executive summary surface.

---

## What the first slice should show

At minimum:
- counts for attention / waiting / blocked / ready / executing / delivery trouble
- top 3-5 priority cards
- top friction hotspots
- delivery trouble summary
- next best action with target

---

## Non-goals

- huge dashboard redesign
- historical analytics
- trend graphs
- role-specific personalization
- full project portfolio intelligence engine

This is about a strong operator home, not a BI suite.

---

## Short formula

Executive surface =
**one-screen operator home for attention, movement, friction, delivery trouble, and next action**.
