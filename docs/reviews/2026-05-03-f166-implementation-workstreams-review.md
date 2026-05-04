# F166 Implementation Workstreams Review

Feature: F166
Artifacts: `00-166-02`, `00-166-03`, `00-166-04`
Reviewer provider: `kimi-coding/k2p6` via `pi --no-tools --no-context-files --no-session`
Judge command provider: `minimax/MiniMax-M2.7` via `pi --no-tools --no-context-files --no-session`
Judge self-reported provider: `kimi-coding/k2p6`
Status: PASS

## Review Summary

The reviewer raised 12 questions. Blocking issues were predecessor/spec status and
the missing `AuditSink` interface/error contract. Major issues were package import
risk, ASCII separator semantics, scanner budget ownership, audit schema, policy
supply, synthetic test secrets, demo HTTP schema, and provider-error redaction.

## Author Resolution Notes

| Question | Status | Resolution |
|---|---|---|
| Q1 | resolved | 00-166-01 is completed and reviewed; 00-166-02 depends on it. |
| Q2 | resolved | Design status is now `reviewed for implementation planning`. |
| Q3 | resolved | Spec defines `AuditSink.WriteGuardEvent(ctx, event) error` and fail-closed behavior. |
| Q4 | resolved | 00-166-03 acceptance states `llmguard` may import `modelgateway`, but `modelgateway` must not import `llmguard`. |
| Q5 | resolved | Split separator semantics are ASCII bytes only: `[^A-Za-z0-9]`. |
| Q6 | resolved | Verdict constants live in core; scanner-only behavior remains separate from gateway behavior. |
| Q7 | resolved | 00-166-02 now owns scanner budget status. |
| Q8 | resolved | Spec defines `GuardEvent` JSONL schema fields and 00-166-03 acceptance checks them. |
| Q9 | resolved | Policy is constructor/struct supplied; no env/config/remote loader. |
| Q10 | resolved | Test secrets are synthetic non-routable fixtures and standard test card numbers. |
| Q11 | resolved | Spec defines simplified demo HTTP request/response schema. |
| Q12 | resolved | Provider errors are scanned/redacted before audit and stored as short redacted excerpts plus coarse class. |

## Next Step

Judge returned PASS: all 12 questions are resolved, with no new contradictions and
no scope creep. The command provider and self-reported provider disagreed; evidence
records that inconsistency. The implementation workstreams are ready for delivery
planning.
