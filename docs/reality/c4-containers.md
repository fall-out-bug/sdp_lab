# Reality C4 Containers

- Generated At: `2026-03-13T07:33:34Z`
- System Name: `reality-pro-clone reposet`

## Containers

| Container | Repo | Technology | Responsibilities |
|---|---|---|---|
| `reality-pro-clone` | `reality-pro-clone` | `Go workspace` | Contains core implementation logic; Documents expected behavior and usage; Exposes operator-facing commands; Owns executable runtime and delivery automation |
| `sdp` | `sdp` | `Go, Markdown, JSON Schema` | Contains core implementation logic; Documents expected behavior and usage; Exposes operator-facing commands; Publishes prompts, contracts, and protocol runtime surfaces |
| `sdp_dev` | `sdp_dev` | `Go workspace` | Contains core implementation logic; Documents expected behavior and usage; Exposes operator-facing commands; Owns executable runtime and delivery automation |

## Relationships

- `container:reality-pro-clone` -> `container:sdp`: consumes contracts from
- `container:reality-pro-clone` -> `container:sdp`: contains
- `container:sdp_dev` -> `container:sdp`: consumes contracts from
- `container:sdp_dev` -> `container:reality-pro-clone`: contains
- `container:sdp_dev` -> `container:sdp`: contains
