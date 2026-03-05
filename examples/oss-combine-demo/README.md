# OSS Combine Demo

This demo provides a reproducible local flow for F069-03:

- create a Beads task
- execute a full task lifecycle (`open -> in_progress -> closed`)
- demonstrate a guard denial and recovery step
- generate an evidence bundle and a human-readable summary

## Run

Dry-run (default, no mutations):

```bash
scripts/demo/oss_combine_run.sh
```

Execute mode:

```bash
scripts/demo/oss_combine_run.sh --execute
```

## Outputs

When run with `--execute`, artifacts are written to:

`examples/oss-combine-demo/artifacts/<UTC-TIMESTAMP>/`

Expected files:

- `demo-auto-attest.json` - attestation envelope
- `demo-auto-attest-report.json` - machine-readable summary report
- `summary.md` - compact demo summary

## Failure Path Included

The script intentionally runs a blocked guard command (`git push --force`) and then recovers with a safe command (`git status`).

## Runtime Guard

The script enforces a hard 30-minute cap and exits non-zero if exceeded.
