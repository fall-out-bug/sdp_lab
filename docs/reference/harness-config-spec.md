# Harness Config Manifest — Reference Spec

**Schema:** `schema/harness-config-manifest.schema.json` (JSON Schema draft-07)

## What the Manifest Is For

`harness-config-manifest.json` is a machine-readable description of which AI coding harnesses
are active in a project, where their config files live, and what lifecycle stage the project is in.

Three purposes:
1. **Drift detection** — the `version` field lets tooling detect schema changes and trigger regeneration.
2. **Onboarding automation** — agents discover every harness config file without reading the whole repo.
3. **Generation targeting** — code-gen scripts read the manifest to know which files to write and which lifecycle defaults to apply.

## Lifecycle Stages

| Stage | Condition | Action |
|-------|-----------|--------|
| `greenfield` | No existing codebase; first 0–5 commits, no go.sum/package-lock.json | Generate maximally opinionated defaults. All harnesses enabled. Strict gates from day one. |
| `brownfield-new` | Existing codebase, but harness config files do not exist yet | Generate conservative defaults. Warn on missing tests. Enable harnesses incrementally. |
| `brownfield-mature` | Existing codebase with established harness configs present | Diff-only updates. Never overwrite custom rules. Append-only for new harnesses. |

**Decision rules (apply highest matching stage):**
1. `git log --oneline | wc -l` returns 0–5 AND no `go.sum`/`package-lock.json` → `greenfield`
2. Harness config files (`CLAUDE.md`, `.cursor/rules/`) do not exist → `brownfield-new`
3. Otherwise → `brownfield-mature`

## Supported Harnesses

| Name | Config File Path | Notes |
|------|-----------------|-------|
| `claude-code` | `CLAUDE.md` | Read by Claude Code on startup |
| `codex-cli` | `AGENTS.md` | Shared canonical operator rules file |
| `cursor` | `.cursor/rules/project.mdc` | MDC frontmatter; supports `globs` and `alwaysApply` |
| `opencode` | `.opencode/config.json` | JSON config; `agent` key for sub-agent dispatch |
| `copilot` | `.github/copilot-instructions.md` | Plain markdown, read by GitHub Copilot Chat |
| `zed` | `.zed/settings.json` | JSON settings; `assistant.default_model` key |
| `warp` | `.warp/rules.md` | Plain markdown Warp AI rules file |

## Schema Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `version` | string (semver) | Yes | Schema version. Increment on breaking changes for drift detection. |
| `lifecycle_stage` | enum | Yes | One of `greenfield`, `brownfield-new`, `brownfield-mature` |
| `harnesses` | array | Yes | One or more harness objects (minItems: 1) |
| `language` | string | No | Primary language: `"go"`, `"typescript"`, etc. |
| `rules_file` | string (path) | No | Path to the single-source-of-truth patterns doc |

**Harness object:**

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `name` | enum | Yes | — | Canonical harness identifier (see table above) |
| `config_file` | string | Yes | — | Relative path to harness config file |
| `enabled` | boolean | No | `true` | `false` = retain docs, exclude from generation |

## Example Manifest

```json
{
  "version": "1.0.0",
  "lifecycle_stage": "brownfield-mature",
  "language": "go",
  "rules_file": "docs/reference/go-patterns.md",
  "harnesses": [
    { "name": "claude-code", "config_file": "CLAUDE.md",                       "enabled": true  },
    { "name": "codex-cli",   "config_file": "AGENTS.md",                       "enabled": true  },
    { "name": "cursor",      "config_file": ".cursor/rules/project.mdc",       "enabled": true  },
    { "name": "opencode",    "config_file": ".opencode/config.json",           "enabled": true  },
    { "name": "copilot",     "config_file": ".github/copilot-instructions.md", "enabled": false }
  ]
}
```

## Validation

```bash
go test ./internal/harnesscfg/... -v
```
