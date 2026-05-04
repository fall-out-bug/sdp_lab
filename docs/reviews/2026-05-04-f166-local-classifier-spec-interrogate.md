# F166 Local Classifier Spec Interrogate

Date: 2026-05-04
Feature: F166
Workstream: 00-166-06
Mode: `pi-review` plus Socratic review

## Verdict

The local chunked classifier spec is ready for implementation planning.

- Final `pi-review`: APPROVED, 0 P0, 0 P1, 12 advisory findings.
- Provider coverage degraded: Zai and MiniMax succeeded; Kimi failed in the final
  run. This is accepted for design hardening evidence, not as a merge gate.
- Socratic judge: PASS, 12/12 critic questions resolved, no new contradictions,
  no scope creep.

## Pi Review

Final command:

```bash
go run ./cmd/sdp-pi-review --scope working-tree --feature F166 --test-command "go test ./internal/workstream ./internal/pireview ./cmd/sdp-pi-review" --model-timeout 4m --write-verdict --round 3
```

Final result:

- Run: `pireview-8cbd0ae9e78f-1777889148973`
- Scope: working tree, 5 files reviewed
- Verdict: `APPROVED`
- Findings: 0 P0, 0 P1, 12 total advisory findings
- Models: 2/3 succeeded

Applied findings across the loop:

- Changed contradictory classifier JSON example from `allow` plus
  `prompt_injection` spans to `needs_review`.
- Added `classifier_advisory_allowed` for demo-mode input classifier warnings.
- Synced F167 summary wording in `docs/workstreams/INDEX.md`.
- Added the F166 detailed workstream table.
- Added missing `00-166-01` through `00-166-04` Beads mappings.
- Unchecked `00-166-06` acceptance criteria so design work does not claim
  implementation/test completion.
- Added loopback-only trusted-local endpoint boundary.
- Added classifier prompt-injection hardening, strict schema parsing, and
  malformed-output failure behavior.
- Added boundary-window chunk classification and explicit cross-chunk known gap.
- Added bounded concurrency, total timeout, chunk cap, deduped overlap spans, and
  partial-failure policy.
- Added `GuardEvent` classifier-field requirement and redacted excerpt cap.

Not applied:

- Raw `.sdp/runs/pi-review/*` telemetry is not committed.
- `.sdp/review_verdict.json` is treated as transient for this docs pass because
  the durable review record is this compact report plus evidence JSON.

## Socratic Review

Critic provider: `zai/glm-5.1`
Judge provider: `minimax/MiniMax-M2.7`

The critic raised 12 questions:

| ID | Severity | Topic | Resolution |
|---|---|---|---|
| Q1 | blocking | Local classifier trust boundary | Resolved: local endpoint is trusted loopback-only compute for MVP; remote classifier out of scope. |
| Q2 | blocking | Classifier prompt injection | Resolved: chunks are wrapped as untrusted data; tools disabled; strict JSON schema; malformed output fails per policy. |
| Q3 | blocking | Cross-chunk payloads | Resolved with limitation: adjacent boundary-window pass required; arbitrary non-adjacent reassembly is known gap. |
| Q4 | blocking | Deterministic tests | Resolved: CI uses fake endpoints and stubbed classifier responses; live accuracy belongs to eval corpus. |
| Q5 | major | Endpoint unreachable behavior | Resolved: strict mode fail-closed; demo mode deterministic-only advisory if scanner is clean. |
| Q6 | major | Measured value proposition | Resolved: classifier remains opt-in until corpus measures scanner gap and classifier TP/FP. |
| Q7 | major | Model schema reliability | Resolved: known-good presets required; poor schema adherence is config failure in strict mode. |
| Q8 | major | Valid block semantics | Resolved: reducer validates schema/enums/confidence/spans/policy combinations. |
| Q9 | major | False-positive observability | Resolved: audit stores safe metadata, redacted excerpts, hashes; full replay uses original local request. |
| Q10 | major | Latency budget | Resolved: bounded parallel chunks and total classifier timeout. |
| Q11 | minor | `benign_security_fixture` semantics | Resolved: informational; may relax low-severity only by policy, never high severity. |
| Q12 | major | Confidence threshold | Resolved: default `0.75`, policy configurable, explicitly uncalibrated until corpus evaluation. |

Judge result: PASS.

## Follow-Up

Implementation should start from `00-166-06` with tests first. Do not wire this
as default production behavior until the classifier evaluation corpus exists.
