# Beads Scenario Backlog Template (Private)

Status: executable template
Use: seed autonomy backlogs for the two north-star scenarios

## Scenario 1: User feature -> PR

Hierarchy:

- parent `epic`: autonomy program scope
- child `feature`: scenario-specific feature
- child `task`: executable units

Parent feature template:

- title: `AUTO-FEAT: <feature-name>`
- type: `feature`
- labels: `autonomy`, `strict-evidence`, `scenario:user-feature`
- spec-id: `<path-to-private-plan>`

Child task sequence:

1. `INTAKE` - normalize request and acceptance criteria
2. `DESIGN` - decomposition and dependency graph
3. `BUILD` - implementation in feature branch
4. `VERIFY` - full strict evidence and gates
5. `PR` - create PR and link trace bundle

Dependency rule:

- each task blocks the next
- no parallel tasks in v1 unless risk class is `low`

## Scenario 2: Agent-initiated improvement -> PR

Hierarchy is the same: `epic -> feature -> task`.

Parent feature template:

- title: `AUTO-IMPROVE: <opportunity-name>`
- type: `feature`
- labels: `autonomy`, `strict-evidence`, `scenario:agent-initiated`
- spec-id: `<path-to-private-plan>`

Child task sequence:

1. `DISCOVERY` - identify opportunity and expected impact
2. `JUSTIFY` - value/risk gate and human-readable rationale
3. `DESIGN` - scoped workstream plan
4. `BUILD` - implementation in feature branch
5. `VERIFY` - strict evidence and adversarial review
6. `PR` - create PR with rationale and risk notes

Dependency rule:

- `JUSTIFY` must be approved by policy gate before `DESIGN`

## Required labels for all executable tasks

- `autonomy`
- `strict-evidence`
- `risk:<low|medium|high|critical>`
- `lane:<commit|explore>`

## Acceptance checklist for all `PR` tasks

- strict evidence has all 7 sections
- trace links beads -> branch -> commits -> PR
- risk notes include residual risk and out-of-scope items
- PR includes explicit "manual merge required" note
