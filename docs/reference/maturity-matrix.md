# SDP Maturity Matrix

> **Canonical source** for component maturity labels and release surface classification. Referenced by README, roadmap, GoReleaser, and compliance docs.
> **Last updated:** 2026-04-27
> **Workstream:** 00-150-02 (F150-02, sdplab-8rk7)

## Maturity Levels

| Level | Label | Meaning | Graduation Criteria | Rollback Trigger |
|-------|-------|---------|---------------------|------------------|
| 3 | **GA** | Production-ready. Used by default in all workflows. Backward-compatible within major version. | 90-day burn-in in production; zero P0/P1 regressions; test coverage >= 80%; docs complete (reference + runbook). | 2+ P1 regressions in 30 days; documented design flaw requiring breaking change. |
| 2 | **Beta** | Functional for intended use case. API may change. Not recommended as sole workflow path. | Feature-complete for stated scope; test coverage >= 60%; reference doc exists; at least 1 non-author user has used successfully. | P0 regression in GA dependency; scope expansion requiring API redesign. |
| 1 | **Experimental** | Proof-of-concept or partial implementation. May have zero callers. Not guaranteed to compile in all configurations. | Compiles; basic happy-path test passes; documented intent in reference or plan doc. | No commits in 90 days; superseded by alternative implementation. |

## Release Surface Classification

| Classification | Meaning | In GoReleaser formula? |
|---|---|---|
| **stable** | Ships in the default Homebrew formula. User-facing product surface. | Yes (primary build) |
| **tooling** | Operator and developer tooling. Ships in formula but not the first-run promise. | Yes (primary or opt-in) |
| **lab-only** | Used only inside sdp_lab development. Not shipped in the formula. | No |
| **experimental** | Research/benchmark tool. Not shipped in the formula. | No |
| **retired** | Deprecated and slated for removal or replaced by `sdp` subcommands. | No |
| **future** | Product direction candidate (e.g., sdp-pr-gate). Not yet implemented as a product. | No |

## CLI Binaries (cmd/) — Full Inventory

### Stable Product Surface (formula default)

| Binary | Maturity | Surface | Owner | Notes |
|--------|----------|---------|-------|-------|
| `sdp` | GA | stable | platform | Main CLI entry point. Subcommands: scout, metrics, index, spec, bootstrap, card, board, doctor, manifest, init, generate-adapters, skills, coverage-scan, rules, build, telemetry, reset, phase, discover, architect. See subcommand classification below. |

### Operator Tooling (formula, not first-run)

| Binary | Maturity | Surface | Owner | Notes |
|--------|----------|---------|-------|-------|
| `sdp-orchestrate` | GA | tooling | platform | Feature-level orchestration. GoReleaser build target. |
| `sdp-orchestrate-daemon` | Beta | tooling | platform | Daemon variant of orchestrate. |
| `sdp-guard` | GA | tooling | platform | Scope enforcement. GoReleaser build target. |
| `sdp-ci-loop` | GA | tooling | platform | CI feedback loop. GoReleaser build target. |
| `sdp-doc-sync` | GA | tooling | platform | Doc link checker + sync. |
| `sdp-beads-bridge` | GA | tooling | platform | Beads <-> SDP bridge. |
| `sdp-gh-findings-sync` | GA | tooling | platform | GitHub findings -> Beads sync. |
| `sdp-ready` | GA | tooling | platform | Pre-flight readiness check. |
| `sdp-protocol-check` | GA | tooling | platform | Protocol validation. |
| `sdp-ws-verdict-validate` | GA | tooling | platform | Workstream verdict validation. |
| `sdp-evidence` | GA | tooling | platform | in-toto evidence attestations. GoReleaser build target. |
| `sdp-export` | GA | tooling | platform | Evidence bundle export. |
| `sdp-omc-guard` | Beta | tooling | platform | OhMyOpenCode pre-tool-call guard hook. |
| `sdp-session-audit` | Beta | tooling | platform | Session audit trail. |
| `sdp-healthcheck` | GA | tooling | platform | Health check endpoint. |

### Lab-Only (not in formula)

