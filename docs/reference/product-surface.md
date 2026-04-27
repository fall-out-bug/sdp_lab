# SDP Product Surface

Status: canonical user-facing product map  
Updated: 2026-04-27

Use this doc when the question is:

- what is SDP today?
- what should a CTO, architect, or developer try first?
- what is stable, what is useful tooling, and what is still experimental?

## Positioning

SDP is a governed AI software delivery harness.

Short version:

> From idea to accepted PR, with evidence.

More precise version:

> SDP is the operating layer above coding agents. It adds scope, workstreams, gates, evidence, findings loops, and QA/UAT around agent-assisted delivery.

SDP is not trying to be a better IDE or a black-box AI software engineer. Codex, Claude Code, Cursor, Copilot, OpenCode, and other harnesses can own execution UX. SDP owns the delivery contract around that execution.

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
- `sdp manifest validate`, `sdp manifest parity`, `sdp generate-adapters`, `sdp doctor adapters`
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
sdp scout --format text .
sdp metrics --format markdown .
sdp index build --format text .
sdp spec --format text .
sdp bootstrap --dry-run --mode brownfield .
```

This path is low-risk because it reads the repo and writes only when you explicitly run bootstrap without `--dry-run`.

### 3. Run Operator Mode

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

## What To Avoid Claiming

Do not claim that SDP:

- replaces code review
- guarantees compliance
- is a complete autonomous software engineer
- has market-grade IDE UX
- makes every task faster
- is tamper-proof

Use the canonical wording in [trust-guarantees.md](trust-guarantees.md): tamper-evident evidence, human review required, policy enforcement configurable and advisory by default unless configured otherwise.
