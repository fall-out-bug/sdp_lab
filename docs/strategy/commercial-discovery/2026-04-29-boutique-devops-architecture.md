# 2026-04-29 Discovery Interview: DevOps Architecture Lead

Status: source evidence for F152-03
Source: [Fireflies transcript `01KQCW4BZS014BHVR7NB9J44X3`](https://app.fireflies.ai/view/01KQCW4BZS014BHVR7NB9J44X3)
Workstream: `00-152-03`
Bead: `sdplab-qnyr.3`

## Context

The participant tested SDP against a key internal monorepo and reported feedback from the perspective of a DevOps architecture lead responsible for organization-wide developer infrastructure.

The test environment is commercially useful because it matches the hard version of the target problem:

- team-level and cross-team AI development process is being piloted
- existing harness/process work already exists
- repo shape is not a simple open-source demo project
- buyer concern is controlled adoption, not raw agent speed

## Repo And Tooling Setup

- Monorepo with Kotlin and Java services.
- Bazel build, not Maven or Gradle.
- Internal model gateway through LiteLLM.
- Tested with internal Qwen-family models and Minimax 2.5-class model access.
- Evaluation started from "can SDP assess code quality?", then discovered SDP is broader than assessment.

## What Failed

### Project Detection

SDP did not handle the target repo shape cleanly. JVM/Bazel/Kotlin was treated too much like generic Java or fell through `Mixed` / `Unknown` style heuristics. This caused weak downstream findings because language and build assumptions were wrong.

### Evidence Depth

Some runs produced surface metrics without enough file reads. A quality report that mainly counts files/classes/lines is not credible for a CTO unless it shows what it inspected and where the claims came from.

### Score Trust

Opaque health scores were not trusted. The participant explicitly objected to a score where the formula is unclear and reads like unsupported LLM judgment rather than a defensible assessment.

### Finding Consistency

Some findings appeared internally inconsistent, for example treating test volume as both healthy coverage and a performance risk without a clear rubric separating the two claims.

### Skill Routing

The evaluator selected `reality` because it looked closest to the need. That is a product UX issue: users should not need to understand the internal skill catalog to choose between repository mapping, quality assessment, feature review, and full delivery adoption.

## Strongest Product Signal

The strongest signal is not "support Bazel better", though that is real. The stronger signal is:

> Buyers already have agent workflows. They want SDP to validate and control those workflows before they consider replacing them.

The participant's core question was whether SDP can be applied to an existing harness as a quality gate. Requiring a full SDP harness migration first would create too much adoption risk.

## Buyer Language

Use this language in product docs:

- "Apply SDP as a quality gate over your existing harness."
- "Assess whether AI-assisted delivery is degrading quality."
- "Show evidence, scope, and decision records before asking teams to change their workflow."
- "Start with a read-only assessment path; migrate only where the evidence justifies it."

Avoid this as the first message:

- "Replace your current harness with SDP."
- "Run the whole Operator Mode loop before you see value."
- "Trust a health score without traceable evidence."

## Pricing And Pilot Implications

This interview supports the `sdp-pr-gate` / ChangePassport wedge more than an Operator Mode-first sale.

Likely willingness-to-pay anchor:

- reduced review and governance burden for AI-assisted PRs
- reduced risk of quality degradation across teams
- lower cost of standardizing agent practices across repos

Open commercial questions for the remaining F152 interviews:

- How many AI-assisted PRs per week create enough pain to pay?
- Who owns the budget: CTO, platform engineering, DevOps, or engineering managers?
- Is the first paid object a PR gate, a repo assessment, or a team process audit?
- What level of false-block rate is tolerable before developers bypass the gate?

## Product Follow-Ups

These are not F152 implementation scope, but they should feed the roadmap:

- Define a first-class "existing harness quality gate" path in docs.
- Make evidence-backed assessment output explicit: `pass`, `warn`, `fail`, `not_assessed`.
- Replace opaque health score language with a rubric and evidence links.
- Add JVM/Kotlin/Bazel detection to assessment/scout paths.
- Add a skill/task decision tree so users do not route themselves into `reality` by guessing.

## F152 Signal

This counts as one target-ICP discovery input, but it does not complete F152-03. It validates a major objection: customers may not accept a harness replacement pitch, but they are interested in an SDP gate that sits on top of what they already use.
