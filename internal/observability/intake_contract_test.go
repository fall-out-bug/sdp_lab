package observability

import (
	"strings"
	"testing"
)

func TestDefaultUnifiedMetricsTraceSchemaCoverage(t *testing.T) {
	schema := DefaultUnifiedMetricsTraceSchema()
	if schema.ContractVersion != UnifiedMetricsTraceSchemaVersion {
		t.Fatalf("unexpected contract version: got=%s want=%s", schema.ContractVersion, UnifiedMetricsTraceSchemaVersion)
	}

	required := map[string]string{
		"trace.run_id":                  "string",
		"protocol.issue_id":             "string",
		"protocol.phase":                "string",
		"protocol.status":               "string",
		"system.component":              "string",
		"system.agent_role":             "string",
		"model.name":                    "string",
		"metrics.latency_bucket":        "string",
		"resilience.retry_count":        "integer",
		"resilience.fallback_used":      "boolean",
		"resilience.escalated":          "boolean",
		"linkage.evidence_context_link": "string",
		"linkage.pr_url":                "string",
	}

	seen := map[string]struct{}{}
	for _, field := range schema.RequiredFields {
		if field.Path == "" || field.Type == "" {
			t.Fatalf("invalid field spec: %+v", field)
		}
		if _, ok := seen[field.Path]; ok {
			t.Fatalf("duplicate required field path: %s", field.Path)
		}
		seen[field.Path] = struct{}{}
		if wantType, ok := required[field.Path]; ok {
			if field.Type != wantType {
				t.Fatalf("unexpected type for %s: got=%s want=%s", field.Path, field.Type, wantType)
			}
		}
	}

	for path := range required {
		if _, ok := seen[path]; !ok {
			t.Fatalf("missing required field: %s", path)
		}
	}

	if len(schema.AllowedStatus) == 0 {
		t.Fatal("expected allowed statuses")
	}
	if len(schema.LatencyBucketLabels) < 2 {
		t.Fatal("expected latency bucket coverage")
	}
}

func TestValidateUnifiedMetricsTraceEvent(t *testing.T) {
	valid := map[string]any{
		"trace": map[string]any{"run_id": "run-42"},
		"protocol": map[string]any{
			"issue_id": "sdp_dev-2aq.20.1",
			"phase":    "intake",
			"status":   "success",
		},
		"system": map[string]any{
			"component":  "opencode-agent",
			"agent_role": "orchestrator",
		},
		"model": map[string]any{"name": "glm-5"},
		"metrics": map[string]any{
			"latency_bucket": "le_250ms",
		},
		"resilience": map[string]any{
			"retry_count":   0,
			"fallback_used": false,
			"escalated":     false,
		},
		"linkage": map[string]any{
			"evidence_context_link": "artifact://runs/sdp_dev-2aq.20.1/evidence.json",
			"pr_url":                "https://example.invalid/org/repo/pull/42",
		},
	}

	if errs := ValidateUnifiedMetricsTraceEvent(valid); len(errs) != 0 {
		t.Fatalf("expected valid payload, got errors: %v", errs)
	}

	invalid := map[string]any{
		"trace": map[string]any{"run_id": ""},
		"protocol": map[string]any{
			"issue_id": "",
			"phase":    "verify",
			"status":   "unknown",
		},
		"system": map[string]any{"component": "swarm-worker"},
		"model":  map[string]any{},
		"metrics": map[string]any{
			"latency_bucket": "le_999ms",
		},
		"resilience": map[string]any{
			"retry_count": 2,
		},
		"linkage": map[string]any{
			"pr_url": "",
		},
	}

	errs := ValidateUnifiedMetricsTraceEvent(invalid)
	if len(errs) < 6 {
		t.Fatalf("expected multiple validation errors, got=%v", errs)
	}

	requiredSubstrings := []string{
		"missing required field: trace.run_id",
		"missing required field: protocol.issue_id",
		"missing required field: system.agent_role",
		"missing required field: model.name",
		"missing required field: resilience.fallback_used",
		"missing required field: resilience.escalated",
		"missing required field: linkage.evidence_context_link",
		"missing required field: linkage.pr_url",
		"invalid metrics.latency_bucket",
		"invalid protocol.status",
	}

	for _, expected := range requiredSubstrings {
		if !containsError(errs, expected) {
			t.Fatalf("expected validation error containing %q, got=%v", expected, errs)
		}
	}
}

func containsError(errs []string, want string) bool {
	for _, err := range errs {
		if strings.Contains(err, want) {
			return true
		}
	}
	return false
}