| Binary | Maturity | Surface | Owner | Notes |
|--------|----------|---------|-------|-------|
| `sdp-control` | GA (deprecated) | retired | platform | DEPRECATED. Use `sdp` instead. Prints deprecation warning on start. |
| `sdp-dispatch` | Beta | lab-only | platform | Dispatch layer. Development routing only. |
| `sdp-up` | GA | lab-only | platform | Project bootstrap (profile provisioning). Lab setup only. |
| `gt-adapter` | GA | lab-only | platform | Guard/convoy test adapter. Internal development tool. |

### Experimental / Research (not in formula)

| Binary | Maturity | Surface | Owner | Notes |
|--------|----------|---------|-------|-------|
| `sdp-harness` | Experimental | experimental | platform | AgentLoop session-based harness. Requires LiveGateway (F106). |
| `sdp-a2a` | Beta | experimental | platform | Agent-to-agent protocol server. Research. |
| `sdp-eval` | Beta | experimental | platform | Eval runner. GoReleaser build target but research-oriented. |
| `sdp-strataudit` | GA | experimental | platform | Strategic LLM audit. Standalone research tool. |
| `sdp-mcp` | Beta | experimental | platform | MCP (Model Context Protocol) server. |
| `sdp-tower` (sdp tower) | Beta | experimental | platform | Tower orchestration layer. |

### Research / Benchmark (not in formula, build-tag isolated)

| Binary | Maturity | Surface | Owner | Notes |
|--------|----------|---------|-------|-------|
| `sdp-cascade-replay` | Experimental | lab-only | platform | F145 CascadingInvoker replay bench. |
| `sdp-confidence-replay` | Experimental | lab-only | platform | F144 confidence checker replay bench. |
| `sdp-decompose-bench` | Experimental | lab-only | platform | F146 decomposition A/B benchmark. |
| `sdp-microfirst-bench` | Experimental | lab-only | platform | MicroFirst micro-classifier benchmark harness. |
| `sdp-bd-suggest` | Experimental | lab-only | platform | Beads issue classifier using MicroFirst kNN. |
| `sdp-ft-baseline` | Experimental | lab-only | platform | F133 fine-tune baseline runner. |
| `sdp-ft-dataset` | Experimental | lab-only | platform | F133 fine-tune dataset assembler. |
| `sdp-ft-run` | Experimental | lab-only | platform | F133 fine-tune backend driver. |
| `sdp-ft-validate` | Experimental | lab-only | platform | F133 fine-tune JSONL validator. |

### Future Product Candidates (not in formula, not in code)

| Binary | Maturity | Surface | Owner | Notes |
|--------|----------|---------|-------|-------|
| `sdp-pr-gate` (ChangePassport) | N/A | future | TBD | Merge-readiness product. Namespace locked per F150 memo. No implementation yet. |

## cmd/sdp Subcommand Classification

### Stable (visible in formula help, product promise)

| Subcommand | Maturity | Product Layer | Notes |
|---|---|---|---|
| `scout` | GA | SDP Toolkit (Toolbox) | Fast repo map. |
| `metrics` | GA | SDP Toolkit (Toolbox) | Git-derived process health. |
| `index` | GA | SDP Toolkit (Toolbox) | Codebase memory. |
| `spec` | GA | SDP Toolkit (Toolbox) | Spec recovery from code. |
| `bootstrap` | GA | SDP Toolkit (Toolbox) | Brownfield agent setup. |
| `init` | GA | SDP Toolkit | Initialize SDP in a repo. |
| `manifest` | GA | SDP Toolkit | Manifest validate/parity. |
| `generate-adapters` | GA | SDP Toolkit | Adapter generation. |
| `doctor` | GA | SDP Toolkit | Diagnostic checks. |
| `coverage-scan` | GA | SDP Toolkit | Coverage scanning. |
| `rules` | GA | SDP Toolkit (Toolbox) | Rules update from evidence. |
| `skills` | GA | SDP Toolkit | Skills augment/update. |

### Operator Mode (default Toolkit Happy Path)

