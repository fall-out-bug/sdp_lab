Find the highest-priority ready FEATURE in beads (`bd ready --json`), claim it atomically, create a git worktree, then run the full delivery loop:

1. Build all workstreams in parallel subagents (haiku/sonnet)
2. Review in a fresh subagent — repeat until APPROVED with zero findings including P3
3. Run `./scripts/run_go_quality_gates.sh` — must be green before PR
4. Create PR with `gh pr create`
5. Run `/codex:rescue` telling it explicitly to run `./scripts/run_go_quality_gates.sh` and report all test failures AND code findings
6. Fix all findings in subagents, push, repeat until codex reports zero findings and tests pass

Follow `.agents/skills/build.md` Session Bootstrap for worktree creation and checkpoint.
Follow `.agents/skills/delivery-loop.md` for the full loop rules (model policy, compaction recovery, checkpoint updates).

Do not stop to ask questions. If the beads queue is empty, report that and stop.
