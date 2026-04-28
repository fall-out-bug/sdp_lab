# sdp_lab <-> sdp Publish Workflow

If your real goal is "install SDP into my own repo and start using it", leave this doc and go to [docs/QUICKSTART.md](QUICKSTART.md). This file is for contributors who need to publish selected protocol artifacts from `sdp_lab` to the distilled `sdp` repo.

## How It Works

All files live natively in `sdp_lab`. There is no submodule, no separate git checkout, and no pointer updates.

| Path | Repo | Remote | Typical change |
|------|------|--------|----------------|
| Root, `internal/`, `cmd/`, `docs/`, `sdp/` | `sdp_lab` | `https://github.com/fall-out-bug/sdp` | All work happens here |
| `fall-out-bug/sdp` (separate repo) | `sdp` (distilled distribution repo) | `https://github.com/fall-out-bug/sdp` | Published artifacts only |

Historical note: many workstreams and beads IDs still use `sdp_dev-*` as a legacy label for the root repo. The Go module path was migrated from `sdp_dev` to `sdp_lab` in F150-03. That is history, not a third repo.

## When to Publish

Publish to the distilled `sdp` repo when protocol artifacts change and external consumers need the update:

- Schema changes (evidence, intent, ws-verdict)
- New or updated prompts/skills
- Hook changes
- Harness entrypoint files (`.cursorrules`, `.codex/*`, `.opencode/*`)
- Fallback and operator-facing reference docs
- Quickstart or CLI reference updates

**Not published via this script:** arbitrary Go source is not part of the distillation surface unless the publish manifest explicitly includes it.

## Publish with sdp-publish.sh

```bash
# Publish changed artifacts (copy, commit, push to sdp repo):
scripts/sdp-publish.sh

# Preview what would be published without pushing:
scripts/sdp-publish.sh --dry-run

# Check for drift between sdp_lab source and published sdp repo:
scripts/sdp-publish.sh --check
```

The script copies the relevant files from `sdp_lab` into a checkout of `fall-out-bug/sdp`, commits, and pushes. It uses a manifest to determine which paths to publish.

## Artifacts Published

| Source (sdp_lab) | Destination (sdp repo) |
|---|---|
| `prompts/` | `prompts/` (includes `prompts/skills/`) |
| `schema/` | `schema/` |
| `templates/` | `templates/` |
| `scripts/hooks/` | `hooks/` |
| `.claude/hooks/` | `.claude/hooks/` |
| `.claude/patterns/` | `.claude/patterns/` |
| `.opencode/hooks/` | `.opencode/hooks/` |
| `.cursorrules` | `.cursorrules` |
| `.cursor/README.md` | `.cursor/README.md` |
| `.cursor/worktrees.json` | `.cursor/worktrees.json` |
| `.codex/AGENTS.md` | `.codex/AGENTS.md` |
| `.codex/INSTALL.md` | `.codex/INSTALL.md` |
| `.codex/skills/README.md` | `.codex/skills/README.md` |
| `.opencode/README.md` | `.opencode/README.md` |
| `docs/reference/FALLBACK_MODE.md` | `docs/reference/FALLBACK_MODE.md` |
| `prompts/commands.yml` | `prompts/commands.yml` |

The manifest is maintained in `sdp_lab`. Add or remove paths there when the publish surface changes.

## Branch Defaults

| Repo | Default branch |
|------|----------------|
| `sdp_lab` | `main` |
| `sdp` | `main` |

## Commit Workflow

1. Make changes in `sdp_lab`. Protocol artifacts live at native tracked paths (`prompts/`, `schema/`, `templates/`, `.claude/hooks/`, harness entrypoints, fallback docs). The `sdp/` directory is an optional local checkout of the distilled repo used only by the publish script.
2. Commit and push in `sdp_lab` as usual.
3. After merge to `main`, run `scripts/sdp-publish.sh` if protocol artifacts changed and the distilled repo needs them.

## CI Integration

(TODO: add CI workflow for drift detection.) The intended design is that CI runs `scripts/sdp-publish.sh --check` on every push to `main`. If the check fails, the published `sdp` repo has drifted from the source and someone needs to publish. This workflow has not been implemented yet.

## Historical Reference

The `sdp/` directory was previously a git submodule. That model was retired in F128 (ADR: `docs/architecture/adr-retire-sdp-submodule.md`). The old two-phase commit workflow (commit in `sdp/` submodule, then update pointer in `sdp_lab`) no longer applies.
