# Beads Transport

Updated: 2026-04-01

## Current Model

- `bd 0.61.0` removed `bd sync` and the JSONL sync pipeline.
- This repo currently has no configured Dolt remote for primary Beads replication.
- The live Beads database is local Dolt state under `.beads/dolt/`.
- The portable recovery snapshot is exported under `.beads/backup/`.

## Repo Rule

Use `scripts/beads_transport.sh` instead of calling `bd sync`.

```bash
./scripts/beads_transport.sh fetch
./scripts/beads_transport.sh export
./scripts/beads_transport.sh status
```

Behavior:

- if a real Dolt remote exists, the helper runs `bd dolt pull` / `bd dolt push`
- otherwise it uses `bd backup fetch-git` / `bd backup export-git` against `origin/beads-backup`

## Startup

After `git pull --rebase`, run:

```bash
./scripts/beads_transport.sh fetch
```

If `origin/beads-backup` does not exist yet, fetch is a no-op.

## Shutdown

Before `git push`, run:

```bash
./scripts/beads_transport.sh export
```

This publishes the current `.beads/backup/` snapshot to `origin/beads-backup` when no Dolt remote is configured.

## What This Does Not Solve

- It does not provision a real shared Dolt remote for primary replication.
- It does not make tracked `.beads/issues.jsonl` authoritative again.
- It does not clean every historical doc that still mentions `bd sync`.

Those remain separate remediation work unless and until the project provisions an actual Dolt backend.
