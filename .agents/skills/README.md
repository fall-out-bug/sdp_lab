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
│   ├── git-worktree.md
│   ├── llm-council.md
│   ├── parallel-dispatch.md
│   ├── review-readiness.md
│   └── strataudit.md
├── skills/                   ← compat symlinks → .agents/skills/*.md
│   ├── git-worktree.md     → ../.agents/skills/git-worktree.md
│   ├── llm-council.md      → ../.agents/skills/llm-council.md
│   ├── parallel-dispatch.md → ../.agents/skills/parallel-dispatch.md
│   ├── review-readiness.md → ../.agents/skills/review-readiness.md
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

`sdp/prompts/skills/` is the submodule-publish path for artifacts that are released to the public `sdp` protocol repo. Internal build/lab skills live in `.agents/skills/`. Publication is via a separate PR in the submodule (see `docs/MULTI-REPO-WORKFLOW.md`).

## References

- [F127 design](../../docs/plans/2026-04-16-f127-multi-harness-modernization-design.md)
- [OpenCode Skills](https://opencode.ai/docs/skills/)
- [AGENTS.md spec](https://agents.md/)
