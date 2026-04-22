# Deliver Skill Review — Design (Stage 3 Synthesis)

**Date:** 2026-04-22
**Workstream:** sdplab-wjwa
**Method:** @think (4 parallel experts) → synthesis → Socratic dialogue → @llm-council critique → consensus
**Target artifacts:**
- `.claude/commands/deliver.md` (36 lines)
- `.agents/skills/delivery-loop.md` (149 lines)

## 1. Cross-cutting themes

Across four independent reviews (UX / Architecture / SDLC / SRE) the same five themes recur. Anything appearing in ≥3 reviews is treated as **consensus-evident** and progresses to the fix plan without further debate.

| Theme | UX | Architect | SDLC | SRE | Status |
|---|---|---|---|---|---|
| **1. Unbounded loop iterations** (Phase 1 "until zero P3", Phase 3 "until clean") | P0 (abort) | P2 (loopback) | P1 | P0 | consensus-evident |
| **2. Multi-harness contract broken** (skill claims 4 harnesses, command only for Claude) | P1 | P0 | – | – | consensus-evident |
| **3. Checkpoint schema inadequate for recovery** (no version, not atomic, missing cycle/findings/pr fields) | P0 (vocab) | P1 | – | P0 | consensus-evident |
| **4. No precondition / preflight gate** (feature may not exist, no workstreams, CLIs missing) | P0 | – | P0 | P1 | consensus-evident |
| **5. No abort / rollback path** ("Do not stop to ask" + no Ctrl-C runbook) | P0 | – | – | P1 | consensus-evident |

Five additional issues appear in exactly one review but are nevertheless structurally sound and cheap to fix: **boundary inversion** (Architect P0), **codex non-determinism stable-N** (SRE P0), **post-merge cleanup** (SDLC P2), **bead-close atomicity with merge** (SDLC P0), **traceability gate** (SDLC P2).

## 2. Consolidated findings (de-duped, re-ranked)

### P0 — block release

**P0-A. Multi-harness policy violation**
Skill `delivery-loop.md:13-17` declares `compatibility: [claude-code, opencode, cursor, codex]`, but the operator entry-point lives only at `.claude/commands/deliver.md`. Confirmed absent in `prompts/commands/` and in `.codex/`, `.opencode/`, `.cursor/` command dirs. Policy (F127-01) requires cross-harness parity.

**P0-B. Boundary inversion — command duplicates skill logic**
Steps 1–3 in `deliver.md:1-23` (pick feature, identify workstreams, claim + worktree + checkpoint) are domain logic. Skill's entry condition (`delivery-loop.md:27`) already assumes all three are done — so Claude operators get a pre-loop phase that Codex/OpenCode operators will not. Feature-picking heuristics will drift between the command and the skill.

**P0-C. Unbounded loops — runaway token/cost risk**
Phase 1 (`delivery-loop.md:32-42`) says "repeat until APPROVED with zero findings **including P3**", and Phase 3 (`:49-57`) "until codex reports zero findings AND tests pass". Codex is non-deterministic (different findings each run). Flaky tests. No iteration cap. A single feature can silently burn the token budget.

**P0-D. Checkpoint schema is too thin for recovery**
The JSON example at `delivery-loop.md:91-103` omits: `schema_version`, `cycle_number` (per phase), `ws_in_progress`, `findings[]` with id/status, `pr_number` pre-Phase-3, `worktree_path`, `claimed_epic_id`, `last_heartbeat`. Writes are not atomic; parallel subagents can race. On mid-fix compaction the loop cannot tell "patched" from "pending".

**P0-E. No preflight / precondition gate**
Neither the command nor the skill validates: `bd` responsive, `gh auth status`, `codex` CLI, Docker daemon (for quality gates), feature claimed to **this** operator, worktree exists, `docs/workstreams/backlog/FXXX-*.md` present, WS files match bead children. An expired `gh auth` fails **after** build work is done.

### P1 — fix before wider dogfood

**P1-F. No abort / rollback runbook**
`deliver.md:35` ("Do not stop to ask questions") combined with no documented Ctrl-C handler means an aborted session leaves: claimed epic, live subagents, worktree, stale checkpoint. SRE and UX both flag this. Need a `/deliver --abort` or explicit recovery steps in the skill.