| Subcommand | Maturity | Product Layer | Notes |
|---|---|---|---|
| `orchestrate` | GA | Operator Mode | Feature-level orchestration via `sdp`. |
| `card` | GA | Operator Mode | FeatureCard CRUD. |
| `board` | GA | Operator Mode | Board build/show. |
| `phase` | GA | Operator Mode | Phase plan/review/eval. |
| `build` | GA | Operator Mode | Build planner. |
| `deploy` | GA | Operator Mode | Deploy staging/prod/rollback. |
| `reset` | GA | Operator Mode | Checkpoint reset. |
| `discover` | GA | Operator Mode | Discovery pipeline (Stage 0). |
| `architect` | GA | Operator Mode | C4 architecture analysis. |

### Query / Insight (require beads/dual mode)

| Subcommand | Maturity | Product Layer | Notes |
|---|---|---|---|
| `why` | GA | Operator Mode | Show why a card is blocked. |
| `next` | GA | Operator Mode | Show next actionable items. |
| `missing` | GA | Operator Mode | Show items lacking evidence. |
| `approve` | GA | Operator Mode | Resolve a human gate. |
| `trace` | GA | Operator Mode | Full feature trace. |
| `status` | GA | Operator Mode | Card status and phase. |
| `stuck` | GA | Operator Mode | Stuck/long-running cards. |
| `attention` | GA | Operator Mode | Attention items. |

### Pipeline / Dispatch (Operator Mode internals)

| Subcommand | Maturity | Product Layer | Notes |
|---|---|---|---|
| `dispatch` | GA | Operator Mode | Dispatch routing. |
| `result` | GA | Operator Mode | Result ingestion. |
| `intent` | GA | Operator Mode | Create intake card from raw intent. |
| `eval` | GA | Operator Mode | Manual build evaluation. |
| `clarify` | GA | Operator Mode | Manual clarification. |
| `plan` | GA | Operator Mode | Show plan for a card. |
| `approve-plan` | GA | Operator Mode | Approve a pending plan. |
| `tower` | Beta | Operator Mode | Tower orchestration layer. |

### Experimental / Research Subcommands

| Subcommand | Maturity | Product Layer | Notes |
|---|---|---|---|
| `telemetry` | Beta | SDP Toolkit | Telemetry init/span/daemon. Beta; opt-in by design per F150 telemetry policy. |

## Internal Packages (internal/) — Full Inventory

### Used by Stable Release Surface (formula build targets)

These packages are imported by binaries in the GoReleaser build config or by `cmd/sdp` subcommands classified as stable/tooling.

