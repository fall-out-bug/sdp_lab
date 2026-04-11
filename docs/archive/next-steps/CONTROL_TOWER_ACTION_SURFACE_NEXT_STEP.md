# Control Tower Action Surface — Next Step

Status: working next step
Date: 2026-03-23
Scope: first thin operational action layer for executive/board/card surfaces

## Goal

Make the control tower surfaces actionable with concrete command suggestions.

## Do now

1. Add thin action recommendation helpers in the render layer.
2. For executive surface:
   - show top actionable commands for the highest-priority items
3. For project board / card detail:
   - show the likely next command
   - show a fallback command where helpful
4. Keep commands copy-pasteable and tied to real card/project ids.
5. Avoid broad CLI redesign or interactive controls.

## Constraints

- thin slice only
- no TUI
- no command palette
- no permission workflow engine
- no fake actions

## Desired outcome

After this slice, a colleague should be able to open executive, board, or card detail view and know not just what is wrong, but exactly which SDP command to run next.
