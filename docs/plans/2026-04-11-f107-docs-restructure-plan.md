# F107: Docs Restructure Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Создать полноценную документацию SDP — от Vision до агентских гайдов — и архивировать ~400 устаревших файлов.

**Architecture:** Структура mirror-ит фазы SDP: VISION → ARCHITECTURE → phases/ → guides/ → reference/. Один вход на аудиторию. Всё старое в docs/archive/.

**Design doc:** `docs/plans/2026-04-11-docs-restructure-design.md`
**Beads:** sdplab-sdi

---

## P0: Критические документы (создать первыми)

### Task 1: VISION.md

**Files:**
- Create: `VISION.md` (в корне sdp_lab)

**Step 1: Создать файл**

Содержание должно отвечать на один вопрос: *что такое SDP и зачем он нужен*. Не технические детали. Не roadmap. Одна страница.

Структура:
```markdown
# SDP — Software Development Platform

## Что это

[2-3 предложения: SDP = AI PDLC+SDLC. От идеи до задеплоенной фичи через структурированные AI-фазы.]

## Обещание

[Пользовательская история: "Я закидываю идею → Discovery агенты исследуют и шейпят → Delivery агенты реализуют через чёткие фазы и gates → фича задеплоена с доказательствами."]

## Две фазы

**Discovery** — [что это, когда нужно, что производит]
**Delivery** — [что это, как работает, что гарантирует]

## Режимы

- Local Mode: [когда, для кого]
- Operator Mode: [когда, для кого]

## Что SDP не является

[Явно исключить: не просто CI/CD, не просто governance, не просто agentloop wrapper]
```

**Step 2: Проверка**
- [ ] Читается за 2 минуты
- [ ] Не содержит технических деталей компонентов
- [ ] Явно называет Discovery и Delivery как первоклассные фазы
- [ ] Описывает пользовательский путь (идея → деплой)
- [ ] Не противоречит ARCHITECTURE.md (который ещё не написан — держи в уме)

**Step 3: Commit**
```bash
git add VISION.md
git commit -m "docs(F107): add VISION.md — AI PDLC+SDLC north star"
```

---

### Task 2: docs/phases/DISCOVERY.md

**Files:**
- Create: `docs/phases/DISCOVERY.md`
- Read first: `docs/plans/2026-04-08-discovery-pipeline-design.md`, `internal/discovery/` package headers

**Step 1: Создать директорию**
```bash
mkdir -p docs/phases
```

**Step 2: Создать файл**

Содержание: Discovery как первоклассная фаза SDP. Основано на реальном коде `internal/discovery`.

Структура:
```markdown
# Discovery Phase

## Что это
[Discovery = исследование идеи до начала реализации. Производит: spec, scope decision, risk assessment.]

## Когда запускается
[Триггеры: новая идея, brownfield audit, стратегический вопрос]

## Шаги
1. Frame — [что делает internal/discovery step frame]
2. Hypothesize — [...]
3. Scan — [...]
4. Validate — [...]
(взять из реального кода internal/discovery)

## Агенты и инструменты
- `sdp discover "идея"` — запуск pipeline
- `sdp architect analyze` — анализ архитектуры существующего кода
- `sdp-strataudit run` — стратегический аудит
- `llm-council` скил — deliberation для ключевых решений

## Выходные артефакты
[Что попадает в docs/discovery/ — реальные файлы из кода]

## Связь с Delivery
[Когда Discovery завершена → что передаётся в Delivery]
```

**Step 3: Проверка**
- [ ] Основано на реальном коде, не aspirational
- [ ] Называет конкретные команды (`sdp discover`, `sdp architect`)
- [ ] Объясняет роль council в Discovery
- [ ] Описывает конкретные артефакты на выходе

**Step 4: Commit**
```bash
git add docs/phases/DISCOVERY.md
git commit -m "docs(F107): add phases/DISCOVERY.md — first-class discovery phase"
```

---

### Task 3: docs/phases/DELIVERY.md

**Files:**
- Create: `docs/phases/DELIVERY.md`
- Read first: `internal/agentloop/router.go` (фазы), `docs/CANONICAL_SDP_PIPELINE.md`

