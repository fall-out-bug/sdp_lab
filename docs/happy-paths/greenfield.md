# Happy Path: Greenfield Project

> **Ситуация:** Новый проект с нуля. Ни кода, ни структуры, ни SDP. Как запустить?

> **Status:** operator/lab recipe, not the friendly first-run Toolkit path.
> If you are evaluating SDP from a fresh repo, start with
> [../START_HERE.md](../START_HERE.md) and [../QUICKSTART.md](../QUICKSTART.md).
> This document still uses `sdp-up` and `sdp-harness`, which are not the stable
> first-run Toolkit promise.

---

## Шаг 1: Инициализация проекта

```bash
# Создать репозиторий
mkdir my-project && cd my-project
git init

# Инициализировать SDP
sdp-up
```

`sdp-up` создаёт:
- `.sdp/` — директория сессий, evidence, gates
- `docs/workstreams/` — структура для workstreams
- `AGENTS.md` — базовые инструкции для агентов

---

## Шаг 2: Discovery для первой идеи

У тебя есть большая идея — что именно строить?

```bash
sdp discover "описание продукта/системы которую хочу построить"
```

Для greenfield особенно важно пройти Discovery полностью:
- **Frame**: что именно строим, для кого, какой appetite (не пытайся сделать всё)
- **Hypothesize**: ключевые риски нового проекта — технические и продуктовые
- **Scan**: что уже существует, не изобретаем ли велосипед
- **Validate**: проверка core assumptions

**Если сложная архитектура**: `sdp architect` после SCAN — он поможет выбрать стек.

---

## Шаг 3: Архитектурные решения

Для нового проекта architecture = первое что надо решить.

```bash
# Если есть похожий codebase для reference
sdp architect analyze --path=../similar-project

# LLM council для key architectural decisions
# Читай: skills/llm-council.md
```

Зафиксируй решения как ADR в `docs/decisions/`:

```markdown
# ADR-001: <технология/решение>

**Date:** YYYY-MM-DD
**Status:** Accepted
**Context:** Почему это решение нужно
**Decision:** Что выбрали
**Consequences:** Что это означает
```

---

## Шаг 4: Первый workstream — Foundation

Первый workstream для greenfield всегда Foundation:

```bash
bd create \
  --title="Foundation: project scaffold + CI" \
  --description="Базовая структура проекта: модуль, CI, linting, первый тест" \
  --type=feature \
  --priority=0
```

Workstream файл для Foundation:
```markdown
## Goal
Создать рабочий scaffold с CI — основа для всех последующих workstreams.

## Scope Files
- go.mod / package.json / etc.
- .github/workflows/ci.yml
- cmd/<binary>/main.go (skeleton)
- Makefile

## Acceptance Criteria
- [ ] `go build ./...` / `npm run build` проходит
- [ ] CI зелёный на пустом main
- [ ] `go test ./...` / `npm test` запускается (0 тестов = OK)
- [ ] README.md с "что это и как запустить"

## Out of Scope
- Любая бизнес-логика
- Любые внешние интеграции
```

---

## Шаг 5: Delivery Foundation

```bash
bd ready
bd update <foundation-id> --claim
git checkout -b feature/foundation

sdp-harness new --card-id=<id> --project=<project-name>
sdp-harness run --session=<id> --prompt="Создай project scaffold..."
```

Foundation должна пройти все gate быстро — это самый простой workstream.

---

## Шаг 6: Итеративное построение

После Foundation — Feature за Feature:

```
Feature 1: Core domain model
Feature 2: API layer
Feature 3: Persistence
Feature 4: CLI / UI
...
```

Каждый Feature → Discovery (если нетривиальный) → Workstreams → Delivery.

**Правило greenfield:** Не строй все фичи сразу. Appetite = small для первых трёх Features.

---

## Особенности Greenfield vs. New Feature

| Аспект | New Feature | Greenfield |
|--------|-------------|------------|
| Discovery | Опциональна | Обязательна для product vision |
| Первый WS | Feature logic | Foundation scaffold |
| ADRs | Редко | Много — архитектура решается сейчас |
| Appetite | Средний | Маленький — сначала работающий MVP |
| Council | При сложных решениях | Почти всегда для tech stack |

---

## Итог

```
Идея продукта
  → sdp discover (полный pipeline)
  → Архитектурные ADRs
  → Feature: Foundation (scaffold + CI)
  → Feature: Core Domain
  → Feature: [next...]
  → Working product
```
