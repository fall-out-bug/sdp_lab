# sdp_lab <-> sdp Workflow

If your real goal is "install SDP into my own repo and start using it", leave this doc and go to [../sdp/docs/QUICKSTART.md](../sdp/docs/QUICKSTART.md). This file is only for contributors working across the private parent repo and the public submodule.

Use this repo as a two-repo workspace:

| Path | Repo | Remote | Typical change |
|------|------|--------|----------------|
| Root, `internal/`, `cmd/`, `docs/` | `sdp_lab` | `https://github.com/fall-out-bug/sdp_lab` | Lab code, research docs, orchestration |
| `sdp/` | `sdp` submodule | `https://github.com/fall-out-bug/sdp.git` | Public protocol artifacts, prompts, hooks, `sdp-plugin` |

Historical note: many workstreams and beads IDs still use `sdp_dev-*` as a legacy label for the root repo. That is history, not a third repo.

## Branch defaults

| Repo | Default branch |
|------|----------------|
| `sdp_lab` | `main` |
| `sdp` | `main` |

## Decide the repo first

- If the file path starts with `sdp/`, you are in the OSS repo.
- If the file path is outside `sdp/`, you are in `sdp_lab`.
- If the request says "protocol", "public CLI", "prompts", "schema", or "hooks", confirm whether it belongs in `sdp`.

## Commit order

1. Commit and push inside `sdp/` first when you changed public protocol files.
2. Return to `sdp_lab`, stage the submodule pointer with `git add sdp`.
3. Commit and push the parent repo update.

## Canonical submodule rule

- `.gitmodules` must point `sdp` at `https://github.com/fall-out-bug/sdp.git`.
- Do not use `../sdp` or another local sibling path as the canonical submodule URL.
- Local sibling clones are fine for manual work, but they must not leak into the tracked submodule config.

## Recovery

If `git submodule status` prints `-<sha> sdp`, the path is missing.

```bash
git submodule sync -- sdp
git submodule update --init --checkout sdp
```

If the printed sha does not exist in `fall-out-bug/sdp`, the parent repo points at a broken commit. Fix it by checking out a valid commit inside `sdp/`, then stage the new gitlink:

```bash
cd sdp
git fetch origin
git checkout origin/main
cd ..
git add sdp
```

## PR flow

- `sdp` changes get their own PR in `fall-out-bug/sdp`.
- `sdp_lab` changes get their own PR in `fall-out-bug/sdp_lab`.
- When one task touches both repos, link the two PRs in their descriptions.
