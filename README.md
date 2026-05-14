# SDP Lab

Public build, planning, and orchestration workspace for SDP.
GitHub repo name: `sdp_lab`. Go module path: see `go.mod`.

## What SDP Is

SDP is a governed AI software delivery harness.

Short version:

> From idea to accepted PR, with evidence.

SDP does not try to replace Codex, Claude Code, Cursor, OpenCode, Copilot, or other coding agents. It adds the delivery contract around them: scope, workstreams, gates, evidence, findings loops, and QA/UAT.

In SDP, a **harness** means the coding-agent runtime a developer uses to talk to
models and edit code, for example Claude Code, OpenCode, Codex, Cursor, or Pi.
SDP wraps that runtime with repo-local instructions, adapters, evidence, and
review discipline.

The problem SDP targets: agent-assisted delivery often produces code without a
clear scope contract, durable evidence, or an honest record of what was not
checked. SDP makes those missing delivery controls explicit.

## Product Layers

SDP is organized into seven product layers. Only the first installable surface ships today.

| # | Layer | What it is | Status |
|---|---|---|---|
| 1 | **SDP Lab** | Research workspace (this repo). Where SDP is built and exercised. | Active |
| 2 | **SDP Toolbox** | Single-purpose repo-inspection utilities (`scout`, `metrics`, `index`, `spec`, `bootstrap`). Freemium funnel for the SDP family. | GA |
| 3 | **SDP Toolkit** | Installable developer surface. The `sdp` CLI, installed via Homebrew or the install script. | GA |
| 4 | **Operator Mode** | Default Toolkit happy path. Stateful orchestration: features, workstreams, evidence, QA/UAT. | GA inside sdp_lab |
| 5 | **ChangePassport** (`sdp-pr-gate`) | Merge-readiness product. A separate product surface for PR governance. | Product direction, not yet shipped |
| 6 | **Enterprise Delivery Governance** | Enterprise governed delivery control plane. | Hypothesis |
| 7 | **Shared Substrates** | Versioned semver packages (`sdp-evidence-core`, `sdp-policy-core`, etc.) | Implicit today |

Canonical reference: [`docs/strategy/2026-04-27-sdp-product-layering-4d.md`](docs/strategy/2026-04-27-sdp-product-layering-4d.md)

### What ships in the formula

The default Homebrew formula installs the `sdp` binary (SDP Toolkit). It includes stable subcommands: `scout`, `metrics`, `index`, `spec`, `bootstrap`, `init`, `manifest`, `generate-adapters`, `doctor`, and more.

The formula does **not** include:
- lab-only binaries (`sdp-control`, `sdp-dispatch`, `sdp-up`)
- experimental binaries (`sdp-harness`, `sdp-a2a`, `sdp-strataudit`)
- research/benchmark binaries (`sdp-cascade-replay`, `sdp-decompose-bench`, etc.)
- ChangePassport (`sdp-pr-gate`) — separate product surface, not yet implemented

Operator tooling (`sdp-orchestrate`, `sdp-guard`, `sdp-ci-loop`, `sdp-evidence`, etc.) is included in the release build, but it is not the first-run promise.

Full inventory: [`docs/reference/maturity-matrix.md`](docs/reference/maturity-matrix.md)

## What This Repo Does

`sdp_lab` is the primary public workspace where SDP is built and exercised.

- platform code lives here: Go binaries, orchestration, evals, adapters, K8s manifests
- planning lives here: roadmap, workstreams, design docs, execution runbooks
- protocol artifacts live at native paths: `prompts/`, `schema/`, `templates/`, `.claude/hooks/`, harness entrypoints such as `.cursorrules`, `.codex/`, `.opencode/hooks/`, and fallback docs
- the `sdp` repo is now a distilled distribution/mirror surface, not the upstream source of truth

If your goal is to **use SDP inside your own project**, start with [docs/QUICKSTART.md](docs/QUICKSTART.md).
If you are unsure where to start, use [docs/START_HERE.md](docs/START_HERE.md).

## Clone

```bash
git clone https://github.com/fall-out-bug/sdp_lab sdp_lab
cd sdp_lab
go build -tags "sqlite_fts5" ./...
```

## Rules

- This repo is the default place for strategic planning.
- Do not commit customer-private architecture, secrets, enterprise scope, or commercial details.
- Publish distilled artifacts to `fall-out-bug/sdp` only through `scripts/sdp-publish.sh` when that mirror needs an update.

## Choose Your Path

