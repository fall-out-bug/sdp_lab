# F166 Runtime LLM Guard Gateway Spec Interrogate

Feature: F166
Artifact: `docs/plans/2026-05-03-f166-runtime-llm-guard-gateway-design.md`
Mode: socratic
Critic provider: `zai/glm-5.1` via `pi --no-tools --no-context-files --no-session`
Judge provider: `kimi-coding/k2p6` via `pi --no-tools --no-context-files --no-session`
Status: PASS after author clarification

## Critic Summary

The critic raised 25 questions. Blocking questions focused on:

- `block` versus `redact` security semantics
- provider failure behavior after input guard pass
- streaming scope
- audit sink failure policy
- typed placeholder leakage
- audit leakage verification
- adoption path for existing call sites

Major questions focused on:

- actual `modelgateway` interface contract
- scope boundary versus observability gateway creep
- policy ownership and mutability
- cost pricing freshness
- scanner resource budgets
- benign PII false positives
- evidence/Beads join mechanism
- demo proxy acceptance ambiguity
- known misses contract

## Author Resolution Notes

| Question | Status | Resolution |
|---|---|---|
| Q001 | resolved | Spec now states block is default for high-severity credentials; redact is best-effort and explicit policy, not equivalent security. |
| Q002 | resolved | Spec now names the exact non-streaming `modelgateway.Provider.Chat(ctx, *ChatRequest)` contract. |
| Q003 | resolved | Spec separates core cost audit from deferred proxy rate limiting/metrics. |
| Q004 | resolved | Policy is immutable after gateway construction for MVP. |
| Q005 | resolved | JSONL is MVP proof; signed/queryable audit is out of scope. |
| Q006 | resolved | Provider failures get `provider_error_after_input_pass` verdict and audit. |
| Q007 | resolved | Streaming is explicitly out of scope for MVP. |
| Q008 | resolved | Spec now defines MVP verdict states. |
| Q009 | resolved | Missing pricing records `unknown_pricing`, not zero cost. |
| Q010 | resolved | Spec now defines layered tests and where audit assertions apply. |
| Q011 | resolved | Audit is synchronous and fail-closed by default. |
| Q012 | resolved | Spec now sets scan budget expectations and budget-exceeded verdict. |
| Q013 | resolved | PII is lower severity by default; context-aware classification is out of scope. |
| Q014 | resolved | MVP scans text and text-shaped JSON payloads only. |
| Q015 | resolved | Typed placeholders are operator/audit metadata; provider redaction can be untyped by policy. |
| Q016 | resolved | Audit serialization must be tested against raw secret leakage. |
| Q017 | resolved | Audit records include join fields but do not write Beads directly. |
| Q018 | resolved | Real-time metrics are deferred; JSONL is MVP surface. |
| Q019 | resolved | Guard-generated UUID is canonical; caller id is correlation metadata. |
| Q020 | resolved | Known misses require scanner tests and documented gaps. |
| Q021 | resolved | Wrapper tests use fake `modelgateway.Provider`. |
| Q022 | resolved | Adoption is opt-in for F166; wiring all call sites is follow-up. |
| Q023 | resolved | Demo proxy is optional unless a later workstream asks for runnable Day-13 artifact. |
| Q024 | resolved | Post-MVP scanner comparison is noted; regex is MVP, not permanent promise. |
| Q025 | resolved | Spec records why generic gateways are references, not MVP dependencies. |

## Next Step

The judge returned PASS with one partially resolved minor wording issue on request
id semantics. The spec now separates guard-generated `event_id` from caller-supplied
`correlation_id`, eliminating the ambiguity before workstream review.

Next step: run clean-context `pi` review over the workstream contract before using
it for implementation planning.
