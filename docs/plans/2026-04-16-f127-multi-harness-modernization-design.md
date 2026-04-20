# Multi-Harness Agents & Skills Modernization — Design

> **Status:** Design (2026-04-16) · **Owner:** Andrei · **Target feature:** F127
>
> **Numbering note:** черновик изначально использовал F123, но F123 занят `Toolkit Spec Recovery` в `docs/plans/2026-04-13-sdp-toolkit-implementation-plan.md`. Перенумеровано в F127 (следующее свободное после F120…F126).
> **Scope:** `sdp_lab` + публикация в `sdp/` submodule (где релевантно)
> **Parent vision:** `docs/plans/2026-04-13-sdp-toolkit-vision-design.md` + `2026-04-13-sdp-skill-architecture-design.md`

## 1. Why now

На апрель 2026 вокруг SDP сложились три независимых давления:

1. **AGENTS.md стал де-факто стандартом.** Свыше 30 tools (OpenAI Codex CLI, Cursor, Copilot, Windsurf, Amp, Devin, Aider, VS Code, Zed, Warp, Jules, Factory, JetBrains Junie, Gemini CLI, Kilo Code, goose, Phoenix, Semgrep, Ona, RooCode) читают его нативно. Claude Code — единственный заметный holdout и читает `CLAUDE.md`. У нас уже есть оба файла — но они расходятся: `AGENTS.md` канонический (24K), а `CLAUDE.md` дублирует RTK-инструкции из `RTK.md`.
2. **Skills 2.0 и multi-harness skill discovery.** OpenCode и Cursor читают `.agents/skills/` и `.claude/skills/` нативно. Наш текущий `skills/` в корне + симлинк `.claude/skills → sdp/prompts/skills` не покрывают OpenCode/Cursor/Codex.
3. **Формализация multi-agent patterns.** Anthropic опубликовал [пять паттернов координации](https://claude.com/blog/multi-agent-coordination-patterns): Generator-Verifier, Orchestrator-Subagent, Agent Teams, Message Bus, Shared State. У нас есть sub-agent dispatch, но без документированного выбора паттерна. MEMORY-запись фиксирует реальный failure mode: OpenCode Sisyphus делегирует и выходит до edits.
4. **MCP Roadmap 2026** — Server Cards (`.well-known/mcp`), stateless transport, DPoP/WIF auth, audit trails. SDP планирует MCP (`docs/plans/2026-04-13-sdp-mcp-design.md`) — надо сразу заложить эти вещи.

Pi в 2026 — не coding assistant (Inflection свернул consumer-направление, Pi остался "empathy-first companion"). Исключаем его из multi-harness scope.

## 2. Goals / Non-goals

**Goals:**
- Сделать `AGENTS.md` единственным источником правды; `CLAUDE.md` оставить thin override.
- Единая директория skills, читаемая Claude Code, OpenCode, Cursor, Codex; migration без поломки текущих путей.
- Добавить в SKILL.md frontmatter поля `version` и `compatibility` — без жёсткого enforcement на старте.
- Задокументировать выбор multi-agent паттернов (когда Orchestrator-Subagent vs Agent Teams vs Generator-Verifier).
- Зафиксировать OpenCode deadlock fix и сделать его discoverable.
- В MCP-design заложить Server Card и audit trail.
- Убрать Pi из всего, что обещает multi-harness.

**Non-goals:**
- Не переписываем intent-based skill architecture (отдельный плей-бук `2026-04-13-sdp-skill-architecture-design.md`).
- Не трогаем Go-код `cmd/sdp-dispatch` — только доки, конфиги, skill-файлы.
- Не реализуем MCP в этом эпике — только требования к будущему MCP дизайну.
- Не внедряем `.well-known/opencode` (org-wide defaults) — это отдельный research.

## 3. Approach per workstream

### F127-01 · Canonicalize AGENTS.md, thin CLAUDE.md

**Проблема:** `CLAUDE.md:37-169` содержит инлайн RTK-блок (133 строки), дублирующий `RTK.md` и `@RTK.md` в `AGENTS.md:548`. Это расхождение: RTK обновляется в трёх местах.

**Решение:**
- Заменить inline-блок на `@RTK.md` (Claude Code поддерживает `@import`).
- Убедиться, что `AGENTS.md` покрывает весь non-Claude-specific контент `CLAUDE.md` (hard rules, read order).
- Оставить в `CLAUDE.md` только то, что специфично Claude Code (beads transport note, claim rules, `@RTK.md`, pointer на `AGENTS.md`).

**Acceptance:** `CLAUDE.md < 40 строк`, RTK-блок живёт только в `RTK.md`, обе Claude Code и любой AGENTS.md-aware harness видят консистентную инструкцию.

### F127-02 · Multi-harness skills directory

**Проблема:** Skills лежат в `/skills/` (2 файла) и через симлинк `.claude/skills → sdp/prompts/skills` (submodule может быть пустым). OpenCode/Cursor/Codex не видят наших skills. `sdp/prompts/skills/` сейчас вообще не существует — submodule не содержит этой папки.

**Решение:**
- Создать `.agents/skills/` как канонический путь, читаемый OpenCode и Cursor.
- Существующие пути (`skills/`, `.claude/skills/`) остаются симлинками на `.agents/skills/` для обратной совместимости.
- Для каждого реального skill в `/skills/` — переместить в `.agents/skills/` + симлинк в `skills/`.
- Не трогаем `sdp/prompts/skills/` — это отдельный submodule-publish path.

**Acceptance:** `.agents/skills/llm-council.md` и `.agents/skills/strataudit.md` доступны; `skills/*.md` продолжают резолвиться; README объясняет политику.

### F127-03 · SKILL.md frontmatter: version & compatibility

**Проблема:** Текущие skills не имеют `version`, `compatibility`. Невозможно tracking breaking changes, невозможно marketplace-индексация.

**Решение:**
- В каждый skill добавить `version: 1.0.0` (semver) и `compatibility: [claude-code, opencode, cursor, codex]` в YAML frontmatter.
- Документировать в `docs/reference/skill-authoring.md`: обязательные поля (`name`, `description`, `version`), рекомендованные (`compatibility`, `requires_mcp`).
- Добавить lint-правило в `sdp-protocol-check` (warning, не error) на отсутствие `version`.

**Acceptance:** Оба существующих skill имеют валидный frontmatter; authoring guide написан; lint warn работает (реализация lint-правила — отдельная F127-08, см. ниже).

### F127-04 · Multi-agent orchestration patterns — doc

**Проблема:** У нас есть Orchestrator (orchestrate) и sub-agent dispatch, но нет чёткого guide, когда какой паттерн применять. Текущий SDP code в `internal/agentloop` смешивает Orchestrator-Subagent и Shared State без формализации.

**Решение:**
- Написать `docs/reference/multi-agent-patterns.md` — адаптация пяти Anthropic-паттернов к SDP.
- Для каждого паттерна: SDP-пример, когда использовать, anti-patterns, cost estimate.
- Добавить decision tree (3+ independent / no shared state / clear file boundaries → parallel dispatch).
- Ссылки из `AGENTS.md` и `docs/phases/DELIVERY.md`.

**Acceptance:** Doc на 300-500 строк, pointer из AGENTS.md, decision tree.

### F127-05 · OpenCode Sisyphus deadlock — discoverable fix

**Проблема:** MEMORY фиксирует: Sisyphus делегирует sub-agents в background, `opencode run` закрывает сессию до edits. Workaround `--agent implementer`, но нигде в репо это не написано.

**Решение:**
- Добавить в `docs/reference/harness-integration.md` (создать) секцию OpenCode с гайдом по `--agent implementer` и примером запуска.
- В `cmd/sdp-dispatch` (если релевантно) — добавить default-флаг `--agent implementer` для OpenCode harness (только документация + беда; фактическое изменение Go-кода — в отдельной беде, если команда решит).

**Acceptance:** Новичок, запустивший OpenCode dispatch, видит в доке правильный флаг. MEMORY-запись остаётся, но doc становится primary source.

### F127-06 · MCP Server Cards + audit trail in design

**Проблема:** `2026-04-13-sdp-mcp-design.md` не упоминает MCP Server Cards (`.well-known/mcp`), audit trails и enterprise-auth (DPoP/WIF). На апрель 2026 эти вещи — часть MCP Roadmap 2026.

**Решение:**
- Дополнить `2026-04-13-sdp-mcp-design.md` секцией "MCP 2026 alignment": Server Card, stateless Streamable HTTP, audit trail hook, prep для DPoP/WIF.
- Не реализовывать — только design update + ссылки на MCP Roadmap и SEP-1932/1933.

**Acceptance:** Design doc содержит новую секцию; ссылки на MCP Roadmap; checklist "ready for enterprise adoption".

### F127-07 · Remove Pi from multi-harness scope

**Проблема:** MEMORY-файл `project_unified_coding_agents.md` и `docs/` упоминают Pi как потенциальный harness. Pi больше не coding assistant.

**Решение:**
- Найти все упоминания Pi в `docs/` и code comments (`rtk grep -i "^pi\b\|pi harness\|pi cli"`), удалить или пометить "removed 2026-04".
- Обновить MEMORY-запись.

**Acceptance:** `rtk grep -i "pi" docs/` показывает только false-positives (pipeline, api и т. п.).

### F127-08 · Cross-harness skill validation (optional, P3)

**Проблема:** Нет автомата, проверяющего, что skill работает во всех harness'ах (нет hardcoded "Claude Code" упоминаний, валидный frontmatter).

**Решение:**
- Добавить `sdp-protocol-check --lint-skills` — сканер `.agents/skills/*.md`: проверка frontmatter обязательных полей + warning на hardcoded harness-specific фразы.
- Интеграция в pre-commit hook (warning, не block).

**Acceptance:** Работает на обоих существующих skills; CI пример в `.github/workflows/`.

## 4. Rollout order & dependencies

```
F127-01 (AGENTS canonical) ──┬──> F127-03 (SKILL.md frontmatter)
                             │
F127-02 (.agents/skills) ────┴──> F127-08 (lint)
F127-04 (patterns doc)       ──>  linked from AGENTS.md
F127-05 (OpenCode fix)       ──>  standalone
F127-06 (MCP 2026)           ──>  standalone (touches design doc)
F127-07 (Pi cleanup)         ──>  standalone
```

Parallel: F127-01, F127-02, F127-04, F127-05, F127-06, F127-07 независимы.
Sequential: F127-03 после F127-02 (знает, куда писать skills); F127-08 после F127-03 (lint требует frontmatter).

## 5. Artifacts

- Этот design doc.
- Beads epic `F127` + восемь детей (`F127-01…F127-08`).
- Documentation updates: `docs/reference/skill-authoring.md`, `docs/reference/multi-agent-patterns.md`, `docs/reference/harness-integration.md` (все новые).
- `.agents/skills/` directory + симлинки.
- `CLAUDE.md` thin version.
- `2026-04-13-sdp-mcp-design.md` addendum.

## 6. Open questions

- **Q1:** Стоит ли публиковать pattern doc в `sdp/` native directory (public protocol)? Рекомендация: да, в `sdp/docs/patterns/` — это общая вещь, не sdp_lab-specific.
- **Q2:** Сохранить ли `/skills/` как симлинк или удалить (breaking для ссылок в commands.json)? Рекомендация: сохранить симлинк — zero breakage.
- **Q3:** Включать ли Windsurf, Amp, Devin явно в `compatibility`? Рекомендация: нет — достаточно `[claude-code, opencode, cursor, codex]`, остальные читают AGENTS.md автоматически.

## 7. References

- [AGENTS.md official](https://agents.md/)
- [Anthropic — Multi-agent coordination patterns](https://claude.com/blog/multi-agent-coordination-patterns)
- [OpenCode Skills docs](https://opencode.ai/docs/skills/)
- [MCP 2026 Roadmap](https://blog.modelcontextprotocol.io/posts/2026-mcp-roadmap/)
- Внутренние: `docs/plans/2026-04-13-sdp-skill-architecture-design.md`, `docs/plans/2026-04-13-sdp-mcp-design.md`, `docs/plans/2026-04-13-sdp-toolkit-vision-design.md`
