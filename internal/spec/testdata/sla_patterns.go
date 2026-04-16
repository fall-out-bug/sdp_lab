package testdata

import (
	"context"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

// SLAConfig holds timeout and retry configuration.
type SLAConfig struct {
	Timeout    time.Duration
	MaxRetries int
}

// NewHTTPClient creates a client with timeout.
func NewHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
	}
}

// RetryLoop demonstrates retry pattern with max retries.
func RetryLoop(fn func() error) error {
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		if err := fn(); err == nil {
			return nil
		}
	}
	return nil
}

// RateLimitedHandler demonstrates rate limiting.
func RateLimitedHandler() http.Handler {
	limiter := rate.NewLimiter(10, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			http.Error(w, "rate limited", http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	return mux
}

// HealthCheckHandler demonstrates a health endpoint.
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// ContextWithDeadline demonstrates context deadline setting.
func ContextWithDeadline() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Minute)
}
