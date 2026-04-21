# Mini-Harness + Orchestrator with Small-Model Dispatch — BASELINE Design

**Date:** 2026-04-20
**Status:** BASELINE — draft для углублённой проработки
**Related:** [2026-04-10-sdp-mini-harness-design.md](../archive/plans/2026-04-10-sdp-mini-harness-design.md) (archived — другой scope: phase-based single-session harness), [2026-04-20-sdp-framework-normalization-design.md](2026-04-20-sdp-framework-normalization-design.md) (surface-contract dependency via `F137`)
**Owner:** TBD
**Dependencies:** Surface Contract Normalization `F137-02` + `F137-03` (нужен stable `sdpcli` registry and migrated command surface)

---

## 1. Контекст и проблема

Основной способ ведения работы в `sdp_lab` сегодня — одна long-running Claude Code сессия, в которой агент последовательно или через subagent'ы выполняет задачи (`/deliver`, planned `/sweep`). Это бутылочное горлышко:

1. **Rate-limit = death.** Claude 429 → сессия встаёт. Нет automatic fallback.
2. **Compaction рушит контекст.** При подходе к лимиту токенов Claude Code делает compaction, ломая сессию и workflow. Это подтверждено session-audit.
3. **Одна модель для всего.** Triage mundane-задач (классификация ошибок, routing, пересказ diff) выполняется той же моделью, что и решения — overkill по стоимости.
4. **Нет параллелизма между harness'ами.** Есть Claude CLI, Codex CLI, OpenCode CLI, Cursor agent mode. Но оркестрации между ними — нет.
5. **Fragmented existing code.** `cmd/sdp-dispatch` (58 LOC), `cmd/sdp-harness` (464 LOC), `cmd/sdp-orchestrate` (218 LOC), `cmd/sdp-orchestrate-daemon` (35 LOC), `cmd/sdp-a2a` (44 LOC), `cmd/sdp-control` (708 LOC) — все адресуют близкие задачи, но не собраны в coherent pipeline. Первоначальный mini-harness design 2026-04-10 ушёл в другую сторону (phase-based loop внутри одной сессии).

### Мотивация пользователя

> "Требуется конфигурация с dispatch-балансировкой по разным harness, чтобы лимит не убивал процесс, а заставлял оркестратор отправлять больше задач на другой harness (сделать оркестратор на pi?)"

Цель: отделить координацию от исполнения. Оркестратор = лёгкий процесс (может жить на Raspberry Pi) — он знает очередь задач (beads), пул harness'ов и их бюджеты, раздаёт задачи, собирает результаты. Harness'ы = тяжёлые LLM CLI — выполняют код.

---

## 2. Цели (North Star)

1. **Harness pool с rotation.** Оркестратор держит N harness'ов разного типа. При 429/quota/timeout — перекидывает задачу.
2. **Small models как первый фильтр.** Задачи classify/triage/route через haiku/gpt-5-mini. Большие модели — только на решения.
3. **Pi-deployable.** Оркестратор = <100MB RAM, стабильный процесс, state в local SQLite.
4. **Транспорт-агностик.** Harness'ы могут жить на той же машине (local exec), в Docker, на другой машине (SSH/HTTP).
5. **Beads = источник правды.** Оркестратор читает `bd ready`, пишет статусы, фиксирует claim.
6. **Не ломает текущий workflow.** Можно включать incrementally — сначала для sweep, потом для deliver.

---

## 3. Компонентная архитектура

```
┌────────────────────────────────────────────────────────────────┐
│                    Orchestrator (на Pi / local)                 │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────┐        │
│  │ TaskQueue    │   │ HarnessPool  │   │ DispatchLogic│        │
│  │ (beads)      │─▶ │ (registry)   │◀──│ (policy)     │        │
│  └──────────────┘   └──────────────┘   └──────────────┘        │
│         │                   │                   │               │
│         ▼                   ▼                   ▼               │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────┐        │
│  │ StateStore   │   │ BudgetTracker│   │ TriageModel  │        │
│  │ (SQLite)     │   │ (per-harness)│   │ (haiku/mini) │        │
│  └──────────────┘   └──────────────┘   └──────────────┘        │
└────────────────────────────────────────────────────────────────┘
                            │
          ┌─────────────────┼─────────────────┬────────────┐
          ▼                 ▼                 ▼            ▼
     ┌─────────┐       ┌─────────┐       ┌─────────┐   ┌─────────┐
     │ Claude  │       │ Codex   │       │ OpenCode│   │ Cursor  │
     │ harness │       │ harness │       │ harness │   │ harness │
     └─────────┘       └─────────┘       └─────────┘   └─────────┘
          │                 │                 │            │
          └─────────────────┴─────────────────┴────────────┘
                            │
                   Git worktrees + beads
```

