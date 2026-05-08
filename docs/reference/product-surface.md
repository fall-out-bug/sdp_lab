# SDP Product Surface

Status: canonical user-facing product map
Updated: 2026-04-27
Workstream: 00-150-02 (F150-02, sdplab-8rk7)

Use this doc when the question is:

- what is SDP today?
- what should a CTO, architect, or developer try first?
- what is stable, what is useful tooling, and what is still experimental?
- what ships in the Homebrew formula?

## Positioning

SDP is a governed AI software delivery harness.

Short version:

> From idea to accepted PR, with evidence.

More precise version:

> SDP is the operating layer above coding agents. It adds scope, workstreams, gates, evidence, findings loops, and QA/UAT around agent-assisted delivery.

SDP is not trying to be a better IDE or a black-box AI software engineer. Codex, Claude Code, Cursor, Copilot, OpenCode, and other harnesses can own execution UX. SDP owns the delivery contract around that execution.

## Product Layers (F150 memo v3)

| # | Layer | Kind | Status | First commercial role |
|---|---|---|---|---|
| 1 | SDP Lab | Research workspace | Active | None -- feeds others |
| 2 | SDP Toolbox | Subordinate freemium funnel (with IIP flag) | Partial: F120-F124 done | Free dev adoption |
| 3 | SDP Toolkit | Installable developer surface (`sdp` CLI) | GA inside sdp_lab | Free dev adoption |
| 4 | Operator Mode | Default Toolkit Happy Path; stateful orchestration | GA inside sdp_lab | Team adoption |
| 5 | ChangePassport / `sdp-pr-gate` | Merge-readiness product | Product direction (no code) | First paid wedge |
| 6 | Enterprise Delivery Governance | Enterprise governed delivery control plane | Hypothesis | Enterprise paid wedge |
| 7 | Shared Substrates | Versioned semver packages | Implicit today | Internal contracts |

Canonical reference: [docs/strategy/2026-04-27-sdp-product-layering-4d.md](../strategy/2026-04-27-sdp-product-layering-4d.md)

## Audience

SDP is for:

- CTOs and architects evaluating structured AI PDLC/SDLC for a team
- developers who want agent work to leave reviewable evidence
- consulting teams that need client-readable delivery discipline
- platform owners who need repo-native controls before adding more autonomy

SDP is not a good first choice for:

- solo toy projects where raw agent speed matters more than traceability
- teams that only want autocomplete or chat inside an IDE
- buyers looking for a compliance certification product

## CLI Inventory -- Release Surface Classification

### Formula Default Install (stable product surface)

| Binary | Maturity | What it is |
|--------|----------|------------|
| `sdp` | GA | Main CLI. All product subcommands live here. |

**Stable `sdp` subcommands** (visible in formula help, product promise):

- `scout` -- Fast map of an unfamiliar repository
- `metrics` -- Git-derived process health: churn, hotspots, bus-factor risk
- `index` -- Persistent codebase memory and symbol/query support
- `spec` -- Recovering implicit APIs, rules, invariants, and SLAs from code
- `bootstrap` -- Brownfield-safe generation of agent setup artifacts
- `init` -- Initialize SDP in a repo
- `manifest` -- Manifest validate/parity
- `generate-adapters` -- Adapter generation
- `doctor` -- Diagnostic checks
- `coverage-scan` -- Coverage scanning
- `rules` -- Rules update from evidence
- `skills` -- Skills augment/update

**Operator Mode subcommands** (default Toolkit Happy Path):

- `orchestrate` -- Feature-level orchestration
- `card` -- FeatureCard CRUD
- `board` -- Board build/show
- `phase` -- Phase plan/review/eval
- `build` -- Build planner
- `deploy` -- Deploy staging/prod/rollback
- `discover` -- Discovery pipeline (Stage 0)
- `architect` -- C4 architecture analysis
- `why`, `next`, `missing`, `approve`, `trace` -- Query/insight commands
- `status`, `stuck`, `attention` -- Pipeline status
- `dispatch`, `result`, `intent`, `eval`, `clarify`, `plan`, `approve-plan` -- Pipeline internals
- `tower` -- Tower orchestration (Beta)
- `reset` -- Checkpoint reset

**Experimental subcommands**:

- `telemetry` -- Telemetry init/span/daemon (Beta, opt-in by design)

### Operator Tooling (available in formula or opt-in tap)

