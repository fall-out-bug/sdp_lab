# Canonical SDP Happy Path Consistency Design

> **Status:** Draft design
> **Date:** 2026-04-05
> **Goal:** Define one coherent SDP product path from task intake to deploy, then align docs, help, and operator surfaces around that path.

Related:

- [docs/reviews/2026-04-05-sdp-happy-path-audit.md](../../reviews/2026-04-05-sdp-happy-path-audit.md)
- [docs/plans/2026-03-15-canonical-sdp-loop-and-agent-stack.md](2026-03-15-canonical-sdp-loop-and-agent-stack.md)
- [docs/SDP_OPERATOR_WORKFLOW.md](../../SDP_OPERATOR_WORKFLOW.md)
- [docs/REAL_FEATURE_TO_PR_RUNBOOK.md](../../REAL_FEATURE_TO_PR_RUNBOOK.md)
- [docs/CONTROL_TOWER_V2_WORKING_MODEL.md](../../CONTROL_TOWER_V2_WORKING_MODEL.md)
- [docs/decisions/ADR-BEADS-FIRST-SOURCE-OF-TRUTH.md](../../decisions/ADR-BEADS-FIRST-SOURCE-OF-TRUTH.md)

---

## 1. Problem

SDP currently describes different happy paths depending on which surface the user reads first.

Today we have three competing stories:

1. public docs story: `@feature -> @oneshot -> @review -> @deploy`
2. CLI story: `sdp init -> sdp plan -> sdp apply -> sdp status/next`
3. internal operator story: `Beads -> draft PR -> findings loop -> QA/UAT -> merge`

This is not progressive disclosure.
It is product split-brain.

The result:

- newcomers cannot tell what the default path is
- contributors learn a stale workflow
- CLI help and docs disagree about first-success behavior
- the "drop work on the board and let SDP carry it to deploy" promise is not stated as one coherent system

The system needs one design, not more local copy edits.

---

## 2. Product Decision

SDP will describe one canonical product story:

> A task enters the board-backed queue. SDP shapes it, asks only for missing information, executes it through agents, routes findings back into the same queue, and drives it to a clean deploy path with visible proof.

Everything else is a disclosure layer or control surface for that same story.

That means:

- the canonical state model is board-to-deploy
- the board is not optional in the full product promise
- skills and CLI are not separate workflows
- public install/onboarding is a shorter on-ramp into the same system

This design resolves the ambiguity by introducing two explicit operating modes.

---

## 3. Operating Modes

### 3.1 Local Mode

Use this when a user wants SDP inside one repo and is not yet running a shared queue.

Goals:

- install SDP
- initialize repo scaffolding
- inspect health and next step
- shape and execute work locally

Characteristics:

- `Beads` is optional
- there may be no visible board UI
- CLI and skills operate directly in the repo
- user still gets the same stages, but without the full operator queue

Primary first-success path:

1. install SDP
2. `sdp init`
3. `sdp doctor`
4. shape a task into executable work
5. execute and verify

### 3.2 Operator Mode

Use this when SDP is presented as a task system that can carry work from queue to deploy.

Goals:

- intake tasks into a durable queue
- shape work with progressive disclosure
- let agents execute with visible status
- route findings back into execution
- reach QA/UAT and deploy with traceable evidence

Characteristics:

- `Beads` is required
- the board is a projection over Beads-backed operational truth
- PR, findings, QA/UAT, and deploy are first-class stages
- `status` and `next` explain where the item is and what happens next

Primary path:

1. task enters board-backed queue
2. clarification and shaping
3. executable graph becomes ready
4. early draft PR exists
5. agents execute
6. findings re-enter queue
7. QA/UAT passes
8. human approves merge or deploy step

### 3.3 Why both modes exist

This is not two products.

It is one system with progressive disclosure:

- `Local Mode` is the adoption ramp
- `Operator Mode` is the full product promise

Public docs must stop pretending that the full board-to-deploy story is available without a queue.

---

## 4. Source-of-Truth Model

The system must state explicitly what is authoritative at each layer.

| Layer | Canonical truth | Why |
|------|------------------|-----|
| Operational queue state | `Beads` | Accepted by ADR; durable task graph, readiness, blockers, status |
| Semantic intent | `feature` and `workstream` artifacts | These define meaning and acceptance |
| Integration state | branch + early `draft PR` | Review and merge happen here |
| Verification proof | `evidence`, `trace`, `drift`, `QA/UAT` artifacts | This is completion proof, not queue truth |
| Board/status views | derived projections | They explain state; they do not own it |
| CLI and skills | control surfaces | They operate the system; they do not define truth |

### 4.1 Board semantics decision

The board is a derived view over Beads-backed operational truth.

This is not optional.
It follows directly from [ADR-BEADS-FIRST-SOURCE-OF-TRUTH.md](../../decisions/ADR-BEADS-FIRST-SOURCE-OF-TRUTH.md).

Therefore:

- SDP must not describe the board as an independent store
- SDP must not imply a future control-tower UI is already the canonical source
- public docs must be honest that today's board-backed mode is Beads-first, with richer board views as projections

