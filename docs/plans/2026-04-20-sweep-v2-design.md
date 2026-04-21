# Sweep v2: Full-Graph Autonomous Delivery — BASELINE Design

**Date:** 2026-04-20
**Status:** BASELINE v2 — ревизия v1 для углублённой проработки
**Supersedes:** [2026-04-19-sweep-design.md](2026-04-19-sweep-design.md)
**Dependencies (HARD):**
- [2026-04-20-sdp-framework-normalization-design.md](2026-04-20-sdp-framework-normalization-design.md) — до `F137-03` как минимум; `F139` желателен для full parity/discovery
- [2026-04-20-mini-harness-orchestrator-design.md](2026-04-20-mini-harness-orchestrator-design.md) — до M3 (dispatch core) минимум; M4+ желательно

**Sequencing:** sweep — **последний из трёх** эпиков. Не начинаем implementation пока `F137` CLI contract и mini-harness dispatch core не зелёные.

---

## 1. Диф vs v1 (краткая версия — полный список в §9)

1. **k8s удалён везде.** probes = `docker`, `gh`, `go`. Нет `kubectl`, `minikube`.
2. **Sweep = `sdp sweep` subcommand**, не `cmd/sdp-sweep/`. Лежит в `cmd/sdp/cmd_sweep.go` + `internal/sweep/`.
3. **Harness dispatch через оркестратор.** Sweep не вызывает Claude subagents напрямую — делегирует задачи в mini-harness orchestrator (pool claude/codex/opencode/cursor с rotation).
4. **bd writes централизованы в оркестраторе**, не в sweep. flock остаётся как belt-and-braces.
5. **Новые AD:** harness dispatch integration (AD-11), sequencing gate criteria (AD-12), оркестратор ownership (AD-13).

---

## 2. Контекст

Контекст v1 сохраняется: нужна одна команда, которая прогоняет весь backlog автономно, строит/ревьюит каждый issue, собирает findings как новые beads, после закрытия epic запускает регрессию + happy-paths + e2e + docs, открывает PR, прогоняет через codex:rescue.

Что изменилось: теперь sweep не рождается в вакууме — он встраивается в унифицированный `sdp` CLI (normalization epic) и использует orchestrator для dispatch задач (mini-harness epic). Это меняет границы и некоторые AD из v1.

---

## 3. Цели / Non-goals

### Цели (v2)

1. Одна команда `sdp sweep` / `/sweep` обрабатывает весь backlog без вмешательства
2. Parallel build 4–5 worktrees через orchestrator, harness-rotation на 429/quota
3. Epic gate реально выполняет регрессию и happy-paths как исполняемые скрипты
4. Recovery после компакшена / краша / рестарта оркестратора без потери прогресса
5. Graceful skip недоступной инфраструктуры (docker/gh) — не фальсификация
6. Sweep — часть `sdp`, не standalone (consistent с остальными subcommands)

### Non-goals

1. Не auto-merge epic PR в main — финальное merge остаётся за человеком
2. Не автогенерация happy-path скриптов — если нет исполняемого smoke, создаётся P2 задача на его написание
3. **НЕТ k8s** в probes, happy paths, smoke tests. Исключено целиком.
4. Не модификация правил priority / topological порядка
5. Не собственный harness — всё через orchestrator pool
6. Не работа через MCP/API как primary interface (но MCP exposure автоматом через registry — см. normalization AD-3)

---

## 4. Принципы (обновлены)

- **Состояние — в Go, не в prose.** State-машина в `internal/sweep/`.
- **Orchestrator = единственный owner harness-вызовов.** Sweep шлёт задачи в orchestrator, не знает какой harness выполнил.
- **Orchestrator = единственный writer в beads** при выполнении задач. Sweep пишет только state-снапшоты и sweep-specific issues (P0 findings после review).
- **Правда > оптимизм.** Отсутствие happy-path скрипта — failure, не "проверил мысленно".
- **Graceful skip ≠ silent skip.** Пропуски явно перечисляются в итоговом отчёте + P2/P3 issues.
- **Идемпотентность.** `sdp sweep resume` после любого сбоя восстанавливает точное состояние.
- **One PR per epic.** Leaf-ветки вливаются в epic-ветку через `merge --no-ff`, PR в main один.

