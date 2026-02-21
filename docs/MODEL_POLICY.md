# Model Policy (Private)

Status: active restriction
Scope: orchestrator and swarm workers

## Allowed models

- `glm-5` (default primary)
- `glm-4.7` (default fallback)

Provider-prefixed models:

- `zhipuai-coding-plan/glm-5`, `zhipuai-coding-plan/glm-4.7`
- `openrouter/gpt-4o`, `openrouter/gpt-4o-mini`, `openrouter/claude-sonnet-4`, `openrouter/claude-3.5-sonnet`

See `docs/OPENROUTER_API_KEY.md` for where to add the OpenRouter API key.

Routing uses `internal/policy/provider.go` and `internal/policy/model_chain.go` for provider-agnostic resolution.

## Disallowed by default

- any model not listed above or not matching provider-prefix allowlist

## Routing baseline

1. Attempt execution on `glm-5`.
2. On transient or provider-side failure, retry by policy.
3. If fallback needed, switch to `glm-4.7`.
4. If both fail or policy mismatch occurs, escalate to human.

## Change management

- expanding allowed model set requires private policy change review
- no runtime auto-discovery of new models
- every model switch must be recorded in evidence `trace`
