# Control Tower UX — Working Model

Status: working model
Date: 2026-03-23
Scope: human/admin UX for SDP control tower surfaces

## Purpose

SDP should not stop at process correctness.
For colleagues, it needs excellent UX.

That means the control tower must help a human answer, fast:
- what needs attention now?
- where is the bottleneck?
- what can move without me?
- what specifically needs my input?
- what is the best next action?

This doc defines the first UX working model for SDP.

---

## UX thesis

The control tower is not a raw board dump.
It is an **action-oriented operator surface**.

Good UX here means:
- low cognitive load
- clear priority
- explicit next action
- state that is readable without opening five files
- smooth transition from read → act

Bad UX here means:
- YAML-shaped human interfaces
- giant tables with no hierarchy
- buried waiting-on-human items
- dashboards that look impressive but do not help decide

---

## Primary personas

### 1. Operator / orchestrator
Needs to know:
- what should move now
- which cards are rotting
- which cards require dispatch, escalation, or follow-up

### 2. Project owner / colleague
Needs to know:
- what is happening in their project
- what needs human input
- what is blocked
- what just completed

### 3. Admin / reviewer
Needs to know:
- which cards require approval or decision
- where execution lacks trace/evidence
- whether the system is healthy enough to trust

---

## Core UX rule

Every surface should answer two layers of question:

### Layer 1 — Situation
- what is the state?
- what matters most?

### Layer 2 — Action
- what should I do next?
- what command/action should I take?

If a surface answers only Layer 1, it is informational but not operational.
That is not enough.

---

## Surface model

## 1. Operator Home

This is the top-level control tower summary.

It should show, in priority order:
1. needs attention now
2. waiting on human
3. blocked debt
4. ready to execute
5. executing now
6. recently completed
7. stale/problem cards
8. recommended next action

### UX requirements
- one-screen summary first
- counts + top offenders
- no raw schema dump
- obvious urgency hierarchy

### Good output example shape
- ATTENTION NOW: 3
- WAITING ON HUMAN: 2
- BLOCKED: 4
- READY: 5
- EXECUTING: 2
- NEXT ACTION: Dispatch card X / ask owner about card Y

---

## 2. Project Control View

A project-level summary surface.

It should answer:
- what is the project bottleneck?
- what cards are moving?
- which cards need human/admin input?
- what is ready now?
- what is stale or blocked?

### UX requirements
- grouped by state, not just listed flat
- show only enough metadata to decide
- make waiting/blocked items visually obvious
- surface recommended action per project

---

## 3. Card Detail View

The card detail view is the operator-facing representation of one control object.

It should show:
- title + status
- raw request
- normalized intent
- scope
- risk
- recommended next step
- feedback/decision state
- dispatch metadata
- linked beads / artifacts
- latest executor result
- what is missing / why blocked

### UX requirements
- readable narrative hierarchy
- not a raw YAML blob
- the “missing thing” should be explicit
- actions should be obvious from the detail view

---

## 4. Action surfaces

A surface is incomplete if it only shows state.
The UX must support action-first operation.

Important actions:
- clarify card
- mark needs_input
- mark ready
- park
- execute
- dispatch next
- export feedback
- resume after answer
- ingest result
- show next action

### Rule
The user should be able to see an item and understand the command/action to move it.

---

## Information hierarchy

Use this priority order for control-tower UX:

1. **Urgency / actionability**
2. **State bucket**
3. **Why it matters**
4. **What is missing**
5. **Recommended next action**
6. **Trace metadata**
7. **Detailed supporting context**

This means:
- “blocked because waiting on admin decision” is more important than 12 minor metadata fields
- “ready and stale for 4 days” is more important than raw timestamps alone

---

## CLI vs board vs messaging split

### CLI
Best for:
- operators
- implementation sessions
- debugging
- precise action loops

CLI should be:
- fast
- compact
- action-oriented
- readable in one screen

### Board / dashboard
Best for:
- team visibility
- project overview
- scanning multiple cards quickly

Board should be:
- grouped by state
- visually hierarchical
- not source of truth

### Messaging surface
Best for:
- nudges
- decisions
- approvals
- concise summaries

Messaging should be:
- short
- situation + ask
- linked back to richer surfaces

---

## First implementation rule

Do **not** jump straight into a big web UI.

First prove the UX model through:
- doctor summary improvements
- board show improvements
- attention surface improvements
- maybe card detail CLI rendering

Why:
- CLI is the fastest place to validate information hierarchy
- if CLI UX is bad, the future dashboard will also be bad
- good UX model should survive transport changes

---

## Excellent UX for colleagues means

A colleague can open the control tower and immediately understand:
- what is on fire
- what is waiting on them
- what can be ignored
- what is the next best action
- where to click/run/respond next

If the system needs a long explanation every time, the UX is failing.

---

## Recommended next thin slice

### UX-first CLI surface pass

Improve these first:
1. `sdp doctor control`
   - grouped issues
   - counts by severity/type
   - top stale/problem items
   - compact summary first

2. `sdp board show`
   - replace raw JSON dump with human-readable action-oriented summary by default
   - keep machine-readable output available as an option later

3. `sdp attention`
   - sharpen to a one-screen operator digest

4. optional: `sdp card show`
   - human-readable card detail surface

---

## Non-goals right now

- giant dashboard rewrite
- pixel-perfect UI system
- front-end component library
- advanced role permission UX
- full notification strategy

This working model is about making the current system feel good to use.

---

## Short formula

- control tower UX = **situation + action**
- great UX for colleagues = **clarity + priority + next move**
- first prove it in CLI, then grow outward
