# Branch Protection Setup

## Overview

This guide explains how to configure branch protection for SDP repositories.

## Manual Setup (GitHub UI)

### Step 1: Navigate to Settings

1. Go to repository Settings
2. Click "Branches" in sidebar
3. Click "Add rule" or edit existing

### Step 2: Configure master branch

**Branch name pattern:** `master` (or `main` for repos using main)

**Settings:**
- ✅ Require a pull request before merging
  - ✅ Require approvals: 1
- ✅ Require status checks to pass before merging
  - ✅ Require branches to be up to date
  - **Required checks:**
    - `build-test`
    - `evidence-gate`
    - `policy-gate`
    - `scope-gate`
- ✅ Do not allow bypassing the above settings

### Step 3: Optional — dev branch

**Branch name pattern:** `dev`

**Settings:**
- ✅ Require status checks to pass before merging
  - **Required checks:** `build-test`, `evidence-gate`, `policy-gate`, `scope-gate`

## Verify Setup

```bash
# Check protection status
gh api repos/{owner}/{repo}/branches/master/protection

# Test by creating failing PR
git checkout -b test-protection
echo "test" >> internal/something.go
git add internal/something.go
git commit -m "test: verify evidence-gate"
git push -u origin test-protection
gh pr create --title "Test protection" --body "Should be blocked (no evidence)"
```


## Verification Tests (F055-03)

After configuring branch protection, verify gates block:

1. **evidence-gate fail (no evidence):** PR with `internal/` or `cmd/` change, no `.sdp/evidence/*.json` in diff → evidence-gate must fail.
2. **evidence-gate fail (invalid evidence):** PR with `internal/` change + `.sdp/evidence/F055.json` containing invalid JSON → evidence-gate must fail.
3. **scope-gate fail:** PR with out-of-scope file (e.g. change file not in WS scope) → scope-gate must fail.

## Troubleshooting

### Check not appearing

Ensure job names match exactly from `.github/workflows/ci.yml`:
- `build-test`
- `evidence-gate`
- `scope-gate`
- `policy-gate`

Required check uses job name, not workflow name.

### Admin bypass

By default, admins can bypass. To prevent:
- ✅ Do not allow bypassing the above settings

## Notes

- Jobs are from `.github/workflows/ci.yml`
- PRs cannot merge until all required checks pass
- GitHub branch protection requires GitHub Pro (or public repo) for private repositories
