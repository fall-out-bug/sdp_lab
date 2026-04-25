# CLI Command Inventory — F137

> **Source of truth** for `cmd/sdp` subcommands and `cmd/sdp-*` standalone binaries.
> Produced by WS 00-137-01. Disposition decisions feed WS 00-137-02 (registry) and 00-137-04 (shims).
> **Date:** 2026-04-25

## cmd/sdp Subcommands (34)

| # | Subcommand | Source | Brief | Owner Collision |
|---|-----------|--------|-------|-----------------|
| 1 | `card` | cmd_card.go | Card lifecycle (create/show/clarify/ready/park/execute/heartbeat/feedback/resume/deliver) | — |
| 2 | `board` | cmd_board.go | Board visualization (build/show) | — |
| 3 | `doctor` | cmd_doctor.go | Health checks (control store validation) | — |
| 4 | `dispatch` | cmd_dispatch.go | Dispatch cards to executors (card/next) | — |
| 5 | `result` | cmd_result.go | Ingest executor results | — |
| 6 | `orchestrate` | cmd_orchestrate.go | Orchestration loop (once) | — |
| 7 | `attention` | cmd_query.go | Portfolio attention summary | — |
| 8 | `why` | cmd_query.go | Show why a card is blocked | — |
| 9 | `next` | cmd_query.go | Show next actionable items | — |
| 10 | `missing` | cmd_query.go | Show items lacking evidence | — |
| 11 | `approve` | cmd_query.go | Resolve human gates | — |
| 12 | `tower` | cmd_tower.go | Tower web UI server | — |
| 13 | `trace` | cmd_trace.go | Full feature trace with evidence | — |
| 14 | `deploy` | cmd_deploy.go | Deployment orchestration (staging/prod/rollback) | — |
| 15 | `intent` | cmd_pipeline.go | Create intake card from raw intent | **F125** intent pipeline |
| 16 | `status` | cmd_pipeline.go | Show card status and phase | **F125** intent pipeline |
| 17 | `stuck` | cmd_pipeline.go | Show stuck/long-running cards | **F125** intent pipeline |
| 18 | `eval` | cmd_pipeline.go | Run build evaluation manually | **F125** intent pipeline |
| 19 | `clarify` | cmd_pipeline.go | Run clarification manually | **F125** intent pipeline |
| 20 | `plan` | cmd_pipeline.go | Show plan for a card | **F125** intent pipeline |
| 21 | `approve-plan` | cmd_pipeline.go | Approve a pending plan | **F125** intent pipeline |
| 22 | `discover` | cmd_discover.go | Discovery pipeline (FRAME+SCAN+checkpoint) | **F125** intent model |
| 23 | `architect` | cmd_architect.go | Architecture analysis (analyze/c4/eval/render) | — |
| 24 | `scout` | cmd_scout.go | Codebase reconnaissance | — |
| 25 | `spec` | cmd_spec.go | Generate specs (api/rules/invariants/sla) | — |
| 26 | `metrics` | cmd_metrics.go | Collect code metrics | — |
| 27 | `index` | cmd_index.go | Codebase indexing (build/refresh/stats/manifest/query/deps/find/rank) | — |
| 28 | `bootstrap` | cmd_bootstrap.go | Initialize SDP in project | — |
| 29 | `rules` | cmd_rules.go | Update constraint rules | — |
| 30 | `build` | cmd_build.go | Execute one executable leaf workstream with TDD | — |
| 31 | `coverage-scan` | main_coverage.go | Coverage scanning with thresholds | — |
| 32 | `phase` | cmd_phase.go | Phase gates (plan/review/eval) | **F134** phase semantics |
| 33 | `reset` | cmd_reset.go | Reset checkpoint for a feature | — |

## cmd/sdp-* Standalone Binaries (21)

