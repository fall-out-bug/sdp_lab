# GitHub Branch Protection Requirement (F030-01)

## Status: Partial Implementation

As of 2026-04-26, this repository is using **CODEOWNERS** as a partial substitute for GitHub branch protection, which requires GitHub Pro for private repositories.

## What We Have Now

### CODEOWNERS File
A `.github/CODEOWNERS` file has been created that:
- Requires maintainer approval for all changes
- Defines specific ownership for critical paths (protocol artifacts, core platform code, CI/CD)
- Serves as a human-readable policy for who should review what

### CI Gates
The repository has comprehensive CI gates that provide enforcement:
- **build-test**: Compile and test coverage
- **evidence-gate**: Attestation validation
- **scope-gate**: Workstream boundary compliance
- **protocol-compliance**: Harness contract validation
- **consistency-gate**: Repository consistency checks
- **policy-gate**: OPA-based policy enforcement

These gates run on every pull request and must pass before merging.

## What We Need for Full Protection

To enable server-side branch protection (bypass-proof enforcement), we need one of:

### Option 1: GitHub Pro (Recommended for Private Repos)
- **Cost**: $4/user/month
- **Benefits**:
  - Required status checks before merging
  - "Require branches to be up to date before merging"
  - "Do not allow bypassing the above settings"
  - Restrict who can push to protected branches
- **Action**: Upgrade at https://github.com/settings/billing

### Option 2: Make Repository Public
- **Cost**: Free
- **Benefits**: Branch protection available for free
- **Drawback**: Private code becomes public
- **Action**: Only viable if all code can be public

### Option 3: Use CODEOWNERS + Required Reviews (Current Approach)
- **Cost**: Free
- **Benefits**: Works with free plan
- **Drawback**: Relies on developer discipline + CI gates (not bypass-proof)
- **Current Status**: ✅ Implemented

## Configuration for GitHub Pro

If the repository upgrades to GitHub Pro, configure branch protection as follows:

1. Go to repository Settings → Branches
2. Add rule for `master` branch:
   - ✅ Require status checks to pass before merging
     - `build-test`
     - `evidence-gate`
     - `policy-gate`
   - ✅ Require branches to be up to date before merging
   - ✅ Do not allow bypassing the above settings
   - ✅ Restrict who can push to matching branches (repo owner only)

## Testing Branch Protection

### Without GitHub Pro (Current Setup)
```bash
# Test that CODEOWNERS is recognized
git push origin <branch>
# Should show "Code owner approval required" message in PR
```

### With GitHub Pro (Future)
```bash
# Attempt direct push to master without passing CI
git push origin master
# Should be REJECTED by GitHub
```

## Migration Checklist

When upgrading to GitHub Pro:
- [ ] Upgrade repository to GitHub Pro
- [ ] Configure branch protection rule for `master`
- [ ] Add required status checks
- [ ] Enable "Do not allow bypassing"
- [ ] Test with direct push (should fail)
- [ ] Document in team onboarding
- [ ] Update this document

## References

- GitHub Branch Protection: https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches
- GitHub CODEOWNERS: https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-code-owners
- Workstream: [00-030-01](../workstreams/backlog/00-030-01.md)
