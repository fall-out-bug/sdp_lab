# SDP Quickstart

Get SDP installed in a repo and run the first useful checks.

Audience: CTOs, architects, and developers evaluating SDP as a structured AI PDLC/SDLC harness layer.

In this guide, a **harness** means the coding-agent runtime your team uses to
interact with models and edit code, for example Claude Code, OpenCode, Codex,
Cursor, or Pi. SDP installs repo-local adapters around those runtimes; it does
not replace them.

If you are not sure whether you need Toolkit evaluation, repo contribution, or
Operator Mode, start with [START_HERE.md](START_HERE.md).

## What You Are Installing

You are installing **SDP Toolkit** — the installable developer surface. It is the `sdp` CLI, installed into your repo.

SDP Toolkit installs a repo-local harness surface:

- `sdp.manifest.yaml` — source of truth for skills, commands, agents, and harness adapters
- generated adapter files for Claude Code, OpenCode, Codex, Cursor, and Pi
- `sdp.lock` — installed SDP version pin
- optional `.sdp/` outputs from scout, metrics, index, specs, evidence, and later operator runs

It does not ask for model API keys during install. Model/provider setup belongs to the harness you use.

The `sdp_lab` repo is the research workspace where SDP is built. You do not need to clone it unless you are contributing to the platform. ChangePassport (`sdp-pr-gate`) and Enterprise Delivery Governance are separate product surfaces — they are product directions, not part of this install.

## Prerequisites

| Requirement | Minimum | Why |
|---|---:|---|
| Git | 2.30+ | Clone/install and repo analysis |
| Go | 1.26+ | Build the `sdp` CLI from source |
| macOS, Linux, or WSL | - | Native Windows is not supported in v1 |
| One AI harness | optional | Claude Code, OpenCode, Codex, Cursor, or Pi if you want generated commands |

## Install

Run this from the root of the repo where you want SDP installed:

```bash
curl -fsSL https://raw.githubusercontent.com/fall-out-bug/sdp_lab/main/scripts/install.sh | bash
```

The installer:

1. clones the `sdp_lab` source repo to get the canonical manifest and prompts
2. ignores any existing `sdp` on `PATH` unless `SDP_TRUST_PATH_SDP=1`
3. builds `cmd/sdp` with the `sqlite_fts5` tag
4. runs `init --harness auto` through the chosen installer binary
5. writes `sdp.manifest.yaml`, `prompts/`, generated harness adapter dirs, `.sdp/generated/`, and `sdp.lock`
6. verifies the repo-local CLI and leaves it at `./.sdp/bin/sdp`

Environment overrides:

```bash
SDP_HARNESS=claude-code,opencode \
SDP_TARGET=/path/to/repo \
curl -fsSL https://raw.githubusercontent.com/fall-out-bug/sdp_lab/main/scripts/install.sh | bash
export PATH="/path/to/repo/.sdp/bin:$PATH"
```

If you are already inside this repo and want the local binary:

```bash
SDP_SOURCE_DIR="$PWD" SDP_TARGET=/path/to/your/repo bash scripts/install.sh
```

## Verify

From the target repo:

```bash
./.sdp/bin/sdp manifest validate
./.sdp/bin/sdp doctor adapters
export PATH="$PWD/.sdp/bin:$PATH"
command -v sdp
```

Expected result:

- manifest validation exits 0
- adapter doctor reports 0 drifts
- manifest output reports the SDP inventory, currently 30 skills, 25 commands, and 12 agents
- `sdp.lock` exists
- `.sdp/bin/sdp` exists
- one or more harness dirs exist: `.claude/`, `.opencode/`, `.codex/`, `.cursor/`, `.pi/`
- after the `export`, `command -v sdp` points at the target repo's `.sdp/bin/sdp`

If `sdp manifest validate` or `sdp scout` says the command is missing, you are running an older global `sdp`. Use `./.sdp/bin/sdp ...` or move `$PWD/.sdp/bin` before the global binary in `PATH`.

## First Useful Run

Run low-risk toolkit commands first. `scout`, `metrics`, and `spec` only inspect the repo; `index build` writes a local `.sdp/index.db` cache.

```bash
./.sdp/bin/sdp scout --format text .
./.sdp/bin/sdp metrics --format markdown .
./.sdp/bin/sdp index build --format text .
./.sdp/bin/sdp spec --format text .
```

After the first index build, use the cache for follow-up questions:

```bash
./.sdp/bin/sdp index query . "auth flow"
./.sdp/bin/sdp index find . Handler
./.sdp/bin/sdp index deps . ./internal/api
./.sdp/bin/sdp index stats .
```

Then preview the delivery-planning surface:

```bash
./.sdp/bin/sdp build "Add a small feature with tests" --dry-run --format text
```

For brownfield agent setup, preview generated artifacts before writing:

```bash
./.sdp/bin/sdp bootstrap --dry-run --mode brownfield .
```

## From Install to Harness Commands

The installed CLI (`./.sdp/bin/sdp`) is the install/support surface.
To execute SDP workflows, jump to your harness-specific command surface:

The command forms below are harness commands and are not part of the `sdp` CLI.
The `@`/`/` prefix is the harness entrypoint marker:

- `@build` (OpenCode/Cursor): harness command syntax.
- `/build` (Claude Code): slash-command syntax.

