# Skill Authoring -- SDP Multi-Harness

> **Audience:** Authors creating a new SDP skill.
> **Canonical structured source:** `prompts/skills/<name>/SKILL.md` (manifest-backed).
> **Runtime alias surface:** `.agents/skills/<name>.md` for harnesses that read flat skills.
> **Policy source:** F127-03 (`docs/plans/2026-04-16-f127-multi-harness-modernization-design.md`).

## Why a policy

SDP-skills must work identically in all major harnesses (Claude Code, OpenCode, Cursor, Codex) without rewriting. A unified format makes a skill:
- indexable (marketplace, lint, search);
- versionable (semver → breaking changes are visible);
- portable between harnesses without modifications.

## File Location

**Canonical structured source:** `prompts/skills/<skill-name>/SKILL.md`.

This is the path listed in `sdp.manifest.yaml`, published to the public `sdp`
repo when protocol artifacts are exported, and used by generated adapters.

**Runtime alias/stub surface:** `.agents/skills/<skill-name>.md`.

Flat `.agents/skills/*.md` files serve harnesses that discover flat skills
directly, especially OpenCode/Cursor/Kimi-style loaders. Keep the alias/stub in
sync with the structured source when changing behavior.

**Do not** put real files in:
- `skills/` (removed in F128; no longer exists);
- `.claude/skills/` (symlink/adapted surface for harness loading);
- generated adapter directories such as `.sdp/generated/`, `.codex/`, `.opencode/`, or `.pi/`.

For structured skills, the directory is `<kebab-case>/` and frontmatter
`name:` matches the directory. For flat aliases, the filename is
`<kebab-case>.md` and matches `name:`.

## YAML Frontmatter -- required

```yaml
---
name: short-kebab-name
description: one sentence, 60-120 characters, what the skill does
version: 1.0.0
---
```

| Field | Required | Rule |
|------|----------|------|
| `name` | yes | kebab-case, matches filename without extension. No spaces, prefixes like `sdp-`. |
| `description` | yes | One sentence, 60-120 characters. Starts with a verb or noun, not "This is a skill that…". |
| `version` | yes | Semver. Start with `1.0.0`. Breaking changes → bump major. Non-breaking content updates → bump patch/minor. |

## YAML Frontmatter -- recommended

```yaml
compatibility:
  - claude-code
  - opencode
  - cursor
  - codex
requires_mcp: []           # list of MCP servers if skill expects MCP-tools
requires_cli: []           # list of CLI binaries on PATH (sdp, bd, gh, …)
tags: [discovery, review]  # free tags for search
tool_risk_classes: [perception, analysis]
```

| Field | When to specify |
|------|-----------------|
| `compatibility` | Always, except for explicitly Claude-only skills. List of harnesses where the skill is tested. |
| `requires_mcp` | If skill expects a specific MCP server (e.g., `beads`, `claude-api`). Empty array = no requirements. |
| `requires_cli` | CLI binaries without which the skill will not run (example: `[sdp, bd]`). |
| `tags` | For lint/search/marketplace indexing. |
| `tool_risk_classes` | Side-effect classes used by the skill. Use the vocabulary in [harness-risk-and-evidence.md](harness-risk-and-evidence.md). |

`compatibility` is a runtime claim only after the harness has dispatch evidence.
Before that, treat it as intended portability and report runtime coverage as
`not_assessed_runtime` for that harness.

## Body structure (recommended)

```markdown
---
name: my-skill
description: …
version: 1.0.0
compatibility: [claude-code, opencode, cursor, codex]
---

# My Skill -- <human-readable title>

## Purpose
1-3 paragraphs. What it does, what outcome.

## Use When
- bullet-list of situations when to apply

## Do Not Use When
- common false triggers and anti-patterns

## Inputs / Outputs
What skill expects on input (context, files, args), what it returns.

## Process
Execution steps. If >5 steps -- break into subsections.

## Verification
How the agent proves the skill did what it claims. Prefer deterministic evidence:
tests, lint output, schema validation, file existence, Beads/GitHub state, or
captured runtime logs.

## Degraded Evidence
How the skill reports missing or partial evidence. Use
`not_assessed`, `timeout`, `empty_output`, `off_task`, `unavailable_cli`,
`not_assessed_runtime`, or `manual_gate_only` rather than collapsing unknowns
into pass.

## References
- related skills
- design docs (docs/plans/YYYY-MM-DD-*.md)
- external sources
```

Do not duplicate common rules (beads rules, git workflow) -- reference `AGENTS.md`.
Use [harness-risk-and-evidence.md](harness-risk-and-evidence.md) for shared
tool-risk classes and degraded-evidence vocabulary.

## Boundary With AGENTS.md

Skills own executable workflows: triggers, preconditions, steps, outputs, stop
conditions, and recovery behavior.

Do not put module facts in a skill:

- package API contracts
- allowed or forbidden internal imports
- runtime assumptions for one subtree
- provider credentials for one package
- local gates that apply only under one directory

Put those facts in the nearest module-local `AGENTS.md` instead. Root `AGENTS.md`
owns repo-wide policy only. See [agent-instruction-cascade.md](agent-instruction-cascade.md)
for the full layering model.

## Length limit

Per [ADR-007 `docs/decisions/007-skill-length-limit.md`](../decisions/007-skill-length-limit.md) target length is ≤100 lines. Exceptions (`llm-council.md`, ~330 lines) are acceptable if the skill really requires tables/protocols. If exceeded -- check if details can be moved to a separate reference doc.

## Harness-neutral prose

- **Do not write** "In Claude Code, use the Task tool to…" -- instead "To dispatch a sub-agent, use your harness's native sub-agent mechanism (Task tool in Claude Code, `@agent` in OpenCode, etc.).".
- **Do not hardcode** harness commands without a wrapper "if this harness applies".
- **MCP-tools** specify via `requires_mcp`, not "you need the Beads MCP server" in body.

## Versioning

- Stable release skills: `1.x.y+`.
- Experimental / draft: `0.x.y`.
- Breaking change = new major. Document breaking in body in `## Changelog` section or move to git history.
- For marketplace-like tooling (claudemarketplaces.com, future SDP registry) exactly semver-compatible tags are important.

## Example -- minimal valid skill

```markdown
---
name: repo-scout
description: 30-second project card for unknown repo -- file counts, languages, recent activity, primary build system.
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

F127-08 added `sdp-protocol-check --lint-skills` -- scan `.agents/skills/*.md`:

```bash
# Locally
go run ./cmd/sdp-protocol-check --lint-skills

# JSON output for CI
go run ./cmd/sdp-protocol-check --lint-skills --format json
```

**Errors (exit code 2):**
- missing required frontmatter key `name`, `description` or `version`.

**Warnings (exit code 0):**
- missing `compatibility` -- skill does not declare harness-portability.
- `name` does not match `<filename>.md` (kebab-case).
- `description` outside 60-120 character window.
- body contains hardcoded harness-specific phrases (e.g., "In Claude Code,", "Use the Task tool", "Claude Code only").

In CI (`.github/workflows/ci.yml`, `consistency-gate` job) runs non-blocking -- findings are written to `.sdp/findings/sdp-skill-lint-*.json`.

## References

- [F127 design](../plans/2026-04-16-f127-multi-harness-modernization-design.md)
- [.agents/skills/README.md](../../.agents/skills/README.md)
- [ADR-007: Skill Length Limit](../decisions/007-skill-length-limit.md)
- [OpenCode Skills docs](https://opencode.ai/docs/skills/)
- [AGENTS.md spec](https://agents.md/)
