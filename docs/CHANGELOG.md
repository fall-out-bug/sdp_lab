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

