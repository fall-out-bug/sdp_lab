# F129: Autonomous Operations & Regression Hardening — Design

**Date:** 2026-04-16
**Status:** Proposed
**Owner:** Андрей
**Feature:** F129
**Related:** F106 (agentloop), F124 (Toolkit Bootstrap), F125 (Toolkit UX), F126 (Toolkit MCP), F127 (Multi-harness)

---

## Контекст

После второго аудита (2026-04-16) выявлены 5 системных проблем, снижающих автономность SDP:

1. **Низкое качество оркестрации** — работа требует подталкивания пользователем; pull-based FSM требует явных `--advance` на каждом gate.
2. **Сабагенты не по умолчанию** — исследования и review выполняются в основном контексте; чистый контекст (fresh subagent) даёт более сфокусированный результат.
3. **Gap superpowers → SDP** — в `~/.claude/plugins/.../superpowers/skills/` есть ключевые паттерны (`dispatching-parallel-agents`, `using-git-worktrees`, `verification-before-completion`), отсутствующие в `.agents/skills/`.
4. **Документация не обновляется при работе** — рабочие артефакты (workstream status, INDEX, ROADMAP) и пользовательская документация (CHANGELOG, ссылки, drift) дрейфуют.
5. **Регресс обнаруживается случайно** — coverage не enforced, `tests/architect/*` не в CI, флаки тихо мигают, snapshot-тесты CLI отсутствуют.

---

## Принципы дизайна

- **Распределение, а не дублирование.** Часть работы логически принадлежит существующим лейнам (F106 — оркестрация; F124 — bootstrap/hooks; F125 — skill UX). Не создаём в F129 то, что должно жить в профильном эпике.
- **Сразу через pattern, не через скрипты.** Новые skills (`parallel-dispatch`, `git-worktree`, `verify-before-completion`) — harness-neutral, ≤100 строк, проходят `sdp-protocol-check --lint-skills`.
- **Регресс как gate, не отчёт.** Coverage threshold, architect-тесты, snapshot CLI — это блокирующие gates, не advisory строки в логе.
- **YAGNI для mutation testing.** Mutation/flaky advisory — non-blocking на первом шаге; только если infra реально работает — промоутим в блокирующее.

---

## Архитектурные решения

### AD-1: Распределение работы между лейнами