| Package | Maturity | Surface | Owner | Used by | Notes |
|---------|----------|---------|-------|---------|-------|
| `internal/evidence` | GA | stable | platform | sdp-evidence, sdp-guard, sdp-orchestrate, sdp-ci-loop, sdp-eval, sdp (reset) | in-toto attestations, EvidenceStore |
| `internal/orchestrate` | GA | stable | platform | sdp-guard, sdp-orchestrate, sdp-ci-loop, sdp (phase, reset, orchestrate, helpers) | Feature-level phase orchestration |
| `internal/guard` | GA | stable | platform | sdp-guard | Scope enforcement logic |
| `internal/ciloop` | GA | stable | platform | sdp-orchestrate, sdp-ci-loop | CI feedback loop logic |
| `internal/control` | GA | stable | platform | sdp (card, board, discover, helpers, result, query) | FeatureCard store |
| `internal/executor` | GA | stable | platform | sdp (dispatch, orchestrate, pipeline) | ServeBridge -> DispatchAndRun |
| `internal/cli` | GA | stable | platform | sdp (card, board, doctor, query) | CLI helpers |
| `internal/scout` | GA | stable | platform | sdp (scout, bootstrap, rules) | Repo scanning |
| `internal/metrics` | GA | stable | platform | sdp (metrics) | Git process health |
| `internal/index` | GA | stable | platform | sdp (index) | Codebase memory |
| `internal/spec` | GA | stable | platform | sdp (spec) | Spec recovery |
| `internal/bootstrap` | GA | stable | platform | sdp (bootstrap, rules) | Brownfield setup |
| `internal/manifest` | GA | stable | platform | sdp (manifest, doctor, init, generate-adapters) | Manifest handling |
| `internal/adapters` | GA | stable | platform | sdp (doctor, init, generate-adapters) | Adapter management |
| `internal/discovery` | GA | stable | platform | sdp (discover, checkpoint) | 4-phase LLM pipeline |
| `internal/architect` | GA | stable | platform | sdp (architect) | C4 analysis + runtime coupling |
| `internal/evals` | Beta | stable | platform | sdp-eval | Evaluation framework |
| `internal/deploy` | Beta | stable | platform | sdp (deploy) | Docker Compose wrapper |
| `internal/gate` | GA | stable | platform | sdp (phase) | Gate filesystem |
| `internal/delta` | GA | stable | platform | sdp (phase) | Delta calculation |
| `internal/workstream` | GA | stable | platform | sdp-protocol-check, sdp-beads-bridge | Workstream parsing + validation |
| `internal/beads` | GA | stable | platform | sdp-beads-bridge, sdp-ready | Beads/Dolt integration |
| `internal/bridge` | GA | stable | platform | sdp-gh-findings-sync | Bridge abstractions |
| `internal/docsync` | GA | stable | platform | sdp-doc-sync | Doc sync + link checker |
| `internal/harness` | GA | stable | platform | sdp-guard | Harness lifecycle |
| `internal/sdputil` | GA | stable | platform | sdp (reset) | SDP utilities |
| `internal/gitutil` | GA | stable | platform | multiple | Git utilities |
| `internal/executil` | GA | stable | platform | sdp (coverage) | Exec utilities |
| `internal/verify` | GA | stable | platform | multiple | Verification tools |
| `internal/prompt` | GA | stable | platform | discovery | Prompt construction |
| `internal/build` | GA | stable | platform | sdp (build) | Build planner |
| `internal/promote` | GA | stable | platform | sdp (build, phase) | Promotion logic |
| `internal/rules` | GA | stable | platform | sdp (bootstrap, rules) | Rules engine |
| `internal/harnessadapter` | GA | stable | platform | sdp (bootstrap, rules) | Harness adapter generation |
| `internal/harnesscfg` | GA | stable | platform | sdp (bootstrap, rules) | Harness configuration |
| `internal/skills` | GA | stable | platform | sdp (skills) | Skills management |
| `internal/export` | GA | stable | platform | sdp-export | Export logic |
| `internal/backlog` | GA | stable | platform | sdp (doctor backlog) | Backlog management |
| `internal/trace` | Beta | stable | platform | sdp (telemetry) | Trace primitives |
| `internal/healthcheck` | GA | stable | platform | sdp-healthcheck | Health check logic |
| `internal/sessionaudit` | Beta | stable | platform | sdp-session-audit | Session audit |
| `internal/common` | GA | stable | platform | multiple | Common utilities |
| `internal/kernel` | GA | stable | platform | executor | Execution kernel primitives |
| `internal/runtime` | GA | stable | platform | multiple | Runtime utilities |
| `internal/router` | GA | stable | platform | multiple | Request routing |
| `internal/session` | GA | stable | platform | sdp-harness | Session store (SQLite WAL) |
| `internal/snapshot` | GA | stable | platform | sdp (test) | Snapshot testing |
| `internal/testwriter` | GA | stable | platform | sdp (coverage) | Test file writing |

### Operator Tooling / Lab-Only Packages

| Package | Maturity | Surface | Owner | Notes |
|---------|----------|---------|-------|-------|
| `internal/tower` | Beta | tooling | platform | Tower orchestration layer |
| `internal/dispatch` | Beta | tooling | platform | Dispatch routing |
| `internal/a2a` | Beta | tooling | platform | Agent-to-agent protocol |
| `internal/monitor` | Beta | tooling | platform | Metrics/monitoring |
| `internal/profile` | Beta | tooling | platform | Profile management |
| `internal/policy` | Beta | tooling | platform | Policy engine |
| `internal/augmentation` | Beta | tooling | platform | Context augmentation |
| `internal/convoy` | GA | tooling | platform | Guard/convoy adapter |
| `internal/strataudit` | GA | tooling | discovery | Strategic audit, provider-neutral |
| `internal/mcp` | Beta | tooling | platform | MCP server integration |
| `internal/coveragegate` | GA | tooling | platform | Coverage gate logic |
| `internal/readiness` | GA | tooling | platform | Readiness checks |

