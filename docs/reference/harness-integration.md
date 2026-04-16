# Harness Integration Reference

> **Scope:** как SDP диспатчит и запускает внешние coding-agent harness'ы. Живое состояние per harness с известными подводными камнями.
> **Policy source:** F127-05 (`docs/plans/2026-04-16-f127-multi-harness-modernization-design.md`).
> **Related:** `cmd/sdp-dispatch/` · `.agents/skills/README.md` · `AGENTS.md`.

## Supported harnesses (April 2026)

| Harness | Binary | Status | Reliability |
|---------|--------|--------|-------------|
| Claude Code | `/opt/homebrew/bin/claude` | Primary | Высокая — все агенты работают |
| Codex CLI | `/opt/homebrew/bin/codex` | Secondary | Высокая для edits; не коммитит (sandbox) |
| OpenCode | `/opt/homebrew/bin/opencode` | Experimental | См. [OpenCode Sisyphus deadlock](#opencode-sisyphus-deadlock) |
| Cursor Agent | `~/.local/bin/agent` | Experimental | Не протестирован в wave 2026-04 |
| Kilo Code | — | Planned | Под roadmap |

**Excluded:** Pi (Inflection). В 2026 Pi — не coding assistant, не включён. См. F127-07.

## Common invocation pattern

SDP dispatch-router (`cmd/sdp-dispatch`) выбирает harness + модель на основе CapabilityProfile и отправляет prompt через нативный CLI. Индивидуальные команды ниже.

### Claude Code

```bash
claude -p "prompt text" --output-format text
```

- Читает `CLAUDE.md` (+ через `@AGENTS.md` подтягивает общий).
- Симлинки в `.claude/agents` ведут на `sdp/prompts/agents` (submodule).
- Sub-agent dispatch — через native `Task` tool.

### Codex CLI

```bash
codex exec --full-auto "prompt text"
```

- Читает `AGENTS.md` (native).
- Edits выполняет надёжно.
- **Ограничение:** sandbox не позволяет `git commit`. После Codex нужен отдельный шаг commit/push (из Claude Code или вручную).

### OpenCode

```bash
opencode run --dir <repo> --agent implementer "prompt text"
```

- Читает `AGENTS.md` + `.agents/skills/` (native).
- См. ниже — **всегда используй `--agent implementer`** для non-interactive dispatch.

### Cursor Agent

```bash
agent -p "prompt text"
```

- Читает `AGENTS.md` + `.cursor/rules/*.mdc`.
- Untested в текущей wave — используй с наблюдением.

## OpenCode Sisyphus deadlock

### Symptom

В non-interactive режиме (`opencode run`) default-агент **Sisyphus** делегирует работу sub-agents в background, а parent-сессия завершается до того, как edits реально записаны. Итог: команда возвращает success, но файлы не изменены.

Впервые зафиксировано в UX-audit 2026-04-05.

### Root cause

Sisyphus сконструирован как lead-orchestrator, ожидающий interactive loop. В `opencode run` event loop закрывается по завершении orchestrator-хода, не дожидаясь sub-agent completion.

### Fix (discoverable)

**Всегда** передавай `--agent implementer` для batch/CI/CLI-dispatch:

```bash
opencode run --dir "$REPO" --agent implementer "$PROMPT"
```

Implementer работает прямо в parent-сессии, без делегирования — edits происходят до возврата управления.

### Do

- Для автоматизированного dispatch (sdp-dispatch, CI) использовать `--agent implementer`.
- Если нужна orchestration-логика — выполнять её на стороне SDP (через `sdp-orchestrate`), а OpenCode запускать только как "worker"-harness.

### Don't

- Не полагаться на default Sisyphus agent в non-interactive режиме.
- Не увеличивать таймауты в надежде "подождать" — ход parent'а уже завершён, ждать нечего.
- Не повторять запуск — он вернёт success без edits.

### Related

- MEMORY record: `feedback_opencode_dispatch.md`.
- SDP dispatch-router должен иметь default-флаг `--agent implementer` для OpenCode harness (TODO, отдельная беда если потребуется).

## Codex: no-commit constraint

Codex sandbox запрещает shell-операции за пределами workspace edit — `git` недоступен. Pattern:

1. Dispatch Codex с prompt'ом только под edits.
2. Возврат в SDP-controller.
3. SDP или Claude Code запускают `git add/commit/push`.

Не просить Codex «commit and push» — он вернёт failure, не отразив сделанные edits.

## Cursor Agent: untested surface

На апрель 2026 Cursor Agent не валидирован в SDP dispatch pipeline. Известно:
- Читает `AGENTS.md` native.
- Имеет agent-mode в IDE и CLI-mode через `agent -p`.
- Sub-agent dispatch через internal agent runtime, семантика отличается от Claude Code Task tool.

Рекомендация: использовать как **secondary validator** для независимой проверки edits, не как primary worker.

## Capability-based routing

`cmd/sdp-dispatch` ведёт `CapabilityProfile`-файлы в `.sdp/dispatch/profiles/*.json`:

- `TestPassRate` per `task-type:language` per harness+model.
- ColdStartStrategy: `capability-heuristic` (default), `round-robin`, `fallback-chain`.
- Profile считается stale после 30 дней (варнинг при диспатче).

Full design: `docs/OPENCODE_HARNESS_GATES_TELEMETRY_SPEC.md`.

## Troubleshooting

| Symptom | Likely cause | Fix |
|---------|-------------|-----|
| `opencode run` returns 0 но нет edits | Sisyphus deadlock | `--agent implementer` |
| Codex returns с «git: command not found» | Sandbox | Не проси Codex commit; делай снаружи |
| Claude Code не видит skill | Симлинки сломаны | `git submodule update --init`; проверь `.agents/skills/` |
| Cursor Agent висит | Wave-2026-04: untested | Fallback на Claude Code |
| Dispatch routes на Pi | Старый конфиг | Удали Pi из profile; Pi не coding agent |

## References

- [F127 design](../plans/2026-04-16-f127-multi-harness-modernization-design.md)
- MEMORY (local): `feedback_opencode_dispatch.md`, `reference_harness_clis.md`, `project_unified_coding_agents.md`
- `cmd/sdp-dispatch/` (code)
- `docs/OPENCODE_HARNESS_GATES_TELEMETRY_SPEC.md`
