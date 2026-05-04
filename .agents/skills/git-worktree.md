---
name: git-worktree
description: Safety-first git worktree setup — directory selection, gitignore, baseline checks, and cleanup.
version: 1.0.0
tags:
  - git
  - isolation
requires_cli:
  - git
compatibility:
  - claude-code
  - opencode
  - cursor
  - codex
---

# git-worktree

## Purpose

Create isolated worktrees for parallel feature work with mandatory safety
checks. Work on multiple branches simultaneously without stashing or switching.

## Directory Selection Priority

1. `.worktrees/` — preferred (hidden, project-local)
2. `worktrees/` — alternative project-local
3. Project docs (CLAUDE.md / AGENTS.md) configured preference
4. Ask user — offer global location as fallback

If both `.worktrees/` and `worktrees/` exist, prefer `.worktrees/`.

## Safety Guards

Never skip these checks:

- **Gitignore** — `git check-ignore -q <dir>` before creating project-local
  worktree. If not ignored, add to `.gitignore` and commit first.
- **Clean working tree** — no uncommitted changes, or user explicitly accepts.
- **Baseline tests** — run test suite after creation. Report failures, never
  proceed silently.
- **No duplicate branches** — verify target branch has no existing worktree.

## Creation Steps

1. **Detect project:** `basename "$(git rev-parse --show-toplevel)"`
2. **Select directory:** follow priority above
3. **Verify gitignore:** `git check-ignore -q <dir>` (project-local only)
4. **Create worktree:** `git worktree add <dir>/<name> -b <branch-name>`
5. **Install deps** — auto-detect: `package.json`→npm, `go.mod`→go mod
   download, `Cargo.toml`→cargo build, `requirements.txt`→pip install
6. **Verify baseline:** run project test suite, report results

## Cleanup Cadence

| When | Action |
|------|--------|
| Feature merged | `git worktree remove <path>` + `git branch -d <branch>` |
| Feature abandoned | `git worktree remove <path>` + `git branch -D <branch>` |
| Stale (>7 days idle) | `git worktree list`, remind user |
| End of session | Report active worktrees, suggest cleanup |

After a PR merge, clean up from a different worktree if the feature branch is
currently checked out. `gh pr merge --delete-branch` may delete the remote branch
but fail to delete the local branch when that branch is still attached to an
open worktree. Verify with `git ls-remote --heads origin <branch>` and then run
`git worktree remove <path>` followed by `git branch -d <branch>`.

Never infer delivery state from an open worktree alone. Confirm PR state via
GitHub and Beads state via `bd`; stale worktrees are cleanup debt, not evidence
that a feature is still unmerged.

## Conflict Prevention

- One worktree per branch — never share a branch across worktrees.
- `git worktree prune` before add and after remove.
- Resolve cross-worktree conflicts on a dedicated branch, not on feature
  branches directly.
- Submodule pointers are shared — coordinate changes across worktrees. (Historical: submodules were retired in F128.)

## Quick Reference

| Situation | Action |
|-----------|--------|
| Dir not gitignored | Add to `.gitignore` + commit first |
| Baseline tests fail | Report + ask; do not proceed silently |
| Branch already has worktree | Remove old or use different branch |
| Unsure which dir | Follow priority list, then ask user |
