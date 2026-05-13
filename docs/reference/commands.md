# SDP Command Map

Status: honest map of the current `sdp_lab` command surface.

This page is a friendly orientation map, not a promise that every tool is
product-ready. The source of truth for the top-level CLI is:

```bash
go run ./cmd/sdp --help
```

Use this page to answer: "Which command family am I looking at, and should I
use it for normal work?"

## Quick Decision

| Need | Start here | Notes |
|---|---|---|
| Adopt SDP in a repo | `sdp init`, `sdp bootstrap`, `sdp doctor adapters` | Toolkit surface for real users and harness setup. |
| Understand an unknown repo | `sdp scout`, `sdp metrics`, `sdp index build`, `sdp architect analyze` | Repo analysis and indexing. |
| Work an SDP feature in `sdp_lab` | `/feature`, `/design`, `/build`, `/review`, `/oneshot` | Harness slash commands and skills, not top-level `sdp` subcommands. |
| Check quality/backlog/doc consistency | `sdp quality`, `sdp doctor backlog`, `sdp phase *`, `sdp-protocol-check`, `sdp-doc-sync` | `sdp quality` is an sdp_lab-local advisory quality-axis report; repo-maintenance surfaces are not broad downstream promises. |
| Find ready Beads work | `sdp-ready`, `bd ready`, `bd show <id>` | Beads is the task tracker authority. |
| Run Go quality gates | `./scripts/run_go_quality_gates.sh` | No Python gates are required by this repo's normal Go flow. |
| Run model/code review tooling | `sdp-pi-review`, `/review`, `/codereview` | Review evidence, not automatic merge approval. |

## Main `sdp` CLI

Run as:

```bash
go run ./cmd/sdp <command> ...
```

or as `sdp <command> ...` after installing/building it onto your `PATH`.

### User-facing toolkit commands

These are the clearest public-facing commands today.

| Command | What it is for |
|---|---|
| `sdp init [--harness <list>] [--target <dir>]` | Install/update SDP harness adapters in a target project. |
| `sdp bootstrap [--dry-run] [--mode greenfield|brownfield] <repo-path>` | Preview or apply SDP conventions and optional Beads support. Use `--dry-run` for first-run safety. |
| `sdp bootstrap status <repo-path>` | Inspect bootstrap state. |
| `sdp doctor adapters [--manifest <path>] [--strict]` | Validate generated harness adapters. |
| `sdp manifest validate` | Validate the SDP manifest. |
| `sdp manifest parity [--write]` | Check or update harness parity material. |
| `sdp generate-adapters [--check|--write|--diff]` | Generate harness adapter files from the manifest. |
| `sdp skills augment --stack <config.json>` | Add skill recommendations from a stack config. |
| `sdp skills update [--project-root DIR]` | Update project skills. |

### Repo analysis commands

These help inspect a codebase. They are useful for onboarding, discovery, and
operator analysis.

| Command | What it is for |
|---|---|
| `sdp scout <repo-path>` | Quick repo map in text, JSON, or card format. |
| `sdp metrics <repo-path>` | Git-derived process and activity metrics. |
| `sdp quality [--full]` | sdp_lab-local quality-axis report. Default prints F168 states only; `--full` runs coverage and test/code ratio checks. |
| `sdp spec <repo-path>` | Extract API/rules/invariants/SLA-oriented spec signals. |
| `sdp architect analyze <repo-path>` | Tiered architecture analysis. |
| `sdp architect c4 <repo-path>` | Generate C4-oriented architecture output. |
| `sdp index build <repo-path>` | Build the local code index. |
| `sdp index refresh <repo-path>` | Refresh the index. |
| `sdp index query <repo-path> <query>` | Query indexed repo content. |
| `sdp index find <repo-path> <term>` | Find indexed terms. |
| `sdp index deps <repo-path> <module>` | Explore dependency relationships. |
| `sdp index stats <repo-path>` | Show index stats. |
| `sdp index manifest <repo-path>` | Emit index manifest data. |
| `sdp index rank <repo-path>` | Rank indexed files/modules. |

### Operator and delivery commands

These are active `sdp` subcommands, but several are still lab/operator
machinery rather than polished public UX.

