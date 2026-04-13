# SDP Skills Reference

Status: canonical reference

Canonical design reference:

- `docs/reference/canonical-happy-path.md`
- `docs/plans/2026-04-05-canonical-sdp-happy-path-consistency.md`

This document defines the public SDP skill surface and the internal or conditional skills that support the canonical workflow.

Skills are the guided control surface over the canonical stage model.
They are not a second workflow separate from board state, Beads, CLI, PR, or `QA/UAT`.

Mode note:

- `Local Mode` may use skills without a full shared queue
- full board-backed `Operator Mode` still depends on Beads-backed operational truth

## Canonical Public Surface

The public happy path is:

- `@vision`
- `@feature`
- `@oneshot`
- `@review`
- `@qa`
- `@deploy`

Rule:

- if a skill is part of the default user path, it must map directly to one SDP stage

## Stage Mapping

| SDP stage | Primary skill | Result |
|-----------|---------------|--------|
| `vision` | `@vision` | updated project map |
| `feature` shaping | `@feature` | accepted `feature` |
| `workstream` + `beads issue` mapping | `@feature` | executable graph |
| early `draft PR` + graph execution | `@oneshot` | active `PR` and progressing execution |
| engineering review | `@review` | pass or typed findings |
| `QA/UAT` | `@qa` | `qa:pass` or `qa:fail` |
| release or deploy path | `@deploy` | post-merge delivery actions |

## Public Skills

### `@vision`

Use when:

- the user is shaping or revising project direction
- the project map is incomplete or outdated

Updates:

- `vision`

Must emit:

- updated project map or explicit unresolved questions

### `@feature`

Use when:

- the user wants to define, refine, or decompose one `feature`

Updates:

- `feature`
- `workstream`
- linked `beads issue`

Must emit:

- accepted `feature`
- executable `workstream`
- linked `beads issue` graph

Notes:

- internal feature clarification absorbs what older `@idea` and `@design` paths used to do on the happy path
- separate `plan` is optional when the `beads issue` graph is already sufficient
- not every `workstream` is directly executable: only `leaf workstream` entries bind to live execution

### `@oneshot`

Use when:

- the feature is ready for execution
- the orchestrator should walk the ready `beads issue` graph

Updates:

- branch state
- early `draft PR`
- execution state for executable `leaf workstream` entries
- `evidence`, `trace`, and `drift` inputs through the loop

Must emit:

- progressing or clean `PR`
- explicit blockers when the graph cannot advance

### `@review`

Use when:

- engineering review and gate validation are needed

Updates:

- review state
- finding issues in `beads`

Must emit:

- pass verdict, or
- typed `beads issue` findings with source and blocking metadata

### `@qa`

Use when:

- engineering gates are clean and the feature needs intent validation

Updates:

- `QA/UAT` verdict
- blocking or non-blocking `beads issue` findings when needed

Must emit:

- `qa:pass` with `UAT evidence`, or
- `qa:fail` with blocking `beads issue`

### `@deploy`

Use when:

- the work is merge-ready and there is a real release or deploy path to execute

Updates:

- release or deployment state

Must emit:

- deployment action or explicit human gate requirements

## Internal or Conditional Skills

These are useful, but they are not the public happy path.

### `@build`

Purpose:

- execute one `leaf workstream` or one ready `beads issue`

Use when:

- the orchestrator is executing one unit of work
- a human wants a narrow execution step instead of full `@oneshot`

### `@debug`

Purpose:

- systematic failure analysis

Use when:

- execution or verification is failing
- the next step is not obvious from evidence alone

### `@issue`

Purpose:

- classify bug or failure work and route it into the right path

Use when:

- incoming work is a bug, incident, or unclear failure report

### `@reality` and `@reality-check`

Purpose:

- verify repo reality against docs, expectations, or assumptions

Use when:

- a feature or workstream may be drifting from code reality
- an audit is needed before risky changes

### `@strataudit`

Purpose:

- run a document-backed strategy traceability audit as a reusable discovery capability with explicit trust boundaries

Use when:

- the user needs evidence-backed alignment analysis across strategy, architecture, design, or execution documents
- the user needs one of these modes: `corpus-audit`, `traceability-audit`, `coverage-audit`, `evidence-pack`, `report-redraft`
- the harness can inject a native runtime, or the repo has a configured compatible network runtime, or reusable `.strataudit/` artifacts already exist

Must emit:

- `.strataudit/report.json`
- `.strataudit/report.html`
- explicit runtime choice or artifact-only path
- key trust caveats and what is not claimed

References:

- `docs/STRATAUDIT.md`
- `docs/reference/strataudit-evidence-policy.md`
- `docs/reference/strataudit-runtime-policy.md`
- `docs/reference/strataudit-output-modes.md`

## Skills To Absorb or Demote

- `@idea` is internal to `@vision` and `@feature`
- `@design` is internal to `@feature` unless manual decomposition is explicitly requested
- explicit `plan` flow is conditional, not a mandatory public step
- `superpowers` is an upstream capability source, not a second canonical skill tree for SDP

Current intake decision:

- keep the public path unchanged;
- absorb only curated capabilities from `superpowers` into SDP;
- wave 1 candidates are the visual companion, a conditional local worktree helper, and the internal two-stage review pattern used by `@oneshot`.

Reference:

- `docs/plans/2026-04-01-superpowers-curated-intake-wave-1.md`

Skills that still depend on phantom commands, wrong language assumptions, or duplicated workflow logic should be rewritten or removed.

## Skill Quality Rule

Every surviving skill must answer clearly:

- when it is the right entry point
- what SDP entity it updates
- what artifact or verdict it emits

If a skill cannot answer those three, it should not stay on the public path.

## Operator Rule

When in doubt, prefer the canonical path:

- `@vision`
- `@feature`
- `@oneshot`
- `@review`
- `@qa`
- `@deploy`

Everything else is support, exception handling, or narrow execution.
