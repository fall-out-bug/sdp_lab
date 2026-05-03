# docs — Agent Contract

## Scope

This subtree owns SDP documentation: stable references, plans, strategy, roadmap,
workstreams, reviews, evidence, and archived history.

## Contract

Documentation must make the current source of truth easy to find and must not
create competing workflows. Stable behavior belongs in `docs/reference/`; dated
rationale belongs in `docs/plans/`, `docs/strategy/`, or `docs/archive/`.

## Dependencies

Docs may reference code, manifests, schemas, Beads, and workstreams. When docs and
code disagree, verify code or command output before changing policy text.

## Runtime Assumptions

Agents use docs for orientation and evidence. Docs are not a substitute for live
Beads state, command output, or CI logs.

## Local Rules

- Use `docs/reference/project-map.md` for read order and source-of-truth routing.
- Put stable references under `docs/reference/` and link them from its README.
- Keep workstream files under `docs/workstreams/backlog/` with Beads links and
  acceptance criteria.
- Do not edit generated reference outputs by hand when the file says generated.
- Preserve historical docs unless the task explicitly asks for archival cleanup.
