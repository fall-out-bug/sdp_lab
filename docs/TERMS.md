# SDP Terms & Definitions

**Версия:** 2.0  
**Дата:** 2026-04-11  
**Основа:** GLOSSARY.md v1.0 (2026-01-29) + актуальный code audit

> Единый глоссарий. При конфликте определений — этот файл является источником истины.

---

## Система в целом

### SDP (Software Development Platform)

Платформа для полного цикла AI-управляемой разработки. Автоматизирует Product Development Life Cycle (PDLC) + Software Development Life Cycle (SDLC) — от идеи до задеплоенной фичи через структурированные AI-фазы, агентов и gates.

**Не является:** просто CI/CD, просто governance, просто agentloop wrapper.

**Фреймирование:** AI PDLC + SDLC. Две первоклассные фазы: Discovery и Delivery.

**Устарело:** ~~Spec-Driven Protocol~~ (старый акроним, заменён).

---

### PDLC (Product Development Life Cycle)

Цикл от идеи до решения: исследование, шейпинг, scope decision. В SDP реализован через Discovery фазу.

---

### SDLC (Software Development Life Cycle)

Цикл от spec до deploy: план, реализация, тестирование, релиз. В SDP реализован через Delivery фазу.

---

## Discovery Phase

### Discovery

Первая фаза SDP. Превращает сырую идею в validated spec и scope decision через 4-phase LLM pipeline.

**Этапы:** Frame → Hypothesize → Scan → Validate (→ Experiment, если нужно).

**Команда:** `sdp discover "идея"`  
**Артефакты:** `docs/discovery/<slug>/` — frame.md, hypothesis.md, scan.md, validation.md  
**Реализация:** `internal/discovery`  
**Статус:** Production.

---

### Frame

Phase 1 Discovery. Структурирует идею через JTBD canvas: problem statement, jobs-to-be-done, appetite (small/medium/large), scope.

---

### Hypothesize

Phase 2 Discovery. Формулирует проверяемые гипотезы через Strategyzer Test Card. Ранжирует по RAT.

**RAT (Riskiest Assumption Test):** Метод приоритизации допущений по `risk × uncertainty`. RAT-rank 1 = самое рискованное допущение → проверять первым.

---

### Scan

Phase 3 Discovery. Параллельный скан 7 типов источников. Разделяет результаты на Section A (settled, coverage ≥ 0.4) и Section B (flagged, нужен human review).

**Depth Signal:** coverage_score per result — 7 эвристик для оценки достаточности исследования.

---

### Validate

Phase 4a Discovery. Desk research validation: проверяет 3–5 ключевых claims, выносит вердикт per claim (supported/contradicted/insufficient_data) и overall verdict.

---

### Discovery Verdict

Финальное решение после Phase 4:
- **GO** — создать feature card + передать в Delivery
- **PIVOT** — вернуться к Phase 2 с новыми данными
- **KILL** — закрыть, артефакты сохранить как ADR

---

### llm-council (Council)

Multi-model LLM deliberation. Несколько моделей независимо анализируют вопрос, синтезируется итоговое решение с учётом minority reports.

**Когда использовать:** архитектурные решения, риск-анализ, валидация spec.  
**Реализация:** `skills/llm-council.md` (skill, не бинарь).  
**Статус:** Production.

---

### ai-architect

Инструмент анализа существующего кода. C4-анализ, runtime coupling detection, адаптивная кластеризация.

**Команда:** `sdp architect analyze`  
**Реализация:** `internal/architect`  
**Статус:** Production.

---

### strataudit

Стратегический LLM-аудит высокого уровня. Для вопросов без конкретной идеи: "стоит ли нам строить X?"

**Команда:** `sdp-strataudit run`  
**Реализация:** `internal/strataudit`  
**Статус:** Production.

---

## Delivery Phase

### Delivery

Вторая фаза SDP. Реализует фичу от spec до задеплоенного кода через agentloop FSM с gate enforcement.

**Вход:** Feature card + workstream файл из Discovery (или напрямую от оператора).  
**Реализация:** `internal/agentloop` + `internal/executor`  
**Статус:** Production (требует LiveGateway — F106).

---

### agentloop

Ядро Delivery. Phase FSM: Discover → Plan → Build → Review → Eval. Каждая фаза завершается GateEngine evaluation.

**Компоненты:** PhaseRouter, GateEngine, EvidenceAccumulator, SQLite WAL sessions.  
**Реализация:** `internal/agentloop`  
**Статус:** Логика готова; нет production callers без LiveGateway (F106).

---

### Phase (agentloop фаза)

Один шаг FSM в Delivery. Каждая фаза имеет: входной prompt, набор tools, выходной gate.

| Фаза | Назначение | Gate |
|------|-----------|------|
| Discover | Изучить spec и codebase | — |
| Plan | Составить план изменений | contract-approve |
| Build | Реализовать | — |
| Review | Self-review + тесты | review-pass |
| Eval | QA + evidence | qa-pass |

---

### GateEngine

