# Sweep: Full-Graph Autonomous Delivery — Design

**Date:** 2026-04-19
**Status:** Draft
**Owner:** Андрей
**Feature:** TBD (создать FXXX после approval)

---

## Проблема

Текущий `/deliver` обрабатывает одну фичу за раз. Пользователь хочет одну команду `/sweep`, которая автономно пройдёт **весь** открытый backlog:

- Обходит граф beads с учётом зависимостей и приоритета
- Строит каждый issue (включая bugs и leaf-задачи)
- Проводит review каждого сделанного issue, находки оформляет как новые beads (P0 — в голову очереди)
- После закрытия epic/feature запускает жёсткий gate: регрессия + happy paths + e2e + docs alignment
- Создаёт PR на epic, прогоняет через `codex:rescue`, фиксит до чистоты
- Работает пока `bd list --status=open` не пуст

Прямолинейная markdown-инструкция на 200 строк не справится: нет надёжного state, конфликты в параллельных worktrees, поддельные "проверки" happy paths через prose, race на concurrent `bd create`, нет watchdog на сабагенты.

---

## Цели / Non-goals

### Цели

1. Одна команда `/sweep` обрабатывает весь backlog без вмешательства пользователя
2. Parallel build 4–5 worktrees без конфликтов по файлам
3. Epic gate реально выполняет регрессию и happy-paths как исполняемые скрипты
4. Recovery после компакшена / краша без потери прогресса
5. Ограничение на размер пробега: wall time + issue count + fix cycles
6. Graceful skip недоступной инфраструктуры (docker/kubectl/minikube), не фальсификация

### Non-goals

1. Не auto-merge epic PR в main — финальное merge остаётся за человеком
2. Не автогенерация happy-path скриптов — если нет исполняемого smoke, создаётся P2 задача на его написание
3. Не работа через MCP/API — только local beads + git + gh
4. Не модификация правил priority / topological порядка (это отдельная задача)

---

## Принципы

- **Состояние — в Go-бинарнике, не в prose.** Markdown-команда `/sweep` оркестрирует LLM-вызовы, но всё state-управление — в `cmd/sdp-sweep`.
- **Правда > оптимизм.** Если happy-path-скрипт отсутствует — это failure, не "проверил мысленно".
- **Graceful skip ≠ silent skip.** Пропущенные проверки явно перечисляются в итоговом отчёте и создают P2 issue на недоступную инфраструктуру.
- **Идемпотентность.** `sdp-sweep resume` после любого сбоя восстанавливает точное состояние.
- **One PR per epic.** Leaf-ветки вливаются в epic-ветку, PR в main один.

---

## Архитектурные решения

### AD-1: Разделение orchestrator (Go) и dispatcher (markdown)

**Проблема:** Markdown-команда не может надёжно хранить state, делать топосортировку, держать worktree pool.

**Решение:** Два слоя:

| Слой | Что делает |
|------|-----------|
| `cmd/sdp-sweep` (Go CLI) | state machine, граф, worktree pool, file locks, `bd` ops (сериализованные), git/gh ops, quality gates, graceful infrastructure probes, watchdog |
| `.claude/commands/sweep.md` | цикл: `next-batch` → dispatch субагентов → `record-*` → повторить; LLM-специфичное (codex:rescue, парсинг findings) |

Binary нельзя напрямую dispatching субагентов Claude Code — это делает агент. Но вся координация вокруг — код, не prose.

### AD-2: Ветвление — epic-branch с leaf-ветками

```
main
  └── epic/f134-ai-sdlc-wiring                   ← долгоживущая epic-ветка
        ├── f134-01-phase-commands               ← leaf, merge в epic при APPROVED
        ├── f134-02-...
        └── f134-07-...
```

