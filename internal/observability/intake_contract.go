package observability

import (
	"fmt"
	"sort"
	"strings"
)

const UnifiedMetricsTraceSchemaVersion = "observability-metrics-trace/v1"

type FieldSpec struct {
	Path        string
	Type        string
	Domain      string
	Required    bool
	Description string
}

type UnifiedMetricsTraceSchema struct {
	ContractVersion     string
	RequiredFields      []FieldSpec
	AllowedStatus       []string
	LatencyBucketLabels []string
}

var requiredFields = []FieldSpec{
	{Path: "trace.run_id", Type: "string", Domain: "protocol", Required: true, Description: "Deterministic run identifier for trace and metric joins."},
	{Path: "protocol.issue_id", Type: "string", Domain: "protocol", Required: true, Description: "Beads issue identifier for the execution flow."},
	{Path: "protocol.phase", Type: "string", Domain: "protocol", Required: true, Description: "Current protocol phase (intake/plan/execute/verify/review/publish)."},
	{Path: "protocol.status", Type: "string", Domain: "protocol", Required: true, Description: "Terminal or in-flight state for the current protocol event."},
	{Path: "system.component", Type: "string", Domain: "system", Required: true, Description: "Emitter component name (opencode-agent/swarm-worker/swarm-reviewer)."},
	{Path: "system.agent_role", Type: "string", Domain: "system", Required: true, Description: "Role emitting the event (orchestrator/worker/reviewer)."},
	{Path: "model.name", Type: "string", Domain: "model", Required: true, Description: "Model selected for the decision or action."},
	{Path: "metrics.latency_bucket", Type: "string", Domain: "system", Required: true, Description: "Latency bucket label for SLI/SLO aggregation."},
	{Path: "resilience.retry_count", Type: "integer", Domain: "protocol", Required: true, Description: "Retry attempts consumed for the event."},
	{Path: "resilience.fallback_used", Type: "boolean", Domain: "model", Required: true, Description: "Whether fallback model/route handling was applied."},
	{Path: "resilience.escalated", Type: "boolean", Domain: "protocol", Required: true, Description: "Whether handling escalated beyond normal automation."},
	{Path: "linkage.evidence_context_link", Type: "string", Domain: "protocol", Required: true, Description: "Evidence artifact pointer for drill-down."},
	{Path: "linkage.pr_url", Type: "string", Domain: "protocol", Required: true, Description: "PR URL linked to the run, if published."},
}

var allowedStatus = []string{
	"running",
	"success",
	"failed",
	"blocked",
	"retrying",
	"fallback",
	"escalated",
}

var latencyBucketLabels = []string{
	"le_50ms",
	"le_100ms",
	"le_250ms",
	"le_500ms",
	"le_1000ms",
	"le_2500ms",
	"le_5000ms",
	"gt_5000ms",
}

func DefaultUnifiedMetricsTraceSchema() UnifiedMetricsTraceSchema {
	fields := append([]FieldSpec(nil), requiredFields...)
	sort.Slice(fields, func(i, j int) bool { return fields[i].Path < fields[j].Path })
	statuses := append([]string(nil), allowedStatus...)
	sort.Strings(statuses)
	buckets := append([]string(nil), latencyBucketLabels...)

	return UnifiedMetricsTraceSchema{
		ContractVersion:     UnifiedMetricsTraceSchemaVersion,
		RequiredFields:      fields,
		AllowedStatus:       statuses,
		LatencyBucketLabels: buckets,
	}
}

func ValidateUnifiedMetricsTraceEvent(event map[string]any) []string {
	schema := DefaultUnifiedMetricsTraceSchema()
	errs := make([]string, 0)
	for _, field := range schema.RequiredFields {
		value, ok := valueAtPath(event, field.Path)
		if !ok || isEmptyValue(value) {
			errs = append(errs, fmt.Sprintf("missing required field: %s", field.Path))
		}
	}

	if statusValue, ok := valueAtPath(event, "protocol.status"); ok {
		status, _ := statusValue.(string)
		if !contains(schema.AllowedStatus, strings.TrimSpace(status)) {
			errs = append(errs, fmt.Sprintf("invalid protocol.status: %q", status))
		}
	}

	if bucketValue, ok := valueAtPath(event, "metrics.latency_bucket"); ok {
		bucket, _ := bucketValue.(string)
		if !contains(schema.LatencyBucketLabels, strings.TrimSpace(bucket)) {
			errs = append(errs, fmt.Sprintf("invalid metrics.latency_bucket: %q", bucket))
		}
	}

	sort.Strings(errs)
	return errs
}

func valueAtPath(payload map[string]any, path string) (any, bool) {
	if len(path) == 0 {
		return nil, false
	}
	parts := strings.Split(path, ".")
	var current any = payload
	for _, part := range parts {
		node, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		value, exists := node[part]
		if !exists {
			return nil, false
		}
		current = value
	}
	return current, true
}

func isEmptyValue(value any) bool {
	switch v := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(v) == ""
	default:
		return false
	}
}

func contains(items []string, candidate string) bool {
	for _, item := range items {
		if item == candidate {
			return true
		}
	}
	return false
}