### Experimental / Research Packages

| Package | Maturity | Surface | Owner | Notes |
|---------|----------|---------|-------|-------|
| `internal/agentloop` | Experimental | experimental | platform | AgentLoop FSM. Needs LiveGateway (F106). |
| `internal/modelgateway` | Experimental | experimental | platform | Library ready, 0 production callers |
| `internal/inference` | Experimental | experimental | platform | Inference/cascade/confidence research |
| `internal/llmclient` | Experimental | experimental | platform | LLM client abstractions |
| `internal/localmodel` | Experimental | experimental | platform | Local model dispatch |
| `internal/memory` | Experimental | experimental | platform | Memory/context research |
| `internal/mutation` | Experimental | experimental | platform | Mutation testing research |
| `internal/finetune` | Experimental | experimental | platform | Fine-tuning pipeline (F133) |
| `internal/planner` | Experimental | experimental | platform | Written, 0 callers |
| `internal/authz` | Experimental | experimental | platform | Written, 0 callers |
| `internal/stream` | Experimental | experimental | platform | Streaming research |
| `internal/secretscan` | Experimental | experimental | platform | Secret scanning |
| `internal/provenance` | Experimental | experimental | platform | Provenance tracking |
| `internal/flaky` | Experimental | experimental | platform | Flaky test detection |
| `internal/glob` | Experimental | experimental | platform | Glob utilities |

### Infrastructure Packages

| Package | Maturity | Surface | Owner | Notes |
|---------|----------|---------|-------|-------|
| `internal/agentloop` | Experimental | experimental | platform | AgentLoop FSM |
| `internal/knowledge` | N/A | N/A | N/A | No directory found. |

## GoReleaser Build Targets

Current GoReleaser config (`.goreleaser.yml`) builds these binaries:

| Build ID | Binary | Surface Classification | In Formula? |
|---|---|---|---|
| `sdp-evidence` | `sdp-evidence` | tooling | Yes |
| `sdp-guard` | `sdp-guard` | tooling | Yes |
| `sdp-orchestrate` | `sdp-orchestrate` | tooling | Yes |
| `sdp-ci-loop` | `sdp-ci-loop` | tooling | Yes |
| `sdp-eval` | `sdp-eval` | experimental | Yes (should be reviewed) |

**Not in GoReleaser but should be considered for formula:**
- `sdp` (main binary) -- NOT in GoReleaser; must be added for formula
- `sdp-doc-sync`, `sdp-beads-bridge`, `sdp-gh-findings-sync`, `sdp-ready`, `sdp-protocol-check`, `sdp-ws-verdict-validate`, `sdp-healthcheck`, `sdp-export` -- tooling; candidates for formula or opt-in tap

**In GoReleaser but should be reviewed:**
- `sdp-eval` -- classified experimental; consider removing from default formula build

## Exclusion Mechanisms

| Mechanism | Purpose | Status |
|---|---|---|
| GoReleaser allowlist | Only listed binaries ship in the formula. `sdp` + 15 tooling binaries. Experimental binaries excluded. | Active |
| `.goreleaser.yml` build IDs | Defines which cmd/ binaries are built and archived | Active |
| Build tags (`sdp_experimental`) | Compile-time isolation for experimental/lab-only cmd/ binaries. Files have `//go:build sdp_experimental` at top. Default `go build ./...` excludes them. | Active (F150-04) |
| Lab-only binary exclusion | Binaries not in GoReleaser are not built for release | Active (implicit) |
| `scripts/check-release-surface.sh` | Validates release manifest consistency + detects experimental drift (checks 5-7) | Active |
| Internal package import graph | Stable packages (executor, architect, discovery) import some experimental packages (llmclient, glob, agentloop) — these internal packages cannot be build-tagged. Isolation is enforced at cmd/ level only. | Documented (F150-04) |

