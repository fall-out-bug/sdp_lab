// Package observability provides shared Prometheus metrics for SDP components.
package observability

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	agentRunsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sdp_agent_runs_total",
		Help: "Total AgentRuns by project, status, and model",
	}, []string{"project", "status", "model"})

	agentRunDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "sdp_agent_run_duration_seconds",
		Help:    "AgentRun duration in seconds by project and role",
		Buckets: prometheus.DefBuckets,
	}, []string{"project", "role"})

	evidenceCompletenessRatio = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "sdp_evidence_completeness_ratio",
		Help: "Evidence completeness ratio (0-1) per project",
	}, []string{"project"})

	dispatchQueueDepth = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sdp_dispatch_queue_depth",
		Help: "Number of tasks in dispatch queue",
	})
)

// IncAgentRuns increments sdp_agent_runs_total for the given project, status, model.
func IncAgentRuns(project, status, model string) {
	if model == "" {
		model = "default"
	}
	agentRunsTotal.WithLabelValues(project, status, model).Inc()
}

// ObserveAgentRunDuration records sdp_agent_run_duration_seconds.
func ObserveAgentRunDuration(project, role string, d time.Duration) {
	agentRunDurationSeconds.WithLabelValues(project, role).Observe(d.Seconds())
}

// SetEvidenceCompleteness sets sdp_evidence_completeness_ratio for project.
func SetEvidenceCompleteness(project string, ratio float64) {
	evidenceCompletenessRatio.WithLabelValues(project).Set(ratio)
}

// SetDispatchQueueDepth sets sdp_dispatch_queue_depth.
func SetDispatchQueueDepth(n int) {
	dispatchQueueDepth.Set(float64(n))
}

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