| Binary | Maturity | What it is good for |
|--------|----------|---------------------|
| `sdp-orchestrate` | GA | Standalone feature-level orchestration binary |
| `sdp-orchestrate-daemon` | Beta | Daemon variant of orchestrate |
| `sdp-guard` | GA | Scope enforcement binary |
| `sdp-ci-loop` | GA | CI feedback loop binary |
| `sdp-doc-sync` | GA | Doc link checker + sync |
| `sdp-beads-bridge` | GA | Beads <-> SDP bridge |
| `sdp-gh-findings-sync` | GA | GitHub findings -> Beads sync |
| `sdp-ready` | GA | Pre-flight readiness check |
| `sdp-protocol-check` | GA | Protocol validation |
| `sdp-ws-verdict-validate` | GA | Workstream verdict validation |
| `sdp-evidence` | GA | in-toto evidence attestations |
| `sdp-export` | GA | Evidence bundle export |
| `sdp-omc-guard` | Beta | OhMyOpenCode pre-tool-call guard |
| `sdp-session-audit` | Beta | Session audit trail |
| `sdp-healthcheck` | GA | Health check endpoint |

### Lab-Only (not in formula)

| Binary | Maturity | What it is |
|--------|----------|------------|
| `sdp-control` | GA (deprecated) | DEPRECATED. Use `sdp` instead. |
| `sdp-dispatch` | Beta | Dispatch layer (development) |
| `sdp-up` | GA | Profile provisioning (lab setup) |
| `gt-adapter` | GA | Guard/convoy test adapter |

### Experimental / Research (not in formula)

| Binary | Maturity | What it is |
|--------|----------|------------|
| `sdp-harness` | Experimental | AgentLoop session harness. Requires LiveGateway (F106). |
| `sdp-a2a` | Beta | Agent-to-agent protocol server |
| `sdp-eval` | Beta | Eval runner |
| `sdp-strataudit` | GA | Strategic LLM audit (standalone research) |
| `sdp-mcp` | Beta | MCP server |

### Research / Benchmark (not in formula, build-tag isolated)

| Binary | Maturity | What it is |
|--------|----------|------------|
| `sdp-cascade-replay` | Experimental | F145 CascadingInvoker replay bench |
| `sdp-confidence-replay` | Experimental | F144 confidence checker replay |
| `sdp-decompose-bench` | Experimental | F146 decomposition A/B benchmark |
| `sdp-microfirst-bench` | Experimental | MicroFirst micro-classifier bench |
| `sdp-bd-suggest` | Experimental | Beads issue kNN classifier |
| `sdp-ft-baseline` | Experimental | F133 fine-tune baseline runner |
| `sdp-ft-dataset` | Experimental | F133 fine-tune dataset assembler |
| `sdp-ft-run` | Experimental | F133 fine-tune backend driver |
| `sdp-ft-validate` | Experimental | F133 fine-tune JSONL validator |

### Future Product Candidates (not in formula, not in code)

| Binary | What it is |
|--------|------------|
| `sdp-pr-gate` (ChangePassport) | Merge-readiness product. Internal namespace locked. No implementation yet. |

## What Works Well Today

| Surface | Status | What it is good for |
|---|---|---|
| Multi-harness install | Beta | Installing SDP skills, commands, and agents into Claude Code, OpenCode, Codex, and Cursor from `sdp.manifest.yaml`. |
| Toolkit scout | GA | Fast map of an unfamiliar repository. |
| Toolkit metrics | GA | Git-derived process health: churn, hotspots, bus-factor-style risk, review/process signals. |
| Toolkit index | GA | Persistent codebase memory and symbol/query support. |
| Spec recovery | GA | Recovering implicit APIs, rules, invariants, and SLAs from code. |
| Bootstrap | GA | Brownfield-safe generation of agent setup artifacts. |
| Beads-backed operator loop | GA inside `sdp_lab` | Durable work graph for feature/workstream execution. |
| Evidence and protocol checks | GA | Validating evidence, workstream hygiene, adapter parity, and doc drift. |
| StratAudit | GA | Evidence-backed strategy and architecture audit reports. |

## Tooling, Not The Product

These are useful building blocks, but they are not the first thing to sell or demo:

- `sdp-orchestrate`, `sdp-ci-loop`, `sdp-guard`, `sdp-doc-sync`, `sdp-ready`
- `manifest validate`, `manifest parity`, `generate-adapters`, `doctor adapters`
- K8s/deploy scripts and legacy swarm paths
- internal design docs under `docs/plans/` and `docs/archive/`

