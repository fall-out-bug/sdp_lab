package llm

import (
	"testing"
	"time"

	"sdp_dev/internal/architect"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Provider != ProviderOpenRouter {
		t.Errorf("expected provider %q, got %q", ProviderOpenRouter, cfg.Provider)
	}
	if cfg.BaseURL != "https://openrouter.ai/api/v1" {
		t.Errorf("expected base URL %q, got %q", "https://openrouter.ai/api/v1", cfg.BaseURL)
	}
	if cfg.Model != "openai/gpt-4o-mini" {
		t.Errorf("expected model %q, got %q", "openai/gpt-4o-mini", cfg.Model)
	}
	if cfg.MaxTokens != 4096 {
		t.Errorf("expected max tokens %d, got %d", 4096, cfg.MaxTokens)
	}
	if cfg.Timeout != 120*time.Second {
		t.Errorf("expected timeout %v, got %v", 120*time.Second, cfg.Timeout)
	}
	if cfg.Retry.MaxRetries != 3 {
		t.Errorf("expected max retries %d, got %d", 3, cfg.Retry.MaxRetries)
	}
	if cfg.FailureThreshold != 5 {
		t.Errorf("expected failure threshold %d, got %d", 5, cfg.FailureThreshold)
	}
	if cfg.CooldownPeriod != 30*time.Second {
		t.Errorf("expected cooldown period %v, got %v", 30*time.Second, cfg.CooldownPeriod)
	}
}

func TestNewClient(t *testing.T) {
	cfg := Config{
		Provider: ProviderLocal,
		BaseURL:  "http://localhost:11434/v1",
		APIKey:   "", // Local provider doesn't need API key
		Model:    "llama3.2",
	}
	filter := architect.NewSecurityFilter()

	client := NewClient(cfg, filter)

	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.cfg.Provider != ProviderLocal {
		t.Errorf("expected provider %q, got %q", ProviderLocal, client.cfg.Provider)
	}
	if client.filter == nil {
		t.Error("expected non-nil security filter")
	}
	if client.cb == nil {
		t.Error("expected non-nil circuit breaker")
	}
	if client.inner == nil {
		t.Error("expected non-nil inner client")
	}
}

func TestClient_MaxTokens(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxTokens = 2048
	client := NewClient(cfg, architect.NewSecurityFilter())

	// Request max takes precedence
	if got := client.maxTokens(4096); got != 4096 {
		t.Errorf("expected request max 4096, got %d", got)
	}

	// Config max used when request max is 0
	if got := client.maxTokens(0); got != 2048 {
		t.Errorf("expected config max 2048, got %d", got)
	}
}

func TestClient_Temperature(t *testing.T) {
	cfg := DefaultConfig()
	client := NewClient(cfg, architect.NewSecurityFilter())

	// Request temp takes precedence
	if got := client.temperature(0.7); got != 0.7 {
		t.Errorf("expected request temp 0.7, got %v", got)
	}

	// Default temp used when request temp is 0
	if got := client.temperature(0); got != 0.2 {
		t.Errorf("expected default temp 0.2, got %v", got)
	}
}

func TestClient_BackoffDelay(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Retry.BaseDelay = 100 * time.Millisecond
	cfg.Retry.MaxDelay = 10 * time.Second
	client := NewClient(cfg, architect.NewSecurityFilter())

	// Test exponential backoff with jitter
	tests := []struct {
		name         string
		attempt      int
		expectedMin  time.Duration
		expectedMax  time.Duration
	}{
		{"attempt 1", 1, 80 * time.Millisecond, 120 * time.Millisecond},   // 1s * 0.8-1.2 = 0.08-0.12s
		{"attempt 2", 2, 160 * time.Millisecond, 240 * time.Millisecond},  // 2s * 0.8-1.2 = 0.16-0.24s
		{"attempt 3", 3, 320 * time.Millisecond, 480 * time.Millisecond},  // 4s * 0.8-1.2 = 0.32-0.48s
		{"attempt 4", 4, 640 * time.Millisecond, 960 * time.Millisecond},  // 8s * 0.8-1.2 = 0.64-0.96s
		{"attempt 5", 5, 1280 * time.Millisecond, 1920 * time.Millisecond}, // 16s * 0.8-1.2 = 1.28-1.92s
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := client.backoffDelay(tt.attempt)
			if delay < tt.expectedMin || delay > tt.expectedMax {
				t.Errorf("attempt %d: delay %v outside range [%v, %v]",
					tt.attempt, delay, tt.expectedMin, tt.expectedMax)
			}
		})
	}
}

func TestClient_IsRetriable(t *testing.T) {
	client := NewClient(DefaultConfig(), architect.NewSecurityFilter())

	tests := []struct {
		name      string
		err       error
		retriable bool
	}{
		{"network error", &testError{"http: connection refused"}, true},
		{"timeout", &testError{"context timeout exceeded"}, true},
		{"5xx error", &testError{"status 500: internal server error"}, true},
		{"4xx error", &testError{"status 400: bad request"}, false},
		{"nil error", nil, false},
		{"auth error", &testError{"status 401: unauthorized"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.isRetriable(tt.err)
			if got != tt.retriable {
				t.Errorf("isRetriable(%v) = %v, want %v", tt.err, got, tt.retriable)
			}
		})
	}
}

func TestClient_Complete_CircuitBreaker(t *testing.T) {
	// This test would require mocking the HTTP client
	// For now, we just verify the client structure
	cfg := DefaultConfig()
	client := NewClient(cfg, architect.NewSecurityFilter())

	if client.cb == nil {
		t.Fatal("expected non-nil circuit breaker")
	}

	// Verify circuit breaker starts in closed state
	if !client.cb.Allow() {
		t.Error("expected circuit breaker to allow requests initially")
	}
}

// testError is a simple error type for testing.
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// Benchmark backoff delay calculation
func BenchmarkBackoffDelay(b *testing.B) {
	client := NewClient(DefaultConfig(), architect.NewSecurityFilter())
	for i := 0; i < b.N; i++ {
		client.backoffDelay(3)
	}
}
