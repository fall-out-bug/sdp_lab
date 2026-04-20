## Step 1 — Pick the feature

Run: `bd ready -n 50`

From the output, find the highest-priority issue where ALL of these are true:
- type is `[epic]` or `[feature]` (shown in brackets after the issue ID)
- NOT type `[bug]`
- NOT a leaf workstream task (leaf tasks have ` ← F` in the title)

If no epic/feature is ready, stop and report: "No ready features. Ready bugs: <list>."

## Step 2 — Identify workstreams

From the feature number FXXX (e.g. F134):
1. `bd list -n 200 | grep "F134-"` — find all leaf workstream beads issues for this feature
2. `ls docs/workstreams/backlog/ | grep "^00-134-"` — find all WS files
3. Cross-reference: every WS file must have a corresponding open beads issue

## Step 3 — Claim and create worktree

1. `bd update <epic-id> --claim` — claim the FEATURE (the epic), not a leaf task
2. Create worktree per `.agents/skills/build.md` Session Bootstrap
3. Write `.sdp/checkpoint.json`

## Step 4 — Delivery loop

Follow `.agents/skills/delivery-loop.md` exactly:
- Build all WS in parallel subagents (haiku/sonnet, one per WS file)
- Review in fresh subagent — repeat until APPROVED with zero findings including P3
- `./scripts/run_go_quality_gates.sh` green before PR
- `gh pr create`
- `/codex:rescue` with explicit instruction to run `./scripts/run_go_quality_gates.sh` and report all test failures
- Fix → push → repeat until codex: zero findings + tests pass

Do not stop to ask questions at any step.
