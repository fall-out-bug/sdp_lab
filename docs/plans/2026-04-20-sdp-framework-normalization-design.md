# SDP Framework Normalization — BASELINE Design

**Date:** 2026-04-20
**Status:** BASELINE — draft для углублённой проработки
**Supersedes:** частично объединяет идеи [2026-04-13-sdp-toolkit-vision-design.md](2026-04-13-sdp-toolkit-vision-design.md), [2026-04-13-sdp-mcp-design.md](2026-04-13-sdp-mcp-design.md), [2026-04-13-sdp-skill-architecture-design.md](2026-04-13-sdp-skill-architecture-design.md)
**Owner:** TBD
**Dependencies:** none (этот эпик — фундамент, блокирует sweep и mini-harness)

---

## 1. Контекст и проблема

Репо `sdp_lab` накопил техдолг организации кода и документации. Симптомы:

1. **Дубликат CLI surface.** Существует два параллельных способа вызова SDP-команд:
   - Монолит `cmd/sdp/` (30+ subcommands: scout, architect, metrics, index, bootstrap, dispatch, orchestrate, card, board, doctor, …) — ≈200K LOC
   - Отдельные бинари `cmd/sdp-*/` (23 директории: sdp-dispatch, sdp-harness, sdp-mcp, sdp-orchestrate, sdp-orchestrate-daemon, sdp-a2a, sdp-control 708 LOC, sdp-ci-loop, sdp-doc-sync, sdp-session-audit, …)

   Часть пересекается (`cmd/sdp/cmd_dispatch.go` vs `cmd/sdp-dispatch/main.go`). Владелец правды — не определён. Новые фичи добавляются то туда, то сюда.

2. **Скилы раздуты.** `.agents/skills/` содержит **41 файл**. Vision-доки 2026-04-13 предполагали 5 интентов (`@understand`, `@build`, `@fix`, `@review`, `@operate`) + практики, но фактически в репо живут:
   - 5 целевых интентов (частично): build.md, review.md, operate.md, understand.md
   - Legacy / дубли: bugfix.md+bug-fix.md, fix.md, hotfix.md, feature.md+feature-delivery.md, oneshot.md, debug.md, design.md, architect.md, vision.md, idea.md, ux.md, landscape.md, plan.md+plan-phase.md, eval-phase.md, review-phase.md, deploy.md, ci-triage.md, metrics.md, scout.md, strataudit.md, reality-check.md, verify-workstream.md, llm-council.md (14.5K!), parallel-dispatch.md, test-coverage.md, test-writer.md, git-worktree.md, session-audit.md, smoke-test.md, feature-delivery.md, issue.md, prototype.md, gate.md, research.md, understand.md
   - Многие — stub-файлы 300-400 байт (`vision.md` 363B, `plan.md` 362B, `design.md` 373B) без содержания