Use these when operating or developing SDP. Do not make new users read them before the first successful local run.

## Experimental Or Research Lanes

Be explicit about these. They are promising, but not the default onboarding promise.

| Lane | Status | Current boundary |
|---|---|---|
| `agentloop` + `sdp-harness` delivery runtime | Experimental | The strict FSM exists, but full primary-path runtime wiring is still in progress. |
| Model gateway and provider cascade | Experimental/Beta by component | Useful for inference research; not required for basic SDP adoption. |
| MicroFirst, confidence, decomposition | Experimental | Inference efficiency and quality-control research. |
| Telemetry daemon | Beta/Experimental | Trace schema and local daemon work exists; product dashboard is not the default path. |
| K8s/swarm/control tower | Experimental | Background or future enterprise runtime lane, not first-run onboarding. |
| ChangePassport | Product direction, separate boundary | Merge-readiness packet concept. `sdp_lab` can feed it, but it should become a stable product surface only after schema/API hardening. |

## Happy Paths

### 1. Try SDP In Your Repo

Use this when you want to see what SDP installs and what it can inspect.

1. Run the installer from your repo root.
2. Validate the manifest and generated adapters.
3. Run `scout`, `metrics`, `index`, and `spec` on the repo.
4. Optionally run `sdp build --dry-run` to see the delivery-planning surface.

Canonical doc: [../QUICKSTART.md](../QUICKSTART.md)

### 2. Adopt The Toolkit

Use this when the immediate job is understanding or preparing a brownfield repo.

Default commands:

```bash
./.sdp/bin/sdp scout --format text .
./.sdp/bin/sdp metrics --format markdown .
./.sdp/bin/sdp index build --format text .
./.sdp/bin/sdp spec --format text .
./.sdp/bin/sdp bootstrap --dry-run --mode brownfield .
```

This path is low-risk because the analysis commands inspect the repo and `index build` writes only a local `.sdp/index.db` cache. Bootstrap writes project artifacts only when you run it without `--dry-run`.

### 3. Run Operator Mode (Default Toolkit Happy Path)

Use this when a team wants queue-backed delivery with explicit ownership and PR discipline.

Default loop:

1. shape a feature
2. decompose it into workstreams
3. link Beads issues
4. branch from `main`
5. open an early draft PR
6. execute work
7. route review/CI/drift/QA findings back into Beads
8. merge only after QA/UAT passes

Operator Mode is the default Happy Path embodying governed delivery. It is a stateful orchestration layer. It is not a separate paid SKU now, but a provisional pricing hypothesis is required before any pilot.

Canonical doc: [canonical-happy-path.md](canonical-happy-path.md)

### 4. Evaluate PR Governance

Use this when the business problem is review burden and client handoff.

Current SDP building blocks:

- evidence schema and validation
- workstream scope
- CI and review findings loop
- `QA/UAT` closure
- traceability docs

Current limitation: the stable "ChangePassport" packet is a product direction, not yet the default public CLI.

## Formula Install Surface

The default Homebrew formula installs the `sdp` binary. It does NOT install:

- lab-only binaries (`sdp-control`, `sdp-dispatch`, `sdp-up`, `gt-adapter`)
- experimental binaries (`sdp-harness`, `sdp-a2a`, `sdp-strataudit`, `sdp-mcp`)
- research/benchmark binaries (`sdp-cascade-replay`, `sdp-confidence-replay`, `sdp-decompose-bench`, `sdp-microfirst-bench`, `sdp-bd-suggest`, `sdp-ft-*`)

Operator tooling binaries (`sdp-orchestrate`, `sdp-guard`, `sdp-ci-loop`, `sdp-evidence`, etc.) are included in the release build, but they are not the first-run promise.

Exclusion mechanism: GoReleaser allowlist (`.goreleaser.yml`). Build tags (`sdp_experimental`) for compile isolation. See [maturity-matrix.md](maturity-matrix.md) for the full inventory.

## What To Avoid Claiming

Do not claim that SDP:

- replaces code review
- guarantees compliance
- is a complete autonomous software engineer
- has market-grade IDE UX
- makes every task faster
- is tamper-proof

Use the canonical wording in [trust-guarantees.md](trust-guarantees.md): tamper-evident evidence, human review required, policy enforcement configurable and advisory by default unless configured otherwise.
