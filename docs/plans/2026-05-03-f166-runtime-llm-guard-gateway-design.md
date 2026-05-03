# F166 Runtime LLM Guard Gateway Design

Status: reviewed for implementation planning
Date: 2026-05-03
Feature: F166
Beads: sdplab-mp83

## Decision

F166 is core-first.

Build a reusable Go guard layer for SDP LLM calls before building any HTTP proxy.
The HTTP proxy is an acceptance/demo surface, not the product center. SDP should
not clone LiteLLM, Portkey, or Helicone. Those projects already own generic AI
gateway routing, observability, and provider proxying. SDP's need is narrower:
protect agentic delivery calls where prompts, evidence, workstreams, Beads, and
provider responses cross trust boundaries.

Existing gateway projects are integration references, not dependencies for MVP.
They are too broad for the first SDP boundary because the required contract must
compose with `modelgateway.ChatRequest`, Beads/workstream provenance, and SDP
evidence rules without introducing a provider proxy control plane. Post-MVP, SDP
may compare against specialist scanners such as gitleaks-style entropy/rule engines
for broader coverage.

## Problem

F164 made prompt-injection risk measurable. It did not put an enforcement boundary
around live model calls. SDP still needs a runtime layer that can:

- stop or redact obvious secrets before a provider call
- inspect model output before it becomes user-visible or evidence-visible text
- log what was blocked, redacted, allowed, and charged
- preserve enough provenance for workstream, Beads, and session audits
- run deterministic tests without live providers

The day-13 challenge asks for an LLM proxy. SDP should keep the spirit, not the
letter: implement the guard and audit contract at the model-call boundary, then
optionally expose it through a small proxy command.

## Users

- SDP operator running agentic delivery flows.
- SDP maintainer reviewing evidence and security regressions.
- Future enterprise adopter who needs proof that provider calls are audited and
secrets are not sent blindly.

## UX Outcome

When a prompt contains a likely secret, the operator sees a concrete warning:
what class was found, whether the call was blocked or redacted, and which evidence
event records the decision. The operator should not need to inspect raw logs to
know why a model call did not happen.

When output is suspicious, the caller receives a guarded response state instead
of silently trusting model text. The warning must distinguish "blocked" from
"allowed with findings" so the flow can continue when the risk is advisory.

## DX Outcome

Call sites should wrap existing modelgateway providers or routers with a small
interface, not reimplement provider clients.

Expected integration shape:

```go
guarded := llmguard.NewGateway(provider, policy, auditSink)
resp, verdict, err := guarded.Chat(ctx, req)
```

The wrapper targets the current non-streaming `modelgateway.Provider` contract:
`Chat(ctx context.Context, req *modelgateway.ChatRequest) (*modelgateway.ChatResponse, error)`.
Streaming is explicitly out of scope for MVP. A future streaming guard must scan
accumulated output or a bounded rolling window before releasing chunks because
findings can span provider chunks.

The core package must be testable without network, env vars, or real API keys.
Gateway tests use a fake provider implementing the existing `modelgateway.Provider`
interface.

## Scope

Core package:

- `internal/llmguard` with input scanning, output scanning, redaction, verdicts,
  audit event types, and deterministic tests.
- Rule coverage for OpenAI-style `sk-` / `sk-proj-`, GitHub `ghp_`, AWS `AKIA`,
  email addresses, card-like numbers with Luhn validation, phone-like strings,
  bearer tokens, and base64-decoded secret candidates.
- Split-secret normalization for simple adjacent fragments such as `sk-` plus
  `proj-abc...`.
- "Simple adjacent" means fragments appear in the same message, in original order,
  with at most 16 ASCII non-alphanumeric bytes (`[^A-Za-z0-9]`) between fragments.
  Cross-message reconstruction, Unicode category joins, locale-aware joins, and
  arbitrary non-adjacent joining are out of scope for MVP.
- Configurable input action: `block` or `redact`.
- Output guard findings for generated secrets, prompt/system prompt disclosure
  attempts, suspicious URLs, and shell-command-like text.
- Audit records that store classifications and redacted excerpts, never raw
  secrets.
- Token and cost accounting from `modelgateway.TokenUsage` plus a static pricing
  table supplied by policy.
- A bounded scanner budget: max input bytes, max decoded candidate bytes, and one
  base64 decode pass for MVP. Budget exhaustion returns an advisory finding and
  can fail closed when policy requires strict mode.

Integration:

- A modelgateway wrapper/decorator that applies input guard before provider calls
  and output guard after responses.
- No live provider dependency in CI.
- Adoption is opt-in for F166. The first delivery target is the reusable package
  and wrapper contract. Wiring every live SDP call site is a follow-up unless a
  later workstream names a specific call site.

Demo surface:

- `cmd/sdp-llm-gateway` may be added after the core contract is stable.
- It should be a thin HTTP demo over the core, with per-IP request limit and JSON
  audit output.