---

## 5. Архитектурные решения

### AD-1 — Sweep = `sdp sweep` subcommand (REVISED)

**v1 сказал:** `cmd/sdp-sweep/` + `.claude/commands/sweep.md`.

**v2:** `cmd/sdp/cmd_sweep.go` регистрирует команду в `sdpcli.Registry`. Реализация — `internal/sweep/`. Тонкий `.claude/commands/sweep.md` форвардит на `sdp sweep start|resume`.

**Почему:** консистентность с остальными командами (normalization AD-1), automatic MCP exposure (normalization AD-3).

Subcommands sweep'а (second level):
```
sdp sweep start [--limits ...]
sdp sweep next-batch
sdp sweep record-build <id> --result --findings
sdp sweep record-review <id> --verdict --findings
sdp sweep merge-leaf <id>
sdp sweep epic-gate <F>
sdp sweep create-pr <F>
sdp sweep record-codex <F> --findings
sdp sweep resume
sdp sweep status
sdp sweep done
sdp sweep abort --reason "..."
```

### AD-2 — Ветвление: epic-branch с leaf-ветками (UNCHANGED)

Структура и логика из v1 сохраняются:
```
main
  └── epic/f134-ai-sdlc-wiring
        ├── f134-01-phase-commands
        ├── f134-02-...
```

Leaf → epic: `git merge --no-ff`. PR только в main из epic branch. Без изменений относительно v1.

### AD-3 — Независимость по Scope Files (UNCHANGED)

Парсер секции `## Scope Files` в WS-файлах. Overlap → sequential. Без секции → wide-scope, solo. Ни одного изменения относительно v1.

### AD-4 — State model (MINOR CHANGE)

`.sdp/sweep/` — та же структура, что в v1. Но добавляется поле `orchestrator_run_id` в `state.json`:

```json
{
  "run_id": "sweep-2026-04-20-abc",
  "orchestrator_run_id": "orch-...",  // NEW: привязка к прогону оркестратора
  "phase": "running|gating|paused|done",
  ...
}
```

И в `worktrees/<id>.json`:
```json
{
  "issue_id": "sdplab-kkk",
  "dispatch_id": "disp-xyz",        // NEW: id задачи в orchestrator
  "harness_used": "claude-1",        // NEW: фиксация какой harness выполнил
  "attempts_per_harness": {"claude-1": 1, "codex-1": 1}, // NEW: retry trace
  ...
}
```

### AD-5 — CLI API (REVISED под `sdp sweep`)

См. AD-1 — все операции через `sdp sweep <subcmd>`. Write-ops сериализуются через flock на `.sdp/sweep/state.json`.

### AD-6 — Recovery (UPDATED)

`sdp sweep resume`:
1. Читает state
2. Запрашивает у orchestrator: `sdp orchestrate status --run-id <orchestrator_run_id>` — актуальные статусы задач
3. Reconciliation: orchestrator может знать "задача X завершена", sweep state ещё нет → применить
4. Worktrees с протухшим deadline и пустыми коммитами → failed, attempts++
5. Epic с `gate_state=running` → перезапустить gate с нуля
6. Возвращает агенту next action

Новое: sweep полагается на orchestrator как persistent state — если orchestrator daemon перезагружен, но state.db цел, sweep продолжает без потерь.

### AD-7 — Happy paths: только исполняемые (UNCHANGED)

Требование: `scripts/happy-paths/<name>.sh` exit 0 = pass, exit 77 = skip, иначе fail. Markdown-only → P2 issue "happy-path: convert <name> to executable smoke".

### AD-8 — Infrastructure probes — БЕЗ k8s (REVISED)

**v1 было:** `docker info`, `kubectl cluster-info`, `gh auth status`, `go version`.

**v2:** `docker info`, `gh auth status`, `go version`. Точка.

Нет `kubectl`, нет `minikube`. Это не часть проекта.

Exit-коды скриптов:
- 0 → pass
- 77 → skipped gracefully (infra missing), P3 note-issue
- другое → fail → P1 bug

