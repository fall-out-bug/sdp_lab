# ADR: Retire sdp/ Git Submodule for Extra-Hop Publish Model

**Status**: Accepted
**Date**: 2026-04-19
**Decision makers**: Andrei
**Feature**: F128-05 (sdplab-utn3)

## Context

`sdp_lab` uses `sdp/` as a git submodule (`https://github.com/fall-out-bug/sdp.git`, branch `main`) containing public protocol artifacts: prompts, agents, skills, JSON schemas, git hooks, document templates, and the `sdp-plugin` Go CLI. The submodule was introduced so that external consumers could reference a single public repo for SDP protocol files without seeing the private lab workspace.

### Current Setup

- `.gitmodules` pins `sdp` at `https://github.com/fall-out-bug/sdp.git`, branch `main`.
- Two symlinks bridge submodule content into harness-visible paths:
  - `.claude/agents` -> `../../sdp/prompts/agents`
  - `.claude/hooks` -> `../../sdp/.claude/hooks`
- The commit workflow is two-phase: commit inside `sdp/`, push, then update the submodule pointer in `sdp_lab` and push again. See `docs/MULTI-REPO-WORKFLOW.md` (Step 8 in AGENTS.md).
- `docs/architecture/REPO-BOUNDARY.md` documents which components live where and when to publish.

### Problems

1. **Broken symlinks on cold start.** If a clone omits `--recurse-submodules`, `.claude/agents` and `.claude/hooks` dangle. The agent harness cannot find skills, agent definitions, or hooks. This breaks every session until the developer runs `git submodule update --init`.

2. **Double git workflow.** Every protocol change requires committing in `sdp/` first, pushing, then returning to `sdp_lab` to update the submodule pointer (`git add sdp`) and pushing again. See `docs/MULTI-REPO-WORKFLOW.md` for the full dance. This adds friction and is easy to get wrong -- the parent repo can end up pointing at a commit that does not exist in the `sdp` remote, requiring manual recovery.

3. **Cold-start friction.** New contributors and CI runners must remember `--recurse-submodules` or add a separate `git submodule update --init` step. CLAUDE.md explicitly warns about this; AGENTS.md documents submodule recovery commands. The fact that recovery documentation is needed is itself a signal.

4. **Sub-agent complexity.** Sub-agents (Agent tool, Codex rescue, OpenCode implementer) that dispatch into this repo need the submodule initialized to function. Each harness has its own initialization path, and forgetting any of them produces confusing errors about missing agents or hooks.

5. **CI overhead.** Every CI pipeline needs an extra `git submodule update --init` checkout step. The `run_go_quality_gates.sh` script and GitHub Actions workflows must account for the submodule.

6. **Asymmetric edit frequency.** The `sdp/` contents change rarely (only when publishing protocol artifacts), but the submodule machinery adds overhead to every clone, every branch, and every CI run. The cost is continuous; the benefit is episodic.

## Decision

Retire the `sdp/` git submodule. Move all files that currently live in the submodule into `sdp_lab` as native files. Introduce an explicit publish step (`scripts/sdp-publish.sh`) that copies protocol artifacts to `github.com/fall-out-bug/sdp` when a release is warranted.

### New Model: Extra-Hop Publish

1. **All files live natively in sdp_lab.** Prompts, schemas, hooks, agent definitions, and the `sdp-plugin` source code reside under their canonical paths in the monorepo. No symlinks, no submodule pointers. Note: `sdp-plugin` Go source code lives in the public `sdp` repo only; it is not a publish artifact managed by `sdp-publish.sh`.

2. **Publishing is an explicit script.** `scripts/sdp-publish.sh` copies the relevant files from `sdp_lab` into a checkout of `fall-out-bug/sdp`, commits, and pushes. The script supports `--dry-run` for CI validation.

3. **Symlinks become real directories.** `.claude/agents` and `.claude/hooks` become regular directories containing the files directly. No indirection, no dangling references.

4. **`docs/MULTI-REPO-WORKFLOW.md` is retired or rewritten.** The two-phase commit workflow document becomes unnecessary. A shorter publish guide replaces it.

5. **`.gitmodules` and the `sdp/` gitlink are removed.** No more submodule state in the repository.

### Publish Script Contract

```
scripts/sdp-publish.sh              # Copy artifacts, commit, push to sdp repo
scripts/sdp-publish.sh --dry-run    # Show what would be published (for CI)
scripts/sdp-publish.sh --check      # Fail if sdp_lab and published sdp have drifted
```

Artifacts published by the script (`scripts/sdp-publish.sh`) -- protocol artifacts only:
- `prompts/` -> `sdp/prompts/` (includes `prompts/skills/`)
- `schema/` -> `sdp/schema/`
- `templates/` -> `sdp/templates/`
- `scripts/hooks/` -> `sdp/hooks/`
- `.claude/hooks/` -> `sdp/.claude/hooks/`
- `.claude/patterns/` -> `sdp/.claude/patterns/`

