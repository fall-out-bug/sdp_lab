# F168: Onboarding Quality Taxonomy Design

Status: design
Feature bead: sdplab-o8gk
Date: 2026-05-13

## Cold Start

1. This is platform work, not "use SDP in my project" onboarding.
2. Feature owner: F168 / sdplab-o8gk.
3. Canonical docs: this design, `docs/workstreams/backlog/00-168-*.md`, `docs/reference/project-map.md`, `docs/reference/product-surface.md`, and `docs/reference/pi-review-spec.md`.
4. Phase: Discovery-to-Delivery bridge. The first slice is taxonomy/spec; later slices implement checks.
5. Protocol publishing: not required until a workstream changes public protocol artifacts.

## Problem

SDP has several quality and trust mechanisms, but they are not presented or
enforced as one honest onboarding contract. A new user can still hit three
trust failures:

- docs promise a command, mode, or maturity state that the current CLI does not
  actually support;
- review output claims green when the evidence is absent, empty, advisory, or
  provider-dependent;
- quality axes are named as if they are gates, while some are only manual
  review topics and some are not assessed at all.

The target is not "more checks everywhere." The target is transparent first-run
truth: every promised onboarding element has a real command, a real maturity
state, and a visible evidence state.

## Goals

- Define one quality-axis taxonomy for onboarding and PR review.
- Separate deterministic gates, model-review evidence, advisory checks, and
  unimplemented metrics.
- Preserve `not_assessed` and `cannot_verify` instead of converting missing
  evidence into pass/fail.
- Make work-without-spec visible: code changes need a linked workstream with
  acceptance criteria and scope, not just a bead reference.
- Turn onboarding truth into a repeatable audit: docs promises are checked
  against actual CLI/help/code behavior.
- Route model review through independent planes: requirements, CleanCode,
  CleanArchitecture, Security, DX, UX, and docs completeness.
- Calibrate onboarding against explicit reader lenses: new developer, CTO or
  architect, and cold-start agent, each across zero-knowledge, experienced, and
  multi-harness variants.

## Non-Goals

- Do not make model review a substitute for deterministic gates.
- Do not invent a Go Maintainability Index formula to satisfy a numeric target.
- Do not make legacy backlog drift block all work on day one.
- Do not collapse F167 into this feature. F167 remains the security verdict
  gate; F168 consumes it as one plane.
- Do not publish raw `.sdp/runs/pi-review/*` telemetry as normal review
  evidence.

## Axis Contract

| Axis | Initial status | Enforcement target |
|---|---|---|
| Modern Go patterns | evidence_only | Add a ratcheted modern/static-check report. Re-enable linter families by baseline, not all at once. |
| CRAP < 5 | not_assessed | Define Go calculation and coverage source before claiming a gate. Start changed-functions evidence-only. |
| Cognitive Complexity < 15 | not_assessed | Add `gocognit -over 15` evidence on changed Go functions first, repo-wide later. |
| Maintainability Index > 70 | not_assessed | Select and document a Go MI formula/tool before adding any threshold. |
| Spec drift | evidence_only | Emit a single drift report that separates docs, workstream, beads, contract, and CLI drift. |
| Work without spec | evidence_only | Promote to blocking for new code changes after baseline: code needs a workstream with AC and scope. |
| CleanCode | model_review | Add a dedicated pi-review plane and deterministic smell candidates where cheap. |
| CleanArchitecture | model_review | Add package-boundary evidence first; model review remains advisory. |
| Security | mixed | Use F167 for model security verdict; add deterministic scanners separately. |
| DX | evidence_only | Add install/init/doctor/help smoke transcripts and actionable-error checks. |
| UX | not_assessed | Add CLI UAT transcript evidence; do not fake UX with unit tests. |
| Documentation completeness | evidence_only | Changed public surface requires matching docs or explicit docs-not-needed rationale. |

## Evidence States

| State | Meaning |
|---|---|
| `pass` | The declared gate ran over the declared scope and met the threshold. |
| `fail` | The declared gate ran over the declared scope and found a blocking defect. |
| `warn` | The declared check found non-blocking defects. |
| `evidence_only` | Evidence exists, but the repo has not made it a merge-blocking gate. |
| `not_assessed` | The repo has no selected metric/tool/scope for this axis yet. |
| `cannot_verify` | The check is in scope, but the required tool, secret, provider, or artifact is unavailable. |

## Initial Review Findings

These findings seed F168. They came from independent read-only review planes and
must be verified in each implementation slice before fixing.

### Onboarding Truth

