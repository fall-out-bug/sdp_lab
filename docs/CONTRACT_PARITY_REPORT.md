# Contract Parity Report (Private)

Status: initial baseline

## Scope

Compares OpenCode and OpenClaw runtime capability sets against shared contract.

## Inputs

- `specs/runtime/opencode-capabilities.json`
- `specs/runtime/openclaw-capabilities.json`
- `specs/autonomy-runtime-contract.yaml`

## Validation command

```bash
go run ./cmd/runtime-parity-check --a specs/runtime/opencode-capabilities.json --b specs/runtime/openclaw-capabilities.json
```

## Current result

- parity: expected `equal=true`
- model policy parity: `glm-5`, `glm-4.7`
- strict evidence parity: all 7 keys aligned

## Notes

- This report validates contract-level parity.
- Runtime implementation parity in OpenClaw deployment is tracked in Stage B/B2 tasks.