| Goal | Start here |
|---|---|
| **I am not sure which SDP path I need** | **[docs/START_HERE.md](docs/START_HERE.md)** |
| **Install SDP Toolkit into your repo** | **[Install in 30 seconds](#install-in-30-seconds)** below, or [docs/QUICKSTART.md](docs/QUICKSTART.md) |
| See the command map | [`docs/reference/commands.md`](docs/reference/commands.md) |
| See the skill and agent map | [`docs/reference/agent-skill-entry-map.md`](docs/reference/agent-skill-entry-map.md) |
| Understand what SDP is good at today | [`docs/reference/product-surface.md`](docs/reference/product-surface.md) |
| Understand component maturity (GA/Beta/Experimental) | [`docs/reference/maturity-matrix.md`](docs/reference/maturity-matrix.md) |
| Understand what `sdp_lab` is and what lives here | [`docs/reference/project-map.md`](docs/reference/project-map.md) |
| Contribute to the platform/runtime | [`AGENTS.md`](AGENTS.md), [`docs/MULTI-REPO-WORKFLOW.md`](docs/MULTI-REPO-WORKFLOW.md) (publish workflow), [`docs/roadmap/ROADMAP.md`](docs/roadmap/ROADMAP.md) |
| Adopt SDP in a greenfield or brownfield project | [`docs/QUICKSTART.md`](docs/QUICKSTART.md), then [`docs/runbooks/onboarding-downstream-repo.md`](docs/runbooks/onboarding-downstream-repo.md) |
| Trust, security guarantees, and limitations | [`docs/reference/trust-guarantees.md`](docs/reference/trust-guarantees.md) |
| CI gates and local reproduce commands | [`docs/reference/ci-gates-map.md`](docs/reference/ci-gates-map.md) |

## First Proof

For a cold pilot, prove the small thing first:

```bash
./.sdp/bin/sdp scout --repo .
./.sdp/bin/sdp metrics --repo .
./.sdp/bin/sdp doctor
```

Then read the generated findings before trying orchestration. The first useful
SDP result is not "the agent changed code"; it is an explicit map of scope,
evidence, limits, and next actions.

## Install in 30 seconds

Run in the root of your downstream repo (requires `git` and `go`):

```bash
curl -fsSL https://raw.githubusercontent.com/fall-out-bug/sdp_lab/main/scripts/install.sh | bash
```

Local-source install while working inside this repo:

```bash
SDP_SOURCE_DIR="$PWD" SDP_TARGET=/path/to/myrepo bash scripts/install.sh
```

The installer clones `sdp_lab` to bring in the canonical manifest and prompts, builds a repo-local `./.sdp/bin/sdp` unless you explicitly allow PATH reuse, runs `init --harness=auto` through the installer binary, verifies the repo-local CLI, and writes `sdp.lock`. No manual file copying.

### What you get

The skills, commands, and agents declared in `sdp.manifest.yaml` are rendered
into the native surfaces each harness supports (`.claude/`, `.opencode/`,
`.codex/`, `.cursor/`, `.pi/`).

This is static adapter coverage. It proves files are generated from one
manifest; it does not prove each harness is ready for autonomous SDP dispatch.
Claude Code is the stable primary harness today. OpenCode requires
`--agent implementer`; Cursor, Codex, and Pi are validation/manual-assist
surfaces unless their runtime readiness row says otherwise.

Parity snapshot (full table: [`docs/reference/harness-parity-matrix.md`](docs/reference/harness-parity-matrix.md)):

| Command | claude-code | opencode | codex | cursor | pi |
|---|---|---|---|---|---|
| `build` | ✓ | ✓ | ✓ | ✓ | ✓ |
| `feature` | ✓ | ✓ | ✓ | ✓ | ✓ |
| `deploy` | ✓ | ✓ | ✓ | ✓ | ✓ |
| `review` | ✓ | ✓ | ✓ | ✓ | ✓ |

`sdp.manifest.yaml` is the single source of truth for the current command, skill, and agent counts.

### Selective install

Install only the harnesses you use:

```bash
./.sdp/bin/sdp init --harness=claude-code,opencode
./.sdp/bin/sdp init --harness=auto           # detect by existing dirs
./.sdp/bin/sdp init --harness=all            # all five harnesses
./.sdp/bin/sdp init --harness=auto --target=/path/to/myrepo
```

`--harness=auto` installs all harnesses if no harness dirs exist yet.

### Update / pin version

```bash
# Validate manifest is well-formed
./.sdp/bin/sdp manifest validate

# Re-generate the .sdp/generated cache from manifest
./.sdp/bin/sdp generate-adapters --write

# Re-install live harness adapter files from manifest/cache
./.sdp/bin/sdp init --update

# Check for drift (adapter files out of sync with manifest)
./.sdp/bin/sdp doctor adapters

# Optional shell convenience after local verification
export PATH="$PWD/.sdp/bin:$PATH"
command -v sdp
```

`sdp.lock` pins the SDP version used at install time. `./.sdp/bin/sdp doctor adapters` fails if installed adapters diverge from the manifest — safe to run in CI or as a pre-commit hook. If `sdp manifest` or `sdp scout` says the command is missing, the shell is using an older global `sdp`; run `./.sdp/bin/sdp ...` or fix `PATH`.

### Customize without forking

The canonical inventory is `sdp.manifest.yaml`. Generated adapter files land in `.sdp/generated/`; `sdp init --update` installs them into harness dirs (`.claude/commands/`, `.opencode/`, `.codex/`, `.cursor/rules/`, `.pi/`).

**Do not edit harness adapter files directly** — `sdp doctor` will flag the drift. Instead:

1. Edit `sdp.manifest.yaml` (and/or templates in `internal/adapters/templates/<harness>/`)
2. Run `./.sdp/bin/sdp generate-adapters --write`
3. Run `./.sdp/bin/sdp init --update` to refresh live harness files
4. Commit the manifest, `.sdp/generated/`, and regenerated harness adapter files

Overlay system (per-repo customization without touching the manifest) is planned for F142+. For now, the workflow is: edit manifest → regenerate cache → install adapters → commit.

Full design: [`docs/plans/2026-04-25-f141-multi-harness-install-bootstrap-design.md`](docs/plans/2026-04-25-f141-multi-harness-install-bootstrap-design.md)  
Onboarding runbook: [`docs/runbooks/onboarding-downstream-repo.md`](docs/runbooks/onboarding-downstream-repo.md)

## What Works Today

**SDP Toolkit (installable product):**

- multi-harness install from `sdp.manifest.yaml`
- `scout`, `metrics`, `index`, `spec`, `bootstrap`

**Operator Mode (default Toolkit happy path):**

- Beads-backed operator workflow in this repo
- evidence, protocol, adapter, and documentation checks
- StratAudit reports

**Operator tooling (included in the release build, not the first-run promise):**

- `sdp-orchestrate`, `sdp-ci-loop`, `sdp-guard`, `sdp-doc-sync`, `sdp-ready`
- `manifest validate`, `manifest parity`, `generate-adapters`, `doctor adapters`

**Lab / research (not in formula):**

- strict `agentloop` + `sdp-harness` primary delivery runtime
- model gateway, provider cascade, MicroFirst inference, telemetry daemon
- K8s/swarm/control tower paths

**Product direction (not yet shipped):**

- ChangePassport (`sdp-pr-gate`) — separate merge-readiness product surface
- Enterprise Delivery Governance — enterprise governed delivery control plane

Full map: [`docs/reference/product-surface.md`](docs/reference/product-surface.md)

## Harness Support Today

- generated adapter install supports `Claude Code`, `OpenCode`, `Codex`, `Cursor`, and `Pi`
- MCP integration is documented separately in [`docs/reference/installation.md`](docs/reference/installation.md)
- model keys and provider credentials stay in the harness/provider you choose; the installer does not collect them

## Main Components

- `cmd/`, `internal/` — platform binaries, orchestration, evals, kernel, adapters
- `deploy/` — deployable runtime and observability manifests
- `docs/` — planning and execution surfaces (roadmap, workstreams, plans, runbooks, architecture)
- `sdp/` — optional local checkout of the distilled `sdp` repo (used by `scripts/sdp-publish.sh` only); canonical protocol artifacts live at `prompts/`, `schema/`, `templates/`, `.claude/hooks/`, and harness entrypoints in this repo

## CLI Binaries (`cmd/`)

Main CLI:

- `cmd/sdp/` — top-level CLI. Subcommands include `card`, `board`, `doctor`, `dispatch`, `result`, `orchestrate`, `attention`, `why`, `next`, `missing`, `approve`, `trace`, `deploy`, `discover`, `intent`, `status`, `stuck`, `eval`, `clarify`, `plan`, `approve-plan`, `scout`, `spec`, `metrics`, `index`, `bootstrap`, `build`, `manifest`, `generate-adapters`, `init`, and `telemetry`. Source of truth: `cmd/sdp/main.go`.

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
- `cmd/sdp-llm-gateway/` — local demo gateway for guarded model calls; not the production model gateway
- `cmd/sdp-pi-eval/` — local prompt-injection eval runner
- `cmd/sdp-pi-review/` — local multi-model PR/review runner
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

**From GitHub Releases** (after tagging):

```bash
# Linux amd64
curl -sSL https://github.com/fall-out-bug/sdp_lab/releases/download/<tag>/sdp-evidence_<version>_linux_amd64.tar.gz | tar xz -C /usr/local/bin

# macOS (darwin/arm64)
curl -sSL https://github.com/fall-out-bug/sdp_lab/releases/download/<tag>/sdp-evidence_<version>_darwin_arm64.tar.gz | tar xz -C /usr/local/bin
```

**From source:**

```bash
go install github.com/fall-out-bug/sdp_lab/cmd/sdp-evidence@latest
```

### Usage

```bash
sdp-evidence validate --evidence .sdp/evidence/run-123.json
sdp-evidence inspect --evidence .sdp/evidence/run-123.json
```
