# SDP Mini-Harness (agentloop) — Справка

## Место в SDP Pipeline

`internal/agentloop` — это **execution kernel** SDP: многофазный agentic loop с
персистентным FSM-состоянием, заменяющий одиночный `opencode`-вызов в `ExecutorBridge`.

```
sdp dispatch next --execute
        │
        ▼
  ExecutorBridge.DispatchAndRun()
        │
        │  СЕЙЧАС: orchestrate.LLMInvoker → opencode subprocess
        │  ЦЕЛЬ:   agentloop.RestoreHarness → h.RunPhase() per phase
        │
        ▼
  executor-results/   ←── outcome + evidence
        │
        ▼
  sdp orchestrate once  (auto-ingest)
```

**Вместо одного prompt → opencode**, harness прогоняет карточку через полный цикл:

```
Beads card (ready)
  │
  ▼ dispatcher строит userPrompt из objective + acceptance criteria
  │
  ├─ [discover] research phase  ──gate──► pass/escalate
  ├─ [plan]     beads task creation ──gate──► pass/escalate
  ├─ [build]    TDD implementation  ──gate──► pass/escalate
  ├─ [review]   code review         ──gate──► pass/escalate
  └─ [eval]     final verification  ──gate──► done / human escalation
```

---

## Статус интеграции

| Компонент | Статус |
|-----------|--------|
| `internal/agentloop` — ядро (Loop, Harness, GateEngine, PhaseRouter) | ✅ реализован, 128 тестов |
| `cmd/sdp-harness` — standalone MVP CLI | ✅ реализован |
| `ExecutorBridge` → `agentloop` wiring | ⬜ не реализован (OpenCode сейчас) |
| `sdp dispatch next --execute` → harness | ⬜ не реализован |
| Evidence → `.sdp/evidence/<card-id>.json` | ⬜ не реализован |

**Для production использования нужно:** реализовать wiring в `ExecutorBridge.DispatchAndRun()`
(заменить `LLMInvoker.Invoke` → `agentloop.RestoreHarness` + `h.RunPhase`).

---

## Standalone MVP: sdp-harness CLI

До интеграции с `sdp dispatch` — CLI для ручного запуска harness на карточке.

### Сборка

```bash
go build -o sdp-harness ./cmd/sdp-harness/
```

### Полный цикл вручную

```bash
# 1. Найти готовую карточку в beads
bd show sdplab-XYZ   # прочитать objective + acceptance criteria

# 2. Создать harness-сессию (ID = card ID)
sdp-harness new --session=sdplab-XYZ

# 3. Запустить discover (prompt = objective карточки)
sdp-harness run \
  --session=sdplab-XYZ \
  --prompt="$(bd show sdplab-XYZ --field=description)"

# 4. Продолжать run до eval фазы (каждый run — один turn,
#    автоматический переход фаз через completion_signal + gate)
sdp-harness run --session=sdplab-XYZ --prompt="Continue"

# 5. Если gate escalated — увидишь: human_gate <decisionID>
#    Решить через Go API (ApproveGate / Rollback / Stop)
```

Сессия персистируется в `$SDP_DATA_DIR/<id>.db` (default: `$HOME/.sdp/sdplab-XYZ.db`).
После падения процесса — просто перезапустить `sdp-harness run` с тем же session ID.

### CLI Reference

```
sdp-harness new  --session=<id>
sdp-harness run  --session=<id> --prompt="<text>" [--token=<tok>]

Env: SDP_DATA_DIR   (default: $HOME/.sdp)
```

| Флаг | Команда | Описание |
|------|---------|----------|
| `--session` | оба | ID сессии = ID beads-карточки (строка) |
| `--prompt` | run | Контекст для этого turn (objective, acceptance criteria) |
| `--token` | run | Owner token (если сессия была создана с токеном) |

---

## Интеграция с ExecutorBridge (целевое состояние)

Чтобы `sdp dispatch next --execute` использовал harness, нужно обновить
`internal/executor/bridge.go`:

