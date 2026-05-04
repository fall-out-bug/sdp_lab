# Changelog

## 2026-04-09

### Commits
- `fa83967` fix(review): concurrency, security, FSM logic fixes from deep code review
- `67c925d` fix(review): error handling, git hygiene, doc consistency

### Changed Files
- `cmd/sdp/cmd_dispatch.go` - dispatch command fixes
- `internal/beads/sql_client.go` - SQL client improvements
- `internal/discovery/artifacts.go` - artifact handling fixes
- `internal/discovery/llm.go` - LLM interaction fixes
- `internal/dispatch/beads_bridge.go` - Beads bridge fixes
- `internal/dispatch/route.go` - routing improvements
- `internal/executil/runner.go` - execution utility fixes
- `internal/executor/bridge.go` - executor bridge fixes
- `internal/executor/bridge_serve.go` - bridge server fixes
- `internal/executor/invoker_fallback.go` - fallback invoker fixes
- `internal/executor/omoclient/supervisor.go` - supervisor fixes
- `internal/gitutil/default_branch.go` - git utilities
- `internal/guard/scope_check.go` - guard scope checking
- `internal/orchestrate/advance.go` - orchestration advances
- `internal/orchestrate/attest.go` - attestation fixes
- `internal/orchestrate/dispatch_integration.go` - dispatch integration
- `internal/orchestrate/fsm_v2.go` - FSM v2 fixes
- `internal/orchestrate/hooks.go` - hooks system
- `internal/orchestrate/hydrate_sources.go` - source hydration
- `internal/orchestrate/invoke_opencode.go` - OpenCode invocation
- `internal/orchestrate/llm.go` - LLM orchestration
- `internal/orchestrate/state_machine.go` - state machine fixes
- `AGENTS.md` - documentation updates
- `README.md` - documentation updates
- `.gitignore` - ignore rules

### Summary
Deep code review addressing 24 findings across 6 categories:
concurrency safety, error handling, resource management, security,
API contract consistency, and test quality. All Critical and Important
issues resolved.

## 2026-03-01

### Commits
- `b8e9fe9` F067: Implement discrepancy detection for agent vs CI attestations (2026-03-01)

### Changed Files
- `docs/roadmap/ROADMAP.md`
- `docs/workstreams/INDEX.md`
- `docs/workstreams/backlog/00-066-01.md`
- `docs/workstreams/backlog/00-067-01.md`
- `internal/evidence/discrepancy.go`
- `internal/evidence/discrepancy_test.go`

## 2026-04-26

