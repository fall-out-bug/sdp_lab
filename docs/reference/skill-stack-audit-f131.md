# Skills Stack-Specific Audit — F131-01

**Date**: 2026-04-25  
**Purpose**: Classify all 43 skills in `.agents/skills/` for stack-specific marker convention needs  
**Criteria**: Skills that reference specific CLI tools (go test, pytest, npm, etc.) or language-specific commands

## Audit Results

| Skill | Classification | Rationale |
|-------|---------------|-----------|
| **README.md** | Stack-Agnostic | Documentation only |
| **architect.md** | Stack-Agnostic | System design, no tool-specifics |
| **bug-fix.md** | **Needs-Stack-Specific** | References `go test ./...`, `go build ./...`, `go vet ./...` |
| **bugfix.md** | **Needs-Stack-Specific** | Legacy bugfix, likely Go tool references |
| **build.md** | **Needs-Stack-Specific** | TDD workflow, needs stack-specific test/build commands |
| **ci-triage.md** | Stack-Agnostic | Deprecated, redirects to @operate |
| **debug.md** | Stack-Agnostic | Deprecated, redirects to @fix |
| **delivery-loop.md** | Stack-Agnostic | Workflow orchestration, tool-agnostic |
| **deploy.md** | **Needs-Stack-Specific** | Deployment needs language-specific build/deploy steps |
| **design.md** | Stack-Agnostic | Design documentation, no tool specifics |
| **eval-phase.md** | Stack-Agnostic | Phase protocol, tool-agnostic |
| **feature-delivery.md** | **Needs-Stack-Specific** | Explicit `go build ./...`, `go test ./...`, `go vet ./...` |
| **feature.md** | Stack-Agnostic | Feature planning, no tool specifics |
| **fix.md** | **Needs-Stack-Specific** | Bug fixing workflow, needs stack-specific debugging |
| **gate.md** | Stack-Agnostic | Shared protocol, tool-agnostic |
| **git-worktree.md** | **Needs-Stack-Specific** | Multi-language dependency detection (`package.json`→npm, `go.mod`→go mod, `Cargo.toml`→cargo) |
| **hotfix.md** | **Needs-Stack-Specific** | Hotfix workflow, needs stack-specific fixes |
| **idea.md** | Stack-Agnostic | Brainstorming, no tool specifics |
| **issue.md** | Stack-Agnostic | Issue analysis, tool-agnostic |
| **landscape.md** | Stack-Agnostic | Codebase analysis, tool-agnostic |
| **llm-council.md** | Stack-Agnostic | LLM coordination, no tool specifics |
| **metrics.md** | Stack-Agnostic | Metrics collection, tool-agnostic |
| **oneshot.md** | **Needs-Stack-Specific** | Quick execution, needs stack-specific commands |
| **operate.md** | **Needs-Stack-Specific** | DevOps/deployment, needs stack-specific deploy steps |
| **parallel-dispatch.md** | Stack-Agnostic | Agent coordination, no tool specifics |
| **plan-phase.md** | Stack-Agnostic | Phase protocol, tool-agnostic |
| **plan.md** | Stack-Agnostic | Planning workflow, tool-agnostic |
| **prototype.md** | **Needs-Stack-Specific** | Prototyping, needs stack-specific build/test |
| **reality-check.md** | Stack-Agnostic | Documentation vs code check, tool-agnostic |
| **research.md** | Stack-Agnostic | Research workflow, tool-agnostic |
| **review-phase.md** | Stack-Agnostic | Phase protocol, tool-agnostic |
| **review.md** | Stack-Agnostic | Code review, tool-agnostic |
| **scout.md** | Stack-Agnostic | Exploration, tool-agnostic |
| **session-audit.md** | Stack-Agnostic | Audit logging, tool-agnostic |
| **smoke-test.md** | **Needs-Stack-Specific** | Explicit `go test -tags=smoke ./test/smoke/... -v` |
| **spec-interrogate.md** | Stack-Agnostic | Spec analysis, tool-agnostic |
| **strataudit.md** | Stack-Agnostic | Evidence collection, tool-agnostic |
| **test-coverage.md** | **Needs-Stack-Specific** | Explicit `go test -coverprofile`, `go tool cover -func` |
| **test-writer.md** | **Needs-Stack-Specific** | Explicit `go test`, `go tool cover`, Go-specific AST parsing |
| **understand.md** | Stack-Agnostic | Code understanding, tool-agnostic |
| **ux.md** | Stack-Agnostic | UX research, no tool specifics |
| **verify-workstream.md** | Stack-Agnostic | Workstream validation, tool-agnostic |
| **vision.md** | Stack-Agnostic | Strategic planning, no tool specifics |

## Summary

- **Total Skills**: 43
- **Needs-Stack-Specific**: 14 skills (32.6%)
- **Stack-Agnostic**: 29 skills (67.4%)

## Skills Requiring Stack-Specific Sections (Priority Order)

1. **test-coverage.md** - Go-specific `go test -coverprofile`, `go tool cover -func`
2. **test-writer.md** - Go-specific AST parsing, `go test`, coverage tools
3. **smoke-test.md** - Go-specific `go test -tags=smoke`
4. **build.md** - TDD workflow, needs stack-specific test/build commands (POC target)
5. **fix.md** - Bug fixing, needs stack-specific debugging commands
6. **feature-delivery.md** - Go-specific quality gates
7. **bug-fix.md** - Go-specific test/build commands
8. **git-worktree.md** - Multi-language dependency detection
9. **deploy.md** - Language-specific deployment steps
10. **operate.md** - DevOps, needs stack-specific deploy/monitoring
11. **prototype.md** - Prototyping, needs stack-specific build/test
12. **oneshot.md** - Quick execution, needs stack-specific commands
13. **hotfix.md** - Hotfixes, needs stack-specific fixes
14. **bugfix.md** - Legacy bugfix, likely Go tool references

## Common Stack-Specific Patterns Found

### Go
- `go test ./...`
- `go build ./...`
- `go vet ./...`
- `go test -coverprofile=$(mktemp)`
- `go tool cover -func=<profile>`
- `go mod download`
- `go test -tags=smoke`
- `go test -short`
- AST-based parsing (`go/ast`)

### Detected Multi-Language Support
- `package.json` → npm
- `go.mod` → go mod
- `Cargo.toml` → cargo build
- `requirements.txt` → pip install

## Recommended Sections for Markers

1. **test** - Test execution commands per stack
2. **build** - Build compilation commands per stack
3. **lint** - Linting/static analysis per stack
4. **quality-gate** - Quality gate commands per stack
5. **debug** - Debugging commands per stack
6. **coverage** - Coverage analysis per stack

## Next Steps

1. ✅ Create marker convention document
2. ✅ Implement POC on `build.md` with Go, Python, TypeScript markers
3. Apply markers to other 13 identified skills (F131-02 augmenter)
