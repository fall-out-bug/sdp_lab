# Reality Multi-Repo Map

- Generated At: `2026-03-13T07:33:34Z`
- Repositories Indexed: `3`

## Repo Roles

| Repo | Role | Root | Modules |
|---|---|---|---|
| `reality-pro-clone` | `service` | `/Users/fall_out_bug/projects/vibe_coding/sdp_dev/.codex-clones/reality-pro-clone` | `ci, cmd, docs, internal, scripts, tests` |
| `sdp` | `protocol` | `/Users/fall_out_bug/projects/vibe_coding/sdp_dev/.codex-clones/reality-pro-clone/sdp` | `.claude, .github, ci, cmd, examples, hooks, internal, root, scripts, sdp-plugin, src` |
| `sdp_dev` | `service` | `/Users/fall_out_bug/projects/vibe_coding/sdp_dev` | `.tmp, beads-viz-plugin, ci, cmd, docs, internal, scripts, tests` |

## Ownership Zones

- none reconstructed yet

## Team Metadata

- none ingested yet

## Boundary Edges

- `repo:reality-pro-clone` consumes contracts from `repo:sdp`
- `repo:reality-pro-clone` contains `repo:sdp`
- `repo:sdp_dev` contains `repo:reality-pro-clone`
- `repo:sdp_dev` consumes contracts from `repo:sdp`
- `repo:sdp_dev` contains `repo:sdp`

## Evidence Sources

- Total Sources: `459`

| Kind | Repo | Count |
|---|---|---|
| `adr` | `repo:reality-pro-clone` | `5` |
| `doc` | `repo:reality-pro-clone` | `431` |
| `runbook` | `repo:reality-pro-clone` | `23` |

### Sample Sources

| Kind | Repo | Locator |
|---|---|---|
| `adr` | `repo:reality-pro-clone` | `docs/ADR-0001-go-first-stack.md` |
| `doc` | `repo:reality-pro-clone` | `docs/AGENT_ARTIFACT_COMMUNICATION_PROTOCOL.md` |
| `doc` | `repo:reality-pro-clone` | `docs/AGENT_HANDOFF.md` |
| `doc` | `repo:reality-pro-clone` | `docs/AGENT_HOOKS_SPEC.md` |
| `doc` | `repo:reality-pro-clone` | `docs/AGENT_SKILLS_SPEC.md` |
| `doc` | `repo:reality-pro-clone` | `docs/AGENT_TEAMS.md` |
| `doc` | `repo:reality-pro-clone` | `docs/ARTIFACT_BUS_PR_SHIPPING_CHECKLIST.md` |
| `doc` | `repo:reality-pro-clone` | `docs/ARTIFACT_PROVENANCE_HASH_CHAIN_CONTRACT.md` |
| `doc` | `repo:reality-pro-clone` | `docs/ARTIFACT_PROVENANCE_INTAKE.md` |
| `runbook` | `repo:reality-pro-clone` | `docs/AUTONOMY_WORKER_RUNBOOK.md` |
| `doc` | `repo:reality-pro-clone` | `docs/BEADS_AUTONOMY_SPEC.md` |
| `doc` | `repo:reality-pro-clone` | `docs/BEADS_SCENARIO_BACKLOG_TEMPLATE.md` |
| `doc` | `repo:reality-pro-clone` | `docs/BEADS_SDP_REQUIREMENTS.md` |
| `doc` | `repo:reality-pro-clone` | `docs/CHANGELOG.md` |
| `doc` | `repo:reality-pro-clone` | `docs/CODE_QUALITY_FINDINGS.md` |
| `doc` | `repo:reality-pro-clone` | `docs/CODE_QUALITY_FINDINGS_5.md` |
| `doc` | `repo:reality-pro-clone` | `docs/CONCURRENCY_FINDINGS.md` |
| `doc` | `repo:reality-pro-clone` | `docs/CONTRACT_PARITY_REPORT.md` |
| `doc` | `repo:reality-pro-clone` | `docs/DOGFOODING_NOTES.md` |
| `doc` | `repo:reality-pro-clone` | `docs/ERROR_HANDLING_FINDINGS.md` |

## Persistent Questions

- reality-pro-clone: how does this repo coordinate versioning with the rest of the reposet?
- reality-pro-clone: ownership zones are not explicit; add CODEOWNERS or OWNERS metadata
- sdp: how does this repo coordinate versioning with the rest of the reposet?
- sdp: ownership zones are not explicit; add CODEOWNERS or OWNERS metadata
- sdp_dev: how does this repo coordinate versioning with the rest of the reposet?