**Step 1: Создать файл**

Структура:
```markdown
# Delivery Phase

## Что это
[Delivery = structured execution от spec до задеплоенной фичи через agentloop FSM.]

## Фазы agentloop
| Фаза | Что происходит | Gate на выходе |
|------|---------------|----------------|
| Discover | [...] | [...] |
| Plan | [...] | [...] |
| Build | [...] | [...] |
| Review | [...] | [...] |
| Eval | [...] | [...] |

## Gates
[Что такое gate, какие бывают, как работает GateEngine]

## Evidence
[Что такое evidence в контексте Delivery, как EvidenceAccumulator собирает из tool outputs]

## Агенты и инструменты
- `sdp-harness new/run` — запуск сессии
- `sdp-orchestrate` — feature-level orchestration
- `sdp-guard` — scope enforcement
- `sdp-ci-loop` — CI feedback loop

## Связь с Discovery
[Что принимает на вход, формат контракта]
```

**Step 2: Проверка**
- [ ] Описывает реальные фазы agentloop (не aspirational phases из pipeline doc)
- [ ] Разделяет что работает сейчас vs. что planned
- [ ] Содержит конкретные команды

**Step 3: Commit**
```bash
git add docs/phases/DELIVERY.md
git commit -m "docs(F107): add phases/DELIVERY.md — agentloop-based delivery"
```

---

### Task 4: ARCHITECTURE.md

**Files:**
- Create: `docs/ARCHITECTURE.md`
- Read first: результаты code audit (все cmd/ и internal/ из этой сессии)

**Step 1: Создать файл**

Ключевые секции:
```markdown
# SDP Architecture

## Обзор (diagram)
[ASCII или Mermaid: User → Discovery Layer → Delivery Layer → Deploy]

## Discovery Layer
| Компонент | Что делает | Статус |
|-----------|-----------|--------|
| internal/discovery | 8-step LLM pipeline | Production |
| internal/architect | C4 + code analysis | Production |
| internal/strataudit | Strategic LLM audit | Production |
| skills/llm-council | Multi-model deliberation | Production |

## Delivery Layer
| Компонент | Что делает | Статус |
|-----------|-----------|--------|
| internal/agentloop | Phase FSM + sessions | Production (needs LiveGateway — F106) |
| internal/executor | ExecutorBridge → OmO | Production |
| internal/orchestrate | Feature-level phases | Production |
| internal/gate | Gate filesystem | Production |
| internal/evidence | in-toto attestation | Production |
| internal/deploy | Docker compose wrapper | Production |

## Infrastructure Layer
[modelgateway, beads, control, a2a, guard, ci-loop]

## Интеграции
[OpenRouter, Beads/Dolt, GitHub, harnesses]

## Что работает сейчас vs. Что planned
[Явное разделение — не вводить в заблуждение]
```

**Step 2: Проверка**
- [ ] Каждый компонент имеет статус (Production/Beta/Planned)
- [ ] Нет aspirational компонентов без пометки
- [ ] Понятно как Discovery → Delivery → Deploy связаны
- [ ] agentloop правильно описан (работает, но нужен LiveGateway)

**Step 3: Commit**
```bash
git add docs/ARCHITECTURE.md
git commit -m "docs(F107): add ARCHITECTURE.md — honest component map"
```

---

### Task 5: docs/TERMS.md

**Files:**
- Create: `docs/TERMS.md`
- Read first: `docs/reference/GLOSSARY.md` (основа, датирован 2026-01-29)

**Step 1: Взять GLOSSARY.md за основу, обновить**

Добавить отсутствующие термины (из кода 2026-04):
- **agentloop** — Phase FSM execution kernel (internal/agentloop)
- **strataudit** — Strategic audit pipeline (internal/strataudit)
- **ai-architect** — Architecture analysis tool (internal/architect)
- **LiveGateway** — Production ModelGateway impl (F106, WS-01)
- **council** — Multi-model LLM deliberation (skills/llm-council.md)
- **mini-harness** — sdp-harness CLI (agentloop-based)
- **ServeBridge** — internal/executor/bridge_serve.go
- **EvidenceAccumulator** — tool output → evidence mapping
- **GateEngine** — circuit breaker между фазами
- **PhaseRouter** — фаза → модель + tools + prompt
- **Discovery** — первая фаза SDP (research → spec)
- **Delivery** — вторая фаза SDP (implementation → deploy)
- **PDLC** — Product Development Life Cycle
- **SDLC** — Software Development Life Cycle