- The demo command is not required for the first core implementation workstream.
  It is required only if a later Day-13 acceptance workstream asks for a runnable
  proxy artifact.

## Non-Goals

- Generic multi-provider proxy product.
- Provider routing rewrite.
- Perfect secret detection.
- Runtime sandboxing for tools.
- Blocking all suspicious model output by default.
- Live-provider tests in normal CI.
- Storing raw prompts or raw provider responses in audit logs.
- Multi-part image/audio content scanning. MVP scans string message content and
  JSON-encoded tool/function payloads when they are represented as text in the
  current modelgateway request shape.
- Queryable audit storage, signed logs, and Prometheus metrics. JSONL evidence is
  enough for MVP; real-time monitoring is a follow-up.

## Architecture

### Components

`internal/llmguard.Scanner`

Detects findings in text. It returns typed spans and severity. It must expose the
normalized scan path used for base64 and split-fragment checks so tests can record
what was caught and what remains a known miss.

The scanner API returns both findings and scan traces. A scan trace records the
mode (`raw`, `base64_decoded`, `split_joined`), the redacted candidate excerpt, and
whether a rule matched. Tests use traces to prove why encoded or split cases were
caught without exposing raw secrets.

`internal/llmguard.Redactor`

Replaces matched spans with stable placeholders such as `[REDACTED_API_KEY]`,
`[REDACTED_AWS_KEY]`, `[REDACTED_EMAIL]`, and `[REDACTED_CARD]`.
Typed placeholders are intentional in operator-facing warnings and audit finding
classes. Policy may use an untyped provider placeholder (`[REDACTED]`) when the
redacted prompt is sent onward, so the provider does not learn the secret class.

`internal/llmguard.Policy`

Defines input action, output action, rate-limit parameters for proxy/demo callers,
and model pricing. The default policy should be strict for input secrets and
advisory for suspicious output unless the output includes generated secrets.
Policy is immutable after gateway construction for MVP. Per-call metadata may
override request ids and SDP provenance, but not guard behavior. Runtime policy
mutation and policy-change audit are out of scope.
Policy is supplied as a Go struct literal or constructor argument. MVP does not
load policy from env vars, config files, or remote services. Default policy and
pricing fixtures may live in code for tests and demos, but production callers must
pass the policy explicitly.

Input redaction is a best-effort operational convenience, not the highest-security
posture. Default policy blocks high-severity provider/API credentials. Redact mode
is acceptable only when the caller explicitly chooses it for lower-sensitivity or
developer-demo flows. Tests must assert the configured policy, not treat block and
redact as interchangeable security outcomes.

`internal/llmguard.Gateway`

Wraps a modelgateway provider or equivalent `Chat(ctx, req)` function. It must
return a verdict even when the provider is not called.

`internal/llmguard.AuditSink`

Writes redacted JSONL events. Events include request id, tenant/session/feature/ws
metadata when present, input verdict, output verdict, provider id, model, token
usage, estimated cost, and blocked/redacted finding types.

Audit is synchronous and fail-closed by default. If the sink cannot record a
blocked, redacted, or allowed provider event, the gateway returns an audit failure
verdict and does not release a provider response. A best-effort audit mode may be
added later for low-risk demos, but it is not the MVP default.

Every call gets a guard-generated UUID `event_id` before scanning begins.
Caller-supplied request ids are stored separately as `correlation_id`; they never
replace the guard event id and do not need to be globally unique. If audit writing
fails, the gateway returns `audit_failed` with the already generated `event_id` and
`correlation_id` so the caller can surface a stable failure reference even though
the JSONL record was not persisted. MVP does not retry audit writes.

Audit leakage is tested by scanning serialized audit events for raw corpus secrets
before writes in tests. Runtime self-scanning of audit bytes is not required for
MVP because it risks recursion and extra latency, but the encoder must never use
raw prompt/response fields.

Audit records are not Beads records. They include `feature_id`, `ws_id`,
`beads_id`, `session_id`, and `evidence_ref` fields when the caller supplies them
so later evidence tooling can join records. Direct Beads writes are out of scope.

MVP audit interface:

```go
type AuditSink interface {
    WriteGuardEvent(ctx context.Context, event GuardEvent) error
}
```

`GuardEvent` is the JSONL schema. It includes `event_id`, `correlation_id`,
`feature_id`, `ws_id`, `beads_id`, `session_id`, `evidence_ref`, `timestamp`,
`provider_id`, `model`, `verdict_state`, `input_findings`, `output_findings`,
`redaction_summary`, `token_usage`, `cost_status`, `estimated_cost_usd`,
`provider_error_class`, and `provider_error_excerpt`.

Provider error messages are scanned and redacted before audit. Audit stores a short
redacted excerpt and a coarse provider error class, not raw provider error text.

`internal/llmguard.CostEstimator`

Computes estimated cost only when token usage and model pricing are both present.
If pricing is missing, the audit event records `cost_status: "unknown_pricing"`
instead of `$0.00`. Provider-reported token counts remain separate from estimated
USD cost.