### How Isolation Works (F150-04)

Three layers prevent experimental code from entering release builds:

1. **GoReleaser allowlist** (primary): `.goreleaser.yml` only includes `sdp` + 15 tooling binaries. All experimental, lab-only, research, and retired binaries are excluded.

2. **Build tags** (`sdp_experimental`): All 18 experimental/lab-only cmd/ binaries have `//go:build sdp_experimental` on every .go file. Default `go build ./...` and `go test ./...` skip them entirely. Local dev builds opt in with:
   ```bash
   go build -tags sdp_experimental ./cmd/sdp-strataudit
   go test -tags "sqlite_fts5 sdp_experimental" ./...
   ```

3. **Drift detection** (`scripts/check-release-surface.sh`): Checks 5-7 verify:
   - No experimental binary appears in `.goreleaser.yml`
   - All experimental cmd/ files have the build tag
   - No stable binary accidentally gets the build tag

### Packages Not Tagged (intentional)

These experimental internal packages are imported by stable code and cannot be build-tagged:

| Package | Imported by stable | Reason |
|---|---|---|
| `internal/llmclient` | architect, discovery | Shared LLM HTTP client |
| `internal/glob` | executor | Used by evaluator |
| `internal/agentloop` | executor (bridge_serve) | Harness bridge |
| `internal/secretscan` | deploy | Deploy scanning |
| `internal/stream` | cmd/sdp-watch (test) | Watch test |
| `internal/mutation` | CI workflow | Mutation testing |

These packages remain in the default build. Their isolation is the cmd/ binary exclusion -- no experimental cmd/ binary can enter the release formula.

## Summary

### By Surface Classification

| Surface | Binaries | Notes |
|---|---|---|
| stable | 1 (`sdp`) | Main CLI, all product subcommands |
| tooling | 15 | Operator/developer tooling binaries |
| lab-only | 4 | Development and test tools |
| experimental | 6 | Research tools |
| research/benchmark | 9 | Benchmarks and fine-tune tooling |
| retired | 1 (`sdp-control`) | Deprecated, use `sdp` |
| future | 1 (`sdp-pr-gate`) | Product direction, no code |
| **Total binaries** | **37** | |

### By Maturity

| Maturity | Count | Percentage |
|----------|-------|------------|
| GA | 47 | 60% |
| Beta | 15 | 19% |
| Experimental | 16 | 21% |
| **Total components** | **78** | **100%** |

### Formula Default Install Surface

The default Homebrew formula should install:
- **`sdp`** -- the main CLI binary (stable product surface)

The formula should NOT install by default:
- Lab-only binaries (`sdp-control`, `sdp-dispatch`, `sdp-up`, `gt-adapter`)
- Experimental binaries (`sdp-harness`, `sdp-a2a`, `sdp-strataudit`, `sdp-mcp`)
- Research/benchmark binaries (`sdp-cascade-replay`, `sdp-confidence-replay`, `sdp-decompose-bench`, `sdp-microfirst-bench`, `sdp-bd-suggest`, `sdp-ft-*`)

Opt-in mechanism: operator tooling binaries available via separate formula tap or build tag (deferred to F150-08).

## Coverage Targets by Maturity (F150-06)

Coverage expectations are tiered by product maturity rather than enforced as a single blanket percentage. This aligns enforcement with the actual risk profile of each component.

### Target Table

| Maturity | Coverage Target | CI Behavior | Denominator |
|----------|----------------|-------------|-------------|
| **Happy-path (GA subcommands on stable surface)** | >= 80% | **Blocking**: coverage gate enforces this target for packages classified as happy-path. | Per-package line coverage for packages that implement the canonical happy-path scenarios (see [canonical-happy-path.md](canonical-happy-path.md)). |
| **GA** (non happy-path) | >= 60% | **Blocking**: coverage gate enforces this target for all GA-maturity packages and commands not on the happy-path. | Per-package line coverage (`go test -cover`). |
| **Beta** | >= 50% | **Advisory**: coverage is reported but does not block merge. Used as graduation signal toward GA. | Per-package line coverage (`go test -cover`). |
| **Experimental** | No target | **Exempt**: experimental packages and commands are excluded from coverage gate enforcement entirely. | N/A. |

