# Git Remote Bootstrap (Private)

Use this when origin repository has no default branch.

## Symptoms

- `gh repo view --json defaultBranchRef` returns empty branch name
- `cmd/pr-publish` fails with:
  - `repository has no default branch; initialize remote with first commit before PR publishing`

## Bootstrap steps

1. Create first commit in this repository.
2. Push to `origin` and establish default branch (usually `main`).
3. Re-run PR publish flow.

## Validation

```bash
gh repo view --json defaultBranchRef --jq .defaultBranchRef.name
```

Expected: non-empty branch name.
