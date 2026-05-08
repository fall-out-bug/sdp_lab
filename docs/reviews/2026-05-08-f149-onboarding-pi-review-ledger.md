# F149 Onboarding Pi Review Ledger

Date: 2026-05-08
PR: https://github.com/fall-out-bug/sdp_lab/pull/152
Primary issue: `sdplab-wwkn`
Pi review finding issues: `sdplab-i21n`, `sdplab-m034`

Raw `.sdp/runs/pi-review/*` telemetry was inspected locally and is not committed.

| id | severity | plane | reviewer/source | finding | disposition | evidence |
| --- | --- | --- | --- | --- | --- | --- |
| R1 | P1 | public surface | minimax | Public installer URL may point to a missing `fall-out-bug/sdp` script. | accepted_narrower | `curl -fsSI https://raw.githubusercontent.com/fall-out-bug/sdp/main/scripts/install.sh` returned `HTTP/2 200`; `gh api repos/fall-out-bug/sdp/contents/scripts/install.sh?ref=main` returned `path=scripts/install.sh`. The narrower valid risk is publish drift after this PR; `scripts/sdp-publish.sh --dry-run` includes `scripts/install.sh`, so publish is required after merge. |
| R2 | P2 | docs | kimi | Later onboarding examples still used bare `sdp` and could hit a stale global binary. | accepted_fixed | README, Quickstart, and runbook copy/paste command blocks now use `./.sdp/bin/sdp` until `PATH` is explicitly trusted. |
| R3 | P2 | installer | kimi | Installer capability checks depended on root help-text substrings. | accepted_fixed | `scripts/install.sh` now probes `manifest validate --help` and `scout --help`, then runs functional `manifest validate` and `doctor adapters` checks with explicit failure messages. |
| R4 | P3 | tests | zai | Contract tests stopped at the first missing string. | accepted_fixed | `cmd/sdp/install_onboarding_contract_test.go` now uses subtests for each checked path or string. |
| R5 | P3 | tests | zai | Docker tests used mixed Docker availability guards. | accepted_fixed | Docker sandbox tests now consistently call `skipIfNoDocker(t)` for Docker-dependent cases. |
| R6 | P0 | public surface | minimax | Public `fall-out-bug/sdp` installer can drift from this repo-local installer after the PR merges. | accepted_narrower | This is a post-merge release gate, not a pre-merge source PR blocker. Publishing before merge would still leave the public installer cloning old `sdp_lab` main. Keep `sdplab-m034` open until PR #152 lands, run `scripts/sdp-publish.sh`, verify the public raw installer, then close it. |
| R7 | P2 | installer | zai | `supports_current_init` still uses a help-text grep for opt-in PATH reuse. | accepted_narrower | The precheck is intentionally non-mutating because functional `sdp init` would write target files. `scripts/install.sh` now documents that trade-off inline; functional verification remains on the repo-local binary after install. |
| R8 | P2 | tests | zai | Contract test subtest names used long checked strings. | accepted_fixed | `cmd/sdp/install_onboarding_contract_test.go` now uses stable short subtest names and keeps the checked string in the failure message. |