### AD-9 — Failure taxonomy (ADJUSTED)

| Событие | Куда | Приоритет | Прерывает ли sweep |
|---------|------|-----------|--------------------|
| Build fails after 3 retries (all harness) | новый bug | P0 | нет |
| Review P1/P2 finding | новый bug | P0 (head of queue) | нет |
| Review P3 finding | новый task | P3 (tail) | нет |
| Orchestrator rotation exhausted | новый bug "all harness rate_limited" | P1 | pause until reset |
| Subagent timeout (per-harness) | orchestrator retries | — | нет до max_attempts |
| Gate regression red | новый bug | P0 | epic не закрывается |
| Gate happy-path red | новый bug | P0 | epic не закрывается |
| Gate e2e red | новый bug | P1 | epic не закрывается |
| Gate e2e skipped (infra missing) | новый note | P3 | нет, gate partial |
| Gate docs drift | новый task | P2 | epic не блокируется |
| Codex:rescue P0 finding | новый bug | P0 | нет |
| Fix cycle limit hit | abort | — | да |
| Wall time limit hit | graceful pause | — | да (resume позже) |

### AD-10 — Beads write ownership (REVISED — v1 был противоречив)

**v1 говорил:** субагенты не вызывают bd, orchestrator sweep пишет под flock.

**v2:** Теперь есть два уровня orchestration:
- **Mini-harness orchestrator** владеет `bd claim`, `bd close` для выполняемых задач
- **Sweep** владеет созданием новых issues (findings → P0/P1/P2/P3) и epic gate outcomes

Invariant: каждый `bd <op>` имеет единственного владельца. Для per-task state (claim, close) — orchestrator. Для derived issues (findings, gate results) — sweep.

flock остаётся как safety net на `.sdp/sweep/state.json`, не на beads DB — для beads надеемся на orchestrator sequencing.

### AD-11 — Harness dispatch via Orchestrator (NEW)

Sweep **не вызывает** Claude Code / Codex subagents напрямую. Sweep:
1. Формирует `Request` (task description, workdir=worktree, capability=`code_gen` для build, `code_review` для review)
2. Отправляет в orchestrator: `sdp orchestrate submit --request <json>` → получает `dispatch_id`
3. Подписывается на events (long-poll или stream): `sdp orchestrate events --dispatch <id>`
4. По `task_completed` / `task_failed` — обрабатывает result, `sdp sweep record-<phase>`

Orchestrator сам выбирает harness, retry'ит на 429/timeout/rotation, возвращает финальный Result.

Fallback: если `sdp orchestrate daemon` недоступен, sweep может работать в **single-harness legacy mode** (exec `claude` напрямую), помечая runs как `degraded`. Это временно до stable orchestrator.

### AD-12 — Sequencing gate (NEW)

Sweep **проверяет зависимости** в `sdp sweep start`:
1. `sdp version` даёт `{registry_hash, mcp_schema_version}` → требуется minimum ≥ X (из normalization)
2. `sdp orchestrate status` → требуется `daemon_up=true`, `harness_count≥1` (из mini-harness)

Если не удовлетворены:
- registry mismatch → abort с "Run `sdp upgrade` first"
- orchestrator down → warn + offer legacy single-harness mode (interactive prompt? no — `--allow-degraded` flag)

### AD-13 — Watchdog и лимиты (UNCHANGED from v1)

