# SDP Quickstart

Get SDP installed in a repo and run the first useful checks.

Audience: CTOs, architects, and developers evaluating SDP as a structured AI PDLC/SDLC harness layer.

## What You Are Installing

SDP installs a repo-local harness surface:

- `sdp.manifest.yaml` — source of truth for skills, commands, agents, and harness adapters
- generated adapter files for Claude Code, OpenCode, Codex, and Cursor
- `sdp.lock` — installed SDP version pin
- optional `.sdp/` outputs from scout, metrics, index, specs, evidence, and later operator runs

It does not ask for model API keys during install. Model/provider setup belongs to the harness you use.

## Prerequisites

| Requirement | Minimum | Why |
|---|---:|---|
| Git | 2.30+ | Clone/install and repo analysis |
| Go | 1.26+ | Build the `sdp` CLI from source |
| macOS, Linux, or WSL | - | Native Windows is not supported in v1 |
| One AI harness | optional | Claude Code, OpenCode, Codex, or Cursor if you want generated commands |

## Install

Run this from the root of the repo where you want SDP installed:

```bash
curl -fsSL https://raw.githubusercontent.com/fall-out-bug/sdp_lab/main/scripts/install.sh | bash
```

The installer:

1. uses `sdp` from `PATH` if it already exists
2. otherwise clones `fall-out-bug/sdp_lab`
3. builds `cmd/sdp`
4. runs `sdp init --harness auto`
5. writes `sdp.manifest.yaml`, harness adapter dirs, and `sdp.lock`

Environment overrides:

```bash
SDP_HARNESS=claude-code,opencode \
SDP_TARGET=/path/to/repo \
curl -fsSL https://raw.githubusercontent.com/fall-out-bug/sdp_lab/main/scripts/install.sh | bash
```

If you are already inside this repo and want the local binary:

```bash
go build -tags "sqlite_fts5" -o "$(go env GOPATH)/bin/sdp" ./cmd/sdp
sdp init --harness auto --target /path/to/your/repo
```

## Verify

From the target repo:

```bash
sdp manifest validate
sdp doctor adapters
```

Expected result:

- manifest validation exits 0
- adapter doctor reports 0 drifts
- `sdp.lock` exists
- one or more harness dirs exist: `.claude/`, `.opencode/`, `.codex/`, `.cursor/`

## First Useful Run

Run read-only toolkit commands first. They show SDP's value without changing your repo:

```bash
sdp scout --format text .
sdp metrics --format markdown .
sdp index build --format text .
sdp spec --format text .
```

Then preview the delivery-planning surface:

```bash
sdp build "Add a small feature with tests" --dry-run --format text
```

For brownfield agent setup, preview generated artifacts before writing:

```bash
sdp bootstrap --dry-run --mode brownfield .
```

## Choose A Path

| Path | Use when | Start with |
|---|---|---|
| **Toolkit evaluation** | You want to inspect an existing repo and recover useful context. | `scout`, `metrics`, `index`, `spec`, `bootstrap --dry-run` |
| **Local delivery** | You want a lightweight idea-to-change loop in one repo. | `sdp build --dry-run`, then harness-specific commands |
| **Operator mode** | You need queue-backed delivery, explicit ownership, PR gates, and QA/UAT. | [reference/canonical-happy-path.md](reference/canonical-happy-path.md) |
| **MCP integration** | You want an AI harness to call SDP tools directly. | [reference/installation.md](reference/installation.md) |

## What Works Today

Stable or ready to evaluate:

- multi-harness manifest/adapters: 29 skills, 24 commands, 12 agents
- toolkit commands: `scout`, `metrics`, `index`, `spec`, `bootstrap`
- Beads-backed operator workflow in `sdp_lab`
- evidence/schema/protocol checks
- StratAudit reports

Useful but operator-facing:

- `sdp-orchestrate`, `sdp-ci-loop`, `sdp-guard`, `sdp-doc-sync`, `sdp-ready`
- `sdp manifest parity`, `sdp generate-adapters`, `sdp doctor adapters`

Experimental or research:

- strict `agentloop` + `sdp-harness` primary delivery runtime
- model gateway, cascades, MicroFirst inference, telemetry daemon
- K8s/swarm/control tower paths
- ChangePassport as a separate merge-readiness product direction

Canonical map: [reference/product-surface.md](reference/product-surface.md)

## Configure Harnesses

The manifest supports four harness names:

```bash
sdp init --harness all
sdp init --harness auto
sdp init --harness claude-code,opencode
sdp init --harness cursor,codex --target /path/to/repo
```

`auto` detects existing harness directories. If none exist, it installs all four.

Generated adapters are owned by the manifest. Do not edit generated harness files directly. Change `sdp.manifest.yaml`, then regenerate:

```bash
sdp generate-adapters --write
sdp doctor adapters
```

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