**P1-G. No stable-N acceptance for codex non-determinism**
Phase 3 (`:53`) treats the first "clean" codex run as terminal. One lucky run exits. Need: require 2 consecutive clean cycles, dedupe findings across cycles by `file:line:rule` hash, auto-close findings that disappear for 2 cycles as "non-reproducible".

**P1-H. No locking — multi-operator collision**
Nothing prevents two `/deliver` invocations from racing on `.sdp/checkpoint.json`, worktree dirs, or PR branch names. Bead `--claim` is atomic but worktree/checkpoint are not.

**P1-I. No timeouts on subagents / phases / whole loop**
Hung subagent stalls the parallel batch indefinitely. Need per-subagent wallclock (build=20m, review=10m, fix=15m, codex=30m) and a whole-loop budget (e.g. 8h).

**P1-J. Bead-close not atomic with merge**
`delivery-loop.md:28` exit condition is "PR merged". No step defines when each **child** WS bead closes, or whether bead close is ordered against `scripts/beads_transport.sh export` and `git push` of the beads db. If any step in that tail fails, state diverges between repo and tracker.

**P1-K. Hidden design-gap loopback**
`:41-42` lets review spawn `@design` which creates new WS and re-enters build. No max-cycle, no planner sign-off, no `scope_delta` counter. Infinite scope-creep vector.

**P1-L. Multi-harness output: `/codex:rescue` hardcoded**
`:51` and `.claude/commands/deliver.md:32` both hardcode the Claude-only slash command `/codex:rescue`. Operators on other harnesses need a mapped invocation (e.g. direct `codex exec`).

### P2 — fix before GA

**P2-M. No file-overlap declaration on WS**
`:79` says "check for file overlap before dispatching parallel @build" — no mechanism. WS frontmatter doesn't declare `touches:`. Parallel builds can race `go.mod` / shared packages.

**P2-N. Quality gate placement: after PR creation, not before**
`:75` says "never create PR with red tests" but the ordering is unclear; `:47` has gates run after `gh pr create`. Red PRs can hit CI briefly.

**P2-O. No post-merge teardown**
No steps for: worktree removal, branch delete (local+remote), checkpoint archive, beads export push. Cruft accumulates.

**P2-P. No traceability enforcement**
AC IDs in tests, schema changes requiring validator coverage — never checked. Left implicit for reviewer.

**P2-Q. Vocabulary drift — verdict vs findings**
`:100` uses `"verdict": "APPROVED|FINDINGS"` in the checkpoint; `:120` uses "findings count" in output. Inconsistent and confusing on resume.

**P2-R. Hardcoded phases / no extension points**
Phases are prose, not data. Adding a pre-build stage, skipping codex offline, or adding a new review dimension requires editing prose in two files. Should be a declarative YAML list.

**P2-S. Contract drift — skill anchors not tested**
`deliver.md:27` references the anchor "Session Bootstrap" in `build.md` by prose. A rename silently breaks the command. No `scripts/verify_skill_anchors.sh`.

### P3 — nice-to-have

**P3-T. `/codex:rescue` prompt is prose, not structured JSON output**
Parsing codex output is brittle. Require `{tests_passed: bool, findings: [{file, line, severity, rule, msg}]}` to enable P1-G dedup.

**P3-U. Max-parallel-5 is advisory**
`:81` "queue beyond 5" — no primitive. Either implement the queue or soften the wording to "operator discretion".

**P3-V. Observability gap — no heartbeat log**
Operator walks away; no file tells them "stuck vs working". Append-only `.sdp/delivery.log` + `checkpoint.last_heartbeat` fix this.

## 3. Proposed design (consensus direction)

### 3.1 Boundary (P0-A, P0-B, P1-L)

**Decision:** Move domain logic into the skill; keep the command as a three-line dispatcher. Ship the dispatcher once in `prompts/commands/deliver.md` and symlink into `.claude/commands/`, `.codex/commands/`, `.cursor/commands/`, `.opencode/commands/`.

`prompts/commands/deliver.md` (new, ~10 lines):

