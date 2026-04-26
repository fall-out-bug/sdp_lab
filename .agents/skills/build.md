---
name: build
description: Create new features, components, or systems — from idea to implementation with TDD.
version: 1.0.0
tags:
  - feature
  - implementation
  - tdd
requires_cli:
  - sdp
compatibility:
  - claude-code
  - opencode
  - cursor
  - codex
---

# build

## Purpose

Create something new: features, components, systems, prototypes. TDD by default.
Absorbs: @feature, @idea, @design, @ux, @vision, @oneshot, @prototype.

## When to Use

Implementing features, creating designs, prototyping, user-facing work with acceptance criteria.

## Session Bootstrap (ALWAYS FIRST — before any other step)

**On fresh start:**
1. `bd update <id> --claim` — claim the beads issue atomically
2. Determine branch slug from feature ID: `f<N>-<slug>` (e.g. `f132-testing-intel`)
3. `git worktree add .worktrees/<slug> -b <slug> main` — create isolated worktree
4. Write `.sdp/checkpoint.json`:
   ```json
   {"skill":"build","feature_id":"<id>","branch":"<slug>","worktree":".worktrees/<slug>","step":"bootstrap","ts":"<iso>"}
   ```
5. All subsequent work runs inside the worktree. Never edit files in the main tree.

**On compaction recovery** (user says "continue", "продолжай", or session restart with existing checkpoint):
1. `cat .sdp/checkpoint.json` — read last state
2. `bd list --status=in_progress` — verify claim is still held
3. `cd <worktree>` — switch to the worktree
4. Resume from `checkpoint.step` — do NOT restart from scratch

## Modes

**idea:** Problem → design doc (no implementation). Output: design with requirements, approach, tradeoffs. For: "design a...", "how should we...", planning phase.
**feature:** Idea → design → implement → test → PR (full cycle). Output: complete feature with tests. For: "implement...", "build...", "add...".
**prototype:** Skip design, build quickly, mark experimental. Output: working prototype + TODO list. For: "prototype...", "quick mock...", "proof of concept".

## Routing Rules

Scope based on: (1) Request type: "Design..."→idea, "Implement..."→feature, "Prototype..."→prototype.
(2) Available context: design doc exists? skip to implementation. (3) User preference.
(4) Complexity: single button→prototype, new auth system→feature.

## Input Expectations

- **Requirement:** Clear description of what to build
- **Context:** Design docs (optional), codebase context (auto-detected)
- **Mode:** Optional — auto-detected from request language
- **Acceptance criteria:** Optional — generated if not provided

## Legacy Aliases

@feature→feature mode, @idea/@design/@ux/@vision→idea mode, @oneshot→prototype mode, @prototype→prototype mode

## Embedded Practices

**@tdd:** Test-first DEFAULT workflow. Write failing test → verify fails → implement → verify passes → refactor. NEVER skipped for production code. NOT a separate skill — embedded in @build and @fix.

**@guard:** Pre-commit quality gate runs automatically via hooks. NOT invoked manually.

**@go-modern:** Language style convention (documented in CLAUDE.md). Applied automatically during implementation. NOT a skill entry point.

## Artifacts Created

**idea:** Design document (`docs/design/*.md`)
**feature:** Implementation code with TDD tests, docs, PR
**prototype:** Working code, [PROTOTYPE] label, TODO list for productionization

## Strict Mode (SDP Flow 2)

When `--strict-mode` flag or `SDP_STRICT=true` env is active:

1. MUST invoke `sdp phase plan --feature-id <F> --strict --evidence-path .sdp/evidence/plan.json` before implementation
2. MUST invoke `sdp phase eval --feature-id <F> --strict --evidence-path .sdp/evidence/eval.json` after implementation
3. MUST add provenance trailer to every commit using `internal/provenance` package

## Provenance Pattern

Every commit in SDP-managed repos SHOULD carry provenance trailer:
- Agent commits: `AI-Attribution: agent/<model>/<session>`
- Human commits: No trailer needed
- Hybrid commits: `AI-Attribution: hybrid/<model>/<session>`