---

## 4. Архитектурные решения

### AD-1 — Orchestrator as daemon, не embedded

Оркестратор — отдельный long-running процесс `sdp orchestrate daemon`. Альтернатива (embedded library в sweep/deliver) отклонена: state должен переживать падение CLI-инициатора.

`cmd/sdp-orchestrate-daemon/` (35 LOC сейчас) расширяется до полноценного daemon с HTTP + gRPC endpoint'ами.

### AD-2 — Harness abstraction

`internal/harness/harness.go`:

```go
type Harness interface {
    Name() string                               // "claude", "codex", ...
    Capabilities() []Capability                 // code_gen, review, planning, ...
    Invoke(ctx context.Context, req Request) (*Result, error)
    Health() Status                             // alive, rate_limited, dead
    Budget() BudgetSnapshot                     // remaining tokens / requests / $
}

type Request struct {
    TaskID       string
    Prompt       string
    Workdir      string             // путь к git worktree
    Model        string             // опциональный hint
    Tools        []string           // allowlist
    Timeout      time.Duration
    Metadata     map[string]string  // beads id, phase, etc.
}

type Result struct {
    Status       ResultStatus       // success, failed, rate_limited, timeout
    Output       string
    Artifacts    []Artifact         // files changed, PRs created
    TokensUsed   int
    Duration     time.Duration
    HarnessError error
}
```

Реализации: `ClaudeHarness` (exec `claude` CLI), `CodexHarness` (exec `codex` CLI), `OpenCodeHarness`, `CursorHarness`. Реализация `MockHarness` для тестов.

### AD-3 — Dispatch policy = pluggable, default = capability-match + budget-aware

```go
type DispatchPolicy interface {
    Select(pool []Harness, task Task) (Harness, error)
}
```

Default implementation (`BudgetAwareCapabilityPolicy`):
1. Отфильтровать harness'ы по Capabilities ∋ task.required_capability
2. Отфильтровать по Health() == alive (rate_limited → backoff until reset)
3. Отсортировать по (budget_remaining DESC, last_used ASC)
4. Top-1

При 429 → `harness.MarkRateLimited(reset_at)`, задача re-dispatch через policy.

Альтернативные policies: RoundRobin (dev), CostOptimized (prefer cheapest), AffinityBased (sticky per-epic).

### AD-4 — Small-model triage layer

`TriageModel` — haiku или gpt-5-mini вызов перед основным dispatch:

Входы:
- Task description (from beads)
- Task type (build/review/fix)

Выходы (JSON):
- `required_capability`: code_gen | code_review | planning | trivial_edit
- `estimated_complexity`: low | medium | high
- `suggested_harness`: claude | codex | opencode | any
- `skip_reason`: "duplicate" | "ambiguous" | "needs_human" | null

Результат triage → влияет на policy selection. Задачи с `needs_human` → `bd human <id>`.

Где живёт TriageModel: отдельный subprocess через тот же Harness interface (MiniModelHarness), чтобы rate-limit маленькой модели тоже балансировался.

### AD-5 — State persistence: SQLite + beads

- Оркестратор-внутренний state: `~/.sdp/orchestrator/state.db` (SQLite):
  - `dispatches` (task_id, harness, started_at, ended_at, result)
  - `budget_snapshots` (harness, ts, tokens_left, requests_left)
  - `rate_limits` (harness, reset_at, reason)
- Task ownership: **beads** (`bd update <id> --claim`). Оркестратор claim'ит от своего имени (`assignee=orchestrator`), harness получает worktree + task description.

