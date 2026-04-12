# Happy Path: Brownfield Integration

> **Ситуация:** Существующий проект (legacy или активный). Хочешь добавить фичу или встроить SDP в процесс. С чего начать?

---

## Шаг 1: Аудит существующего кода

Прежде чем писать что-то новое — понять что уже есть.

```bash
# Архитектурный анализ existing codebase
sdp architect analyze --path=.
```

`sdp architect analyze` производит:
- C4-диаграмму компонентов
- Выявление coupling и зависимостей
- Рекомендации по точкам интеграции

Артефакты сохраняются в `docs/discovery/architect-<date>/`.

---

## Шаг 2: Discovery с контекстом brownfield

```bash
# Discovery с явным контекстом существующей системы
sdp discover "что хочу добавить в существующую систему"
```

В Frame агент должен явно учесть:
- `scope`: что в existing system затрагивается, что нет
- `appetite`: brownfield всегда сложнее — appetite обычно medium/large
- `jobs`: кто сейчас использует систему, как это изменится

После Scan особое внимание на:
- Существующие паттерны в codebase (architect анализ)
- Coupling — что сломается при изменении
- Technical debt, который надо учесть

---

## Шаг 3: LLM Council для integration strategy

Brownfield = высокий риск breaking changes. Рекомендуется council:

```
# Читай: skills/llm-council.md
# Вопрос для council: "как встроить X в существующую систему Y
#   с минимальным риском регрессий?"
```

---

## Шаг 4: Инициализация SDP в существующем проекте

Если SDP не был инициализирован:

```bash
# В корне существующего проекта
sdp-up

# sdp-up НЕ затрагивает существующий код
# Создаёт только .sdp/ и docs/workstreams/
```

---

## Шаг 5: Workstream с явным brownfield scope

Для brownfield workstream особенно важен раздел Out of Scope:

```markdown
## Acceptance Criteria
- [ ] Новая функциональность работает
- [ ] Все существующие тесты PASS (регрессии нет)
- [ ] `go test ./...` полностью зелёный

## Out of Scope (КРИТИЧНО)
- Рефакторинг существующего кода вне точки интеграции
- Изменение интерфейсов которые используются другими компонентами
- Обновление зависимостей без явного одобрения

## Integration Points
- Файлы которые нужно изменить: [список]
- Файлы которые НЕЛЬЗЯ менять: [список]
```

---

## Шаг 6: Delivery с усиленным guard

```bash
bd ready && bd update <id> --claim

# Delivery
sdp-harness new --card-id=<id> --project=<name>
sdp-harness run --session=<id> --prompt="..."
```

Во время Build агент будет вызывать `sdp-guard check` для каждого файла.

**Если guard срабатывает:** агент должен остановиться — не обходить. Brownfield out-of-scope = реальный риск.

---

## Шаг 7: Регрессионное тестирование

После Build, перед Eval:

```bash
# Полный test suite включая существующие тесты
go test ./...

# Если есть integration tests
go test -tags integration ./...

# CI
sdp-ci-loop
```

Если существующий тест упал — это твоя ответственность исправить (не skip).

---

## Особенности Brownfield

| Риск | Митигация |
|------|-----------|
| Регрессии | Полный test suite обязателен, не только новые тесты |
| Scope creep | Out of Scope секция = закон, sdp-guard enforcement |
| Hidden coupling | sdp architect analyze перед началом |
| Breaking changes | Council для integration strategy |
| Legacy debt | Не "попутно рефакторить" — отдельный workstream |

---

## Типичные ошибки Brownfield

**"Пока тут, заодно исправлю X"** → Нет. Создай отдельный Beads issue.

**"Этот тест устарел, уберу"** → Нет. Найди почему он упал.

**"Не буду делать architect analyze, и так знаю систему"** → Делай. Architect находит coupling, который ты забыл.

---

## Итог

```
Существующая система
  → sdp architect analyze (понять что есть)
  → sdp discover (с контекстом brownfield)
  → Council (integration strategy)
  → Workstream с жёстким Out of Scope
  → Delivery (с sdp-guard enforcement)
  → Регрессионный test suite green
  → PR + merge
```
