# F168 Status Recovery And Calibration

Date: 2026-05-13
Branch: `feature/F168-onboarding-quality-taxonomy`
Mode: recovery after interrupted agent session

## Summary

F168 is partially implemented, not complete.

Implemented enough to treat as branch evidence:

- `00-168-01` taxonomy contract and evidence states.
- `00-168-05` quality-axis verdict schema.
- Most blocking F168 pi-review findings around onboarding truth, harness parity,
  generated OpenCode commands, command symlink overwrite risk, and empty model
  output handling.

Still not complete:

- `00-168-02` onboarding truth audit is improved, but first-run public `sdp`
  versus lab-local `sdp_lab` guidance still needs final consistency review.
- `00-168-03` deterministic quality matrix exists as evidence, but cognitive
  complexity, CRAP, and work-without-spec are not full gates.
- `00-168-04` pi-review hardening is improved, but egress sanitization remains
  pattern-based hardening, not a comprehensive secret scanner.
- `00-168-06` operator report UX exists only as script/reference output, not as
  a first-class `sdp` command.
- `00-168-07` CI/advisory rollout is partial; quality-axis verdicts are not one
  unified CI artifact yet.
- `00-168-08` calibration had no committed artifact before this recovery note.

## Reader-Lens Calibration

| Lens | Zero-knowledge variant | Experienced variant | Multi-harness variant | Status |
|---|---|---|---|---|
| Developer trying SDP | Quickstart explains install, local binary, first smoke commands, and what changes in the repo. | Quickstart maps SDP commands onto harness command syntax. | Quickstart separates Claude Code, OpenCode, Cursor, Codex, and Pi readiness. | `evidence_only` |
| CTO or architect | `START_HERE.md` states the business risk and first-session proof path. | Product surface says SDP is an overlay around existing harnesses, not a new IDE. | Harness parity matrix separates generated file parity from runtime readiness. | `evidence_only` |
| Cold-start agent | Project map, command reference, and agent-skill entry map describe repo purpose and entrypoints. | Skill/command routing is documented for developer requests. | Harness integration docs list strengths, gaps, and failure modes without equivalence claims. | `evidence_only` |

No row is a full `pass` yet because the calibration was reconstructed from
committed docs and local checks, not from fresh independent user walkthroughs.

## Quality-Axis Status

| Axis | Status | Evidence |
|---|---|---|
| Modern Go patterns | `evidence_only` | `scripts/quality-metrics.sh` reports existing Go vet/golangci evidence and disabled modern linter families. |
| CRAP | `not_assessed` | No selected Go CRAP formula/tool. |
| Cognitive complexity | `not_assessed` | Root `.golangci.yml` does not enable `gocognit` or `gocyclo` thresholds. |
| Maintainability Index | `not_assessed` | No selected Go MI formula/tool. |
| Spec drift | `evidence_only` | Protocol/doc consistency tools own this; not one unified verdict yet. |
| Work without spec | `cannot_verify` in this branch check | No checkpoint evidence in diff against `origin/main`; this is not approval. |
| CleanCode | `model_review` | Requires explicit pi-review plane; not independently closed by this recovery. |
| CleanArchitecture | `model_review` | Requires explicit pi-review plane; not independently closed by this recovery. |
| Security | `mixed` | pi-review prompt/egress hardening exists; comprehensive secret scanning is future work. |
| DX | `evidence_only` | Quickstart, runbook, and generated harness docs improved. |
| UX | `not_assessed` | No fresh live UAT transcript committed. |
| Documentation completeness | `evidence_only` | Public docs and command references were updated; strict doc-sync still has historical backlog risk. |

## F168 Finding Disposition

| Bead | Current branch disposition | Rationale |
|---|---|---|
| `sdplab-36kr` | fixed on branch | `.cursor/commands` and `.opencode/commands` symlink overwrite surfaces were removed/replaced with generated files. |
| `sdplab-4ovv` | fixed on branch | Cursor onboarding now names secondary-validator status and first-run commands. |
| `sdplab-72ev` | fixed on branch | Harness parity matrix separates static adapter parity from runtime readiness. |
| `sdplab-zliq` | fixed on branch | OpenCode command generation rewrites Claude skill path references and tests reject leakage. |
| `sdplab-hn0s` | fixed on branch | OpenCode docs now require `--agent implementer` for non-interactive runs. |
| `sdplab-o7v6` | partial | Prompt boundary and pattern redaction exist; comprehensive egress scanning does not. |
| `sdplab-oqi1` | fixed on branch after recovery patch | Empty output still fails; explicit clean review must use `{"verdict":"PASS","findings":[]}`. |

These beads should remain administratively open until the branch is reviewed and
merged under repo policy.

## Verification

Executed during recovery:

```bash
go test ./internal/pireview ./cmd/sdp-pi-review ./internal/adapters ./internal/manifest ./internal/orchestrate
```

Result: passed after the recovery patch.

```bash
./scripts/quality-metrics.sh
```

Result: produced the F168 quality matrix and existing repo-wide coverage debt.
This is evidence, not a clean gate.

```bash
go run ./cmd/sdp-pi-review --scope branch --base origin/main --feature F168 --test-command "go test ./internal/pireview ./cmd/sdp-pi-review ./internal/adapters ./internal/manifest ./internal/orchestrate" --model-timeout 4m --write-verdict --round 5
```

Result: `ESCALATED`, 0/3 model reviewers succeeded, 0 P0/P1 findings. This is
not approval and not useful model-review evidence.

After compact-error hardening, a short timeout degradation run also wrote a
compact `.sdp/review_verdict.json` without raw prompt/diff leakage:

```bash
go run ./cmd/sdp-pi-review --scope branch --base origin/main --feature F168 --test-command "go test ./internal/pireview ./cmd/sdp-pi-review ./internal/adapters ./internal/manifest ./internal/orchestrate" --model-timeout 1s --write-verdict --round 6
```

Result: `ESCALATED`, 0/3 model reviewers succeeded, compact sanitized errors
only.

## Next Work

1. Implement or explicitly scope down `00-168-06` as a first-class command or
   documented script surface for operator quality reports.
2. Decide whether `sdplab-o7v6` closure means narrow F168 hardening or a real
   egress scanner. If the latter, keep it open and split a dedicated workstream.
3. Run fresh calibration with actual developer, CTO, and cold-start agent
   prompts through pi/codex-subagents and record raw summary, not raw telemetry.
4. Run full pre-merge gates: quality gates, manifest validate/parity,
   protocol-check, doc-sync, and final branch-scope pi-review.
