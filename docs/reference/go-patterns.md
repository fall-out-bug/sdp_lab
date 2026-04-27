# Go Patterns & Coding Standards

> Canonical rules file for all AI harnesses (Claude Code, Codex, Cursor, OpenCode).
> **This file counts as part of cold-start context** — always read before writing Go code.

## Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.26 |
| Module | `sdp_dev` |
| LLM integration | `github.com/mark3labs/mcp-go v0.48.0` |
| Persistence | SQLite (`github.com/mattn/go-sqlite3`) |
| Supply-chain security | sigstore-go, in-toto-golang |
| Serialization | protobuf, yaml.v3 |
| Testing | testify/assert + testify/require |
| Sync primitives | `golang.org/x/sync`, standard `sync` |

## Folder Structure

```
sdp_lab/
├── cmd/                  # One main.go per binary (thin — delegates to internal/)
│   └── sdp-foo/
│       └── main.go       # ≤60 lines; parse flags → call internal; no business logic
├── internal/
│   └── foo/
│       ├── foo.go        # Public types + constructor NewFoo()
│       ├── foo_test.go   # Unit tests (t.Run, table-driven)
│       ├── store.go      # Storage interface + implementations
│       └── gateway.go    # External interface + stub for tests
├── scripts/              # Shell helpers, hooks
├── deploy/               # K8s / Helm manifests
├── docs/                 # All planning and reference docs
├── schema/               # JSON schemas
└── testdata/             # Golden files, fixtures
```

**Rule:** Business logic never lives in `cmd/`. `cmd/` parses flags and calls `internal/`.

## Naming Conventions

| Element | Convention | Example |
|---------|-----------|---------|
| Package | `lowercase`, short, no underscores | `agentloop`, `ciloop`, `harness` |
| Module path | `sdp_dev/internal/...` (import prefix is `sdp_dev`, NOT the GitHub path) | `import "github.com/fall-out-bug/sdp_lab/internal/healthcheck"` |
| Exported type | `PascalCase` | `Session`, `ModelGateway`, `LoopConfig` |
| Interface | `PascalCase`, noun (not `-er` unless idiomatic) | `ModelGateway`, `SessionStore`, `ContextManager` |
| Private struct | `camelCase` | `harnessState`, `atomicDoc` |
| Constructor | `NewFoo(deps) (*Foo, error)` — single type; `NewFooRunner` or `NewFooRegistry` when orchestrating a slice | `NewSession()`, `NewRunner()` |
| FSM states | `camelCase` const iota | `hStateIdle`, `hStateRunning` |
| Errors | `ErrFoo` sentinel; wrap with `fmt.Errorf("%w", err)` | `ErrHarnessTerminated` |
| Test file | `foo_test.go` (same package); `foo_external_test.go` for black-box | |
| Stub/mock | `StubFoo` or `fakeFoo` (unexported in test file) | `StubGateway`, `fakeChecker` |

## Architecture Patterns

### 1. FSM with mutex guard

All stateful types use an explicit state machine with a mutex-protected guard check before each transition:

```go
type myState int

const (
    stateIdle    myState = iota
    stateRunning
    stateStopped // terminal
)

type Worker struct {
    mu    sync.Mutex
    state myState
    // ...
}

func (w *Worker) Start(ctx context.Context) error {
    w.mu.Lock()
    if w.state != stateIdle {
        w.mu.Unlock()
        return fmt.Errorf("worker busy: state=%d", w.state)
    }
    w.state = stateRunning
    w.mu.Unlock()
    // ... do work ...
    return nil
}
```

### 2. Interface-based dependency injection

Dependencies are always injected via interfaces — never instantiated inside constructors:

```go
// types.go — interface definitions, one per abstraction
type ModelGateway interface {
    Call(ctx context.Context, msgs []Message, cfg LoopConfig) (<-chan Event, error)
    IsAvailable(model string) bool
}

// Config struct carries all injected deps
type LoopConfig struct {
    Model          string
    Gateway        ModelGateway   // required — nil → error at Run()
    ContextManager ContextManager // optional — nil = passthrough
}

// Validate at entry points, not inside the loop
func Run(ctx context.Context, msgs []Message, cfg LoopConfig) (<-chan Event, error) {
    if cfg.Gateway == nil {
        return nil, fmt.Errorf("Run: cfg.Gateway must not be nil")
    }
    // ...
}
```

