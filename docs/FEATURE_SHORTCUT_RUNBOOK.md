# /feature Shortcut Runbook

Goal: create a coding task and return a PR with one command.

## Command

```bash
./scripts/feature_to_pr.sh \
  --host fall_out_bug@192.168.50.219 \
  --port 2222 \
  --title "Classify k8s manifest changes as high risk" \
  --workstream policy-k8s-risk-high
```

## What it does

1. Creates a Beads `task` with strict-autonomy labels.
2. Triggers in-cluster worker+reviewer orchestration.
3. Waits for terminal state and prints the PR URL.

## Current supported workstreams

- `policy-k8s-risk-high`
- `model-chain-default-fallback`

## Notes

- This is the implementation step toward the chat-level `/feature` UX.
- The orchestrator remains the source of truth; operator work is scheduled later.