**Step 2: Проверка**
- [ ] Все термины из reference/GLOSSARY.md перенесены
- [ ] Новые термины добавлены
- [ ] Каждый термин: название, определение, где используется
- [ ] Нет конфликтующих определений

**Step 3: Commit**
```bash
git add docs/TERMS.md
git commit -m "docs(F107): add TERMS.md — unified glossary updated to 2026-04"
```

---

### Task 6: docs/reference/components.md

**Files:**
- Create: `docs/reference/components.md`
- Read first: результаты code audit из этой сессии (все cmd/ + internal/)

**Step 1: Создать честный каталог всех компонентов**

```markdown
# SDP Components

> Статусы: 🟢 Production | 🟡 Beta (работает, но MVP) | 🔴 Planned (нет реализации)

## CLI Binaries (cmd/)

| Бинарь | Назначение | Фаза | Статус |
|--------|-----------|------|--------|
| sdp | Главный CLI | All | 🟢 |
| sdp-harness | agentloop сессии | Delivery | 🟡 (нет LiveGateway) |
| sdp-orchestrate | Feature orchestration | Delivery | 🟢 |
| ... | | | |

## Internal Packages (internal/)

| Пакет | Назначение | Фаза | Статус |
|-------|-----------|------|--------|
| agentloop | Phase FSM | Delivery | 🟡 (нет LiveGateway) |
| discovery | 8-step pipeline | Discovery | 🟢 |
| architect | C4 analysis | Discovery | 🟢 |
| strataudit | Strategic audit | Discovery | 🟢 |
| modelgateway | LLM adapters | Infra | 🟡 (0 production callers) |
| authz | RBAC | Infra | 🔴 (0 callers) |
| planner | Task graph | Delivery | 🔴 (0 callers) |
| ... | | | |
```

**Step 2: Проверка**
- [ ] Каждый cmd/ бинарь в таблице
- [ ] Каждый internal/ пакет в таблице
- [ ] Статусы честные (🔴 для пакетов с 0 callers)
- [ ] agentloop помечен 🟡 с объяснением

**Step 3: Commit**
```bash
git add docs/reference/components.md
git commit -m "docs(F107): add reference/components.md — honest component status"
```

---

## P0: Обновить входные точки

### Task 7: Обновить AGENTS.md

**Files:**
- Modify: `AGENTS.md`

**Step 1: Добавить преамбулу в начало файла** (после заголовка, до "Start Here")

```markdown
## Что такое SDP

SDP — AI-управляемая платформа для полного цикла разработки (PDLC + SDLC).
Пользователь подаёт идею → Discovery агенты исследуют и шейпят → Delivery агенты
реализуют через структурированные фазы и gates → фича задеплоена с доказательствами.

Две фазы:
- **Discovery**: `sdp discover` / `sdp architect` / `llm-council` → spec + scope decision
- **Delivery**: `agentloop` FSM (discover→plan→build→review→eval) → PR + evidence

Полный vision: [VISION.md](../VISION.md)
Архитектура: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
```

**Step 2: Добавить секцию про agentloop** (после "SDP Tools")

```markdown
## Execution Kernel: agentloop

`internal/agentloop` — FSM для Delivery фазы. Phases: Discover → Plan → Build → Review → Eval.
Запускается через `sdp-harness new/run`. Gates принудительны — FSM не переходит без прохождения gate.
Production gateway (F106): подключается через `agentloop.ModelGateway` → OpenRouter.

Reference: [docs/reference/agentloop.md](docs/reference/agentloop.md)
```

**Step 3: Добавить в Cold Start секцию**