## MUST DO
- Run quality gates before committing
- Claim beads issue before starting work
- Add provenance trailer to commits when in SDP context
- Follow TDD: test → implement → refactor (never skip for production code)
- Create worktree for isolated development (bootstrap step 3)
- Write checkpoint.json before starting implementation
- Resume from checkpoint on compaction recovery (don't restart)

## MUST NOT DO
- Skip quality gates
- Commit without claiming beads issue
- Push directly to main branch
- Skip TDD for production code
- Edit files in main tree (always use worktree)

## Response Format

After completing work, report:

1. **Implementation summary**: What was built and why
2. **File changes**: List of modified/created files (absolute paths)
3. **Tests**: Test results including coverage
4. **Quality gates**: Confirmation all gates passed
5. **Next steps**: PR ready, additional work needed, or follow-up tasks

Example:
```
## Implementation Summary

Built [feature description] to address [requirement].

## File Changes

- /path/to/implementation.go
- /path/to/implementation_test.go
- /path/to/docs.md

## Tests

✓ All tests passed (N tests, 100% coverage)
✓ Quality gates passed

## Next Steps

Ready for PR review.
```

## Stack-Specific Workflows

<!-- STACK_SPECIFIC:BEGIN section="test" stack="go" -->
### Go Testing

Run all tests:
```bash
go test ./... -v
```

Run with coverage:
```bash
go test -coverprofile=/tmp/coverage.out ./...
go tool cover -html=/tmp/coverage.out -o /tmp/coverage.html
```

Run specific package:
```bash
go test ./pkg/mypackage -v
```

Run with race detection:
```bash
go test -race ./...
```

Skip integration tests:
```bash
go test -short ./...
```
<!-- STACK_SPECIFIC:END -->

<!-- STACK_SPECIFIC:BEGIN section="build" stack="go" -->
### Go Building

Build all packages:
```bash
go build ./...
```

Build specific binary:
```bash
go build -o bin/myapp ./cmd/myapp
```

Build with optimizations:
```bash
go build -ldflags="-s -w" -o bin/myapp ./cmd/myapp
```

Install to GOPATH/bin:
```bash
go install ./cmd/myapp
```
<!-- STACK_SPECIFIC:END -->

<!-- STACK_SPECIFIC:BEGIN section="test" stack="python" -->
### Python Testing

Run all tests:
```bash
pytest -v
```

Run with coverage:
```bash
pytest --cov=. --cov-report=html --cov-report=term
```

Run specific test file:
```bash
pytest tests/test_specific.py -v
```

Run with markers:
```bash
pytest -m "not slow" -v
```

Skip integration tests:
```bash
pytest -m "not integration" -v
```
<!-- STACK_SPECIFIC:END -->

<!-- STACK_SPECIFIC:BEGIN section="build" stack="python" -->
### Python Building

Build package:
```bash
python -m build
```

Install in development mode:
```bash
pip install -e .
```

Install with dependencies:
```bash
pip install -r requirements.txt
```

Check for dependency issues:
```bash
pip check
```
<!-- STACK_SPECIFIC:END -->

<!-- STACK_SPECIFIC:BEGIN section="test" stack="typescript" -->
### TypeScript Testing

Run all tests:
```bash
npm test
```

Run with coverage:
```bash
npm run test:cov
```

Run specific test file:
```bash
npm test -- src/path/to/test.spec.ts
```

Run in watch mode:
```bash
npm test -- --watch
```

Run with debugger:
```bash
npm test -- --inspect-brk
```
<!-- STACK_SPECIFIC:END -->

<!-- STACK_SPECIFIC:BEGIN section="build" stack="typescript" -->
### TypeScript Building

Build all:
```bash
npm run build
```

Build with watch mode:
```bash
npm run build:watch
```

Build specific target:
```bash
npx tsc --project tsconfig.app.json
```

Check types without emitting:
```bash
npx tsc --noEmit
```

Clean build artifacts:
```bash
npm run clean
```
<!-- STACK_SPECIFIC:END -->

<!-- STACK_SPECIFIC:BEGIN section="quality-gate" stack="go" -->
### Go Quality Gates

Run all quality checks:
```bash
go vet ./... && go test ./... && go build ./...
```

Format check:
```bash
gofmt -l . && test -z "$(gofmt -l .)"
```

Run with race and coverage:
```bash
go test -race -coverprofile=/tmp/coverage.out ./...
```

Static analysis:
```bash
staticcheck ./...
```
<!-- STACK_SPECIFIC:END -->

<!-- STACK_SPECIFIC:BEGIN section="quality-gate" stack="python" -->
### Python Quality Gates

Run all quality checks:
```bash
flake8 . && pytest -v && mypy .
```

Format check:
```bash
black --check .
```

Lint with pylint:
```bash
pylint **/*.py
```

Type checking:
```bash
mypy .
```
<!-- STACK_SPECIFIC:END -->

<!-- STACK_SPECIFIC:BEGIN section="quality-gate" stack="typescript" -->
### TypeScript Quality Gates

Run all quality checks:
```bash
npm run lint && npm test && npm run build
```

Format check:
```bash
npm run format:check
```

Lint with ESLint:
```bash
npm run lint
```

Type checking:
```bash
npx tsc --noEmit
```
<!-- STACK_SPECIFIC:END -->

## Acceptance Boundaries

NOT for: understanding code (@understand), fixing bugs (@fix), review (@review), deployment (@operate)

@build may internally call @understand for context without explicit user invocation when work spans multiple intents.

Quality gates: all tests pass, follows conventions, docs updated, PR ready
