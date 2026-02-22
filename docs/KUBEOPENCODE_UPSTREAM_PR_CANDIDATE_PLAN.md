# KubeOpenCode Upstream Contribution Plan and First PR Candidate

<!-- markdownlint-disable MD013 -->

Status: delivery artifact for `sdp_dev-2aq.7.2`

## Goal

Define an upstream-safe contribution path for kubeopencode and package the first contribution candidate so it can be submitted without SDP-private leakage.

## Candidate Upstream Changes

| Candidate ID | Generic capability | Why it belongs upstream | Scope boundary |
| --- | --- | --- | --- |
| `UP-001` | Task retry budget contract (`maxAttempts`, retry policy profile, terminal reason taxonomy). | Retry and escalation semantics are generic operator concerns and apply beyond SDP. | No SDP policy logic, no private risk classes, no tenant metadata. |
| `UP-002` | Multi-role dependency gating primitives (explicit dependency list and readiness condition). | DAG-like role dependency semantics are reusable for any multi-agent run. | No SDP evidence strictness or private approval workflow. |
| `UP-003` | Status trace linkage fields (`runContextRef`, `evidenceRef`, optional `prURL`). | Generic auditability and observability improve operator usability for all adopters. | No references to private storage paths or internal endpoint topology. |

## Maintainers and Stakeholders

| Group | Responsibility in acceptance path | Engagement strategy |
| --- | --- | --- |
| kubeopencode maintainers (repo owners) | Product direction and merge authority. | Share concise RFC summary before opening PR; ask for scope fit and naming guidance. |
| Controller/runtime maintainers | Validate reconciliation behavior and backward compatibility. | Include reconcilation-path impact table and upgrade notes in PR body. |
| CRD/API reviewers | Validate schema compatibility and API evolution quality. | Provide OpenAPI diff and defaulting behavior notes for each field addition. |
| External adopters/watchers | Early signal for broad utility. | Keep examples generic and include migration notes with opt-in defaults. |

## Acceptance Strategy

1. Lead with smallest broadly useful slice (`UP-001`) to reduce review load.
2. Keep changes additive and optional (no required CRD migration, no breaking defaults).
3. Include compatibility proof in PR body:
   - old manifests still valid,
   - default behavior unchanged when new fields omitted,
   - controller tests cover retry budget + terminal reason serialization.
4. Keep SDP-private constraints out of patch and documentation.
5. Capture traceability fields in Beads notes (`issue id`, candidate id, compare link/PR link).

## First PR Candidate (Prepared)

### Candidate summary

- Candidate ID: `UP-001`
- Proposed title: `Add generic retry budget and terminal reason contract to Task API`
- PR type: additive API + controller behavior; no SDP-only logic
- Repository target: `kubeopencode/kubeopencode`

### Planned patch path

1. Add retry contract fields in Task spec (optional):
   - `retry.maxAttempts`
   - `retry.profile` (`linear`, `exponential`)
2. Add normalized terminal reason fields in Task status:
   - `terminalReason.code`
   - `terminalReason.message`
3. Update reconciliation loop to honor retry budget without changing existing defaults.
4. Add unit/integration tests for retry exhaustion and terminal reason publication.
5. Add docs section with migration notes and examples.

### Traceable link package

- Beads task: `sdp_dev-2aq.7.2`
- Upstream compare candidate link: `https://github.com/kubeopencode/kubeopencode/compare/main...sdp-contrib:feat/retry-budget-terminal-reason`
- Submission command prepared:

```bash
gh pr create --repo kubeopencode/kubeopencode --base main --head sdp-contrib:feat/retry-budget-terminal-reason --title "Add generic retry budget and terminal reason contract to Task API" --body-file docs/upstream/UP-001-pr-body.md
```

The link and command provide a deterministic handoff path even when final submission is executed from a credentialed environment.

## Explicit Non-Upstream Scope (Excluded)

- SDP model allowlist policy decisions.
- SDP private risk thresholds and escalation classes.
- Tenant boundary and internal egress controls.
- SDP evidence redaction rules and private provenance keys.

These remain internal and are tracked in `docs/KUBEOPENCODE_SDP_INTERNAL_HARDENING_PATCHSET.md`.

## Acceptance Mapping (`sdp_dev-2aq.7.2`)

- **AC1**: candidate list, maintainer/stakeholder groups, and acceptance strategy are defined in this plan and companion spec.
- **AC2**: first upstream PR candidate is prepared with title, patch path, and traceable compare/submission links.
- **AC3**: explicit exclusion section defines SDP-private constraints that must not enter upstream scope.

## Evidence Notes (Task Closure)

- Added upstream plan and first PR candidate path at `docs/KUBEOPENCODE_UPSTREAM_PR_CANDIDATE_PLAN.md`.
- Added machine-readable contribution plan at `specs/runtime/kubeopencode-upstream-pr-candidate-plan.json`.
- Added schema for validation at `specs/runtime/schemas/kubeopencode-upstream-pr-candidate-plan.schema.json`.

<!-- markdownlint-enable MD013 -->
