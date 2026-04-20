---
name: spec-interrogate
description: Закалка спеки через сократический диалог — context-stripped агент итеративно задаёт вопросы автору до convergence.
version: 1.0.0
tags:
  - discovery
  - spec
  - quality-gate
  - socratic
compatibility:
  - claude-code
  - opencode
  - cursor
  - codex
---

# spec-interrogate

## Purpose

Pressure-test любой артефакт (спека, план, дизайн-doc) через сократический диалог.
Interrogator получает **только артефакт** — без истории разговора, без beads-контекста, без имплицитных допущений автора. Цель: найти gap'ы, которые автор не видит из-за proximity bias.

## Роли

**Author** — агент или человек, написавший артефакт. Владеет итерацией.

**Interrogator** — отдельный агент, запускаемый с нуля. Получает ТОЛЬКО файл артефакта. Не правит, не предлагает решений. Только задаёт вопросы.

Принцип: Author решает что менять. Interrogator обнаруживает что непонятно.

## Конфигурация

| Параметр | По умолчанию | Описание |
|----------|-------------|----------|
| `--questions` | 5 | Максимум вопросов за раунд |
| `--rounds` | 5 | Максимум раундов диалога |
| `--mode` | `socratic` | Режим (см. ниже) |

### Режимы (`--mode`)

**`socratic` (default):** Interrogator задаёт вопросы → Author отвечает правками → повтор. Итеративно, дорого, максимально точно.

**`cold-read`:** Один проход — Interrogator пересказывает понятое своими словами. Быстро, выявляет грубые ambiguities.

**`adversarial`:** Interrogator ищет дыры без вопросов — выдаёт список уязвимостей. Для security/risk review.

**`impl-test`:** Interrogator пытается составить implementation plan только по артефакту. Провал → спека неполная.

## Протокол (режим `socratic`)

### Шаг 1 — Подготовка

Author передаёт Interrogator:
- Файл артефакта (путь или содержимое)
- Параметры: `--questions N --rounds M`

Interrogator НЕ получает:
- Историю разговора
- beads issue / контекст фичи
- Объяснения автора
- PR / код

### Шаг 2 — Раунд

Interrogator читает артефакт и задаёт до N вопросов, ранжированных по impact:

**Taxonomy вопросов (по приоритету):**
1. **WHY** — почему именно это решение, какова мотивация
2. **Undefined terms** — термин используется, но не определён в артефакте
3. **Missing error path** — что происходит когда X не работает
4. **Scope ambiguity** — граница ответственности не ясна
5. **Unstated assumption** — автор предполагает, но не пишет

Interrogator НЕ задаёт вопросы про: стиль, форматирование, предпочтения без impact на смысл.

### Шаг 3 — Ответ Author

Author обновляет артефакт. Не отвечает словами — только правит документ.

### Шаг 4 — Convergence check

- **Converged:** Interrogator не нашёл новых вопросов → артефакт прошёл gate.
- **Not converged:** Есть вопросы, раундов не исчерпано → следующий раунд.
- **Rounds exhausted:** Достигнут лимит раундов, вопросы остаются → **возврат на доработку**.

## Исходы

| Исход | Условие | Следующий шаг |
|-------|---------|---------------|
| **PASS** | Interrogator: 0 вопросов | Артефакт готов к следующей фазе |
| **REWORK** | Раунды исчерпаны, вопросы остаются | Author дорабатывает артефакт, запускает снова |
| **ABORT** | Author явно останавливает | Беадс-тикет с фиксацией открытых вопросов |

При **REWORK**: Interrogator выдаёт финальный список незакрытых вопросов как `rework-report`. Author не начинает следующую фазу до нового прогона.

## SDP Phase Integration

Встраивается как **agent discipline** перед эмиссией Plan gate. Это не tooling-enforcement — агент обязан запустить `@spec-interrogate` и уважать REWORK вердикт до вызова `sdp phase plan`.

### Discovery → Plan (обязательно для non-trivial фич)

```
@spec-interrogate docs/discovery/<slug>/validation.md --feature-id <F>
# → .sdp/evidence/spec-interrogate.json

# Только после PASS:
sdp phase plan --feature-id <F> --strict --evidence-path .sdp/evidence/plan.json
```

При **REWORK** агент не вызывает `sdp phase plan`. Discovery возобновляется с `rework-report` как входными данными.

### Plan → Build (опционально, для сложных архитектурных планов)

```
@spec-interrogate docs/plans/<feature>-design.md --feature-id <F> --mode adversarial
```

### Evidence schema

Скилл записывает `.sdp/evidence/spec-interrogate.json`:

```json
{
  "interrogate_verdict": "PASS | REWORK | ABORT",
  "rounds_completed": 3,
  "open_questions_count": 0,
  "artifact_path": "docs/discovery/my-feature/validation.md",
  "mode": "socratic"
}
```

### Когда пропустить

- Тривиальные задачи (bugfix, однострочные изменения)
- Прямой Delivery без Discovery (контекст уже известен)
- Явный флаг `--skip-interrogate` с обоснованием в beads issue

## Invocation Contract

```
@spec-interrogate <artifact-path> [--questions N] [--rounds M] [--mode MODE] [--feature-id F]
```

Примеры:
```bash
# Дефолт: socratic, 5 вопросов, 5 раундов
@spec-interrogate docs/discovery/my-feature/validation.md --feature-id F042

# Быстрая проверка перед совещанием
@spec-interrogate docs/plans/arch-decision.md --mode cold-read

# Жёсткий прогон перед сложным PR
@spec-interrogate docs/plans/auth-redesign.md --rounds 3 --mode adversarial
```

## Артефакты

- **`.sdp/evidence/spec-interrogate.json`** — machine-readable результат для Plan gate
- **Round log** — вопросы и версии артефакта по раундам (в stdout)
- **Rework-report (при REWORK)** — нумерованный список незакрытых вопросов для следующего прогона

## Acceptance Boundaries

NOT for: реализацию (@build), code review (@review), исследование (@research).

Этот скилл работает только с **текстовыми артефактами** (md, txt, json-schema). Не применять к коду напрямую — для кода используй @review.
