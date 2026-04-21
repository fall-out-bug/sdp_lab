---
name: spec-interrogate
description: Validate a spec through context-stripped questioning before planning or implementation begins.
version: 1.0.0
tags:
  - discovery
  - spec
  - quality-gate
  - socratic
compatibility:
  - claude-code
  - opencode
  - cursor
  - codex
---

# spec-interrogate

## Purpose

Pressure-test a text artifact before the next delivery step.
The interrogator gets only the artifact, with no chat history, beads context, or author explanation. That strips proximity bias and exposes what the future implementer still cannot infer from the document alone.

## Use When

- Before `sdp phase plan` for non-trivial Discovery output.
- Before committing to a risky architecture or API plan.
- When a spec looks coherent to the author, but may still hide undefined terms, missing failure paths, or scope leaks.

Do not use this skill for code review, implementation, or general research. Use `@review`, `@build`, or `@research` for those jobs.

## Roles

**Author** owns the artifact and makes edits.

**Interrogator** is a fresh agent. It receives only the artifact and invocation parameters. It does not edit, negotiate scope, or invent product decisions. It only challenges what the artifact fails to make explicit.

## Inputs

| Parameter | Default | Meaning |
|---|---|---|
| `artifact-path` | required | Path to the spec, plan, design doc, or schema under review |
| `--mode` | `socratic` | Interrogation mode |
| `--questions` | `5` | Max questions per round in `socratic` mode |
| `--rounds` | `5` | Max rounds in `socratic` mode |
| `--feature-id` | optional | Feature identifier for SDP traces |
| `--evidence-path` | `.sdp/evidence/spec-interrogate.json` | Machine-readable result |
| `--report-path` | `.sdp/reports/spec-interrogate.md` | Human-readable blocking report |

## Common Contract

Every mode must produce both outputs:

1. **Human report** at `report-path`
2. **Evidence JSON** at `evidence-path`

### Report Requirements

The report is mandatory for `PASS`, `REWORK`, and `ABORT`.
It must contain:

- artifact path and mode
- short summary of what was tested
- ordered list of unresolved questions or vulnerabilities
- explicit verdict
- next action for the author

For `PASS`, the unresolved list is empty.
For `REWORK` and `ABORT`, the unresolved list is the canonical handoff artifact. Stdout is only a transport, not the source of truth.

### Evidence Requirements

The JSON file must reference the report and preserve the blocking questions in structured form:

```json
{
  "interrogate_verdict": "PASS",
  "artifact_path": "docs/discovery/my-feature/validation.md",
  "feature_id": "F042",
  "mode": "socratic",
  "rounds_completed": 2,
  "max_rounds": 5,
  "open_questions_count": 0,
  "open_questions": [],
  "report_path": ".sdp/reports/spec-interrogate.md",
  "report_summary": "No unresolved implementation-blocking questions remain."
}
```

Each `open_questions[]` item must contain:

```json
{
  "id": "Q1",
  "type": "scope-ambiguity",
  "impact": "plan-blocking",
  "question": "What is the fallback behavior when the upstream model call times out?"
}
```

`open_questions_count` is the count of unresolved items in the final report, not the total number ever raised.

## Question Taxonomy

Prioritize questions that block planning or implementation:

1. `why` — why this decision exists
2. `undefined-term` — a key term appears but is not defined
3. `missing-error-path` — failure behavior is unspecified
4. `scope-ambiguity` — ownership or boundary is unclear
5. `unstated-assumption` — the author relies on context not present in the artifact

Do not spend rounds on style, formatting, or taste unless they hide meaning.

## Protocol (mode `socratic`)

Use this when the author can iteratively revise the artifact.

### Steps

1. Interrogator reads only the artifact and parameters.
2. It asks up to `N` high-impact questions.
3. Author updates the artifact. The answer is the edit, not a chat reply.
4. Interrogator re-reads the updated artifact and checks whether the blocking questions are resolved.
5. Repeat until convergence or `--rounds` is exhausted.

### Verdict Rules

- `PASS`: final round finds `0` unresolved questions
- `REWORK`: round limit reached and unresolved questions remain
- `ABORT`: author explicitly stops the process