```md
Invoke `@delivery-loop` with no arguments.

The skill handles: feature selection (`bd ready`), workstream identification,
claim + worktree + checkpoint bootstrap, build→review→fix loop, PR creation,
codex review loop.

Abort with `@delivery-loop --abort` (cleans claim + worktree + checkpoint).
Resume with `@delivery-loop --resume` after compaction.
```

`delivery-loop.md` gains a new **Phase 0: Bootstrap** that absorbs the command's old Steps 1–3, plus a **Phase 0.5: Preflight** (see P0-E).

### 3.2 Preflight gate (P0-E)

New Phase 0.5 block in `delivery-loop.md`, runs before any claim:

```bash
# Binary checks
command -v bd gh go codex || exit 2
gh auth status >/dev/null || { echo "gh not authed"; exit 2; }

# Feature sanity
ws_count=$(ls docs/workstreams/backlog/${FEATURE}-*.md 2>/dev/null | wc -l)
bd_count=$(bd list -n 200 | grep -c "^${FEATURE}-")
[[ $ws_count -eq 0 ]] && { echo "ERR: no workstreams for $FEATURE"; exit 2; }
[[ $ws_count -ne $bd_count ]] && { echo "ERR: WS/bead drift"; exit 2; }

# Epic not already claimed by someone else
current=$(bd show $EPIC_ID -o json | jq -r '.assignee')
[[ -n "$current" && "$current" != "$USER" ]] && { echo "ERR: claimed by $current"; exit 2; }
```

On failure, emit actionable message (not "unknown error") and exit without touching state.

### 3.3 Iteration caps + stable-N (P0-C, P1-G, P1-K)

- **Phase 1 max cycles:** 5. On cap: convert remaining P3 to follow-up beads via `bd create --parent $EPIC`, continue to Phase 2.
- **Phase 2.5 design-gap cycles:** max 2. On cap: halt, require human re-plan at epic level. Add `scope_delta_count` to checkpoint.
- **Phase 3 codex cycles:** max 4, and require **2 consecutive** zero-finding/tests-green cycles to exit. Dedupe findings across cycles by `file:line:rule` hash. Auto-close findings absent for 2 cycles.
- **Whole-loop wallclock budget:** 8h. On cap: halt, mark checkpoint `phase_status: "exhausted"`, post PR comment summarizing unresolved findings.
- **Exponential backoff between cycles:** 30s / 2m / 5m (absorbs transient infra flakes).

### 3.4 Checkpoint schema v2 (P0-D, P2-Q)

Publish `docs/reference/checkpoint-schema.md` and mandate atomic writes via `scripts/sdp-checkpoint-write.sh` (tmp+rename).

```json
{
  "schema_version": 2,
  "skill": "delivery-loop",
  "feature_id": "F134",
  "epic_bead_id": "sdplab-xxx",
  "worktree_path": ".worktrees/F134",
  "pr_number": null,
  "phase": 1,
  "step": "build",
  "phase_status": "running",
  "cycle_number": 2,
  "max_cycles": 5,
  "consecutive_clean_cycles": 0,
  "ws_done": ["00-134-01"],
  "ws_in_progress": ["00-134-02", "00-134-03"],
  "findings": [
    {"id":"F-001","file":"x.go","line":42,"severity":"P2","status":"fixing","hash":"..."}
  ],
  "scope_delta_count": 0,
  "last_heartbeat": "2026-04-22T11:00:00Z",
  "started_at": "2026-04-22T09:00:00Z",
  "deadline": "2026-04-22T17:00:00Z"
}
```

Writes restricted to the orchestrator; subagents return structured output which the orchestrator merges. One field, `findings[].hash = sha1(file:line:rule)`, enables dedup.

### 3.5 Locking (P1-H)

```
.sdp/locks/deliver-${FEATURE}.lock   # flock, PID + hostname + ts
.sdp/checkpoints/${FEATURE}.json     # per-feature, not single-slot
```

Stale lock (>2h, pid dead) → explicit `--force`. Bead claim is already atomic; this extends the guarantee to worktree + checkpoint.

### 3.6 Timeouts (P1-I)

Per-subagent wallclock, enforced by orchestrator:
- build: 20m
- fix: 15m
- review: 10m
- codex: 30m
- whole loop: 8h

