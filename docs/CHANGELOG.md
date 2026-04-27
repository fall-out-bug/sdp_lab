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