Circuit breaker между фазами agentloop. Оценивает gate за ≤5 секунд из EvidenceAccumulator. При fail: gate-pending (awaiting human) или terminal error.

**Не проходит gate** → FSM блокируется. Gate fail — не обходится, а фиксируется.

---

### EvidenceAccumulator

Сборщик evidence в agentloop. Извлекает доказательства только из real tool call outputs — не из текста агента (self-report не считается).

**Хранит:** tool call ID, tool name, output, timestamp. Передаёт в GateEngine.

---

### Gate

Контрольная точка между фазами. Типы:

| Gate | Тип | Enforcement |
|------|-----|-------------|
| `contract-approve` | human | Mandatory |
| `scope-review` | human | Mandatory при out-of-scope |
| `review-pass` | automated | Mandatory после F106 |
| `qa-pass` | automated | Mandatory после F106 |
| `ci` | automated | Mandatory |
| `staging-approve` | human | Mandatory |
| `prod-approve` | human | Mandatory |

---

### PhaseRouter

Компонент agentloop. Маппит фазу → модель + tools + system prompt.

---

### ServeBridge

`internal/executor/bridge_serve.go`. Точка входа для запуска agentloop сессии. `DispatchAndRun(ctx, projectID, cardID)` → создаёт Harness → запускает RunPhase.

---

### Harness

Обёртка над agentloop сессией. Управляет жизненным циклом: создание, RestoreHarness (при crash), Stop (terminal marker в SQLite).

**ErrHarnessTerminated:** sentinel variable `var ErrHarnessTerminated = errors.New("harness: session was terminated")`. Проверять через `errors.Is`, не через strings.Contains.

---

### LiveGateway

Production реализация `agentloop.ModelGateway`. Подключает agentloop к реальному OpenRouter.

**Статус:** Planned (F106, WS-01).  
**Без LiveGateway:** agentloop изолирован от реальных LLM calls.

---

### Session (agentloop)

Состояние одной delivery сессии. Хранится в `.sdp/sessions/<cardID>.db` (SQLite WAL).  
Содержит: TurnRecord, PhaseRecord, tool calls, evidence.

---

## Структуры данных

### Feature Card

Основная единица Delivery. Содержит: ID, NormalizedIntent, acceptance criteria, feature_id.

**Хранится:** `internal/control.Store`  
**Формат Beads:** `bd create --type=feature`

---

### Workstream (WS)

Атомарная задача в рамках Feature. Файл: `docs/workstreams/backlog/PP-FFF-SS.md`.

**Формат ID:** `PP-FFF-SS` — project-feature-sequence.  
**Размеры:** XS / S / M / L (L → разбить на несколько WS).

---

### ExecutorResultPacket

Результат `ServeBridge.DispatchAndRun`. Поля: `Status` (completed/gate-pending/failed), `CardID`, `Error`.

---

## Инфраструктура

### Beads

Git-backed issue tracker на Dolt. Хранит features, workstreams, dependencies, memories.

**CLI:** `bd create/update/close/ready/show/dep/remember`  
**Реализация:** `internal/beads`  
**Статус:** Production.

---

### ModelGateway

Абстракция над LLM providers. Интерфейс: `internal/modelgateway.ModelGateway`. Реализации: Anthropic, OpenAI, selfhosted adapters + PolicyRouter.

**Статус:** Библиотека готова, 0 production callers (до F106).

---

### Evidence (in-toto)

Attestation-based доказательная база. Связывает tool outputs с gate decisions. Формат: in-toto attestations.

**Команда:** `sdp-evidence`  
**Реализация:** `internal/evidence`

---

### mini-harness

Разговорное название для `sdp-harness` CLI — бинарь для запуска agentloop сессий.

---

## Концепции процесса

### Happy Path

Сценарий успешного прохождения всего SDP пайплайна для типичного случая. Документация: `docs/happy-paths/`.

| Сценарий | Файл |
|----------|------|
| New Feature | `happy-paths/new-feature.md` |
| Greenfield | `happy-paths/greenfield.md` |
| Brownfield | `happy-paths/brownfield.md` |
| Cold Start | `happy-paths/cold-start.md` |

---

### Critical Path

Нетривиальный сценарий с edge cases или enforcement механизмами. Документация: `docs/critical-paths/`.

---

### Decision Owner

Человек, который выносит финальное решение на GO/NO-GO gate (человеческий gate). Не агент.

---

### ADR (Architecture Decision Record)

Зафиксированное архитектурное решение с контекстом и последствиями. Хранится: `docs/decisions/`.

---

### Minority Report

Позиция модели в LLM council, которая расходится с консенсусом. Обязательно включается в validation.md — не игнорируется.

---

## Устаревшие термины

| Устарело | Заменено на |
|---------|------------|
| Spec-Driven Protocol | Software Development Platform |
| evidence layer | Discovery + Delivery phases |
| OmO (One man Orchestra) | agentloop + ServeBridge |
| Unified Orchestrator | sdp-orchestrate |
| Epic, Sprint, Task | Feature, Workstream |

---

*Предыдущая версия: `docs/reference/GLOSSARY.md` (2026-01-29) — архивирован.*
