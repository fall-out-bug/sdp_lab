# F168 CTO Pilot Packet

Date: 2026-05-14
Branch: `feature/F168-onboarding-quality-taxonomy`
Audience: CTO or engineering leader evaluating SDP without a long tooling dig

## Decision

Run SDP as a narrow delivery-governance pilot, not as a broad multi-harness
rollout.

The pilot should answer one question:

> Does SDP make AI-assisted work easier to trust by exposing scope, evidence,
> missing checks, and follow-up findings?

It should not try to prove autonomous delivery across every harness.

## Pilot Shape

Use one existing repo and one contained change. The change should have a real
acceptance criterion, a reviewable diff, and a known rollback path.

Run:

```bash
./.sdp/bin/sdp scout --format text .
./.sdp/bin/sdp metrics --format text .
./.sdp/bin/sdp doctor
./.sdp/bin/sdp quality
```

Inside `sdp_lab`, `sdp quality` is available as an advisory quality-axis
surface. Outside `sdp_lab`, treat it as repo-local until the downstream repo has
the same quality script support.

## Expected Packet

The pilot output should include:

- repo map and dependency/process signals from `scout` and `metrics`
- one explicit scope contract or workstream
- deterministic gate evidence, including failures
- a compact review verdict when model review is available
- explicit `not_assessed` entries for metrics with no selected tool
- explicit `cannot_verify` entries for missing providers, missing artifacts, or
  unavailable local commands
- follow-up findings instead of silent TODOs

## Stop/Go Criteria

Go forward when:

- a developer can see what SDP checked and what it did not check
- a reviewer can reproduce the deterministic evidence
- missing model providers or tools are shown as `cannot_verify` or
  `not_assessed`, not as green
- the next action is a concrete issue, workstream, or PR finding

Stop or narrow the rollout when:

- the team cannot explain why a check passed
- runtime harness readiness is inferred from generated command parity
- raw model telemetry becomes the durable artifact instead of a compact verdict
- the process requires everyone to learn every SDP surface before the first
  useful result

## Harness Position

Claude Code is the stable primary harness. OpenCode is experimental and should
use `--agent implementer` for non-interactive work. Cursor, Codex, and Pi are
useful validation/manual-assist surfaces unless their runtime readiness row says
otherwise.

Generated adapter parity proves that SDP can render files for harnesses from one
manifest. It does not prove that each harness is ready for autonomous dispatch.

## Current F168 Evidence

- `sdp quality` prints the deterministic quality-axis matrix and next actions
  for each axis.
- Cognitive complexity, CRAP, and Maintainability Index remain
  `not_assessed`.
- Work-without-spec is `cannot_verify` outside checkpoint/PR evidence.
- Pi-review compact verdicts preserve provider/model degradation instead of
  approving empty or missing output.
- Raw `.sdp/runs/pi-review/*` telemetry stays local and untracked by default.

## Current Non-Claims

F168 does not claim:

- all quality metrics are implemented
- model review is a merge approval
- every harness has equal runtime readiness
- `sdp quality` is portable to every downstream repo without repo-local quality
  script support
- provider failures are product approval
