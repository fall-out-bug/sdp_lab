# Skill Authoring — SDP Multi-Harness

> **Audience:** Authors создающие новый SDP skill.
> **Canonical location:** `.agents/skills/<name>.md` (multi-harness; см. `.agents/skills/README.md`).
> **Policy source:** F127-03 (`docs/plans/2026-04-16-f127-multi-harness-modernization-design.md`).

## Why a policy

SDP-skills должны одинаково работать во всех основных harness'ах (Claude Code, OpenCode, Cursor, Codex) без переписывания. Единый формат делает skill:
- индексируемым (marketplace, lint, search);
- версионируемым (semver → breaking changes видны);
- переносимым между harness'ами без модификаций.

## File location

**Канон:** `.agents/skills/<skill-name>.md`.

**Нельзя** класть реальные файлы в:
- `skills/` (старый корневой каталог — зарезервирован под compat-симлинки);
- `.claude/skills/` (симлинк на `.agents/skills/`);
- `sdp/prompts/skills/` (submodule-publish путь — только для публикуемых артефактов, проходит через отдельный PR в `sdp` repo).

Имя файла — `<kebab-case>.md`, совпадает с `name:` во frontmatter.

## YAML Frontmatter — required

```yaml
---
name: short-kebab-name
description: одно предложение, 60-120 символов, что делает skill
version: 1.0.0
---
```

| Поле | Обязательно | Правило |
|------|-------------|---------|
| `name` | да | kebab-case, совпадает с именем файла без расширения. Никаких пробелов, префиксов вроде `sdp-`. |
| `description` | да | Одно предложение, 60-120 символов. Начинается с глагола или существительного, не «This is a skill that…». |
| `version` | да | Semver. Стартуй с `1.0.0`. Breaking changes → bump major. Non-breaking content updates → bump patch/minor. |

## YAML Frontmatter — recommended

```yaml
compatibility:
  - claude-code
  - opencode
  - cursor
  - codex
requires_mcp: []           # список MCP-серверов если skill ожидает MCP-tools
requires_cli: []           # список CLI-бинарей на PATH (sdp, bd, gh, …)
tags: [discovery, review]  # свободные теги для поиска
```

| Поле | Когда указывать |
|------|-----------------|
| `compatibility` | Всегда, кроме явно Claude-only skills. Список harness'ов, где skill протестирован. |
| `requires_mcp` | Если skill ожидает конкретный MCP-сервер (например, `beads`, `claude-api`). Пустой массив = нет требований. |
| `requires_cli` | CLI-бинари, без которых skill не запустится (пример: `[sdp, bd]`). |
| `tags` | Для lint/search/marketplace индексации. |

## Body structure (рекомендуемый)

```markdown
---
name: my-skill
description: …
version: 1.0.0
compatibility: [claude-code, opencode, cursor, codex]
---

# My Skill — <human-readable title>

## Purpose
1-3 параграфа. Что делает, какой outcome.

## Use When
- bullet-list ситуаций, когда применять
- и когда НЕ применять (anti-patterns)

## Inputs / Outputs
Что skill ожидает на входе (context, файлы, args), что возвращает.

## Process
Шаги выполнения. Если >5 шагов — разбей на подсекции.

## References
- связанные skills
- design docs (docs/plans/YYYY-MM-DD-*.md)
- внешние источники
```

Не дублируй общие правила (beads rules, git workflow) — ссылкой на `AGENTS.md`.

## Length limit

По [ADR-007 `docs/decisions/007-skill-length-limit.md`](../decisions/007-skill-length-limit.md) целевая длина — ≤100 строк. Исключения (`llm-council.md`, ~330 строк) допустимы, если skill реально требует таблиц/протоколов. Если превысил — проверь, можно ли вынести детали в отдельный reference doc.

## Harness-neutral прозе

- **Не пиши** «In Claude Code, use the Task tool to…» — вместо этого «To dispatch a sub-agent, use your harness's native sub-agent mechanism (Task tool в Claude Code, `@agent` в OpenCode, и т. д.).».
- **Не хардкодь** команды harness'а без обёртки «если этот harness применим».
- **MCP-tools** указывай через `requires_mcp`, а не через «you need the Beads MCP server» в body.

## Versioning

- Stable release skills: `1.x.y+`.
- Experimental / draft: `0.x.y`.
- Breaking change = новый major. Документируй breaking в body в `## Changelog` секции или вынеси в git history.
- Для marketplace-подобного tooling (claudemarketplaces.com, будущий SDP registry) важны exactly semver-совместимые теги.

## Example — minimal valid skill

```markdown
---
name: repo-scout
description: 30-second project card for unknown repo — file counts, languages, recent activity, primary build system.
version: 1.2.0
compatibility: [claude-code, opencode, cursor, codex]
requires_cli: [sdp]
---

# Repo Scout

## Purpose
Quickly orient in an unknown codebase before deeper investigation.

## Use When
- Cold-start on a repo you have never seen.
- Comparing two repos at a glance.

## Process
Run `sdp scout .`; read `scout.json`; summarize for the user.

## References
- `docs/plans/2026-04-13-sdp-scout-design.md`
```

## Validation

После F127-08 `sdp-protocol-check --lint-skills` будет сканировать `.agents/skills/*.md`:
- **error:** отсутствует `name`, `description` или `version`.
- **warning:** отсутствует `compatibility`; hardcoded harness-specific фразы; kebab-case не совпадает с filename.

До F127-08 — ручная проверка.

## References

- [F127 design](../plans/2026-04-16-f127-multi-harness-modernization-design.md)
- [.agents/skills/README.md](../../.agents/skills/README.md)
- [ADR-007: Skill Length Limit](../decisions/007-skill-length-limit.md)
- [OpenCode Skills docs](https://opencode.ai/docs/skills/)
- [AGENTS.md spec](https://agents.md/)
