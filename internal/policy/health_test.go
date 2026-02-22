package policy

import (
	"os"
	"testing"
)

func TestStubProviderHealthChecker(t *testing.T) {
	var c StubProviderHealthChecker
	if !c.IsAvailable("anthropic_direct") {
		t.Error("stub should report available")
	}
	tok, ok := c.QuotaRemaining("openai_direct")
	if !ok || tok <= 0 {
		t.Errorf("QuotaRemaining: got %d, ok=%v", tok, ok)
	}
}

func TestResolveProviderForModel_NilChecker(t *testing.T) {
	if got := ResolveProviderForModel("glm-4.7", nil); got != "glm" {
		t.Errorf("glm-4.7: got %q, want glm", got)
	}
	if got := ResolveProviderForModel("anthropic/claude-sonnet-4.6", nil); got != "openrouter" {
		t.Errorf("anthropic model: got %q, want openrouter", got)
	}
	if got := ResolveProviderForModel("openai/gpt-5.2", nil); got != "openrouter" {
		t.Errorf("openai model: got %q, want openrouter", got)
	}
}

func TestResolveProviderForModel_WithChecker(t *testing.T) {
	stub := StubProviderHealthChecker{}
	if got := ResolveProviderForModel("anthropic/claude-sonnet-4.6", stub); got != "anthropic_direct" {
		t.Errorf("anthropic with stub: got %q, want anthropic_direct", got)
	}
	if got := ResolveProviderForModel("glm-4.7", stub); got != "glm" {
		t.Errorf("glm: got %q, want glm", got)
	}
}

func TestEnvProviderHealthChecker(t *testing.T) {
	os.Setenv("ANTHROPIC_DIRECT_AVAILABLE", "false")
	os.Setenv("ANTHROPIC_DIRECT_QUOTA_REMAINING", "5000")
	defer os.Unsetenv("ANTHROPIC_DIRECT_AVAILABLE")
	defer os.Unsetenv("ANTHROPIC_DIRECT_QUOTA_REMAINING")

	c := &EnvProviderHealthChecker{}
	if c.IsAvailable("anthropic_direct") {
		t.Error("anthropic_direct should be unavailable when env=false")
	}
	tok, ok := c.QuotaRemaining("anthropic_direct")
	if !ok || tok != 5000 {
		t.Errorf("QuotaRemaining: got %d, ok=%v", tok, ok)
	}
}