On timeout: mark WS failed, record in checkpoint, retry once, then escalate.

### 3.7 Abort / rollback (P1-F)

`@delivery-loop --abort` performs:
1. Kill tracked subagent PIDs (orchestrator maintains list in checkpoint).
2. `bd update $EPIC --status blocked --notes "aborted at phase=$P cycle=$C"` (avoid `--release` / unclaim so the claim history persists).
3. `git stash push -m "delivery-loop abort"` in worktree.
4. `mv .sdp/checkpoints/$FEATURE.json .sdp/archive/aborted/$FEATURE-$TS.json`.
5. Release the lock.
6. Print recovery steps (where stash is, how to resume).

`deliver.md` no longer says "Do not stop to ask questions" unconditionally. Replace with: *"Do not stop for routine fix/rebuild decisions. **Do** stop to escalate: (a) tests fail unrelated to feature code; (b) merge conflicts; (c) ambiguous findings with no clear fix."*

### 3.8 Bead-close atomicity + post-merge teardown (P1-J, P2-O)

New Phase 4 **Closeout**:
1. Confirm merge (`gh pr view $PR --json state`).
2. Batch close children: `bd close $WS1 $WS2 ... --reason "merged via PR#$PR"`.
3. Close epic: `bd close $EPIC`.
4. `scripts/beads_transport.sh export && git push`.
5. `git worktree remove .worktrees/$FEATURE`.
6. `git push origin --delete $BRANCH`.
7. Archive checkpoint.

Any failure in steps 2–4 logs to `.sdp/delivery.log` and surfaces a manual-recovery runbook reference.

### 3.9 Traceability gate (P2-P)

Phase 2.2 (pre-PR):
- For each WS: grep `AC[0-9]+` in touched test files; error if WS declares ACs that appear nowhere.
- For schema changes: require adjacent `_test.go` with `jsonschema.Validate`.
- Gate is advisory at P2 (emits warnings), can be promoted once stable.

### 3.10 Extension points (P2-R)

Convert phases to a YAML block at the top of `delivery-loop.md`:

```yaml
phases:
  - {name: bootstrap,     required: true}
  - {name: preflight,     required: true}
  - {name: build,         required: true, max_cycles: 5}
  - {name: review,        required: true}
  - {name: traceability,  required: false}
  - {name: impact,        required: true}
  - {name: pr,            required: true}
  - {name: codex,         required: true, max_cycles: 4, stable_n: 2}
  - {name: closeout,      required: true}
```

Override via `.sdp/delivery.yaml` (offline mode = disable codex; `contract-gen` mode = inject stage before build).

### 3.11 Contract tests (P2-S)

`scripts/verify_skill_anchors.sh` greps referenced anchors ("Session Bootstrap" in `build.md`, "dimensions" in `review.md`) and fails pre-commit if missing.

## 4. Open questions for Socratic dialogue

Three proposals above are contested or interact in non-obvious ways. Stage 3a (Socratic) will pressure-test them before Stage 3b (@llm-council).

1. **Q1 — Is the boundary fix (§3.1) right, or does collapsing Steps 1–3 into the skill leak harness concerns into a harness-agnostic artifact?** Specifically, `bd ready` output format and `/codex:rescue` are harness-specific; can the skill abstract them cleanly without becoming a mini-harness-runtime itself?

2. **Q2 — Does P3-auto-spawn (§3.3, "convert remaining P3 to follow-up beads") undermine the "Never skip P3" rule, or is it the only way to make the loop actually terminate?** There is a real tension between review strictness and delivery throughput.

3. **Q3 — Is stable-N acceptance for Codex (§3.3) sound, or does it mask regressions?** If Codex finds a real bug on cycle 1 that disappears by cycle 3 (because the offending code was patched), we accept. But if it finds a real bug that disappears because codex is flaky, we also accept — indistinguishable from the call-site.

## 5. Socratic dialogue — round 1

