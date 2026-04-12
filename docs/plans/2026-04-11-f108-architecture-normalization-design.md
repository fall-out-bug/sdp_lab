# F108: Architecture Normalization — Design

**Date:** 2026-04-11
**Status:** Approved
**Owner:** Андрей
**Feature:** F108

---

## Проблема

Проект имеет пять архитектурных разрывов, которые делают невозможным сборку системы в единое целое:

1. **agentloop не подключён к реальному LLM** — `sdp-harness` использует `StubGateway` (заглушка)
2. **Три независимых LLM-клиента** — `discovery.LLMClient`, `strataudit/model`, `agentloop.ModelGateway` делают одно и то же по-разному
3. **Нет контракта Discovery → Delivery** — GO verdict создаёт Beads issue через shell exec, дальше всё ручное; `FeatureCard` не создаётся, workstream-стаб не создаётся
4. **`ErrHarnessTerminated` — не sentinel** — строковая ошибка вместо `errors.New()`, что нарушает F106 spec и делает обработку хрупкой
5. **`internal/modelgateway` — мёртвый код без явного решения** — 0 production callers, судьба неизвестна

---

## Принципы дизайна

- **Правильно сразу** — MVP-решения запрещены. Временное становится вечным.
- **Интерфейс определяется в пакете потребителя** — не в пакете реализации
- **Один LLM-клиент** — `internal/llmclient` как единственная точка входа для HTTP к OpenRouter
- **Явная роль каждого компонента** — ServeBridge и agentloop сосуществуют, потому что выполняют разные роли

---

## Архитектурные решения

### AD-1: Единый LLM-клиент — `internal/llmclient`

Новый пакет `internal/llmclient` извлекается из `internal/discovery/llm.go`.

**Два метода:**
- `Chat(ctx, req) (*ChatResponse, error)` — для простых запрос-ответ (discovery, strataudit, architect)
- `Stream(ctx, req) (<-chan StreamEvent, error)` — SSE-стриминг для LiveGateway

`StreamEvent` — нейтральный тип (не зависит от agentloop):
```
Type:  "text_delta" | "tool_call" | "finish" | "error"
Text:  string
Tool:  *ToolCallChunk  — накапливается из delta-чанков
```

SSE-парсер аккумулирует delta chunks по `index`. При `finish_reason == "tool_calls"` отправляет финализированные tool calls. При `data: [DONE]` закрывает канал.

UUID генерируется здесь если провайдер вернул пустой tool call ID.

`modelgateway` — изолируется явно (README в пакете: "Future: multi-tenant credential management. 0 production callers. Intentionally not wired.").

**Миграция:**
- `internal/discovery/llm.go` → удаляется, заменяется `llmclient.New()`
- `internal/architect` → если есть собственный HTTP-клиент, заменяется
- `internal/strataudit` → то же

---

### AD-2: `LiveGateway` — полноценная реализация `agentloop.ModelGateway`

Пакет `internal/agentloop/livegw/`.

**Принцип:** реализует интерфейс `agentloop.ModelGateway` через `llmclient.Stream()`.

`Call()` — полный цикл конвертации:
```
[]agentloop.Message
  → llmclient.ChatRequest (+ Tools из LoopConfig)
  → llmclient.Stream()
  → накопление tool_call chunks
  → <-chan agentloop.Event
```

Маппинг событий:
```
llmclient text_delta  → agentloop text_delta
llmclient tool_call   → накапливать → agentloop tool_call (при finish_reason=tool_calls)
llmclient finish      → agentloop turn_end + done
llmclient error       → agentloop error
```

`IsAvailable(model)` — не блокирующий. Проверяет наличие API-ключа, не делает сетевых вызовов.

`New(apiKey, baseURL) (*LiveGateway, error)` — возвращает ошибку если `apiKey == ""`.

---

### AD-3: `ErrHarnessTerminated` — sentinel

В `internal/agentloop/harness.go`:
- `var ErrHarnessTerminated = errors.New("harness: session was terminated")` — экспортированная sentinel-переменная
- `RestoreHarness` оборачивает через `%w` — все вызывающие используют `errors.Is()`
- Строковое сравнение запрещено

---

### AD-4: `sdp-harness` — wire с LiveGateway

`cmd/sdp-harness/main.go`:
- `NewStubGateway()` заменяется на `livegw.New()`
- API-ключ из `OPENROUTER_API_KEY` env var
- Если ключ пуст — явная ошибка, не silent fallback

---

### AD-5: Discovery → Delivery — контракт перехода

На GO verdict `cmd/sdp/cmd_discover.go` выполняет три действия:

1. **Beads feature issue** — уже есть, оставляем
2. **`control.FeatureCard`** — создаётся с полями:
   - `NormalizedIntent` = `frame.ProblemStatement`
   - `DiscoveryDir` = путь к `docs/discovery/<slug>/` (новое поле)
   - `Status` = `"shaping"`
3. **Workstream-стаб** — `docs/workstreams/backlog/00-FFF-01.md` с:
   - `Goal` = `frame.ProblemStatement`
   - `Acceptance Criteria` = `hypothesis.Requirements` (из Phase 2)
   - `Out of Scope` = `frame.Scope`
   - `discovery_dir` = ссылка на артефакты

`DiscoveryDir` в `FeatureCard` — контракт между Discovery и Delivery. Delivery-агент читает его, не ищет по slug.

---

### AD-6: Разделение ролей ServeBridge и agentloop

**Явное решение:** они не дублируют друг друга.

| | ServeBridge | agentloop.Harness |
|---|---|---|
| **Роль** | Dispatch к внешнему харнессу | SDP сам является агентом |
| **Цель** | Claude Code, Cursor, opencode serve | Внутренний LLM loop SDP |
| **Статус** | Production | Production после F108 |

Оба компонента получают документальный комментарий о своей роли.

---

## Воркстримы

| WS | Задача | Размер | Зависит от |
|----|--------|--------|-----------|
| 00-108-01 | `internal/llmclient` — Chat + Stream + SSE-парсер | M | — |
| 00-108-02 | Миграция discovery/architect/strataudit на llmclient | S | 00-108-01 |
| 00-108-03 | `LiveGateway` в `internal/agentloop/livegw` | M | 00-108-01 |
| 00-108-04 | `ErrHarnessTerminated` sentinel + все callers | XS | — |
| 00-108-05 | Wire `sdp-harness` с LiveGateway + интеграционный тест | S | 00-108-03, 00-108-04 |
| 00-108-06 | Discovery → Delivery: FeatureCard + WS stub на GO | M | — |
| 00-108-07 | Документация ролей: modelgateway, ServeBridge, agentloop | XS | — |

DAG:
```
00-108-01 → 00-108-02
00-108-01 → 00-108-03 → 00-108-05
00-108-04 ──────────────────────→ 00-108-05
00-108-06 (независим)
00-108-07 (независим)
```

Параллельный запуск: 00-108-01, 00-108-04, 00-108-06, 00-108-07.
