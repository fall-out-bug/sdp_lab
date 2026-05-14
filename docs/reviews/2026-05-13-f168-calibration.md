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
- The CTO/buyer path now has a sample pilot packet, but it is still a narrow
  pilot packet rather than broad rollout approval.
- The harness parity table is truthful, but skimming users can still overread
  command parity as runtime readiness.

## Findings And Disposition

| Finding | Disposition |
|---|---|
| `sdp quality` was in CLI help but absent from command/product maps. | Fixed in `docs/reference/commands.md` and `docs/reference/product-surface.md`. |
| `sdp quality` default output said some checks were "checked below" although default mode only prints the matrix. | Fixed in `scripts/quality-metrics.sh`; default output now says `--full` is required for coverage/ratio checks. |
| Old operator docs referenced nonexistent `sdp quality all`. | Fixed to `sdp quality --full` in `docs/SDP_OPERATOR_WORKFLOW.md`. |
| Quality gates reference still mixes current Go truth and legacy Python examples. | Narrowed with an explicit legacy examples boundary. Full archival cleanup remains follow-up scope. |
| README exposes product taxonomy before first proof. | Narrowed by adding a `First Proof` block before install. A broader README rewrite remains out of scope. |
| CTO path lacks sample decision packet. | Fixed by `docs/reviews/2026-05-14-f168-cto-pilot-packet.md`. |
| MCP/install docs still carry stale global PATH-style guidance. | Narrowed by adding a first-run note that routes cold downstream users to Quickstart and keeps MCP setup scoped to MCP wiring. |

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
- Branch pi-review round 9 returned `ESCALATED`: 0/3 usable model outputs,
  0 P0/P1 findings, 168 files reviewed. The stale local
  `kimi-coding/k2p6` setting was corrected before this run; the remaining
  failures are live provider/runtime degradation and empty model output, not
  approval and not product evidence.

## Next Work

1. Decide whether README should receive a broader proof-before-taxonomy rewrite.
2. Promote `sdp quality` beyond sdp_lab-local script support before advertising
   it as a portable downstream Toolkit command.
3. Replace narrow pi-review prompt/egress hardening with a comprehensive secret
   scanner only if that becomes the acceptance threshold for a future feature.
4. Re-run branch-scope pi-review after provider credentials/routes return usable
   model outputs.