**Правила:**
- Epic branch создаётся при первом leaf-taske этого epic
- Каждый leaf task: worktree от epic branch, не от main
- После review APPROVED: `git merge --no-ff` leaf в epic branch, удаление leaf worktree
- Epic gate: на уже объединённой epic branch
- PR создаётся из `epic/f134-*` в `main`

**Почему не N независимых PR:** теряется "epic gate" семантика — регрессию нужно гонять на объединённом состоянии, иначе два leaf'а могут пройти по отдельности и сломать друг друга при merge.

**Почему не rebase:** долгоживущая ветка + rebase = conflict storm при N>3 leaf'ах. Merge --no-ff сохраняет историю и проще для восстановления.

### AD-3: Независимость по Scope Files

Worktrees могут работать параллельно **только если** их file scopes не пересекаются.

**Источник scope:**
1. WS-файл `docs/workstreams/backlog/00-FFF-SS.md` имеет секцию `## Scope Files` со списком путей
2. Если секции нет — task считается wide-scope, стартует соло (sequential)
3. Два tasks с overlap хотя бы одного файла — sequential

Sweep orchestrator парсит Scope Files и строит граф conflict, топосортирует с учётом.

**Оптимизация:** `make-wide` workflow — если в FXXX нашлось ≥3 no-scope tasks подряд, создаётся P2 задача "добавить Scope Files к 00-FFF-XX".

### AD-4: State model

`.sdp/sweep/` — директория со всем state-ом пробега. Структура:

```
.sdp/sweep/
├── state.json                     # глобальное состояние пробега
├── queue.jsonl                    # очередь задач (append-only для auditability)
├── worktrees/
│   ├── f134-01.json               # per-worktree state
│   └── f134-02.json
├── epics/
│   └── f134.json                  # per-epic state (leaf progress, gate results)
└── log/
    └── 2026-04-19T10-00-00.jsonl  # event log для recovery и аналитики
```

**`state.json`:**
```json
{
  "started_at": "2026-04-19T10:00:00Z",
  "run_id": "sweep-2026-04-19-abc",
  "phase": "running|gating|paused|done",
  "stats": {
    "issues_processed": 12,
    "issues_remaining": 47,
    "fix_cycles_used": 3,
    "wall_seconds": 4521
  },
  "limits": {
    "max_issues": 200,
    "max_fix_cycles": 100,
    "max_wall_seconds": 28800,
    "max_parallel_worktrees": 5
  },
  "epics_in_flight": ["f134"],
  "blocked_on": []
}
```

**`worktrees/<id>.json`:**
```json
{
  "issue_id": "sdplab-kkk",
  "feature": "F134",
  "workstream": "00-134-01",
  "branch": "f134-01-phase-commands",
  "worktree_path": ".worktrees/f134-01-phase-commands",
  "parent_branch": "epic/f134-ai-sdlc-wiring",
  "scope_files": ["cmd/sdp/main.go", "internal/phases/..."],
  "step": "build|review|merged|failed",
  "dispatched_at": "2026-04-19T10:15:00Z",
  "subagent_deadline": "2026-04-19T10:45:00Z",
  "attempts": 1
}
```

**`epics/<f>.json`:**
```json
{
  "feature": "F134",
  "epic_branch": "epic/f134-ai-sdlc-wiring",
  "leaf_issues": {
    "sdplab-kkk": {"status": "merged"},
    "sdplab-zf6": {"status": "in_progress"}
  },
  "gate_state": "pending|running|passed|failed",
  "gate_results": {
    "regression": {"status": "pass", "failures": []},
    "happy_paths": {"status": "skipped", "reason": "no executable scripts"},
    "e2e": {"status": "skipped", "reason": "docker not running"},
    "docs_alignment": {"status": "fail", "issues_created": ["sdplab-new1"]}
  },
  "pr_number": null,
  "codex_cycles": []
}
```

### AD-5: CLI API бинарника

