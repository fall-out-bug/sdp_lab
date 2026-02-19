# Risk Policy (Private)

Status: baseline v1
Scope: autonomous path to PR

## 1. Risk classes

- `low`
  - docs changes
  - non-critical refactor
  - tests-only changes that do not alter runtime logic

- `medium`
  - regular business logic
  - CLI behavior changes without security impact
  - adapters and integration glue

- `high`
  - orchestration scheduler/routing logic
  - beads state and dependency behavior
  - evidence pipeline and trace chain logic
  - git safety and branch management logic

- `critical`
  - authentication and authorization
  - secret handling and credential flow
  - policy engine and security controls
  - compliance-critical mechanisms

## 2. Mandatory gate stack

- `low`
  - unit tests
  - lint
  - basic contract checks

- `medium`
  - all `low` gates
  - coverage gate
  - integration smoke tests

- `high`
  - all `medium` gates
  - adversarial review (required)
  - trace completeness hard check

- `critical`
  - all `high` gates
  - manual security signoff

## 3. Automatic escalation rules

Always escalate on:

- critical risk with uncertain impact
- change touches security-sensitive paths
- conflict between policy and requested action
- repeated verification failures past threshold

## 4. Notes on publishability

Only class definitions and public-safe gate names may be exported to OSS.
Internal thresholds, exceptions, and security-sensitive path lists remain private.
