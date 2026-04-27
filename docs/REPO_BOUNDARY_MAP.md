# Repo Boundary Map (Historical)

> Historical pre-F128 split. Current canonical boundary: [architecture/REPO-BOUNDARY.md](architecture/REPO-BOUNDARY.md). `sdp_lab` is now the primary public workspace; `sdp` is a distilled distribution repo.

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

## Historical Anti-leak Rules

- Security and self-evolution internals should not be committed if they are customer-private or commercially sensitive.
- OSS exports are blocked unless `cmd/redaction-check` passes.
- Export process always uses `docs/OSS_EXPORT_TEMPLATE.md`.
