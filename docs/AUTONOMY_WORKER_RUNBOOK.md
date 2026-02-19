# Autonomy Worker Runbook (Private)

## Purpose

`cmd/autonomy-worker` is the first executable step toward self-picking agents.
It selects the next eligible autonomy task, claims it, and prepares protocol artifacts.

## What it does

1. reads Beads issues via `bd list --json`
2. filters tasks:
   - `issue_type=task`
   - `status=open`
   - label includes `autonomy`
3. checks dependencies (ignores `parent-child`, enforces other blockers)
4. picks highest priority oldest task
5. enforces model policy (`glm-5`, `glm-4.7`)
6. updates task to `in_progress`
7. writes:
   - `.sdp/runs/<issue-id>.json`
   - `.sdp/evidence/<issue-id>.json`
   - run packet includes protocol `flow` state
8. appends Beads note with model/branch/paths

## Usage

Dry run:

```bash
go run ./cmd/autonomy-worker --dry-run
```

Debug candidate filtering:

```bash
go run ./cmd/autonomy-worker --dry-run --debug
```

Execute:

```bash
go run ./cmd/autonomy-worker
```

## Expected output

JSON with selected `issue_id`, `title`, `model`, and `branch`.

## Limitations (current)

- does not execute code changes yet
- does not auto-open PR yet
- does not complete full strict evidence content (only creates skeleton)

These are implemented in later Stage A tasks.
