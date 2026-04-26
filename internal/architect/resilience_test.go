package architect

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 1*time.Second, config.BaseDelay)
	assert.Equal(t, 30*time.Second, config.MaxDelay)
}

func TestRetryConfig_CalculateBackoff(t *testing.T) {
	config := DefaultRetryConfig()

	// First retry should use BaseDelay
	backoff := config.calculateBackoff(0)
	assert.InDelta(t, 1*time.Second, backoff, float64(200*time.Millisecond)) // 20% jitter

	// Second retry should double
	backoff = config.calculateBackoff(1)
	assert.InDelta(t, 2*time.Second, backoff, float64(400*time.Millisecond))

	// Third retry should double again
	backoff = config.calculateBackoff(2)
	assert.InDelta(t, 4*time.Second, backoff, float64(800*time.Millisecond))

	// Fourth retry should double again
	backoff = config.calculateBackoff(3)
	assert.InDelta(t, 8*time.Second, backoff, float64(1600*time.Millisecond))

	// Fifth retry should double again
	backoff = config.calculateBackoff(4)
	assert.InDelta(t, 16*time.Second, backoff, float64(3200*time.Millisecond))

	// Sixth retry should cap at MaxDelay (30s)
	backoff = config.calculateBackoff(5)
	assert.InDelta(t, 30*time.Second, backoff, float64(6*time.Second))
}

func TestNewCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker()

	assert.NotNil(t, cb)
	assert.Equal(t, "closed", cb.State)
	assert.Equal(t, 0, cb.Failures)
	assert.Equal(t, 5, cb.FailureThreshold)
	assert.Equal(t, 30*time.Second, cb.CooldownPeriod)
}

func TestCircuitBreaker_Allow_ClosedState(t *testing.T) {
	cb := NewCircuitBreaker()

	// In closed state, all requests should be allowed
	assert.True(t, cb.Allow())
	assert.True(t, cb.Allow())
	assert.True(t, cb.Allow())
}

func TestCircuitBreaker_RecordFailure(t *testing.T) {
	cb := NewCircuitBreaker()

	// Record failures
	cb.RecordFailure()
	assert.Equal(t, 1, cb.Failures)
	assert.Equal(t, "closed", cb.State)

	cb.RecordFailure()
	assert.Equal(t, 2, cb.Failures)
	assert.Equal(t, "closed", cb.State)

	// Record more failures until threshold
	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}

	// Should now be open
	assert.Equal(t, "open", cb.State)
	assert.GreaterOrEqual(t, cb.Failures, 5)
}

func TestCircuitBreaker_Allow_OpenState(t *testing.T) {
	cb := NewCircuitBreaker()

	// Force circuit open
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}

	assert.Equal(t, "open", cb.State)

	// Requests should be blocked
	assert.False(t, cb.Allow())
	assert.False(t, cb.Allow())
}

func TestCircuitBreaker_Allow_HalfOpenState(t *testing.T) {
	cb := NewCircuitBreaker()

	// Force circuit open
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}
	assert.Equal(t, "open", cb.State)

	// Set last failure time to past to trigger half-open
	cb.LastFailure = time.Now().Add(-31 * time.Second)

	// First request should be allowed (transition to half-open)
	assert.True(t, cb.Allow())
	assert.Equal(t, "half-open", cb.State)

	// Second request should be blocked (only one probe in half-open)
	assert.False(t, cb.Allow())
}

func TestCircuitBreaker_RecordSuccess(t *testing.T) {
	cb := NewCircuitBreaker()

	// Force circuit open
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}
	assert.Equal(t, "open", cb.State)

	// Record success (should close circuit)
	cb.RecordSuccess()

	assert.Equal(t, "closed", cb.State)
	assert.Equal(t, 0, cb.Failures)
}

func TestCircuitBreaker_Execute_Success(t *testing.T) {
	cb := NewCircuitBreaker()

	ctx := context.Background()
	called := false
	err := cb.Execute(ctx, func() error {
		called = true
		return nil
	})

	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, "closed", cb.State)
}

func TestCircuitBreaker_Execute_Failure(t *testing.T) {
	cb := NewCircuitBreaker()

	ctx := context.Background()
	testErr := errors.New("test error")
	err := cb.Execute(ctx, func() error {
		return testErr
	})

	assert.Error(t, err)
	assert.Equal(t, testErr, err)
}

func TestCircuitBreaker_Execute_OpenCircuit(t *testing.T) {
	cb := NewCircuitBreaker()

	// Force circuit open
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}

	ctx := context.Background()
	err := cb.Execute(ctx, func() error {
		return nil
	})

	assert.Error(t, err)

	var cbErr *CircuitBreakerError
	assert.ErrorAs(t, err, &cbErr)
	assert.Equal(t, StateOpen, cbErr.State)
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker()

	// Force circuit open
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}
	assert.Equal(t, "open", cb.State)

	// Reset by recording success
	cb.RecordSuccess()

	assert.Equal(t, "closed", cb.State)
	assert.Equal(t, 0, cb.Failures)
}

