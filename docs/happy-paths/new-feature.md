# Happy Path: New Feature

> **Ситуация:** Проект работает. SDP настроен. У тебя есть идея новой фичи. С чего начать?

---

## Полный путь

### 1. Discovery (10–30 мин, для нетривиальных фич)

```bash
sdp discover "краткое описание идеи"
```

Pipeline пройдёт: Frame → Hypothesize → Scan → Validate.

На каждом checkpoint отвечай конкретно. Если pipeline запрашивает уточнение — один ответ, не развёрнутое эссе.

**Вывод:** `docs/discovery/<slug>/` + overall verdict.

**Если идея простая и однозначная** — Discovery опциональна. Переходи сразу к шагу 3.

---

### 2. Review Discovery (человек, 5 мин)

Открой `docs/discovery/<slug>/validation.md`.

- **GO** → продолжай
- **PIVOT** → Discovery переходит к Phase 2 с новыми вводными
- **KILL** → закрой, артефакты в docs/discovery/ сохраняются как история

**Если сложное архитектурное решение:** запусти council перед GO.
Читай `skills/llm-council.md` → запусти → minority reports обязательно читай.

---

### 3. Shape (создать workstream, 10 мин)

```bash
# Создать feature card в Beads
bd create \
  --title="Feature: <название>" \
  --description="<normalized_intent из frame.md>" \
  --type=feature \
  --priority=2

# Запомни feature ID (sdplab-XXX)

# Создать workstream файл
# Шаблон: docs/workstreams/backlog/00-FFF-01.md
# Acceptance criteria → из validation.md
```

Workstream файл содержит:
- Goal (одно предложение)
- Scope Files (конкретные файлы, не "вся система")
- Acceptance Criteria (checkbox список, верифицируемые)
- Out of Scope (явный список)

---

### 4. Delivery — Claim и запуск

```bash
# Проверить готовность
bd ready
bd show <workstream-beads-id>

# Claim
bd update <id> --claim

# Новая ветка
git checkout main && git pull
git checkout -b feature/FXXX-short-name

# Запуск agentloop сессии
sdp-harness new --card-id=<cardID> --project=<projectID>
sdp-harness run --session=<id> --prompt="<prompt из workstream>"
```

Агент пройдёт: Discover → Plan → (contract-approve) → Build → Review → Eval.

---

### 5. Gates

| Gate | Кто | Действие |
|------|-----|---------|
| `contract-approve` | Ты | Прочитай план, одобри или попроси изменить |
| `review-pass` | Автоматически | Тесты должны быть зелёными |
| `qa-pass` | Автоматически | Full test suite + evidence |
| `ci` | GitHub Actions | Не merge с красным CI |

---

### 6. Review и деплой

```bash
# PR
git push -u origin HEAD
gh pr create --base main --title "FXXX: <название>"

# CI должен быть зелёным
# После code review → merge

# Staging
sdp deploy staging
# Проверь вручную → approve

# Production
sdp deploy prod  # только после staging-approve
```

---

## Варианты ответвлений

### Discovery → PIVOT

Validation говорит "this assumption is wrong". Что делать:

1. Прочитай что именно contradicted
2. Вернись к Frame с новой постановкой
3. `sdp discover` запустится с обновлёнными вводными
4. Не торопись к GO — PIVOT дешевле неправильной реализации

### Gate fail в Build

```
review-pass failed: 2 tests failing
```

1. Прочитай какие именно тесты упали
2. Исправь — не пропускай тест и не пиши `t.Skip`
3. Перезапусти `sdp-harness run`

### Scope creep

Агент начинает менять файлы вне declared scope:

1. `sdp-guard check` покажет нарушение
2. Откатить изменения вне scope
3. Если нужно расширить scope → новый workstream, не текущий

### Заблокирован зависимостью

```bash
bd blocked       # показать все заблокированные issues
bd show <id>     # найти что блокирует
```

Реши blocker или переключись на другой ready workstream.

---

## Итог

```
Идея
  → sdp discover (10-30 мин)
  → GO verdict
  → bd create + workstream файл (10 мин)
  → sdp-harness (Deliver фаза: Discover→Plan→Build→Review→Eval)
  → PR + CI green
  → Staging approve
  → Deploy prod
```

Весь путь от идеи до деплоя: 1–3 дня для medium фичи.
