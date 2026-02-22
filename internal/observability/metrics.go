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

	modelSelectionTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sdp_model_selection_total",
		Help: "Model selection events by role, tier, model, and reason",
	}, []string{"role", "tier", "model", "reason"})

	llmUsageTokensTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sdp_llm_usage_tokens_total",
		Help: "LLM token usage by project, role, model (prompt + completion)",
	}, []string{"project", "role", "model", "type"})

	llmUsageCostUSD = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sdp_llm_usage_cost_usd",
		Help: "Estimated LLM cost in USD by project, role, model",
	}, []string{"project", "role", "model"})

	providerUsedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sdp_provider_used_total",
		Help: "Provider dispatch count by project, provider, model (WS-013-01)",
	}, []string{"project", "provider", "model"})

	costSavedUSD = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sdp_cost_saved_usd",
		Help: "Estimated cost saved by using direct subscription vs OpenRouter (WS-013-01)",
	}, []string{"project", "provider"})
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

// IncModelSelection records a model selection event (role, tier used, model, reason).
func IncModelSelection(role, tier, model, reason string) {
	if role == "" {
		role = "unknown"
	}
	if tier == "" {
		tier = "unknown"
	}
	if model == "" {
		model = "default"
	}
	if reason == "" {
		reason = "ok"
	}
	modelSelectionTotal.WithLabelValues(role, tier, model, reason).Inc()
}

// IncProviderUsed records a provider dispatch (project, provider, model).
func IncProviderUsed(project, provider, model string) {
	if project == "" {
		project = "default"
	}
	if provider == "" {
		provider = "unknown"
	}
	if model == "" {
		model = "default"
	}
	providerUsedTotal.WithLabelValues(project, provider, model).Inc()
}

// AddCostSaved records estimated cost saved when using direct subscription vs OpenRouter.
func AddCostSaved(project, provider string, amountUSD float64) {
	if project == "" {
		project = "default"
	}
	if provider == "" || amountUSD <= 0 {
		return
	}
	costSavedUSD.WithLabelValues(project, provider).Add(amountUSD)
}

// ObserveLLMUsage records token usage and estimated cost.
func ObserveLLMUsage(project, role, model string, promptTokens, completionTokens int, costUSD float64) {
	if project == "" {
		project = "default"
	}
	if role == "" {
		role = "unknown"
	}
	if model == "" {
		model = "default"
	}
	llmUsageTokensTotal.WithLabelValues(project, role, model, "prompt").Add(float64(promptTokens))
	llmUsageTokensTotal.WithLabelValues(project, role, model, "completion").Add(float64(completionTokens))
	if costUSD > 0 {
		llmUsageCostUSD.WithLabelValues(project, role, model).Add(costUSD)
	}
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