**Not published:** `sdp-plugin` Go source code lives exclusively in the public `sdp` repo (`sdp/sdp-plugin/`). It is not part of the publish surface because it is Go source code, not a protocol artifact. The publish script covers only prompts, schemas, hooks, and templates.

## Consequences

### Positive

- **Single git workflow.** One repo, one commit, one push. No more two-phase commits or submodule pointer updates.
- **No broken symlinks.** Agents, hooks, and skills are always present. Cold start is `git clone` and nothing else.
- **Simpler CI.** Remove `git submodule update --init` from all workflows. Fewer failure modes.
- **Sub-agents work out of the box.** No special initialization required for any harness.
- **Single source of truth.** Files live in one place. The published `sdp` repo is a downstream mirror, not an upstream dependency.
- **Faster onboarding.** New contributors clone and start working immediately.
- **Affected docs simplify.** `AGENTS.md` Cold Start section, `CLAUDE.md` submodule init warning, `docs/MULTI-REPO-WORKFLOW.md`, and `docs/architecture/REPO-BOUNDARY.md` all become shorter or are retired.

### Negative

- **Explicit publish step can be forgotten.** Protocol changes in `sdp_lab` are not automatically reflected in the public `sdp` repo. Someone must run the publish script.
- **No real-time bi-directional sync.** External consumers of `fall-out-bug/sdp` see updates only when published, not on every commit. In practice this is fine because the submodule model was never real-time either -- it required manual pointer updates.
- **Git history split.** The full history of protocol files lives in `sdp_lab` from the migration point forward. The `sdp` repo retains pre-migration history; post-migration history is only publish commits.
- **Larger clone for sdp-only consumers.** External users who only want the protocol artifacts must clone the full `sdp_lab` or rely on the published `sdp` repo mirror. The publish script mitigates this by keeping the mirror up to date.

### Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Publishing forgotten after protocol changes | Manual: run `scripts/sdp-publish.sh --check` locally to detect drift. (TODO: add CI workflow for automated drift detection on push to `main`.) |
| Drift between sdp_lab source and published sdp repo | `sdp-publish.sh --dry-run` shows what would change. `--check` exits non-zero on drift. (TODO: integrate `--check` into CI pipeline.) |
| File paths change during migration, breaking existing references | Migration script updates all internal references (docs, agents, CI configs) in one commit. |
| External consumers depend on specific commit SHAs in sdp repo | Publish script preserves file paths in the sdp repo. Consumers pin to tags, not SHAs. |
| Publish script copies stale or incorrect files | Manifest file in `sdp_lab` lists exact paths to publish. Script validates against manifest before pushing. |

## Alternatives Considered

1. **Keep submodule (status quo).** The problems listed above persist. Onboarding friction, broken symlinks, and double-git workflow continue to waste time. Rejected because the costs are continuous while the benefits (independent versioning of a rarely-changed directory) are minimal.

2. **Full monorepo (merge sdp into sdp_lab entirely).** This ADR is essentially this option for file management. The public `sdp` repo becomes a downstream mirror rather than a separate source of truth. The `sdp-plugin` Go code moves into `sdp_lab` as a native module.

3. **npm/pip package.** Publish protocol artifacts as a versioned package (e.g., `@fall-out-bug/sdp-protocol`). Overkill for prompts and JSON schemas. Adds a language-specific dependency where none is needed. Rejected.

4. **Git subtrees.** Replace the submodule with `git subtree` to merge sdp content into sdp_lab while preserving the ability to push back. Similar operational complexity with different tradeoffs (larger repo size, merge conflicts on subtree pull). Does not solve the fundamental problem of needing to synchronize two repos. Rejected.

## Affected Documents

The following documents require updates after this decision is implemented:

- `AGENTS.md` -- Remove Step 8 (Protocol Changes submodule workflow), simplify Cold Start section, update repo topology table.
- `CLAUDE.md` -- Remove "Submodule init" hard rule.
- `docs/MULTI-REPO-WORKFLOW.md` -- Retire or replace with a shorter publish guide.
- `docs/architecture/REPO-BOUNDARY.md` -- Rewrite to reflect native ownership; the "When to Publish" section stays but references the publish script instead of submodule workflow.
- `docs/reference/project-map.md` -- Update SOT split table; remove submodule references.
- `.gitmodules` -- Delete.
- `.claude/agents` symlink -> real directory. Agent definitions live at `prompts/agents/` (tracked native path); `.claude/agents` was not converted since the actual definitions are already in `prompts/agents/`.
- `.claude/hooks` symlink -> real directory. Hook scripts are tracked natively at `.claude/hooks/` after migration.
