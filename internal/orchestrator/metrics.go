// Package orchestrator provides Prometheus metrics for feature-orchestrator.
package orchestrator

import (
	"context"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	activeRunsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sdp_feature_orchestrator_active_runs",
		Help: "Number of active AgentRuns (Running, Pending, etc.)",
	})
	dispatchedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sdp_feature_orchestrator_dispatched_total",
		Help: "Total AgentRuns dispatched",
	})
	escalatedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sdp_feature_orchestrator_escalated_total",
		Help: "Total AgentRuns escalated (timeout)",
	})
)

// SetActiveRuns updates the active runs gauge.
func SetActiveRuns(n int) { activeRunsGauge.Set(float64(n)) }

// IncDispatched increments the dispatched counter.
func IncDispatched() { dispatchedTotal.Inc() }

// IncEscalated increments the escalated counter.
func IncEscalated() { escalatedTotal.Inc() }

// ServeMetrics starts HTTP server for /metrics on addr (e.g. ":8080").
func ServeMetrics(ctx context.Context, addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	srv := &http.Server{Addr: addr, Handler: mux}
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()
	return srv.ListenAndServe()
}
