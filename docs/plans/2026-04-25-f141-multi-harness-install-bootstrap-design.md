# Multi-Harness Install Bootstrap & Adapter Parity — Design

> **Status:** Design (2026-04-25) · **Owner:** Andrei · **Target feature:** F141
>
> **Numbering note:** F127–F140 заняты (см. `docs/plans/` + beads). F141 — следующий свободный.
> **Scope:** `sdp_lab` (генератор и манифест), публикация в downstream repos через `sdp install`.
> **Parent design:** [2026-04-16-f127-multi-harness-modernization-design.md](2026-04-16-f127-multi-harness-modernization-design.md) — F127 закрыт по факту через серию F128-* задач (skill consolidation, AGENTS.md cleanup), но один из его исходных goals — equal discoverability across harnesses — остался незавершённым на уровне *consumer-repo onboarding*.

## 1. Why now

1. **Consumer repos копируют SDP кусками.** Агенты в downstream-проектах (Claude Code, OpenCode, Codex, Cursor) не имеют one-shot способа поставить SDP. Они вытаскивают отдельные skill-файлы, копируют команды без зависимостей, теряют hooks. Результат — каждый репозиторий получает inconsistent subset SDP.
2. **Существует proven UX-эталон.** `oh-my-openagent` (curl|bash bootstrap + структурированная установка `.opencode/`) показал, что разработчики ждут именно такой опыт: "одна команда — полный комплект".
3. **F128 убрал submodule, но drift вернётся вручную.** После F128-01 (move sdp/ submodule → native) каноническим стал `.agents/skills/`, симлинки в `.claude/skills/` и т.д. Это работает в `sdp_lab`, но при копировании в чужой репо синхронизация рвётся — нет инструмента, который перегенерирует адаптеры из канонического дерева.
4. **Паритет сейчас держится на дисциплине документации.** [AGENTS.md](../../AGENTS.md) описывает skill discovery для четырёх harness'ов, но пути конфигурируются вручную. Любое изменение списка skills/commands требует синхронных правок в четырёх деревьях.

## 2. Goals / Non-goals

