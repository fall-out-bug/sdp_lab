package architect

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
)

// RetryConfig configures exponential backoff with jitter for LLM API calls.
type RetryConfig struct {
	MaxRetries int           // default: 3
	BaseDelay  time.Duration // default: 1s
	MaxDelay   time.Duration // default: 30s
}

// NewRetryConfig returns a RetryConfig with sensible defaults.
func NewRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries: 3,
		BaseDelay:  1 * time.Second,
		MaxDelay:   30 * time.Second,
	}
}

// CircuitBreaker tracks per-provider failures and prevents calls when open.
type CircuitBreaker struct {
	mu               sync.Mutex
	Failures         int           // consecutive failures
	FailureThreshold int           // default: 5
	CooldownPeriod   time.Duration // default: 30s
	LastFailure      time.Time
	State            string // "closed", "open", "half-open"
	HalfOpenMax      int    // max concurrent probes in half-open state, default: 1
	halfOpenProbes   int
}

// NewCircuitBreaker returns a CircuitBreaker with sensible defaults in closed state.
func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{
		Failures:         0,
		FailureThreshold: 5,
		CooldownPeriod:   30 * time.Second,
		LastFailure:      time.Time{},
		State:            "closed",
		HalfOpenMax:      1,
		halfOpenProbes:   0,
	}
}

// Allow returns true if the circuit breaker allows a request through.
// In "closed" state, all requests are allowed.
// In "open" state, requests are blocked until the cooldown period elapses,
// at which point the breaker transitions to "half-open" and allows one request.
// In "half-open" state, one request is allowed to probe the service.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	switch cb.State {
	case "closed":
		return true
	case "open":
		if time.Since(cb.LastFailure) >= cb.CooldownPeriod {
			cb.State = "half-open"
			cb.halfOpenProbes = 1 // Count the transition as a probe
			return true
		}
		return false
	case "half-open":
		if cb.halfOpenProbes >= cb.HalfOpenMax {
			return false
		}
		cb.halfOpenProbes++
		return true
	default:
		return true
	}
}

// RecordSuccess resets the failure count and transitions back to closed.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.Failures = 0
	cb.State = "closed"
	cb.halfOpenProbes = 0
}

// RecordFailure increments the failure count. If the threshold is reached,
// the breaker transitions to "open" state.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.Failures++
	cb.LastFailure = time.Now()
	cb.halfOpenProbes = 0
	if cb.Failures >= cb.FailureThreshold {
		cb.State = "open"
	}
}

// EnrichmentError captures a per-node failure within the enrichment pipeline.
type EnrichmentError struct {
	NodeID    string // which node failed
	Stage     string // "scrub", "sanitize", "wrap", "api", "validate"
	Retriable bool   // true = transient, false = permanent
	Err       error
}

// Error implements the error interface for EnrichmentError.
func (e EnrichmentError) Error() string {
	return fmt.Sprintf("enrichment error: node=%s stage=%s retriable=%v: %v",
		e.NodeID, e.Stage, e.Retriable, e.Err)
}

// Unwrap returns the underlying error for errors.Is/As compatibility.
func (e EnrichmentError) Unwrap() error {
	return e.Err
}

// EnrichmentResult holds the outcome of the LLM enrichment pipeline.
// Completed is true only if ALL nodes were processed successfully.
type EnrichmentResult struct {
	Completed  bool                     // true only if ALL nodes processed
	Enrichment map[string]LLMEnrichment // successfully processed nodes only
	Failed     []EnrichmentError        // per-node failures
	RequestID  string                   // unique identifier for this enrichment request
	Duration   time.Duration            // how long the enrichment took
}

// CircuitState represents the state of a circuit breaker.
type CircuitState int

const (
	// StateClosed means the circuit is closed and requests are allowed.
	StateClosed CircuitState = iota

	// StateOpen means the circuit is open and requests are blocked.
	StateOpen

	// StateHalfOpen means the circuit is half-open and a test request is allowed.
	StateHalfOpen
)

// String returns the string representation of the circuit state.
func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// calculateBackoff calculates the backoff duration with jitter for RetryConfig.
func (rc RetryConfig) calculateBackoff(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}

	// Exponential backoff
	backoff := rc.BaseDelay * time.Duration(1<<uint(attempt))
	if backoff > rc.MaxDelay {
		backoff = rc.MaxDelay
	}

	// Add jitter (+/- 20%)
	jitter := int64(float64(backoff) * 0.2 * (rand.Float64()*2 - 1))
	backoff += time.Duration(jitter)

	return backoff
}

// CircuitBreakerError represents an error from the circuit breaker.
type CircuitBreakerError struct {
	Provider   string
	State      CircuitState
	Message    string
	RetryAfter time.Duration
}

func (e *CircuitBreakerError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("circuit breaker for provider %q is %s: %s (retry after %v)",
			e.Provider, e.State, e.Message, e.RetryAfter)
	}
	return fmt.Sprintf("circuit breaker for provider %q is %s: %s",
		e.Provider, e.State, e.Message)
}

// Execute wraps the existing CircuitBreaker.Allow method in a Execute interface.
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	// Check if we should allow execution
	if !cb.Allow() {
		return &CircuitBreakerError{
			Provider: "unknown",
			State:    StateOpen,
			Message:  "circuit breaker is open",
			RetryAfter: cb.CooldownPeriod - time.Since(cb.LastFailure),
		}
	}

	// Execute the function
	err := fn()

	// Update circuit state based on result
	if err != nil {
		cb.RecordFailure()
	} else {
		cb.RecordSuccess()
	}

	return err
}