- `docs/reference/product-surface.md` over-promises `sdp orchestrate` as
  feature-level orchestration. The `sdp orchestrate` command currently supports
  `once` and `loop`; feature-level orchestration belongs to the separate
  `sdp-orchestrate --feature` binary.
- Top-level `sdp --help` omits current `bootstrap` flags even though onboarding
  asks users to run `bootstrap --dry-run --mode brownfield`.
- Stable-surface lists drift across `project-map.md`, `product-surface.md`, and
  `docs/QUICKSTART.md`.
- Onboarding under-documents useful existing `sdp index` follow-up commands.
- `architect` needs a clear maturity decision: second-run product value or
  operator-only tooling.

### Quality Taxonomy

- Current hard gates are mostly build, test, vet, selected lint, schema/evidence
  checks, and configurable OPA policy.
- `.golangci.yml` disables several linter families; "modern Go" is not an
  explicit gate.
- CRAP, cognitive complexity, Maintainability Index, UX, CleanCode, and
  CleanArchitecture are not deterministic gates today.
- Work-without-spec is only partially covered: a bead reference is weaker than
  a live workstream with acceptance criteria and scope files.

### Security And Review Runner

- `pi-review` can include `.sdp/config.yml`, full touched files, diffs, and
  rule files in prompts sent to external model providers. F168 must require a
  pre-egress sanitizer/blocker before broader model-review rollout.
- Raw `.sdp/runs/pi-review/*` artifacts are written with broad permissions and
  are easy to commit accidentally. They should be ignored by default and written
  as private local telemetry unless explicitly promoted in sanitized form.
- Empty or unparsable model output can currently still count as successful
  review evidence. Missing structured reviewer output must degrade quorum and
  block `APPROVED` unless an explicit override records the risk.
- `--scope auto` can review no useful files when only ignored telemetry dirties
  the tree. Empty effective scope must fail or fall back to the branch diff.
- `scripts/install_kubeopencode_remote.sh` needs host validation and `ssh --`
  handling to prevent SSH option injection.

## Workstream Plan

| WS | Purpose | Owner bead |
|---|---|---|
| 00-168-01 | Taxonomy contract and state semantics | sdplab-f16801 |
| 00-168-02 | Onboarding truth audit and promise map | sdplab-f16802 |
| 00-168-03 | Deterministic quality checks matrix | sdplab-f16803 |
| 00-168-04 | Model review planes over pi-review | sdplab-f16804 |
| 00-168-05 | Evidence schema for quality-axis verdicts | sdplab-f16805 |
| 00-168-06 | Operator-facing quality report UX | sdplab-f16806 |
| 00-168-07 | CI/advisory rollout and Beads findings loop | sdplab-f16807 |
| 00-168-08 | End-to-end onboarding quality calibration run | sdplab-f16808 |

Dependencies: `01 -> {02,03,04,05}; {02,03,04,05} -> 06 -> 07 -> 08`.

## Reader-Lens Calibration

F168 review must not use one generic "new user" persona. The calibration run
must cover this matrix:

| Lens | Zero-knowledge variant | Experienced variant | Multi-harness variant |
|---|---|---|---|
| Developer trying SDP | Can install, verify, and explain what changed without knowing SDP vocabulary | Can map SDP onto Claude/OpenCode/Cursor habits without stale command paths | Can distinguish static adapter parity from runtime dispatch readiness |
| CTO or architect | Can state the business risk reduced and the first-session proof | Can judge whether SDP is an overlay above existing tools, not another IDE | Can decide rollout order across harnesses without assuming equivalence |
| Agent entering cold | Can describe repo purpose, stable surfaces, and limits from canonical docs | Can choose the right skill/command path for a developer request | Can report harness strengths and gaps without overclaiming |

Blocking evidence for F168-08: any row that cannot be tested or answered must be
recorded as `not_assessed` or a follow-up finding. It cannot be hidden inside a
general onboarding pass.

## Acceptance Bar

F168 is not complete until:

- onboarding docs and current CLI agree on the first-run and second-run surface;
- the axis report distinguishes deterministic gates, model review, advisory
  evidence, `not_assessed`, and `cannot_verify`;
- absent reviewer output cannot produce `APPROVED`;
- raw review telemetry is not a normal tracked artifact;
- changed code can be traced to a live workstream/spec or is explicitly flagged
  as work-without-spec;
- the developer, CTO, and cold-start agent reader lenses above have explicit
  test evidence or explicit gaps;
- the final calibration run records unresolved gaps instead of hiding them.