Добавить в список вопросов:
```markdown
5. Это Discovery-задача (исследование, council, spec) или Delivery-задача (реализация)?
```

Добавить в Minimum first pass:
```markdown
6. прочитай [VISION.md](../VISION.md) чтобы понять контекст системы
```

**Step 4: Добавить упоминание llm-council**

В секцию "SDP Tools" добавить:
```markdown
### llm-council skill
`skills/llm-council.md` — multi-model deliberation для ключевых решений в Discovery.
Вызывать когда: архитектурное решение, риск-анализ, валидация spec.
```

**Step 5: Проверка**
- [ ] Первые 10 строк AGENTS.md дают понять что такое SDP
- [ ] Discovery и Delivery упомянуты явно
- [ ] llm-council в каталоге инструментов
- [ ] agentloop объяснён

**Step 6: Commit**
```bash
git add AGENTS.md
git commit -m "docs(F107): update AGENTS.md — add SDP overview, Discovery/Delivery, agentloop"
```

---

### Task 8: Обновить MANIFESTO.md

**Files:**
- Modify: `docs/MANIFESTO.md`

**Step 1: Найти и заменить позиционирование**

Найти: "evidence layer" / "SDP is not an orchestrator" (старый фрейминг)
Заменить на: AI PDLC+SDLC фрейминг

Найти первый абзац описания SDP, обновить до:
```markdown
SDP — это AI-управляемая платформа полного цикла разработки.
От идеи до задеплоенной фичи через структурированные Discovery и Delivery фазы с AI-агентами.
```

**Step 2: Проверка**
- [ ] Нет "evidence layer" как primary positioning
- [ ] Discovery упомянута как первая фаза
- [ ] Не противоречит VISION.md

**Step 3: Commit**
```bash
git add docs/MANIFESTO.md
git commit -m "docs(F107): update MANIFESTO.md — AI PDLC+SDLC positioning"
```

---

## P1: Guides и Happy Paths

### Task 9: docs/guides/agent-discovery.md

**Files:**
- Create: `docs/guides/agent-discovery.md`
- Create dir: `docs/guides/`

**Step 1: Создать полный протокол для Discovery агента**

```markdown
# Agent Guide: Discovery

## Когда ты это читаешь
Ты — агент ведущий Discovery для пользователя. Твоя задача: превратить идею в чёткий spec и scope decision.

## Что ты производишь
- `docs/discovery/<idea>/frame.md` — постановка задачи
- `docs/discovery/<idea>/hypothesis.md` — гипотезы
- `docs/discovery/<idea>/scan.md` — исследование
- `docs/discovery/<idea>/validation.md` — выводы
- Рекомендация: GO / NO-GO / PIVOT

## Инструменты
- `sdp discover "<идея>"` — запустить discovery pipeline
- `sdp architect analyze` — если brownfield (существующий код)
- `skills/llm-council.md` — для ключевых архитектурных решений

## Протокол
1. Frame: [шаги]
2. Hypothesize: [шаги]
3. Scan: [шаги]
4. Validate: [шаги]
5. Council (если нужно): [когда вызывать, как]
6. Передать в Delivery: [что и как]

## Правила
- Не начинай Delivery до явного GO от Decision Owner
- Minority reports из council обязательно включать в validation.md
- ...
```

**Step 2: Commit**
```bash
git add docs/guides/agent-discovery.md
git commit -m "docs(F107): add guides/agent-discovery.md"
```

---

### Task 10: docs/guides/agent-delivery.md

**Files:**
- Create: `docs/guides/agent-delivery.md`

**Step 1: Создать полный протокол для Delivery агента**

```markdown
# Agent Guide: Delivery

## Когда ты это читаешь
Ты — агент реализующий фичу. Discovery завершена, у тебя есть spec и workstream.

## Вход
- Beads issue с acceptance criteria
- Workstream файл в docs/workstreams/backlog/
- (Опционально) discovery артефакты

## Фазы agentloop
| Фаза | Твои действия | Gate |
|------|--------------|------|
| Discover | Читай spec, изучи codebase | — |
| Plan | Составь план изменений | plan-review |
| Build | Реализуй | — |
| Review | Self-review + тесты | review-pass |
| Eval | QA + evidence | qa-pass |

## Инструменты
- `sdp-harness run` — запуск фазы
- `sdp-guard` — проверка scope
- `sdp-ci-loop` — CI feedback

## Правила
- Не выходи за scope workstream файла
- Evidence только из реальных tool outputs (не self-report)
- Gate fail = стоп, не обход
- ...
```

