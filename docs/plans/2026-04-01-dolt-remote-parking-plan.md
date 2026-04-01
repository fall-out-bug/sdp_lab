# Dolt Remote Parking Plan

Updated: 2026-04-01

## Why This Is Parked

The repo now has:

- `scripts/beads_transport.sh` for normal transport
- `scripts/beads_dolt_remote_bootstrap.sh` for real remote bootstrap
- `origin/beads-backup` as working fallback transport

What it does not have:

- a real shared Dolt remote URL for `sdplab`
- `DOLT_REMOTE_USER`
- `DOLT_REMOTE_PASSWORD`

That means the remaining work is infra provisioning, not more repo-side implementation.

## Current State

- `./scripts/beads_transport.sh status` returns `mode=git-backup`
- `bd dolt remote list` returns `No remotes configured.`
- `origin/beads-backup` exists and is updated as the current fallback transport
- `sdplab-bqc` is blocked on external infra input

## Resume Checklist

When infra is ready:

1. Export credentials in the shell that will own Beads replication:

```bash
export DOLT_REMOTE_USER=...
export DOLT_REMOTE_PASSWORD=...
```

2. Bootstrap the shared remote:

```bash
./scripts/beads_dolt_remote_bootstrap.sh --url https://doltremoteapi.dolthub.com/<org>/<repo> --push
```

3. Verify transport switched:

```bash
./scripts/beads_transport.sh status
bd dolt remote list
bd dolt pull
bd dolt push
```

4. Update the backlog item:

- move `sdplab-bqc` from `blocked` to `in_progress`
- close `00-096-03` only after `mode=dolt-remote` is true and the remote is exercised from the real environment

## Explicit Non-Goals While Parked

- Do not fake a local filesystem remote and call it shared infra
- Do not remove `origin/beads-backup` before real Dolt pull/push is proven
- Do not pretend tracked `.beads/issues.jsonl` is authoritative again
