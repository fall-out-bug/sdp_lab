# SDP Skills — Multi-Harness Skill Directory

Канонический путь для SDP-skills, читаемый всеми основными agent-harness'ами.

## Почему `.agents/skills/`

На апрель 2026 `.agents/skills/` — единственная директория, которую **нативно читают**:

| Harness | Путь, который harness сканирует |
|---|---|
| Claude Code | `.claude/skills/` (у нас симлинк → `.agents/skills/`) |
| OpenCode | `.agents/skills/` (native) + `.claude/skills/` (fallback) |
| Cursor | `.agents/skills/` (native) + `.claude/skills/` (fallback) |
| Codex CLI | читает через AGENTS.md + explicit path |

См. [OpenCode Skills docs](https://opencode.ai/docs/skills/).

## Layout

```
sdp_lab/
├── .agents/skills/           ← canonical source (здесь живут реальные файлы)
│   ├── README.md             ← этот файл
│   ├── llm-council.md
│   └── strataudit.md
├── skills/                   ← compat симлинки на .agents/skills/*.md
│   ├── llm-council.md  →  ../.agents/skills/llm-council.md
│   └── strataudit.md   →  ../.agents/skills/strataudit.md
└── .claude/skills → ../.agents/skills   (весь каталог симлинкой)
```

Старые пути (`skills/*.md` и `.claude/skills/*.md`) продолжают резолвиться — `.claude/commands.json` и ссылки в `docs/` не ломаются.

## SKILL.md frontmatter (ожидается)

YAML frontmatter обязательные поля:

```yaml
---
name: short-kebab-name
description: one-line что делает skill (60-120 символов)
version: 1.0.0                # semver
compatibility:                 # optional but recommended
  - claude-code
  - opencode
  - cursor
  - codex
---
```

Полный гайд по авторингу: `docs/reference/skill-authoring.md` (F127-03).

## Почему не `sdp/prompts/skills/`

`sdp/prompts/skills/` — submodule-publish путь для artifacts, которые релизятся в публичный `sdp` protocol repo. Внутренние build/lab skills лежат в `.agents/skills/`. Публикация — отдельным PR в submodule (см. `docs/MULTI-REPO-WORKFLOW.md`).

## References

- [F127 design](../../docs/plans/2026-04-16-f127-multi-harness-modernization-design.md)
- [OpenCode Skills](https://opencode.ai/docs/skills/)
- [AGENTS.md spec](https://agents.md/)
