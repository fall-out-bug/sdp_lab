package policy

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// ProviderHealthChecker checks provider availability and quota for subscription routing.
// WS-013-01: Prefer direct subscription over OpenRouter when available.
type ProviderHealthChecker interface {
	// IsAvailable returns true if provider can accept requests.
	IsAvailable(provider string) bool
	// QuotaRemaining returns remaining tokens and true if quota is known.
	QuotaRemaining(provider string) (tokens int64, ok bool)
	// Latency returns recent latency for the provider.
	Latency(provider string) time.Duration
}

// StubProviderHealthChecker always reports providers as available.
// Use when subscription APIs are not configured or for testing.
type StubProviderHealthChecker struct{}

// IsAvailable always returns true.
func (StubProviderHealthChecker) IsAvailable(provider string) bool { return true }

// QuotaRemaining returns a large value and true (quota "known").
func (StubProviderHealthChecker) QuotaRemaining(provider string) (int64, bool) {
	return 1 << 30, true
}

// Latency returns 0.
func (StubProviderHealthChecker) Latency(provider string) time.Duration { return 0 }

// EnvProviderHealthChecker reads provider availability and quota from env (WS-013-01).
// Env vars: {PROVIDER}_AVAILABLE (true/false), {PROVIDER}_QUOTA_REMAINING (int).
// Example: ANTHROPIC_DIRECT_AVAILABLE=true, ANTHROPIC_DIRECT_QUOTA_REMAINING=1000000
type EnvProviderHealthChecker struct{}

func (e *EnvProviderHealthChecker) IsAvailable(provider string) bool {
	k := strings.ToUpper(strings.ReplaceAll(provider, "-", "_")) + "_AVAILABLE"
	v := os.Getenv(k)
	if v == "" {
		return true
	}
	b, _ := strconv.ParseBool(v)
	return b
}

func (e *EnvProviderHealthChecker) QuotaRemaining(provider string) (int64, bool) {
	k := strings.ToUpper(strings.ReplaceAll(provider, "-", "_")) + "_QUOTA_REMAINING"
	v := os.Getenv(k)
	if v == "" {
		return 1 << 30, true
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

func (e *EnvProviderHealthChecker) Latency(provider string) time.Duration { return 0 }

// ResolveProviderForModel returns preferred provider for model: subscription → openrouter → glm.
// When checker is nil, returns "openrouter" for OpenRouter models, "glm" for GLM.
func ResolveProviderForModel(model string, checker ProviderHealthChecker) string {
	provider, _ := ParseProviderModel(model)
	// Map model prefix to subscription provider
	switch {
	case strings.HasPrefix(provider, "anthropic") || strings.Contains(model, "claude"):
		if checker != nil && checker.IsAvailable("anthropic_direct") {
			return "anthropic_direct"
		}
		return "openrouter"
	case strings.HasPrefix(provider, "openai") || strings.Contains(model, "gpt"):
		if checker != nil && checker.IsAvailable("openai_direct") {
			return "openai_direct"
		}
		return "openrouter"
	case provider == "" || provider == "glm" || strings.HasPrefix(model, "glm-"):
		return "glm"
	default:
		return "openrouter"
	}
}
