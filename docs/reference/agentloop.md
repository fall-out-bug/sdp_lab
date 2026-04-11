# SDP Mini-Harness — Руководство по использованию

`internal/agentloop` — Go-пакет и CLI для запуска AI-сессий разработки с персистентным FSM-состоянием.
Сессия разбита на **фазы** (discover → plan → build → review → eval). После каждой фазы — **gate**:
автоматическая проверка качества. При провале gate управление передаётся человеку.

---

## Быстрый старт

### 1. Собрать бинарь

```bash
cd /path/to/sdp_lab
go build -o sdp-harness ./cmd/sdp-harness/
```

### 2. Создать сессию

```bash
sdp-harness new --session=my-feature-001
# session "my-feature-001" created at /Users/<you>/.sdp/my-feature-001.db
```

### 3. Запустить первую фазу

```bash
sdp-harness run \
  --session=my-feature-001 \
  --prompt="Исследуй конкурентов в пространстве AI coding assistants"
# phase turn complete for session "my-feature-001"
```

### 4. Продолжить следующую фазу

```bash
sdp-harness run \
  --session=my-feature-001 \
  --prompt="Составь план реализации фичи X на основе найденных конкурентов"
```

Сессия сохраняется между запусками. Процесс может упасть — перезапустить можно тем же `run`.

---

## CLI Reference

### `sdp-harness new`

Создаёт новую сессию.

```
sdp-harness new --session=<id>
```

| Флаг | Обязателен | Описание |
|------|-----------|----------|
| `--session` | ✓ | Уникальный ID сессии (строка) |

Создаёт файл `$SDP_DATA_DIR/<id>.db` (SQLite, WAL mode).
Если файл уже существует — ошибка.

---

### `sdp-harness run`

Запускает один turn текущей фазы.

```
sdp-harness run --session=<id> --prompt="<text>" [--token=<tok>]
```

| Флаг | Обязателен | Описание |
|------|-----------|----------|
| `--session` | ✓ | ID сессии (должна существовать) |
| `--prompt` | ✓ | Пользовательский запрос для этого turn |
| `--token` | — | Owner token (нужен только если сессия была создана с токеном) |

**Поведение:**
- Восстанавливает сессию из SQLite
- Строит сообщения из истории turn'ов
- Запускает `Loop.Run()` (LLM → инструменты → loop)
- Если агент вызвал `completion_signal` → проверяет gate
- Если gate passed → переходит к следующей фазе
- Если gate escalated → выводит `human_gate` событие с `DecisionID`
- Если gate ещё не вызван → ждёт следующего `run`

---

### Переменные окружения

| Переменная | По умолчанию | Описание |
|------------|-------------|----------|
| `SDP_DATA_DIR` | `$HOME/.sdp` | Директория для `.db` файлов сессий |

---

## Жизненный цикл сессии

```
new
 │
 ▼
 [discover] ──run──► agent works ──completion_signal──► gate
                                                          │
                                                    passed │ escalated
                                                          │    │
                                                          ▼    ▼
                                                       [plan]  human_gate
                                                          │    (ApproveGate / Rollback)
                                                         ...
                                                          ▼
                                                       [eval] ──► done
```

### FSM-состояния Harness

| Состояние | Описание | Что разрешено |
|-----------|----------|--------------|
| `idle` | Готов к следующему prompt | `run` |
| `running` | Loop.Run активен | Ничего (конкурентные `run` возвращают ошибку) |
| `awaiting_human` | Gate escalated, ждёт решения | `ApproveGate` / `Rollback` / `Stop` (через Go API) |
| `stopped` | Терминальное состояние | Ничего (RestoreHarness вернёт ошибку) |

---

## Фазы и модели

| Фаза | Модели (в порядке приоритета) | Доступные инструменты | Gate |
|------|------------------------------|----------------------|------|
| `discover` | deepseek/deepseek-v3.2, openai/gpt-4.1 | web_search, read_file, bd_search | ✓ |
| `plan` | openai/gpt-4.1, anthropic/claude-opus-4-5 | read_file, glob, bd_create | ✓ |
| `build` | anthropic/claude-sonnet-4-6, openai/gpt-4.1 | read_file, edit_file, bash, glob | ✓ |
| `review` | openai/gpt-4.1, deepseek/deepseek-v3.2 | read_file, grep, bd_comment | ✓ |
| `eval` | anthropic/claude-sonnet-4-6, openai/gpt-4.1 | bash, read_file | ✓ |

`completion_signal` добавляется **автоматически** в каждую фазу — его не нужно включать в `PhaseConfig.Tools`.

---

## Работа с Gate Escalation (Go API)

