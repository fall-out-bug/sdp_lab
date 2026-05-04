# F167 Security Verdict Gate Design

Status: draft accepted for workstream scaffolding
Date: 2026-05-04
Feature: F167
Beads: sdplab-xe5c

## Problem

Day 14 asks for a security step in an execution loop after generation/tests and
before commit. In SDP, the literal boundary is not one `git commit` call. The
real product boundary is commit/promotion-readiness in Operator Mode: code has
been generated, tests have passed, and the system is about to present work as
safe to advance.

F167 adds a runtime security verdict gate at that boundary.

This is not a replacement for F164. F164 measures prompt-injection risk through
corpus, evals, and prompt-surface hardening. F167 turns the Day-14 challenge into
a delivery control: gateway-backed security review plus deterministic gate
semantics.

## Goals

- Run a security review after tests are green and before commit/promotion-ready
  state.
- Route every LLM call used by the gate through the existing gateway path.
- Sanitize model-bound prompt content for secrets, tokens, and high-confidence
  PII before provider egress.
- Block Critical/High findings and return actionable feedback to Build.
- Allow Medium/Low findings with warning evidence.
- Produce a Day-14 evidence pack with three unsafe task scenarios and logs.

## Non-Goals

- Do not add a new standalone toy loop.
- Do not claim blanket security hardening for all SDP surfaces.
- Do not make normal CI depend on live provider availability.
- Do not intercept every manual `git commit`; protect SDP's own promotion gate.
- Do not duplicate F164's prompt-injection corpus or MCP write-tool policy work.

## Relationship To F164

F167 consumes F164's trust-boundary language and prompt-injection test taxonomy
where it affects security-review prompts. It does not extend the F164 corpus,
MCP write-tool policy, or prompt-bundle supply-chain checks.

For F167, untrusted diff, file content, Beads text, and evidence snippets must be
delimited as data before reviewer invocation. The LLM reviewer is not a trusted
security boundary. Its output becomes one input to a deterministic gate that also
checks sanitizer status, test evidence, schema validity, and severity mapping.

The interface to F164 is document-level in this feature: F167 references the
named F164 attack classes and trust-boundary definitions. No generated F164 data
contract is consumed in F167. If F164 later ships a machine-readable taxonomy,
binding F167 to that taxonomy is a separate follow-up.

## User Experience

The operator should see one compact result:

- `blocked`: Critical/High finding. Output includes file, line, issue, and fix
  prompt for the next Build pass.
- `warn`: Medium/Low only. Output names warning count and evidence path.
- `pass`: no findings and prompt sanitation clean or redacted-only.
- `escalated`: security reviewer unavailable, malformed verdict, or evidence
  missing.

This keeps the loop understandable. Security is a gate, not another review
ceremony the operator must manually stitch together.

Escalation uses the existing human-gate pattern. The operator-visible actions
are:

- retry the security review after transient provider failure
- roll back to Build with the escalation reason
- approve with recorded risk only when policy allows a manual override
- stop the session

Escalation is non-passing by default, but it must not be terminal unless the
operator chooses stop or policy disallows override.

## Architecture

F167 introduces a small `SecurityVerdictGate` instead of a new FSM phase. The
gate evaluates:

1. test evidence from the build path
2. changed files/diff/context packet
3. gateway sanitation evidence
4. security reviewer verdict

The concrete insertion point is the Build phase completion path. When the agent
signals completion for `RoleBuild`, SDP evaluates accumulated tool evidence. If
test evidence is not `passed`, F167 does not call the security reviewer and the
normal build gate remains responsible for blocking. If test evidence is `passed`,
F167 runs before the session can transition from Build toward Review or any
commit/promotion-ready state derived from that transition.

This means `promotion-ready` for F167 is not a new Beads status. It is the
agentloop pre-transition boundary after Build has produced passing test evidence
and before SDP records the work as ready for the next delivery stage. In
operator terms: the work is not allowed to leave Build as review-ready until
the security gate records `pass` or `warn`, or a human-approved escalation
override is recorded.

The gate decision maps severities as:

| Challenge term | SDP priority | Gate behavior |
|---|---|---|
| Critical | P0 | block |
| High | P1 | block |
| Medium | P2 | warn and allow |
| Low | P3 | warn and allow |

Provider failure, malformed model output, missing test evidence, or sanitizer
failure is not a pass. It escalates.

Deterministic gate inputs are:

- test evidence status: only `passed` allows reviewer invocation
- sanitizer status: `blocked_before_provider` blocks or escalates before provider egress
- sanitizer evidence validity: missing or invalid evidence escalates
- reviewer output schema validity: malformed output escalates
- severity mapping: Critical/High blocks, Medium/Low warns, clean passes

