# Onboarding SDP into a Downstream Repo

> F141-07 · audience: any agent or developer dropping SDP into a new repo.
> Time to working `/build`: under 5 minutes.

## 1. Pre-flight

- `git` and `go` 1.21+ on PATH
- macOS, Linux, or WSL (Windows native not supported in v1)

## 2. Install

From your target repo root:

```bash
curl -fsSL https://raw.githubusercontent.com/fall-out-bug/sdp_lab/main/scripts/install.sh | bash
```

What happens: clones `sdp_lab` (shallow), builds `sdp` binary, runs `sdp init --harness=auto`,
detects existing harness dirs (installs all four if none found), writes `sdp.lock`.

Override vars: `SDP_HARNESS=claude-code,opencode`, `SDP_TARGET=/path/to/repo`.

## 3. Verify install

```bash
sdp manifest validate     # manifest well-formed
sdp doctor adapters       # no drift
ls .claude/ .opencode/ .codex/ .cursor/   # dirs present
```

Both commands must exit 0.

## 4. First slash-command

| Harness | Command |
|---|---|
| Claude Code | `/build` |
| OpenCode | `opencode run sdp build` |
| Codex CLI | `codex skill build` |
| Cursor | command palette → `build` |

All four invoke the same SDP `build` skill contract.

## 5. Customize

```bash
$EDITOR sdp.manifest.yaml          # edit inventory
sdp generate-adapters --write      # regenerate adapter files
sdp doctor adapters                # verify no drift
git add sdp.manifest.yaml .claude/ .opencode/ .codex/ .cursor/
git commit -m "chore: update SDP adapters"
```

Do not edit harness adapter files directly — `sdp doctor` flags drift.

## 6. Update SDP

Re-run the installer — it pulls latest `sdp_lab` and regenerates adapters:

```bash
curl -fsSL https://raw.githubusercontent.com/fall-out-bug/sdp_lab/main/scripts/install.sh | bash
```

Or if `sdp` is on PATH: `sdp init --update` (keeps existing manifest, regenerates adapters).

Commit `sdp.lock` to pin the version for your team.

## 7. Troubleshooting

| Symptom | Fix |
|---|---|
| `sdp doctor`: adapter diverges from manifest | `sdp generate-adapters --write` then commit |
| `sdp doctor`: orphan file not in manifest | Add entry to `sdp.manifest.yaml` or delete orphan |
| No harness dirs after install | `sdp init --harness=claude-code` (explicit) |
| `sdp` binary not on PATH after install | `go build -o /usr/local/bin/sdp ./cmd/sdp` in sdp_lab |
| `warning: manifest load failed` | Install used empty template; add entries to `sdp.manifest.yaml` |

---

Reference: [`harness-parity-matrix.md`](../reference/harness-parity-matrix.md) · [`F141 design`](../plans/2026-04-25-f141-multi-harness-install-bootstrap-design.md)
