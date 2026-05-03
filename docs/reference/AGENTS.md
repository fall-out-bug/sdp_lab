# docs/reference — Agent Contract

## Scope

This subtree owns stable reference documentation for current SDP behavior.

## Contract

Reference docs answer durable questions: source-of-truth routing, public product
surface, canonical workflow, agent/skill ownership, commands, gates, and glossary.

## Dependencies

Reference docs may summarize dated plans, but current behavior must link to the
active source of truth. Historical rationale belongs in `docs/plans/` or
`docs/archive/`.

## Runtime Assumptions

Agents use this subtree after root `AGENTS.md` and `docs/reference/project-map.md`
to orient before implementation or review.

## Local Rules

- Prefer one canonical reference per question.
- Do not duplicate long sections from root `AGENTS.md`; link instead.
- If a reference conflicts with code, command help, or manifest output, call out
  the conflict and update the canonical owner.
- Keep new reference docs discoverable from `README.md` or `project-map.md`.