### 4.2 PR semantics decision

The PR is the integration and review surface, not the planning surface.

Rules:

- the PR is opened early
- review findings re-enter Beads
- QA/UAT happens after engineering gates are clean
- merge remains a human approval step

`@deploy` and `sdp deploy` must be documented accordingly.
They cannot keep implying "automatic production deploy" if the real system is "clean PR plus approval and evidence."

---

## 5. Canonical Happy Path

The canonical happy path is stage-first.
Commands and skills are mapped onto stages, not the other way around.

### Stage 0: Install and trust bootstrap

Outcome:

- repo has SDP installed
- supported IDE integration is present
- environment is healthy enough to continue

Primary artifacts:

- `.sdp/config.yml`
- integration-specific prompt surface
- optional CLI install

Primary surfaces:

- `README`
- `QUICKSTART`
- `sdp init`
- `sdp doctor`

### Stage 1: Intake

Outcome:

- a task exists in the board-backed queue
- raw request is captured
- ownership and current state are visible

Primary artifacts:

- `Beads` issue
- optional intake artifact or source link

Primary surfaces:

- queue or board entrypoint
- `bd ready`, `bd show`
- future board or control-tower projection

### Stage 2: Clarification and shaping

Outcome:

- SDP asks only for missing information
- a `feature` and linked `workstream` contracts exist
- executable Beads graph is ready

Primary artifacts:

- `feature` definition
- `workstream` files
- linked Beads issues

Primary surfaces:

- conversational shaping skills
- `sdp plan`
- feature agent

### Stage 3: Execution setup

Outcome:

- active branch exists
- early draft PR exists
- ready work is visible

Primary artifacts:

- branch
- draft PR
- queue metadata and trace linkage

Primary surfaces:

- orchestrator
- PR tooling
- `sdp status`
- `sdp next`

### Stage 4: Execution

Outcome:

- agents implement a ready slice
- evidence is emitted
- drift inputs are available

Primary artifacts:

- code/doc changes
- execution evidence
- trace links
- drift inputs

Primary surfaces:

- `@oneshot`
- `@build`
- `sdp apply`
- `sdp build`
- implementer agent

### Stage 5: Findings loop

Outcome:

- review, CI, and drift findings become queue items
- blocking vs non-blocking work is visible
- the system returns to execution only when needed

Primary artifacts:

- typed findings in Beads
- updated trace
- PR state

Primary surfaces:

- reviewer agent
- CI
- drift checks
- `sdp status`
- `sdp next`

### Stage 6: QA/UAT

Outcome:

- feature behavior is checked against intent
- pass or fail is explicit

Primary artifacts:

- `qa:pass` evidence or `qa:fail` findings

Primary surfaces:

- QA agent
- operator review surface

### Stage 7: Delivery

Outcome:

- clean PR exists
- approval state is visible
- deploy or merge proof is recorded

Primary artifacts:

- merged PR or approved release state
- deploy or approval event
- final trace linkage

Primary surfaces:

- human approval
- `@deploy`
- `sdp deploy`

---

## 6. Interface Model

The system must stop documenting commands as if they define separate workflows.

### 6.1 Skills

Skills are guided journeys.

Use them when the user wants:

- conversation
- clarification
- progressive disclosure
- agent-driven execution

They should be described as the guided surface over canonical stages.

### 6.2 CLI

CLI commands are explicit control primitives.

Use them when the user wants:

- terminal control
- scripting
- machine-readable state
- deterministic automation

They should be described as stage controls and observability tools.

### 6.3 Board and queue

Board or queue views are state visibility and intake surfaces.

They answer:

- what exists
- what needs attention
- what is waiting on a human
- what is blocked
- what is ready
- what reached review or deploy

They do not replace CLI or skills.
They make the system legible.

### 6.4 Design rule

Docs must describe one stage model and then show:

- guided surface example
- explicit CLI example

Never two separate happy paths.

---

## 7. Progressive Disclosure Design

The product must disclose complexity in layers.

### Layer A: Product promise

Audience:

- newcomer
- evaluator
- OSS user

Question answered:

- what does SDP do for me?

Owned by:

- `README`

Must state:

- one-sentence promise
- two modes: local and operator
- honest note that full board-backed mode requires Beads
- where to start next

### Layer B: First success

Audience:

- repo adopter
- first-time operator

Question answered:

- how do I get one task through the system?

Owned by:

- `QUICKSTART`

Must state:

- install and init
- health check
- how task intake works today
- local mode first success
- operator mode first success

### Layer C: Stage navigation

Audience:

- active user in the terminal

Question answered:

- what command should I run now?

Owned by:

- CLI help
- `sdp status`
- `sdp next`
- `CLI_REFERENCE`

Must state:

- stage purpose
- input and output
- adjacent commands
- when this command is not the right tool

### Layer D: Detailed operating model

Audience:

- contributor
- advanced operator
- maintainer

Question answered:

- how is the system supposed to work end to end?

