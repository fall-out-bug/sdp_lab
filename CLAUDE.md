# sdp_lab — Claude Code Project Instructions

> **Claude-specific override.** Canonical рабочие правила — в [AGENTS.md](AGENTS.md) (читается всеми harness'ами: Claude Code, Codex CLI, Cursor, OpenCode, Copilot, Zed, Warp и т. д.). Этот файл держи thin: только то, что специфично Claude Code поверх `AGENTS.md`.
>
> Policy: [docs/plans/2026-04-16-f127-multi-harness-modernization-design.md](docs/plans/2026-04-16-f127-multi-harness-modernization-design.md) (F127-01).

## Read Order (cold start, Claude Code)

1. **[AGENTS.md](AGENTS.md)** — операторные правила, workflow, команды, политика редактирования (читают все harness'ы)
2. **[docs/reference/project-map.md](docs/reference/project-map.md)** — canonical SOT split, входные точки
3. **[docs/MULTI-REPO-WORKFLOW.md](docs/MULTI-REPO-WORKFLOW.md)** — когда задача трогает `sdp/` submodule
4. **[docs/roadmap/ROADMAP.md](docs/roadmap/ROADMAP.md)** — текущее продуктовое направление

## Claude-Specific Hard Rules

Всё общее (beads, sessions, quality gates, repo topology) — в `AGENTS.md`. Здесь только то, что актуально только для Claude Code:

- **Issue tracking — только beads (`bd`).** `TodoWrite` запрещён в этом репо. SessionStart hook и `AGENTS.md` дают полный референс.
- **Claim атомарно:** `bd update <id> --claim` (не `--status in_progress`).
- **Session close:** `scripts/beads_transport.sh export` → `git push`. Работа не закончена, пока не запушена.
- **Submodule init:** клонируй с `--recurse-submodules` или запусти `git submodule update --init`. Иначе симлинки `.claude/agents`, `.claude/hooks`, пути `sdp/docs/*` ломаются.

## Quality Gates (перед push)

```bash
./scripts/run_go_quality_gates.sh     # build + test + vet (контейнер)
SDP_GO_QUALITY_MODE=host ./scripts/run_go_quality_gates.sh   # fallback без Docker
```

## Token-Optimized Shell (RTK)

Все shell-команды префиксируй `rtk`. Полный референс команд импортируется ниже:

@RTK.md
