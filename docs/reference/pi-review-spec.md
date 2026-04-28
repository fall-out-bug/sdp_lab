# Pi Review Gate

Status: draft
Feature: F161
Primary bead: sdplab-tffu
User surface: `sdp pi-review`
Skill surface: `sdp:pi-review`

## Purpose

`sdp pi-review` is an external second-opinion review gate for SDP-managed delivery work. It lets a Codex or Claude subagent finish a task to PR-quality, then asks `pi` to run independent model reviewers against the actual touched working tree. Findings are written back to beads, and the delivery loop keeps fixing and reviewing until the gate returns clean.

This is review-only. The reviewer never edits files, commits, rebases, or resolves findings.

## Product Decision

The review target is the local working tree or branch diff, not GitHub PR state. PR metadata may be attached when available, but it is not the authority for scope.

Reason: subagents often produce reviewable work before a PR exists, and hidden working-tree files are a larger quality risk than noisy PR comments. The command must review untracked files by default.

## User Surface

Recommended command:

```bash
sdp pi-review --scope auto --base main --feature F161 --create-beads --write-verdict
```

Shell alias:

```bash
sdp-pi-review --scope auto --base main --feature F161
```

Skill alias:

```text
sdp:pi-review
```

Required behavior:

- `--scope auto` reviews uncommitted working-tree changes first; if the tree is clean and `--base` is present, it reviews `<base>...HEAD`.
- `--scope working-tree` reviews staged, unstaged, and untracked files.
- `--scope branch` reviews `<base>...HEAD` and fails if `--base` is missing.
- `--feature` links the run to the SDP feature and bead context.
- `--create-beads` files actionable findings into beads.
- `--write-verdict` writes `.sdp/review_verdict.json`.

## Minimal Context Packet

SDP, not each model, owns context selection. The packet should include only:

- git status, current branch, base ref, and head SHA when available
- unified diff for reviewed changes
- full content for touched files when size allows
- file hashes for reviewed files
- project rules: `AGENTS.md`, `CLAUDE.md`, `.codex/AGENTS.md`, `.sdp/config.yml`, and matching prompt/skill rules when present
- linked beads for the feature or workstream
- deterministic test evidence captured by SDP before model calls

The packet must not include arbitrary repository dumps. If scope exceeds budget, SDP should degrade explicitly by file priority and record omitted files in telemetry.

## Test Evidence

Models do not run tests in the MVP. SDP runs the configured evidence command once and passes the result into the packet.

Default evidence strategy:

1. Use explicit `--test-command` when supplied.
2. Else use project config when available.
3. Else use a conservative detector such as `go test ./...`, `npm test`, or `pytest`, only when the repo makes that command obvious.
4. Else record `test_evidence.status = "skipped"` with a reason.

Reviewers may distrust or challenge the evidence, but they must not invent test results.

## Model Policy

MVP reviewers:

- GLM through `pi` subscription-backed provider configuration
- Kimi through `pi` subscription-backed provider configuration

Fallback:

- OpenRouter model configured in `pi` environment

SDP does not own provider keys. Keys, subscriptions, provider names, and model IDs live in the `pi` runtime environment. SDP passes a context packet and requested reviewer slots to the local `pi` binary.

Recommended slots:

| Slot | Purpose | Required |
|---|---|---|
| `glm` | broad correctness and maintainability review | yes |
| `kimi` | adversarial code review and missed-edge search | yes |
| `openrouter-fallback` | only if GLM or Kimi fails or times out | no |
| `synthesizer` | normalize findings into SDP verdict | yes |

## Finding Contract

Findings use SDP review priorities:

- `P0`: data loss, security break, build-breaking regression, unsafe automation
- `P1`: likely user-visible bug, missing required behavior, serious maintainability risk
- `P2`: meaningful but non-blocking defect or missing test
- `P3`: polish, clarity, or future improvement

`P0` and `P1` block a clean verdict. `P2` and `P3` are tracked unless the synthesizer explicitly escalates them as release-blocking.

Each actionable finding must include:

- priority
- title
- affected file path
- line range when available
- reviewer source model
- concise rationale
- suggested fix direction
- dedupe key stable across line shifts where possible

## Verdict Contract

`sdp pi-review` writes raw telemetry to `.sdp/runs/pi-review/<run_id>/` and the compact gate verdict to `.sdp/review_verdict.json`.

The compact verdict must remain compatible with `schema/review-verdict.schema.json`. For compatibility with existing SDP gates, it still includes the seven reviewer role buckets: `qa`, `security`, `devops`, `sre`, `techlead`, `docs`, and `promptops`. Pi-review populates those buckets from synthesized model findings rather than spawning seven separate role agents.

Additional pi-review fields are optional and advisory:

- `reviewer_runtime`: `pi`
- `context_packet`: hashes and reviewed file list
- `model_panel`: model/provider status and raw artifact references
- `raw_artifacts`: files containing raw model output
- `findings_detail`: structured details for findings that were also filed to beads

## Telemetry

Each run writes:

- `.sdp/runs/pi-review/<run_id>/context.json`
- `.sdp/runs/pi-review/<run_id>/context.diff`
- `.sdp/runs/pi-review/<run_id>/test-evidence.json`
- `.sdp/runs/pi-review/<run_id>/models/<slot>.json`
- `.sdp/runs/pi-review/<run_id>/synthesis.json`
- `.sdp/runs/pi-review/<run_id>/run.json`

`run.json` validates against `schema/pi-review-run.schema.json`.

Telemetry must record omitted files, model failures, fallback use, token/cost metadata when available, and artifact hashes. It must not record provider secrets.

## Beads Integration

When `--create-beads` is enabled:

- every `P0` and `P1` finding creates or updates a blocking bead
- `P2` and `P3` create tracking beads only when not already represented
- dedupe key should include feature, normalized file path, finding category, and stable symbol or snippet hash
- findings use labels: `pi-review`, `review-finding`, `F{NNN}`, `round-{N}`, and priority label
- clean verdict requires no open `P0` or `P1` pi-review findings for the reviewed feature and round

## Delivery Loop

The delivery loop should run:

1. subagent implements or fixes assigned beads in a worktree
2. SDP captures deterministic evidence
3. `sdp pi-review` runs model panel review through local `pi`
4. SDP files findings into beads and writes verdict
5. subagent fixes blocking findings
6. repeat until verdict is `APPROVED`

The loop should stop and escalate when:

- the same blocking finding survives two fix attempts
- model outputs conflict on a `P0` or `P1` issue
- provider failures leave fewer than two successful reviewer perspectives
- the context packet omits files required to judge a blocking finding

## Non-goals

- replacing `@review`
- giving models direct write access
- requiring a GitHub PR before review
- making SDP manage provider subscriptions
- hand-rolling provider APIs inside SDP when `pi` already owns model runtime