| Проблема | Workstream | Лейн | Обоснование |
|----------|-----------|------|-------------|
| 1. Оркестрация требует подпинывания | `00-106-07 autonomous-loop-flag` | **F106** | Autonomous execution — это достройка agentloop/pull-FSM, ядро F106. Не дублируем в F129. |
| 2. Сабагенты не default | `00-129-01 subagent-default-policy` | **F129** | Политика использования harness-механизмов, не привязана к agentloop. |
| 2. Parallel dispatch skill | `00-129-02 parallel-dispatch-skill` | **F129** | SDP-аналог superpowers skill, harness-neutral. |
| 3. git-worktree skill | `00-129-03 git-worktree-skill` | **F129** | SDP skill для изолированных worktree на feature-branches. |
| 3. verify-before-completion | `00-125-05 review-readiness-mode` | **F125** | Семантика `@review --mode readiness` = verify-before-completion; существующий intent uuh. |
| 4. Post-bd-close hook | `00-124-05 post-close-sync-hook` | **F124** | Hooks — это Bootstrap-lane artifact (setup policies/hooks/git-config). |
| 4. sdp-doc-sync --mode fix | `00-129-04 docsync-fix-mode` | **F129** | Расширение существующего инструмента `internal/docsync`. |
| 4. CHANGELOG auto-gen | `00-129-05 changelog-autogen` | **F129** | Novel CI-integration, не принадлежит ни одному существующему эпику. |
| 4. Markdown link auto-fix | `00-129-06 markdown-link-fix` | **F129** | Расширение `internal/docsync.checkMarkdownLinks` — consolidated с WS-04. |
| 5. Coverage enforcement gate | `00-129-07 coverage-gate` | **F129** | CI-level quality gate, новый. |
| 5. Architect tests в CI | `00-129-08 architect-tests-ci` | **F129** | Наблюдательный gap (tests/architect/* не в CI). |
| 5. CLI snapshot tests | `00-129-09 cli-snapshot-tests` | **F129** | Новая категория тестирования. |
| 5. Flaky/mutation advisory | `00-129-10 flaky-advisory` | **F129** | Non-blocking на Phase 1. |

**Downstream F126 (MCP):** Когда F126 будет реализован, новые skills (`parallel-dispatch`, `git-worktree`) должны быть экспонированы как MCP tools. Добавляется `00-126-05 expose-f129-skills` как downstream dep.

### AD-2: Autonomous loop (F106-07) — spec

`sdp-orchestrate --autonomous` = batch-mode `--next-action` + `--advance` в цикле:

```
while true:
  action = sdp-orchestrate --next-action
  if action == DONE: break
  if action == HUMAN_GATE:
    if --accept-gates=<list> и gate в allowlist: advance
    else: exit 3 (human required)
  if action ∈ {build,review,qa}: execute sub-agent → parse evidence → advance
  if N > MAX_ITERATIONS or same action M раз: exit 4 (loop-stuck)
```

**Safety:** MAX_ITERATIONS=50, SAME_ACTION_LIMIT=3, `--dry-run` обязателен перед первым запуском.

### AD-3: Subagent default (F129-01) — политика

Обновить `AGENTS.md` + shared skill preamble:

> Для любой задачи, которая требует исследования ≥3 файлов или включает ≥2 независимых под-задач — **дефолт**: делегировать в subagent (Task tool в Claude Code, `@agent <role>` в OpenCode, etc.). Чистый контекст = более фокусированный результат.

**Не навязываем** для однократных edits или тривиальных lookups. `parallel-dispatch` skill (WS-02) даёт decision-tree.

### AD-4: docsync --mode fix (F129-04)

`internal/docsync` уже имеет `UpdateChangelog()` и `checkMarkdownLinks()`. Расширение:

- `--mode fix` → применяет авто-исправления для простых случаев:
  - Relative links переиндексируются после перемещения файлов (используя git log --follow).
  - Trailing-slash нормализация.
  - Code fence language tag для `bash`/`go`/`yaml` inference.
- Сложные случаи (broken anchor, missing target) — остаются в `--mode check` с exit 2.

### AD-5: Coverage gate (F129-07)

В CI `.github/workflows/ci.yml` добавить job `coverage-gate`:

```yaml
- run: go test -coverprofile=cov.out ./...
- run: go tool cover -func=cov.out | grep total | awk '{print $3}' > cov.pct
- run: |
    BASE=$(git show origin/main:cov.pct 2>/dev/null || echo 70.0)
    CUR=$(cat cov.pct | tr -d '%')
    python3 -c "exit(0 if float('$CUR') >= float('$BASE') - 2.0 else 1)"
```

Threshold: не более 2 pp падения относительно base. `cov.pct` коммитится в `.sdp/metrics/coverage.txt`.

---

## Skills — новые артефакты

### `.agents/skills/parallel-dispatch.md`

Harness-neutral pattern для независимых подзадач. Ссылается на superpowers эталон. ≤80 строк. `requires_cli: []`. Tags: `[orchestration, efficiency]`.

### `.agents/skills/git-worktree.md`

Safety-first worktree setup для параллельных feature branches. Включает cleanup cadence, conflict prevention. ≤80 строк. `requires_cli: [git]`.

### `.agents/skills/verify-before-completion.md` (опц.)

Если F125-05 реализуется как чистый intent-mode — самостоятельный skill не нужен (semantics живёт в `@review --mode readiness`). Иначе — standalone skill. **Решение:** делаем intent-mode, не skill.

---

## DAG

```
F129-01 ──┐
F129-02 ──┤
F129-03 ──┼─> F125-05 (review readiness использует verify-паттерн из F129)
F129-04 ──┤
F129-06 ──┘
F129-05 (independent)
F129-07 ──┐
F129-08 ──┤
F129-09 ──┼─> CI green
F129-10 ──┘

F106-07 (autonomous) — независим от F129; но autonomous loop без subagent default (F129-01)
  = бесполезен, потому F106-07 depends on F129-01.

F124-05 (post-close hook) — независим.

F126-05 (MCP exposure) — depends on F129-02, F129-03 (skills должны существовать перед exposure).
```

---

## Phases / Order

**Phase 1 (неделя 1):** F129-01 (политика, документ), F129-02 + F129-03 (skills), F124-05 (hook).
**Phase 2 (неделя 2):** F125-05 (@review readiness), F129-04 + F129-06 (docsync fix), F129-05 (changelog).
**Phase 3 (неделя 3):** F129-07 (coverage), F129-08 (architect CI), F129-09 (snapshots), F129-10 (flaky advisory).
**Phase 4 (неделя 4):** F106-07 (autonomous loop) — зависит от стабильной Phase 1-3.

**F126-05** — deferred до начала F126 implementation (не P1).

---

## Metrics / Success

- **Автономность:** `@feature X` без вмешательства пользователя проходит от discovery до PR open в 80%+ случаев (baseline: ~40%).
- **Subagent usage:** ≥60% review/research задач делегированы в fresh subagent (baseline: ~15%).
- **Doc drift:** `sdp-doc-sync --mode check` = 0 findings на main (baseline: ~40 warnings).
- **Regression detection:** coverage-gate + architect-CI ловят ≥90% регрессий до merge (baseline: ~50%, случайные).

---

## Out of Scope

- **Mutation testing infra** (инструмент, reporting) — только advisory строка в summary, без blocking.
- **Flaky quarantine system** — только detection + retry, без автоматического skip/quarantine.
- **Documentation generation from code** (docusaurus/mkdocs) — сохраняем текущий markdown workflow.
- **MCP-level exposure (F126-05)** — only stub, implementation с F126.

---

## References

- [superpowers skills (source of gaps)](plugin:superpowers:skills)
- [Toolkit Vision](2026-04-13-sdp-toolkit-vision-design.md) — F120-F126 lane
- [agentloop Integration](2026-04-11-agentloop-integration-spec.md) — F106 basis для autonomous loop
- [MCP Design](2026-04-13-sdp-mcp-design.md) — F126 для downstream exposure
- [F127 Phase 2 design](2026-04-16-f127-multi-harness-modernization-design.md) — skill lint как инфра
