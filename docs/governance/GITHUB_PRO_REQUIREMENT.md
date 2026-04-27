# GitHub Branch Protection Requirement (F030-01)

## Status: Public Repo Path Available

As of 2026-04-27, `sdp_lab` is public. GitHub branch protection is available without the private-repo GitHub Pro constraint. CODEOWNERS remains useful for review ownership, but it is not the substitute for server-side branch protection anymore.

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

Enable server-side branch protection on `main`:

1. Go to repository Settings → Branches
2. Add rule for `main` branch:
   - ✅ Require status checks to pass before merging
     - `build-test`
     - `evidence-gate`
     - `policy-gate`
   - ✅ Require branches to be up to date before merging
   - ✅ Do not allow bypassing the above settings
   - ✅ Restrict who can push to matching branches (repo owner only)

## Testing Branch Protection

```bash
# Attempt direct push to main without passing CI
git push origin main
# Should be REJECTED by GitHub
```

## Migration Checklist

For the current public repo:
- [ ] Configure branch protection rule for `main`
- [ ] Add required status checks
- [ ] Enable "Do not allow bypassing"
- [ ] Test with direct push (should fail)
- [ ] Document in team onboarding
- [ ] Update this document

## References

- GitHub Branch Protection: https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches
- GitHub CODEOWNERS: https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-code-owners
- Workstream: [00-030-01](../workstreams/backlog/00-030-01.md)
