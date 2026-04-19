# SDP Lab Control Workspace

Private build, planning, and orchestration workspace for SDP.
GitHub repo name: `sdp_lab`. The Go module is still named `sdp_dev` (see `go.mod`) — the same root repo; treat as legacy naming.

## What This Repo Actually Does

`sdp_lab` is the private repo where we build and steer the SDP platform itself.

- platform code lives here: Go binaries, orchestration, evals, adapters, K8s manifests
- planning lives here: roadmap, workstreams, private design docs, execution runbooks
- protocol artifacts live at native paths: `prompts/`, `schema/`, `templates/`, `.claude/hooks/`, harness entrypoints such as `.cursorrules`, `.codex/`, `.opencode/hooks/`, and fallback docs (published to the public `sdp` repo downstream via `scripts/sdp-publish.sh`)

If your goal is to **use SDP inside your own project**, this repo is not the primary onboarding surface. Start with the [SDP Quickstart](https://github.com/fall-out-bug/sdp/blob/main/docs/QUICKSTART.md).

## Clone

```bash
git clone https://github.com/fall-out-bug/sdp_lab
cd sdp_lab
go build ./...
```

## Rules

- This repo is the default place for strategic planning.
- Do not publish private architecture, enterprise scope, or commercial details into OSS repos.
- Export to OSS only through sanitized artifacts.

## Choose Your Path

| Goal | Start here |
|---|---|
| Understand what `sdp_lab` is and what lives here | [`docs/reference/project-map.md`](docs/reference/project-map.md) |
| Contribute to the platform or private lab runtime | [`AGENTS.md`](AGENTS.md), [`docs/MULTI-REPO-WORKFLOW.md`](docs/MULTI-REPO-WORKFLOW.md) (publish workflow), [`docs/roadmap/ROADMAP.md`](docs/roadmap/ROADMAP.md) |
| Adopt SDP in a greenfield or brownfield project | [SDP Quickstart](https://github.com/fall-out-bug/sdp/blob/main/docs/QUICKSTART.md), then [SDP README](https://github.com/fall-out-bug/sdp) |

## IDE Support Today

- public onboarding flow is first-class for `Claude Code`, `Cursor`, and `OpenCode` / `Windsurf`
- `Codex` prompt compatibility exists in [.codex/](https://github.com/fall-out-bug/sdp/tree/main/.codex), but the public install flow is still manual rather than auto-detected
- if the question is "can I give SDP my keys and start working?", the honest answer lives in `sdp/docs/`, not in the private-lab runbooks here

## Main Components

- `cmd/`, `internal/` — platform binaries, orchestration, evals, kernel, adapters
- `deploy/` — deployable runtime and observability manifests
- `docs/` — planning and execution surfaces (roadmap, workstreams, plans, runbooks, architecture)
- `sdp/` — optional local checkout of the public `sdp` repo (used by `scripts/sdp-publish.sh` only); canonical protocol artifacts live at `prompts/`, `schema/`, `templates/`, `.claude/hooks/`, and harness entrypoints in this repo

## CLI Binaries (`cmd/`)

Main CLI:

- `cmd/sdp/` — top-level CLI. Subcommands: `card`, `board`, `doctor`, `dispatch`, `result`, `orchestrate`, `attention`, `why`, `next`, `missing`, `approve`, `trace`, `deploy`, `discover`, `intent`, `status`, `stuck`, `eval`, `clarify`, `plan`, `approve-plan`. Analytical / toolkit (not in `--help`): `tower`, `architect`, `scout`, `metrics`, `index` (F122). Source of truth: `cmd/sdp/main.go`.

Standalone binaries:

- `cmd/sdp-orchestrate/` — oneshot outer loop (`--advance`, `--status`, `--next-action`, `--feature`)
- `cmd/sdp-orchestrate-daemon/` — long-running orchestration daemon
- `cmd/sdp-harness/` — agentloop FSM harness (`new`, `run`, `compile-lock`, `release`, `events`)
- `cmd/sdp-evidence/` — evidence envelope `validate` + `inspect` (zero K8s dep)
- `cmd/sdp-dispatch/` — routing and profiling for harness dispatch
- `cmd/sdp-ci-loop/` — CI feedback loop with deterministic autofix
- `cmd/sdp-eval/` — evaluation framework runner
- `cmd/sdp-guard/` — permission scope gate for agent invocations
- `cmd/sdp-omc-guard/` — OMO client guard for tool policy enforcement
- `cmd/sdp-beads-bridge/` — Beads issue tracker bridge
- `cmd/sdp-ready/` — find ready work from Beads with SDP WS mapping
- `cmd/sdp-protocol-check/` — SDP protocol hygiene validator
- `cmd/sdp-doc-sync/` — docs consistency + changelog automation
- `cmd/sdp-strataudit/` — stratified audit / trace explorer (F117)
- `cmd/sdp-a2a/` — agent-to-agent communication server
- `cmd/sdp-control/` — control plane CLI
- `cmd/sdp-ws-verdict-validate/` — workstream verdict validation
- `cmd/sdp-gh-findings-sync/` — sync GitHub findings into local Beads queue
- `cmd/sdp-up/` — bootstrap and deploy SDP components

## Key Docs

See **[`docs/reference/project-map.md`](docs/reference/project-map.md)** for the canonical SOT split and full read order. High-frequency entry points:

- [`AGENTS.md`](AGENTS.md) — operator rules, workflow, command tree
- [`docs/MULTI-REPO-WORKFLOW.md`](docs/MULTI-REPO-WORKFLOW.md) — publish workflow for protocol artifacts
- [`docs/architecture/REPO-BOUNDARY.md`](docs/architecture/REPO-BOUNDARY.md) — what belongs where
- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — system architecture
- [`docs/roadmap/ROADMAP.md`](docs/roadmap/ROADMAP.md) — current product direction
- [`docs/phases/DISCOVERY.md`](docs/phases/DISCOVERY.md), [`docs/phases/DELIVERY.md`](docs/phases/DELIVERY.md) — phase contracts
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — 5-minute dev setup
- [`VISION.md`](VISION.md) — что такое SDP

Specs: `specs/autonomy-runtime-contract.yaml`, `specs/brain-decision-api.yaml`, `specs/strict-evidence-template.json`. Evidence schema: `schema/evidence-envelope.schema.json`.

Для конкретных runbooks (observability, k8s bootstrap, PR gate, opencode agent launch) — смотри `docs/` по имени, не полагайся на ручной каталог здесь.

## sdp-evidence CLI

Standalone binary for validating and inspecting evidence envelopes. No K8s dependency.

### Install

**From GitHub Releases** (after tagging e.g. `v0.1.0`):

```bash
# Linux amd64
curl -sSL https://github.com/OWNER/REPO/releases/download/v0.1.0/sdp-evidence_0.1.0_linux_amd64.tar.gz | tar xz -C /usr/local/bin

# macOS (darwin/arm64)
curl -sSL https://github.com/OWNER/REPO/releases/download/v0.1.0/sdp-evidence_0.1.0_darwin_arm64.tar.gz | tar xz -C /usr/local/bin
```

**From source:**

```bash
go install ./cmd/sdp-evidence@latest
```

### Usage

```bash
sdp-evidence validate --evidence .sdp/evidence/run-123.json
sdp-evidence inspect --evidence .sdp/evidence/run-123.json
```
