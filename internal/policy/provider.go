// Package policy provides provider-agnostic model routing for multi-LLM support.
package policy

import (
	"strings"
)

// ProviderPrefix separates provider from model ID, e.g. "openrouter/gpt-4o" -> provider=openrouter, model=gpt-4o.
func ParseProviderModel(s string) (provider, model string) {
	if s == "" {
		return "", ""
	}
	idx := strings.Index(s, "/")
	if idx <= 0 {
		return "", s
	}
	return s[:idx], s[idx+1:]
}

// ProviderRegistry holds provider config (endpoint, auth env). Used for future OpenRouter/GLM routing.
type ProviderConfig struct {
	ID       string // e.g. "openrouter", "zhipuai-coding-plan"
	Endpoint string
	AuthEnv  string // env var for API key, e.g. "OPENROUTER_API_KEY"
}

// DefaultProviders returns the built-in provider configs.
func DefaultProviders() []ProviderConfig {
	return []ProviderConfig{
		{ID: "zhipuai-coding-plan", Endpoint: "", AuthEnv: "Z_AI_API_KEY"},
		{ID: "openrouter", Endpoint: "https://openrouter.ai/api/v1", AuthEnv: "OPENROUTER_API_KEY"},
	}
}

// NormalizeModel returns the model ID for allowlist check. Provider prefix is stripped for legacy allowlist.
func NormalizeModel(s string) string {
	_, model := ParseProviderModel(s)
	if model != "" {
		return model
	}
	return s
}