### 3. Durable-first mutations

Persist to storage **before** mutating in-memory state. If persist fails → return error, state unchanged:

```go
func (h *Harness) transitionTo(next Role) error {
    // Write to durable store FIRST
    if err := h.store.PersistPhaseRecord(h.session.ID, PhaseRecord{Phase: next}); err != nil {
        return err // in-memory untouched
    }
    // Only mutate memory after durable commit
    h.session.Phase = next
    h.accumulator.Reset()
    return nil
}
```

### 4. Event-driven goroutine with buffered channel

Async operations return a channel; goroutine lifetime is bounded by `defer close(out)`:

```go
func Run(ctx context.Context, input Input) (<-chan Event, error) {
    out := make(chan Event, 64) // buffered — prevents goroutine leak on slow consumer
    go func() {
        defer close(out) // always closed, even on error paths
        for {
            select {
            case <-ctx.Done():
                out <- Event{Type: "error", Err: ctx.Err()}
                return
            default:
            }
            // ... emit events ...
        }
    }()
    return out, nil
}
```

### 5. Test stubs via interface

Test doubles implement the same interface as production code:

```go
// gateway.go — shared by tests and production
type StubGateway struct {
    responses map[string][][]Event // FIFO queue per model
    callIdx   map[string]int
    Calls     []ModelCall // recorded for assertions
}

func (sg *StubGateway) Call(ctx context.Context, msgs []Message, cfg LoopConfig) (<-chan Event, error) {
    sg.Calls = append(sg.Calls, ModelCall{Model: cfg.Model, Messages: msgs})
    // ... emit queued events or safe fallback ...
}
```

### 6. Subprocess calls (exec.CommandContext)

Always use `exec.CommandContext` (not `exec.Command`) so callers can cancel. Set `cmd.Dir` explicitly:

```go
func (c *myChecker) Run(ctx context.Context) CheckResult {
    cmd := exec.CommandContext(ctx, "go", "build", "./...")
    cmd.Dir = c.projectRoot                  // never rely on os.Getwd()
    out, err := cmd.CombinedOutput()         // capture stderr too
    if err != nil {
        return CheckResult{Status: StatusFail, Detail: strings.TrimSpace(string(out))}
    }
    return CheckResult{Status: StatusPass, Detail: "ok"}
}
```

### 7. Exit codes for warn vs fail

`StatusWarn` exits 0 (advisory, not blocking). `StatusFail` exits 1. Document this in the type:

```go
// exitCode returns 1 only for hard failures; warns are advisory.
func exitCode(results []CheckResult) int {
    for _, r := range results {
        if r.Status == StatusFail {
            return 1
        }
    }
    return 0
}
```

## Examples of Good Code

### Example 1: Canonical internal package layout

**File:** [`internal/agentloop/harness.go`](../../internal/agentloop/harness.go)

Key patterns in this file:
- FSM with 4 states, mutex-guarded transitions (lines 107–136)
- `validateToken` before state access
- Durable-first: `PersistTurnRecord` before gate check
- `ApproveGate` / `Rollback` check state atomically before acting

### Example 2: Interface definition file

**File:** [`internal/agentloop/types.go`](../../internal/agentloop/types.go)

- All interfaces in one file, clearly separated from structs
- Config struct carries all injected interfaces
- Nil-check contract documented in `Run()`

### Example 3: Event loop with deterministic exit

**File:** [`internal/agentloop/loop.go`](../../internal/agentloop/loop.go)

- Buffered channel (size 64), `defer close` on all paths
- Context cancellation check at loop start and after blocking calls
- Parallel tool execution: `executeCalls` uses `sync.WaitGroup`

### Example 4: Test double (StubGateway)

**File:** [`internal/agentloop/gateway.go`](../../internal/agentloop/gateway.go)

- FIFO queue per model key
- Safe fallback: exhausted queue → `{Type: "done"}`
- `Calls []ModelCall` for black-box test assertions

### Example 5: Deterministic state reconstruction

**File:** [`internal/agentloop/session.go`](../../internal/agentloop/session.go)

- `MessagesFromTurnRecords()` is pure — no mutations
- Handles empty assistant content correctly (API compliance)
- Error tool results propagated as content string (visible to LLM)

## Antipatterns — FORBIDDEN

### ❌ panic() in non-init code