| Command | What it is for |
|---|---|
| `sdp intent "description"` | Create an intake card from raw intent. |
| `sdp discover "raw idea"` | Run the Stage 0 discovery pipeline. |
| `sdp build "<idea>"` | Run local build orchestration for an idea. |
| `sdp card <...>` | Card lifecycle operations: create, clarify, ready, execute, deliver, resume, feedback, and related flows. |
| `sdp board <build|show>` | Build or show the board view. |
| `sdp status <card-id>` | Show card status and phase. This is card-level, not feature-level. |
| `sdp stuck` | Show stuck or long-running cards. |
| `sdp clarify <card-id>` | Run clarification manually. |
| `sdp plan <card-id>` | Show the plan for a card. |
| `sdp approve-plan <card-id>` | Approve a pending plan. |
| `sdp eval <card-id>` | Run build evaluation manually. |
| `sdp why <card-id>` | Explain why a card is blocked. |
| `sdp next [--limit N]` | Show next actionable items. |
| `sdp missing [project-id]` | Show items lacking evidence. |
| `sdp approve <card-id>` | Resolve a human gate. |
| `sdp trace <card-id>` | Show the feature/card trace. |
| `sdp dispatch card` | Dispatch one card. |
| `sdp dispatch next` | Dispatch next available work. |
| `sdp result ingest` | Ingest execution results. |
| `sdp orchestrate once` | Run one orchestration step. |
| `sdp orchestrate loop` | Run the orchestration loop. |

### Gates, deploy, and runtime commands

| Command | What it is for |
|---|---|
| `sdp doctor control` | Control-plane health diagnostics. |
| `sdp doctor backlog` | Backlog/workstream hygiene diagnostics. |
| `sdp phase plan` | Validate or emit plan-phase evidence. |
| `sdp phase review` | Validate or emit review-phase evidence. |
| `sdp phase eval` | Validate or emit eval-phase evidence. |
| `sdp deploy staging [project-root]` | Staging deploy path. Use only when the workflow explicitly calls for deploy. |
| `sdp deploy prod <staging-image-tag> [project-root]` | Production deploy path. Requires normal release approval. |
| `sdp deploy rollback <previous-tag> [project-root]` | Roll back to a previous image tag. |
| `sdp reset --feature F042` | Reset a feature checkpoint. |
| `sdp coverage-scan` | Go coverage scan/report helper. |
| `sdp rules update <repo-path>` | Update rules from evidence/manifest inputs. |
| `sdp telemetry <...>` | Local telemetry/span/daemon commands. |
| `sdp tower [--addr <host:port>]` | Start the local control tower UI/server. |
| `sdp attention` | Attention/triage helper. |

## Standalone Go Binaries

The repo also has many `cmd/sdp-*` binaries. Run them with:

```bash
go run ./cmd/<binary> --help
```

Use the standalone binary when repo docs or a skill name it directly. Do not
assume a standalone binary is part of the public `sdp` CLI.

### Common repo-maintenance binaries

| Binary | Use |
|---|---|
| `sdp-ready` | Show ready SDP work from Beads/workstream mapping. Supports JSON/status-view output and action instructions. |
| `sdp-orchestrate` | Feature-level orchestrator. Use `--feature FXXX --status` or `--next-action` for "continue FXXX" flows. |
| `sdp-protocol-check` | Validate roadmap/workstream/protocol hygiene. Supports strict and strict-Beads modes. |
| `sdp-doc-sync` | Check/fix doc consistency or update changelog text. |
| `sdp-ws-verdict-validate` | Validate workstream verdict artifacts. |
| `sdp-session-audit` | Audit session outputs. |
| `sdp-pi-review` | Run model review and optionally write `.sdp/review_verdict.json`. |
| `sdp-strataudit` | Structure/strategy audit tool; may be build-tag gated in local builds. |

### Lab and integration binaries

These are real source directories, but they are primarily internal, experimental,
or integration-specific:

