# FR-013: ChatGPT/Claude Subscription Integration

Priority: P1
Effort: 2d
Dependencies: FR-012 (model routing works)

## Problem

OpenCode and OpenClaw support direct connection to ChatGPT/Claude via subscriptions (without OpenRouter). It is necessary to:
- Configure API keys for direct subscriptions
- Route requests through direct API when subscription is available (cheaper)
- Fallback to OpenRouter when direct subscription is exhausted/unavailable

## Design

### Provider Priority Chain

```
1. Direct subscription (OpenAI API / Anthropic API) — cheapest
2. OpenRouter — universal fallback
3. GLM — guaranteed availability
```

### Configuration

Extend `policy.yaml` from FR-012:

```yaml
providers:
  anthropic_direct:
    type: subscription
    api_key_secret: anthropic-direct-key
    base_url: https://api.anthropic.com/v1
    quota:
      requests_per_minute: 50
      tokens_per_day: 1000000
  openai_direct:
    type: subscription
    api_key_secret: openai-direct-key
    base_url: https://api.openai.com/v1
    quota:
      requests_per_minute: 60
      tokens_per_day: 2000000
  openrouter:
    type: pay_per_use
    api_key_secret: openrouter-api-key
    base_url: https://openrouter.ai/api/v1

routing:
  prefer_subscription: true
  fallback_order:
    - anthropic_direct
    - openai_direct
    - openrouter
    - glm
```

### Provider Health Check

```go
type ProviderHealthChecker interface {
    IsAvailable(provider string) bool
    QuotaRemaining(provider string) (tokens int64, ok bool)
    Latency(provider string) time.Duration
}
```

### Runtime Integration

OpenCode and OpenClaw already have built-in support:
- OpenCode: `OPENAI_API_KEY`, `ANTHROPIC_API_KEY` env vars
- OpenClaw: native provider config

Adapter-controller passes env vars into Task spec → kubeopencode operator → agent pod.

## Acceptance Criteria

- [ ] Direct subscription keys in K8s Secrets
- [ ] Provider health check (availability, quota)
- [ ] Routing: subscription → openrouter → glm
- [ ] Quota tracking per provider
- [ ] Env var propagation via Task CRD → pod
- [ ] Telemetry: provider used, cost saved vs openrouter
- [ ] Graceful degradation on quota exhaustion
