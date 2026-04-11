# Agent Guide: Delivery

> Ты читаешь это, потому что тебе поручена Delivery-задача. Discovery завершена — у тебя есть spec и workstream. Твоя цель: реализовать фичу через agentloop фазы с gate enforcement.

---

## Вход

Перед началом убедись что у тебя есть:

| Артефакт | Где найти |
|----------|----------|
| Beads issue | `bd show <id>` |
| Workstream файл | `docs/workstreams/backlog/00-FFF-SS.md` |
| Feature card | `bd show <feature-id>` или `internal/control.Store` |
| (Опционально) Discovery артефакты | `docs/discovery/<slug>/` |

Прочитай workstream файл целиком до начала работы. Acceptance Criteria — твои единственные критерии "done".

---

## Фазы agentloop

| Фаза | Твои действия | Gate на выходе |
|------|--------------|----------------|
| **Discover** | Читай spec, workstream, изучи codebase. Нет реализации. | — |
| **Plan** | Составь конкретный план изменений (файлы, функции, тесты). | `contract-approve` (human) |
| **Build** | TDD: тест → реализация → зелёный. Маленькие коммиты. | — |
| **Review** | Self-review: соответствие spec, покрытие тестами, scope. | `review-pass` (automated) |
| **Eval** | QA: запусти полный test suite, зафиксируй evidence. | `qa-pass` (automated) |

---

## Команды

```bash
# Запуск новой сессии
sdp-harness new --card-id=<cardID> --project=<projectID>

# Запуск фазы
sdp-harness run --session=<id> --prompt="..."

# Проверка scope
sdp-guard check --file=<path>

# CI feedback loop
sdp-ci-loop

# Проверка evidence
sdp-evidence validate --evidence .sdp/evidence/<run>.json
```

---

## Протокол по фазам

### Discover

**Цель:** Понять задачу, не делать ничего.

1. Прочитай workstream файл — все секции, включая Out of Scope
2. Прочитай acceptance criteria — это закон, не рекомендация
3. Изучи codebase в area of change: `grep`, `find`, `read` — без редактирования
4. Если что-то неясно — задай один вопрос сейчас, не после Build
5. Запиши в памяти: "что я меняю", "что я точно не меняю"

**Стоп-сигнал:** Если workstream зависит от незакрытой задачи → `bd blocked`, остановись.

---

### Plan

**Цель:** Конкретный план изменений на уровне файлов и функций.

1. Составь список: `Create`, `Modify`, `Delete` файлов
2. Для каждого Modify: какие функции/типы затронуты
3. Опиши тестовую стратегию: unit / integration / e2e
4. Проверь scope через `sdp-guard check` для каждого файла в плане
5. Человеческий gate `contract-approve` — ждёт одобрения плана

**Out of scope = не включай в план.** Если discovery показывает, что нужно изменить "попутно" — создай отдельный Beads issue.

---

### Build

**Цель:** Реализовать по TDD. Маленькие шаги.

Цикл для каждого изменения:

```bash
# 1. Напиши failing test
# 2. Убедись что он падает (FAIL = правильно)
# 3. Напиши минимальную реализацию
# 4. Убедись что тест проходит
go test ./... -run TestSpecificTest

# 5. Коммит
git add <files>
git commit -m "feat: <описание>"
```

**Правила Build:**
- Не пиши код без failing теста
- Один коммит = одно логическое изменение
- Не редактируй файлы вне scope (sdp-guard заблокирует)
- Если сломал существующий тест — исправь сейчас, не потом

---

### Review

**Цель:** Self-review перед автоматическим gate.

Пройди чеклист:

```
[ ] Все acceptance criteria из workstream файла выполнены
[ ] Нет файлов вне declared scope
[ ] Нет пропущенных тестов (go test ./... green)
[ ] Нет TODO/FIXME без Beads issue
[ ] go vet ./... чист
[ ] Нет debug code или логов для dev только
[ ] Каждый новый публичный тип/функция задокументирован
```

Gate `review-pass` оценивается из tool evidence — не из твоего самоотчёта.

---

### Eval

**Цель:** QA + финальная фиксация evidence.

```bash
# Полный test suite
go test ./...

# Quality gates
./scripts/run_go_quality_gates.sh

# CI feedback (если нужен)
sdp-ci-loop

# Validate evidence
sdp-evidence validate --evidence .sdp/evidence/run-*.json
```

После `qa-pass`: пуш + PR.

---

## Gate Fails

| Gate | При fail | Действие |
|------|----------|---------|
| `contract-approve` | Человек не одобрил план | Уточни план, жди одобрения |
| `scope-review` | Out-of-scope обнаружен | Убери изменения вне scope или создай отдельный WS |
| `review-pass` | Тесты не прошли / AC не выполнены | Исправь, не обходи gate |
| `qa-pass` | QA failure | Найди root cause, fix, не skip |
| `ci` | CI fail | Исправь CI — не merge с красным CI |

**Gate fail = стоп, не обход.** Нет исключений. Если gate кажется неправильным → создай Beads issue для review gate logic, не обходи сейчас.

---

## Evidence

Evidence собирается автоматически из real tool call outputs через EvidenceAccumulator.

**Что считается evidence:** реальные tool outputs (go test output, linter output, sdp-guard output).  
**Что НЕ считается:** твой текстовый self-report ("я проверил и всё хорошо").

Не нужно специально "писать evidence" — выполняй инструменты, evidence накапливается.

---

## Правила

| Правило | Почему |
|---------|--------|
| Не выходи за scope workstream файла | Scope creep = gate fail + грязная история |
| Evidence только из real tool outputs | Self-report не верифицируем |
| Gate fail = стоп | Gates = доказательство качества |
| Один WS = один коммит-набор | История должна быть читаемой |
| `bd close` только при qa-pass | Незавершённое закрытие = потеря context |

---

## Типичные ошибки

**"Я почти закончил, пропущу review"** → Review gate обязателен. "Почти" = не done.

**"Этот тест мешает, закомментирую"** → Никогда. Найди root cause.

**"Попутно исправлю вот это"** → Создай отдельный Beads issue. Не сейчас.

**"Gate кажется слишком строгим"** → Создай issue для обсуждения. Не обходи.

---

*Полная документация фазы: [docs/phases/DELIVERY.md](../phases/DELIVERY.md)*  
*Компоненты: [docs/reference/components.md](../reference/components.md)*