`gt-adapter`, `sdp-a2a`, `sdp-bd-suggest`, `sdp-beads-bridge`,
`sdp-cascade-replay`, `sdp-ci-loop`, `sdp-confidence-replay`, `sdp-control`,
`sdp-decompose-bench`, `sdp-dispatch`, `sdp-eval`, `sdp-evidence`,
`sdp-export`, `sdp-ft-baseline`, `sdp-ft-dataset`, `sdp-ft-run`,
`sdp-ft-validate`, `sdp-gh-findings-sync`, `sdp-guard`, `sdp-harness`,
`sdp-healthcheck`, `sdp-llm-gateway`, `sdp-mcp`, `sdp-microfirst-bench`,
`sdp-omc-guard`, `sdp-orchestrate-daemon`, `sdp-pi-eval`, and `sdp-up`.

Some of these are protected by build tags or environment assumptions. If
`go run ./cmd/<name> --help` fails with build constraints, treat that as
`not available in this local build`, not as a product failure.

## Harness Slash Commands

Slash commands live in `prompts/commands/` and are mirrored into
`.claude/commands/`. They are harness workflows, not `sdp` CLI subcommands.

Use them when operating inside Claude Code or another harness that loads these
command files.

| Slash command | Use |
|---|---|
| `/idea` | Clarify a new feature idea. |
| `/vision` | Strategic product planning. |
| `/feature` | Turn an idea into feature/workstream structure. |
| `/design` | Produce design/workstream planning. |
| `/build` | Execute one workstream with guard/TDD workflow. |
| `/oneshot` | Run autonomous feature execution via `sdp-orchestrate`. |
| `/review` | Feature-level multi-agent review. |
| `/codereview` | Code review workflow. |
| `/verify-workstream` | Validate workstream docs against code reality. |
| `/reality` | Deeper codebase reality analysis. |
| `/reality-check` | Quick docs-vs-code validation. |
| `/protocol-consistency` | Audit consistency across docs, CLI, and CI workflows. |
| `/debug` | Systematic debugging. |
| `/issue` | Classify and route a bug/problem. |
| `/bugfix` | P1/P2 quality bug-fix workflow. |
| `/hotfix` | P0 emergency hotfix workflow. |
| `/ci-triage` | Investigate failing GitHub Actions. |
| `/test` | TDD cycle helper. |
| `/prototype` | Rapid prototype workflow. |
| `/beads` | Beads task tracker helper. |
| `/ship` | Release handoff/deployment orchestration. |
| `/deploy` | Older deployment command; prefer `/ship` when repo guidance says so. |
| `/deliver` | Autonomous feature delivery wrapper. |
| `/prd` | PRD generation/maintenance; current command points users toward `/vision`. |
| `/submit-to-swarm` | Submit a task to the SDP swarm intake gateway. |

## Skills

Skills live in two formats while migration is in progress:

- structured: `prompts/skills/<name>/SKILL.md`
- flat: `.agents/skills/<name>.md`

Commands and skills are related but not identical. A slash command is the
harness entrypoint; a skill is the workflow body or reusable procedure behind
that entrypoint. When both exist, follow the command first, then the invoked
skill file.

Common skills include `build`, `review`, `delivery-loop`, `go-modern`,
`protocol-consistency`, `spec-interrogate`, `strataudit`, `ux`, `beads`, and
`verify-workstream`.

## Quality Gates

This is a Go repo. The normal code gate is:

```bash
./scripts/run_go_quality_gates.sh
```

If Docker is unavailable and the workflow allows host execution:

```bash
SDP_GO_QUALITY_MODE=host ./scripts/run_go_quality_gates.sh
```

Documentation and protocol checks commonly used in this repo:

```bash
go run ./cmd/sdp-protocol-check --format json
go run ./cmd/sdp-doc-sync --mode check --strict
```

Do not invent Python gates for this repo. There is no default requirement here
for `mypy --strict`, `ruff`, Python coverage thresholds, or Python file-size
rules.

## Naming Traps

- `sdp status <card-id>` is card-level. For feature-level status, use
  `go run ./cmd/sdp-orchestrate --feature FXXX --status`.
- `/build` is a harness command. `sdp build "<idea>"` is a top-level CLI command
  with different semantics.
- `/review` and `sdp-pi-review` produce review evidence. They do not replace
  deterministic gates, CI status, or human approval where required.
- `bd ready` and `bd show <id>` remain authoritative for Beads state. Docs and
  mappings are helpful projections, not the live task tracker.
- `sdp/` is an optional local checkout of the public distilled repo. Normal work
  happens in `sdp_lab`.
