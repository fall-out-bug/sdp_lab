# F141-06b Migration Audit

> **Status:** Audit (2026-04-25) · **Issue:** sdplab-m5xf · **Branch:** feature/F141-multi-harness-install-bootstrap
>
> This document records the per-item migration decisions for replacing live harness adapter trees with output from `sdp generate-adapters`.

## 1. Generator output (after F141-06a body embedding)

`sdp generate-adapters --write --out .sdp/generated` produces 130 files:

| Path | Count | Source field |
|---|---|---|
| `.claude/commands/<name>.md` | 24 | `commands[].path` body embed |
| `.claude/agents/<name>.md` | 12 | `agents[].system_prompt_path` body embed |
| `.opencode/agent/<name>.json` | 12 | metadata only (body omitted by design) |
| `.opencode/skill/<name>.md` | 29 | `skills[].path` body embed |
| `.codex/skills/<name>.md` | 29 | `skills[].path` body embed |
| `.cursor/rules/<name>.mdc` | 24 | `commands[].path` body embed |

## 2. Live tree inventory

Existing items in `.claude/`, `.opencode/`, `.codex/`, `.cursor/` and the migration decision per item.

### `.claude/`

| Item | Type | In manifest? | Decision | Reason |
|---|---|---|---|---|
| `commands/` | real dir | — | keep dir | container |
| `commands/deliver.md` | symlink → `../../prompts/commands/deliver.md` | yes (`commands[name=deliver]`) | **remove** | generator overwrites with embedded body |
| `commands/sweep.md` | real file (5.3K) | no | keep | not in manifest, F141 leaves it alone |
| `agents` | symlink → `../prompts/agents` | yes (12 agents) | **remove + recreate as dir** | generator writing through this symlink would clobber `prompts/agents/*` (canonical source) |
| `skills` | symlink → `../prompts/skills` | n/a | keep | generator does not write to `.claude/skills/` |
| `commands.json` | config (8.9K) | — | keep | Claude Code config, not adapter content |
| `settings.json`, `settings.local.json` | config | — | keep | Claude Code permissions/MCP |
| `hooks/`, `patterns/`, `worktrees/` | dirs | — | keep | infrastructure, out of F141 scope |

### `.opencode/`

| Item | Type | In manifest? | Decision | Reason |
|---|---|---|---|---|
| `agents` | symlink → `../prompts/agents` | n/a | keep | generator writes to `.opencode/agent/` (singular), no path conflict |
| `commands` | symlink → `../prompts/commands` | n/a | keep | generator does not produce OpenCode commands yet (deferred) |
| `agent/` (new) | will be created | yes | create | generator writes 12 .json files |
| `skill/` (new) | will be created | yes | create | generator writes 29 .md files |
| `hooks/`, `node_modules/`, `package*.json`, `opencode.json`, `mem-config.json`, `system-prompt-local.md`, `README.md`, `.gitignore` | configs | — | keep | OpenCode runtime config |

### `.codex/`

| Item | Type | In manifest? | Decision | Reason |
|---|---|---|---|---|
| `skills/` | real dir (only README.md) | — | keep dir | container |
| `skills/README.md` | real file (314B) | no | keep | not in manifest |
| `skills/<name>.md` (new) | will be created | yes | create | generator writes 29 files |
| `commands` | symlink → `../prompts/commands` | n/a | keep | generator does not produce Codex commands yet |
| `AGENTS.md`, `INSTALL.md` | docs | — | keep | Codex onboarding |

### `.cursor/`

| Item | Type | In manifest? | Decision | Reason |
|---|---|---|---|---|
| `commands` | symlink → `../prompts/commands` | n/a | keep | duplicate of `.cursor/rules/` discovery surface |
| `rules/` (new) | will be created | yes | create | generator writes 24 .mdc files |
| `hooks/`, `hooks.json`, `worktrees.json`, `README.md` | config | — | keep | Cursor runtime config |

## 3. Symlinks summary

After migration, these symlinks survive (intentional — they cover paths the generator does not emit yet):

- `.claude/skills → ../prompts/skills`
- `.opencode/agents → ../prompts/agents`
- `.opencode/commands → ../prompts/commands`
- `.codex/commands → ../prompts/commands`
- `.cursor/commands → ../prompts/commands`

These are removed:

- `.claude/agents → ../prompts/agents` (replaced with real dir + 12 generated files)
- `.claude/commands/deliver.md → ../../prompts/commands/deliver.md` (replaced with generated file)

Per F128-06 the surviving symlinks remain "tracked symlinks → prompts/". This is a pragmatic compromise:
F141-02 generator covers only a subset of (skill × command × agent) × (4 harnesses) cells; cells the generator does not fill remain symlink-discovered. A future expansion of templates will let us drop the rest.

## 4. Files preserved unchanged ("not in manifest, leave alone")

- `.claude/commands/sweep.md`
- `.claude/commands.json`, `.claude/settings.json`, `.claude/settings.local.json`
- `.claude/hooks/**`, `.claude/patterns/**`, `.claude/worktrees/**`
- `.opencode/hooks/**`, `.opencode/node_modules/**`, `.opencode/package*.json`, `.opencode/opencode.json`, `.opencode/mem-config.json`, `.opencode/system-prompt-local.md`, `.opencode/README.md`, `.opencode/.gitignore`
- `.codex/AGENTS.md`, `.codex/INSTALL.md`, `.codex/skills/README.md`
- `.cursor/hooks/**`, `.cursor/hooks.json`, `.cursor/worktrees.json`, `.cursor/README.md`

## 5. Acceptance check

After migration:

```bash
sdp generate-adapters --check --out .   # exit 0
git diff --stat                          # only expected changes (per this audit)
```

Followed by `sdp doctor --strict` once F141-04 lands — that gate will catch future drift.