**Step 2: Commit**
```bash
git add docs/guides/agent-delivery.md
git commit -m "docs(F107): add guides/agent-delivery.md"
```

---

### Task 11: docs/happy-paths/new-feature.md

**Files:**
- Create dir: `docs/happy-paths/`
- Create: `docs/happy-paths/new-feature.md`
- Migrate from: `docs/reference/canonical-happy-path.md` (Operator Mode секция)

**Step 1: Создать файл для сценария "новая фича"**

```markdown
# Happy Path: New Feature

## Ситуация
У тебя есть идея фичи для существующего проекта. SDP уже установлен.

## Полный путь

### 1. Discovery (10-30 мин)
```bash
sdp discover "идея фичи"
```
→ Артефакты в docs/discovery/
→ Если сложное архитектурное решение: запустить llm-council

### 2. Review Discovery (человек)
Прочитать docs/discovery/<idea>/validation.md
Решение: GO / NO-GO / PIVOT

### 3. Shaping (создать workstream)
```bash
# Создать feature card в Beads
bd create --title="Feature X" ...
# Создать workstream файл
```

### 4. Delivery
```bash
bd ready          # найти готовую задачу
sdp-harness new --session=<id>
sdp-harness run --session=<id> --prompt="..."
```

### 5. Review и деплой
```bash
sdp-ci-loop
sdp deploy staging
# approve → sdp deploy prod
```

## Варианты ответвлений
- Discovery → NO-GO: [что делать]
- Gate fail в Build: [что делать]
- CI не проходит: [что делать]
```

**Step 2: Commit**
```bash
git add docs/happy-paths/new-feature.md
git commit -m "docs(F107): add happy-paths/new-feature.md"
```

---

### Task 12: Остальные happy paths

По той же схеме что Task 11:

- `docs/happy-paths/greenfield.md` — новый проект с нуля
- `docs/happy-paths/brownfield.md` — встройка в legacy (`sdp architect analyze` первым)
- `docs/happy-paths/cold-start.md` — первый запуск SDP в проекте

**Commit каждый файл отдельно.**

---

## P2: Архивирование

### Task 13: Создать структуру archive и переместить файлы

**Files:**
- Create dirs: `docs/archive/plans/`, `docs/archive/k8s/`, `docs/archive/next-steps/`, `docs/archive/council-rounds/`

**Step 1: Создать archive/README.md**
```markdown
# Archive

Исторические документы. Не удалены — сохранены для понимания решений.
Не редактировать. Ссылаться только если нужна историческая справка.

| Директория | Содержимое |
|------------|-----------|
| plans/ | Рабочие планы до 2026-04-01 |
| k8s/ | K8s эпоха (архивирована в ADR-002) |
| next-steps/ | Выполненные *_NEXT_STEP.md |
| council-rounds/ | Протоколы LLM council сессий |
```

**Step 2: Переместить K8s документы**
```bash
mkdir -p docs/archive/k8s
mv docs/K8S_*.md docs/archive/k8s/
mv docs/KUBEOPENCODE_*.md docs/archive/k8s/
mv docs/SECRETS.md docs/archive/k8s/
mv docs/prd/PRD.md docs/archive/k8s/
```

**Step 3: Переместить NEXT_STEP файлы**
```bash
mkdir -p docs/archive/next-steps
mv docs/*_NEXT_STEP.md docs/archive/next-steps/
```

