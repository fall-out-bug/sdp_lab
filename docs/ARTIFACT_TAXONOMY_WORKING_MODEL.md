# SDP Artifact Taxonomy — Working Model

Status: working model
Date: 2026-03-22
Scope: current-state SDP process/artifact layer

## Purpose

Зафиксировать **текущую** artifact taxonomy для SDP.

Это не финальная модель полного AI PDLC/SDLC.
Это рабочая схема, которая нужна уже сейчас, чтобы:
- отличать process layer от OmO agent layer
- явно понимать, какие артефакты нужны в каком классе задач
- стабилизировать handoff между orchestrator, SDP, OmO, и project-local layers

## Important constraint

SDP **может вырасти** в полноценный AI PDLC/SDLC layer.
Но это **roadmap**, а не текущее baseline-предположение.

Поэтому этот документ описывает:
- what SDP must do now
- what artifacts are useful now
- what minimum process evidence is expected now

and explicitly avoids pretending that the full future system already exists.

---

## 1. Core position of SDP

### OmO vs SDP

- **OmO** answers: who thinks, explores, codes, reviews, verifies in the dev-agent layer
- **SDP** answers: what process shape is required, what artifacts matter, what evidence is enough

### Therefore

SDP artifact taxonomy should define:
- required deliverables
- optional deliverables
- handoff artifacts
- evidence artifacts
- acceptance artifacts

It should **not** define model routing, prompt families, or universal execution personas.

---

## 2. Artifact classes

We use five artifact classes in the current working model.

### A. Framing artifacts
Used to shape understanding before execution.

Examples:
- task brief
- feature brief
- problem statement
- scope note

### B. Planning artifacts
Used to turn intent into an executable path.

Examples:
- implementation plan
- execution plan
- workstream plan
- contract definition / contract lock note

### C. Execution artifacts
Used to show what was actually done.

Examples:
- execution summary
- implementation note
- patch/PR evidence bundle
- evidence envelope
- trace update

### D. Verification artifacts
Used to prove the work is good enough.

Examples:
- verification note
- review note
- test/gate output summary
- drift verdict
- QA/UAT verdict

### E. Shipping / continuity artifacts
Used when the change has broader impact or requires downstream continuity.

Examples:
- release note
- migration note
- decision record
- handoff note
- backlog/follow-up note

---

## 3. Canonical current-state artifact set

This is the recommended current canonical vocabulary.

### 3.1 `task-brief`

**Purpose:** define the task in human and process terms.

Contains:
- intent
- scope
- non-goals
- target repo/context
- risk guess
- expected outcome

Use when:
- task is non-trivial
- scope is ambiguous
- multiple layers are involved

May be skipped for:
- tiny, low-risk, single-file work

---

### 3.2 `implementation-plan`

**Purpose:** define how the change should be carried out.

Contains:
- approach
- steps
- boundaries
- key risks
- expected checks
- open questions

Use when:
- feature work
- refactor
- medium/high-risk bugfix
- cross-cutting change
- anything that should not go straight to blind execution

May be skipped for:
- trivial local fixes

---

### 3.3 `execution-summary`

**Purpose:** summarize what was actually done.

Contains:
- changed areas
- major actions taken
- what was intentionally not done
- unexpected findings

Use when:
- any real execution happened

This is the minimum post-execution artifact.

---

### 3.4 `verification-note`

**Purpose:** record what was checked and with what result.

Contains:
- checks performed
- results
- notable warnings
- unverified areas

Use when:
- medium/high-risk change
- behavior changed
- contracts/runtime/provider/release-sensitive areas affected

Optional for:
- tiny low-risk changes

---

### 3.5 `review-note`

**Purpose:** capture human/agent review conclusions.

Contains:
- verdict
- key concerns
- requested fixes or rationale for approval

Use when:
- high-risk changes
- architectural changes
- release-sensitive changes
- non-obvious refactors
- cross-package impact

Optional for:
- small straightforward fixes

---

### 3.6 `decision-record`

**Purpose:** preserve significant reasoning that should survive the task.

Contains:
- decision made
- alternatives considered
- why this path won
- consequences

Use when:
- architectural choice
- process choice with long tail
- contract or boundary decision
- irreversible or high-cost decision

Do not require for routine implementation details.

---

### 3.7 `release-note`

**Purpose:** capture shipping-relevant external meaning.

Contains:
- user-visible impact
- compatibility impact
- important behavioral changes
- notable fixes/features

Use when:
- users or integrators would care
- behavior/config/API/command surface changed

---

### 3.8 `migration-note`

