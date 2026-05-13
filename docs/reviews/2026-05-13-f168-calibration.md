# F168 Persona Calibration

Date: 2026-05-13
Branch: `feature/F168-onboarding-quality-taxonomy`
Mode: read-only persona calibration plus targeted remediation

## Method

Three independent read-only calibration passes reviewed the current branch:

- Developer trying SDP after learning Cursor, Claude Code, and OpenCode.
- CTO/tool buyer deciding whether SDP is worth a narrow pilot.
- Cold-start agent asked to describe the repo and what it can safely do.

Each pass covered zero-knowledge, experienced, and multi-harness variants.

## Verdict

F168 is improved enough to support honest first-run orientation, but not enough
to claim full multi-harness rollout readiness.

What is now clear:

- SDP is framed as a governed delivery layer around existing coding harnesses,
  not as a replacement IDE.
- Claude Code is the stable primary path; OpenCode is experimental with
  `--agent implementer`; Cursor, Codex, and Pi are not equivalent primary
  workers.
- Quality axes preserve `evidence_only`, `not_assessed`, and `cannot_verify`
  instead of fake green.

What remains constrained:

- `sdp quality` is an sdp_lab-local advisory surface because it depends on
  `scripts/quality-metrics.sh`.
- The CTO/buyer path still needs a stronger sample pilot packet before broad
  external rollout.
- The harness parity table is truthful, but skimming users can still overread
  command parity as runtime readiness.

## Findings And Disposition

| Finding | Disposition |
|---|---|
| `sdp quality` was in CLI help but absent from command/product maps. | Fixed in `docs/reference/commands.md` and `docs/reference/product-surface.md`. |
| `sdp quality` default output said some checks were "checked below" although default mode only prints the matrix. | Fixed in `scripts/quality-metrics.sh`; default output now says `--full` is required for coverage/ratio checks. |
| Old operator docs referenced nonexistent `sdp quality all`. | Fixed to `sdp quality --full` in `docs/SDP_OPERATOR_WORKFLOW.md`. |
| Quality gates reference still mixes current Go truth and legacy Python examples. | Narrowed with an explicit legacy examples boundary. Full archival cleanup remains follow-up scope. |
| README exposes product taxonomy before first proof. | Recorded as remaining UX debt; not fixed in this slice to avoid a broad README rewrite. |
| CTO path lacks sample decision packet. | Recorded as next work; this calibration artifact is not a full buyer packet. |
| MCP/install docs still carry stale global PATH-style guidance. | Recorded as follow-up; outside this F168 slice unless onboarding docs get a deeper pass. |

## Persona Results

| Persona | Zero-knowledge | Experienced | Multi-harness |
|---|---|---|---|
| Developer | Understandable if landing on `START_HERE.md` or `QUICKSTART.md`; README remains too taxonomy-heavy. | Value proposition is credible, but the local delivery path still stops near preview. | Runtime readiness is clearly separated from generated adapter parity. |
| CTO/tool buyer | Value partly clear; rollout risk medium without a sample decision packet. | Value clear as delivery governance around existing tools. | Rollout risk clearer than rollout plan; pilot sequencing needs a concrete packet. |
| Cold-start agent | Safe enough to describe repo purpose from canonical docs. | Safe enough to choose orientation vs execution when feature/workstream/bead ownership exists. | Significantly safer: harness strengths and gaps are explicit. |

## Current Evidence

- `go run -tags sqlite_fts5 ./cmd/sdp quality` prints the fast F168 quality-axis matrix.
- Focused tests for `cmd/sdp`, `cmd/sdp-pi-review`, `internal/pireview`,
  `internal/adapters`, `internal/manifest`, and `internal/orchestrate` passed
  during this slice.
- Branch pi-review round 7 returned `ESCALATED`: 0/3 usable model outputs,
  0 P0/P1 findings. Each model artifact contained only
  `Warning: No models match pattern "kimi-coding/k2p6"`, which points to stale
  local pi model settings. This is provider/config degradation, not approval and
  not product evidence.

## Next Work

1. Add a one-page CTO pilot packet with expected artifacts, stop/go criteria,
   and explicit `not_assessed` examples.
2. Decide whether README should route cold readers to proof before taxonomy.
3. Clean MCP/install docs so repo-local and global install guidance are not
   mixed for downstream users.
4. Update the local pi enabled model list so it no longer includes stale
   `kimi-coding/k2p6`, then re-run branch-scope pi-review after the model panel
   returns usable outputs.
