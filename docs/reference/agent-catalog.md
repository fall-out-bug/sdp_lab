# Agent Catalog

Status: canonical reference

Canonical design reference:

- `docs/reference/canonical-happy-path.md`
- `docs/plans/2026-04-05-canonical-sdp-happy-path-consistency.md`
- `docs/plans/2026-04-13-sdp-skill-architecture-design.md` (F125 intent model)

This document defines the default SDP agent workflow for the canonical happy path.

Mode note:

- `Local Mode` is the adoption ramp
- this catalog primarily describes the agent ownership model for full `Operator Mode`
- skills (intents) and CLI remain control surfaces over the same stage model

## Intent-Based Agent Model (F125)

With the five-intent skill model, agents are aligned to intents rather than granular skills:

| Intent | Primary Agent | Agent Focus |
|--------|---------------|-------------|
| @understand | `scout` | Codebase discovery, architecture, health synthesis |
| @build | `builder` | Feature creation from design to implementation |
| @fix | `fixer` | Bug resolution from hotfix to systematic investigation |
| @review | `reviewer` | Quality gates across multiple dimensions |
| @operate | `operator` | Deployment, CI triage, backlog planning |

The intent skills (@understand, @build, @fix, @review, @operate) compose CLI tools and workflows to accomplish their goals. The agents listed above are the primary agents that implement those intents.

## Canonical SDP Loop

The default loop is:

- `vision` (via @understand deep)
- `feature` (via @build feature)
- `workstream` graph (via @build or @operate plan)
- `beads issue`
- early `draft PR` (via @build feature)
- `execution` (via @build or @fix)
- findings as `beads issue` (via @review)
- `QA/UAT` (via @review readiness)
- clean `PR` (via @operate deploy)

The default agent stack exists only to move work through that loop.

## Canonical Agents

### `scout` (implements @understand)

Owns:

- codebase discovery and synthesis
- architecture analysis
- health and metrics assessment
- documentation generation
- knowledge base construction (`.sdp/manifest.md`)

Used when:

- first time working with a codebase
- need to understand architecture or dependencies
- checking codebase health or technical debt
- before starting feature work or major refactors

Primary output:

- project card (quick mode)
- complete understanding with architecture and health (standard mode)
- knowledge base with documentation and index (deep mode)

### `builder` (implements @build)

Owns:

- feature creation from idea to implementation
- design documentation
- TDD workflow (test-first implementation)
- prototype creation
- PR preparation

Used when:

- implementing new features or components
- creating designs or system architectures
- prototyping quick solutions
- user-facing work with clear acceptance criteria

Primary output:

- design document (idea mode)
- complete feature with tests and PR (feature mode)
- working prototype with experimental label (prototype mode)

### `fixer` (implements @fix)

Owns:

- bug resolution and error diagnosis
- root cause analysis
- regression testing (TDD for fixes)
- hotfix execution
- systematic issue resolution

Used when:

- known bugs with clear reproduction steps
- test failures or CI breaks
- production incidents or errors
- error logs or stack traces available

Primary output:

- minimal fix with regression test (quick mode)
- root cause and reproduction steps (investigate mode)
- complete fix with comprehensive tests and RCA (systematic mode)

### `reviewer` (implements @review)

Owns:

- engineering review across multiple dimensions
- quality gate verification
- finding categorization and routing
- release readiness assessment

Used when:

- PR ready for engineering review
- before merging to main
- architecture or security concerns
- release readiness verification

Primary output:

- pass/fail verdict with specific findings by dimension
- beads issues for blocking findings
- re-review criteria for failures

Dimensions: code, architecture, security, performance, readiness, reality

### `operator` (implements @operate)

Owns:

- deployment execution and verification
- CI failure triage and diagnosis
- backlog planning and workstream decomposition
- system monitoring and alerting

Used when:

- deploying to production or staging
- CI failures need investigation
- system monitoring or alerts
- converting insights into backlog

Primary output:

- deployed system with verification (deploy mode)
- categorized failures and assigned issues (triage mode)
- structured backlog with dependencies (plan mode)

## Canonical Stage Routing (Intent-Based)

| Stage | Primary intent | Primary agent | Required result |
|-------|---------------|---------------|-----------------|
| `vision` | @understand | `scout` | updated project map, knowledge base |
| `feature` shaping | @build | `builder` | accepted `feature`, design doc |
| `workstream` + `beads issue` mapping | @build or @operate | `builder` or `operator` | executable leaf graph |
| early `draft PR` | @build | `builder` | active branch and draft PR |
| `execution` | @build or @fix | `builder` or `fixer` | change or blocker |
| review and gates | @review | `reviewer` | pass or typed findings |
| `QA/UAT` | @review | `reviewer` | `qa:pass` or `qa:fail` |
| merge readiness | @review | `reviewer` | clean `PR` |
| release or deploy | @operate | `operator` | deployed system |

## Optional Advisors

These are not part of the default happy path but may be invoked for specialized needs:

- `oracle` - hard architecture, debugging, security, or tradeoff cases
- `reality` - repo audits and reality checks (now part of @review reality dimension)
- specialist review modes such as security, sre, devops, or ux when a feature truly needs them (now @review dimensions)

Rule:

- if a role does not own a unique SDP transition or intent, it should not be a top-level default agent

## Reduction Rules

Merge or delete when possible:

- planning personas that duplicate `@build` idea mode or `@operate` plan mode
- supervisor or synthesis roles that duplicate `@operate` plan mode
- multiple review personas — use `@review` dimensions instead
- market or growth personas on the default engineering path

Canonical target:

- 5 primary agents (aligned to 5 intents)
- small optional advisor bench for specialized needs

## Agent Quality Rule

Every top-level agent must answer three questions clearly:

- what intent it implements
- what SDP stage or entity it updates
- what artifact or verdict it must emit

If an agent cannot answer those three, it should not stay top-level.