The reviewer can discover issues, but it cannot bypass deterministic checks.
`blocked_before_provider` can stop the gate even if no reviewer verdict exists.

The gate is synchronous for the Build completion transition. If a future runtime
makes security review asynchronous, the transition must remain pending until the
security verdict or escalation decision is recorded.

## Gateway And Redaction

The gate must not send raw known secrets or high-confidence PII to providers.
F167 should reuse the existing architect security filters where practical, but
the gateway contract must be local to model egress so future LLM call sites can
share it.

For F167, "high-confidence" means deterministic matches from the existing
secret filter patterns plus explicit auth-header/private-key/connection-string
patterns. PII redaction uses the default PII scrubber classes with conservative
defaults; URLs and generic high-entropy API-key guesses are not blocking unless
they match a known secret/token pattern.

The reusable surface is a sanitizer interface at model egress. F167 wires it for
the security reviewer path only. Broader rollout to historical LLM call sites is
out of scope.

Evidence records counts and classes only:

- `passed_clean`
- `redacted`
- `blocked_before_provider`
- `sanitizer_error`

Evidence must not store the raw secret, token, or PII value.

Actionable findings may store file path, line range, secret class, hash, and
suggested remediation. They must not store the raw matched value or full line
content when the line contains a redacted secret. Hashes use SHA-256 to match
existing evidence conventions.

Evidence should include pattern-class reason codes such as `private_key`,
`auth_header`, `connection_string`, `known_token`, or `pii_email`. It must not
include provider-specific regex internals or raw matched values.

## Security Prompt

The prompt is SDP/Go-oriented, not mobile-stack boilerplate. It must check:

- hardcoded secrets and tokens
- PII in logs or model-facing evidence
- HTTP or insecure endpoint usage where HTTPS is expected
- shell command construction and injection risk
- path traversal and worktree escape
- missing input validation
- unsafe persistence of auth material
- prompt-injection-sensitive trust boundary violations when diff/content tries to
  influence the reviewer

The prompt output must be structured. Free-form model prose is not sufficient
for a gate.

Seed output shape:

```json
{
  "verdict": "pass|warn|block|escalated",
  "findings": [
    {
      "severity": "critical|high|medium|low",
      "priority": "P0|P1|P2|P3",
      "file": "path",
      "start_line": 1,
      "end_line": 1,
      "title": "short finding",
      "rationale": "why this is exploitable or risky",
      "suggested_fix": "specific fix guidance",
      "evidence_ref": "sanitized evidence id"
    }
  ]
}
```

Reviewer stubs in tests must return fixed structured verdict fixtures, including
malformed output and provider failure fixtures. Gate tests must not depend on
live provider behavior.

Validation ownership:

- `00-167-01` owns the verdict schema and parser contract.
- `00-167-03` owns invoking that parser in the gate and escalating malformed
  output.
- Structurally valid but semantically weak model output is handled by severity
  mapping and demo ground-truth tests; it is not treated as malformed.

## Day-14 Demo Scenarios

The demo pack should run three sanitized tasks:

1. "save auth token" -> unsafe storage or hardcoded token risk
2. "log all requests" -> PII/token leakage in logs
3. "call an API" -> insecure HTTP or missing input validation

For each scenario, report:

- what the gateway blocked or redacted before provider egress
- what the security reviewer found
- what both layers missed
- final gate decision

Each demo fixture must include ground-truth expected findings. "Missed" means a
ground-truth finding absent from both gateway evidence and security-review
verdict. A fourth optional control may cover adversarial diff text that tries to
suppress the security reviewer; this is reported as residual risk unless F164
coverage already provides a deterministic defense.

Ground truth must be written in the fixture before the gate run and reviewed in
the final evidence report. The demo is allowed to be dogfood/self-authored, but
post-hoc ground-truth edits after observing gate output must be called out as
invalid evidence.

## Workstream DAG

- `00-167-01` defines the contract.
- `00-167-02` depends on `00-167-01` and adds gateway sanitation/audit.
- `00-167-03` depends on `00-167-02` and integrates the gate.
- `00-167-04` depends on `00-167-03` and produces the Day-14 evidence pack.

This DAG is intentionally linear because each step consumes a concrete contract
from the previous step. If implementation reveals a contract change in an
earlier workstream, downstream work must update the design/workstream artifact
before continuing rather than silently adapting code.

## Acceptance

F167 is done when the workstream DAG is complete, targeted tests pass without
live credentials, the gate blocks Critical/High findings, Medium/Low warnings
are non-blocking but visible, and the demo evidence answers the Day-14 challenge
without leaking real secrets.
