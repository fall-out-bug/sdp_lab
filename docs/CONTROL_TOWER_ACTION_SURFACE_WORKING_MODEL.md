# Control Tower Action Surface — Working Model

Status: working model
Date: 2026-03-23
Scope: make the control tower operational, not just observable

## Purpose

The control tower now has meaningful visibility:
- board
- card detail
- review/delivery trace
- executive summary

The next step is to make it *operable*.

A good control tower should not only show what is happening.
It should also make it obvious how to move the system.

---

## Product rule

If an operator can see the problem but still has to mentally reconstruct the command/action flow, the UX is still incomplete.

The tower should answer both:
- what is happening?
- what should I do, exactly?

---

## Action-surface goals

### 1. Suggest concrete next moves
Not generic advice like “investigate this card”, but specific actions:
- clarify this card
- request input from admin
- dispatch this ready card
- park this card
- record delivery outcome

### 2. Keep actions near visibility
If the user is already in executive, board, or card detail view, the likely action should be visible there.

### 3. Minimize command hunting
The user should not need to remember the CLI shape from memory.

### 4. Stay thin
This does not require an interactive TUI or full command palette yet.
Textual action suggestions are enough for the first slice.

---

## First-slice action model

### Executive surface
Should show:
- next best action
- suggested command for the top item if applicable
- maybe 1-3 actionable suggestions for the highest-priority cards

### Board surface
For high-priority cards, should show:
- likely next command
- why that command is the right one

### Card detail surface
Should show a dedicated action section with:
- primary action
- fallback action if blocked
- useful command examples

---

## Suggested command categories

### Clarification / shaping
- `sdp card clarify --project ... --id ...`
- `sdp card needs-input --project ... --id ...`
- `sdp card ready --project ... --id ...`
- `sdp card park --project ... --id ...`

### Execution
- `sdp dispatch card --project ... --id ...`
- `sdp dispatch next`
- `sdp orchestrate once`

### Feedback / resume
- `sdp card feedback --project ... --id ...`
- `sdp card feedback-export --project ... --id ... --output ...`
- `sdp card resume --project ... --id ...`
- `sdp card resume-import --project ... --id ... --input ...`

### Delivery
- `sdp card deliver --project ... --id ... --state ...`

### Inspection
- `sdp card show --project ... --id ...`
- `sdp board show --project ...`
- `sdp attention`
- `sdp doctor control`

---

## Action recommendation rules

Recommendations should be derived from visible state, not guessed from vibe.

Examples:
- `needs_input` → suggest `card feedback` / `resume` / human follow-up path
- `ready` → suggest `dispatch card` or `dispatch next`
- `blocked` → suggest clarify, park, or explicit unblock flow depending on what is visible
- `review failed` → suggest resolve blocker / clarify / mark ready after rework
- `delivery failed` → suggest `card deliver --state rolled_back` or follow-up record if rollback happened

---

## UX rule

The action surface should be:
- specific
- copy-pasteable
- tied to the actual card/project
- close to the relevant state

Avoid vague pseudo-guidance like:
- “investigate further”
- “check logs”
- “do something with deployment”

If we can name the command, name the command.

---

## Non-goals for first slice

- interactive TUI controls
- keyboard-driven command palette
- clickable terminal UI
- permissions/approval workflow engine
- action history replay

This is the first text-based operational layer.

---

## Short formula

Action surface =
**visible next move + concrete command + reason why it is the right move**.
