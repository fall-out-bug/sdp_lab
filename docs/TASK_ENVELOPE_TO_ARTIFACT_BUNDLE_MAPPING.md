# Task Envelope → Artifact Bundle Mapping

Status: working model
Date: 2026-03-22
Related:
- `docs/ARTIFACT_TAXONOMY_WORKING_MODEL.md`
- `docs/templates/task-brief.template.md`
- `docs/templates/implementation-plan.template.md`
- `docs/templates/verification-note.template.md`
- `docs/templates/review-note.template.md`
- `docs/templates/handoff-note.template.md`

## Purpose

Translate a task envelope into the minimum recommended SDP artifact bundle.

This document exists to answer the practical question:

> given task type, execution mode, and risk level — which artifacts should be created now?

This is a **working mapping**, not a final fully automated rules engine.

---

## 1. Inputs from task envelope

The most important envelope fields for bundle selection are:

- `task_type`
- `execution_mode`
- `risk_level`
- `required_artifacts`
- `required_checks`
- `target_repo`
- `scope_in` / `scope_out`
- `definition_of_done`

### Priority rule

If `required_artifacts` is explicitly present in the envelope, it overrides the default bundle below.

If not, use the default mapping rules in this document.

---

## 2. Default bundle mapping by task type

### A. `task_type = bugfix`

#### Low risk
Bundle:
- `execution-summary`
- optional `verification-note`

#### Medium risk
Bundle:
- `implementation-plan`
- `execution-summary`
- `verification-note`

#### High risk
Bundle:
- `implementation-plan`
- `execution-summary`
- `verification-note`
- `review-note`
- optional `handoff-note`

---

### B. `task_type = feature`

#### Low risk
Bundle:
- `implementation-plan`
- `execution-summary`
- `verification-note`

#### Medium risk
Bundle:
- `implementation-plan`
- `execution-summary`
- `verification-note`
- optional `review-note`

#### High risk
Bundle:
- `task-brief`
- `implementation-plan`
- `execution-summary`
- `verification-note`
- `review-note`
- optional `handoff-note`

Add `release-note` if user-visible behavior changes.
Add `migration-note` if adoption/compatibility steps are required.

---

### C. `task_type = refactor`

#### Low risk
Bundle:
- `implementation-plan`
- `execution-summary`
- `verification-note`

#### Medium risk
Bundle:
- `implementation-plan`
- `execution-summary`
- `verification-note`
- `review-note`

#### High risk
Bundle:
- `task-brief`
- `implementation-plan`
- `execution-summary`
- `verification-note`
- `review-note`
- optional `decision-record`
- optional `handoff-note`

---

### D. `task_type = review`

Bundle:
- `review-note`

Optional:
- `handoff-note`
- `follow-up-note`

Use `follow-up-note` if review discovers deferred non-blocking work.

---

### E. `task_type = research`

#### Exploration only
Bundle:
- `execution-summary`

#### Research with future execution impact
Bundle:
- `task-brief`
- `execution-summary`
- optional `handoff-note`

If research resolves a durable choice, add `decision-record`.

---

### F. `task_type = process-design`

Bundle:
- `task-brief`
- `decision-record` or equivalent design memo
- optional `implementation-plan`
- optional `handoff-note`

---

### G. `task_type = release`

Bundle:
- `implementation-plan`
- `execution-summary`
- `verification-note`
- `review-note`
- `release-note`

Add `migration-note` if user/operator action is required.

---

### H. `task_type = mixed`

Default bundle:
- `task-brief`
- `implementation-plan`
- `execution-summary`
- `verification-note`

Add:
- `review-note` when risk is medium/high
- `handoff-note` when work spans multiple passes
- `decision-record` when a durable architectural/process choice is made

---

## 3. Execution-mode modifiers

### `execution_mode = explore`
Bias toward:
- lighter bundles
- `execution-summary`
- optional `handoff-note`

Do not force `implementation-plan` unless the exploration already turns into a concrete change strategy.

### `execution_mode = plan`
Bias toward:
- `task-brief`
- `implementation-plan`
- optional `decision-record`

### `execution_mode = build`
Bias toward:
- `implementation-plan` (unless clearly trivial)
- `execution-summary`
- `verification-note`

### `execution_mode = review`
Bias toward:
- `review-note`
- optional `follow-up-note`

### `execution_mode = debug`
Bias toward:
- `execution-summary`
- `verification-note`
- `implementation-plan` if the fix path is non-trivial

### `execution_mode = docs`
Bias toward:
- `execution-summary`
- optional `verification-note`
- `release-note` only if docs communicate important product change

### `execution_mode = mixed`
Use task-type rules first, then add review/handoff as needed.

---

## 4. Risk modifiers

### `risk_level = low`
Prefer minimum bundle.

### `risk_level = medium`
Usually add:
- `implementation-plan`
- `verification-note`

### `risk_level = high`
Usually add:
- `task-brief`
- `review-note`
- optional `decision-record`
- optional `handoff-note`

If release/compatibility impact exists, add `release-note` and/or `migration-note`.

---

## 5. Required-checks modifier

If `required_checks` includes non-trivial validation expectations, prefer including `verification-note` even when the default bundle is otherwise light.

Examples:
- tests required
- provider/config sanity required
- migration-sensitive path touched
- docs/release impact must be checked

---

## 6. Suggested template mapping

### `task-brief`
Use:
- `docs/templates/task-brief.template.md`

### `implementation-plan`
Use:
- `docs/templates/implementation-plan.template.md`

### `verification-note`
Use:
- `docs/templates/verification-note.template.md`

### `review-note`
Use:
- `docs/templates/review-note.template.md`

### `handoff-note`
Use:
- `docs/templates/handoff-note.template.md`

---

## 7. Worked examples

### Example 1
Envelope:
- `task_type = bugfix`
- `execution_mode = build`
- `risk_level = low`

Recommended bundle:
- `execution-summary`
- optional `verification-note`

### Example 2
Envelope:
- `task_type = feature`
- `execution_mode = mixed`
- `risk_level = medium`

Recommended bundle:
- `implementation-plan`
- `execution-summary`
- `verification-note`
- optional `review-note`

### Example 3
Envelope:
- `task_type = release`
- `execution_mode = review`
- `risk_level = high`

Recommended bundle:
- `implementation-plan`
- `execution-summary`
- `verification-note`
- `review-note`
- `release-note`
- optional `migration-note`

---

## 8. Minimum-good-enough rule

This mapping is a decision aid, not a bureaucracy trigger.

If the recommended bundle is obviously too heavy for the real task, reduce it.
If the recommended bundle is too light for the actual risk, expand it.

The point is to avoid both:
- under-documented risky work
- over-documented trivial work
