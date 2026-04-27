# modelgateway — SDP Runtime Assumptions

## Context Objects
Chat completions: ChatRequest (model, messages, temp, tokens), ChatResponse (message, usage, finish reason)
Routing: RoutingDecision (provider, fallback, policy), RoutingInput (task class, sensitivity, constraints)
RoutingEvidence: Decision + hash + timestamp for audit

## Environment
API keys: ANTHROPIC_API_KEY, OPENAI_API_KEY, SDP_PROVIDER_<ID>_API_KEY
Custom endpoints: SDP_PROVIDER_<ID>_BASE_URL
No env vars for routing-only mode (PolicyRouter without providers)

## Configuration
Provider (ProviderConfig): ID, key, base URL, model, timeout, retries, rate limits
Router (RouterConfig): Default provider, fallback order, routing rules
Tenant (TenantConfig): Tenant ID, provider, constraints, rate limits
Constraints: Allowed providers/models, max cost per token, data residency

## Dependencies
From `internal/kernel`: ProviderID, ModelID, ModelCapabilities, RoutingInput/Decision/Constraints, TaskClass, SensitivityLevel
**Risk**: Kernel is shared substrate; breaking changes need coordinated updates across modelgateway, agent runtime, policy

## Allowlist Behavior
- Checked in PolicyRouter.Route() before provider selection
- TenantConfig.Constraints.AllowedProviders ([]ProviderID)
- Empty list = allow all; non-empty = restrict to listed providers
- Tenant-level override applies after policy evaluation

## Cost Envelope
Per-request: MaxTokensPerRequest enforced before call
Daily: MaxTokensPerDay aggregated across providers
Cost: CostPerToken (USD/1K) * token count
Rejection: ErrorTypeInvalidInput when limits exceeded
Reset: Midnight UTC

## Fallback Behavior
Ordered chain: FallbackChain attempted sequentially
Retryable: rate limits, timeouts (ErrorTypeRateLimit, ErrorTypeTimeout)
Non-retryable: auth failures, invalid input, model unavailable
Max retries: ProviderConfig.MaxRetries
Exhaustion: ProviderError "ALL_PROVIDERS_FAILED"