**Q1 dialogue.**
- *Claim:* Move Steps 1–3 into skill.
- *Counter:* `bd ready` output parsing lives in the skill, but `bd` is already harness-agnostic (a binary). Same for `git worktree add`. The only truly harness-specific piece is subagent dispatch: Claude uses `@build` as a subagent invocation, Codex uses `codex exec --subagent`, OpenCode uses `--agent implementer`.
- *Resolution:* The skill abstracts a thin dispatch interface. Per-harness mapping lives in a **single table at the top of the skill**:

  | Action | claude-code | codex | opencode | cursor |
  |---|---|---|---|---|
  | dispatch_subagent | `@<skill>` | `codex exec --skill <skill>` | `opencode --agent <skill>` | `cursor agent --skill <skill>` |
  | run_codex_review | `/codex:rescue "..."` | `codex exec "..."` | `codex exec "..."` | `codex exec "..."` |

  The operator entry-point (`deliver.md`) is then genuinely thin. **Q1 answered: yes, move Steps 1–3; keep harness specifics in a single dispatch table.**

**Q2 dialogue.**
- *Claim:* Cap build loop at 5 cycles, convert remaining P3 to follow-up beads.
- *Counter:* "Never skip P3" is written in bold on line 74. If we allow auto-spin-out at cycle 5, reviewers will notice and start gaming the loop ("I'll raise 10 P3s, force a cycle-5 exit"). Worse: the reviewer is a subagent — a badly-tuned reviewer could auto-create noisy follow-up beads forever.
- *Resolution:* Two-layer policy.
  1. **Inside the cap (cycles 1–4):** "Never skip P3" remains strict. All findings block.
  2. **At the cap (cycle 5):** require **operator confirmation** (first break in the "do not ask" rule — by design). If operator confirms spin-out, beads are created with `parent = epic`, `severity = P3`, `source = delivery-loop-auto-spinout`, and a `bd lint` rule flags features with >5 such spin-outs as quality risks.

  This keeps strictness inside the normal flow and escalates explicitly when termination is genuinely contested. **Q2 answered: cap stands, but auto-spin-out requires human confirmation — not silent.**

**Q3 dialogue.**
- *Claim:* Require 2 consecutive clean codex cycles; auto-close findings absent for 2 cycles as "non-reproducible".
- *Counter:* "Non-reproducible" is often "real bug that happened to pass this time". Masks regressions.
- *Resolution:* Strengthen the rule.
  1. A finding is auto-closed only after **both**: (a) absent from ≥2 consecutive codex cycles **and** (b) the code path it touched has not changed since its last appearance (verified via `git log -p -- <file>`). If code changed, the finding stays open until explicitly fixed.
  2. Auto-closed findings are logged to `.sdp/delivery.log` with their hash; a post-merge `bd lint` rule can re-raise them as follow-up beads if desired.

  **Q3 answered: stable-N stands, but "non-reproducible" requires code-unchanged evidence.**

## 6. Stage 3b input for @llm-council

The Socratic round resolved the three contested items. The remaining consensus proposal (sections 3.1–3.11, with Q1/Q2/Q3 refinements folded in) goes to @llm-council for domain-veto critique. Expected challenges:
- *Architect veto:* "Is the dispatch-table fix a real abstraction or a sugar-coated if-harness-eq?"
- *SRE veto:* "Is an 8h whole-loop budget realistic for large features, or will it force premature aborts?"
- *Engineer veto:* "Will operators actually confirm at cycle 5, or will they reflexively approve and accumulate tech debt?"
- *Pragmatist veto:* "Are we gold-plating a 36-line command into a 200-line skill no one will read?"

Council output will feed §7 (Final consensus) and §8 (Implementation WS breakdown).

## 7. Final consensus (post-council)

All six roles issued vetoes in round 1. Round 2 produced refined positions; five vetoes were sustained with concrete fixes and folded into the final consensus. Two minority positions (Engineer, Technician) remain on record for post-dogfood revisit.

### 7.1 Accepted (ship now)

1. **Boundary fix (§3.1 modified).** Collapse command Steps 1–3 into skill Phase 0. Ship `prompts/commands/deliver.md` as the canonical 10-line dispatcher and symlink into `.claude/commands/`, `.codex/commands/`, `.opencode/commands/`, `.cursor/commands/`. **Per-harness dispatch moves to `scripts/sdp-dispatch.sh`** (not a prose table) — adding a harness = one `case` branch in one script, not editing the skill body.