```go
// DispatchAndRun — ТЕКУЩАЯ реализация вызывает opencode.
// ЦЕЛЬ: заменить LLMInvoker.Invoke() → agentloop Harness.
func (b *ExecutorBridge) DispatchAndRun(ctx context.Context, projectID, cardID string) (*ExecutorResultPacket, error) {
    packet, _ := b.Store.LoadExecutionPacket(projectID, cardID)
    card, _    := b.Store.LoadCard(projectID, cardID)

    // 1. Создать/восстановить harness-сессию для этой карточки
    dbPath := filepath.Join(b.ProjectRoot, ".sdp", "sessions", cardID+".db")
    store, _ := agentloop.NewSQLiteStore(dbPath)

    registry := agentloop.NewToolRegistry(buildSDPTools(b.ProjectRoot)) // bash, read_file, edit_file, glob, bd_*
    gateway  := buildOpenRouterGateway(os.Getenv("OPENROUTER_API_KEY"))
    router   := agentloop.NewPhaseRouter(agentloop.DefaultPhaseMap, registry, gateway, nil)
    gate     := agentloop.NewGateEngine(buildContract(card), 0)

    h, _ := agentloop.RestoreHarness(cardID, "", store, router, gate, nil)

    // 2. Сформировать prompt из пакета (objective + acceptance criteria)
    prompt := buildPromptFromPacket(packet, card)

    // 3. Запустить один phase turn
    if err := h.RunPhase(ctx, prompt, ""); err != nil {
        return &ExecutorResultPacket{Status: "failed", Error: err.Error()}, nil
    }

    // 4. Вернуть результат
    return &ExecutorResultPacket{Status: "completed", CardID: cardID}, nil
}
```

---

## Фазы и модели

| Фаза | Модели (приоритет) | Инструменты | SDP операции |
|------|-------------------|-------------|-------------|
| `discover` | deepseek-v3.2, gpt-4.1 | web_search, read_file, bd_search | `bd search`, поиск аналогов |
| `plan` | gpt-4.1, claude-opus-4-5 | read_file, glob, bd_create | `bd create` sub-tasks |
| `build` | claude-sonnet-4-6, gpt-4.1 | read_file, edit_file, bash, glob | TDD, `git commit`, `go test` |
| `review` | gpt-4.1, deepseek-v3.2 | read_file, grep, bd_comment | `bd comment` findings |
| `eval` | claude-sonnet-4-6, gpt-4.1 | bash, read_file | `go test -race`, `go vet` |

`completion_signal` добавляется автоматически в каждую фазу — не включать в `PhaseConfig.Tools`.

---

## Gate Escalation и Human-in-the-Loop

При провале gate → событие `human_gate` с `DecisionID`. Три варианта действий:

```go
store, _ := agentloop.NewSQLiteStore(dbPath)
// ... восстановить router, gate ...
h, _ := agentloop.RestoreHarness(cardID, token, store, router, gate, nil)

// Одобрить — перейти в следующую фазу
h.ApproveGate(ctx, decisionID, token)

// Откатиться — вернуться в RecoveryNext фазу
h.Rollback(ctx, decisionID, token)

// Завершить — закрыть сессию (и закрыть карточку)
h.Stop(ctx, token)
```

`decisionID` виден в событии `human_gate.Delta`, формат: `"<sessionID>-run<N>"`.

---

## EvidenceAccumulator — автоматический сбор proof

Gate не верит самоотчётам агента. Доказательства — только из tool results:

| Tool | Evidence |
|------|----------|
| `bash` с PASS в выводе | `quality["test"] = true` |
| `bash` с FAIL в выводе | `quality["test"] = false` |
| `edit_file` | `"file_modified:<path>"` |
| `bd_create` | `"card_created:<id>"` |
| любой tool → ошибка | `"tool_error:<name>:<err>"` |

Evidence сбрасывается при каждом `transitionTo` (переход фазы).

---

## FSM-состояния Harness

| Состояние | Что разрешено |
|-----------|--------------|
| `idle` | RunPhase — новый turn |
| `running` | ничего (конкурентный run → ошибка) |
| `awaiting_human` | ApproveGate / Rollback / Stop |
| `stopped` | ничего (RestoreHarness → ошибка) |

---

## Типичные ошибки

| Ошибка | Причина | Решение |
|--------|---------|---------|
| `"restore session: not found"` | Сессия не существует | `sdp-harness new --session=<id>` |
| `"harness busy: state=1"` | Параллельный run | Дождаться завершения |
| `"harness busy: state=2"` | Gate escalated | ApproveGate / Rollback через Go API |
| `"session was terminated"` | Stop() вызван | Создать новую сессию |
| `"no available model"` | Gateway не знает модель | Проверить IsAvailable() |
