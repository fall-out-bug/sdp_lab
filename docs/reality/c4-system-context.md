# Reality C4 System Context

- Generated At: `2026-03-12T13:21:35Z`
- System Scope: `sdp_dev reposet`
- Repositories: `repo:sdp`, `repo:sdp_dev`

## Systems

| System | Boundary | Repos | Notes |
|---|---|---|---|
| `sdp` | `internal` | `sdp` | protocol repo with 813 source files across 11 top-level modules. |
| `sdp_dev` | `internal` | `sdp_dev` | service repo with 1237 source files across 8 top-level modules. |

## Relationships

- `person:operator` -> `system:sdp`: reviews and arbitrates open questions
- `system:sdp_dev` -> `system:sdp`: consumes contracts from
- `system:sdp_dev` -> `system:sdp`: contains

## Review Roles

- `Operator`: Human reviewer who resolves intent gaps and arbitrates uncertain claims.
