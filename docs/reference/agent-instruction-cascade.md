# Agent Instruction Cascade

Status: canonical reference

This document defines how SDP splits agent instructions across repo-wide policy,
module-local contracts, skills, commands, and harness adapters.

## Problem

The root `AGENTS.md` has accumulated too many responsibilities:

- repo constitution and behavior policy
- repo topology and publish boundary
- Beads workflow
- delivery flow
- harness loading rules
- skill and agent inventory
- module-specific runtime assumptions

That makes agents slower and less reliable. A model entering `internal/evidence`
does not need the full release workflow in context before it can understand the
evidence contract. A model executing `@build` should not infer skill behavior from
root prose when a skill file owns the workflow.

## Cascade Model

Load instructions from broad to narrow:

1. Root `AGENTS.md`
2. Nearest module-local `AGENTS.md`
3. Invoked skill file
4. Invoked command or harness adapter

The narrower layer may add detail, but it must not weaken broader safety,
repo-boundary, evidence, or quality rules.

## Layer Ownership

| Layer | Owns | Must Not Own |
|---|---|---|
| Root `AGENTS.md` | Repo-wide safety, source-of-truth routing, branch/publish boundaries, cold-start checklist | Package APIs, feature-specific workflows, long rationale |
| Module `AGENTS.md` | Local API contract, invariants, dependencies, runtime assumptions, local gates | Global policy, skill routing, unrelated package rules |
| Skill `SKILL.md` | Executable workflow, preconditions, steps, outputs, stop conditions | Stable repo topology, package API docs |
| Command prompt | Argument handling and command-specific execution contract | New policy not present in root or skill |
| Harness adapter | Harness-specific loading quirks and fallback behavior | Canonical workflow semantics |

## Conflict Rules

- Global safety wins over every local instruction.
- Module-local `AGENTS.md` wins over root only for facts inside that subtree.
- Skill workflow wins over root prose for how to execute that skill.
- Reference docs explain rationale; they do not override executable instructions.
- If two active instructions conflict, stop and report the conflict instead of guessing.

## Module AGENTS.md Standard

Create a module-local `AGENTS.md` when a subtree has one of these:

- Stable API or runtime contract.
- Special dependency boundaries.
- Security, evidence, policy, or model-provider assumptions.
- Local gates not obvious from repo-wide quality gates.
- Extraction or publish semantics different from the repo default.

Keep module files short. Target 30-60 lines.

Required sections:

```markdown
# <module> — Agent Contract

## Scope

What this subtree owns.

## Contract

Primary inputs, outputs, and stable APIs.

## Dependencies

Allowed and risky dependencies.

## Runtime Assumptions

Env vars, filesystem, network, provider, or security assumptions.

## Local Rules

Rules that apply only inside this subtree.
```

## What Moves Out Of Root

Move content out of root `AGENTS.md` when it applies to only one module:

- Go package API contracts -> nearest `internal/<pkg>/AGENTS.md`
- harness quirks -> `.codex/AGENTS.md`, `.cursor/`, `.pi/`, or harness docs
- skill execution steps -> `prompts/skills/<name>/SKILL.md`
- long rationale -> `docs/reference/*.md`
- historical context -> `docs/archive/` or dated plans

Keep in root only if every agent in every subtree must obey it.

## Current Target Split

| Area | Instruction Owner |
|---|---|
| Repo identity, branch policy, publish boundary | root `AGENTS.md` |
| Canonical read order | `docs/reference/project-map.md` |
| Documentation tree rules | `docs/AGENTS.md` |
| Stable reference doc rules | `docs/reference/AGENTS.md` |
| Agent/skill inventory | `sdp.manifest.yaml` plus `docs/reference/agent-catalog.md` and `docs/reference/skills.md` |
| Runtime skill adapter rules | `.agents/skills/AGENTS.md` |
| Agent role prompt rules | `prompts/agents/AGENTS.md` |
| Skill workflow prompt rules | `prompts/skills/AGENTS.md` |
| Harness quirks | `docs/reference/harness-integration.md` and harness-local entrypoints |
| Go command entrypoints | `cmd/AGENTS.md` |
| Go internal packages | `internal/AGENTS.md` plus package-local `internal/<pkg>/AGENTS.md` |
| `internal/context` contract | `internal/context/AGENTS.md` |
| `internal/evidence` contract | `internal/evidence/AGENTS.md` |
| `internal/eval` contract | `internal/eval/AGENTS.md` |
| `internal/modelgateway` contract | `internal/modelgateway/AGENTS.md` |
| `internal/policy` contract | `internal/policy/AGENTS.md` |

## Migration Rule

Do not rewrite the whole instruction surface in one pass.

For each migration PR:

- pick one section or one module family
- move only facts that have a clear narrower owner
- replace root content with a short pointer
- preserve behavior unless the workstream explicitly changes it
- run docs consistency checks when references change

## Completion Discipline

A documentation or prompt change is not finished when files are edited. The agent
must either commit the scoped change or explicitly report why committing would be
unsafe.

Before committing:

- stage only files owned by the current task
- leave pre-existing unrelated dirty files untouched
- never use `git add .` while unrelated dirty files exist
- run the cheapest relevant verification
- create a scoped commit before reporting completion
- push the scoped commit, unless doing so would also push pre-existing unrelated
  commits; in that case report the exact blocker
- mention any repo-wide checks that fail for pre-existing reasons

Never treat "I changed files" as a completed work unit.

## Definition Of Done For This Migration

The cascade migration is complete only when all of these are true:

- root `AGENTS.md` contains repo-wide policy and pointers only
- each major subtree has a short module-local `AGENTS.md`
- package-specific runtime assumptions live below the package they affect
- skill behavior is owned by skill files, not root prose
- runtime skill adapters do not diverge from canonical skill docs
- `sdp.manifest.yaml`, `docs/reference/skills.md`, and runtime skill files agree
  on the active skill surface
- docs/reference read order points to every active instruction SOT
- scoped documentation or prompt changes are committed and pushed, or the agent
  reports exactly why committing would be unsafe

Open known gaps after this first cascade pass:

- root `AGENTS.md` still needs reduction by moving long workflow sections into
  reference docs or skill files
- active skill inventory still needs reconciliation between F125 intent docs,
  `sdp.manifest.yaml`, `prompts/skills`, and `.agents/skills`
- agent inventory still needs reconciliation between the five-intent model and
  the older role prompt set