| # | Binary | Disposition | Target Subcommand | Rationale |
|---|--------|-------------|-------------------|-----------|
| 1 | `sdp-a2a` | out-of-scope | — | HTTP server (infrastructure service), not a CLI command |
| 2 | `sdp-beads-bridge` | shim | `sdp beads` | Beads integration, direct CLI surface mapping |
| 3 | `sdp-ci-loop` | shim | `sdp ci-loop` | CI loop orchestration, standalone for CI convenience only |
| 4 | `sdp-control` | retire | — | Already deprecated (emits warning), legacy control CLI |
| 5 | `sdp-dispatch` | shim | `sdp dispatch profile` | Dispatch profiling/benchmarking, extends existing `sdp dispatch` |
| 6 | `sdp-doc-sync` | shim | `sdp doc-sync` | Documentation consistency checker, direct surface mapping |
| 7 | `sdp-eval` | shim | `sdp skill-eval` | Skill evaluation (different from `sdp eval` which is build eval) |
| 8 | `sdp-evidence` | shim | `sdp evidence` | Evidence file validation, direct surface mapping |
| 9 | `sdp-gh-findings-sync` | shim | `sdp gh-sync` | GitHub findings sync, direct surface mapping |
| 10 | `sdp-guard` | shim | `sdp guard` | Pre-commit guard, direct surface mapping |
| 11 | `sdp-harness` | shim | `sdp harness` | Agent session lifecycle, direct surface mapping |
| 12 | `sdp-healthcheck` | shim | `sdp healthcheck` | Project health checks (related to but distinct from `sdp doctor`) |
| 13 | `sdp-mcp` | out-of-scope | — | MCP server (infrastructure service), **F126** territory |
| 14 | `sdp-omc-guard` | shim | `sdp omc-guard` | Pre-tool-call guard hook, direct surface mapping |
| 15 | `sdp-orchestrate` | shim | `sdp orchestrate daemon` | Pull-FSM orchestration daemon, extends `sdp orchestrate` |
| 16 | `sdp-orchestrate-daemon` | shim | `sdp orchestrate daemon` | Infinite cycle mode; merge with sdp-orchestrate into single subcommand |
| 17 | `sdp-protocol-check` | shim | `sdp protocol-check` | Protocol validation, direct surface mapping |
| 18 | `sdp-ready` | shim | `sdp ready` | Ready-to-work issues (related to but distinct from `sdp next`) |
| 19 | `sdp-session-audit` | shim | `sdp session-audit` | Session analysis, direct surface mapping |
| 20 | `sdp-strataudit` | shim | `sdp strataudit` | StratAudit verification, direct surface mapping |
| 21 | `sdp-up` | shim | `sdp up` | Environment provisioning, direct surface mapping |
| 22 | `sdp-ws-verdict-validate` | shim | `sdp ws-validate` | Verdict JSON validation, direct surface mapping |

## Disposition Summary

| Disposition | Count | Binaries |
|-------------|-------|----------|
| shim | 18 | sdp-beads-bridge, sdp-ci-loop, sdp-dispatch, sdp-doc-sync, sdp-eval, sdp-evidence, sdp-gh-findings-sync, sdp-guard, sdp-harness, sdp-healthcheck, sdp-omc-guard, sdp-orchestrate, sdp-orchestrate-daemon, sdp-protocol-check, sdp-ready, sdp-session-audit, sdp-strataudit, sdp-up, sdp-ws-verdict-validate |
| retire | 1 | sdp-control |
| out-of-scope | 2 | sdp-a2a (HTTP server), sdp-mcp (MCP server) |

## Ownership Collisions

| Feature | Commands | Collision Type | Resolution |
|---------|----------|----------------|------------|
| **F125** (intent model) | intent, status, stuck, eval, clarify, plan, approve-plan, discover | F125 owns intent pipeline routing; F137 must not absorb intent semantics | F137 registers these in the registry for discovery only; behavioral changes stay in F125 |
| **F126** (MCP server) | sdp-mcp | F126 owns MCP server; F137 must not rewrite MCP internals | sdp-mcp classified out-of-scope; F139 handles CLI-to-MCP parity later |
| **F134** (phase commands) | phase (plan/review/eval) | F134 owns phase semantics and runtime; F137 must not absorb phase logic | F137 registers `sdp phase` in registry for discovery only; phase behavior stays in F134 |

## Initial High-Value Commands for Registry Migration

Per WS 00-137-03 scope, the following 5 commands are the initial migration set onto the registry:

1. **scout** — codebase reconnaissance (downstream: mini-harness, sweep)
2. **architect** — architecture analysis (downstream: mini-harness)
3. **metrics** — code metrics collection (downstream: sweep)
4. **dispatch** — card dispatch to executors (downstream: mini-harness)
5. **orchestrate** — orchestration loop (downstream: mini-harness, sweep)

These were chosen because:
- mini-harness and sweep have hard dependencies on discovery and orchestration surface
- They cover the widest downstream surface with the fewest migrations
- They have no ownership collisions with F125, F126, or F134

## Naming Collisions Requiring Explicit Resolution

| Collision | cmd/sdp subcommand | cmd/sdp-* binary | Difference |
|-----------|--------------------|-------------------|------------|
| eval | `sdp eval <card>` (build evaluation via pipeline) | `sdp-eval --skill` (skill evaluation) | Different domain; `sdp-eval` → `sdp skill-eval` resolves |
| dispatch | `sdp dispatch card/next` (card dispatch) | `sdp-dispatch route/limits/profile/bench` (dispatch profiling) | Different domain; `sdp-dispatch` → `sdp dispatch profile` resolves |
| orchestrate | `sdp orchestrate once` (single loop) | `sdp-orchestrate` (pull-FSM daemon) + `sdp-orchestrate-daemon` (infinite cycle) | Same domain, different modes; merge into `sdp orchestrate` with sub-modes |
| healthcheck vs doctor | `sdp doctor control` (control store validation) | `sdp-healthcheck` (project health checks) | Related but distinct; both keep as separate subcommands |
| ready vs next | `sdp next` (next actionable items) | `sdp-ready` (ready-to-work with caching) | Related but distinct; `sdp-ready` → `sdp ready` (different semantics) |