| Лимит | Дефолт | Действие |
|-------|--------|----------|
| `max_issues` | 200 | graceful stop + report |
| `max_fix_cycles` | 100 | graceful stop + report |
| `max_wall_seconds` | 8h | graceful pause (resume) |
| `max_parallel_worktrees` | 5 | throttling |
| `subagent_timeout_seconds` | 1800 | orchestrator кидает на retry |
| `subagent_max_attempts` | 3 (через все harness'ы) | P1 bug, failed |

---

## 6. Компоненты

### `internal/sweep/`
- `graph.go` — топосортировка, conflict graph по Scope Files
- `state.go` — read/write state, flock
- `worktree.go` — git ops, pool
- `scopes.go` — парсер Scope Files
- `gate.go` — регрессия / happy / e2e / docs
- `findings.go` — review/codex findings → bd-create
- `infra.go` — probes (docker/gh/go) — **без k8s**
- `orchestrator_client.go` — NEW: клиент к orchestrator daemon
- `*_test.go`

### `cmd/sdp/cmd_sweep.go`

Subcommands через registry (normalization AD-2). Thin — всё в `internal/sweep/`.

### `.claude/commands/sweep.md` (thin forwarder)

```markdown
Run `sdp sweep start` to begin a new run, or `sdp sweep resume` if .sdp/sweep/state.json exists.

Main loop:
  while $(sdp sweep done) returns "no":
    action=$(sdp sweep next-batch)
    case "$action.type":
      dispatch-build | dispatch-review:
        Orchestrator handles harness selection + execution.
        Wait for completion via `sdp orchestrate events --dispatch $id`.
        `sdp sweep record-build|review <id> --result ... --findings ...`
      epic-gate: `sdp sweep epic-gate <F>`
      create-pr:
        `sdp sweep create-pr <F>`
        Dispatch codex:rescue with "run quality gates, report findings".
        `sdp sweep record-codex <F> --findings ...`

  `sdp sweep status`
```

### `scripts/happy-paths/`

Новая директория. Для каждого `docs/happy-paths/<name>.md` — `<name>.sh` (exit 77 при infra missing, 0 при pass).

---

## 7. Open Questions

### Перенесено из v1
1. **Epic без beads-issue epic-уровня.** Предлагается (b): epic = все F<N>-XX из `docs/workstreams/backlog/`. К обсуждению.
2. **Standalone bugs (без F).** `misc-fixes` epic с daily PR? Или per-bug PR? Мнение склоняется к `epic/fixes-YYYY-MM-DD`.
3. **Failed leaf блокирует epic?** Предлагается `gate_state: blocked_on_failed_leaf`, epic не закрывается, sweep идёт дальше.
4. **Priority инверсия при P0 в runtime.** Предлагается (a): дождаться текущего batch, потом P0. Простое решение.
5. **Interactive prompts (gh auth / git push).** Fail-fast через `GIT_TERMINAL_PROMPT=0` + `gh auth status` в `sdp sweep start`.
6. **Storage мусор.** `sdp sweep cleanup` после merge-leaf.

### Новые (v2)

7. **Orchestrator config location.** Где sweep берёт адрес daemon (unix socket vs HTTP port)? `.sdp/config.toml`? env `SDP_ORCHESTRATOR_URL`?

8. **Quota exhaustion behavior.** Если все harness'ы rate_limited → sweep pause или abort? Предлагается pause с re-check каждые 5 мин до max_wall.

9. **Registry version check granularity.** Проверяем `registry_hash` (точное совпадение) или `cli_version` (semver range)? Первое строже, второе гибче.

10. **Forwarder deprecation.** `.claude/commands/sweep.md` через сколько релизов после появления `sdp sweep` можно удалить?

11. **Sequencing strict or soft.** AD-12 говорит blocker по зависимостям. Но если normalization отстал — пользователь не может запустить sweep для demo. Soft mode с big warning? Решаем на M5.

12. **Orchestrator failure semantics mid-sweep.** Daemon крашится между `submit` и `record-*`. Sweep должен обнаружить (polling status) и пересабмитить? Или requeue из dispatch log?

---

## 8. Dependencies (detailed)

### HARD
| Эпик | Минимальная веха | Причина |
|------|------------------|---------|
| SDP Framework Normalization | M3 (registry), M5 (MCP proxy) | `sdp sweep` subcommand needs registry; MCP exposure automatic |
| Mini-Harness Orchestrator | M3 (dispatch core + SQLite state) | Sweep submits tasks to orchestrator |

### SOFT
| Зависимость | Комментарий |
|-------------|-------------|
| Mini-Harness M4 (multi-harness) | Желательно для rotation, иначе single-harness mode |
| Mini-Harness M5 (triage) | Nice-to-have для cost optimization |
| beads >= v-current | `bd ready --json`, `bd blocked --json`, `bd create` stable |
| docker runtime (host) | Quality gates через `./scripts/run_go_quality_gates.sh` |
| gh CLI auth | `gh pr create`, `gh auth status` |

### NOT a dependency
- ~~kubectl / minikube~~ — убрано в v2
- superpowers skills — не используем

---

## 9. Implementation Plan (M1-M7)

**M1 — Design review + FXXX epic creation** (1 день): прочитать spec, создать epic `sdplab-F135` + leaf issues для M2-M7.

**M2 — `internal/sweep/` core** (4-5 дней): graph, state, scopes, worktree, infra (без k8s). Unit-tests. Orchestrator client — stub.

**M3 — `sdp sweep` subcommand** (3 дня): регистрация в registry, subcommands `start/next-batch/record-*/resume/status/done/abort`. Integration test против fixture backlog.

**M4 — Orchestrator integration** (3-4 дня): real `orchestrator_client.go`, поддержка submit/events/status. Fallback legacy mode. Dry-run на 2-3 real leaf tasks.

**M5 — Epic gate + PR** (3-4 дня): gate.go (регрессия/happy/e2e/docs), `create-pr`, `record-codex`. Happy-paths scripts конвертация (M6 параллельно).

**M6 — `scripts/happy-paths/` conversion** (2 дня): текущие 4 md-файла → исполняемые .sh с exit 77 контрактом.

**M7 — E2E dry-run + docs** (2 дня): полный прогон на small backlog (3-5 issues), README, `docs/reference/sweep.md`.

Итого: ≈2.5 недели calendar после того как зависимости shipped.

---

## 10. Acceptance Criteria

- [ ] `sdp sweep start` создаёт state, снапшот backlog
- [ ] `sdp sweep next-batch` возвращает топологически корректные пачки с non-overlapping scope
- [ ] 5 параллельных worktrees через orchestrator без file-conflict
- [ ] Recovery после `kill -9` sweep И orchestrator восстанавливает точное состояние
- [ ] Harness rotation: убиваем Claude mid-run, sweep продолжает на Codex без остановки
- [ ] Epic gate запускает регрессию + happy-paths + e2e + docs, graceful-skip без docker
- [ ] **Нет k8s-ссылок** нигде в коде / docs / scripts
- [ ] Findings review → P0 bugs, попадающие в голову очереди
- [ ] Codex:rescue findings → P0 bugs в epic branch
- [ ] Один PR на epic с правильным merge --no-ff history
- [ ] Лимиты работают (max_issues, max_wall, max_fix_cycles)
- [ ] Отчёт: closed / PRs / skipped infra / P0 created / wall time / harness distribution

---

## 11. Полный Diff vs v1

1. **k8s удалён** из probes, happy-paths, e2e, всех упоминаний
2. **`cmd/sdp-sweep/` → `cmd/sdp/cmd_sweep.go`** (AD-1)
3. **Добавлена dependency на orchestrator** (HARD) (AD-11, AD-12)
4. **Добавлено поле `orchestrator_run_id`, `dispatch_id`, `harness_used`** в state (AD-4)
5. **Recovery синхронизируется с orchestrator state** (AD-6)
6. **`sdp sweep start` делает sequencing gate check** (AD-12)
7. **Legacy single-harness fallback mode** через `--allow-degraded` (AD-11)
8. **bd write ownership разделён** между orchestrator (per-task) и sweep (findings/gate) (AD-10)
9. **`.claude/commands/sweep.md` стал thin forwarder** на `sdp sweep` (AD-1)
10. **Добавлены open questions 7-12** (orchestrator config, quota exhaustion, version check, forwarder deprecation, soft sequencing, mid-run daemon crash)
11. **Sequencing explicit:** sweep — последний из трёх, не начинаем до norm M3 + orch M3
12. **Acceptance добавил harness rotation test** (killing Claude, continuing on Codex)

---

## 12. Non-Goals (explicit)

- Auto-merge в main
- Параллельные epics одновременно
- Кросс-репозиторные sweep'ы
- Отмена codex:rescue в runtime
- Собственный harness внутри sweep (всё через orchestrator)
- **k8s / minikube / kubectl** — исключено
- WebUI / TUI — только CLI
