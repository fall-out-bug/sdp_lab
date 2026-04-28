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
- `8746f5a` docs: align pi-review mvp with existing pi providers (2026-04-28)

### Changed Files
- `.beads/issues.jsonl`
- `docs/reference/pi-review-spec.md`
- `docs/workstreams/backlog/00-161-03.md`