**Purpose:** capture adoption or upgrade implications.

Contains:
- what changed
- who needs to act
- upgrade path / migration steps
- breakage or compatibility caveats

Use when:
- migration-sensitive paths changed
- compatibility expectations changed
- operational upgrade steps are needed

---

### 3.9 `handoff-note`

**Purpose:** allow continuation without hidden context.

Contains:
- current state
- what remains
- blockers/open risks
- exact recommended next step

Use when:
- work spans sessions
- task was partially completed
- responsibility passes between layers/agents/people

---

### 3.10 `follow-up-note`

**Purpose:** capture discovered-but-not-completed work.

Contains:
- what remains
- why it was not done now
- whether it blocks completion
- where it should re-enter backlog

Use when:
- discovered work is deferred
- non-blocking concerns remain

---

## 4. Required artifact policy by task class

### A. Tiny low-risk fix
Required:
- `execution-summary`

Optional:
- `verification-note`
- `handoff-note`

### B. Standard bugfix
Required:
- `execution-summary`
- `verification-note`

Optional:
- `implementation-plan`
- `review-note`

### C. Feature work
Required:
- `implementation-plan`
- `execution-summary`
- `verification-note`

Optional:
- `review-note`
- `release-note`
- `handoff-note`

### D. Refactor / architecture-sensitive change
Required:
- `implementation-plan`
- `execution-summary`
- `verification-note`
- `review-note`

Optional:
- `decision-record`
- `handoff-note`

### E. Release / migration-sensitive change
Required:
- `implementation-plan`
- `execution-summary`
- `verification-note`
- `review-note`
- `release-note` or `migration-note` (or both, if needed)

Optional:
- `decision-record`

### F. Process-design / protocol-shaping task
Required:
- `task-brief`
- `decision-record` or equivalent architectural memo
- `handoff-note` if implementation is deferred

Optional:
- `implementation-plan`

---

## 5. Artifact ownership by layer

### Клавдий (orchestrator)
Typically owns or initiates:
- `task-brief`
- high-level `handoff-note`
- cross-layer synthesis

### SDP
Owns:
- artifact policy
- artifact taxonomy
- definition of done expectations
- when a task requires which artifact class

### OmO
Usually produces/supports:
- `implementation-plan` (when execution planning is needed)
- `execution-summary`
- input into `verification-note`
- input into `review-note`

### Project-local layer
Usually contributes:
- repo-specific review findings
- release/migration implications
- local constraints for plans
- local handoff notes when repo truth matters

---

## 6. Relationship to task envelopes and handoff contract

This taxonomy is meant to work with:
- `task envelope` fields like `required_artifacts`, `required_checks`, `definition_of_done`
- the cross-layer `handoff contract`

### Translation rule

If a task envelope says:
- `required_artifacts: [implementation-plan, verification-note]`

then SDP is responsible for saying:
- why these are required
- what "good enough" means for them

and OmO/project-local layers are responsible for helping produce them.

---

## 7. Minimum good-enough rule

Not every task needs a mountain of paperwork.

### Rule
Demand only the minimum artifact set that:
- preserves clarity
- supports verification
- prevents hidden decisions
- enables safe continuation or release

### Anti-pattern
Do not turn SDP into a bureaucracy generator.

Bad:
- mandatory decision record for every typo fix
- mandatory review artifact for every tiny local edit
- formal release note for internal-only trivial cleanup

---

## 8. Current gaps in SDP

This working model intentionally leaves room for future maturation.

Likely next upgrades later:
- stronger mapping from task class → artifact bundle
- machine-checkable artifact contracts
- richer traceability across feature/workstream/issue/evidence/review/QA
- fuller AI PDLC/SDLC stage coverage

Current minimal templates now exist in:
- `docs/templates/task-brief.template.md`
- `docs/templates/implementation-plan.template.md`
- `docs/templates/verification-note.template.md`
- `docs/templates/review-note.template.md`
- `docs/templates/handoff-note.template.md`

These are valid roadmap directions, but are not assumed complete today.

---

## 9. Short practical formula

### Current SDP artifact backbone
- `task-brief` for framing
- `implementation-plan` for approach
- `execution-summary` for what happened
- `verification-note` for checks
- `review-note` for review
- `decision-record` for durable decisions
- `release-note` / `migration-note` for shipping impact
- `handoff-note` / `follow-up-note` for continuity

### If in doubt
Ask:
1. What must be understood before execution?
2. What must be proved after execution?
3. What must survive beyond this session/change?

The answers determine the minimum artifact bundle.
