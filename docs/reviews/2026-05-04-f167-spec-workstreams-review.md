# F167 Spec And Workstream Review

Date: 2026-05-04
Feature: F167
Mode: `pi-review` plus Socratic review
Worktree: `feature/F167-day14-security-step-design`

## Verdict

F167 design and workstreams are now good enough to proceed to implementation
planning.

- `pi-review`: APPROVED, 0 P0, 0 P1, 14 advisory findings.
- Branch-scope `pi-review` after the first revision: APPROVED, 0 P0, 0 P1, 2 advisory findings.
- Socratic judge: PASS, 11/11 critic questions resolved, no new contradictions,
  no scope creep.
- Second Socratic judge: PASS, 10/10 new critic questions resolved, no new
  contradictions, no scope creep.
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

Second branch-scope command:

```bash
go run ./cmd/sdp-pi-review --scope branch --base origin/main --feature F167 --test-command "go test ./internal/workstream ./internal/pireview ./cmd/sdp-pi-review" --model-timeout 3m --write-verdict
```

Result:

- Run: `pireview-e863d9081aef-1777873663309`
- Scope: branch, 10 files reviewed
- Verdict: `APPROVED`
- Findings: 0 P0, 0 P1, 2 total advisory findings
- Models: 2/3 succeeded
- Applied finding: fixed stale DAG text in the design doc.

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

## Socratic Review Round 2

Critic provider: `minimax/MiniMax-M2.7`
Judge provider: `zai/glm-5.1`

The second critic raised 10 questions:

| ID | Severity | Topic | Resolution |
|---|---|---|---|
| Q1 | blocking | Malformed verdict validation ownership | Resolved: `00-167-01` owns schema/parser; `00-167-03` invokes parser and escalates malformed output. |
| Q2 | blocking | Deterministic gate inputs unclear | Resolved: test evidence, sanitizer status/evidence, schema validity, and severity mapping are explicit gate inputs. |
| Q3 | major | Operator-visible escalation test missing | Resolved: `00-167-03` now requires pending-decision/human-gate surface tests. |
| Q4 | major | Linear DAG change-control unclear | Resolved: design now requires downstream artifact updates before continuing after contract changes. |
| Q5 | major | F164/F167 interface unclear | Resolved: F164 interface is document-level only; machine-readable binding is future work. |
| Q6 | major | Raw-secret negative tests missing | Resolved: `00-167-02` now requires tests proving no raw secret appears in evidence, findings, logs, or model-bound prompts. |
| Q7 | major | Promotion-ready too implementation-level | Resolved: design now states the operator-visible rule: work cannot leave Build as review-ready until pass/warn or human-approved escalation. |
| Q8 | major | Ground-truth demo rationalization risk | Resolved: ground truth must be committed before demo execution; post-run edits invalidate evidence. |
| Q9 | minor | Redaction/block reason codes unclear | Resolved: pattern-class reason codes added. |
| Q10 | minor | Provider failure simulation unclear | Resolved: existing fixed provider-failure fixtures and no-live-provider test mandate judged adequate. |

Judge result: PASS.

## Follow-Up

Before implementation starts:

- keep F167 separate from F164 corpus work
- build `00-167-01` first
- do not introduce live-provider dependency in unit tests
- preserve the Build pre-transition gate boundary unless implementation proves it
  impossible, in which case update this design before coding further