| Harness | Primary form | Runtime status |
|---|---|---|
| Claude Code | `claude -p "/build 00-XXX-YY"` | Stable primary |
| OpenCode | `opencode run --dir "$PWD" --agent implementer "@build 00-XXX-YY"` | Experimental; requires `--agent implementer` |
| Cursor | `agent -p "@build 00-XXX-YY"` | Secondary validator only; primary dispatch untested |

```bash
# Claude Code
claude -p "/build 00-XXX-YY"

# OpenCode
opencode run --dir "$PWD" --agent implementer "@build 00-XXX-YY"

# Cursor
agent -p "@build 00-XXX-YY"
```

Use real IDs from your workstream/feature context. `00-XXX-YY` is a placeholder
for a real workstream ID.

Inside `sdp_lab` operator mode, find ready workstream IDs with:

```bash
bd ready
```

Cursor is a **secondary validator only** and remains **untested** for primary SDP
dispatch.

If you do not have an internal workstream yet (external user flow), use local
delivery first and skip harness dispatch:

```bash
./.sdp/bin/sdp build "what you want to change" --dry-run --format text
```

Then convert that local plan into `docs/workstreams/...` only when you are ready for
operator-mode execution.

**OpenCode warning:** always use non-interactive `--agent implementer`.
Without it, `opencode run ...` can exit successfully without applying edits.

## Choose A Path

| Path | Use when | Start with |
|---|---|---|
| **Toolkit evaluation** | You want to inspect an existing repo and recover useful context. | `scout`, `metrics`, `index build`, `spec`, `bootstrap --dry-run` |
| **Local delivery** | You want a lightweight idea-to-change loop in one repo. | `sdp build --dry-run`, then harness-specific commands |
| **Operator mode** | You need queue-backed delivery, explicit ownership, PR gates, and QA/UAT. | [reference/canonical-happy-path.md](reference/canonical-happy-path.md) |
| **MCP integration** | You want an AI harness to call SDP tools directly. | [reference/installation.md](reference/installation.md) |

Command map: [reference/commands.md](reference/commands.md)

Skill and agent map: [reference/agent-skill-entry-map.md](reference/agent-skill-entry-map.md)

## What Works Today

**SDP Toolkit (stable, ships in formula):**

- static multi-harness adapter files: 30 skills, 25 commands, 12 agents rendered where the harness has an adapter surface
- first-run toolkit commands: `scout`, `metrics`, `index build`, `spec`, `bootstrap --dry-run`
- install/support commands: `init`, `manifest`, `generate-adapters`, `doctor`

Static adapter parity is not runtime dispatch readiness. Claude Code is the
stable primary harness today; OpenCode, Cursor, Codex, and Pi have explicit
runtime limits. See [reference/harness-parity-matrix.md](reference/harness-parity-matrix.md).

**Operator Mode (default Toolkit happy path):**

- Beads-backed operator workflow in `sdp_lab`
- evidence/schema/protocol checks
- StratAudit reports

**Operator tooling (included in the release build, not the first-run promise):**

- `sdp-orchestrate --feature` for feature/workstream operator runs
- `sdp-ci-loop`, `sdp-guard`, `sdp-doc-sync`, `sdp-ready`
- `manifest parity`, `generate-adapters`, `doctor adapters`

`sdp orchestrate once|loop` is the top-level CLI result-processing loop. It is not the same surface as the standalone `sdp-orchestrate --feature ...` operator driver.

**Lab / research (not in formula):**

- strict `agentloop` + `sdp-harness` primary delivery runtime
- model gateway, cascades, MicroFirst inference, telemetry daemon
- K8s/swarm/control tower paths

**Product direction (not yet shipped):**

- ChangePassport (`sdp-pr-gate`) — separate merge-readiness product surface
- Enterprise Delivery Governance — enterprise governed delivery control plane

Canonical map: [reference/product-surface.md](reference/product-surface.md)

## Configure Harnesses

The manifest supports five harness names:

```bash
./.sdp/bin/sdp init --harness all
./.sdp/bin/sdp init --harness auto
./.sdp/bin/sdp init --harness claude-code,opencode
./.sdp/bin/sdp init --harness cursor,codex,pi --target /path/to/repo
```

`auto` detects existing harness directories. If none exist, it installs all five.

Generated adapters are owned by the manifest. Do not edit generated harness files
directly. Change `sdp.manifest.yaml`, then regenerate:

```bash
./.sdp/bin/sdp generate-adapters --write
./.sdp/bin/sdp init --update
./.sdp/bin/sdp doctor adapters
```

`generate-adapters --write` refreshes `.sdp/generated/`. `init --update`
refreshes the live harness directories from the manifest without overwriting an
existing `sdp.manifest.yaml`.

Safe adapter update sequence:

```bash
./.sdp/bin/sdp manifest validate
./.sdp/bin/sdp generate-adapters --check
./.sdp/bin/sdp init --update
./.sdp/bin/sdp doctor adapters
```

If `generate-adapters --check` reports changes, fix source prompts and run
`generate-adapters --write` after manifest changes. Do not patch generated files
by hand.

## Limits

Be honest in pilots:

- SDP is not a replacement for code review.
- SDP does not guarantee compliance.
- Policy enforcement is advisory by default unless configured otherwise.
- Native Windows is not supported in v1.
- Some older reference docs still describe historical commands; prefer this quickstart, `cmd/sdp/main.go`, and `sdp <command> --help` where implemented.
- Index commands require the `sqlite_fts5` build tag. The installer and local build command above include it.

Trust wording: [reference/trust-guarantees.md](reference/trust-guarantees.md)  
Component status: [reference/maturity-matrix.md](reference/maturity-matrix.md)
