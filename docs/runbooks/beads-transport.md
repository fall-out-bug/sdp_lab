# Beads Transport

Updated: 2026-04-01

## Current Model

- `bd 0.61.0` removed `bd sync` and the JSONL sync pipeline.
- This repo currently has no configured Dolt remote for primary Beads replication.
- The live Beads database is local Dolt state under `.beads/dolt/`.
- The portable fallback transport is an archival `bd export` snapshot stored on `origin/beads-backup`.

## Repo Rule

Use `scripts/beads_transport.sh` instead of calling `bd sync`.

```bash
./scripts/beads_transport.sh fetch
./scripts/beads_transport.sh export
./scripts/beads_transport.sh status
```

Behavior:

- if a real Dolt remote named `origin` exists, the helper runs `bd dolt pull` / `bd dolt push`
- otherwise it publishes a portable `bd export` snapshot through `origin/beads-backup` with a temporary git worktree

## Startup

After `git pull --rebase`, run:

```bash
./scripts/beads_transport.sh fetch
```

If `origin/beads-backup` exists but no real Dolt remote exists, fetch is still a no-op. The archival branch is not a safe source for local Dolt hydration.

## Shutdown

Before `git push`, run:

```bash
./scripts/beads_transport.sh export
```

This publishes the current portable export to `origin/beads-backup` when no Dolt remote is configured.

## Bootstrap A Real Dolt Remote

Input you need:

- a real Dolt remote URL, for example `https://doltremoteapi.dolthub.com/<org>/<repo>`
- `DOLT_REMOTE_USER`
- `DOLT_REMOTE_PASSWORD`

Bootstrap command:

```bash
export DOLT_REMOTE_USER=...
export DOLT_REMOTE_PASSWORD=...
./scripts/beads_dolt_remote_bootstrap.sh --url https://doltremoteapi.dolthub.com/<org>/<repo> --push
```

What it does:

- configures the Dolt remote under the name `origin`
- optionally pushes the current local `sdplab` database to that remote
- makes `scripts/beads_transport.sh fetch/export` switch from git-backup mode to first-class Dolt push/pull

Current blocker in this repo:

- there is no Dolt remote URL configured
- there are no `DOLT_REMOTE_USER` / `DOLT_REMOTE_PASSWORD` credentials in the current environment

Until those inputs exist, `origin/beads-backup` remains only an off-machine archival path.

## What This Does Not Solve

- It does not provision a real shared Dolt remote for primary replication.
- It does not make tracked `.beads/issues.jsonl` authoritative again.
- It does not clean every historical doc that still mentions `bd sync`.

Those remain separate remediation work unless and until the project provisions an actual Dolt backend.