**Goals:**
- One-shot install в любом downstream-репо: `curl -fsSL https://sdp.dev/install.sh | bash` или `sdp init --harness=all` — drops полный комплект адаптеров для Claude Code / OpenCode / Codex / Cursor.
- Single source of truth: `sdp.manifest.yaml` — declarative inventory (skills, commands, agents, hooks, MCP servers, версии).
- Adapter generator (`sdp generate-adapters`): из манифеста делает `.claude/`, `.opencode/`, `.codex/`, `.cursor/rules/` — никакого ручного дублирования.
- `sdp doctor` как drift-gate (CI + pre-commit): любой адаптер, разошедшийся с манифестом, фейлит проверку.
- Документированная **parity matrix** (команды × harness'ы) — readers видят, что именно поставится и где будут различия (если они принципиальны).
- Re-entrant install: повторный запуск в существующем репо обновляет адаптеры без ручной чистки.

**Non-goals:**
- Не переписываем skill content или командную поверхность — это про *упаковку и доставку*, не про дизайн команд.
- Не делаем package registry / versioning marketplace (отдельная работа, потенциально F142+).
- Не трогаем MCP server-side bootstrap — этим занимается отдельный design `2026-04-13-sdp-mcp-design.md`.
- Не покрываем неcoding harness'ы (Pi и аналоги) — non-goal с F127.
- Не вводим Windows-specific код в первой итерации; macOS + Linux + WSL — целевая поверхность.

## 3. Approach per workstream

### F141-01 · Manifest schema (`sdp.manifest.yaml`)

**Проблема:** Сейчас inventory SDP размазан по `.agents/skills/`, `prompts/skills/`, `cmd/sdp/*`, `internal/registry/*`, плюс упоминания в [AGENTS.md](../../AGENTS.md) и [docs/reference/project-map.md](../reference/project-map.md). Нет единого declarative источника.

**Решение:**
- JSON-Schema-валидируемый `sdp.manifest.yaml` в корне `sdp_lab`. Поля:
  - `version: 1.0` (semver манифеста), `sdp_version: <git-tag>` (версия SDP, которую манифест описывает).
  - `skills: [{ name, path, harnesses: [claude-code|opencode|codex|cursor], version, compatibility }]`.
  - `commands: [{ name, type: skill|cli, dispatch: {claude-code: ..., opencode: ..., codex: ..., cursor: ...} }]`.
  - `agents: [{ name, role, harnesses, system_prompt_path }]`.
  - `hooks: [{ event, script, harnesses }]`.
  - `mcp_servers: [{ name, url, scopes, optional: bool }]`.
- Generator берёт манифест как single source — перестать читать filesystem напрямую.
- Lint: каждый файл, упомянутый в манифесте, должен существовать; обратное (orphan files) — warning.

**Acceptance:** `sdp.manifest.yaml` существует, валидируется JSON-Schema, покрывает все текущие skills/commands/agents/hooks; `sdp manifest validate` зелёный.

### F141-02 · Adapter generator (`sdp generate-adapters`)

**Проблема:** Per-harness деревья (`.claude/commands/*`, `.opencode/*.toml`, `.codex/*`, `.cursor/rules/*.mdc`) сейчас пишутся руками. Любое добавление команды требует 4 правок.

**Решение:**
- Go-команда `cmd/sdp/cmd_generate_adapters.go`. Читает манифест, рендерит per-harness адаптеры:
  - **Claude Code:** `.claude/commands/<name>.md` (frontmatter + body), `.claude/agents/<name>.md`, `.claude/hooks/*.sh`, `.claude/settings.json` (permissions/MCP).
  - **OpenCode:** `.opencode/agent/<name>.json`, `.opencode/skill/<name>.md`, `opencode.toml`.
  - **Codex CLI:** `.codex/skills/<name>.md`, `~/.codex/config.toml`-fragment в `.codex/codex.toml.fragment`.
  - **Cursor:** `.cursor/rules/<name>.mdc`, `.cursor/mcp.json`.
- Templates лежат в `internal/adapters/templates/<harness>/`.
- Idempotent: повторный запуск даёт identical output (важно для diff-based gating).
- Mode: `--check` (CI/pre-commit, fail on diff) и `--write` (default).

**Acceptance:** `sdp generate-adapters --check` зелёный после ручной правки манифеста + регенерации; `git diff` после `--write` показывает только ожидаемые изменения; unit-тесты на golden-output для каждого harness.

### F141-03 · One-shot bootstrap installer

**Проблема:** Нет single command, который ставит SDP в чужой репо. Текущий `sdp init` skill — internal, не покрывает downstream onboarding.

**Решение:**
- Bootstrap-скрипт `scripts/install.sh` (публикуется на release как `https://sdp.dev/install.sh` или GitHub Pages). Делает:
  1. Detect platform (`uname`), скачивает соответствующий `sdp` бинарь из GitHub Releases.
  2. Detect harness'ы в текущем репо (наличие `.claude/`, `.opencode/`, `.codex/`, `.cursor/`) → если none, спрашивает или ставит все.
  3. `sdp init --harness=auto` — копирует манифест-template, прогоняет `sdp generate-adapters --write`, создаёт `AGENTS.md` (если отсутствует) и `CLAUDE.md` thin override.
  4. Записывает `sdp.lock` — pin версии SDP, чтобы `sdp upgrade` был осознанным.
- `sdp init --harness=claude-code,opencode` для выборочной установки.
- `sdp init --update` — re-run без перетирания user-modifications (по hash-based merge: rendered files под `.sdp/generated/` + thin pointer-файлы в харнесс-деревьях).

**Acceptance:** Чистый репо → `curl ... | bash` → работают `/feature`, `/build`, `/deploy` в Claude Code и `sdp <skill>` в OpenCode/Codex/Cursor с одинаковым контрактом.

### F141-04 · `sdp doctor` drift gate

**Проблема:** Если адаптер отредактирован руками без обновления манифеста, drift не виден до runtime.

**Решение:**
- `sdp doctor` (расширение существующего `sdp` CLI):
  - Проверяет `sdp generate-adapters --check` → diff = error.
  - Проверяет существование путей из манифеста.
  - Проверяет, что в `.claude/commands/`, `.opencode/`, `.codex/`, `.cursor/rules/` нет orphan-файлов, не упомянутых в манифесте.
  - Проверяет совпадение версий: `sdp.lock` ↔ `sdp --version`.
- CI workflow: `.github/workflows/sdp-doctor.yml` — на каждый PR.
- Pre-commit hook: `scripts/hooks/sdp-doctor-precommit.sh`, опционально устанавливается через `sdp init`.

**Acceptance:** Ручная правка `.claude/commands/build.md` без правки манифеста → `sdp doctor` fails в CI; правка манифеста + `sdp generate-adapters --write` → green.

### F141-05 · Parity matrix doc

**Проблема:** Сейчас единственный способ узнать, какие команды покрыты в каком harness'е — читать `AGENTS.md` + `CLAUDE.md` + `.opencode/` руками.

**Решение:**
- Auto-generated `docs/reference/harness-parity-matrix.md` из манифеста: таблица команда × harness, явное обозначение "intentional gap" vs "missing" (через поле `manifest.commands[].parity_notes`).
- Раздел в `AGENTS.md` со ссылкой и обновлённой read-order.
- Если drift — `sdp doctor` фейлит на расхождении matrix ↔ manifest.

**Acceptance:** Matrix существует, генерируется из манифеста, ссылка добавлена в `AGENTS.md`.

### F141-06 · Migration of existing adapters to generated

**Проблема:** Текущие `.claude/commands/*`, `.opencode/*` и т.д. написаны руками и могут содержать дрейф друг от друга.

**Решение:**
- Inventory pass: для каждого существующего адаптера — найти эквивалент в других harness'ах, зафиксировать diff в `docs/plans/2026-04-25-f141-migration-audit.md`.
- Прописать всё в манифест.
- Перегенерировать → diff vs текущим деревом → resolve расхождений (либо манифест уточняем, либо адаптер фиксим).
- Удалить ручные адаптеры, заменить на generated.

**Acceptance:** `sdp doctor --strict` зелёный на main; никакой `.claude/`, `.opencode/`, `.codex/`, `.cursor/` файл не отредактирован вручную после миграции.

### F141-07 · Downstream consumer recipes & README

**Проблема:** Даже после готового installer-а downstream разработчики не знают, *что* они получают и как обновлять.

**Решение:**
- README в корне `sdp_lab` (или `sdp/` public submodule) с секциями:
  - "Install in 30 seconds" (one-liner curl).
  - "What you get" (короткая parity matrix snippet).
  - "Update / pin version" (`sdp upgrade`, `sdp.lock`).
  - "Selective install" (`--harness=...`).
  - "Customize without forking" (per-repo overlay в `.sdp/overrides/`).
- Quickstart guide `docs/runbooks/onboarding-downstream-repo.md` — пошаговый сценарий для агента, который ставит SDP в чужой репо.

**Acceptance:** README содержит one-liner и parity matrix; runbook на ≤ 100 строк проводит downstream агента от чистого репо до работающего `/build`.

## 4. Risks & open questions

| Risk | Mitigation |
|---|---|
| Symlinks vs copies на Windows/CI | Стартовать с copies (всегда работают), оставить symlink-mode как opt-in `--mode=symlinks`. |
| Manifest становится монолитом и тяжело ревьюится | Поддержать `imports:` — манифест может ссылаться на partials в `.sdp/manifest/*.yaml`. |
| Drift между версиями downstream и upstream | `sdp.lock` + `sdp doctor` ловят несовместимые манифесты. `sdp upgrade --plan` показывает diff перед обновлением. |
| Bootstrap-скрипт ломается при network failure | Скрипт fail-safe: проверяет существующий бинарь, поддерживает offline install через `--from=<path>`. |
| Per-harness конфиг-форматы эволюционируют | Templates изолированы в `internal/adapters/templates/<harness>/`; миграция шаблона — отдельный PR без затрагивания манифеста. |

**Open questions:**
- Где хостить `install.sh`? GitHub Releases + `https://sdp.dev/` (требует домена), или только `https://raw.githubusercontent.com/...` на старте?
- Один манифест или два уровня (public-default + project-overlay)? Склоняюсь к второму, но не на старте.
- Опубликовывать ли `sdp` бинарь как Homebrew tap для macOS-onboarding?

## 5. Sequencing

1. **F141-01** (manifest schema) — фундамент, без него остальное не имеет смысла.
2. **F141-02** (generator) — параллельно с -01, начиная как только schema стабильна.
3. **F141-06** (migration) — после -02, чтобы перегенерация перестраивала current state.
4. **F141-04** (doctor) — после -06, иначе на main фейлит сразу.
5. **F141-03** (bootstrap) — после -02 + -04 (нужны генератор и проверка).
6. **F141-05** (parity matrix) — параллельно с -03, generated step.
7. **F141-07** (README + recipes) — последним, требует working installer.

## 6. Acceptance for F141 (epic-level)

- В чистом downstream repo: `curl -fsSL .../install.sh | bash` ставит SDP, после чего `claude /build`, `opencode run sdp build`, `codex skill build`, и Cursor command "build" — все исполняют equivalent поверхность.
- `sdp doctor` зелёный в CI на `sdp_lab/main`.
- `docs/reference/harness-parity-matrix.md` сгенерирован, не содержит "missing" без `parity_notes`.
- README one-liner работает; downstream onboarding занимает < 5 минут (manual timing на свежем macOS-репо).