### AD-6 — Transport: local exec по умолчанию, SSH/HTTP опционально

- **Local exec** (default): harness CLI бинарь на той же машине. Оркестратор через `os/exec`.
- **SSH exec**: `HarnessConfig.Transport = "ssh"`, `Host = "workstation.local"`. Оркестратор на Pi, harness'ы на мощных машинах.
- **HTTP**: длинный poll API (`POST /invoke`, `GET /result/{id}`). Для облачных раннеров.

Общий interface — один, transport ортогонален. По умолчанию local, SSH — для Pi-deployment.

### AD-7 — Pi deployment profile

Оркестратор оптимизирован для ARM + 1GB RAM:
- Go binary, static link, <30MB
- SQLite state <100MB для месяца работы
- Нет embedded models — triage через API (Anthropic / OpenAI) или remote harness
- Systemd unit `sdp-orchestrate.service`
- Prometheus metrics endpoint `:9090/metrics`
- Health check `/healthz`

Deploy skit: `scripts/deploy_orchestrator_pi.sh` — ssh + systemd + config.

### AD-8 — Budget tracking и rate-limit heuristics

Два уровня:
- **Внешний 429** → harness возвращает `ResultStatus.rate_limited` + `Retry-After` header (если parseable). Оркестратор маркирует harness dead до reset_at.
- **Внутренний budget**: для API-based harness'ов (Claude, Codex через API) — tracking tokens из ответов. Когда budget < 10% — понижаем приоритет harness'а, но не исключаем.

Конфиг: `harness.budget_monthly_usd`, `harness.rate_limit_tpm`. Fallback: если API не даёт usage — heuristic "после N 429 за M минут → assume rate_limit".

### AD-9 — Failure semantics и retry

Матрица:

| Failure | Action | Max attempts |
|---------|--------|--------------|
| 429 / rate_limited | Mark harness, re-dispatch same task | ∞ (other harness) / 1 retry same |
| Timeout | Kill harness, mark degraded, retry on other | 3 |
| Harness crash (non-LLM) | Mark dead, re-dispatch | 3 |
| LLM refusal / bad output | NO retry — flag as `needs_human` | 0 |
| Worktree dirty / lock | Retry after cleanup | 1 |

После max_attempts → создать P1 bug beads, отметить task как failed.

### AD-10 — Integration: sweep и deliver

- `sdp sweep` вызывает `sdp orchestrate submit --batch <task-ids>`
- Оркестратор возвращает stream события: `task_started`, `task_completed`, `task_failed`, `quota_low`
- Sweep/deliver не знают, какой harness выполнил задачу — только результат
- Fallback: если оркестратор недоступен (daemon down) — sweep может работать в legacy single-harness mode

### AD-11 — Security

- Harness CLI могут выполнять произвольный код (это их job)
- Оркестратор sanitize task.prompt перед передачей (размер, absence of secrets patterns via regex)
- Workdir каждой задачи = изолированный git worktree — harness не трогает основной checkout
- Secrets: оркестратор читает из `~/.sdp/secrets.env` (0600); не логирует; passes to harness через env (не через args)

### AD-12 — Observability

- Structured logs (JSON) в `~/.sdp/orchestrator/logs/YYYY-MM-DD.log`
- Prometheus metrics: `sdp_dispatch_total{harness,status}`, `sdp_harness_budget_remaining{harness}`, `sdp_task_duration_seconds{type}`, `sdp_rate_limit_events_total`
- CLI `sdp orchestrate status` — текущее состояние (live dispatches, pool health, budgets)
- Integration с `sdp-session-audit`: оркестратор exporter'ит свои события в тот же формат

---

## 5. Open Questions

1. **Triage cost vs value.** Каждая задача платит маленькой моделью за классификацию. Окупается ли (экономия на основной модели) или overhead? Нужна симуляция на 100 задачах из session-audit.

2. **Capability taxonomy.** `code_gen`, `code_review`, `planning`, `trivial_edit` — достаточно? Или нужны `security_audit`, `architecture_decision`, `test_writing` отдельно?

