# F150-08: Homebrew Formula Dry Run Evidence

**Date:** 2026-04-27 16:18:25 UTC
**Commit:** a10623360946d0176deca11aff0477e09d915b56
**Runner:** fall_out_bug@Andreys-MacBook-Air.local

## What was tested

1. Formula file syntax validation (Ruby + Homebrew compatibility)
2. sdp binary build from source via `go build`
3. Binary smoke tests:
   - `sdp` (no args) prints usage text
   - `sdp scout --help` shows read-only subcommand usage
   - No lab-only/experimental binaries included in build output
4. Brew install from local formula (if available)

## Results

| Test | Result |
|------|--------|
| Formula syntax | Valid Ruby |
| Binary build | PASS |
| sdp usage output | PASS |
| scout --help output | PASS |
| No lab-only binaries | PASS |
| Brew install | true |

**Binary tests:** 4 passed, 0 failed

## Formula surface

The formula at `formula/sdp.rb` installs only the `sdp` binary.
It does NOT install:
- Lab-only binaries: sdp-control, sdp-dispatch, sdp-up
- Experimental binaries: sdp-harness, sdp-a2a, sdp-strataudit
- Research/benchmark binaries: sdp-cascade-replay, sdp-decompose-bench, etc.
- ChangePassport (sdp-pr-gate) — separate product, not yet implemented

Full classification: `docs/reference/maturity-matrix.md`

## What is deferred to actual tap publishing

- Real version from git tag (not `-dryrun` suffix)
- SHA256 from actual GitHub release tarball (not local git archive)
- Tap repository setup (e.g., `homebrew-sdp`)
- CI integration (GoReleaser `brews` section)
- Code signing and notarization
- Bottle (pre-built binary) distribution

## How to run the dry run

```bash
# Full dry run (requires brew + go):
./scripts/homebrew-dry-run.sh

# Skip brew install, just build + test binary:
SKIP_INSTALL=1 ./scripts/homebrew-dry-run.sh

# Validate formula only:
ruby -c formula/sdp.rb
```

## GoReleaser integration note

The `.goreleaser.yml` already has 16 stable binaries configured.
A `brews` section can be added when tap publishing is approved:

```yaml
# Future (tap publishing approved):
brews:
  - name: sdp
    ids:
      - sdp
    tap:
      owner: fall-out-bug
      name: homebrew-sdp
    commit_author:
      name: sdp-bot
      email: bot@sdp.dev
    homepage: "https://github.com/fall-out-bug/sdp_lab"
    description: "SDP Toolkit - governed AI software delivery harness CLI"
    license: "MIT"
    install: |
      bin.install "sdp"
    test: |
      assert_match "usage: sdp <command>", shell_output("#{bin}/sdp 2>&1", 2)
```

This is intentionally NOT added to `.goreleaser.yml` yet.