3. **MCP server не интегрирован.** `internal/mcp/` готов (server.go 20.5K, tools.go, prompts.go, resources.go, templates/*.tmpl) с тестами. Но `cmd/sdp-mcp/main.go` — 100 строк, не привязан к workflow агентов. Ни Claude Code, ни Codex не ходят через MCP сегодня.

4. **Документация дрейфует.** Vision-доки 2026-04-13 описывают целевое состояние. Реальность: `cmd/sdp` имеет свою эволюцию, часть команд из vision (`sdp spec`, `sdp scout`, `sdp index`, `sdp bootstrap`) существует как файлы, но степень реализации не задокументирована.

5. **Отсутствует единая help/reference система.** `sdp --help` и `sdp <subcommand> --help` непоследовательны. Агенту негде взять "список всех команд + schema" одним вызовом — поэтому subagent'ы тратят токены на `ls cmd/` и чтение main.go.

---

## 2. Цели (North Star)

1. **Единый entrypoint.** `sdp <subcommand>` — единственный способ вызова. `cmd/sdp-*` становятся internal или исчезают.
2. **Единый skill catalog.** `.agents/skills/` — минимальный набор интентов + машинно-читаемый `skills/index.json`.
3. **MCP как second-class interface.** Всё, что делает `sdp <subcommand>`, доступно через MCP tools. Агенты могут работать через CLI или MCP.
4. **Docs lint.** `sdp doc-sync` / `bd doctor` ловят drift между реализацией, vision-доками и skills.
5. **Non-breaking migration.** Существующие `cmd/sdp-*` работают до явной deprecation. Агенты и скрипты не ломаются в момент выкатки.

---

## 3. Архитектурные решения

### AD-1 — Единый CLI entrypoint `sdp`, старые бинари → shim wrappers

`cmd/sdp/main.go` — владелец правды. Все `cmd/sdp-*` становятся thin shim'ами:

```go
// cmd/sdp-dispatch/main.go
package main
import "sdp_dev/internal/sdpcli"
func main() { sdpcli.Run(append([]string{"dispatch"}, os.Args[1:]...)) }
```

Альтернатива (отклонена): удалить `cmd/sdp-*` немедленно. Ломает CI, скрипты, привычки.

Бинари, чья функциональность **не в** `cmd/sdp/` (sdp-harness, sdp-orchestrate-daemon, sdp-mcp, sdp-up, sdp-a2a, sdp-healthcheck, sdp-doc-sync, sdp-ci-loop, sdp-session-audit), **мигрируют в `cmd/sdp/`** как subcommands; их директории остаются shim'ами.

### AD-2 — Command registry с discovery

`internal/sdpcli/registry.go`:

```go
type Command struct {
    Name        string
    Short       string
    Long        string
    RunFunc     func(args []string) int
    MCPExpose   bool     // экспонировать ли как MCP tool
    MCPSchema   json.RawMessage
}

func Register(c Command)
func List() []Command
func Lookup(name string) (Command, bool)
```

Каждый subcommand в `cmd/sdp/cmd_<name>.go` регистрируется через `init()`. Help, `--json`, MCP schema — генерируются из registry. Отсутствие команды в registry = не существует.

### AD-3 — MCP server = автоматический proxy в registry

`cmd/sdp-mcp/main.go` (переименовать позже в `sdp mcp serve`) не пишет tools вручную. Цикл:

```go
for _, c := range sdpcli.List() {
    if !c.MCPExpose { continue }
    mcpserver.RegisterTool(c.Name, c.MCPSchema, func(args map[string]any) (string, error) {
        // exec sdp <name> with args mapped to flags
    })
}
```

Новая `sdp` команда автоматически доступна через MCP. Конфиги клиентов (Claude Code, Cursor, OpenCode) — без изменений.

### AD-4 — Skills: до 7 файлов + index.json

Целевой набор:

| Skill | Назначение | Источники (merge) |
|-------|-----------|-------------------|
| `build.md` | Реализация WS / фичи | build.md, oneshot.md, feature.md, feature-delivery.md, prototype.md |
| `fix.md` | Исправление багов | fix.md, bugfix.md, bug-fix.md, hotfix.md, debug.md, issue.md |
| `review.md` | Code / impact / security review | review.md, review-phase.md, reality-check.md, verify-workstream.md |
| `operate.md` | Деплой, CI-triage, плейнинг | operate.md, deploy.md, ci-triage.md, plan.md, plan-phase.md, gate.md |
| `understand.md` | Исследование кодбейза | understand.md, research.md, scout.md, landscape.md, architect.md, metrics.md, strataudit.md |
| `delivery-loop.md` | Автономный цикл build+review+PR | delivery-loop.md, parallel-dispatch.md |
| `session-audit.md` | Аналитика сессий | session-audit.md |

Остальные (vision.md, idea.md, ux.md, design.md, eval-phase.md, test-coverage.md, test-writer.md, llm-council.md, smoke-test.md, git-worktree.md) — удалить или слить в соответствующий интент.

`.agents/skills/index.json` (generated): `{name, path, description, tags, requires_cli, compatibility}` для каждого скила. Используется MCP prompts и CLI `sdp skills list`.

### AD-5 — Deprecation policy

Каждый удаляемый файл / бинарь проходит:
1. Добавление `DEPRECATED` header с датой и указанием замены
2. Runtime warning в stderr при вызове shim'а
3. 2 спринта grace period (≥14 дней)
4. Удаление с bump minor version + CHANGELOG

Фиксируется в `docs/reference/deprecations.md`.

### AD-6 — Docs-as-code lint

`sdp doc-sync check` (уже есть `cmd/sdp-doc-sync`, 143 строки) расширяется правилами:
- Каждая команда в registry → упомянута в `docs/reference/sdp-cli.md`
- Каждый skill → запись в `skills/index.json`, ссылка из `docs/reference/skills.md`
- Vision-доки `docs/plans/2026-04-13-*` → linkcheck + "Status:" field sync

Встроить в `./scripts/run_go_quality_gates.sh` как non-blocking warning; в `bd preflight` — blocker.

### AD-7 — `sdp help` / `sdp <cmd> --help` унификация

- `sdp help` — дерево: intent (build/fix/review/operate/understand) → subcommand → flags
- `sdp help <cmd>` = `sdp <cmd> --help`
- `--json` flag на `help` выдаёт registry dump (для агентов/MCP)
- Flag conventions (AD-adjacent): `--json` везде, где output — структурированный; `--dry-run` на mutating ops; `--verbose` не `-v` (последнее — version).

### AD-8 — Versioning и compatibility

- `sdp version` → `{cli_version, go_version, registry_hash, mcp_schema_version}`
- Major bump при breaking изменении в registry signatures
- MCP schema имеет отдельный version tag — клиент может проверить совместимость

### AD-9 — Config и state

- Runtime config: `$XDG_CONFIG_HOME/sdp/config.toml` (уже частично реализовано — проверить в `cmd/sdp/helpers.go`)
- Project state: `.sdp/` в корне репо (уже используется — `.sdp/architecture/`, `.sdp/metrics/`, `.sdp/checkpoint.json`)
- Новая схема: `.sdp/config.toml` per-repo overrides

### AD-10 — Testing & CI

- Unit tests для каждого subcommand — в `cmd/sdp/cmd_<name>_test.go` (конвенция уже есть: cmd_architect_test.go, cmd_discover_test.go, cmd_phase_test.go)
- Integration test: `sdp_test.go` прогоняет `sdp help`, `sdp --json help`, MCP handshake
- `./scripts/run_go_quality_gates.sh` не должен замедлиться >20%

---

## 4. Migration Path (M1-M7)

**M1 — Inventory (1-2 дня):** audit 23 `cmd/sdp-*` директорий. Для каждой: зависимости, уникальная функциональность, используется ли в скриптах/CI/agents. Артефакт: `docs/reference/cmd-inventory.md`.

**M2 — Skills merge (2-3 дня):** свести 41 skill → 7 файлов по AD-4. Сгенерировать `skills/index.json`. Обновить все `@skill` ссылки в `.claude/commands/*`, `AGENTS.md`, `CLAUDE.md`, `docs/`.

**M3 — Registry core (2-3 дня):** `internal/sdpcli/registry.go` + перевести 5 самых используемых subcommand'ов (`scout`, `architect`, `metrics`, `dispatch`, `orchestrate`) на новый registry. Остальные пока через switch.

**M4 — Shim generation (1 день):** скрипт `scripts/gen_shims.sh` создаёт `cmd/sdp-<name>/main.go` как forwarder. Добавить deprecation warnings.

**M5 — MCP proxy (2 дня):** `sdp-mcp` автогенерирует tools из registry (AD-3). Тестовый MCP client прогоняет handshake и вызов 3-4 команд.

**M6 — Docs lint (1-2 дня):** `sdp doc-sync check` с правилами AD-6. Встроить в quality gates и `bd preflight`.

**M7 — Cleanup (1 день):** удалить skills-дубли, опустевшие legacy файлы, обновить ROADMAP.md и project-map.md.

Итого: ≈2 недели calendar time для solo executor.

---

## 5. Open Questions

1. **Судьба `cmd/sdp/`.** Он монолит с 30+ subcommand и 200K LOC. Рефакторить на registry — риск сломать существующий behavior. Делать incremental (AD-2) или форкнуть в `internal/sdpcli/` и постепенно переносить?

2. **Shim performance.** Каждый вызов `sdp-dispatch` через shim = double exec (shim → sdp). Приемлемо? Или shim'ы тупо alias через `go:linkname`?

3. **Intent routing vs subcommand tree.** Vision-доки говорят про 5 интентов (@understand, @build, @fix, @review, @operate). Это скилы или CLI-команды? Если оба — как синхронизировать (skill читает output `sdp <intent> <mode>`)?

4. **MCP schema source.** Ручной JSON в registry (verbose) vs cobra reflection (требует перевода на cobra) vs генерация из Go-doc (fragile)?

5. **Skills format evolution.** Текущие `.md` с YAML-frontmatter. Достаточно ли этого для версионирования, или нужен `skill.yaml` manifest + `skill.md` body (как в superpowers)?

6. **Legacy `.claude/commands/`.** Это Claude-специфика. Для Codex/OpenCode — свои `.agents/...`? Или `sdp skills install --harness=claude` генерирует из единого источника?

---

## 6. Dependencies

- **Этот эпик — root** для sweep и mini-harness-orchestrator (оба используют core `sdpcli` libs и `sdp <subcommand>` вызовы).
- Блокирующее условие для downstream: M3 (registry core) + M5 (MCP proxy) должны быть shipped.
- Внешних зависимостей нет.

## 7. Acceptance Criteria

- [ ] `sdp help` показывает ≤ 30 команд, все из registry
- [ ] 7 skill файлов + `index.json`, старые 34 удалены или помечены DEPRECATED
- [ ] MCP server автоматически отдаёт tools из registry; Claude Code может вызвать `sdp scout` через MCP tool
- [ ] `cmd/sdp-*` (все 23) либо удалены, либо shim'ы с deprecation warning
- [ ] `./scripts/run_go_quality_gates.sh` зелёный, регрессий по latency ≤ 20%
- [ ] `docs/reference/sdp-cli.md` и `docs/reference/skills.md` обновлены и проходят `sdp doc-sync check`
- [ ] CHANGELOG.md описывает breaking changes и migration path

## 8. Non-Goals

- Переписывание логики subcommand'ов (только каркас)
- Замена beads
- Новые фичи (scout/architect/metrics остаются как есть функционально)
- Poly-language bindings (python/node wrappers) — отдельный эпик