| Команда | Вход | Выход | Назначение |
|---------|------|-------|-----------|
| `sdp-sweep start` | `--limits ...` (опц.) | run_id, initial state | Старт пробега, снапшот backlog |
| `sdp-sweep next-batch` | — | JSON со списком issues до N штук + их scope + worktree paths | Следующая независимая пачка |
| `sdp-sweep record-build <id>` | `--result pass\|fail\|timeout`, `--findings <json>` | — | Сохранить результат build |
| `sdp-sweep record-review <id>` | `--verdict APPROVED\|FINDINGS`, `--findings <json>` | Создаёт P0/P2 bd-issues по findings | Сохранить результат review |
| `sdp-sweep merge-leaf <id>` | — | commit sha | Merge leaf branch в epic branch |
| `sdp-sweep epic-gate <F>` | — | JSON с результатами всех проверок | Регрессия + happy + e2e + docs |
| `sdp-sweep create-pr <F>` | — | PR number | `gh pr create` из epic branch |
| `sdp-sweep record-codex <F>` | `--findings <json>` | Создаёт P0 bd-issues | Результат codex:rescue |
| `sdp-sweep resume` | — | current state + next action hint | Recovery |
| `sdp-sweep status` | — | прогресс в текстовом виде | Для отчёта пользователю |
| `sdp-sweep done` | — | bool + final report | Проверить завершение |
| `sdp-sweep abort` | `--reason "..."` | — | Отменить пробег, оставить worktrees |

Все write-команды сериализуются через flock на `.sdp/sweep/state.json`.

### AD-6: Recovery model

Компакшен / краш / ручной Ctrl-C — всё один путь: `sdp-sweep resume`.

**Что делает resume:**
1. Читает `state.json`, `worktrees/*.json`, `epics/*.json`
2. Для каждого worktree в state=`build|review`: проверяет `git status` — если dirty и `dispatched_at + deadline < now()`, помечает `failed` (subagent timeout), отпускает worktree
3. Для каждого epic в `gate_state=running`: перезапускает gate с нуля (идемпотентно)
4. Возвращает агенту: `{phase: "running", next_action: "dispatch-batch", batch: [...]}` или `{phase: "gating", epic: "f134"}`

**Агент после resume:** делает ровно то, что binary сказал. Не "решает сам".

### AD-7: Happy paths — только исполняемые

Markdown-файлы в `docs/happy-paths/` — **документация**. Для sweep они не проверка.

**Требование:** для каждого `docs/happy-paths/<name>.md` должен существовать `scripts/happy-paths/<name>.sh`, который возвращает exit 0 при успехе.

**Если скрипта нет:** sweep создаёт P2 issue `"happy-path: convert <name> to executable smoke"`, в `gate_results.happy_paths` записывает `status: skipped, reason: "no executable for <name>"`.

**Структура скрипта:**
```bash
#!/usr/bin/env bash
set -euo pipefail
# Precondition: проверить инфраструктуру, выйти 77 (EX_SKIP) если нет
command -v docker >/dev/null || exit 77
docker info >/dev/null 2>&1 || exit 77

# Actual smoke
sdp discover ...
# проверки
```

Exit 77 — "skipped gracefully", sweep различает `fail` и `skip`.

### AD-8: E2e и graceful skip

`scripts/e2e_*.sh` и `scripts/smoke_*.sh` обрабатываются так же:
- exit 0 → pass
- exit 77 → skipped (инфраструктура отсутствует), создать P3 note-issue "infra: run e2e-X when X available"
- exit != 0/77 → fail → P1 bug

**Минимальный infrastructure probe набор:**
```bash
docker info            → наличие docker
kubectl cluster-info   → k8s
gh auth status         → GitHub CLI
go version             → Go (всегда есть)
```

Отсутствие docker/k8s не делает sweep неуспешным — делает epic gate **partial**.

### AD-9: Failure taxonomy