**Step 4: Переместить устаревшие plans/**
```bash
mkdir -p docs/archive/plans
# Переместить plans/ до 2026-04-01 (кроме активных F106/F107)
# Список активных (НЕ трогать):
# docs/plans/2026-04-11-agentloop-integration-spec.md
# docs/plans/2026-04-11-f107-docs-restructure-plan.md
# docs/plans/2026-04-11-docs-restructure-design.md
# docs/plans/2026-04-11-council-sdp-full.md
# docs/plans/2026-04-11-ws*.md
# docs/plans/2026-04-10-ai-architect-design.md
# docs/plans/2026-04-08-discovery-pipeline-design.md
# docs/plans/2026-04-08-discovery-pipeline-impl.md
# docs/plans/2026-03-31-*.md (platform reset — KEEP)
```

**Step 5: Переместить council rounds**
```bash
mkdir -p docs/archive/council-rounds
mv docs/plans/2026-04-11-council-sdp-crisis*.md docs/archive/council-rounds/
# Оставить только: council-sdp-full.md (финальный отчёт)
```

**Step 6: Commit**
```bash
git add docs/archive/
git add -u docs/  # track moves
git commit -m "docs(F107): archive K8s docs, old plans, NEXT_STEP files, council rounds"
```

---

### Task 14: Обновить sdp/ публичные docs

**Files:**
- Modify: `sdp/PRODUCT_VISION.md` (если submodule доступен)
- Modify: `sdp/QUICKSTART.md`
- Modify: `sdp/CLAUDE.md`
- Create: `sdp/VISION.md` (публичная выжимка из VISION.md)

**Step 1: Проверить доступность submodule**
```bash
ls sdp/ || ls ../sdp/
```

**Step 2: Обновить PRODUCT_VISION.md**
Убрать "evidence layer" как primary positioning.
Добавить: AI PDLC+SDLC, Discovery + Delivery фазы.

**Step 3: Обновить QUICKSTART.md**
Добавить явный раздел "Two Phases":
```markdown
## Two Phases

**Discovery** (optional but recommended for complex features):
```bash
sdp discover "your idea"
```

**Delivery** (always):
```bash
@feature "implement X"  # or @oneshot for simple tasks
```
```

**Step 4: Commit в sdp submodule**
```bash
cd sdp  # или cd ../sdp
git add PRODUCT_VISION.md QUICKSTART.md CLAUDE.md VISION.md
git commit -m "docs: update to AI PDLC+SDLC positioning"
git push
cd ..
git add sdp
git commit -m "docs(F107): update sdp submodule — AI PDLC+SDLC"
```

---

## Финальная проверка

### Task 15: Проверка связности

**Step 1: Проверить все ссылки из AGENTS.md**
```bash
# Каждая ссылка в AGENTS.md должна существовать
grep -o '\[.*\](.*\.md)' AGENTS.md | grep -o '(.*md)' | tr -d '()' | while read f; do
  [ -f "$f" ] || echo "BROKEN: $f"
done
```

**Step 2: Проверить структуру**
- [ ] `VISION.md` существует в корне
- [ ] `docs/ARCHITECTURE.md` существует
- [ ] `docs/TERMS.md` существует
- [ ] `docs/phases/DISCOVERY.md` существует
- [ ] `docs/phases/DELIVERY.md` существует
- [ ] `docs/reference/components.md` существует
- [ ] `docs/guides/agent-discovery.md` существует
- [ ] `docs/guides/agent-delivery.md` существует
- [ ] `docs/happy-paths/new-feature.md` существует
- [ ] `docs/archive/` существует с README

**Step 3: Финальный коммит и push**
```bash
git push origin feat/agentloop-impl
```

---

## Порядок выполнения

```
P0 (сначала):
  Task 1  → VISION.md
  Task 2  → phases/DISCOVERY.md
  Task 3  → phases/DELIVERY.md
  Task 4  → ARCHITECTURE.md
  Task 5  → TERMS.md
  Task 6  → reference/components.md
  Task 7  → Update AGENTS.md
  Task 8  → Update MANIFESTO.md

P1 (потом):
  Task 9  → guides/agent-discovery.md
  Task 10 → guides/agent-delivery.md
  Task 11 → happy-paths/new-feature.md
  Task 12 → остальные happy-paths

P2 (можно параллельно с P1):
  Task 13 → Archive
  Task 14 → Update sdp/
  Task 15 → Final check
```
