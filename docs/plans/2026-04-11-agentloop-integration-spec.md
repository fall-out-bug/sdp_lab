# F106 — agentloop Integration Spec

**Date:** 2026-04-11  
**Feature:** F106  
**Status:** Active v1.0 (council-validated 2026-04-11)  
**Owner:** Андрей  
**Council validation:** `docs/plans/2026-04-11-council-sdp-full.md`

---

## 1. Проблема

SDP заявляет governance, но реальный execution path:

```
ServeBridge.DispatchAndRun()
  → omoclient.GovernanceWrapper.PreCall()   ← проверяет scope на входе
  → omoclient.ServeInvoker / openCodeInvoker ← subprocess или SSE до opencode serve
  → exit code + stdout                       ← единственное что видит SDP
  → omoclient.GovernanceWrapper.PostCall()  ← парсит stdout, проверяет scope
```

SDP не видит tool calls, не может остановить агента, не знает фазу. Evidence = stdout. Gates = post-hoc audit, не enforcement.

`internal/agentloop` содержит корректное решение: stateful Harness FSM, PhaseRouter, GateEngine с circuit breaker, EvidenceAccumulator из реальных tool outputs. 128 тестов, -race clean. Нулевых caller'ов за пределами пакета.

---

## 2. Цель

Подключить `agentloop` в `ServeBridge.DispatchAndRun()` как основной execution path.

После реализации:
- SDP владеет каждым tool call через `agentloop.ToolRegistry`
- Gate evaluation происходит по реальному tool output evidence, не stdout
- Фаза карточки tracked в `SessionStore` (SQLite WAL)
- При crash агента → terminal record в SQLite → состояние восстановимо

---

## 3. Scope

**В scope:**
- `internal/agentloop/gateway_live.go` — `LiveGateway` реализует `ModelGateway`
- `internal/agentloop/tools_live.go` — реальные Tool implementations
- `internal/executor/bridge_serve.go` — Harness wiring в `DispatchAndRun`
- `internal/executor/bridge_serve.go` — crash reconciliation defer
- Integration test: `internal/executor/bridge_serve_integration_test.go`
- `docs/CANONICAL_SDP_PIPELINE.md` — Status: Active, gate classification

**Вне scope:**
- Streaming в ModelGateway adapters (Phase 2)
- in-toto attestation rewrite (Phase 2)
- K8s deployment (Phase 9)
- Удаление `ExecutorBridge` (старый subprocess bridge, оставить как fallback)
- web_search tool implementation

---

## 4. Архитектура изменений

### 4.1 LiveGateway

```go
// internal/agentloop/gateway_live.go

type LiveGateway struct {
    router *modelrouter.Router // wraps modelgateway provider adapters
}

func NewLiveGateway(apiKey string) *LiveGateway

// IsAvailable: delegates to router.IsAvailable(model)
// Call: translates []agentloop.Message → modelgateway.ChatRequest
//       makes non-streaming call (MVP)
//       emits: text_delta events + tool_call events + done
//       maps modelgateway.ToolCall → agentloop.ToolCall (ID, Name, Input)
//
// IMPORTANT: LiveGateway MUST generate a UUID for each ToolCall.ID.
//   If the provider returns an empty or missing tool_call.id, LiveGateway
//   generates one via uuid.NewString(). Downstream Harness correlation
//   depends on unique non-empty IDs.
//
// IMPORTANT: NewLiveGateway MUST validate apiKey != "".
//   Return error (not panic, not silent empty) if key is absent.
```

**Модели (из MODEL_POLICY.md):**
```
glm-5, glm-4.7 (default)
anthropic/claude-sonnet-4.6, anthropic/claude-opus-4.6
openai/gpt-5.2-codex
minimax/minimax-m2.5
moonshotai/kimi-k2.5
```

`IsAvailable()` для MVP: возвращает `true` если модель в allowlist (из MODEL_POLICY.md). Не делает probe call — это Phase 2.

### 4.2 Live Tools

```go
// internal/agentloop/tools_live.go

func BashTool(workdir string) Tool
func ReadFileTool(root string) Tool
func EditFileTool(root string) Tool
func GlobTool(root string) Tool
func GrepTool(root string) Tool
func BdSearchTool(store *control.Store) Tool
func BdCreateTool(store *control.Store) Tool
func BdCommentTool(store *control.Store) Tool

func BuildLiveTools(projectRoot string, store *control.Store) []Tool
```

**Критически важно для EvidenceAccumulator:**

`EvidenceAccumulator.OnToolResult()` (`internal/agentloop/evidence.go`) смотрит на:
- `tool_name == "bash"` + `PASS`/`ok ` в output → `quality["test"] = true`
- `tool_name == "bash"` + `FAIL` → `quality["test"] = false`
- `tool_name == "edit_file"` → `artifacts = append(artifacts, "file_modified:<path>")`
- `tool_name == "bd_create"` → `artifacts = append(artifacts, "card_created:<id>")`

