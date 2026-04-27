package providers

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

// TestCursorProvider_Name verifies the provider name.
func TestCursorProvider_Name(t *testing.T) {
	provider := NewCursorProvider(nil, nil, nil)
	got := provider.Name()
	want := "cursor"

	if got != want {
		t.Errorf("Name() = %q, want %q", got, want)
	}
}

// TestCursorProvider_Parser parses fixture cursor_list_models.txt.
func TestCursorProvider_Parser(t *testing.T) {
	fixture := `Available models:

claude-3.5-sonnet - Anthropic Claude 3.5 Sonnet
gpt-4o - OpenAI GPT-4o
gpt-4-turbo - OpenAI GPT-4 Turbo
claude-opus-4 - Anthropic Claude Opus 4
gpt-4 - OpenAI GPT-4
composer-1.5 - Cursor Composer 1.5
composer-2 - Cursor Composer 2
composer-2-fast - Cursor Composer 2 Fast
claude-3-opus - Anthropic Claude 3 Opus
claude-3-sonnet - Anthropic Claude 3 Sonnet
gpt-3.5-turbo - OpenAI GPT-3.5 Turbo
o1 - OpenAI O1
o1-mini - OpenAI O1 Mini

`

	models := parseListModelsOutput(fixture)

	if len(models) < 13 {
		t.Errorf("parseListModelsOutput() returned %d models, want ≥ 13", len(models))
	}

	expectedModels := map[string]bool{
		"claude-3.5-sonnet":   true,
		"gpt-4o":              true,
		"gpt-4-turbo":         true,
		"claude-opus-4":       true,
		"gpt-4":               true,
		"composer-1.5":        true,
		"composer-2":          true,
		"composer-2-fast":     true,
		"claude-3-opus":       true,
		"claude-3-sonnet":     true,
		"gpt-3.5-turbo":       true,
		"o1":                  true,
		"o1-mini":             true,
	}

	for _, model := range models {
		if !expectedModels[model] {
			t.Logf("parseListModelsOutput() returned unexpected model: %q", model)
		}
	}

	for expected := range expectedModels {
		found := false
		for _, model := range models {
			if model == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("parseListModelsOutput() does not contain expected model %q", expected)
		}
	}
}

// TestCursorProvider_CacheTTL verifies cache TTL behavior with injectable clock.
func TestCursorProvider_CacheTTL(t *testing.T) {
	fixture := `Available models:

claude-3.5-sonnet - Anthropic Claude 3.5 Sonnet
gpt-4o - OpenAI GPT-4o
`

	callCount := 0
	runner := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		callCount++
		return []byte(fixture), nil
	}

	fakeNow := time.Now()
	nowFn := func() time.Time {
		return fakeNow
	}

	provider := NewCursorProvider(nil, runner, nowFn)

	// First call should invoke runner
	models1 := provider.Models()
	if callCount != 1 {
		t.Errorf("First Models() call: expected 1 runner invocation, got %d", callCount)
	}
	if len(models1) != 2 {
		t.Errorf("First Models() returned %d models, want 2", len(models1))
	}

	// Second call within TTL (10 min) should use cache
	fakeNow = fakeNow.Add(5 * time.Minute)
	models2 := provider.Models()
	if callCount != 1 {
		t.Errorf("Second Models() call (within TTL): expected 1 total runner invocation, got %d", callCount)
	}
	if len(models2) != 2 {
		t.Errorf("Second Models() returned %d models, want 2", len(models2))
	}

	// Third call after TTL expires should re-fetch
	fakeNow = fakeNow.Add(6 * time.Minute) // Total 11 min elapsed
	models3 := provider.Models()
	if callCount != 2 {
		t.Errorf("Third Models() call (after TTL): expected 2 total runner invocations, got %d", callCount)
	}
	if len(models3) != 2 {
		t.Errorf("Third Models() returned %d models, want 2", len(models3))
	}
}