```go
// BAD — crashes the binary on user input
buf, err := json.Marshal(payload)
if err != nil {
    panic(fmt.Sprintf("marshal: %v", err))
}

// GOOD
buf, err := json.Marshal(payload)
if err != nil {
    return fmt.Errorf("marshal workgraph hash: %w", err)
}
```

`panic()` is only acceptable in `init()` for truly unrecoverable setup (missing required env at startup).

### ❌ Global mutable state (package-level vars)

```go
// BAD — race condition, hidden dependency, hard to test
var currentLevel logLevel

func logDebug(format string, args ...interface{}) {
    if currentLevel <= levelDebug {
        log.Printf(format, args...)
    }
}

// GOOD — inject via struct
type Logger struct {
    level logLevel
}
func (l *Logger) Debug(format string, args ...interface{}) { ... }
```

### ❌ Business logic in cmd/

```go
// BAD — cmd/sdp-foo/main.go with 200 lines of logic
func main() {
    items := loadItems()
    for _, item := range items {
        if item.Status == "ready" && item.Priority < 3 {
            // ... complex filtering logic ...
        }
    }
}

// GOOD — main delegates
func main() {
    cfg := parseFlags()
    if err := foo.Run(cfg); err != nil {
        log.Fatal(err)
    }
}
```

### ❌ String comparison for errors

```go
// BAD — breaks when error message changes
if err.Error() == "connection refused" {
    retry()
}

// GOOD — use errors.Is / errors.As
var netErr net.Error
if errors.As(err, &netErr) && netErr.Timeout() {
    retry()
}

// GOOD — sentinel errors
if errors.Is(err, ErrHarnessTerminated) {
    return
}
```

### ❌ Deleting tests to fix flakiness

```go
// BAD
// (deleted integration test because it was flaky in CI)

// GOOD — skip in CI, keep test
func TestIntegration_LiveGateway(t *testing.T) {
    if testing.Short() {
        t.Skip("integration: requires live LLM endpoint")
    }
    // ...
}
```

CI runs `go test -short ./...`. Integration tests use `testing.Short()` guard.

### ❌ System-command tests without a Short() guard

Tests that shell out to `git`, `bd`, `go build`, or any external binary are **integration tests** — they depend on the environment. Guard them:

```go
// BAD — runs git in every `go test -short` run
func TestGitClean_DirtyRepo(t *testing.T) {
    dir := t.TempDir()
    // git init, write files ...
}

// GOOD
func TestGitClean_DirtyRepo(t *testing.T) {
    if testing.Short() {
        t.Skip("integration: requires git in PATH")
    }
    dir := t.TempDir()
    // git init, write files ...
}
```

## Typical File Template

```go
// Package foo implements [one-line purpose].
package foo

import (
    "context"
    "fmt"

    // stdlib first, then third-party, then internal
    "github.com/fall-out-bug/sdp_lab/internal/bar"
)

// Foo [exported type description].
type Foo struct {
    cfg   Config
    store Store  // injected interface
}

// Config holds Foo construction parameters.
type Config struct {
    MaxRetries int
    Store      Store // required
}

// NewFoo constructs a Foo. Returns error if required fields are missing.
func NewFoo(cfg Config) (*Foo, error) {
    if cfg.Store == nil {
        return nil, fmt.Errorf("NewFoo: cfg.Store must not be nil")
    }
    return &Foo{cfg: cfg, store: cfg.Store}, nil
}

// DoThing [verb phrase describing what it does].
func (f *Foo) DoThing(ctx context.Context, input bar.Input) (bar.Result, error) {
    if input.ID == "" {
        return bar.Result{}, fmt.Errorf("DoThing: input.ID is required")
    }
    // ...
    return bar.Result{}, nil
}
```

**Import order:** stdlib → third-party → `sdp_dev/internal/...` (enforced by `goimports`).

**File layout per package:**
1. `types.go` — interfaces, enums, shared structs
2. `foo.go` — main type + constructor + methods
3. `store.go` — storage interface + implementations
4. `gateway.go` — external interface + stub (test double)
5. `foo_test.go` — unit tests (table-driven, `t.Run`)

## Quality Gates (run before every push)

```bash
./scripts/run_go_quality_gates.sh                        # build + test + vet (Docker)
SDP_GO_QUALITY_MODE=host ./scripts/run_go_quality_gates.sh   # fallback, no Docker
```

CI runs `go test -short ./...` — integration tests must use `t.Skip` with `testing.Short()`.
