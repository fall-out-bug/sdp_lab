# Repo Boundary Map (Private)

## Epic-level split

- OSS
  - `sdp-protocol`
  - `sdp-plugin`
  - `sdp-orchestrator`
- Private
  - `sdp-enterprise`

## Capability ownership matrix

- `sdp-protocol` (OSS)
  - protocol schemas
  - public contracts
  - public evidence format

- `sdp-plugin` (OSS)
  - lightweight CLI integration
  - adapter generation for tools

- `sdp-orchestrator` (OSS)
  - scheduler primitives
  - generic execution and retry framework

- `sdp-enterprise` (Private)
  - security policy packs
  - self-evolution engine and tuning
  - governance controls and approvals
  - commercial and customer-specific logic

## Anti-leak rules

- Security and self-evolution internals never leave private repo.
- OSS exports are blocked unless `cmd/redaction-check` passes.
- Export process always uses `docs/OSS_EXPORT_TEMPLATE.md`.
