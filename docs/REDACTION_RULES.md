# Redaction Rules (Private -> OSS)

## Never publish to OSS

- Enterprise pricing, margin, sales assumptions.
- Private repository names, internal branch strategy, internal hostnames.
- Security architecture details that increase attack surface.
- Customer-specific integrations and compliance exceptions.
- Internal model/provider routing heuristics tied to cost or incident history.

## Publishable in OSS

- Protocol contracts and schemas without internal policy constants.
- Generic orchestration interfaces and lifecycle states.
- Public quality gates and evidence format.
- Adapter interfaces for OpenCode/OpenClaw without private policy packs.

## Export process

1. Write full plan in `PRIVATE_BLUEPRINT.md`.
2. Produce sanitized summary using `OSS_EXPORT_TEMPLATE.md`.
3. Run explicit leak check against this file before publishing.
