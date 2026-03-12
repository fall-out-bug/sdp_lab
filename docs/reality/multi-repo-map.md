# Reality Multi-Repo Map

- Generated At: `2026-03-12T13:21:35Z`
- Repositories Indexed: `2`

## Repo Roles

| Repo | Role | Root | Modules |
|---|---|---|---|
| `sdp` | `protocol` | `/Users/fall_out_bug/projects/vibe_coding/sdp_dev/sdp` | `.claude, .github, ci, cmd, examples, hooks, internal, root, scripts, sdp-plugin, src` |
| `sdp_dev` | `service` | `/Users/fall_out_bug/projects/vibe_coding/sdp_dev` | `.tmp, beads-viz-plugin, ci, cmd, docs, internal, scripts, tests` |

## Boundary Edges

- `repo:sdp_dev` consumes contracts from `repo:sdp`
- `repo:sdp_dev` contains `repo:sdp`

## Evidence Sources

- Total Sources: `457`

| Kind | Repo | Count |
|---|---|---|
| `adr` | `repo:sdp_dev` | `5` |
| `doc` | `repo:sdp_dev` | `429` |
| `runbook` | `repo:sdp_dev` | `23` |

### Sample Sources

| Kind | Repo | Locator |
|---|---|---|
| `adr` | `repo:sdp_dev` | `docs/ADR-0001-go-first-stack.md` |
| `doc` | `repo:sdp_dev` | `docs/AGENT_ARTIFACT_COMMUNICATION_PROTOCOL.md` |
| `doc` | `repo:sdp_dev` | `docs/AGENT_HANDOFF.md` |
| `doc` | `repo:sdp_dev` | `docs/AGENT_HOOKS_SPEC.md` |
| `doc` | `repo:sdp_dev` | `docs/AGENT_SKILLS_SPEC.md` |
| `doc` | `repo:sdp_dev` | `docs/AGENT_TEAMS.md` |
| `doc` | `repo:sdp_dev` | `docs/ARTIFACT_BUS_PR_SHIPPING_CHECKLIST.md` |
| `doc` | `repo:sdp_dev` | `docs/ARTIFACT_PROVENANCE_HASH_CHAIN_CONTRACT.md` |
| `doc` | `repo:sdp_dev` | `docs/ARTIFACT_PROVENANCE_INTAKE.md` |
| `runbook` | `repo:sdp_dev` | `docs/AUTONOMY_WORKER_RUNBOOK.md` |
| `doc` | `repo:sdp_dev` | `docs/BEADS_AUTONOMY_SPEC.md` |
| `doc` | `repo:sdp_dev` | `docs/BEADS_SCENARIO_BACKLOG_TEMPLATE.md` |
| `doc` | `repo:sdp_dev` | `docs/BEADS_SDP_REQUIREMENTS.md` |
| `doc` | `repo:sdp_dev` | `docs/CHANGELOG.md` |
| `doc` | `repo:sdp_dev` | `docs/CODE_QUALITY_FINDINGS.md` |
| `doc` | `repo:sdp_dev` | `docs/CODE_QUALITY_FINDINGS_5.md` |
| `doc` | `repo:sdp_dev` | `docs/CONCURRENCY_FINDINGS.md` |
| `doc` | `repo:sdp_dev` | `docs/CONTRACT_PARITY_REPORT.md` |
| `doc` | `repo:sdp_dev` | `docs/DOGFOODING_NOTES.md` |
| `doc` | `repo:sdp_dev` | `docs/ERROR_HANDLING_FINDINGS.md` |

## Persistent Questions

- sdp: how does this repo coordinate versioning with the rest of the reposet?
- sdp_dev: how does this repo coordinate versioning with the rest of the reposet?