Инструменты должны возвращать output совместимый с этими паттернами.

**Bash tool constraints:**
- `workdir` locked to `projectRoot` (no `../../` traversal)
- timeout: 60s default, max 300s через `{"timeout": N}` в args
- stdin: не предоставляется (non-interactive only)
- Нет PTY для MVP — только pipe stdout+stderr

**Bash tool security (обязательно):**  
`cmd.Dir = workdir` **не предотвращает** `cd / && rm -rf /`. Требуется одно из:
1. **Дополнительный prefix check:** после `cd` в команде — ошибка. Статически отклонять команды содержащие `cd ` (пробел после cd). Это blunt но sufficient для MVP.
2. **Явный deny-list:** команды `rm -rf`, `chmod 777 /`, `dd if=/dev/zero` → reject без выполнения.

Реализовать вариант 2 (deny-list + prefix-safe path check). Неприемлемые команды → ошибка `"command denied: <reason>"`, не выполнять.

**Schema для bash:**
```json
{
  "type": "object",
  "properties": {
    "command": {"type": "string"},
    "timeout": {"type": "integer", "default": 60}
  },
  "required": ["command"]
}
```

### 4.3 ServeBridge wiring

```go
// internal/executor/bridge_serve.go

type ServeBridge struct {
    // ... existing fields ...
    
    // Harness components (nil = use legacy OmO path)
    harnessRouter *agentloop.PhaseRouter
    harnessGate   *agentloop.GateEngine
    harnessData   string // dir для SQLite session files (default: projectRoot/.sdp/sessions)
}

func NewServeBridge(store *control.Store, projectRoot string) *ServeBridge {
    // ... existing init ...
    liveGW := agentloop.NewLiveGateway(os.Getenv("OPENROUTER_API_KEY"))
    tools  := agentloop.BuildLiveTools(projectRoot, store)
    reg    := agentloop.NewToolRegistry(tools)
    router := agentloop.NewPhaseRouter(agentloop.DefaultPhaseMap, reg, liveGW, nil)
    gate   := agentloop.NewGateEngine(nil, 0)
    
    sb.harnessRouter = router
    sb.harnessGate   = gate
    sb.harnessData   = filepath.Join(projectRoot, ".sdp", "sessions")
    return sb
}
```

В `DispatchAndRun()`:

```go
func (b *ServeBridge) DispatchAndRun(ctx, projectID, cardID) (*ExecutorResultPacket, error) {
    // ... existing: load card, build packet, build governed prompt ...
    
    // Harness path (если harnessRouter настроен)
    if b.harnessRouter != nil {
        return b.runWithHarness(ctx, card, packet, governedPrompt)
    }
    
    // Legacy OmO path (fallback)
    return b.runWithOmO(ctx, card, packet, governedPrompt)
}

func (b *ServeBridge) runWithHarness(ctx, card, packet, prompt) (*ExecutorResultPacket, error) {
    dbPath := filepath.Join(b.harnessData, card.ID+".db")
    store, err := agentloop.NewSQLiteStore(dbPath)
    if err != nil { return nil, err }
    
    h, err := agentloop.RestoreHarness(card.ID, "", store, b.harnessRouter, b.harnessGate, nil)
    if err != nil { return nil, err }
    
    // Reconciliation: при crash h.Stop() запишет terminal record
    var stopped bool
    defer func() {
        if !stopped {
            _ = h.Stop(context.Background(), "")
        }
    }()
    
    if err := h.RunPhase(ctx, prompt, ""); err != nil {
        return &ExecutorResultPacket{Status: "failed", Error: err.Error()}, nil
    }
    
    stopped = true
    return &ExecutorResultPacket{Status: "completed", CardID: card.ID}, nil
}
```

**Важно:** `defer h.Stop()` вызывается только при crash/panic — при успешном RunPhase `stopped = true` перед выходом из defer.

### 4.4 Mapping карточки → фаза agentloop

```go
func cardPhase(card *control.FeatureCard) agentloop.Role {
    switch card.Phase {
    case "build":   return agentloop.RoleBuild
    case "review":  return agentloop.RoleReview
    case "qa":      return agentloop.RoleEval
    case "plan":    return agentloop.RolePlan
    default:        return agentloop.RoleDiscover
    }
}
```

Для MVP: `RestoreHarness` восстанавливает фазу из SQLite. При новой карточке — starts at `RoleDiscover`.

---

## 5. Тест-план

### 5.1 Unit тесты (в каждом WS)

- `gateway_live_test.go`: `httptest.Server` мокирует provider endpoint. Тесты: `IsAvailable` false при server down, `Call` возвращает text_delta события.
- `tools_live_test.go`: каждый tool с temp dir. Bash test: проверить что `PASS` в output → evidence accumulator видит его.
- `bridge_serve_test.go`: использует `StubGateway` + in-memory store. Тест: card dispatch → Harness → gate evaluation → result.

### 5.2 Integration test

