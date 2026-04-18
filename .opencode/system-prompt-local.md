# SDP Lab — Go Code Rules (for local models)

## Project
- Module: `sdp_dev` (imports: `sdp_dev/internal/...`, NOT `github.com/...`)
- Go 1.26, Metal 4 Mac, SQLite persistence
- Folder: `cmd/` = thin CLI only (≤60 lines); `internal/` = all business logic

## Naming
- Packages: lowercase, no underscores (`agentloop`, `healthcheck`, `harnesscfg`)
- Types: PascalCase; interfaces: noun-style (`ModelGateway`, `SessionStore`)
- Constructors: `NewFoo(deps) (*Foo, error)`
- FSM states: `camelCase iota` (`stateIdle`, `stateRunning`, `stateStopped`)
- Errors: `ErrFoo` sentinel, wrapped with `fmt.Errorf("%w", err)`

## Architecture Patterns (required)

### FSM + mutex guard
```go
type Worker struct {
    mu    sync.Mutex
    state myState
}
func (w *Worker) Start(ctx context.Context) error {
    w.mu.Lock()
    if w.state != stateIdle { w.mu.Unlock(); return fmt.Errorf("busy") }
    w.state = stateRunning
    w.mu.Unlock()
    return nil
}
```

### Interface DI (never instantiate deps in constructors)
```go
type Config struct {
    Gateway ModelGateway // required
    Store   SessionStore // optional
}
func Run(ctx context.Context, cfg Config) error {
    if cfg.Gateway == nil { return fmt.Errorf("Run: Gateway must not be nil") }
    ...
}
```

### Durable-first (persist before mutating memory)
```go
if err := h.store.Persist(record); err != nil { return err }
h.state = next // only after durable commit
```

### Subprocess calls
```go
cmd := exec.CommandContext(ctx, "go", "build", "./...")
cmd.Dir = c.projectRoot // always explicit Dir
out, err := cmd.CombinedOutput()
```

### Tests
```go
if testing.Short() { t.Skip("integration: requires git") }
dir := t.TempDir() // never hardcode temp paths
```

## Anti-patterns (FORBIDDEN)
- `panic()` outside `init()`
- Package-level mutable vars
- Business logic in `cmd/`
- `errors.New("some string")` comparisons — use sentinel `ErrFoo`
- Deleting tests or commenting them out

## Response format
- Return complete Go code, not pseudocode
- Include package declaration and imports
- Follow the naming and patterns above exactly