Когда gate escalated, CLI выводит событие типа `human_gate` с полем `Delta = "<decisionID>"`.
Дальнейшее управление — через Go API:

```go
// Восстановить harness
store, _ := agentloop.NewSQLiteStore(dbPath)
router := agentloop.NewPhaseRouter(agentloop.DefaultPhaseMap, registry, gateway, nil)
gate := agentloop.NewGateEngine(nil, 0)
h, _ := agentloop.RestoreHarness(sessionID, token, store, router, gate, nil)

// Одобрить переход в следующую фазу
err := h.ApproveGate(ctx, decisionID, token)

// Откатиться к предыдущей фазе (RecoveryNext)
err := h.Rollback(ctx, decisionID, token)

// Завершить сессию насовсем
err := h.Stop(ctx, token)
```

`decisionID` имеет формат `"<sessionID>-run<N>"` — виден в событии `human_gate.Delta`.

---

## Подключение реальных инструментов и LLM Gateway

MVP-режим CLI использует `StubGateway` (без реального LLM). Для production:

```go
// 1. Реализовать ModelGateway
type MyGateway struct{}
func (g *MyGateway) IsAvailable(model string) bool { ... }
func (g *MyGateway) Call(ctx context.Context, msgs []agentloop.Message, cfg agentloop.LoopConfig) (<-chan agentloop.Event, error) { ... }

// 2. Зарегистрировать инструменты
tools := []agentloop.Tool{
    {
        Name: "bash",
        Execute: func(ctx context.Context, id string, args json.RawMessage) (string, error) {
            // выполнить bash команду
        },
    },
    // ... остальные инструменты
}
registry := agentloop.NewToolRegistry(tools)

// 3. Собрать router с реальным gateway
gateway := &MyGateway{}
router := agentloop.NewPhaseRouter(agentloop.DefaultPhaseMap, registry, gateway, nil)

// 4. Запустить RunPhase напрямую (без CLI)
h, _ := agentloop.RestoreHarness(sessionID, token, store, router, gate, nil)
err := h.RunPhase(ctx, userPrompt, token)
```

---

## Восстановление после сбоя

Сессия сохраняется в SQLite после каждого turn. Если процесс упал:

```bash
# Просто перезапустить с тем же session ID
sdp-harness run --session=my-feature-001 --prompt="..."
```

`RestoreHarness` автоматически:
- Восстанавливает историю turn'ов из `turn_records`
- Восстанавливает текущую фазу из последнего `phase_records.next_phase`
- Если был pending gate → устанавливает состояние `awaiting_human`
- Если сессия была остановлена через `Stop()` → возвращает ошибку

---

## Структура данных

SQLite-файл `$SDP_DATA_DIR/<sessionID>.db`:

| Таблица | Содержимое |
|---------|-----------|
| `sessions` | Один ряд: ID, начальная фаза (`discover`) |
| `turn_records` | Append-only лог каждого `RunPhase` (сообщения, tool calls, tool results) |
| `phase_records` | Лог переходов между фазами (current → next, snapshot evidence) |
| `decisions` | Максимум один pending decision на сессию |
| `events` | Все Event объекты (text_delta, tool_end, error, warn, ...) |
| `gate_results` | ComplianceReport за каждую gate проверку |

---

## EvidenceAccumulator — автоматический сбор доказательств

Gate не верит самоотчётам агента. Доказательства собираются из результатов инструментов:

| Инструмент | Что собирается |
|-----------|---------------|
| `bash` | `quality["test"] = true/false` (по наличию PASS/FAIL в выводе) |
| `edit_file` | `evidence: "file_modified:<path>"` |
| `bd_create` | `evidence: "card_created:<id>"` |
| Любой (ошибка) | `evidence: "tool_error:<name>:<err>"` |

Evidence сбрасывается при каждом переходе фазы (`transitionTo` вызывает `accumulator.Reset()`).

---

## Типичные ошибки

| Ошибка | Причина | Решение |
|--------|---------|---------|
| `"no subcommand given"` | Запущен без subcommand | `sdp-harness new ...` или `sdp-harness run ...` |
| `"restore session: ..."` | Сессия не найдена | Сначала `sdp-harness new --session=<id>` |
| `"harness busy: state=1"` | Параллельный `run` | Дождаться завершения предыдущего |
| `"harness busy: state=2"` | Gate escalated | Вызвать `ApproveGate` или `Rollback` через Go API |
| `"session was terminated"` | `Stop()` вызван | Создать новую сессию |
| `"no available model for phase"` | Gateway не знает модель | Проверить `IsAvailable()` в gateway реализации |
