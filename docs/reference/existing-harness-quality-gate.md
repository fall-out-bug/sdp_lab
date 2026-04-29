# Existing Harness Quality Gate

Status: product adoption reference
Updated: 2026-04-29
Related workstream: `00-152-03`

Use this path when a team already has an AI coding workflow and wants to know whether quality is degrading.

This is the right starting point for teams that already use Claude Code, OpenCode, Cursor, Codex, Superpowers, custom prompts, or an internal agent harness. Do not ask these teams to replace their harness before SDP proves value.

## Positioning

SDP can be introduced as a governance and evidence layer around an existing harness.

The first promise is not "use our whole workflow". The first promise is:

> Keep your current AI coding workflow, then use SDP to make its scope, evidence, findings, and readiness decisions reviewable.

This is narrower than Operator Mode and more concrete than a generic repository audit. It is the low-friction path toward `sdp-pr-gate` / ChangePassport.

## When To Use This Path

Use it when:

- the repo already has `AGENTS.md`, prompts, skills, or harness-specific setup
- developers have already been trained on a local process
- the organization worries about AI-assisted code quality, review burden, or process drift
- a full harness migration would create adoption risk
- the buyer wants a controlled pilot before changing team workflow

Do not use it as the first path when:

- the repo has no agent process and the team wants SDP to bootstrap one
- the team is ready for Beads-backed Operator Mode from the start
- the main need is only repo comprehension, where `scout`, `metrics`, `index`, and `spec` are enough

## Adoption Sequence

1. Inventory the existing harness.

   Identify where prompts, skills, commands, repo rules, and generated adapter files live. Treat these as installed process, not clutter to overwrite.

2. Run read-only repo inspection.

   Start with `scout`, `metrics`, `index`, and `spec`. The output should establish repo shape and risk signals before any generated files are written.

3. Define the gate boundary.

   Decide what the gate evaluates: a PR, a feature branch, a module, a service inside a monorepo, or a whole repo. For monorepos, prefer scoped service/module assessment over root-level generic assessment.

4. Require evidence-backed findings.

   Findings must cite inspected files, commands, checks, or source artifacts. Use `not_assessed` when the evidence is missing instead of filling gaps with model judgment.

5. Decide before migration.

   Only after the gate produces useful decisions should the team consider adopting more SDP workflow: bootstrap, Operator Mode, Beads, workstreams, or full delivery orchestration.

## Output Contract

A useful gate output should separate facts from judgment.

Minimum fields:

| Field | Requirement |
|---|---|
| Scope | What repo, branch, PR, service, or module was assessed |
| Inputs | PR diff, issue, workstream, evidence bundle, commands, or repo paths |
| Evidence | Files read, commands run, tests observed, CI data, review comments |
| Findings | Each finding has severity, source, evidence, and confidence |
| Verdict | `pass`, `warn`, `fail`, or `not_assessed` |
| Rubric | How the verdict was computed |
| Overrides | Human override reason and owner, when used |

Avoid single opaque health scores. If a score is shown, it must be derived from an explicit rubric and every component must be explainable.

## Skill And Tool Routing

Use task language, not internal skill names, in user-facing docs.

| User intent | SDP route |
|---|---|
| "What is in this repo?" | `sdp scout`, then `sdp index` |
| "Where are the process risks?" | `sdp metrics` |
| "What implicit contracts does the code contain?" | `sdp spec` |
| "Can my existing harness be trusted?" | existing-harness quality gate path |
| "Is this PR ready?" | `sdp-pr-gate` / ChangePassport direction |
| "Run a governed delivery loop" | Operator Mode |

Do not make evaluators choose `reality`, `review`, or other internal skills by guessing. Skills are implementation details unless the user is already contributing to SDP.

## Monorepo And JVM/Bazel Caveat

For JVM/Kotlin/Bazel monorepos, generic language detection is not enough. Assessment docs and tools should require an explicit scoped target when automatic detection is weak.

Good input:

```text
Assess service //payments/api in this monorepo.
Language: Kotlin.
Build: Bazel.
Do not infer Maven or Gradle conventions unless files prove them.
```

Bad output:

- applying Java-only idioms to Kotlin without saying so
- falling from `Mixed` to `Unknown` and still publishing strong architecture findings
- claiming files were missing without showing search/read evidence
- rating repository health without showing the formula

## Relationship To Product Surfaces

This path uses the stable Toolkit for read-only inspection and points toward the future `sdp-pr-gate` product surface.

It does not mean ChangePassport is GA today. It means the docs should make the migration path legible:

1. read-only inspection
2. evidence-backed quality gate over the existing harness
3. PR readiness packet / ChangePassport when schema and pilot gates are ready
4. Operator Mode only when the team wants SDP to own the delivery workflow
