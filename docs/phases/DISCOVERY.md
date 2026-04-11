# Discovery Phase

> **Статус:** Production — `internal/discovery` активно работает и вызывает OpenRouter.

---

## Что это

Discovery — первая фаза SDP. Превращает сырую идею в структурированный spec и scope decision через 4-фазный LLM-pipeline.

Без Discovery возможна прямая Delivery. Но Discovery устраняет самую дорогую ошибку в разработке: **реализовать не то**.

**Вход:** Идея в свободной форме (строка текста).  
**Выход:** Validated hypothesis + GO / PIVOT / KILL verdict + артефакты для Delivery.

---

## Когда запускается

| Триггер | Пример |
|---------|--------|
| Новая фича или продукт | "Хочу добавить систему нотификаций" |
| Brownfield audit | "Как нам встроить SDP в legacy монолит?" |
| Стратегический вопрос | "Стоит ли строить self-hosted вариант?" |
| Риск неопределённости | Нет ответа на "а это вообще нужно?" |

Для простых задач с известным контекстом — Discovery опциональна, переходи сразу в Delivery.

---

## Четыре фазы

### Phase 1 — FRAME

**Что делает:** Структурирует идею через JTBD (Jobs To Be Done) canvas.

**Результат (`FrameResult`):**
- `problem_statement` — чёткая формулировка проблемы
- `jobs` — кто, что делает и ради чего (`jobs_to_be_done`)
- `appetite` — small / medium / large
- `scope` — что входит, что нет
- `raw_idea` — оригинальная идея

**Checkpoint A:** Агент может запросить уточнение по `missing_info`, `ambiguous_requirement` или `approach_choice` перед переходом к Phase 2.

---

### Phase 2 — HYPOTHESIZE

**Что делает:** Формулирует проверяемые гипотезы через Strategyzer Test Card. Ранжирует по RAT (Riskiest Assumption Test).

**Результат (`HypothesisResult`):**
- `assumptions` — список допущений с `risk_level`, `uncertainty`, `rat_score`, `rat_rank`
- Топ-1 допущение = то, что надо проверить первым

**Checkpoint B:** Сформированные гипотезы представляются для review.

---

### Phase 3 — SCAN

**Что делает:** Параллельный скан 7 типов источников (web, OSS, academic, product, pricing, docs, news). Использует depth signal — coverage score per result.

**Результат (`ScanResult`):**
- `items` — список конкурентов/аналогов с `disposition` (gap/alternative/complement/irrelevant)
- `whitespace` — незанятая ниша
- `recommended_stack` — технологические рекомендации
- **Section A (settled):** coverage ≥ 0.4, уверенные данные
- **Section B (flagged):** требуют human review — `[D]eep-dive / [P]rovisional / [I]gnore`

**Checkpoint C (двухсекционный):** Человек решает что делать с каждым flagged item.

---

### Phase 4a — VALIDATE (Desk Research)

**Что делает:** Проверяет 3–5 ключевых claims из гипотез через desk research. Выносит вердикт per claim.

**Результат (`ValidationResult`):**
- `verdict` per claim: `supported` / `contradicted` / `insufficient_data`
- `overall_verdict`: **GO** / **PIVOT** / **KILL**

**Checkpoint D:** Финальный вердикт с обоснованием.

---

### Phase 4b — EXPERIMENT (условно)

**Когда:** Если Phase 4a вернула `insufficient_data` для критических claims.

**Что делает:** Выбирает самый дешёвый формат эксперимента из матрицы:

| Формат | Когда использовать |
|--------|--------------------|
| `smoke_test` | Быстрая техническая проверка |
| `landing_page` | Проверка demand без продукта |
| `customer_interview` | Качественное понимание |
| `wizard_of_oz` | Симуляция без автоматизации |

**Результат (`ExperimentBrief`):** `format`, `objective`, `hypothesis`, `success_metric`, `time_box_days`, `setup_steps`.

---

## Финальный вердикт

```
GO    → создаётся feature card + beads issue type:feature → передаётся в Delivery
PIVOT → возврат к Phase 2 с новыми входными данными
KILL  → issue закрывается, артефакты сохраняются как ADR
```

---

## Команды

```bash
# Запустить полный pipeline
sdp discover "идея"

# Запустить на brownfield codebase (добавляет анализ существующего кода)
sdp architect analyze

# Стратегический аудит (высокоуровневый вопрос без конкретной идеи)
sdp-strataudit run

# LLM council для ключевых архитектурных решений внутри Discovery
# (запускается агентом через skills/llm-council.md)
```

---

## Выходные артефакты

Все артефакты сохраняются в `docs/discovery/<slug>/`:

| Файл | Содержимое |
|------|-----------|
| `frame.md` | Problem statement, JTBD, appetite, scope |
| `hypothesis.md` | Assumptions с RAT ranking |
| `scan.md` | Landscape с Section A/B |
| `validation.md` | Evidence per claim + overall verdict |
| `experiment.md` | (если Phase 4b) Experiment brief |

---

## Агенты

| Агент | Роль |
|-------|------|
| Discovery agent | Ведёт pipeline Frame→Hypothesize→Scan→Validate |
| LLM Council | Deliberation для архитектурных решений (вызов через skill) |
| Architect agent | C4-анализ существующего кода (brownfield) |

Подробный протокол для агентов: [docs/guides/agent-discovery.md](../guides/agent-discovery.md)

---

## Связь с Delivery

Discovery завершается GO-вердиктом. На выходе:

1. **Feature card** в Beads (type:feature, normalized_intent из frame.md)
2. **Workstream файл** (`docs/workstreams/backlog/`) с acceptance criteria из validation.md
3. **Артефакты** в `docs/discovery/<slug>/` — доступны Delivery агенту для контекста

Delivery агент читает workstream файл и feature card. Discovery артефакты — опциональный контекст.

→ [Delivery Phase](DELIVERY.md)
