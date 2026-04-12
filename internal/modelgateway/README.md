# internal/modelgateway

**Status:** Intentionally not wired (0 production callers as of 2026-04-11).

**Future role:** Multi-tenant LLM credential management for enterprise deployments.

## What this package is

`modelgateway` provides credential-aware LLM routing:
- `CredentialStore` — per-tenant API key storage
- `PolicyRouter` — route requests to providers based on policy
- `AuditLog` — log all LLM calls with provenance

This is an enterprise abstraction layer on top of `internal/llmclient`.

## What this package is NOT

- **Not the current LLM client.** Production code uses `internal/llmclient` directly.
- **Not wired to agentloop.** `agentloop.ModelGateway` is implemented by `internal/agentloop/livegw`.
- **Not deprecated.** It will be wired when multi-tenant credential management is needed.

## When to wire this

When SDP needs:
- Multiple tenants with separate API keys
- Per-request provider routing based on policy
- Full audit log of all LLM calls with provenance chain

Until then, use `internal/llmclient` directly.