| Событие | Куда | Приоритет нового issue | Прерывает ли sweep |
|---------|------|---------|-------------------|
| Build fails after 3 retries | новый bug | P0 | нет, issue помечен failed, sweep идёт дальше |
| Review P1/P2 finding | новый bug | P0 (в голову очереди) | нет |
| Review P3 finding | новый task | P3 (в хвост) | нет |
| Subagent timeout | внутри, увеличить attempts | (нет нового issue) | нет до 3-й попытки, на 3-й → P1 bug |
| Gate regression red | новый bug | P0 | epic не закрывается, sweep идёт к другим epic |
| Gate happy-path red | новый bug | P0 | epic не закрывается |
| Gate e2e red | новый bug | P1 | epic не закрывается |
| Gate e2e skipped (infra) | новый note | P3 | нет, gate partial |
| Gate docs drift | новый task | P2 | epic не блокируется этим |
| Codex P0 finding | новый bug | P0 | нет, фиксится в epic branch, цикл продолжается |
| Fix cycle limit hit | abort pass | — | да, sweep стопается |
| Wall time limit hit | graceful pause | — | да, state сохраняется для resume |

### AD-10: Concurrent bd writes

Sqlite beads + 5 параллельных сабагентов = lock contention.

**Решение:** сабагенты **не вызывают `bd`**. Они возвращают findings JSON в `record-*` команду. Orchestrator Go-бинарника делает `bd create` под одним flock.

Это также централизует формат findings.

### AD-11: Watchdog и лимиты

| Лимит | Дефолт | Действие при срабатывании |
|-------|--------|--------------------------|
| `max_issues` | 200 | graceful stop + report |
| `max_fix_cycles` | 100 | graceful stop + report |
| `max_wall_seconds` | 8h | graceful pause (resume позже) |
| `max_parallel_worktrees` | 5 | throttling, не abort |
| `subagent_timeout_seconds` | 1800 (30m) | отмена сабагента, attempts++ |
| `subagent_max_attempts` | 3 | P1 bug, issue помечен failed |

Все параметры через `sdp-sweep start --max-issues=50 --max-wall=4h` или через env.

---

## Компоненты

### `internal/sweep/`

- `graph.go` — топосортировка, conflict graph по Scope Files
- `state.go` — read/write state files, flock-сериализация
- `worktree.go` — pool, create/cleanup, git ops
- `scopes.go` — парсер Scope Files из WS
- `gate.go` — orchestration регрессии / happy / e2e / docs
- `findings.go` — конверсия review/codex findings в bd-create команды
- `infra.go` — probes (docker/k8s/gh/...)
- `*_test.go` — unit-тесты на каждый модуль, integration на state transitions

### `cmd/sdp-sweep/main.go`

- Тонкий флаг-парсер, каждый subcommand → вызов `internal/sweep` функции
- Структура как у других `sdp-*` бинарников

### `.claude/commands/sweep.md`

Тонкая команда:

```markdown
Run `sdp-sweep start` to begin a new run, or `sdp-sweep resume` if .sdp/sweep/state.json exists.

Main loop (pseudocode):
  while sdp-sweep done returns false:
    action = sdp-sweep next-batch
    if action.type == "dispatch-build":
      parallel dispatch @build subagents per action.batch
      for each: sdp-sweep record-build <id> --result ... --findings ...
    if action.type == "dispatch-review":
      parallel dispatch @review subagents
      for each: sdp-sweep record-review <id> --verdict ... --findings ...
    if action.type == "epic-gate":
      sdp-sweep epic-gate <F>
    if action.type == "create-pr":
      sdp-sweep create-pr <F>
      dispatch codex:rescue with "run quality gates, report all findings"
      sdp-sweep record-codex <F> --findings ...

  report = sdp-sweep status
  print report
```

### `scripts/happy-paths/`

Новая директория. Для каждого файла в `docs/happy-paths/` — одноимённый `.sh`-скрипт.

---

## Открытые вопросы

