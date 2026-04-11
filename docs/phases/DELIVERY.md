# Delivery Phase

## Что это

Delivery — structured execution от spec до задеплоенной фичи. Реализована как Phase FSM в
`internal/agentloop`: пять фаз (Discover → Plan → Build → Review → Eval) с gate на каждом
переходе. Evidence собирается из реальных tool outputs через `EvidenceAccumulator.OnToolResult` —
агент не может self-report прохождение gate.

---

## Вход

Delivery принимает из Discovery:

- `TaskContract` — objective, scope, acceptance criteria, constraints (хранится в
  `.sdp/artifacts/{id}/contract.json`, хеш записан в Beads metadata)
- Beads-карточку feature в статусе `in_progress`, gate `contract-approve` закрыт
- Spec-артефакты: workstream файл, ограничивающий scope для `sdp-guard`

---

## Фазы

| Фаза | Что происходит | Gate на выходе |
|------|---------------|----------------|
| Discover | Читает spec, изучает codebase. Инструменты: `web_search`, `read_file`, `bd_search`. Минимум 200 токенов вывода (MinOutputTokens). | GateEngine: compliance check против contract |
| Plan | Составляет план изменений. Инструменты: `read_file`, `glob`, `bd_create` (создаёт sub-карточки). Модели с большим контекстом (gpt-4.1, claude-opus). | GateEngine: plan-review (compliance) |
| Build | Реализует изменения. Инструменты: `read_file`, `edit_file`, `bash`, `glob`. `edit_file` → `file_modified` в evidence. | GateEngine: compliance; при fail → RecoveryNext: Plan или Build |
| Review | Self-review + чтение diff. Инструменты: `read_file`, `grep`, `bd_comment`. Может вернуть FSM в Build (AllowedNext включает RoleBuild). | GateEngine: review-pass |
| Eval | Запускает тесты и проверки через `bash`. PASS/FAIL в stdout → `quality["test"]` в EvidenceAccumulator. Финальная фаза — AllowedNext пуст, RecoveryNext → Build. | GateEngine: qa-pass |

---

## Gates

**Что такое gate.** Gate — точка принятия решения. Переход между фазами блокируется до тех пор,
пока gate не пройден. Gate fail = стоп. Не advisory, не предупреждение — фаза не переходит дальше.

**GateEngine (`internal/agentloop/gate.go`).** Оборачивает `harness.EvaluateCompliance` с
circuit-breaker timeout (default 5 s). Таймаут — не автоматический pass: возвращает
`Escalated=true` с `GateWarn` нарушением `gate_timeout`. Человек обязан разрешить вручную.

**Filesystem-backed gates (`internal/gate/gate.go`).** Отдельный уровень человеческих gate —
объект `Gate` с полями `Answer`/`Answerer`/`ResolvedAt`. `IsBlocking()` возвращает `true` пока
`ResolvedAt == nil`. `bd gate resolve` закрывает gate в Beads.

**Типы gates в pipeline:**

| Gate ID | Тип | Когда | Критерий |
|---------|-----|-------|----------|
| `contract-approve` | human | После генерации contract (Discovery) | Contract соответствует intent |
| `gate_timeout` | escalated | GateEngine timeout | Требует human review |
| `ci` | automated | После Build/Eval | Все тесты зелёные, lint чистый |
| `staging-approve` | human | Перед staging deploy | Human явно подтвердил |
| `prod-approve` | human | Перед production deploy | Explicit production approval |

**Gate lifecycle в Beads.** Gates — Beads-issue типа `chore` с лейблами `sdp:gate:{type}`.
Создаются как зависимости от parent feature. `bd gate resolve` закрывает.

---

## Evidence

**Что такое evidence.** Structured факты о том, что реально произошло в фазе. Накапливается в
`EvidenceAccumulator` через хук `AfterToolCall` — вызывается после каждого tool execution.
Агент не может объявить gate пройденным — только реальный tool output создаёт evidence.

**Форматы (per-tool extraction, без LLM summarization):**

| Tool | Evidence | Значение |
|------|----------|----------|
| `bash` | PASS в stdout → `quality["test"] = true` | Тест прошёл |
| `bash` | FAIL в stdout → `quality["test"] = false` | Тест упал |
| `edit_file` | `file_modified:<path>` | Артефакт изменён |
| `bd_create` | `card_created:<id>` | Beads-карточка создана |
| любой tool | `tool_error:<name>:<msg>` | Негативная evidence, не игнорируется |

**Reset между фазами.** `EvidenceAccumulator.Reset()` вызывается при каждом `transitionTo` —
следующая фаза начинает с чистым slate.

---

## Инструменты

| Инструмент | Команда | Роль |
|-----------|---------|------|
| Session management | `sdp-harness new/run` | Запуск и продолжение agentloop сессии |
| Feature orchestration | `sdp-orchestrate` | Advance через фазы feature (internal/orchestrate) |
| Scope guard | `sdp-guard` | Pre-commit: diff vs workstream allowlist |
| CI loop | `sdp-ci-loop` | CI polling + autofix цикл |
| Deploy | `sdp deploy staging/prod` | Docker compose deploy (internal/deploy) |
| Gate resolve | `bd gate resolve <id>` | Закрыть human gate в Beads |

---

## Текущий статус (честно)

**Что работает сейчас:**
- `internal/agentloop` Phase FSM полностью реализован: все 5 фаз, GateEngine, EvidenceAccumulator,
  SQLite-backed sessions, completion_signal tool, ContextManager wiring.
- `internal/deploy` — docker compose staging/prod/rollback работает.
- `internal/gate` — Gate struct + IsBlocking() реализованы.
- `sdp-guard`, `sdp-ci-loop` — работают как standalone инструменты.

**Что ещё в разработке (F106 — LiveGateway):**
- LiveGateway не подключён. До завершения F106 execution идёт через `executor` + OmO path
  (`internal/executor` → ExecutorBridge → OmO client), минуя agentloop FSM.
- agentloop запускается через `sdp-harness new/run`, но в production pipeline пока не является
  primary execution path.

**Что не реализовано (не документировать как рабочее):**
- Review агент Momus и QA агент Oracle как отдельные personas.
- Staging smoke tests (автоматические).
- 5-минутный post-deploy health check monitoring.

---

## Связь с Discovery

Delivery принимает от Discovery:

1. Утверждённый `TaskContract` (gate `contract-approve` пройден)
2. Beads feature-карточку с заполненными полями: objective, scope, acceptance criteria
3. Workstream файл — список файлов, которые разрешено менять (используется `sdp-guard`)
4. Spec-артефакты в `.sdp/artifacts/{id}/`

Delivery не начинается, если `contract-approve` gate открыт.

---

## Правила

- **Gate fail = стоп.** Переход в следующую фазу не происходит. Gate не обходится.
- **GateEngine timeout — не pass.** Таймаут эскалирует к человеку (`Escalated=true`).
- **Evidence только из tool outputs.** Self-report агента не засчитывается.
- **Scope ограничен workstream файлом.** `sdp-guard` блокирует pre-commit если diff выходит за границы.
- **Tool errors — negative evidence.** `tool_error:` записывается в evidence, не игнорируется.
- **completion_signal — не в ToolRegistry.** Добавляется `BuildLoopConfig` имплицитно (Fix N6).
