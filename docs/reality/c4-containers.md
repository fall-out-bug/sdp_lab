# Reality C4 Containers

- Generated At: `2026-03-12T10:38:01Z`
- System Name: `sdp reposet`

## Containers

| Container | Repo | Technology | Responsibilities |
|---|---|---|---|
| `sdp` | `sdp` | `Go, Markdown, JSON Schema` | Contains core implementation logic; Documents expected behavior and usage; Exposes operator-facing commands; Publishes prompts, contracts, and protocol runtime surfaces |
| `sdp_dev` | `sdp_dev` | `Go workspace` | Contains core implementation logic; Documents expected behavior and usage; Exposes operator-facing commands; Owns executable runtime and delivery automation |

## Relationships

- `container:sdp_dev` -> `container:sdp`: consumes contracts from
- `container:sdp_dev` -> `container:sdp`: contains
