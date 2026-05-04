# F167 Spec And Workstream Review

Date: 2026-05-04
Feature: F167
Mode: `pi-review` plus Socratic review
Worktree: `feature/F167-day14-security-step-design`

## Verdict

F167 design and workstreams are now good enough to proceed to implementation
planning.

- `pi-review`: APPROVED, 0 P0, 0 P1, 14 advisory findings.
- Socratic judge: PASS, 11/11 critic questions resolved, no new contradictions,
  no scope creep.
- Provider coverage degraded: Kimi failed during `pi-review`; Zai and MiniMax
  succeeded. This is acceptable for design hardening evidence, not a delivery
  gate.

## Pi Review

Command:

```bash
go run ./cmd/sdp-pi-review --scope working-tree --feature F167 --test-command "go test ./internal/workstream ./internal/pireview ./cmd/sdp-pi-review" --model-timeout 3m --write-verdict
```

Result:

- Run: `pireview-1f95c3d9cc5c-1777871978716`
- Scope: working tree, 8 files reviewed
- Verdict: `APPROVED`
- Findings: 0 P0, 0 P1, 14 total advisory findings
- Models: 2/3 succeeded

Useful advisory findings applied:

- Clarified F164/F167 boundary and reviewer trust model.
- Defined the exact Build pre-transition gate insertion point.
- Added escalation recovery actions.
- Defined high-confidence sanitizer scope and SHA-256 evidence hashes.
- Added fixed reviewer-verdict fixtures for tests.
- Removed redundant `00-167-01` dependency from `00-167-03`.
- Clarified demo `missed` semantics through ground-truth expected findings.
- Added F167 DAG note to `docs/workstreams/INDEX.md`.

Advisory findings not applied:

- Splitting `00-167-02` into multiple workstreams. Rejected because the current
  scope is still one coherent M-sized gateway sanitation/audit slice.
- Adding a mandatory fourth demo scenario. Deferred; the three Day-14 scenarios
  stay mandatory, while adversarial reviewer-suppression is optional or delegated
  to F164 with rationale.

## Socratic Review

Critic provider: `zai/glm-5.1`
Judge provider: `minimax/MiniMax-M2.7`

The critic raised 11 questions:

| ID | Severity | Topic | Resolution |
|---|---|---|---|
| Q1 | blocking | Promotion-ready boundary unclear | Resolved: Build completion pre-transition boundary defined. |
| Q2 | blocking | Escalation recovery missing | Resolved: retry, rollback, approve-with-risk, stop actions defined. |
| Q3 | major | Sanitizer high-confidence scope unclear | Resolved: deterministic pattern scope defined. |
| Q4 | major | LLM reviewer prompt-injection circularity | Resolved: reviewer not trusted boundary; F164 taxonomy consumed. |
| Q5 | major | Test-double boundary missing | Resolved: fixed verdict fixtures required. |
| Q6 | major | Workstream file ownership overlap | Resolved: `00-167-03` no longer owns `livegw`. |
| Q7 | major | Gate enforcement mechanism unclear | Resolved: synchronous Build completion pre-transition gate. |
| Q8 | minor | Evidence usefulness vs leakage | Resolved: path/line/class/hash/remediation allowed; raw values forbidden. |
| Q9 | minor | `missed` finding definition unclear | Resolved: ground-truth expected findings required. |
| Q10 | minor | Contract workstream touched implementation scope | Resolved: `internal/agentloop` removed from `00-167-01`. |
| Q11 | minor | Reusable sanitizer vs narrow rollout unclear | Resolved: reusable interface, F167-only integration. |

Judge result: PASS.

## Follow-Up

Before implementation starts:

- keep F167 separate from F164 corpus work
- build `00-167-01` first
- do not introduce live-provider dependency in unit tests
- preserve the Build pre-transition gate boundary unless implementation proves it
  impossible, in which case update this design before coding further
