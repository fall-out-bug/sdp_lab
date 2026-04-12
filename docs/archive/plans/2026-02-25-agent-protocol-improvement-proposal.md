# Архитектурное предложение: улучшение протокола, кодовой базы и взаимодействия с пользователем

**Дата:** 2026-02-25  
**Источники:** анализ логов `cursor_build_00_53_16.md`, `cursor_markdown_file_build_discussion.md`, `cursor_project_design_and_code_analysis.md`  
**Метод:** @think — breakdown → анализ по аспектам → сводка

---

## 1. Проблемы агента

### 1.1 Ошибки размещения и дублирование артефактов

| Проблема | Пример из логов | Рекомендация |
|----------|-----------------|--------------|
| **F053-REVIEW-SUMMARY в backlog/** | Пользователь: «файл явно находится не в той папке» | Жёсткое правило: `docs/reviews/` — только review-артефакты; `docs/workstreams/backlog/` — только WS-файлы |
| **Два плана доработки** | `idea-f053-dorabotka.md` и `idea-phase4-dorabotka.md` созданы параллельно | Перед созданием draft — поиск существующих `docs/drafts/idea-*.md`; один план на фичу |
| **Out of scope по умолчанию** | Пользователь: «никаких out of scope» — агент помечал задачи OOS | Skill @build/@design: по умолчанию все beads из review → в scope; OOS только при явном обосновании |

### 1.2 Дрейф состояния (beads ↔ workstreams ↔ INDEX)

| Проблема | Пример | Рекомендация |
|----------|--------|--------------|
| **Статус WS не совпадает с beads** | 00-053-06..09 Done в review, но `status: pending` в WS-файлах | После каждого @review — автоматическая сверка: `bd list --label F053` vs `grep status docs/workstreams/backlog/00-053-*.md` |
| **Beads уже закрыты кодом** | Агент создаёт 00-053-26..35, пользователь: «А эти находки не сделаны?» — часть beads уже исправлена | Перед созданием WS из beads — `bd show <id>` + grep по коду; чеклист «Bead X — проверить код» в @design |
| **Не закрывает beads после build** | Пользователь: «проверь и закрой сделанные beads» | @build skill: после успешного build — `bd close <bead_id> --reason "00-053-XX done"` в том же шаге |

### 1.3 Упрощение тестов и environment-specific поведение

| Проблема | Пример | Рекомендация |
|----------|--------|--------------|
| **Удаление integration-тестов** | 00-053-20: «Simplifying tests and removing integration tests that may cause hangs» | Запрет: не удалять тесты без явного AC «skip integration in CI»; вместо этого — `-short` flag, `t.Skip()` с проверкой env |
| **«Tests may be slow in this environment»** | Агент оставляет note вместо фиксации | Контракт: `go test -short` для CI; полный прогон — только локально; документировать в AGENTS.md |
| **Windows/Flock** | «syscall.Flock Unix-only... On Windows the build will fail» | Добавить `//go:build !windows` или stub для Windows; не оставлять как «known limitation» без build tag |

### 1.4 Пассивность — не предлагает следующий шаг

| Проблема | Пример | Рекомендация |
|----------|--------|--------------|
| **«Next: /build 00-053-22»** | Агент пишет next action, но пользователь должен вручную вызвать | @build skill: в конце — «Выполнить следующий build?» или кнопка/команда для копирования |
| **Не проверяет «что осталось»** | Пользователь: «Найди все оставшиеся и незакрытые недоделки» | @design: после создания WS — автоматический отчёт «Pending: 00-053-16..25. Рекомендуемый порядок: P0 → P1 → P2» |

---

## 2. Проблемы пользователя

### 2.1 Микроменеджмент и ручная координация

| Проблема | Пример | Рекомендация |
|----------|--------|--------------|
| **20+ последовательных /build** | Пользователь вызывает `/build 00-53-16`, `/build 00-53-17`, … 22 раза подряд | Добавить batch: `/build 00-053-16..25` или `/oneshot F053 --until review` |
| **«Найди оставшиеся», «А эти сделаны?»** | Пользователь вручную запрашивает сводку | Команда `/status F053` — pending WS, open beads, next recommended action |
| **«Разберись с scope-violation»** | Пользователь должен знать, какие beads куда отнести | @review: в выводе — «Рекомендуемые WS для этих beads: 00-053-36 (scope), 00-053-37 (promptops)» |
| **«продолжай F053»** | Неоднозначно: следующий build? design? review? | Конвенция: «продолжай» = `sdp-orchestrate --feature F053 --next-action`; документировать в AGENTS.md |

### 2.2 Несогласованность команд

| Проблема | Пример | Рекомендация |
|----------|--------|--------------|
| **Формат WS ID** | `/build 00-53-16` vs `/build 00-053-16` | Нормализация: всегда `00-FFF-SS` (3 цифры в feature); парсер принимает оба, выводит канонический |
| **Разные entry points** | /build, /oneshot, /review, /design — когда что? | Дерево решений в AGENTS.md: «Новая фича → @feature; исправления по review → @design → @build; полный цикл → @oneshot» |

### 2.3 Каскад review → design → build

| Проблема | Пример | Рекомендация |
|----------|--------|--------------|
| **Длинный цикл** | /review → 28 beads → /design → 10 WS → /build × 10 | Интеграция: @review в конце предлагает «Создать WS для P0/P1?» и вызывает @design с findings |
| **Round 2, Round 4…** | Каждый review добавляет beads; накопление | Консолидация: перед новым round — «Закрыть дубликаты?»; дедупликация по описанию |

---

## 3. Проблемы протокола

### 3.1 Множественные источники истины

| Проблема | Пример | Рекомендация |
|----------|--------|--------------|
| **INDEX vs mapping vs beads vs WS status** | 4 места с состоянием; drift неизбежен | Единый источник: `.sdp/checkpoints/F053.json` + `bd sync`; INDEX генерируется из checkpoint или наоборот |
| **Beads proliferation** | 90+ beads для F053; overlap | Правило: 1 bead = 1 уникальная проблема; перед созданием — поиск по title/description |
| **scope_files не совпадают с реальностью** | Guard помечает INDEX, mapping — out of scope; 00-053-36 создан для «fix scope» | scope_files = «все файлы, которые WS может легитимно менять»; meta-файлы (INDEX, mapping) — в scope для WS, которые их обновляют |

### 3.2 Guard и scope

| Проблема | Пример | Рекомендация |
|----------|--------|--------------|
| **Directory prefix matching** | 00-053-01, 02, 06 используют `verify/`, `quality/` — guard изначально не поддерживал | 00-053-36 добавил `inScope()` с prefix; документировать в guard skill: «scope_files поддерживает directory prefix» |
| **sdp guard activate** | Легко забыть; Git safety в skill, но не в pre-commit | Pre-build hook по умолчанию: `sdp guard activate` если не activated |

### 3.3 Форматы и контракты

| Проблема | Пример | Рекомендация |
|----------|--------|--------------|
| **ws-verdict schema evolution** | 15 файлов созданы; validation добавлена позже | CI: `jq -e .verdict docs/ws-verdicts/*.json` + schema validation в pre-commit |
| **Execution Report** | Добавляется в WS вручную; формат не стандартизирован | Шаблон в @build skill: «Execution Report: AC checklist, commits, quality gates»; парсер для машинной проверки |
| **TDD contract** | 00-053-37 добавил RED/GREEN/REFACTOR; не все builds его соблюдают | @build skill: обязательные маркеры в выводе; guard проверяет наличие |

### 3.4 @oneshot vs @build

| Проблема | Пример | Рекомендация |
|----------|--------|--------------|
| **@oneshot делает много, но без прозрачности** | «CI GREEN — @oneshot complete» — что именно сделано? | @oneshot: в конце — список выполненных WS, beads closed, commits |
| **@build — один WS** | Для 22 WS нужно 22 вызова | Режим `@build 00-053-16..20` — последовательное выполнение с отчётом после каждого |

---

## 4. Архитектурные предложения

### 4.1 Протокол: единый источник истины

```
Предложение: Checkpoint как primary state
- .sdp/checkpoints/F053.json содержит: phase, workstreams[], branch, last_commit
- INDEX.md генерируется: make index или sdp index --feature F053
- .beads-sdp-mapping.jsonl валидируется: wc -l == ls backlog/*.md | wc -l
- bd sync обновляет beads из checkpoint (или наоборот — детали в отдельном design)
```

**Плюсы:** Один источник; меньше drift.  
**Минусы:** Миграция; переписывание скриптов.

### 4.2 Команды: /status и batch build

```
Предложение 1: /status F053
- Вывод: Pending WS (00-053-38..43), Open beads (P0/P1), Next action (sdp-orchestrate --next-action)
- Реализация: skill или sdp status --feature F053

Предложение 2: /build 00-053-16..25
- Последовательное выполнение; после каждого — commit, beads close
- Stop on first failure; отчёт: N done, M failed, next: 00-053-XX
```

### 4.3 Skills: обязательные контракты

| Skill | Контракт | Проверка |
|-------|----------|----------|
| **@build** | TDD: RED/GREEN/REFACTOR в выводе; Execution Report в WS; ws-verdict создан | Post-build hook: grep -q "REFACTOR" или jq .verdict ws-verdicts/*.json |
| **@review** | Findings → beads; вердикт в .sdp/review_verdict.json | bd list --label review-finding |
| **@design** | Draft в docs/drafts/; WS созданы; mapping обновлён | wc -l .beads-sdp-mapping.jsonl == ls backlog/*.md |
| **@oneshot** | One advance per phase; checkpoint обновлён | jq .phase .sdp/checkpoints/F053.json |

### 4.4 Агент: чеклисты перед действиями

| Действие | Чеклист перед выполнением |
|----------|---------------------------|
| **Создать draft** | `ls docs/drafts/idea-*phase4*` — не дублировать |
| **Создать WS из beads** | Для каждого bead: `bd show <id>` + grep в коде — уже исправлено? |
| **После @review** | Сверка beads vs WS status; закрыть дубликаты; обновить INDEX |
| **После @build** | `bd close <bead>` для каждого bead в scope WS |
| **Разместить артефакт** | Review → docs/reviews/; WS → backlog/; plan → docs/plans/ |

### 4.5 Пользователь: сокращение ручной работы

| Было | Стало |
|------|-------|
| 22 × /build N | /build 00-053-16..37 или @oneshot F053 |
| «Найди оставшиеся» | /status F053 |
| «А эти сделаны?» | @design проверяет перед созданием WS |
| «продолжай» | Документировано: = next-action |
| «никаких OOS» | Skill: OOS только по обоснованию |

---

## 5. Приоритизация

| # | Предложение | Effort | Impact | Рекомендация |
|---|-------------|--------|--------|--------------|
| 1 | /status F053 | 1–2 дня | High | Сделать первым |
| 2 | Batch /build 16..25 | 0.5 дня | High | Простая итерация в skill |
| 3 | Чеклист «bead уже исправлен» в @design | 0.5 дня | High | Добавить в design skill |
| 4 | Правило размещения артефактов | 0.5 дня | Medium | Документ в AGENTS.md |
| 5 | Один план на фичу (поиск draft перед созданием) | 0.5 дня | Medium | Добавить в design skill |
| 6 | Checkpoint как primary state | 3–5 дней | High | Отдельный workstream |
| 7 | Post-build auto bd close | 0.5 дня | Medium | Добавить в build skill |
| 8 | Контракт TDD/verdict в guard | 1 день | Medium | Pre-build hook |

---

## 6. Следующие шаги

1. **Обсудить** — какие аспекты углубить? (протокол, skills, команды)
2. **Создать workstreams** — для предложений 1–5 как F053 Phase 5 или отдельная фича
3. **Обновить skills** — @build, @design, @review с чеклистами и контрактами
4. **Документировать** в AGENTS.md — дерево решений, «продолжай» = next-action, правила размещения

---

*Документ создан по результатам анализа логов агента. Обратная связь приветствуется.*