### Commits
- `193e33d` fix: Claude Code subagent tools format (lowercase object → comma-string) (#132) (2026-04-26)

### Changed Files
- `.claude/agents/architect.md`
- `.claude/agents/deployer.md`
- `.claude/agents/devops.md`
- `.claude/agents/implementer.md`
- `.claude/agents/orchestrator.md`
- `.claude/agents/planner.md`
- `.claude/agents/qa.md`
- `.claude/agents/reviewer.md`
- `.claude/agents/security.md`
- `.claude/agents/spec-reviewer.md`
- `.claude/agents/sre.md`
- `.claude/agents/tech-lead.md`
- `.sdp/generated/.claude/agents/architect.md`
- `.sdp/generated/.claude/agents/deployer.md`
- `.sdp/generated/.claude/agents/devops.md`
- `.sdp/generated/.claude/agents/implementer.md`
- `.sdp/generated/.claude/agents/orchestrator.md`
- `.sdp/generated/.claude/agents/planner.md`
- `.sdp/generated/.claude/agents/qa.md`
- `.sdp/generated/.claude/agents/reviewer.md`
- `.sdp/generated/.claude/agents/security.md`
- `.sdp/generated/.claude/agents/spec-reviewer.md`
- `.sdp/generated/.claude/agents/sre.md`
- `.sdp/generated/.claude/agents/tech-lead.md`
- `prompts/agents/README.md`
- `prompts/agents/architect.md`
- `prompts/agents/deployer.md`
- `prompts/agents/devops.md`
- `prompts/agents/implementer.md`
- `prompts/agents/orchestrator.md`
- `prompts/agents/planner.md`
- `prompts/agents/qa.md`
- `prompts/agents/reviewer.md`
- `prompts/agents/security.md`
- `prompts/agents/spec-reviewer.md`
- `prompts/agents/sre.md`
- `prompts/agents/tech-lead.md`



### Commits
- `b5b77ad` merge: mark PR 131 squash merge (2026-04-26)

### Changed Files



### Commits
- `ed393e3` feat(F144): inference confidence and quality control (2026-04-26)

### Changed Files
- `.beads/interactions.jsonl`
- `cmd/sdp-confidence-replay/main.go`
- `docs/plans/2026-04-26-f144-inference-confidence-design.md`
- `docs/research/2026-04-26-f144-confidence-replay-report.md`
- `internal/inference/confidence/README.md`
- `internal/inference/confidence/adapters/architect/architect.go`
- `internal/inference/confidence/adapters/architect/architect_test.go`
- `internal/inference/confidence/adapters/dispatch/dispatch.go`
- `internal/inference/confidence/adapters/dispatch/dispatch_test.go`
- `internal/inference/confidence/adapters/wsverdict/wsverdict.go`
- `internal/inference/confidence/adapters/wsverdict/wsverdict_test.go`
- `internal/inference/confidence/budget_test.go`
- `internal/inference/confidence/checker.go`
- `internal/inference/confidence/checker_test.go`
- `internal/inference/confidence/constraint/README.md`
- `internal/inference/confidence/constraint/strategy.go`
- `internal/inference/confidence/constraint/strategy_test.go`
- `internal/inference/confidence/nsample/README.md`
- `internal/inference/confidence/nsample/strategy.go`
- `internal/inference/confidence/nsample/strategy_test.go`
- `internal/inference/confidence/policy.go`
- `internal/inference/confidence/policy_test.go`
- `internal/inference/confidence/replay/percentile_test.go`
- `internal/inference/confidence/replay/replay.go`
- `internal/inference/confidence/replay/replay_test.go`
- `internal/inference/confidence/result.go`
- `internal/inference/confidence/result_test.go`
- `internal/inference/confidence/selfcheck/README.md`
- `internal/inference/confidence/selfcheck/strategy.go`
- `internal/inference/confidence/selfcheck/strategy_test.go`
- `internal/inference/confidence/status.go`
- `internal/inference/confidence/strategy.go`
- `internal/inference/confidence/testdata/ws-verdict/adversarial/garbage.json`
- `internal/inference/confidence/testdata/ws-verdict/adversarial/malformed-id.json`
- `internal/inference/confidence/testdata/ws-verdict/adversarial/pass-with-failed-tests.json`
- `internal/inference/confidence/testdata/ws-verdict/adversarial/unmet-ac-but-pass.json`
- `internal/inference/confidence/testdata/ws-verdict/correct/clean-fail.json`
- `internal/inference/confidence/testdata/ws-verdict/correct/clean-pass.json`
- `internal/inference/confidence/testdata/ws-verdict/edge/empty-evidence.json`
- `internal/inference/confidence/testdata/ws-verdict/edge/partial-coverage.json`



### Commits
- `e760bdd` merge: fix coverage baseline update after PR 130 (2026-04-26)

### Changed Files
- `.github/workflows/ci.yml`
- `.sdp/metrics/coverage.txt`



### Commits
- `9e329da` merge: mark PR 130 squash merge (2026-04-26)

### Changed Files



### Commits
- `1a5e1e9` feat(F133): Local model dispatch — route tasks to Ollama (#127) (2026-04-26)

### Changed Files
- `.beads/interactions.jsonl`
- `cmd/sdp/cmd_dispatch.go`
- `configs/dispatch.example.yaml`
- `internal/dispatch/classify.go`
- `internal/dispatch/classify_test.go`
- `internal/dispatch/local_ollama.go`
- `internal/dispatch/local_ollama_test.go`
- `internal/orchestrate/dispatch_integration.go`



### Commits
- `3f06f9d` fix(delivery-loop): forbid agent from improvising Phase 3 codex skip (#129) (2026-04-26)

### Changed Files
- `.codex/skills/delivery-loop.md`
- `.opencode/skill/delivery-loop.md`
- `.sdp/generated/.codex/skills/delivery-loop.md`
- `.sdp/generated/.opencode/skill/delivery-loop.md`
- `prompts/skills/delivery-loop/SKILL.md`



### Commits
- `f8970fc` fix(F143-02): unbreak CI on main — adapter drift + 45 errcheck + doc-sync (#128) (2026-04-26)

### Changed Files
- `.codex/skills/build.md`
- `.github/workflows/ci.yml`
- `.golangci.yml`
- `.opencode/skill/build.md`
- `.sdp/generated/.codex/skills/build.md`
- `.sdp/generated/.codex/skills/discovery.md`
- `.sdp/generated/.codex/skills/idea.md`
- `.sdp/generated/.codex/skills/prototype.md`
- `.sdp/generated/.codex/skills/reality.md`
- `.sdp/generated/.codex/skills/tdd.md`
- `.sdp/generated/.codex/skills/ux.md`
- `.sdp/generated/.opencode/skill/build.md`
- `.sdp/generated/.opencode/skill/discovery.md`
- `.sdp/generated/.opencode/skill/idea.md`
- `.sdp/generated/.opencode/skill/prototype.md`
- `.sdp/generated/.opencode/skill/reality.md`
- `.sdp/generated/.opencode/skill/tdd.md`
- `.sdp/generated/.opencode/skill/ux.md`
- `cmd/sdp-harness/main_integration_test.go`
- `cmd/sdp/cmd_architect_render.go`
- `cmd/sdp/cmd_bootstrap_test.go`
- `cmd/sdp/cmd_metrics_test.go`
- `cmd/sdp/cmd_phase_from_run.go`
- `cmd/sdp/cmd_phase_test.go`
- `cmd/sdp/cmd_reset.go`
- `cmd/sdp/cmd_reset_test.go`
- `docs/workstreams/backlog/00-133-01.md`
- `docs/workstreams/backlog/00-135-01.md`
- `docs/workstreams/backlog/00-140-01.md`
- `docs/workstreams/backlog/00-140-02.md`
- `docs/workstreams/backlog/00-140-03.md`
- `docs/workstreams/backlog/00-140-04.md`
- `docs/workstreams/backlog/00-140-05.md`
- `internal/agentloop/harness.go`
- `internal/agentloop/store_sqlite.go`
- `internal/architect/c4/mermaid_test.go`
- `internal/architect/crosslang.go`
- `internal/architect/enricher.go`
- `internal/architect/eval/metrics_test.go`
- `internal/architect/resilience_test.go`
- `internal/bootstrap/hooks_test.go`
- `internal/bootstrap/policy.go`
- `internal/build/docker_sandbox_test.go`
- `internal/build/pipeline.go`
- `internal/build/pipeline_test.go`
- `internal/execloop/loop.go`
- `internal/executor/bridge_serve_harness_test.go`
- `internal/index/manifest.go`
- `internal/localmodel/client_test.go`
- `internal/promote/promote_integration_test.go`
- `internal/promote/promote_test.go`
- `internal/scout/conventions.go`
- `internal/skills/augment_test.go`
- `internal/spec/config_parse.go`
- `internal/spec/coverage.go`
- `internal/spec/diff_test.go`
- `internal/spec/invariant_extract.go`
- `internal/spec/sla_extract.go`
- `internal/spec/spec.go`
- `tests/architect/classify_test.go`

## 2026-04-27

### Commits
- `8901a89` fix(F150-06): make maturity-tiered coverage gate advisory (non-blocking) (2026-04-27)

### Changed Files
- `.beads/issues.jsonl`



### Commits
- `d13280a` feat(F154): Shared Substrates v1 — semver contracts + SDP-runtime assumption docs (#141) (2026-04-27)

### Changed Files
- `internal/context/AGENTS.md`
- `internal/context/contract.go`
- `internal/context/contract_test.go`
- `internal/eval/AGENTS.md`
- `internal/eval/contract.go`
- `internal/eval/contract_test.go`
- `internal/evidence/AGENTS.md`
- `internal/evidence/contract.go`
- `internal/evidence/contract_test.go`
- `internal/modelgateway/AGENTS.md`
- `internal/modelgateway/contract.go`
- `internal/modelgateway/contract_test.go`
- `internal/policy/AGENTS.md`
- `internal/policy/contract.go`
- `internal/policy/contract_test.go`



### Commits
- `93bc759` docs(F152-F160): add missing workstream files for post-F150 epics (2026-04-27)

### Changed Files
- `.beads/issues.jsonl`
- `docs/workstreams/backlog/00-152-01.md`
- `docs/workstreams/backlog/00-152-02.md`
- `docs/workstreams/backlog/00-152-03.md`
- `docs/workstreams/backlog/00-153-01.md`
- `docs/workstreams/backlog/00-153-02.md`
- `docs/workstreams/backlog/00-153-03.md`
- `docs/workstreams/backlog/00-154-01.md`
- `docs/workstreams/backlog/00-154-02.md`
- `docs/workstreams/backlog/00-154-03.md`
- `docs/workstreams/backlog/00-154-04.md`
- `docs/workstreams/backlog/00-154-05.md`
- `docs/workstreams/backlog/00-155-01.md`
- `docs/workstreams/backlog/00-156-01.md`
- `docs/workstreams/backlog/00-157-01.md`
- `docs/workstreams/backlog/00-158-01.md`
- `docs/workstreams/backlog/00-159-01.md`
- `docs/workstreams/backlog/00-160-01.md`



### Commits
- `3ca6606` chore(F151): close beads after PR #139 merge (2026-04-27)

### Changed Files
- `.beads/interactions.jsonl`
- `.beads/issues.jsonl`
- `.sdp/checkpoints/F151.json`



### Commits
- `cba6482` docs(F137): cmd-inventory + sdp-cli.md reference + workstream status sync (#138) (2026-04-27)

### Changed Files
- `docs/reference/cmd-inventory.md`
- `docs/reference/sdp-cli.md`
- `docs/workstreams/backlog/00-137-01.md`
- `docs/workstreams/backlog/00-137-02.md`
- `docs/workstreams/backlog/00-137-03.md`
- `docs/workstreams/backlog/00-137-04.md`
- `docs/workstreams/backlog/00-137-05.md`



### Commits
- `39bc220` F150: product layering and release readiness (#137) (2026-04-27)

### Changed Files
- `.beads-sdp-mapping.jsonl`
- `.beads/issues.jsonl`
- `docs/plans/2026-04-27-f150-product-layering-release-readiness-design.md`
- `docs/roadmap/2026-04-27-roadmap-v3-post-iip.md`
- `docs/roadmap/ROADMAP.md`
- `docs/strategy/2026-04-27-sdp-product-layering-4d.md`
- `docs/strategy/council/2026-04-27-iip/create_beads.py`
- `docs/strategy/council/2026-04-27-iip/r1-architect.md`
- `docs/strategy/council/2026-04-27-iip/r1-critic.md`
- `docs/strategy/council/2026-04-27-iip/r1-philosopher.md`
- `docs/strategy/council/2026-04-27-iip/r1-pragmatist.md`
- `docs/strategy/council/2026-04-27-iip/r1-raw.json`
- `docs/strategy/council/2026-04-27-iip/r1-technician.md`
- `docs/strategy/council/2026-04-27-iip/r2-architect.md`
- `docs/strategy/council/2026-04-27-iip/r2-critic.md`
- `docs/strategy/council/2026-04-27-iip/r2-philosopher.md`
- `docs/strategy/council/2026-04-27-iip/r2-pragmatist.md`
- `docs/strategy/council/2026-04-27-iip/r2-raw.json`
- `docs/strategy/council/2026-04-27-iip/r2-technician.md`
- `docs/strategy/council/2026-04-27-iip/run.py`
- `docs/strategy/council/2026-04-27-iip/synthesis.md`
- `docs/strategy/council/2026-04-27/r1-architect.md`
- `docs/strategy/council/2026-04-27/r1-critic.md`
- `docs/strategy/council/2026-04-27/r1-philosopher.md`
- `docs/strategy/council/2026-04-27/r1-pragmatist.md`
- `docs/strategy/council/2026-04-27/r1-raw.json`
- `docs/strategy/council/2026-04-27/r1-technician.md`
- `docs/strategy/council/2026-04-27/r2-architect.md`
- `docs/strategy/council/2026-04-27/r2-critic.md`
- `docs/strategy/council/2026-04-27/r2-philosopher.md`
- `docs/strategy/council/2026-04-27/r2-pragmatist.md`
- `docs/strategy/council/2026-04-27/r2-raw.json`
- `docs/strategy/council/2026-04-27/r2-technician.md`
- `docs/strategy/council/2026-04-27/retry_technician.py`
- `docs/strategy/council/2026-04-27/run.py`
- `docs/strategy/council/2026-04-27/synthesis.md`
- `docs/workstreams/INDEX.md`
- `docs/workstreams/backlog/00-150-01.md`
- `docs/workstreams/backlog/00-150-02.md`
- `docs/workstreams/backlog/00-150-03.md`
- `docs/workstreams/backlog/00-150-04.md`
- `docs/workstreams/backlog/00-150-05.md`
- `docs/workstreams/backlog/00-150-06.md`
- `docs/workstreams/backlog/00-150-07.md`
- `docs/workstreams/backlog/00-150-08.md`
- `docs/workstreams/backlog/00-150-09.md`
- `docs/workstreams/backlog/00-150-10.md`
- `internal/dispatch/harness/limits_cache.go`



### Commits
- `45337ac` feat(F146): Inference Decomposition Framework — Pipeline[Final], Stage[In,Out], stitchers, ws-verdict adapter, A/B bench (2026-04-27)

### Changed Files
- `.beads-sdp-mapping.jsonl`
- `.sdp/log/events.jsonl`
- `cmd/sdp-decompose-bench/clients.go`
- `cmd/sdp-decompose-bench/main.go`
- `cmd/sdp-decompose-bench/report.go`
- `cmd/sdp-decompose-bench/runner.go`
- `docs/research/2026-04-26-f146-decomposition-replay-report.md`
- `docs/workstreams/INDEX.md`
- `internal/build/.sdp/evidence/20260426-214905/evidence.json`
- `internal/build/.sdp/evidence/20260426-214905/results.csv`
- `internal/build/.sdp/evidence/20260426-214938/evidence.json`
- `internal/build/.sdp/evidence/20260426-214938/results.csv`
- `internal/inference/decompose/README.md`
- `internal/inference/decompose/adapters/wsverdict/aggregate.go`
- `internal/inference/decompose/adapters/wsverdict/classify.go`
- `internal/inference/decompose/adapters/wsverdict/extract.go`
- `internal/inference/decompose/adapters/wsverdict/monolithic.go`
- `internal/inference/decompose/integration.go`
- `internal/inference/decompose/result.go`
- `internal/inference/decompose/stitcher_json.go`
- `internal/inference/decompose/stitcher_test.go`
- `internal/inference/decompose/stitcher_toon.go`
- `internal/inference/replayutil/evidence.go`
- `internal/inference/replayutil/loader.go`
- `internal/inference/replayutil/loader_test.go`



### Commits
- `7adbae1` chore(beads): close F145 children + epic (14 WS + sdplab-ldmq) (2026-04-27)

### Changed Files
- `.beads/interactions.jsonl`

## 2026-04-28

### Commits
- `f1050c9` F150: Product layering and release readiness (#142) (2026-04-29)

### Changed Files
- `.claude/commands/submit-to-swarm.md`
- `.github/workflows/ci.yml`
- `.github/workflows/release.yml`
- `.gitignore`
- `.goreleaser.yml`
- `.sdp/generated/.claude/commands/submit-to-swarm.md`
- `.sdp/generated/.cursor/rules/submit-to-swarm.mdc`
- `AGENTS.md`
- `CONTRIBUTING.md`
- `Makefile`
- `README.md`
- `archive/adapters-sdk/examples/main.go`
- `cmd/gt-adapter/main.go`
- `cmd/sdp-a2a/main.go`
- `cmd/sdp-bd-suggest/main.go`
- `cmd/sdp-bd-suggest/main_test.go`
- `cmd/sdp-bd-suggest/render.go`
- `cmd/sdp-beads-bridge/main.go`
- `cmd/sdp-beads-bridge/query.go`
- `cmd/sdp-cascade-replay/main.go`
- `cmd/sdp-ci-loop/main.go`
- `cmd/sdp-confidence-replay/main.go`
- `cmd/sdp-control/main.go`
- `cmd/sdp-decompose-bench/clients.go`
- `cmd/sdp-decompose-bench/main.go`
- `cmd/sdp-decompose-bench/report.go`
- `cmd/sdp-decompose-bench/runner.go`
- `cmd/sdp-dispatch/cmd_bench.go`
- `cmd/sdp-dispatch/cmd_bench_test.go`
- `cmd/sdp-dispatch/cmd_compare.go`
- `cmd/sdp-dispatch/cmd_compare_test.go`
- `cmd/sdp-dispatch/cmd_limits.go`
- `cmd/sdp-dispatch/cmd_profile.go`
- `cmd/sdp-dispatch/cmd_route.go`
- `cmd/sdp-dispatch/cmd_status.go`
- `cmd/sdp-dispatch/cmd_status_test.go`
- `cmd/sdp-dispatch/helpers.go`
- `cmd/sdp-dispatch/main.go`
- `cmd/sdp-doc-sync/main.go`
- `cmd/sdp-eval/main.go`
- `cmd/sdp-eval/main_test.go`
- `cmd/sdp-evidence/main.go`
- `cmd/sdp-export/main.go`
- `cmd/sdp-export/main_test.go`
- `cmd/sdp-ft-baseline/main.go`
- `cmd/sdp-ft-dataset/main.go`
- `cmd/sdp-ft-run/main.go`
- `cmd/sdp-ft-validate/main.go`
- `cmd/sdp-gh-findings-sync/main.go`
- `cmd/sdp-guard/main.go`
- `cmd/sdp-guard/main_contract.go`
- `cmd/sdp-harness/main.go`
- `cmd/sdp-harness/main_integration_test.go`
- `cmd/sdp-harness/main_test.go`
- `cmd/sdp-harness/test_helpers_test.go`
- `cmd/sdp-healthcheck/main.go`
- `cmd/sdp-mcp/main.go`
- `cmd/sdp-microfirst-bench/main.go`
- `cmd/sdp-microfirst-bench/main_test.go`
- `cmd/sdp-omc-guard/main.go`
- `cmd/sdp-orchestrate-daemon/main.go`
- `cmd/sdp-orchestrate/main.go`
- `cmd/sdp-orchestrate/main_advance.go`
- `cmd/sdp-orchestrate/main_hydrate.go`
- `cmd/sdp-orchestrate/main_nextaction.go`
- `cmd/sdp-orchestrate/main_repair.go`
- `cmd/sdp-orchestrate/main_status.go`
- `cmd/sdp-orchestrate/main_test.go`
- `cmd/sdp-protocol-check/main.go`
- `cmd/sdp-ready/main.go`
- `cmd/sdp-ready/main_test.go`
- `cmd/sdp-session-audit/main.go`
- `cmd/sdp-strataudit/main.go`
- `cmd/sdp-up/main.go`
- `cmd/sdp-watch/main_test.go`
- `cmd/sdp/architect_analyze_test.go`
- `cmd/sdp/checkpoint_c.go`
- `cmd/sdp/checkpoint_c_test.go`
- `cmd/sdp/cmd_architect.go`
- `cmd/sdp/cmd_board.go`
- `cmd/sdp/cmd_bootstrap.go`
- `cmd/sdp/cmd_build.go`
- `cmd/sdp/cmd_card.go`
- `cmd/sdp/cmd_deploy.go`
- `cmd/sdp/cmd_discover.go`
- `cmd/sdp/cmd_discover_test.go`
- `cmd/sdp/cmd_dispatch.go`
- `cmd/sdp/cmd_doctor.go`
- `cmd/sdp/cmd_doctor_backlog.go`
- `cmd/sdp/cmd_generate_adapters.go`
- `cmd/sdp/cmd_index.go`
- `cmd/sdp/cmd_init.go`
- `cmd/sdp/cmd_manifest.go`
- `cmd/sdp/cmd_metrics.go`
- `cmd/sdp/cmd_metrics_test.go`
- `cmd/sdp/cmd_orchestrate.go`
- `cmd/sdp/cmd_phase.go`
- `cmd/sdp/cmd_phase_from_run.go`
- `cmd/sdp/cmd_phase_test.go`
- `cmd/sdp/cmd_pipeline.go`
- `cmd/sdp/cmd_query.go`
- `cmd/sdp/cmd_reset.go`
- `cmd/sdp/cmd_result.go`
- `cmd/sdp/cmd_rules.go`
- `cmd/sdp/cmd_rules_test.go`
- `cmd/sdp/cmd_scout.go`
- `cmd/sdp/cmd_skills_augment.go`
- `cmd/sdp/cmd_skills_update.go`
- `cmd/sdp/cmd_spec.go`
- `cmd/sdp/cmd_telemetry.go`
- `cmd/sdp/cmd_tower.go`
- `cmd/sdp/helpers.go`
- `cmd/sdp/main_coverage.go`
- `cmd/sdp/main_coverage_test.go`
- `cmd/sdp/snapshot_test.go`
- `docs/MANIFESTO.md`
- `docs/MULTI-REPO-WORKFLOW.md`
- `docs/QUICKSTART.md`
- `docs/architecture/REPO-BOUNDARY.md`
- `docs/evidence/F150-08-homebrew-dry-run.md`
- `docs/getting-started.md`
- `docs/guides/adapters/openspec.md`
- `docs/plans/2026-04-08-discovery-pipeline-impl.md`
- `docs/plans/2026-04-11-council-sdp-full.md`
- `docs/plans/2026-04-11-f108-architecture-normalization-plan.md`
- `docs/policy/DECISION_EXPLAINABILITY.md`
- `docs/reference/ci-gates-map.md`
- `docs/reference/go-patterns.md`
- `docs/reference/maturity-matrix.md`
- `docs/reference/product-surface.md`
- `docs/reference/project-map.md`
- `docs/reference/release-readiness.md`
- `docs/reference/sdp-mcp.md`
- `docs/reference/telemetry-schema.md`
- `docs/reviews/2026-02-25-F054-work-and-repo-review.md`
- `docs/reviews/2026-04-27-dependency-audit.md`
- `docs/roadmap/ROADMAP.md`
- `docs/workstreams/INDEX.md`
- `formula/sdp.rb`
- `go.mod`
- `internal/a2a/server.go`
- `internal/a2a/server_test.go`
- `internal/adapters/drift_test.go`
- `internal/adapters/generate.go`
- `internal/adapters/generate_test.go`
- `internal/adapters/sdk/examples/beads_adapter.go`
- `internal/adapters/sdk/examples/gas_town_adapter.go`
- `internal/adapters/sdk/examples/main.go`
- `internal/adapters/sdk/examples/omo_adapter.go`
- `internal/agentloop/evidence.go`
- `internal/agentloop/gate.go`
- `internal/agentloop/gate_test.go`
- `internal/agentloop/harness_test.go`
- `internal/agentloop/livegw/livegw.go`
- `internal/agentloop/livegw/livegw_test.go`
- `internal/agentloop/session.go`
- `internal/agentloop/tools_live.go`
- `internal/agentloop/types.go`
- `internal/agentloop/types_compile_test.go`
- `internal/architect/c4/confidence.go`
- `internal/architect/c4/export.go`
- `internal/architect/c4/generator.go`
- `internal/architect/c4/generator_test.go`
- `internal/architect/c4/level1.go`
- `internal/architect/c4/level2.go`
- `internal/architect/c4/level3.go`
- `internal/architect/c4/level_test.go`
- `internal/architect/c4/mermaid.go`
- `internal/architect/c4/mermaid_test.go`
- `internal/architect/c4/relationship.go`
- `internal/architect/c4/renderer.go`
- `internal/architect/c4/renderer_test.go`
- `internal/architect/classify/hypothesizer.go`
- `internal/architect/eval/diff.go`
- `internal/architect/eval/eval.go`
- `internal/architect/eval/eval_test.go`
- `internal/architect/eval/golden.go`
- `internal/architect/eval/golden_test.go`
- `internal/architect/eval/integration_test.go`
- `internal/architect/eval/metrics.go`
- `internal/architect/eval/metrics_test.go`
- `internal/architect/eval/mock.go`
- `internal/architect/eval/mock_test.go`
- `internal/architect/extract/adapters.go`
- `internal/architect/extract/deps.go`
- `internal/architect/extract/filetree.go`
- `internal/architect/extract/generated.go`
- `internal/architect/extract/git_extract.go`
- `internal/architect/extract/infra.go`
- `internal/architect/extract/java_extract.go`
- `internal/architect/extract/python/README.md`
- `internal/architect/extract/python/extractor.go`
- `internal/architect/extract/python_extract.go`
- `internal/architect/extract/registry.go`
- `internal/architect/extract/specs.go`
- `internal/architect/extract/sql/ddl.go`
- `internal/architect/extract/sql/ddl_test.go`
- `internal/architect/extract/sql/domain.go`
- `internal/architect/extract/sql/extractor.go`
- `internal/architect/extract/sql/migration.go`
- `internal/architect/extract/sql/orm.go`
- `internal/architect/extract/sql/pii.go`
- `internal/architect/extract/sql/pii_test.go`
- `internal/architect/extract/sql_extract.go`
- `internal/architect/extract/ts_extract.go`
- `internal/architect/extract/typescript/converter.go`
- `internal/architect/extract/typescript/extractor.go`
- `internal/architect/llm/client.go`
- `internal/architect/llm/client_test.go`
- `internal/architect/llm/integration_test.go`
- `internal/architect/llm/prompt_hypothesizer.go`
- `internal/architect/llm/prompt_patterns.go`
- `internal/architect/llm/prompt_risks.go`
- `internal/architect/llm_client.go`
- `internal/architect/security_wrapper.go`
- `internal/augmentation/context.go`
- `internal/augmentation/defaults.go`
- `internal/augmentation/hooks.go`
- `internal/augmentation/hooks_test.go`
- `internal/augmentation/loader.go`
- `internal/augmentation/loader_test.go`
- `internal/augmentation/roles.go`
- `internal/backlog/audit_test.go`
- `internal/bootstrap/brownfield.go`
- `internal/bootstrap/brownfield_test.go`
- `internal/bootstrap/roadmap.go`
- `internal/bootstrap/roadmap_test.go`
- `internal/ciloop/autofixer_test.go`
- `internal/ciloop/checkpoint.go`
- `internal/ciloop/checkpoint_test.go`
- `internal/ciloop/classifier_test.go`
- `internal/ciloop/deterministic_fixer_test.go`
- `internal/ciloop/fixer.go`
- `internal/ciloop/fixer_test.go`
- `internal/ciloop/loop_test.go`
- `internal/ciloop/poller.go`
- `internal/ciloop/poller_test.go`
- `internal/ciloop/runfile.go`
- `internal/ciloop/runfile_test.go`
- `internal/cli/control_tower_boards.go`
- `internal/cli/control_tower_card.go`
- `internal/cli/control_tower_commands.go`
- `internal/cli/control_tower_doctor.go`
- `internal/cli/control_tower_view_test.go`
- `internal/control/artifacts.go`
- `internal/control/contract_gen.go`
- `internal/control/contract_gen_test.go`
- `internal/deploy/deploy.go`
- `internal/discovery/artifacts_test.go`
- `internal/discovery/checkpoint_test.go`
- `internal/discovery/clarify_test.go`
- `internal/discovery/depth_test.go`
- `internal/discovery/experiment_test.go`
- `internal/discovery/frame_test.go`
- `internal/discovery/hypothesize_test.go`
- `internal/discovery/llm.go`
- `internal/discovery/llm_test.go`
- `internal/discovery/scan_resolution_test.go`
- `internal/discovery/scan_test.go`
- `internal/discovery/validate_test.go`
- `internal/dispatch/beads_bridge.go`
- `internal/dispatch/cascade/cascade.go`
- `internal/dispatch/cascade/cascade_confidence_test.go`
- `internal/dispatch/cascade/cascade_test.go`
- `internal/dispatch/cascade/confidence_adapter.go`
- `internal/dispatch/cascade/confidence_adapter_test.go`
- `internal/dispatch/cascade/replay.go`
- `internal/dispatch/cascade/replay_test.go`
- `internal/dispatch/cascade/smoke_test.go`
- `internal/dispatch/cascade/types.go`
- `internal/dispatch/conventions_test.go`
- `internal/dispatch/harness/harness_test.go`
- `internal/dispatch/harness/providers/anthropic.go`
- `internal/dispatch/harness/providers/anthropic_test.go`
- `internal/dispatch/harness/providers/cursor.go`
- `internal/dispatch/harness/providers/kimi.go`
- `internal/dispatch/harness/providers/kimi_test.go`
- `internal/dispatch/harness/providers/ollama.go`
- `internal/dispatch/harness/providers/ollama_test.go`
- `internal/dispatch/harness/providers/openai.go`
- `internal/dispatch/harness/providers/openai_test.go`
- `internal/dispatch/harness/providers/providers.go`
- `internal/dispatch/harness/providers/providers_test.go`
- `internal/dispatch/invoker.go`
- `internal/dispatch/invoker_test.go`
- `internal/dispatch/limits.go`
- `internal/dispatch/limits_test.go`
- `internal/dispatch/ollama_client_test.go`
- `internal/dispatch/route.go`
- `internal/dispatch/route_test.go`
- `internal/dispatch/staleness_test.go`
- `internal/dispatch/verify.go`
- `internal/dispatch/verify_test.go`
- `internal/docsync/docsync.go`
- `internal/evals/fixtures.go`
- `internal/evals/runner.go`
- `internal/evals/runner_test.go`
- `internal/evidence/auto_attest.go`
- `internal/evidence/cmd/auto-attest/main.go`
- `internal/evidence/cmd/auto-attest/main_test.go`
- `internal/evidence/trace_validator.go`
- `internal/execloop/loop.go`
- `internal/execloop/loop_test.go`
- `internal/executor/bridge.go`
- `internal/executor/bridge_serve.go`
- `internal/executor/bridge_serve_harness_test.go`
- `internal/executor/bridge_serve_integration_test.go`
- `internal/executor/bridge_test.go`
- `internal/executor/card_validation.go`
- `internal/executor/card_validation_test.go`
- `internal/executor/clarifier.go`
- `internal/executor/clarifier_test.go`
- `internal/executor/clarify_prompt.go`
- `internal/executor/discovery_runner.go`
- `internal/executor/discovery_runner_test.go`
- `internal/executor/eval_prompt.go`
- `internal/executor/evaluator.go`
- `internal/executor/evaluator_test.go`
- `internal/executor/findings.go`
- `internal/executor/findings_test.go`
- `internal/executor/invoker_fallback.go`
- `internal/executor/loop.go`
- `internal/executor/loop_test.go`
- `internal/executor/loop_v2.go`
- `internal/executor/omoclient/adapter.go`
- `internal/executor/omoclient/outofscope.go`
- `internal/executor/plan_prompt.go`
- `internal/executor/planner.go`
- `internal/executor/planner_test.go`
- `internal/executor/provenance.go`
- `internal/executor/provenance_test.go`
- `internal/executor/ranking.go`
- `internal/executor/ranking_test.go`
- `internal/executor/roles.go`
- `internal/executor/summarizer.go`
- `internal/executor/summarizer_test.go`
- `internal/gitutil/default_branch.go`
- `internal/guard/scope_check.go`
- `internal/guard/scope_check_test.go`
- `internal/harness/io.go`
- `internal/harnessadapter/adapter.go`
- `internal/harnessadapter/adapter_test.go`
- `internal/harnessadapter/agents.go`
- `internal/harnessadapter/claude.go`
- `internal/harnessadapter/cursor.go`
- `internal/harnesscfg/manifest_test.go`
- `internal/healthcheck/checks_test.go`
- `internal/healthcheck/healthcheck_test.go`
- `internal/index/builder.go`
- `internal/index/parser_test.go`
- `internal/inference/confidence/README.md`
- `internal/inference/confidence/adapters/architect/architect.go`
- `internal/inference/confidence/adapters/architect/architect_test.go`
- `internal/inference/confidence/adapters/dispatch/dispatch.go`
- `internal/inference/confidence/adapters/dispatch/dispatch_test.go`
- `internal/inference/confidence/adapters/wsverdict/wsverdict.go`
- `internal/inference/confidence/adapters/wsverdict/wsverdict_test.go`
- `internal/inference/confidence/budget_test.go`
- `internal/inference/confidence/checker_test.go`
- `internal/inference/confidence/constraint/strategy.go`
- `internal/inference/confidence/constraint/strategy_test.go`
- `internal/inference/confidence/nsample/strategy.go`
- `internal/inference/confidence/nsample/strategy_test.go`
- `internal/inference/confidence/policy_test.go`
- `internal/inference/confidence/replay/replay.go`
- `internal/inference/confidence/replay/replay_test.go`
- `internal/inference/confidence/result_test.go`
- `internal/inference/confidence/selfcheck/strategy.go`
- `internal/inference/confidence/selfcheck/strategy_test.go`
- `internal/inference/decompose/adapters/wsverdict/aggregate.go`
- `internal/inference/decompose/adapters/wsverdict/classify.go`
- `internal/inference/decompose/adapters/wsverdict/extract.go`
- `internal/inference/decompose/adapters/wsverdict/monolithic.go`
- `internal/inference/decompose/adapters/wsverdict/pipeline.go`
- `internal/inference/decompose/adapters/wsverdict/pipeline_test.go`
- `internal/inference/decompose/adapters/wsverdict/wsverdict_test.go`
- `internal/inference/decompose/confidence_runner_test.go`
- `internal/inference/decompose/escalation_test.go`
- `internal/inference/decompose/integration.go`
- `internal/inference/decompose/integration_test.go`
- `internal/inference/decompose/pipeline_test.go`
- `internal/inference/decompose/stage_test.go`
- `internal/inference/decompose/stitcher_bench_test.go`
- `internal/inference/decompose/stitcher_test.go`
- `internal/inference/microfirst/bdseverity/classifier.go`
- `internal/inference/microfirst/bdseverity/classifier_test.go`
- `internal/inference/microfirst/bdseverity/result.go`
- `internal/inference/microfirst/bdtype/classifier.go`
- `internal/inference/microfirst/bdtype/classifier_test.go`
- `internal/inference/microfirst/bdtype/result.go`
- `internal/inference/microfirst/knn/classify.go`
- `internal/inference/microfirst/knn/knn_test.go`
- `internal/inference/microfirst/routing/classifier.go`
- `internal/inference/microfirst/routing/classifier_test.go`
- `internal/inference/microfirst/wsverdict/classifier.go`
- `internal/inference/microfirst/wsverdict/classifier_test.go`
- `internal/inference/microfirst/wsverdict/result.go`
- `internal/inference/replayutil/loader_test.go`
- `internal/llmclient/llmclient_test.go`
- `internal/localmodel/client_test.go`
- `internal/manifest/load_test.go`
- `internal/manifest/parity_test.go`
- `internal/mcp/contract/mapping_test.go`
- `internal/mcp/parity/prompts_test.go`
- `internal/mcp/parity/resources_test.go`
- `internal/mcp/registry/discovery_test.go`
- `internal/mcp/validation/handshake.go`
- `internal/mcp/validation/handshake_test.go`
- `internal/modelgateway/adapters/adapters_test.go`
- `internal/modelgateway/adapters/anthropic.go`
- `internal/modelgateway/adapters/openai.go`
- `internal/modelgateway/adapters/selfhosted.go`
- `internal/modelgateway/provider.go`
- `internal/modelgateway/router.go`
- `internal/orchestrate/advance.go`
- `internal/orchestrate/advance_test.go`
- `internal/orchestrate/attest.go`
- `internal/orchestrate/attest_test.go`
- `internal/orchestrate/autonomous_test.go`
- `internal/orchestrate/checkpoint.go`
- `internal/orchestrate/checkpoint_integrity_test.go`
- `internal/orchestrate/checkpoint_test.go`
- `internal/orchestrate/cli.go`
- `internal/orchestrate/cli_test.go`
- `internal/orchestrate/contract_gate.go`
- `internal/orchestrate/contract_gate_test.go`
- `internal/orchestrate/discovery_test.go`
- `internal/orchestrate/dispatch_integration.go`
- `internal/orchestrate/dispatch_integration_test.go`
- `internal/orchestrate/findings.go`
- `internal/orchestrate/format_test.go`
- `internal/orchestrate/fsm_test.go`
- `internal/orchestrate/fsm_v2.go`
- `internal/orchestrate/fsm_v2_test.go`
- `internal/orchestrate/hooks_test.go`
- `internal/orchestrate/hydrate.go`
- `internal/orchestrate/hydrate_sources.go`
- `internal/orchestrate/invoke_opencode.go`
- `internal/orchestrate/invoke_opencode_test.go`
- `internal/orchestrate/llm.go`
- `internal/orchestrate/loop_test.go`
- `internal/orchestrate/runfile.go`
- `internal/orchestrate/state_machine_test.go`
- `internal/planner/openspec/openspec_parser.go`
- `internal/policy/contract.go`
- `internal/policy/contract_test.go`
- `internal/policy/evidence_gate.go`
- `internal/policy/evidence_gate_test.go`
- `internal/policy/explain.go`
- `internal/promote/promote.go`
- `internal/promote/promote_integration_test.go`
- `internal/promote/promote_test.go`
- `internal/prompt/augmentation.go`
- `internal/prompt/sections_test.go`
- `internal/readiness/readiness.go`
- `internal/scout/identity.go`
- `internal/scout/scale.go`
- `internal/session/types.go`
- `internal/strataudit/analyze.go`
- `internal/strataudit/analyze_test.go`
- `internal/strataudit/config.go`
- `internal/strataudit/extract_llm.go`
- `internal/strataudit/extract_llm_test.go`
- `internal/strataudit/ingest.go`
- `internal/strataudit/link.go`
- `internal/strataudit/link_test.go`
- `internal/strataudit/llm_diagnostics.go`
- `internal/strataudit/llmclient.go`
- `internal/strataudit/pipeline.go`
- `internal/strataudit/pipeline_test.go`
- `internal/strataudit/report_builder.go`
- `internal/strataudit/report_builder_test.go`
- `internal/strataudit/runtime_test.go`
- `internal/strataudit/store.go`
- `internal/strataudit/store_test.go`
- `internal/trace/client/client.go`
- `internal/trace/consent/consent.go`
- `internal/trace/consent/consent_test.go`
- `internal/trace/daemon/daemon.go`
- `internal/trace/types.go`
- `internal/workstream/protocol_validate.go`
- `internal/workstream/runtime_beads.go`
- `internal/workstream/template.go`
- `internal/workstream/template_test.go`
- `prompts/commands/submit-to-swarm.md`
- `schema/telemetry/sdp-trace-events.schema.json`
- `scripts/build_push_opencode_agent_image_remote.sh`
- `scripts/check-public-metadata.sh`
- `scripts/check-release-surface.sh`
- `scripts/flaky-detect.sh`
- `scripts/homebrew-dry-run.sh`
- `scripts/quality-metrics.sh`
- `scripts/run-protocol-e2e-docker.sh`
- `tests/architect/adapters_integration_test.go`
- `tests/architect/adversarial_test.go`
- `tests/architect/assembler_test.go`
- `tests/architect/classify_test.go`
- `tests/architect/cli_test.go`
- `tests/architect/crosslang_test.go`
- `tests/architect/eval/harness.go`
- `tests/architect/eval/harness_test.go`
- `tests/architect/extract_test.go`
- `tests/architect/git_extract_test.go`
- `tests/architect/go_extract_test.go`
- `tests/architect/infra_test.go`
- `tests/architect/java_extract_test.go`
- `tests/architect/python_extract_test.go`
- `tests/architect/security_test.go`
- `tests/architect/sql_extract_test.go`
- `tests/architect/types_test.go`
- `tests/architect/typescript_extract_test.go`
- `tests/contracts/compatibility_test.go`



### Commits
- `e5205a0` docs: clarify glm maps to zai in pi-review (2026-04-28)

### Changed Files
- `.beads/issues.jsonl`
- `docs/reference/pi-review-spec.md`
- `docs/workstreams/backlog/00-161-03.md`



### Commits
- `8746f5a` docs: align pi-review mvp with existing pi providers (2026-04-28)

### Changed Files
- `.beads/issues.jsonl`
- `docs/reference/pi-review-spec.md`
- `docs/workstreams/backlog/00-161-03.md`

## 2026-04-29

### Commits
- `cb5470e` F162: add Pi skill and command packaging (2026-04-29)

### Changed Files
- `.agents/skills/README.md`
- `.agents/skills/beads/SKILL.md`
- `.agents/skills/bugfix/SKILL.md`
- `.agents/skills/build/SKILL.md`
- `.agents/skills/ci-triage/SKILL.md`
- `.agents/skills/debug/SKILL.md`
- `.agents/skills/delivery-loop/SKILL.md`
- `.agents/skills/deploy/SKILL.md`
- `.agents/skills/design/SKILL.md`
- `.agents/skills/discovery/SKILL.md`
- `.agents/skills/feature/SKILL.md`
- `.agents/skills/go-modern/SKILL.md`
- `.agents/skills/guard/SKILL.md`
- `.agents/skills/hotfix/SKILL.md`
- `.agents/skills/idea/SKILL.md`
- `.agents/skills/init/SKILL.md`
- `.agents/skills/issue/SKILL.md`
- `.agents/skills/oneshot/SKILL.md`
- `.agents/skills/protocol-consistency/SKILL.md`
- `.agents/skills/prototype/SKILL.md`
- `.agents/skills/reality-check/SKILL.md`
- `.agents/skills/reality/SKILL.md`
- `.agents/skills/review/SKILL.md`
- `.agents/skills/spec-interrogate/SKILL.md`
- `.agents/skills/strataudit/SKILL.md`
- `.agents/skills/tdd/SKILL.md`
- `.agents/skills/think/SKILL.md`
- `.agents/skills/ux/SKILL.md`
- `.agents/skills/verify-workstream/SKILL.md`
- `.agents/skills/vision/SKILL.md`
- `.claude/commands.json`
- `.claude/commands/ship.md`
- `.codex/skills/deploy.md`
- `.codex/skills/ship.md`
- `.codex/skills/tdd.md`
- `.cursor/rules/ship.mdc`
- `.github/workflows/ci.yml`
- `.github/workflows/release.yml`
- `.github/workflows/skill-eval.yml`
- `.opencode/skill/deploy.md`
- `.opencode/skill/ship.md`
- `.opencode/skill/tdd.md`
- `.pi/prompts/beads.md`
- `.pi/prompts/bugfix.md`
- `.pi/prompts/build.md`
- `.pi/prompts/ci-triage.md`
- `.pi/prompts/codereview.md`
- `.pi/prompts/debug.md`
- `.pi/prompts/deliver.md`
- `.pi/prompts/deploy.md`
- `.pi/prompts/design.md`
- `.pi/prompts/feature.md`
- `.pi/prompts/hotfix.md`
- `.pi/prompts/idea.md`
- `.pi/prompts/issue.md`
- `.pi/prompts/oneshot.md`
- `.pi/prompts/prd.md`
- `.pi/prompts/protocol-consistency.md`
- `.pi/prompts/prototype.md`
- `.pi/prompts/reality-check.md`
- `.pi/prompts/reality.md`
- `.pi/prompts/review.md`
- `.pi/prompts/ship.md`
- `.pi/prompts/submit-to-swarm.md`
- `.pi/prompts/test.md`
- `.pi/prompts/verify-workstream.md`
- `.pi/prompts/vision.md`
- `.sdp/generated/.claude/commands/ship.md`
- `.sdp/generated/.codex/skills/deploy.md`
- `.sdp/generated/.codex/skills/ship.md`
- `.sdp/generated/.codex/skills/tdd.md`
- `.sdp/generated/.cursor/rules/ship.mdc`
- `.sdp/generated/.opencode/skill/deploy.md`
- `.sdp/generated/.opencode/skill/ship.md`
- `.sdp/generated/.opencode/skill/tdd.md`
- `.sdp/generated/.pi/prompts/beads.md`
- `.sdp/generated/.pi/prompts/bugfix.md`
- `.sdp/generated/.pi/prompts/build.md`
- `.sdp/generated/.pi/prompts/ci-triage.md`
- `.sdp/generated/.pi/prompts/codereview.md`
- `.sdp/generated/.pi/prompts/debug.md`
- `.sdp/generated/.pi/prompts/deliver.md`
- `.sdp/generated/.pi/prompts/deploy.md`
- `.sdp/generated/.pi/prompts/design.md`
- `.sdp/generated/.pi/prompts/feature.md`
- `.sdp/generated/.pi/prompts/hotfix.md`
- `.sdp/generated/.pi/prompts/idea.md`
- `.sdp/generated/.pi/prompts/issue.md`
- `.sdp/generated/.pi/prompts/oneshot.md`
- `.sdp/generated/.pi/prompts/prd.md`
- `.sdp/generated/.pi/prompts/protocol-consistency.md`
- `.sdp/generated/.pi/prompts/prototype.md`
- `.sdp/generated/.pi/prompts/reality-check.md`
- `.sdp/generated/.pi/prompts/reality.md`
- `.sdp/generated/.pi/prompts/review.md`
- `.sdp/generated/.pi/prompts/ship.md`
- `.sdp/generated/.pi/prompts/submit-to-swarm.md`
- `.sdp/generated/.pi/prompts/test.md`
- `.sdp/generated/.pi/prompts/verify-workstream.md`
- `.sdp/generated/.pi/prompts/vision.md`
- `.sdp/generated/.pi/skills/beads/SKILL.md`
- `.sdp/generated/.pi/skills/bugfix/SKILL.md`
- `.sdp/generated/.pi/skills/build/SKILL.md`
- `.sdp/generated/.pi/skills/ci-triage/SKILL.md`
- `.sdp/generated/.pi/skills/debug/SKILL.md`
- `.sdp/generated/.pi/skills/delivery-loop/SKILL.md`
- `.sdp/generated/.pi/skills/deploy/SKILL.md`
- `.sdp/generated/.pi/skills/design/SKILL.md`
- `.sdp/generated/.pi/skills/discovery/SKILL.md`
- `.sdp/generated/.pi/skills/feature/SKILL.md`
- `.sdp/generated/.pi/skills/go-modern/SKILL.md`
- `.sdp/generated/.pi/skills/guard/SKILL.md`
- `.sdp/generated/.pi/skills/hotfix/SKILL.md`
- `.sdp/generated/.pi/skills/idea/SKILL.md`
- `.sdp/generated/.pi/skills/init/SKILL.md`
- `.sdp/generated/.pi/skills/issue/SKILL.md`
- `.sdp/generated/.pi/skills/oneshot/SKILL.md`
- `.sdp/generated/.pi/skills/protocol-consistency/SKILL.md`
- `.sdp/generated/.pi/skills/prototype/SKILL.md`
- `.sdp/generated/.pi/skills/reality-check/SKILL.md`
- `.sdp/generated/.pi/skills/reality/SKILL.md`
- `.sdp/generated/.pi/skills/review/SKILL.md`
- `.sdp/generated/.pi/skills/ship/SKILL.md`
- `.sdp/generated/.pi/skills/spec-interrogate/SKILL.md`
- `.sdp/generated/.pi/skills/strataudit/SKILL.md`
- `.sdp/generated/.pi/skills/tdd/SKILL.md`
- `.sdp/generated/.pi/skills/think/SKILL.md`
- `.sdp/generated/.pi/skills/ux/SKILL.md`
- `.sdp/generated/.pi/skills/verify-workstream/SKILL.md`
- `.sdp/generated/.pi/skills/vision/SKILL.md`
- `README.md`
- `cmd/sdp-pi-review/main.go`
- `cmd/sdp/cmd_init.go`
- `cmd/sdp/cmd_init_test.go`
- `cmd/sdp/templates/sdp.manifest.template.yaml`
- `docs/QUICKSTART.md`
- `docs/reference/harness-integration.md`
- `docs/reference/harness-parity-matrix.md`
- `internal/adapters/drift.go`
- `internal/adapters/generate.go`
- `internal/adapters/generate_test.go`
- `internal/adapters/templates.go`
- `internal/adapters/templates/pi/prompt.tmpl`
- `internal/adapters/templates/pi/skill.tmpl`
- `internal/inference/confidence/adapters/dispatch/dispatch.go`
- `internal/manifest/manifest.go`
- `internal/manifest/parity.go`
- `internal/manifest/schema.json`
- `internal/pireview/pireview.go`
- `internal/pireview/runner.go`
- `prompts/skills/deploy/SKILL.md`
- `prompts/skills/ship/SKILL.md`
- `prompts/skills/tdd/SKILL.md`
- `scripts/smoke/pi_resources.mjs`
- `sdp.manifest.yaml`

## 2026-04-30

### Commits
- `07f259ed` fix: align review promises with implementation (#144) (2026-04-30)

### Changed Files
- `.beads/interactions.jsonl`
- `.beads/issues.jsonl`
- `.github/workflows/sdp-doctor.yml`
- `README.md`
- `cmd/sdp-pi-review/main.go`
- `cmd/sdp/.snapshots/doctor-usage.snap`
- `cmd/sdp/.snapshots/main-usage.snap`
- `cmd/sdp/.snapshots/unknown-command.snap`
- `cmd/sdp/cmd_bootstrap.go`
- `cmd/sdp/cmd_bootstrap_test.go`
- `cmd/sdp/cmd_doctor.go`
- `cmd/sdp/cmd_doctor_backlog.go`
- `cmd/sdp/cmd_doctor_backlog_test.go`
- `cmd/sdp/cmd_init.go`
- `cmd/sdp/cmd_init_test.go`
- `cmd/sdp/main.go`
- `cmd/sdp/main_test.go`
- `cmd/sdp/snapshot_test.go`
- `docs/MULTI-REPO-WORKFLOW.md`
- `docs/QUICKSTART.md`
- `docs/architecture/REPO-BOUNDARY.md`
- `docs/reference/ci-gates-map.md`
- `docs/reference/maturity-matrix.md`
- `docs/reference/pi-review-spec.md`
- `docs/reference/trust-guarantees.md`
- `internal/adapters/generate.go`
- `internal/adapters/generate_test.go`
- `internal/control/repo_dual_test.go`
- `internal/control/repo_file.go`
- `internal/executil/runner.go`
- `internal/executil/runner_test.go`
- `internal/manifest/load.go`
- `internal/manifest/load_test.go`
- `internal/manifest/schema.json`
- `internal/pireview/pireview.go`
- `internal/pireview/runner.go`
- `internal/pireview/runner_test.go`
- `scripts/check-public-metadata.sh`
- `scripts/hooks/pre-push.sh`
- `scripts/install.sh`
- `scripts/install_kubeopencode_remote.sh`
- `scripts/run_go_quality_gates.sh`
- `scripts/run_smoke_tests.sh`
- `scripts/sdp-publish.sh`
- `sdp.manifest.yaml`

## 2026-05-03

### Commits
- `bdcbf74d` F165: Indirect Prompt Injection Through SDP Task Data (#148) (2026-05-04)

### Changed Files
- `.beads-sdp-mapping.jsonl`
- `.sdp/review_verdict.json`
- `.sdp/ws-verdicts/00-165-01.json`
- `.sdp/ws-verdicts/00-165-02.json`
- `.sdp/ws-verdicts/00-165-03.json`
- `.sdp/ws-verdicts/00-165-04.json`
- `.sdp/ws-verdicts/00-165-05.json`
- `cmd/sdp-eval/main.go`
- `cmd/sdp-eval/main_test.go`
- `docs/plans/2026-05-03-f165-indirect-prompt-injection-through-task-data-design.md`
- `docs/reviews/2026-05-03-f165-advisory-review.md`
- `docs/reviews/2026-05-03-f165-design-spec-interrogate-evidence.json`
- `docs/reviews/2026-05-03-f165-design-spec-interrogate.md`
- `docs/reviews/2026-05-03-f165-workstreams-spec-interrogate-evidence.json`
- `docs/reviews/2026-05-03-f165-workstreams-spec-interrogate.md`
- `docs/roadmap/ROADMAP.md`
- `docs/security/f164-prompt-injection-threat-model.md`
- `docs/workstreams/INDEX.md`
- `docs/workstreams/backlog/00-165-00.md`
- `docs/workstreams/backlog/00-165-01.md`
- `docs/workstreams/backlog/00-165-02.md`
- `docs/workstreams/backlog/00-165-03.md`
- `docs/workstreams/backlog/00-165-04.md`
- `docs/workstreams/backlog/00-165-05.md`
- `internal/evals/f165/core.go`
- `internal/evals/f165/core_test.go`
- `internal/evals/f165/fixture_test.go`
- `internal/evals/f165/report.go`
- `internal/evals/f165/report_test.go`
- `internal/evals/f165/types.go`
- `internal/evals/f165/unsafe.go`
- `internal/evals/f165/vector_demo_test.go`
- `internal/evals/testdata/indirect_pi/beads_issue_poisoning.yaml`
- `internal/evals/testdata/indirect_pi/evidence_finding_poisoning.yaml`
- `internal/evals/testdata/indirect_pi/nonobvious_prose.yaml`
- `internal/evals/testdata/indirect_pi/residual_risk_unsupported.yaml`
- `internal/evals/testdata/indirect_pi/workstream_markdown_poisoning.yaml`
- `schema/indirect-pi-corpus.schema.json`



### Commits
- `38ac0cb2` Revert "config(pi): increase provider timeout, document harness payload workaround" (2026-05-03)

### Changed Files
- `.beads/issues.jsonl`
- `.pi/settings.json`
- `AGENTS.md`



### Commits
- `6e1b4922` fix(pi-skills): resolve skill parsing warnings and collisions (2026-05-03)

### Changed Files
- `.agents/skills/build/SKILL.md`
- `.agents/skills/review/SKILL.md`
- `.pi/README.md`
- `.pi/skills/architect/SKILL.md`
- `.pi/skills/deployer/SKILL.md`
- `.pi/skills/devops/SKILL.md`
- `.pi/skills/implementer/SKILL.md`
- `.pi/skills/orchestrator/SKILL.md`
- `.pi/skills/planner/SKILL.md`
- `.pi/skills/prompts-skills`
- `.pi/skills/qa/SKILL.md`
- `.pi/skills/reviewer/SKILL.md`
- `.pi/skills/security/SKILL.md`
- `.pi/skills/spec-reviewer/SKILL.md`
- `.pi/skills/sre/SKILL.md`
- `.pi/skills/tech-lead/SKILL.md`



### Commits
- `aab4a68a` fix(pi): make skills and agents readable by pi harness (2026-05-03)

### Changed Files
- `.pi/README.md`
- `.pi/settings.json`
- `.pi/skills/architect.md`
- `.pi/skills/deployer.md`
- `.pi/skills/devops.md`
- `.pi/skills/implementer.md`
- `.pi/skills/orchestrator.md`
- `.pi/skills/planner.md`
- `.pi/skills/prompts-skills`
- `.pi/skills/qa.md`
- `.pi/skills/reviewer.md`
- `.pi/skills/security.md`
- `.pi/skills/spec-reviewer.md`
- `.pi/skills/sre.md`
- `.pi/skills/tech-lead.md`
- `.sdp/generated/.codex/skills/build.md`
- `.sdp/generated/.codex/skills/delivery-loop.md`
- `.sdp/generated/.opencode/skill/build.md`
- `.sdp/generated/.opencode/skill/delivery-loop.md`
- `.sdp/generated/.pi/skills/build/SKILL.md`
- `.sdp/generated/.pi/skills/delivery-loop/SKILL.md`



### Commits
- `c099b84c` docs: enforce scoped completion in agent skills (2026-05-03)

### Changed Files
- `.agents/skills/AGENTS.md`
- `.agents/skills/build/SKILL.md`
- `.agents/skills/delivery-loop/SKILL.md`
- `AGENTS.md`
- `docs/reference/agent-instruction-cascade.md`
- `internal/workstream/skill_lint.go`
- `internal/workstream/skill_lint_test.go`
- `prompts/skills/AGENTS.md`
- `prompts/skills/build/SKILL.md`
- `prompts/skills/delivery-loop/SKILL.md`
- `scripts/lint-skills.sh`



### Commits
- `1995dc4c` F164: harden prompt-injection defenses (2026-05-03)

### Changed Files
- `.agents/skills/build.md`
- `.agents/skills/review.md`
- `.beads/issues.jsonl`
- `.github/workflows/pi-harness-check.yml`
- `.github/workflows/prompt-injection-corpus.yml`
- `.pi/APPEND_SYSTEM.md`
- `.pi/extensions/sdp.ts`
- `.pi/settings.json`
- `.pi/skills/cpp-dev/SKILL.md`
- `.pi/skills/csharp-dev/SKILL.md`
- `.pi/skills/dart-dev/SKILL.md`
- `.pi/skills/elixir-dev/SKILL.md`
- `.pi/skills/fortran-dev/SKILL.md`
- `.pi/skills/go-dev/SKILL.md`
- `.pi/skills/haskell-dev/SKILL.md`
- `.pi/skills/js-dev/SKILL.md`
- `.pi/skills/julia-dev/SKILL.md`
- `.pi/skills/jvm-dev/SKILL.md`
- `.pi/skills/kotlin-dev/SKILL.md`
- `.pi/skills/lua-dev/SKILL.md`
- `.pi/skills/nim-dev/SKILL.md`
- `.pi/skills/ocaml-dev/SKILL.md`
- `.pi/skills/php-dev/SKILL.md`
- `.pi/skills/python-dev/SKILL.md`
- `.pi/skills/ruby-dev/SKILL.md`
- `.pi/skills/rust-dev/SKILL.md`
- `.pi/skills/swift-dev/SKILL.md`
- `.pi/skills/ux-testing/SKILL.md`
- `.pi/skills/zig-dev/SKILL.md`
- `.sdp/generated/.claude/agents/reviewer.md`
- `.sdp/generated/.claude/agents/security.md`
- `.sdp/generated/.codex/skills/build.md`
- `.sdp/generated/.codex/skills/review.md`
- `.sdp/generated/.codex/skills/spec-interrogate.md`
- `.sdp/generated/.opencode/skill/build.md`
- `.sdp/generated/.opencode/skill/review.md`
- `.sdp/generated/.opencode/skill/spec-interrogate.md`
- `.sdp/generated/.pi/skills/build/SKILL.md`
- `.sdp/generated/.pi/skills/review/SKILL.md`
- `.sdp/generated/.pi/skills/spec-interrogate/SKILL.md`
- `.sdp/ws-verdicts/00-164-01.json`
- `.sdp/ws-verdicts/00-164-02.json`
- `.sdp/ws-verdicts/00-164-03.json`
- `.sdp/ws-verdicts/00-164-04.json`
- `.sdp/ws-verdicts/00-164-05.json`
- `.sdp/ws-verdicts/00-164-06.json`
- `.sdp/ws-verdicts/00-164-07.json`
- `Makefile`
- `cmd/sdp-eval/main.go`
- `cmd/sdp-eval/main_test.go`
- `cmd/sdp-pi-eval/main.go`
- `cmd/sdp-pi-eval/main_test.go`
- `docs/reference/sdp-mcp.md`
- `docs/reference/skills.md`
- `docs/reviews/2026-05-03-f164-prompt-injection-spec-interrogate.md`
- `docs/reviews/2026-05-03-f164-workstream-interrogate.md`
- `docs/security/f164-prompt-injection-test-cases.md`
- `docs/security/f164-prompt-injection-threat-model.md`
- `docs/workstreams/INDEX.md`
- `docs/workstreams/backlog/00-164-01.md`
- `docs/workstreams/backlog/00-164-02.md`
- `docs/workstreams/backlog/00-164-03.md`
- `docs/workstreams/backlog/00-164-04.md`
- `docs/workstreams/backlog/00-164-05.md`
- `docs/workstreams/backlog/00-164-06.md`
- `docs/workstreams/backlog/00-164-07.md`
- `internal/evals/pi_corpus.go`
- `internal/evals/pi_corpus_test.go`
- `internal/evals/pi_live.go`
- `internal/evals/pi_live_test.go`
- `internal/evals/pi_mock.go`
- `internal/evals/pi_mock_test.go`
- `internal/mcp/contract/mapping.go`
- `internal/mcp/contract/mapping_test.go`
- `internal/mcp/server.go`
- `internal/mcp/server_test.go`
- `internal/mcp/tools.go`
- `internal/mcp/tools_test.go`
- `pi-sdp-harness/extensions/sdp.ts`
- `pi-sdp-harness/package.json`
- `pi-sdp-harness/prompts/bugfix.md`
- `pi-sdp-harness/prompts/hotfix.md`
- `pi-sdp-harness/skills/cpp-dev/SKILL.md`
- `pi-sdp-harness/skills/csharp-dev/SKILL.md`
- `pi-sdp-harness/skills/dart-dev/SKILL.md`
- `pi-sdp-harness/skills/elixir-dev/SKILL.md`
- `pi-sdp-harness/skills/fortran-dev/SKILL.md`
- `pi-sdp-harness/skills/go-dev/SKILL.md`
- `pi-sdp-harness/skills/haskell-dev/SKILL.md`
- `pi-sdp-harness/skills/js-dev/SKILL.md`
- `pi-sdp-harness/skills/julia-dev/SKILL.md`
- `pi-sdp-harness/skills/jvm-dev/SKILL.md`
- `pi-sdp-harness/skills/kotlin-dev/SKILL.md`
- `pi-sdp-harness/skills/lua-dev/SKILL.md`
- `pi-sdp-harness/skills/nim-dev/SKILL.md`
- `pi-sdp-harness/skills/ocaml-dev/SKILL.md`
- `pi-sdp-harness/skills/php-dev/SKILL.md`
- `pi-sdp-harness/skills/python-dev/SKILL.md`
- `pi-sdp-harness/skills/ruby-dev/SKILL.md`
- `pi-sdp-harness/skills/rust-dev/SKILL.md`
- `pi-sdp-harness/skills/swift-dev/SKILL.md`
- `pi-sdp-harness/skills/ux-testing/SKILL.md`
- `pi-sdp-harness/skills/zig-dev/SKILL.md`
- `prompts/agents/reviewer.md`
- `prompts/agents/security.md`
- `prompts/skills/build/SKILL.md`
- `prompts/skills/review/SKILL.md`
- `schema/contracts/cli-mcp-mapping.json`
- `schema/prompt-injection-corpus.schema.json`
- `scripts/check-prompt-injection-corpus.sh`
- `scripts/prompt-injection-check.sh`
- `scripts/run_go_quality_gates.sh`



### Commits
- `541d3705` docs: define agent instruction cascade (2026-05-03)

### Changed Files
- `AGENTS.md`
- `docs/reference/AGENTS.md`
- `docs/reference/README.md`
- `docs/reference/agent-instruction-cascade.md`
- `docs/reference/project-map.md`
- `docs/reference/skill-authoring.md`
- `docs/reference/skills.md`
- `prompts/agents/AGENTS.md`
- `prompts/skills/AGENTS.md`

## 2026-05-04

### Commits
- `6c75c476` F166: specify local chunked classifier (2026-05-04)

### Changed Files
- `.beads-sdp-mapping.jsonl`
- `.beads/issues.jsonl`
- `docs/plans/2026-05-03-f166-runtime-llm-guard-gateway-design.md`
- `docs/reviews/2026-05-04-f166-local-classifier-spec-interrogate-evidence.json`
- `docs/reviews/2026-05-04-f166-local-classifier-spec-interrogate.md`
- `docs/roadmap/ROADMAP.md`
- `docs/workstreams/INDEX.md`
- `docs/workstreams/backlog/00-166-06.md`



### Commits
- `ef1b7d53` F166: record gateway substrate decision (2026-05-04)

### Changed Files
- `.beads-sdp-mapping.jsonl`
- `.beads/interactions.jsonl`
- `.beads/issues.jsonl`
- `docs/plans/2026-05-03-f166-runtime-llm-guard-gateway-design.md`
- `docs/workstreams/INDEX.md`
- `docs/workstreams/backlog/00-166-05.md`



### Commits
- `0b164516` F166: Runtime LLM Guard Gateway (2026-05-04)

### Changed Files
- `.sdp/checkpoints/F166.json`
- `cmd/sdp-llm-gateway/main.go`
- `cmd/sdp-llm-gateway/main_test.go`
- `docs/plans/2026-05-03-f166-runtime-llm-guard-gateway-design.md`
- `docs/reviews/2026-05-03-f166-implementation-workstreams-review-evidence.json`
- `docs/reviews/2026-05-03-f166-implementation-workstreams-review.md`
- `docs/reviews/2026-05-03-f166-llm-guard-spec-interrogate-evidence.json`
- `docs/reviews/2026-05-03-f166-llm-guard-spec-interrogate.md`
- `docs/reviews/2026-05-03-f166-workstream-review-evidence.json`
- `docs/reviews/2026-05-03-f166-workstream-review.md`
- `docs/roadmap/ROADMAP.md`
- `docs/workstreams/INDEX.md`
- `docs/workstreams/backlog/00-166-01.md`
- `docs/workstreams/backlog/00-166-02.md`
- `docs/workstreams/backlog/00-166-03.md`
- `docs/workstreams/backlog/00-166-04.md`
- `internal/llmguard/gateway.go`
- `internal/llmguard/gateway_test.go`
- `internal/llmguard/redactor.go`
- `internal/llmguard/redactor_test.go`
- `internal/llmguard/scanner.go`
- `internal/llmguard/scanner_test.go`
- `internal/llmguard/types.go`
- `internal/llmguard/types_test.go`



### Commits
- `032a0946` Merge F167 security verdict gate design docs (2026-05-04)
- `88476997` docs: harden F167 review findings (2026-05-04)
- `0c4dfb48` docs: design F167 security verdict gate (2026-05-04)

### Changed Files
- `.beads-sdp-mapping.jsonl`
- `.sdp/log/events.jsonl`
- `docs/drafts/f167-security-verdict-gate-design.md`
- `docs/reviews/2026-05-04-f167-spec-workstreams-review-evidence.json`
- `docs/reviews/2026-05-04-f167-spec-workstreams-review.md`
- `docs/roadmap/ROADMAP.md`
- `docs/workstreams/INDEX.md`
- `docs/workstreams/backlog/00-167-01.md`
- `docs/workstreams/backlog/00-167-02.md`
- `docs/workstreams/backlog/00-167-03.md`
- `docs/workstreams/backlog/00-167-04.md`



### Commits
- `70c8cdf3` Revert "docs(AGENTS): add operational lessons from F165 session" (2026-05-04)

### Changed Files
- `AGENTS.md`



### Commits
- `819b0568` docs(AGENTS): add operational lessons from F165 session (2026-05-04)

### Changed Files
- `AGENTS.md`