### Accounting Rules

- `rounds_completed` = actual rounds performed
- `max_rounds` = configured cap
- `open_questions_count` = unresolved questions after the final artifact revision

## Protocol (mode `cold-read`)

Use this for a cheap first pass.

### Steps

1. Interrogator reads the artifact once.
2. It writes three sections to the report:
   - `What I believe this artifact says`
   - `What I still cannot infer`
   - `What I would refuse to assume`
3. It converts every unresolved inference into `open_questions[]`.

### Verdict Rules

- `PASS`: the summary is coherent and `open_questions_count = 0`
- `REWORK`: any unresolved inference remains
- `ABORT`: author explicitly stops after the report

### Accounting Rules

- `rounds_completed = 1`
- `max_rounds = 1`

## Protocol (mode `adversarial`)

Use this for security, reliability, or abuse-case review of the artifact itself.

### Steps

1. Interrogator reads the artifact once.
2. It lists concrete failure modes, trust-boundary gaps, abuse vectors, and mitigation holes.
3. Every unresolved issue becomes an `open_questions[]` item with `impact = plan-blocking` unless the gap is explicitly minor.

### Verdict Rules

- `PASS`: no unresolved blocking vulnerabilities or failure-mode gaps
- `REWORK`: any unresolved vulnerability or mitigation gap remains
- `ABORT`: author explicitly stops after the report

### Accounting Rules

- `rounds_completed = 1`
- `max_rounds = 1`

## Protocol (mode `impl-test`)

Use this when the real question is: "Could another agent implement this without hallucinating?"

### Steps

1. Interrogator tries to outline a minimal implementation plan using only the artifact.
2. For each step it cannot ground in the artifact, it records the missing dependency as an `open_questions[]` item.
3. The report must separate:
   - `Grounded implementation steps`
   - `Steps that would require invented assumptions`

### Verdict Rules

- `PASS`: a minimal implementation outline can be produced with `0` invented assumptions
- `REWORK`: any implementation step would require invented assumptions
- `ABORT`: author explicitly stops after the report

### Accounting Rules

- `rounds_completed = 1`
- `max_rounds = 1`

## SDP Integration

This is an agent-discipline precondition before emitting the Plan gate. It is not tooling enforcement. The agent must run `@spec-interrogate` and respect a `REWORK` verdict before invoking `sdp phase plan`.

### Discovery -> Plan

```bash
@spec-interrogate docs/discovery/<slug>/validation.md --feature-id <F>
# writes:
#   .sdp/evidence/spec-interrogate.json
#   .sdp/reports/spec-interrogate.md

# only after PASS:
sdp phase plan --feature-id <F> --strict --evidence-path .sdp/evidence/plan.json
```

If the verdict is `REWORK`, do not call `sdp phase plan`. Resume Discovery using the unresolved questions from `report-path`.

### Plan -> Build

```bash
@spec-interrogate docs/plans/<feature>-design.md --feature-id <F> --mode adversarial
```

## Skip Rules

Skip only when one of these is true:

- trivial change or one-line bugfix
- direct Delivery task with no Discovery artifact
- explicit `--skip-interrogate` decision documented in beads

If skipped, the reason must be explicit. "Felt unnecessary" is not a valid reason.

## Invocation Contract

```bash
@spec-interrogate <artifact-path> \
  [--mode MODE] \
  [--questions N] \
  [--rounds M] \
  [--feature-id F] \
  [--evidence-path PATH] \
  [--report-path PATH]
```

Examples:

```bash
# default iterative hardening
@spec-interrogate docs/discovery/my-feature/validation.md --feature-id F042

# cheap first-pass sanity check
@spec-interrogate docs/plans/arch-decision.md --mode cold-read

# risky plan before implementation
@spec-interrogate docs/plans/auth-redesign.md --mode adversarial --report-path .sdp/reports/auth-redesign-interrogate.md
```

## Acceptance Boundaries

This skill works only with text artifacts such as `md`, `txt`, and schema docs.
Do not apply it directly to code. If the thing under review is executable code, use `@review`.