// TestCursorProvider_CLIAbsent verifies fallback when cursor binary is absent.
func TestCursorProvider_CLIAbsent(t *testing.T) {
	runner := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return nil, exec.ErrNotFound
	}

	provider := NewCursorProvider(nil, runner, nil)

	// Models() should return empty slice without panicking
	models := provider.Models()
	if models != nil && len(models) != 0 {
		t.Errorf("Models() with missing CLI returned %v, want empty slice or nil", models)
	}
}

// TestCursorProvider_CheckLimits verifies CheckLimits stub.
func TestCursorProvider_CheckLimits(t *testing.T) {
	provider := NewCursorProvider(nil, nil, nil)
	ctx := context.Background()

	limits, err := provider.CheckLimits(ctx)

	if err != nil {
		t.Fatalf("CheckLimits() failed: %v", err)
	}

	if limits == nil {
		t.Fatal("CheckLimits() returned nil, want &Limits")
	}

	if limits.Source != "cursor-cli" {
		t.Errorf("CheckLimits() Source = %q, want %q", limits.Source, "cursor-cli")
	}

	if limits.Total != 0 {
		t.Errorf("CheckLimits() Total = %d, want 0 (rate-limit not exposed)", limits.Total)
	}

	if limits.Used != 0 {
		t.Errorf("CheckLimits() Used = %d, want 0", limits.Used)
	}

	if limits.CheckedAt.IsZero() {
		t.Error("CheckLimits() CheckedAt is zero, want non-zero time")
	}
}

// TestCursorProvider_LargeFixture verifies parsing ≥25 models from fixture.
func TestCursorProvider_LargeFixture(t *testing.T) {
	fixture := `Available models:

claude-3.5-sonnet - Anthropic Claude 3.5 Sonnet
gpt-4o - OpenAI GPT-4o
gpt-4-turbo - OpenAI GPT-4 Turbo
claude-opus-4 - Anthropic Claude Opus 4
gpt-4 - OpenAI GPT-4
composer-1.5 - Cursor Composer 1.5
composer-2 - Cursor Composer 2
composer-2-fast - Cursor Composer 2 Fast
claude-3-opus - Anthropic Claude 3 Opus
claude-3-sonnet - Anthropic Claude 3 Sonnet
gpt-3.5-turbo - OpenAI GPT-3.5 Turbo
o1 - OpenAI O1
o1-mini - OpenAI O1 Mini
gpt-4-vision - OpenAI GPT-4 Vision
llama-2-7b - Meta Llama 2 7B
mistral-7b - Mistral 7B
mixtral-8x7b - Mixtral 8x7B
phi-2 - Microsoft Phi 2
falcon-40b - Falcon 40B
yi-34b - Yi 34B
neural-chat-7b - Intel Neural Chat 7B
`

	models := parseListModelsOutput(fixture)

	if len(models) < 21 {
		t.Errorf("parseListModelsOutput() returned %d models, want ≥ 21", len(models))
	}

	// Spot check some expected models
	hasComposer15 := false
	hasComposer2 := false
	hasGpt4 := false

	for _, model := range models {
		if model == "composer-1.5" {
			hasComposer15 = true
		}
		if model == "composer-2" {
			hasComposer2 = true
		}
		if model == "gpt-4" {
			hasGpt4 = true
		}
	}

	if !hasComposer15 {
		t.Error("parseListModelsOutput() missing composer-1.5")
	}
	if !hasComposer2 {
		t.Error("parseListModelsOutput() missing composer-2")
	}
	if !hasGpt4 {
		t.Error("parseListModelsOutput() missing gpt-4")
	}
}

// BenchmarkParseListModelsOutput measures parser performance.
func BenchmarkParseListModelsOutput(b *testing.B) {
	fixture := `Available models:

claude-3.5-sonnet - Anthropic Claude 3.5 Sonnet
gpt-4o - OpenAI GPT-4o
gpt-4-turbo - OpenAI GPT-4 Turbo
claude-opus-4 - Anthropic Claude Opus 4
gpt-4 - OpenAI GPT-4
composer-1.5 - Cursor Composer 1.5
composer-2 - Cursor Composer 2
composer-2-fast - Cursor Composer 2 Fast
`

	for i := 0; i < b.N; i++ {
		parseListModelsOutput(fixture)
	}
}