### Happy-Path Surface

The happy-path coverage tier (>= 80%) applies to packages that implement the default Toolkit Happy Path. These are the packages a new user encounters on first successful use of `sdp`:

| Package | Maturity | Surface | Happy-Path Role |
|---------|----------|---------|-----------------|
| `internal/scout` | GA | stable | `sdp scout` -- first-run repo map |
| `internal/metrics` | GA | stable | `sdp metrics` -- process health |
| `internal/index` | GA | stable | `sdp index` -- codebase memory |
| `internal/bootstrap` | GA | stable | `sdp bootstrap` -- brownfield setup |
| `internal/control` | GA | stable | FeatureCard store (used by card, board, discover) |
| `internal/orchestrate` | GA | stable | `sdp orchestrate` -- feature orchestration |
| `internal/cli` | GA | stable | CLI helpers used across subcommands |
| `internal/manifest` | GA | stable | `sdp manifest` / `sdp init` |
| `internal/evidence` | GA | stable | in-toto attestations |
| `internal/guard` | GA | stable | Scope enforcement |
| `internal/discovery` | GA | stable | `sdp discover` -- 4-phase LLM pipeline |
| `internal/build` | GA | stable | `sdp build` -- build planner |

### How Coverage Is Measured

1. **Tool**: `go test -tags sqlite_fts5 -coverprofile=cov.out ./...` (standard Go coverprofile).
2. **Aggregation**: Per-package line coverage extracted from coverprofile via `go tool cover -func=cov.out`.
3. **Package classification**: Each `internal/*` and `cmd/*` package is classified by maturity using this matrix.
4. **Enforcement**: The CI coverage gate checks per-package coverage against the target for the package's maturity tier. See [ci-gates-map.md](ci-gates-map.md) for enforcement details.

### Package-to-Tier Mapping

Packages are assigned a coverage tier based on their maturity label in this matrix:

| Tier | Maturity Labels | Packages |
|------|----------------|----------|
| Happy-path (>= 80%) | GA + on happy-path surface | See Happy-Path Surface table above |
| GA (>= 60%) | GA (not on happy-path) | All GA packages not listed in happy-path table |
| Beta (>= 50%, advisory) | Beta | All Beta packages and commands |
| Exempt | Experimental | All Experimental packages and commands |

### Relationship to CI Baseline Delta Gate

The maturity-tiered targets (this section) complement the existing baseline delta gate:
- **Baseline delta gate** (2pp threshold): Catches regressions at the repo-total level.
- **Maturity-tiered targets** (this section): Enforce minimum absolute coverage per maturity tier.

Both gates must pass for a PR to merge.

## Change Log

| Date | Change |
|------|--------|
| 2026-04-26 | Initial matrix created from components.md audit |
| 2026-04-26 | F079-01: Added missing CLI binaries (sdp-healthcheck, sdp-mcp, sdp-session-audit), updated counts |
| 2026-04-27 | F150-02 (sdplab-8rk7): Full inventory of all 37 cmd/ binaries, classification by release surface (stable/tooling/lab-only/experimental/retired/future). Added cmd/sdp subcommand classification. Added missing internal packages (40 new entries). Added GoReleaser build target audit. Added exclusion mechanisms section. Added formula default install surface definition. |
| 2026-04-27 | F150-06 (sdplab-q2cb): Added coverage targets by maturity tier. Happy-path >= 80%, GA >= 60%, Beta >= 50% (advisory), Experimental exempt. Added happy-path surface table and package-to-tier mapping. |
| 2026-04-27 | F150-04 (sdplab-5r4x): Experimental code isolation from release builds. Added `sdp_experimental` build tags to 18 experimental/lab-only cmd/ binaries. Updated `.goreleaser.yml` to include `sdp` + 15 tooling binaries (removed `sdp-eval`). Added drift detection checks 5-7 to `scripts/check-release-surface.sh`. Documented isolation layers and packages-not-tagged rationale. |