func TestCircuitBreakerState_String(t *testing.T) {
	assert.Equal(t, "closed", StateClosed.String())
	assert.Equal(t, "open", StateOpen.String())
	assert.Equal(t, "half-open", StateHalfOpen.String())
	assert.Equal(t, "unknown", CircuitState(99).String())
}

func TestCircuitBreakerError_Error(t *testing.T) {
	err := &CircuitBreakerError{
		Provider: "openai",
		State:    StateOpen,
		Message:  "circuit breaker is open",
	}

	errStr := err.Error()
	assert.Contains(t, errStr, "openai")
	assert.Contains(t, errStr, "open")
	assert.Contains(t, errStr, "circuit breaker is open")
}

func TestCircuitBreakerError_ErrorWithRetryAfter(t *testing.T) {
	err := &CircuitBreakerError{
		Provider:   "anthropic",
		State:      StateOpen,
		Message:    "circuit breaker is open",
		RetryAfter: 30 * time.Second,
	}

	errStr := err.Error()
	assert.Contains(t, errStr, "retry after")
}

func TestIsCircuitBreakerError(t *testing.T) {
	cbErr := &CircuitBreakerError{
		Provider: "openai",
		State:    StateOpen,
		Message:  "test",
	}

	assert.True(t, IsCircuitBreakerError(cbErr))
	assert.False(t, IsCircuitBreakerError(errors.New("other error")))
}

func TestNewEnrichmentResult(t *testing.T) {
	requestID := "test-123"
	result := NewEnrichmentResult(requestID)

	assert.False(t, result.Completed)
	assert.NotNil(t, result.Enrichment)
	assert.NotNil(t, result.Failed)
	assert.Equal(t, requestID, result.RequestID)
}

func TestEnrichmentResult_Complete(t *testing.T) {
	result := NewEnrichmentResult("test-123")

	assert.False(t, result.Completed)

	result.Complete()
	assert.True(t, result.Completed)
}

func TestEnrichmentResult_AddFailure(t *testing.T) {
	result := NewEnrichmentResult("test-123")

	err := errors.New("test error")
	result.AddFailure("node-1", "llm_call", err, true)

	assert.Len(t, result.Failed, 1)
	assert.Equal(t, "node-1", result.Failed[0].NodeID)
	assert.Equal(t, "llm_call", result.Failed[0].Stage)
	assert.Equal(t, err, result.Failed[0].Err)
	assert.True(t, result.Failed[0].Retriable)
}

func TestEnrichmentResult_HasFailures(t *testing.T) {
	result := NewEnrichmentResult("test-123")

	assert.False(t, result.HasFailures())

	result.AddFailure("node-1", "stage", errors.New("error"), true)

	assert.True(t, result.HasFailures())
}

func TestEnrichmentResult_HasRetriableFailures(t *testing.T) {
	result := NewEnrichmentResult("test-123")

	result.AddFailure("node-1", "stage", errors.New("error1"), true)
	result.AddFailure("node-2", "stage", errors.New("error2"), false)

	assert.True(t, result.HasRetriableFailures())
	assert.Len(t, result.GetRetriableFailures(), 1)
	assert.Equal(t, "node-1", result.GetRetriableFailures()[0].NodeID)
}

func TestEnrichmentResult_Merge(t *testing.T) {
	result1 := NewEnrichmentResult("req-1")
	result1.Enrichment["key1"] = LLMEnrichment{Description: "value1"}
	result1.AddFailure("node-1", "stage", errors.New("error1"), true)
	result1.Duration = 100 * time.Millisecond

	result2 := NewEnrichmentResult("req-2")
	result2.Enrichment["key2"] = LLMEnrichment{Description: "value2"}
	result2.AddFailure("node-2", "stage", errors.New("error2"), false)
	result2.Duration = 200 * time.Millisecond

	result1.Merge(result2)

	assert.Contains(t, result1.Enrichment, "key1")
	assert.Contains(t, result1.Enrichment, "key2")
	assert.Len(t, result1.Failed, 2)
	assert.Equal(t, 300*time.Millisecond, result1.Duration)
}

func TestEnrichmentError_Error(t *testing.T) {
	err := EnrichmentError{
		NodeID:    "node-1",
		Stage:     "llm_call",
		Retriable: true,
		Err:       errors.New("API error"),
	}

	errStr := err.Error()
	assert.Contains(t, errStr, "node-1")
	assert.Contains(t, errStr, "llm_call")
	assert.Contains(t, errStr, "retriable=true")
	assert.Contains(t, errStr, "API error")
}

func TestEnrichmentError_Unwrap(t *testing.T) {
	originalErr := errors.New("original")
	err := EnrichmentError{
		NodeID: "node-1",
		Stage:  "stage",
		Err:    originalErr,
	}

	assert.Equal(t, originalErr, err.Unwrap())
}

