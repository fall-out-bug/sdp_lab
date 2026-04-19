Autonomous full-graph delivery. Traverse every open beads issue in topological+priority order, build, review, fix, and gate until the backlog is empty.

---

## Termination

Stop when `bd list --status=open` returns zero issues.
Hard limit: MAX_FIX_CYCLES=100 across the entire run. If hit, stop and report the cycle count and remaining open issues.

---

## Phase 0 — Graph snapshot

```bash
bd list --status=open -n 500    # full open set
bd blocked                       # issues with unresolved blockers
```

Build a work queue ordered by:
1. Topological layer (dependencies satisfied first — use `bd dep` / `bd blocked`)
2. Priority within each layer (P0 → P4)

Re-snapshot the queue after every closed issue — new beads created during fix cycles must enter the queue.

---

## Phase 1 — Parallel build (max 4–5 concurrent worktrees)

For each batch of up to 5 independent ready issues (no shared file scope):

**Per issue:**
1. `bd update <id> --claim`
2. Create worktree per `.agents/skills/build.md` Session Bootstrap
3. Write `.sdp/checkpoint.json` with issue id, branch, worktree path, step=build
4. Dispatch `@build` subagent (haiku for leaf tasks, sonnet for epics) with:
   - The issue AC from `bd show <id>`
   - The WS file content from `docs/workstreams/backlog/00-XXX-YY.md` (if exists)
   - Explicit: "follow docs/reference/go-patterns.md, write tests first"
5. After build: run `./scripts/run_go_quality_gates.sh` in the worktree
   - If red → dispatch `@fix` subagent (haiku), re-run gates, max 3 retries
   - If still red after 3 → create new P0 beads issue, skip this issue, continue

**Bugs:** same flow — claim, worktree, `@build` (haiku), gates.

**Independence check before parallel dispatch:** verify no overlapping file paths across the batch. Overlapping issues → sequential.

---

## Phase 2 — Per-issue review

After each issue build passes gates, dispatch `@review` subagent (sonnet, fresh context):

```
Review branch <branch> against main.
Dimensions: code + impact (always). Add security if auth/secrets touched. Add architecture if >3 packages changed.
Severity threshold: block on P1+, warn on P2, record P3.
```

**On findings:**
- P1/P2 finding → `bd create --title="fix: <finding>" --type=bug --priority=0` → add to front of queue → fix before continuing
- P3 finding → `bd create --title="nit: <finding>" --type=task --priority=3` → add to queue, continue

**On APPROVED:** commit + push branch. Do NOT merge yet — accumulate until epic/feature gate.

---

## Phase 3 — Epic/Feature gate (triggers when all leaf issues of an epic are APPROVED)

Detect: `bd list | grep "F<N>-" | all closed/approved`.

Run in a dedicated fresh subagent (sonnet):

### 3a. Regression
```bash
./scripts/run_go_quality_gates.sh    # full suite on merged state
go test ./... -count=1 -race         # race detector
```
Any failure → P0 bug beads → front of queue.

### 3b. Happy path verification
For each file in `docs/happy-paths/`:
- Read the happy path steps
- Execute each CLI step against the built binary
- Verify expected output matches
- Any deviation → P0 bug beads

### 3c. E2E smoke (if `scripts/e2e_*.sh` or `scripts/smoke*.sh` exist)
```bash
ls scripts/e2e_*.sh scripts/smoke*.sh 2>/dev/null | xargs -I{} bash {}
```
Failures → P1 bug beads.

### 3d. Documentation alignment
Dispatch `@review --dimension reality` subagent:
```
Check: does code match docs/reference/, docs/phases/, AGENTS.md?
Find: undocumented exported symbols, stale CLI flag docs, broken cross-references.
```
Gaps → P2 task beads ("docs: <gap>").

### 3e. If gate passes → create PR for the epic
```bash
gh pr create --title "feat(F<N>): <epic title>" --body "..."
```
Then run codex:rescue:
```
/codex:rescue "Review PR #<N>. Run ./scripts/run_go_quality_gates.sh first. Report all test failures AND all code findings. Do not skip tests."
```
Fix codex findings in subagents → push → repeat until codex: zero findings + tests pass.

---

## Phase 4 — Fix priority override

Any P0 bug created during the run goes to the FRONT of the work queue, ahead of all in-progress batches. Pause current batch if needed (finish current issue, then pivot).

---

## Checkpointing

Update `.sdp/checkpoint.json` after every state change:
```json
{
  "skill": "sweep",
  "cycle": <N>,
  "queue_snapshot": ["id1", "id2", ...],
  "in_progress": ["id3", "id4"],
  "completed_this_run": ["id5"],
  "fix_cycles_used": <N>,
  "phase": "build|review|gate|fix"
}
```

On recovery ("Continue", "продолжай", session restart):
1. `cat .sdp/checkpoint.json`
2. Re-snapshot open issues from beads
3. Resume from `phase` field — skip already-completed issues

---

## Progress report (after each closed issue)

```
[sweep] ✓ sdplab-xxx — F134-01 (build+review: APPROVED)
        Queue: 12 remaining | In-flight: 4 | Fix cycles: 3/100 | Epics gated: 1
```

## Final report

```
[sweep] DONE
  Closed: N issues
  PRs created: N
  Fix cycles used: N/100
  Bugs created during run: N
  Docs gaps found: N
```

---

## References

- `.agents/skills/build.md` — Session Bootstrap, compaction recovery
- `.agents/skills/delivery-loop.md` — build/review loop rules, model policy
- `.agents/skills/review.md` — dimensions and severity
- `docs/happy-paths/` — happy path scripts
- `scripts/run_go_quality_gates.sh` — quality gate script
- `AGENTS.md` — beads workflow, repo topology
