package architect

import (
	"fmt"
	"sync"
	"time"
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
			cb.halfOpenProbes = 0
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
}