Owned by:

- operator workflow docs
- this design doc

### Layer E: Deep reference

Audience:

- maintainers
- auditors
- specialists

Question answered:

- what is the exact spec?

Owned by:

- reference docs that match runtime truth

If a reference page is stale, it must be marked `legacy` or removed from primary navigation.

---

## 8. Surface Ownership

Each surface must answer one job and stop freelancing into others.

| Surface | Primary job | Must not do |
|--------|-------------|-------------|
| `sdp/README.md` | state the product promise and route users to the right entrypoint | explain full internal protocol or outdated skill-only path |
| `sdp/docs/QUICKSTART.md` | deliver first success with progressive disclosure | act as full protocol spec |
| `sdp/CONTRIBUTING.md` | explain how to contribute to SDP itself using the canonical flow | teach a stale private workflow as if it is canonical |
| `sdp/docs/CLI_REFERENCE.md` | map command groups to stages and purposes | compete with runtime help or define product strategy |
| CLI `--help` | runtime truth for syntax, behavior, and examples | carry full architectural theory |
| operator workflow docs | explain the full operator loop | pretend to be the first document a newcomer should open |
| reference docs | exact detailed specs | remain in primary navigation when stale |

### 8.1 Help-copy standard

Each important command help surface should answer these questions in order:

1. what stage is this for?
2. when should I use this command?
3. what artifact or state does it read or write?
4. what usually comes before it?
5. what usually comes after it?

Without this structure, component help becomes a pile of flags instead of a usable system guide.

---

## 9. Artifact Chain

The happy path must expose a visible artifact chain from intake to deploy.

| Stage | Artifact |
|------|----------|
| intake | queue item or board-backed task |
| shaping | feature definition and clarification record |
| decomposition | workstream files |
| execution graph | linked Beads issues |
| integration | branch + early draft PR |
| implementation | code/doc change and execution evidence |
| review loop | typed findings and updated queue state |
| QA/UAT | pass or fail evidence |
| delivery | approval or deploy event |

This chain is the minimum needed to make the board promise credible.

---

## 10. Terminology Decision

We need one stable vocabulary for public and private docs.

### Canonical terms

- `board`: visibility surface over Beads-backed queue state
- `queue item`: operational work item in Beads
- `feature`: user-visible outcome and acceptance intent
- `workstream`: execution contract for one change slice
- `finding`: review, CI, drift, or QA issue that re-enters the queue
- `delivery`: clean PR plus approval or deploy proof

### Terms to avoid as primary public language

- `.claude`-only skill paths as if they are universal
- `deploy` as shorthand for "automatically shipped to production" when the real system is approval-gated
- legacy identifiers as if they are current public concepts

---

## 11. Rollout Plan

### Phase 1: Canon lock

Deliverables:

- this design doc
- one short public-facing "canonical happy path" doc if needed
- one stable terminology table

Exit criteria:

- maintainers agree on modes, truth model, and stage model

### Phase 2: Entry surface rewrite

Deliverables:

- rewrite `README`
- rewrite `QUICKSTART`
- rewrite `CONTRIBUTING`

Exit criteria:

- all three surfaces describe the same system

### Phase 3: Help and CLI alignment

Deliverables:

- tighten root help and stage command help
- rewrite `CLI_REFERENCE`
- make `status` and `next` first-class onboarding tools

Exit criteria:

- runtime help and doc examples agree

### Phase 4: Reference cleanup

Deliverables:

- rewrite or mark legacy stale reference pages
- remove `.claude`-only assumptions from primary docs
- align skill/reference docs with current stage model

Exit criteria:

- primary reference set matches runtime truth

### Phase 5: Board-to-deploy proof

Deliverables:

- one end-to-end walkthrough from queue to deploy
- visible artifact examples
- visible findings loop example

Exit criteria:

- a new user can trace one item from intake to delivery without opening historical plans

---

## 12. Non-Goals

This design does not:

- build a new board UI now
- replace Beads as operational truth
- promise unattended production deploy
- rewrite every historical document in one pass
- settle every future control-tower implementation detail

The job here is consistency of the system story and the surfaces that expose it.

---

## 13. Acceptance Criteria

This design is successful only when these statements become true in the shipped surfaces:

1. A newcomer can identify the default SDP path in under 30 seconds.
2. Public docs explain that full board-backed operation requires Beads and that board views are projections over that truth.
3. Skills and CLI are described as two control surfaces for one stage model.
4. `README`, `QUICKSTART`, `CONTRIBUTING`, CLI help, and `CLI_REFERENCE` no longer contradict each other about the primary happy path.
5. The path from intake to deploy exposes a visible artifact chain and a visible findings loop.
6. Contributors can tell where human approval still exists and where agents are expected to operate autonomously.

---

## 14. Recommendation

Lock this design first.

Then execute consistency work in this order:

1. public entry surfaces
2. CLI help and CLI reference
3. stale reference cleanup
4. end-to-end walkthrough proof

Do not start by polishing isolated files.
That is exactly how the current inconsistency was created.