### Data Flow

1. Caller builds `modelgateway.ChatRequest`.
2. Input scanner scans all user/system/assistant message content.
3. Policy decides block or redact.
4. If blocked, no provider call happens. Gateway returns guarded warning and audit.
5. If redacted or clean, provider call executes.
6. If the provider fails, gateway returns provider error plus a verdict state
   `provider_error_after_input_pass`; audit records the provider failure.
7. Output scanner checks the raw assistant response before any response redaction
   or release to the caller.
8. Gateway returns response plus verdict.
9. Audit sink records redacted event before response release.

## Verdict States

The MVP verdict state machine is:

| State | Provider called | Caller action |
|---|---:|---|
| `clean_allowed` | yes | use response |
| `redacted_allowed` | yes | use response, surface warning |
| `input_blocked` | no | show warning, do not retry unchanged |
| `output_blocked` | yes | do not show raw response; show warning |
| `allowed_with_output_findings` | yes | use response only if caller accepts advisory risk |
| `provider_error_after_input_pass` | yes | handle as provider failure, not security block |
| `audit_failed` | maybe | fail closed by default |
| `scan_budget_exceeded` | policy-dependent | fail closed in strict mode, advisory otherwise |

The verdict is a struct containing the state, input findings, output findings,
redaction summary, scan budget status, provider error class when present, and audit
event id when written.

All verdict state constants live in `internal/llmguard` so scanner, gateway, and
demo command share one vocabulary. Scanner-only code may only produce scanner-level
statuses, but the core package still owns the complete enum to avoid cross-package
string drift.

## Test Corpus

Minimum deterministic cases:

| Case | Expected |
|---|---|
| AWS access key in prompt | caught, blocked or redacted |
| OpenAI `sk-proj-` key | caught, placeholder emitted |
| GitHub `ghp_` token | caught |
| card-like number passing Luhn | caught |
| card-like number failing Luhn | not classified as card |
| email address | caught |
| phone-like string | caught, lower severity |
| base64-encoded secret | caught via decoded scan |
| split `sk-` + `proj-...` fragments | caught by normalized scan |
| clean prompt | allowed with no findings |
| model output leaking key | output blocked |
| model output containing prompt-extraction text | output finding |
| suspicious URL | output finding |
| shell command text | output finding |
| benign security doc with examples | allowed with findings or clean per policy, not blocked by keyword alone |
| unknown high-entropy string without known prefix | documented known miss |

Known misses must be explicit scanner tests that assert `clean_allowed` or no
finding for an accepted limitation, plus a short comment naming why the miss is
accepted for MVP.

Acceptance uses layered tests:

- scanner unit tests for detection and known misses
- redactor tests for placeholder behavior
- gateway integration tests with a fake provider
- audit tests asserting serialized events do not contain raw corpus secrets

Gateway corpus tests assert both the verdict and the safe audit shape. Scanner-only
tests do not need an audit sink.

Corpus secrets are synthetic fixtures, not real credentials. They should be
generated or clearly marked invalid where possible, while still matching detection
patterns. AWS/OpenAI/GitHub examples must use non-routable test values; card-like
numbers use standard test numbers only.

## Demo HTTP Schema

The optional demo proxy accepts a simplified envelope, not raw provider-specific
API shapes:

```json
{
  "model": "gpt-4o-mini",
  "messages": [{"role": "user", "content": "text"}],
  "metadata": {
    "correlation_id": "optional",
    "feature_id": "F166",
    "ws_id": "00-166-04",
    "beads_id": "sdplab-w8v7"
  }
}
```

Blocked response:

```json
{
  "verdict_state": "input_blocked",
  "warning": "request blocked by input guard",
  "findings": [{"type": "openai_key", "severity": "high"}],
  "event_id": "uuid"
}
```

Allowed response:

```json
{
  "verdict_state": "clean_allowed",
  "message": {"role": "assistant", "content": "text"},
  "event_id": "uuid",
  "usage": {"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2}
}

## Risks

- Regex-only detection will miss high-entropy secrets without known prefixes.
  Accept this for MVP and record known misses.
- Overblocking benign security docs hurts UX. Keep benign controls in the corpus.
- Cost tracking can lie if provider usage is absent. Mark estimates separately from
  provider-reported token usage.
- Logging is itself a leak risk. Raw secrets must never be written to audit events.
- Email and phone detection has a high false-positive risk. MVP treats them as
  lower-severity PII findings unless policy escalates them. Context-aware PII
  classification is out of scope.

## Acceptance Bar

F166 is ready for implementation when:

- a leaf workstream exists and links to Beads
- the guard package boundary is accepted
- MVP test corpus and known misses are explicit
- no claim is made that SDP now has a full generic LLM gateway
- the spec-interrogate report has no unresolved blocking questions, or unresolved
  questions are explicitly deferred with owner and rationale
