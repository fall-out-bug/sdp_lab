# Onboarding SDP Into A Repo

> F141-07 · audience: any agent or developer dropping SDP into a new repo.
> Time to first verified install: under 5 minutes.

## 1. Pre-flight

- `git` and `go` 1.26+ on PATH
- macOS, Linux, or WSL (Windows native not supported in v1)
- optional: Claude Code, OpenCode, Codex, or Cursor if you want harness commands

## 2. Install

From your target repo root:

```bash
curl -fsSL https://raw.githubusercontent.com/fall-out-bug/sdp_lab/main/scripts/install.sh | bash
```

What happens: clones `sdp_lab` (shallow) to get the canonical manifest and prompts, builds `./.sdp/bin/sdp` unless explicitly allowed to trust a PATH binary, runs `init --harness=auto` through the installer binary,
detects existing harness dirs (installs all five if none found), writes generated adapters plus `sdp.lock`, verifies the repo-local CLI, and prints the PATH export command.

Override vars: `SDP_HARNESS=claude-code,opencode`, `SDP_TARGET=/path/to/repo`, `SDP_SOURCE_DIR=/path/to/sdp_lab` for local-source testing.

## 3. Verify install

```bash
./.sdp/bin/sdp manifest validate     # manifest well-formed
./.sdp/bin/sdp doctor adapters       # no drift
test -x .sdp/bin/sdp                # repo-local CLI exists
ls .claude/ .opencode/ .codex/ .cursor/   # dirs present
export PATH="$PWD/.sdp/bin:$PATH"
command -v sdp
```

Both `./.sdp/bin/sdp` commands must exit 0. After the `export`, `command -v sdp` must point at this repo's `.sdp/bin/sdp`.

## 4. First useful commands

Start with low-risk repo analysis. `index build` writes a local `.sdp/index.db` cache; the other commands below only inspect the repo unless you pass an output directory.

```bash
./.sdp/bin/sdp scout --format text .
./.sdp/bin/sdp metrics --format markdown .
./.sdp/bin/sdp index build --format text .
./.sdp/bin/sdp spec --format text .
```

Preview delivery setup without changing code:

```bash
./.sdp/bin/sdp build "Add a small feature with tests" --dry-run --format text
./.sdp/bin/sdp bootstrap --dry-run --mode brownfield .
```

## 5. First harness command

| Harness | Command |
|---|---|
| Claude Code | `/build` |
| OpenCode | `opencode run sdp build` |
| Codex CLI | `codex skill build` |
| Cursor | command palette → `build` |
| Pi | `/build` |

All five invoke the same SDP `build` skill contract.

The installer does not configure model keys. Keep credentials in the harness/provider you use.

## 6. Customize

```bash
$EDITOR sdp.manifest.yaml          # edit inventory
./.sdp/bin/sdp generate-adapters --write      # regenerate adapter files
./.sdp/bin/sdp doctor adapters                # verify no drift
git add sdp.manifest.yaml .claude/ .opencode/ .codex/ .cursor/
git commit -m "chore: update SDP adapters"
```

Do not edit harness adapter files directly — `./.sdp/bin/sdp doctor adapters` flags drift.

## 7. Update SDP

Re-run the installer — it pulls latest `sdp_lab` and regenerates adapters:

```bash
curl -fsSL https://raw.githubusercontent.com/fall-out-bug/sdp_lab/main/scripts/install.sh | bash
./.sdp/bin/sdp init --update
```

Or if `command -v sdp` points at this repo's `.sdp/bin/sdp`: `sdp init --update` (keeps existing manifest, regenerates adapters).

Commit `sdp.lock` to pin the version for your team.

## 8. Troubleshooting

| Symptom | Fix |
|---|---|
| `sdp doctor`: adapter diverges from manifest | `./.sdp/bin/sdp generate-adapters --write` then commit |
| `sdp doctor`: orphan file not in manifest | Add entry to `sdp.manifest.yaml` or delete orphan |
| No harness dirs after install | `./.sdp/bin/sdp init --harness=claude-code` (explicit) |
| `sdp` binary not on PATH after install | `export PATH="$PWD/.sdp/bin:$PATH"` from the target repo root |
| `sdp manifest` or `sdp scout` says no such command | You are running an older global `sdp`; use `./.sdp/bin/sdp ...` or check `command -v sdp` after the PATH export |
| Installer finds an old `sdp` binary | Re-run the installer; it ignores PATH binaries by default and falls back to the branch build unless `SDP_TRUST_PATH_SDP=1` |
| `warning: manifest load failed` | Install used empty template; add entries to `sdp.manifest.yaml` |

---

Reference: [`product-surface.md`](../reference/product-surface.md) · [`harness-parity-matrix.md`](../reference/harness-parity-matrix.md) · [`F141 design`](../plans/2026-04-25-f141-multi-harness-install-bootstrap-design.md)