2. **Preflight gate (§3.2)** — accepted as written.

3. **Iteration caps (§3.3 modified).**
   - Phase-level budgets: build=4h, codex=2h (replaces the single 8h wall budget, which was Story-scale not Feature-scale).
   - Whole-loop cap = 72h — **runaway detector only, not a delivery SLO**.
   - Per-cycle caps stand: build=5 cycles, codex=4 cycles.
   - Finding dedup keyed on `rule + symbol_path + normalized_snippet` (NOT `file:line:rule`, which shifts on inserts above the line).
   - Auto-close of "non-reproducible" codex findings is **downgraded to "operator confirms in Phase 4"** unless/until AST-level unchanged-check lands (Technician dissent logged).

4. **Cycle-5 escape hatch with friction (§3.3 modified).** Operator must **manually paste the deferred-P3 list** (`file:line: description`) into the spin-out bead description. Plain y/N prompt is rejected. Engineer minority prefers eliminating the hatch entirely — revisit after 10 real deliveries: if >30% of features hit the cap, narrow the caps rather than widen the hatch.

5. **Checkpoint schema v2 (§3.4)** — accepted; hash field updated per §7.1(3).

6. **Locking (§3.5), timeouts (§3.6), abort (§3.7), closeout (§3.8)** — accepted as written.

7. **Invariant renamed.** In `delivery-loop.md`, "Never skip P3" becomes **"Resolve or explicitly defer P3 with operator signoff."** Rule semantics honesty (Philosopher).

### 7.2 Deferred to follow-up beads (Pragmatist cut)

These were in the synthesis but not judged urgent; shipping them now would gold-plate a 36-line command into a 400-line skill with no triggering pain.

- **§3.9 Traceability gate** (AC-ID grep + schema-validator coverage check).
- **§3.10 YAML phase declarations + `.sdp/delivery.yaml` override.**
- **§3.11 Skill-anchor contract verifier (`scripts/verify_skill_anchors.sh`).**

### 7.3 Minority positions (logged for revisit)

- **Technician:** if AST-level unchanged-check is skipped at v1, auto-close should be labeled *"manual Phase-4 triage"*, not *"auto-close"* — naming sets expectations.
- **Engineer:** prefers eliminating the cycle-5 escape hatch entirely (halt + manual triage) over the friction compromise. Revisit trigger: >30% of features hit the cap.
- **Architect:** if a fifth harness lands within 3 months, consider converting `scripts/sdp-dispatch.sh` to declarative TOML loaded by the skill.

## 8. Implementation breakdown (WS)

| WS | File(s) | Scope |
|---|---|---|
| 01 | `prompts/commands/deliver.md` (new), `.claude/commands/deliver.md` (replace with symlink), `.codex/commands/` `.opencode/commands/` `.cursor/commands/` (create + symlink) | Shared 10-line dispatcher |
| 02 | `scripts/sdp-dispatch.sh` (new) | Per-harness subagent + codex invocation |
| 03 | `.agents/skills/delivery-loop.md` | Add Phase 0 (bootstrap), 0.5 (preflight), 4 (closeout). Update caps, stable-N, timeouts, abort, invariant rename |
| 04 | `.agents/skills/delivery-loop.md` + `docs/reference/checkpoint-schema.md` (new) | Checkpoint v2 schema + atomic-write helper `scripts/sdp-checkpoint-write.sh` |
| 05 | `.agents/skills/delivery-loop.md` | Lock + multi-operator safety (`.sdp/locks/`, `.sdp/checkpoints/${FEATURE}.json`) |
| 06 | `docs/plans/2026-04-22-deliver-skill-review-design.md` (this file) | Design closeout |

Deferred (new beads): §3.9, §3.10, §3.11.

## 9. Signoff

- **@think Stage 3 synthesis:** complete (§§1–6).
- **Socratic round 1:** 3 questions, 3 resolutions (§5).
- **@llm-council round 1+2:** 6 vetoes, 5 sustained and folded, 1 withdrawn (§7).
- **Consensus:** reached. 7 accepted items + 3 deferred + 3 minority positions.

Proceed to implementation of WS 01–06 (in this session).
