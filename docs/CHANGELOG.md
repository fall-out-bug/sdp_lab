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

