# SDP Skills -- Multi-Harness Skill Directory

Canonical path for SDP-skills, read by all major agent harnesses.

## Why `.agents/skills/`

As of April 2026 `.agents/skills/` is the only directory that is **natively read by**:

| Harness | Path that the harness scans |
|---|---|
| Claude Code | `.claude/skills/` (we have a symlink → `.agents/skills/`) |
| OpenCode | `.agents/skills/` (native) + `.claude/skills/` (fallback) |
| Cursor | `.agents/skills/` (native) + `.claude/skills/` (fallback) |
| Codex CLI | reads via AGENTS.md + explicit path |

See [OpenCode Skills docs](https://opencode.ai/docs/skills/).

## Layout

```
sdp_lab/
├── .agents/skills/           ← canonical source (real files live here)
│   ├── README.md             ← this file
│   ├── build.md              ← F125 intent skills
│   ├── fix.md
│   ├── operate.md
│   ├── review.md
│   ├── understand.md
│   ├── git-worktree.md       ← pre-F125 skills
│   ├── llm-council.md
│   ├── parallel-dispatch.md
│   ├── strataudit.md
│   ├── architect.md          ← F125 legacy redirect stubs (deprecated)
│   ├── bugfix.md
│   ├── ... (20 stubs total)
│   └── vision.md
├── docs/reference/internal/  ← non-skill artifacts
│   ├── deprecated-aliases.md ← machine-readable mapping (NOT a skill)
│   └── review-readiness.md   ← @review dimension extension (NOT standalone)
├── skills/                   ← compat symlinks → .agents/skills/*.md
│   ├── build.md            → ../.agents/skills/build.md
│   ├── ... (symlinks per skill)
│   └── strataudit.md       → ../.agents/skills/strataudit.md
└── .claude/skills → ../.agents/skills   (entire directory symlink)
```

Old paths (`skills/*.md` and `.claude/skills/*.md`) continue to resolve -- `.claude/commands.json` and links in `docs/` are not broken.

## SKILL.md frontmatter (expected)

YAML frontmatter required fields:

```yaml
---
name: short-kebab-name
description: one-line what the skill does (60-120 characters)
version: 1.0.0                # semver
compatibility:                 # optional but recommended
  - claude-code
  - opencode
  - cursor
  - codex
---
```

Full authoring guide: `docs/reference/skill-authoring.md` (F127-03).

## Why not `sdp/prompts/skills/`

`sdp/prompts/skills/` is a retired path from the submodule era. The current publish surface is `prompts/` (native path in this repo), which includes `prompts/skills/`. Publication is via `scripts/sdp-publish.sh` which exports protocol artifacts to the public sdp repo.

### Relationship between `.agents/skills/` and `prompts/skills/`

- `prompts/skills/` is the comprehensive source containing full SKILL.md files with frontmatter, descriptions, and instructions.
- `.agents/skills/` contains runtime stub/alias files used by harnesses for skill discovery. These are the files that agent harnesses actually scan at runtime.
- Both paths are kept in sync. The authoritative content for public publishing is `prompts/skills/`.

## References

- [F127 design](../../docs/plans/2026-04-16-f127-multi-harness-modernization-design.md)
- [OpenCode Skills](https://opencode.ai/docs/skills/)
- [AGENTS.md spec](https://agents.md/)
