package modelgateway

import "context"

// ContractVersion defines the v1.0.0 semver contract for sdp-modelgw-core.
// This version formalizes the API surface for provider abstraction, routing,
// credential management, and runtime constraints. Breaking changes require
// a major version bump.
const ContractVersion = "v1.0.0"

// ProviderV1 defines the v1 contract for LLM provider implementations.
// Providers are responsible for executing chat completions against specific
// model APIs (Anthropic, OpenAI, self-hosted, etc.).
//
// Implementations must be thread-safe and handle their own rate limiting,
// retry logic, and error classification.
type ProviderV1 interface {
	// ID returns the unique identifier for this provider (e.g., "anthropic", "openai").
	ID() ProviderID

	// Chat executes a chat completion request against the provider's API.
	// Returns a ChatResponse on success or a ProviderError on failure.
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// Capabilities returns the provider's supported models and features.
	Capabilities() ModelCapabilities

	// IsAvailable performs a health check on the provider.
	// Returns false if the provider is unreachable or degraded.
	IsAvailable(ctx context.Context) bool

	// ValidateRequest validates the request before sending to the provider.
	// Returns an error if the request is malformed or exceeds provider limits.
	ValidateRequest(req *ChatRequest) error
}

// RouterV1 defines the v1 contract for routing chat requests to providers.
// Routers implement policy-based routing decisions, considering tenant
// constraints, cost limits, and fallback chains.
//
// Implementations may wrap ProviderRegistry for provider lookup.
type RouterV1 interface {
	// Route selects a provider for the given request.
	// Returns ProviderError if no suitable provider is available.
	Route(req *ChatRequest) (Provider, error)

	// Chat routes the request and executes the chat completion.
	// Delegates to the selected provider's Chat method.
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// ChatWithFallback attempts the request with retry on failure.
	// Iterates through the fallback chain on retryable errors.
	// Returns ProviderError if all providers fail.
	ChatWithFallback(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
}

// CredentialManagerV1 defines the v1 contract for managing provider API keys.
// Credential managers handle storage, rotation, and lifecycle of credentials
// per tenant and provider.
//
// Implementations must be thread-safe and audit all credential operations.
type CredentialManagerV1 interface {
	// GetCredential retrieves the active credential for a tenant/provider pair.
	// Returns CredentialError if not found, expired, or revoked.
	GetCredential(ctx context.Context, tenantID string, providerID ProviderID) (*Credential, error)

	// CreateCredential stores a new credential with the given API key.
	// Accepts optional CredentialOption funcs for expiry, base URL, etc.
	// Returns the created Credential on success.
	CreateCredential(ctx context.Context, tenantID string, providerID ProviderID, apiKey string, opts ...CredentialOption) (*Credential, error)

	// RotateCredential replaces an existing credential with a new API key.
	// Marks the old credential as "rotating" status before creating the new one.
	// Returns the new Credential on success.
	RotateCredential(ctx context.Context, tenantID string, providerID ProviderID, newAPIKey string) (*Credential, error)

	// RevokeCredential permanently disables a credential.
	// Clears the API key and sets status to "revoked".
	RevokeCredential(ctx context.Context, tenantID string, providerID ProviderID) error

	// CheckExpiry returns credentials nearing expiration within the alert window.
	// Auto-updates expired credentials from "active" to "expired" status.
	CheckExpiry(ctx context.Context, tenantID string) ([]*Credential, error)
}

// AllowlistContract documents the provider allowlist behavior.
//
// The allowlist is checked before routing decisions to enforce tenant-level
// provider constraints. If a tenant has a configured allowlist (non-empty),
// only providers on that list are eligible for routing. If no allowlist is
// configured (empty or nil), the system defaults to allow-all behavior.
//
// Allowlists are configured via TenantConfig.Constraints.AllowedProviders.
// The PolicyRouter applies allowlist constraints during the Route() method.
//
// Default behavior: allow-all (no restrictions if list is empty).
type AllowlistContract struct{}

// CostEnvelope defines token and cost limits for provider usage.
//
// These limits are applied per request and aggregated daily per tenant.
// Exceeding limits results in request rejection before provider execution.
type CostEnvelope struct {
	// MaxTokensPerRequest is the maximum token count allowed for a single request.
	// Includes both prompt and completion tokens. Requests exceeding this limit
	// are rejected with ErrorTypeInvalidInput.
	MaxTokensPerRequest int

	// MaxTokensPerDay is the maximum total tokens allowed per tenant per day.
	// Aggregated across all providers. Once exceeded, all requests are rejected
	// until the daily window resets.
	MaxTokensPerDay int

	// CostPerToken is the cost multiplier in USD per 1K tokens.
	// Used to calculate actual spend against MaxCostPerToken in RoutingConstraints.
	CostPerToken float64
}

// FallbackContract documents the fallback behavior for failed requests.
//
// Fallback is triggered only for retryable errors (rate limits, timeouts).
// Non-retryable errors (auth, invalid input) fail immediately without fallback.
// The fallback chain is ordered: each provider is attempted in sequence until
// one succeeds or all are exhausted.
//
// Max retries is configurable per provider via ProviderConfig.MaxRetries.
// The router classifies errors via isRetryableError() based on ProviderError.Type.
//
// Retryable error types: ErrorTypeRateLimit, ErrorTypeTimeout.
// Non-retryable: ErrorTypeAuth, ErrorTypeInvalidInput, ErrorTypeModelNotAvailable.
type FallbackContract struct{}
