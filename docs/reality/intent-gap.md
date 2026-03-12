# Reality Intent Gap Report

- Generated At: `2026-03-12T13:21:34Z`
- Gaps: `3`
- Conflicts: `2`
- Current Readiness: `not_ready`
- Target Readiness: `ready`

## Intent Gaps

| Gap | Severity | Status | Repos | Next Step |
|---|---|---|---|---|
| `Service-to-protocol coordination is only partially evidenced` | `high` | `triaged` | `sdp, sdp_dev` | Add contract rollout checks before cross-repo changes land. |
| `Hotspot concentration still limits autonomous change scope` | `medium` | `accepted` | `sdp, sdp_dev` | Fence the largest hotspots with narrow workstreams before orchestration expands scope. |
| `Open questions still block confident synthesis` | `medium` | `triaged` | `sdp, sdp_dev` | Promote unresolved questions into explicit bootstrap backlog items with owners. |

## Conflicts

- `conflict:contract-boundary`: Synthesis reviewer kept the finding but downgraded certainty: Topology proves dependency direction, but governance and rollout discipline are still inferred rather than observed.
- `conflict:unresolved-questions`: Synthesis reviewer kept the finding but downgraded certainty: Open questions clearly exist, but severity should stay moderate until tied to concrete failures.

## Bootstrap Workstreams

- `Service-to-protocol coordination is only partially evidenced` [P1/sequenced]: Add contract rollout checks before cross-repo changes land.
- `Hotspot concentration still limits autonomous change scope` [P2/blocked]: Fence the largest hotspots with narrow workstreams before orchestration expands scope.
- `Open questions still block confident synthesis` [P2/proposed]: Promote unresolved questions into explicit bootstrap backlog items with owners.
