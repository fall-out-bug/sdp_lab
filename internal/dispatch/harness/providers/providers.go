// Package providers manages metadata-only AI provider implementations
// (OpenAI, Anthropic, Cursor, Kimi, Ollama, ...) registered for use by
// the dispatch Router. Mirrors the harness.Registry pattern.
//
// Providers expose Models() catalogs and CheckLimits() — they do NOT
// hold credentials. Auth lives at the harness CLI level (codex/cursor/
// opencode read their own env vars).
package providers

import (
	"log/slog"

	"sdp_dev/internal/dispatch/harness"
)

// ProviderRegistry manages a collection of named harness.Provider implementations.
type ProviderRegistry struct {
	providers map[string]harness.Provider
}

// NewRegistry returns an initialised, empty ProviderRegistry.
func NewRegistry() *ProviderRegistry {
	return &ProviderRegistry{providers: make(map[string]harness.Provider)}
}

// Register adds p to the registry under its Name(). A second registration
// with the same name overwrites the first (mirrors harness.Registry.Register).
func (r *ProviderRegistry) Register(p harness.Provider) {
	slog.Debug("registering provider", "name", p.Name())
	r.providers[p.Name()] = p
}

// Get returns the Provider registered under name, or nil if not found.
func (r *ProviderRegistry) Get(name string) harness.Provider {
	return r.providers[name]
}

// All returns the names of every registered provider.
func (r *ProviderRegistry) All() []string {
	names := make([]string, 0, len(r.providers))
	for n := range r.providers {
		names = append(names, n)
	}
	return names
}

// defaultRegistry is the package-level singleton used by bootstrap code in
// cmd/sdp-harness/main.go. Tests should construct their own via NewRegistry().
var defaultRegistry = NewRegistry()

// Default returns the package-level default registry.
func Default() *ProviderRegistry {
	return defaultRegistry
}
