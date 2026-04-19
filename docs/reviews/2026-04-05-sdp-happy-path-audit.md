# SDP Happy Path Audit

Status: review
Date: 2026-04-05
Scope: public onboarding surfaces in `sdp/` plus canonical operator workflow in `sdp_lab/`

## Goal

Evaluate whether SDP currently presents one coherent happy path from:

1. task enters the board or queue
2. the system shapes it with progressive disclosure
3. agents execute and verify the work
4. findings loop back into execution
5. the feature reaches deploy with visible proof

## Desired Happy Path

The intended product story is:

1. user drops work onto the board
2. SDP turns that into a shaped feature and executable work
3. agents ask only the missing questions
4. agents implement, verify, review, and surface findings
5. the operator sees clear status and next action
6. the system reaches a clean deploy path with evidence

This must be legible through:

- `README`
- `QUICKSTART`
- `CONTRIBUTING`
- CLI help and CLI reference
- operator workflow docs
- skill and reference docs

## Verdict

There is no single canonical happy path today.

Current surfaces describe at least three different products:

1. public docs path: `@feature -> @oneshot -> @review -> @deploy`
2. CLI path: `sdp init -> sdp plan -> sdp apply -> sdp status/next`
3. private operator path: `Beads -> draft PR -> findings loop -> QA/UAT -> merge`

These paths overlap, but they are not presented as progressive disclosure of one system. They compete.

## Findings

### Critical

1. No public entrypoint explains the board-to-deploy loop.

   Public docs in [`sdp/README.md`](https://github.com/fall-out-bug/sdp/blob/main/README.md) and [`sdp/docs/QUICKSTART.md`](https://github.com/fall-out-bug/sdp/blob/main/docs/QUICKSTART.md) start from install and then jump to skills, but the actual execution system in [`docs/SDP_OPERATOR_WORKFLOW.md`](../SDP_OPERATOR_WORKFLOW.md) is Beads and PR driven. The user requirement "drop task on board and let SDP agents drive it to deploy" is not documented as the primary path anywhere.

2. Public docs and CLI describe different primary interfaces.

   Public docs lead with skill invocations like `@feature` and `@oneshot`. Runtime help from `sdp --help`, `sdp plan --help`, and `sdp apply --help` leads with CLI commands as the visible workflow. This is not progressive disclosure. It is two different products speaking at once.

3. Reference docs are materially stale and contradict current runtime truth.

   [`sdp/docs/reference/skills.md`](https://github.com/fall-out-bug/sdp/blob/main/docs/reference/skills.md), [`sdp/docs/reference/design-spec.md`](https://github.com/fall-out-bug/sdp/blob/main/docs/reference/design-spec.md), [`sdp/docs/reference/review-spec.md`](https://github.com/fall-out-bug/sdp/blob/main/docs/reference/review-spec.md), and [`sdp/docs/reference/build-spec.md`](https://github.com/fall-out-bug/sdp/blob/main/docs/reference/build-spec.md) still describe old `.claude`-only paths, Python-specific gates, `@design`-centric planning, and deployment behavior that no longer matches current CLI or current operator loop.

### Major

1. `QUICKSTART` has no honest progressive disclosure from newcomer to operator.

   [`sdp/docs/QUICKSTART.md`](https://github.com/fall-out-bug/sdp/blob/main/docs/QUICKSTART.md) mixes install, init, skills, optional Beads, and flow summary, but it never says which path is default for:

   - solo user in a repo
   - team using Beads
   - operator using PR and QA/UAT gates

2. `CONTRIBUTING` is not aligned with the actual delivery model.

   [`sdp/CONTRIBUTING.md`](https://github.com/fall-out-bug/sdp/blob/main/CONTRIBUTING.md) still teaches `@idea -> @design -> @build -> @review -> @deploy` as the contribution workflow. That ignores the current CLI path and ignores the Beads/PR/QA/UAT loop that SDP itself uses internally.

3. CLI help is locally coherent but not connected to the public docs story.

   `sdp --help`, `sdp init --help`, `sdp doctor --help`, `sdp status --help`, `sdp next --help`, `sdp plan --help`, `sdp apply --help`, `sdp build --help`, `sdp verify --help`, and `sdp deploy --help` are much closer to a usable CLI product surface than the reference docs. The problem is not runtime help quality alone. The problem is that README and reference docs do not treat CLI help as source of truth.

4. The docs do not clearly explain where Beads is optional and where it is required.

   In public docs Beads is positioned as optional tooling. In the actual operator workflow it is the live execution graph. That ambiguity is acceptable only if SDP explicitly defines two modes:

   - lightweight local mode
   - full operator mode

   Right now that split is implicit.

### Minor

1. `README` references `docs/INSTALL.md`, but that file does not exist.
2. Several reference docs still assume `.claude/skills/*` as the only skill surface despite Cursor, OpenCode, and Codex support.
3. Terms like `feature`, `idea`, `workstream`, `task`, `board`, `Beads issue`, and `deploy` are used across public and private docs without one stable operator glossary for the current loop.

## Recommended Direction

Do not patch surfaces independently.

First define one canonical happy path with two explicit layers:

1. `Adopt SDP in my repo`
   - install
   - `sdp init`
   - `sdp doctor`
   - choose interaction mode: skills or CLI
   - run a first success path

2. `Run SDP as an operator system`
   - intake onto board or Beads-backed queue
   - shape feature and workstreams
   - execute through agents
   - loop findings back into queue
   - verify, QA/UAT, deploy

That second layer must explicitly state whether "board" is:

- a Beads-backed queue
- a projected read model over Beads
- future control-tower UI that is not yet the default public path

Without that answer, the product promise remains aspirational rather than usable.

## Remediation Order

### Phase 1: Canonical path spec

Produce one short source-of-truth doc for:

- user types
- default path
- optional advanced path
- artifacts created at each step
- where progressive disclosure happens
- where human approval is required

### Phase 2: Entry surfaces

Bring these into alignment with the canonical path:

- [`sdp/README.md`](https://github.com/fall-out-bug/sdp/blob/main/README.md)
- [`sdp/docs/QUICKSTART.md`](https://github.com/fall-out-bug/sdp/blob/main/docs/QUICKSTART.md)
- [`sdp/CONTRIBUTING.md`](https://github.com/fall-out-bug/sdp/blob/main/CONTRIBUTING.md)
- [`sdp/docs/CLI_REFERENCE.md`](https://github.com/fall-out-bug/sdp/blob/main/docs/CLI_REFERENCE.md)

### Phase 3: Runtime help and reference cleanup

Treat runtime help as product truth and make reference docs subordinate:

- remove stale spec pages or mark them legacy
- stop documenting old `.claude`-only skill paths as canonical
- rewrite skill/reference docs around the chosen happy path

### Phase 4: Board-to-deploy proof

Add one concrete walkthrough that proves the promise:

1. task enters board or queue
2. clarifications requested only when needed
3. work decomposed into executable units
4. implementation completed
5. findings loop visible
6. review and deploy completion visible

## Acceptance Criteria For This Product Slice

- A newcomer can identify one canonical entry doc in under 30 seconds.
- A contributor can tell whether the default flow is skill-first, CLI-first, or Beads-first without inference.
- A user can see how a task moves from board or queue to deploy without opening private historical plans.
- CLI help, quickstart, and contribution docs no longer describe conflicting primary workflows.
- Reference docs either match runtime truth or are explicitly labeled legacy.