1. **Epic без beads-issue epic-уровня.** Некоторые features в репо представлены только leaf-тасками (sdplab-lqb = F106-07, но есть ли sdplab для F106 целиком?). Если нет — sweep не знает что epic "закрылся". Варианты:
   - (a) Создавать epic issue при первой встрече leaf'а FXXX
   - (b) Epic считается закрытым когда все F<N>-XX из `docs/workstreams/backlog/` закрыты
   - Склоняюсь к (b), проще и детерминированнее.

2. **Bugs не в epic.** Standalone bugs (без F-номера) получают собственные PR или идут в "misc-fixes" epic? Предлагаю misc-epic `epic/fixes-YYYY-MM-DD`, PR раз в день.

3. **Что если leaf-таск провалился 3 раза?** Sweep идёт дальше, но epic gate не пройдёт из-за неготового leaf'а. Блокировать весь epic или закрывать "частично"? Предлагаю: epic помечается `gate_state: blocked_on_failed_leaf`, gate не запускается, PR не создаётся, sweep переходит к следующему epic.

4. **Priority инверсия.** P0 bug создан во время build F134. Предлагается "P0 в голову". Но 5 worktrees уже работают на F134. Варианты:
   - (a) Дождаться завершения текущего batch, потом P0
   - (b) Прервать worktrees на F134 низшего приоритета
   - Склоняюсь к (a), проще.

5. **Interactive моменты.** `gh pr create` может запросить auth. `git push` может запросить credentials. Sweep должен fail-fast при таких prompts, не зависать. Через `GIT_TERMINAL_PROMPT=0` + проверка `gh auth status` в `sdp-sweep start`.

6. **Storage мусор.** После сотни worktrees `.worktrees/` разрастётся. `sdp-sweep cleanup` удаляет merged worktrees. Вызывать автоматически после merge-leaf.

---

## Non-goals (явно не делаем)

- Автомерж в main — финальный merge делает человек
- Параллельные epics — один epic gate за раз (избегаем гонок на main-level регрессии)
- Кросс-репозиторные sweep'ы — только в текущем репо
- Отмена codex в runtime — ждём таймаут

---

## План реализации

1. **M1 — Design review** (этот документ) + создание FXXX epic
2. **M2 — `internal/sweep/` core** (graph, state, scopes, worktree) + unit-тесты
3. **M3 — `cmd/sdp-sweep`** с subcommands `start/next-batch/record-*/resume/status/done`
4. **M4 — `epic-gate` + `create-pr`** + integration test против реального repo
5. **M5 — `.claude/commands/sweep.md` thin wrapper**
6. **M6 — happy-paths скрипты** — конвертировать существующие 4 файла
7. **M7 — E2E dry-run** на маленьком suite (2–3 leaf tasks)

---

## Критерии приёмки

- [ ] `sdp-sweep start` создаёт state, снапшот backlog
- [ ] `next-batch` возвращает топологически корректные пачки с non-overlapping scope
- [ ] 5 параллельных worktrees без file-conflict
- [ ] Recovery после `kill -9` восстанавливает точное состояние
- [ ] Epic gate запускает регрессию + happy-paths + e2e + docs, graceful-skip без docker/k8s
- [ ] Findings review → P0 bugs, попадающие в голову очереди
- [ ] Codex:rescue findings тоже → P0 bugs в epic branch
- [ ] Один PR на epic с правильным merge --no-ff history
- [ ] Лимиты работают (max_issues, max_wall, max_fix_cycles)
- [ ] Отчёт показывает: closed / PRs / skipped infra / P0 created / wall time

---

## Ссылки

- `AGENTS.md` — beads workflow, quality gates
- `.agents/skills/build.md` — Session Bootstrap
- `.agents/skills/delivery-loop.md` — существующий per-feature loop
- `.agents/skills/review.md` — dimensions
- `docs/happy-paths/` — текущие 4 happy paths (markdown)
- `docs/reference/go-patterns.md` — Go conventions
