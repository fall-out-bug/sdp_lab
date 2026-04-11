# Agent Guide: Discovery

> Ты читаешь это, потому что тебе поручена Discovery-задача. Твоя цель: превратить идею в validated spec и scope decision — не начинать реализацию.

---

## Что ты производишь

| Артефакт | Путь | Описание |
|----------|------|---------|
| Frame | `docs/discovery/<slug>/frame.md` | JTBD, appetite, scope |
| Hypothesis | `docs/discovery/<slug>/hypothesis.md` | Assumptions с RAT ranking |
| Scan | `docs/discovery/<slug>/scan.md` | Landscape (settled + flagged) |
| Validation | `docs/discovery/<slug>/validation.md` | Evidence per claim + overall verdict |
| Experiment | `docs/discovery/<slug>/experiment.md` | (если нужно) Cheapest experiment brief |
| Feature card | Beads issue type:feature | Создаётся при GO |
| Workstream | `docs/workstreams/backlog/` | Создаётся при GO |

---

## Инструменты

```bash
# Запустить полный discovery pipeline
sdp discover "сырая идея"

# Brownfield: если это существующий кодобаза
sdp architect analyze

# Стратегический аудит высокого уровня
sdp-strataudit run

# LLM council (читай skill перед вызовом)
# → skills/llm-council.md
```

---

## Протокол

### Шаг 1 — Frame

**Цель:** Превратить сырую идею в чёткую постановку задачи.

1. Запусти `sdp discover "<идея>"` — pipeline начнёт с Frame
2. Проверь вывод: `problem_statement`, `jobs`, `appetite`, `scope`
3. Если pipeline запросит уточнение (Checkpoint A) → ответь на конкретный вопрос, не пиши роман
4. Frame считается завершённым когда: проблема сформулирована одним предложением, appetite задан (small/medium/large)

**Стоп-сигналы:**
- Идея слишком размытая → вернись к пользователю с одним конкретным вопросом
- Appetite = large → предложи разбить на несколько independent идей

---

### Шаг 2 — Hypothesize

**Цель:** Определить самые рискованные допущения.

Pipeline автоматически переходит к Phase 2. Проверь:

- Топ-1 по RAT rank — это то, что надо проверить первым
- `rat_score = risk × uncertainty` — высокий score = проверяй в первую очередь
- Минимум 3 assumption, максимум 7

Если assumptions выглядят очевидными (низкий rat_score у всех) → идея хорошо изучена, можно ускориться через Scan.

---

### Шаг 3 — Scan

**Цель:** Понять landscape — что уже существует, что это означает.

Pipeline запустит параллельный скан. Checkpoint C выдаст две секции:

**Section A (settled):** Принимай как есть. Не тратить время на re-research.

**Section B (flagged):** Для каждого flagged item реши:
- `[D]` Deep-dive — запусти дополнительное исследование
- `[P]` Provisional — принять с оговоркой в validation.md
- `[I]` Ignore — документируй причину игнора

Если `whitespace` пустой (нет незанятой ниши) → серьёзный сигнал для PIVOT.

---

### Шаг 4 — Validate

**Цель:** Проверить ключевые claims. Вынести вердикт.

Pipeline проверит 3-5 claims из hypothesis. Каждый claim получит:
- `supported` — evidence есть, proceed
- `contradicted` — доказательства против → обязательно отметить
- `insufficient_data` → переход к Phase 4b (Experiment)

**Overall verdict:**
- **GO** → переходи к Шагу 6
- **PIVOT** → возвращайся к Шагу 2 с новыми вводными, кратко объясни что изменилось
- **KILL** → документируй причину, закрой Beads issue, сохрани артефакты как ADR

---

### Шаг 5 — Experiment (если нужно)

Запускается только когда Phase 4a вернула `insufficient_data` для критических claims.

Pipeline предложит формат: smoke_test / landing_page / customer_interview / wizard_of_oz.

Минимальный experiment brief: objective, hypothesis, success_metric, time_box_days.

После experiment → вернись к Validate с новыми данными.

---

### Шаг 6 — Передача в Delivery (только при GO)

**Не начинай Delivery до явного GO от Decision Owner.**

При GO:

```bash
# Создать feature card в Beads
bd create --title="<название фичи>" \
  --description="NormalizedIntent из frame.md" \
  --type=feature \
  --priority=2

# Создать workstream файл
# → docs/workstreams/backlog/00-FFF-01.md
# → скопировать acceptance criteria из validation.md
```

Передать Delivery агенту:
1. Beads feature ID
2. Workstream файл путь
3. (Опционально) путь к `docs/discovery/<slug>/` для контекста

---

## Правила

| Правило | Обоснование |
|---------|------------|
| Не начинай Delivery до GO | PIVOT дешевле чем рефактор реализованной фичи |
| Minority reports из council → в validation.md | Несогласная модель часто права в edge case |
| Acknowledge contradicted claims | Игнор contradicted evidence → технический долг |
| Один вопрос при уточнении | Множественные вопросы не получают ответа |
| Appetite = large → разбей | Large scope → Discovery сама по себе слишком большая |

---

## Когда вызывать llm-council

Читай `skills/llm-council.md` перед вызовом.

Вызывать когда:
- Архитектурное решение с non-obvious трейдоффами
- validation.md содержит contradicted claim для core assumption
- Два или больше approach выглядят примерно одинаково хорошо
- Риск необратимого решения

Minority reports из council **обязательно** включать в validation.md — не выбрасывать.

---

## Структура `validation.md`

```markdown
# Validation: <slug>

**Date:** YYYY-MM-DD
**Overall Verdict:** GO | PIVOT | KILL

## Claims

### Claim 1: <формулировка>
**Verdict:** supported / contradicted / insufficient_data
**Evidence:** [источник, вывод]

### Claim 2: ...

## Council Input (если был)
**Question:** ...
**Consensus:** ...
**Minority:** ... [обязательно если было несогласие]

## Scope Decision
**In scope:**
**Out of scope:**
**Appetite:** small | medium | large
```

---

*Полная документация фазы: [docs/phases/DISCOVERY.md](../phases/DISCOVERY.md)*