// DefaultRetryConfig returns a default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries: 3,
		BaseDelay:  1 * time.Second,
		MaxDelay:   30 * time.Second,
	}
}

// DefaultCircuitBreakerConfig returns default circuit breaker configuration.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		CooldownPeriod:   30 * time.Second,
	}
}

// CircuitBreakerConfig configures circuit breaker behavior.
type CircuitBreakerConfig struct {
	FailureThreshold int
	CooldownPeriod   time.Duration
}

// NewEnrichmentResult creates a new enrichment result.
func NewEnrichmentResult(requestID string) *EnrichmentResult {
	return &EnrichmentResult{
		Completed:   false,
		Enrichment:  make(map[string]LLMEnrichment),
		Failed:      make([]EnrichmentError, 0),
		RequestID:   requestID,
	}
}

// Complete marks the enrichment as completed.
func (r *EnrichmentResult) Complete() {
	r.Completed = true
}

// AddFailure adds a failure to the result.
func (r *EnrichmentResult) AddFailure(nodeID, stage string, err error, retriable bool) {
	r.Failed = append(r.Failed, EnrichmentError{
		NodeID:    nodeID,
		Stage:     stage,
		Err:       err,
		Retriable: retriable,
	})
}

// HasFailures returns if there were any failures.
func (r *EnrichmentResult) HasFailures() bool {
	return len(r.Failed) > 0
}

// HasRetriableFailures returns if there are any retriable failures.
func (r *EnrichmentResult) HasRetriableFailures() bool {
	for _, f := range r.Failed {
		if f.Retriable {
			return true
		}
	}
	return false
}

// GetRetriableFailures returns all retriable failures.
func (r *EnrichmentResult) GetRetriableFailures() []EnrichmentError {
	var retriable []EnrichmentError
	for _, f := range r.Failed {
		if f.Retriable {
			retriable = append(retriable, f)
		}
	}
	return retriable
}

// Merge merges another enrichment result into this one.
func (r *EnrichmentResult) Merge(other *EnrichmentResult) {
	for k, v := range other.Enrichment {
		r.Enrichment[k] = v
	}
	r.Failed = append(r.Failed, other.Failed...)
	r.Duration += other.Duration
	if !other.Completed {
		r.Completed = false
	}
}

// String returns a string representation of the result.
func (r *EnrichmentResult) String() string {
	if r.Completed {
		return fmt.Sprintf("EnrichmentResult{completed=true, request_id=%s, duration=%v}",
			r.RequestID, r.Duration)
	}
	return fmt.Sprintf("EnrichmentResult{completed=false, request_id=%s, failures=%d, duration=%v}",
		r.RequestID, len(r.Failed), r.Duration)
}

// GenerateRequestID generates a unique request ID for enrichment.
func GenerateRequestID() string {
	return uuid.New().String()
}

// RetryWithBackoff executes a function with retry and exponential backoff.
func RetryWithBackoff(ctx context.Context, config RetryConfig, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			backoff := config.calculateBackoff(attempt - 1)
			select {
			case <-time.After(backoff):
				// Continue with retry
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Execute function
		err := fn()
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if we should retry
		if attempt == config.MaxRetries {
			break // Max retries reached
		}
	}

	return fmt.Errorf("max retries (%d) reached, last error: %w", config.MaxRetries, lastErr)
}

// SafeExecutor provides safe execution with retry and circuit breaker.
type SafeExecutor struct {
	retryConfig  RetryConfig
	breakerMgr   *CircuitBreakerManager
}

// CircuitBreakerManager manages circuit breakers for multiple providers.
type CircuitBreakerManager struct {
	mu       sync.RWMutex
	breakers map[string]*CircuitBreaker
	config   CircuitBreakerConfig
}

// NewCircuitBreakerManager creates a new circuit breaker manager.
func NewCircuitBreakerManager(config CircuitBreakerConfig) *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[string]*CircuitBreaker),
		config:   config,
	}
}

// GetOrCreate gets or creates a circuit breaker for a provider.
func (m *CircuitBreakerManager) GetOrCreate(provider string) *CircuitBreaker {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cb, exists := m.breakers[provider]; exists {
		return cb
	}

	cb := NewCircuitBreaker()
	cb.FailureThreshold = m.config.FailureThreshold
	cb.CooldownPeriod = m.config.CooldownPeriod
	m.breakers[provider] = cb
	return cb
}

// NewSafeExecutor creates a new safe executor.
func NewSafeExecutor(retryConfig RetryConfig, breakerConfig CircuitBreakerConfig) *SafeExecutor {
	return &SafeExecutor{
		retryConfig: retryConfig,
		breakerMgr:  NewCircuitBreakerManager(breakerConfig),
	}
}

// Execute executes a function with retry and circuit breaker protection.
func (se *SafeExecutor) Execute(ctx context.Context, provider string, fn func() error) error {
	breaker := se.breakerMgr.GetOrCreate(provider)

	// Wrap function with circuit breaker
	wrappedFn := func() error {
		return breaker.Execute(ctx, fn)
	}

	// Execute with retry
	return RetryWithBackoff(ctx, se.retryConfig, wrappedFn)
}

// GetCircuitBreakerState returns the current state of a provider's circuit breaker.
func (se *SafeExecutor) GetCircuitBreakerState(provider string) string {
	breaker := se.breakerMgr.GetOrCreate(provider)
	return breaker.State
}

// IsCircuitBreakerError checks if an error is a circuit breaker error.
func IsCircuitBreakerError(err error) bool {
	var cbErr *CircuitBreakerError
	return errors.As(err, &cbErr)
}
