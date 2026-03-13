# Reality C4 System Context

- Generated At: `2026-03-13T07:33:34Z`
- System Scope: `reality-pro-clone reposet`
- Repositories: `repo:reality-pro-clone`, `repo:sdp`, `repo:sdp_dev`

## Systems

| System | Boundary | Repos | Notes |
|---|---|---|---|
| `reality-pro-clone` | `internal` | `reality-pro-clone` | service repo with 249 source files across 6 top-level modules. |
| `sdp` | `internal` | `sdp` | protocol repo with 813 source files across 11 top-level modules. |
| `sdp_dev` | `internal` | `sdp_dev` | service repo with 1237 source files across 8 top-level modules. |

## Relationships

- `person:operator` -> `system:reality-pro-clone`: reviews and arbitrates open questions
- `system:reality-pro-clone` -> `system:sdp`: consumes contracts from
- `system:reality-pro-clone` -> `system:sdp`: contains
- `system:sdp_dev` -> `system:sdp`: consumes contracts from
- `system:sdp_dev` -> `system:reality-pro-clone`: contains
- `system:sdp_dev` -> `system:sdp`: contains

## Review Roles

- `Operator`: Human reviewer who resolves intent gaps and arbitrates uncertain claims.