3. **Cursor harness.** Cursor agent mode — что именно CLI-invocable? Нужен research pass. Если не подходит — выкидываем из pool.

4. **OpenCode reliability.** Предыдущий feedback пользователя: "OpenCode dispatch failure — Sisyphus делегирует в sub-agents, session exits before edits". Нужно ли делать OpenCode-specific harness или пока исключить?

5. **Pi или не Pi.** Реальная нагрузка оркестратора — 5-10 задач в минуту на пике? Если да — Pi справится. Если стабильно 50+ — нужна machine стабильнее. Profiling на M2 Mac до Pi.

6. **Beads contention.** Если оркестратор и harness оба пишут в beads (harness делает `bd close` после работы) — race. Решение: harness **не** пишет в beads, возвращает result, оркестратор закрывает. Принять этот invariant или подумать alternate?

7. **State backup.** `state.db` на Pi — на SD-карту. Износ. Нужен ли периодический sync на облако/NAS?

---

## 6. Dependencies

- **HARD:** [Surface Contract Normalization](2026-04-20-sdp-framework-normalization-design.md) до `F137-02` (registry/discovery) и `F137-03` (command migration). Оркестратор использует `internal/sdpcli` и расширяет existing `cmd/sdp-orchestrate*` через unified surface.
- **SOFT:** beads stable API (`bd claim`, `bd close`, `bd ready --json`)
- **EXTERNAL:** установленные CLI harness'ы (`claude`, `codex`, `opencode`, опционально `cursor-agent`)

## 7. Migration / Implementation Plan (M1-M7)

**M1 — Audit existing** (2-3 дня): детальный разбор `cmd/sdp-dispatch`, `cmd/sdp-harness`, `cmd/sdp-orchestrate`, `cmd/sdp-orchestrate-daemon`, `cmd/sdp-a2a`, `cmd/sdp-control`. Что сохранить, что deprecate. Артефакт: `docs/reference/orchestrator-inventory.md`.

**M2 — Harness interface + Claude impl** (3-4 дня): `internal/harness/`, `ClaudeHarness` через exec `claude` CLI, `MockHarness` для тестов. Unit + integration tests.

**M3 — Dispatch core + single-harness mode** (3-4 дня): `internal/orchestrator/` (TaskQueue, Pool, BudgetAwareCapabilityPolicy), SQLite state, basic daemon `cmd/sdp-orchestrate-daemon`. Sweep может работать в single-harness Claude mode.

**M4 — Multi-harness** (3-4 дня): Codex + OpenCode harness impl. Policy test с 3-pool, rate-limit simulation.

**M5 — Triage layer** (2-3 дня): `internal/triage/`, MiniModelHarness, интеграция в dispatch. A/B test на 50 задачах.

**M6 — SSH transport + Pi profile** (2-3 дня): `HarnessConfig.Transport`, deploy script, systemd unit, telemetry.

**M7 — Observability + docs** (2 дня): Prometheus metrics, `sdp orchestrate status`, README, integration guide для sweep/deliver.

Итого: ≈3 недели calendar.

## 8. Acceptance Criteria

- [ ] `sdp orchestrate daemon start` запускает процесс, отвечающий на `/healthz`
- [ ] `sdp orchestrate submit --task <id>` → задача выполнена одним из harness'ов (любым)
- [ ] Симуляция: kill Claude harness mid-dispatch → задача автоматически переподана на Codex
- [ ] 429 от Claude (симулированный) → pool корректно маркирует, последующие задачи идут в другие harness
- [ ] Triage layer срабатывает: задачи с `trivial_edit` идут в cheapest harness
- [ ] Deploy на Raspberry Pi 4/5: daemon стабилен 24ч, RSS < 150MB
- [ ] Sweep через оркестратор: полный прогон batch 5 tasks без manual intervention
- [ ] Prometheus metrics корректны, `sdp-session-audit` подхватывает события

## 9. Non-Goals

- Замена beads
- Собственная LLM-модель (triage всегда внешняя)
- WebUI (в этом эпике — только CLI + metrics)
- Multi-tenancy (один пользователь / одна команда = один orchestrator)
- Авто-scaling harness fleet
