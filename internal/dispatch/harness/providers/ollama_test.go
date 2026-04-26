package providers

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"sdp_dev/internal/dispatch/harness"
)

func TestOllamaProvider_Name(t *testing.T) {
	p := NewOllamaProvider("")
	if p.Name() != "ollama" {
		t.Errorf("expected Name() = 'ollama', got %q", p.Name())
	}
}

func TestOllamaProvider_Models_FromFixture(t *testing.T) {
	// Load the test fixture
	fixtureData, err := os.ReadFile(filepath.Join("testdata", "ollama_list.txt"))
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	// Create a provider with injected command runner
	cmdOutput := string(fixtureData)
	p := NewOllamaProviderWithRunner("", func(ctx context.Context, name string, args ...string) ([]byte, error) {
		// Simulate successful ollama list command
		if name == "ollama" && len(args) > 0 && args[0] == "list" {
			return []byte(cmdOutput), nil
		}
		return nil, nil
	})

	models := p.Models()
	if models == nil {
		t.Fatal("Models() returned nil")
	}

	// Verify models are parsed correctly
	expectedModels := []string{
		"qwen2.5-coder:7b",
		"llama3.2:3b",
	}
	if len(models) != len(expectedModels) {
		t.Errorf("expected %d models, got %d: %v", len(expectedModels), len(models), models)
	}
	for i, expected := range expectedModels {
		if i >= len(models) {
			break
		}
		if models[i] != expected {
			t.Errorf("model %d: expected %q, got %q", i, expected, models[i])
		}
	}
}

func TestOllamaProvider_CheckLimits_Unlimited(t *testing.T) {
	p := NewOllamaProviderWithRunner("", func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return nil, nil
	})

	limits, err := p.CheckLimits(context.Background())
	if err != nil {
		t.Errorf("CheckLimits() returned error: %v", err)
	}
	if limits == nil {
		t.Fatal("CheckLimits() returned nil")
	}
	if limits.Total != 999999 {
		t.Errorf("expected Total=999999, got %d", limits.Total)
	}
	if limits.Used != 0 {
		t.Errorf("expected Used=0, got %d", limits.Used)
	}
	if limits.Source != "local" {
		t.Errorf("expected Source='local', got %q", limits.Source)
	}
	if limits.CheckedAt.IsZero() {
		t.Error("CheckedAt should not be zero")
	}
}

func TestOllamaProvider_Models_Cached(t *testing.T) {
	// Track how many times the command is called
	callCount := 0
	p := NewOllamaProviderWithRunner("", func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if name == "ollama" && len(args) > 0 && args[0] == "list" {
			callCount++
			return []byte("NAME                       ID              SIZE      MODIFIED\ntest-model:1b              abc123          1.0 GB    1 day ago"), nil
		}
		return nil, nil
	})

	// First call should execute command
	models1 := p.Models()
	if callCount != 1 {
		t.Errorf("expected 1 command call after first Models(), got %d", callCount)
	}

	// Second call within TTL should use cache
	models2 := p.Models()
	if callCount != 1 {
		t.Errorf("expected cached result (still 1 call), got %d calls", callCount)
	}

	// Verify both calls return the same data
	if len(models1) != len(models2) {
		t.Errorf("cached result differs from fresh result")
	}
	if len(models1) > 0 && models1[0] != models2[0] {
		t.Errorf("cached model differs: %q vs %q", models1[0], models2[0])
	}
}

func TestOllamaProvider_Host_Default(t *testing.T) {
	p := NewOllamaProvider("")
	// Just verify it doesn't panic and returns a provider
	if p == nil {
		t.Fatal("NewOllamaProvider('') returned nil")
	}
}

func TestOllamaProvider_Host_Custom(t *testing.T) {
	customHost := "http://example.com:8000"
	p := NewOllamaProvider(customHost)
	if p == nil {
		t.Fatal("NewOllamaProvider(customHost) returned nil")
	}
	// Provider should be usable
	if p.Name() != "ollama" {
		t.Errorf("expected Name() = 'ollama', got %q", p.Name())
	}
}

// Test that Models() implements the Provider interface
func TestOllamaProvider_ImplementsProvider(t *testing.T) {
	var _ harness.Provider = NewOllamaProvider("")
}

// Test cache TTL expiration (simplified — with injectable clock would be better)
func TestOllamaProvider_Models_Stale(t *testing.T) {
	callCount := 0
	p := NewOllamaProviderWithRunner("", func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if name == "ollama" && len(args) > 0 && args[0] == "list" {
			callCount++
			return []byte("NAME                       ID              SIZE      MODIFIED\nstale:1b                   def456          1.0 GB    now"), nil
		}
		return nil, nil
	})

	// First call
	_ = p.Models()
	if callCount != 1 {
		t.Errorf("expected 1 call for first Models(), got %d", callCount)
	}

	// Artificially expire cache by forcing a wait past TTL
	// For now, just verify the mechanism doesn't crash
	// A real test would use a clock mock
	p = NewOllamaProviderWithRunner("", func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if name == "ollama" && len(args) > 0 && args[0] == "list" {
			callCount++
			return []byte("NAME                       ID              SIZE      MODIFIED\nfresh:1b                   ghi789          1.0 GB    now"), nil
		}
		return nil, nil
	})

	// Even if we call again, cache should be used (unless we implement TTL invalidation)
	models2 := p.Models()
	// This test is basic; a real cache-expiry test requires clock injection
	if len(models2) == 0 {
		t.Error("Models() returned empty after second call")
	}
}

// Test context timeout: Models() should return empty list on timeout
func TestOllamaProvider_Models_Timeout(t *testing.T) {
	p := NewOllamaProviderWithRunner("", func(ctx context.Context, name string, args ...string) ([]byte, error) {
		// Simulate a hung process by selecting on context cancellation
		// When the 5-second timeout fires, ctx.Done() closes and we return error
		<-ctx.Done()
		return nil, ctx.Err()
	})

	// Models() should return within ~6 seconds (5s timeout + margin)
	done := make(chan []string)
	go func() {
		done <- p.Models()
	}()

	start := time.Now()
	select {
	case models := <-done:
		elapsed := time.Since(start)
		// Should return empty list on timeout
		if len(models) != 0 {
			t.Errorf("expected empty list on timeout, got %d models", len(models))
		}
		// Verify it returned quickly (within ~6 seconds due to 5s timeout)
		if elapsed > 7*time.Second {
			t.Errorf("expected quick return on timeout, took %v", elapsed)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Models() did not return within 10 seconds (timeout not enforced)")
	}
}