func TestGenerateRequestID(t *testing.T) {
	id1 := GenerateRequestID()
	id2 := GenerateRequestID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
}

func TestRetryWithBackoff_Success(t *testing.T) {
	config := DefaultRetryConfig()
	ctx := context.Background()

	calls := 0
	err := RetryWithBackoff(ctx, config, func() error {
		calls++
		if calls < 2 {
			return errors.New("temporary error")
		}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 2, calls)
}

func TestRetryWithBackoff_MaxRetries(t *testing.T) {
	config := DefaultRetryConfig()
	ctx := context.Background()

	calls := 0
	err := RetryWithBackoff(ctx, config, func() error {
		calls++
		return errors.New("persistent error")
	})

	assert.Error(t, err)
	assert.Equal(t, 4, calls) // initial + 3 retries
	assert.Contains(t, err.Error(), "max retries")
}

func TestRetryWithBackoff_ContextCancellation(t *testing.T) {
	config := DefaultRetryConfig()
	config.MaxRetries = 100 // Set high to test context cancellation

	ctx, cancel := context.WithCancel(context.Background())
	calls := 0

	// Cancel after first attempt
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := RetryWithBackoff(ctx, config, func() error {
		calls++
		return errors.New("error")
	})

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestSafeExecutor_Execute(t *testing.T) {
	retryConfig := DefaultRetryConfig()
	breakerConfig := DefaultCircuitBreakerConfig()

	executor := NewSafeExecutor(retryConfig, breakerConfig)
	ctx := context.Background()

	calls := 0
	err := executor.Execute(ctx, "openai", func() error {
		calls++
		if calls < 2 {
			return errors.New("temporary error")
		}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 2, calls)
}

func TestSafeExecutor_CircuitBreakerIntegration(t *testing.T) {
	retryConfig := RetryConfig{
		MaxRetries: 0, // No retries
		BaseDelay:  1 * time.Millisecond,
		MaxDelay:   10 * time.Millisecond,
	}

	breakerConfig := CircuitBreakerConfig{
		FailureThreshold: 3,
		CooldownPeriod:   100 * time.Millisecond,
	}

	executor := NewSafeExecutor(retryConfig, breakerConfig)
	ctx := context.Background()

	// First 3 failures should open the circuit
	for i := 0; i < 3; i++ {
		err := executor.Execute(ctx, "test-provider", func() error {
			return errors.New("error")
		})
		assert.Error(t, err)
	}

	// Next call should fail with circuit breaker error
	err := executor.Execute(ctx, "test-provider", func() error {
		return nil
	})

	assert.Error(t, err)
	assert.True(t, IsCircuitBreakerError(err))
}

func TestSafeExecutor_GetCircuitBreakerState(t *testing.T) {
	retryConfig := DefaultRetryConfig()
	breakerConfig := DefaultCircuitBreakerConfig()

	executor := NewSafeExecutor(retryConfig, breakerConfig)

	state := executor.GetCircuitBreakerState("openai")
	assert.Equal(t, "closed", state)
}

func TestCircuitBreakerManager(t *testing.T) {
	config := DefaultCircuitBreakerConfig()
	manager := NewCircuitBreakerManager(config)

	// Get or create breaker for provider
	cb1 := manager.GetOrCreate("openai")
	cb2 := manager.GetOrCreate("openai")
	cb3 := manager.GetOrCreate("anthropic")

	// Should return same instance for same provider
	assert.Same(t, cb1, cb2)

	// Should return different instance for different provider
	assert.NotSame(t, cb1, cb3)
}

func TestEnrichmentResult_String(t *testing.T) {
	t.Run("completed result", func(t *testing.T) {
		result := NewEnrichmentResult("req-1")
		result.Complete()
		result.Duration = 500 * time.Millisecond

		str := result.String()
		assert.Contains(t, str, "completed=true")
		assert.Contains(t, str, "req-1")
		assert.Contains(t, str, "500ms")
	})

	t.Run("failed result", func(t *testing.T) {
		result := NewEnrichmentResult("req-2")
		result.AddFailure("node-1", "stage", errors.New("error"), true)
		result.Duration = 200 * time.Millisecond

		str := result.String()
		assert.Contains(t, str, "completed=false")
		assert.Contains(t, str, "failures=1")
		assert.Contains(t, str, "200ms")
	})
}

func TestRetryConfig_NegativeBackoff(t *testing.T) {
	config := DefaultRetryConfig()

	// Negative attempt should use 0
	backoff := config.calculateBackoff(-1)
	assert.InDelta(t, 1*time.Second, backoff, float64(200*time.Millisecond))
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := NewCircuitBreaker()
	ctx := context.Background()

	// Concurrent executions
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			_ = cb.Execute(ctx, func() error {
				return nil
			})
		}()
	}

	// Wait for all to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should still be in closed state
	assert.Equal(t, "closed", cb.State)
}
