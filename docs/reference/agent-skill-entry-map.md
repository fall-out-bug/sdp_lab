# Agent And Skill Entry Map

Status: friendly bridge

This page answers one practical question: "Where do I start?"

Use the five human intents for conversation. Use the manifest when you need the
exact inventory. Use repo-local CLI checks when you need proof that the local
adapter surface still matches the repo.

## The Short Version

| Human intent | Say this first | Then route to |
|---|---|---|
| Understand the repo | "What is this codebase?" | `@understand`, discovery skills, `scout` / `architect`-style agents |
| Build something | "Create or change this feature" | `@build`, feature/design/prototype skills, implementation agents |
| Fix something | "Something is broken" | `@fix`, bugfix/debug/hotfix skills, fixer-style agents |
| Review quality | "Is this good enough?" | `@review`, review/reality/readiness skills, reviewer/security/qa agents |
| Operate the system | "Keep it moving or running" | `@operate`, ship/ci-triage/planning skills, deployer/devops/sre agents |

The human-facing model is intentionally small. Operators should not have to
memorize every manifest entry before asking for useful work.

## What The Numbers Mean

`sdp.manifest.yaml` is the canonical inventory. At the time of this document it
declares:

- 5 human intents: understand, build, fix, review, operate
- 30 manifest skills under `skills:`
- 12 agent prompts under `agents:`

The five intents are the menu. The 30 skills are executable workflows and
legacy-compatible surfaces. The 12 agent prompts are role prompts for focused
work, review, and specialist judgment.

Do not update this document as the source of truth for counts. Update
`sdp.manifest.yaml`, then verify the generated and adapter surfaces.

## Canonical Paths

| Path | Purpose |
|---|---|
| `sdp.manifest.yaml` | Single inventory for generated skills, commands, agents, and harness metadata |
| `prompts/skills/` | Structured skill source, one directory per skill with `SKILL.md` |
| `.agents/skills/` | Flat skill source for OpenCode, Cursor, Kimi, and harnesses without plugin discovery |
| `prompts/agents/` | Canonical agent prompt source |

If these disagree, treat it as drift. Do not paper over the mismatch in docs.

## How Humans Should Choose

Start with the intent, not the implementation surface:

- choose `@understand` when the missing thing is context
- choose `@build` when the desired outcome is a new user-visible capability
- choose `@fix` when there is a bug, failing test, regression, or incident
- choose `@review` when the question is quality, readiness, or trust
- choose `@operate` when the work is CI, release, deploy, triage, or backlog flow

After that, the operator or agent can pick the specific manifest skill, command,
or agent prompt. This keeps the UX friendly while keeping the system precise.

## How Agents Should Verify

Before relying on a global `sdp` binary, prefer the repo-local CLI. A stale
global binary can make valid repo commands look missing.

Run these from the repo root:

```bash
./.sdp/bin/sdp manifest validate
./.sdp/bin/sdp manifest parity --check
```

If `./.sdp/bin/sdp` is not installed in this checkout, use the source fallback:

```bash
go run ./cmd/sdp manifest validate
go run ./cmd/sdp manifest parity --check
```

For documentation consistency checks, use:

```bash
go run ./cmd/sdp-protocol-check --format json
go run ./cmd/sdp-doc-sync --mode check --strict
```

Expected behavior: the manifest validates, parity reports no unexpected drift,
and documentation checks either pass or report concrete files to fix.

## Rule Of Thumb

Humans speak in intents. Skills execute workflows. Agents bring role focus. The
manifest decides what exists. Repo-local CLI checks decide whether the working
tree is actually consistent.
