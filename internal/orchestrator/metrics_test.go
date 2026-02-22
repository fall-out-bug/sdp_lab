package orchestrator

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func TestSetActiveRuns(t *testing.T) {
	SetActiveRuns(5)
	SetActiveRuns(0)
	// No panic; gauge updated
}

func TestIncDispatched(t *testing.T) {
	IncDispatched()
	IncDispatched()
	// No panic
}

func TestIncEscalated(t *testing.T) {
	IncEscalated()
	// No panic
}

func TestServeMetrics_Handler(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("metrics: got status %d", rr.Code)
	}
	if rr.Body.Len() == 0 {
		t.Error("metrics body empty")
	}
}