```go
// internal/executor/bridge_serve_integration_test.go
// +build integration

func TestServeBridgeHarnessRealLLM(t *testing.T) {
    // Requires: OPENROUTER_API_KEY env var
    // 1. Create real ServeBridge with LiveGateway
    // 2. Create test card in temp control.Store
    // 3. Call DispatchAndRun
    // 4. Assert: session file exists in .sdp/sessions/
    // 5. Assert: at least one phase completed
    // 6. Assert: evidence accumulated (not empty)
}
```

Этот тест является failing до завершения всех WS. Passing = feature done.

---

## 6. Gate classification (для pipeline doc)

| Gate | Type | Mandatory/Advisory после F106 |
|------|------|-------------------------------|
| `contract-approve` | human | Mandatory |
| `scope-review` | human | Mandatory (if out-of-scope detected) |
| `review-pass` | automated | **Mandatory** — GateEngine evaluates from tool evidence |
| `qa-pass` | automated | **Mandatory** — GateEngine evaluates from tool evidence |
| `ci` | automated | Mandatory |
| `staging-approve` | human | Mandatory |
| `prod-approve` | human | Mandatory |

После F106: `review-pass` и `qa-pass` gates имеют реальный evidence. До — advisory only.

---

## 7. Риски

| Риск | Вероятность | Митигация |
|------|-------------|-----------|
| modelgateway ↔ agentloop type mismatch | Высокая | Wrapper struct в `gateway_live.go`, не менять оригиналы |
| Bash tool security (directory traversal) | Средняя | `filepath.Clean` + prefix check vs `projectRoot` |
| SQLite WAL locking при concurrent dispatches | Средняя | Один `Harness` per `cardID`; `ServeBridge.MaxConcurrent` уже ограничен |
| LLM returns empty tool calls | Средняя | `StubGateway` fallback pattern уже обрабатывает это в `Run()` |
| Legacy OmO path breakage | Низкая | Nil check: если `harnessRouter == nil` → old path. Existing tests не изменяются |

---

## 8. Зависимости между WS

```
00-106-01 (LiveGateway)
  └── 00-106-02 (Live Tools)
        └── 00-106-03 (ServeBridge wiring)
              └── 00-106-04 (crash reconciliation)
                    └── 00-106-05 (integration test)

00-106-06 (pipeline doc) — независим, можно параллельно с 00-106-04
```

---

## 9. Gate-fail semantics

Когда `GateEngine` блокирует фазу (gate не пройден):

| Ситуация | Семантика | Поведение |
|----------|-----------|-----------|
| `gate-pending` (human gate) | Awaiting human approval | Harness state = AwaitingHuman. Session жива. `DispatchAndRun` возвращает `status: "gate-pending"`. |
| Automated gate fail (review-pass/qa-pass) | Retryable by default | Harness state = Running (может продолжить в следующем turn). `RunPhase` возвращает ошибку с типом `*agentloop.GateError`. |
| Session terminated (crash + Stop) | Terminal | `RestoreHarness` вернёт `ErrHarnessTerminated`. Старый db удаляется, новый создаётся. |

**`ErrHarnessTerminated`** — sentinel variable (не строка):
```go
// internal/agentloop/harness.go
var ErrHarnessTerminated = errors.New("harness: session was terminated")
```

WS-04 проверяет `errors.Is(err, agentloop.ErrHarnessTerminated)`, а не `strings.Contains(err.Error(), "was terminated")`.

---

## 10. Governance KPIs (F106 done = когда)

F106 считается завершённой когда:

| KPI | Целевое значение | Измерение |
|-----|-----------------|-----------|
| Integration test pass | `go test -tags integration` PASS | CI job |
| Automated gate enforcement | ≥1 `review-pass` или `qa-pass` gate evaluation с реальным evidence в prod session | Session SQLite inspection |
| Legacy OmO path stable | Все существующие `./internal/executor/...` тесты PASS | CI |
| Evidence non-empty | Session содержит ≥1 TurnRecord с tool_result | SQLite |

---

## 11. Implementation bug fix (WS-03)

В оригинальном implementation note WS-03 есть баг — recreate store не проверяет ошибку второго `NewSQLiteStore`. Корректная версия:

```go
h, err := agentloop.RestoreHarness(card.ID, "", store, b.harnessRouter, b.harnessGate, nil)
if errors.Is(err, agentloop.ErrHarnessTerminated) {
    // Старая сессия terminated — начать заново
    if removeErr := os.Remove(dbPath); removeErr != nil && !os.IsNotExist(removeErr) {
        return nil, fmt.Errorf("remove terminated session: %w", removeErr)
    }
    store, err = agentloop.NewSQLiteStore(dbPath)
    if err != nil {
        return nil, fmt.Errorf("recreate session store: %w", err)
    }
    h, err = agentloop.RestoreHarness(card.ID, "", store, b.harnessRouter, b.harnessGate, nil)
}
if err != nil {
    return nil, fmt.Errorf("restore harness: %w", err)
}
```

Используем `errors.Is` (sentinel), не `strings.Contains`.
