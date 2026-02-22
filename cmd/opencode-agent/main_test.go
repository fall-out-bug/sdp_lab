package main

import (
	"testing"
	"time"

	"sdp_dev/internal/observability"
)

func TestBuildOpencodeObservabilityRecordsValidatorCompatible(t *testing.T) {
	records := buildOpencodeObservabilityRecords(
		"sdp_dev-2aq.20.2",
		"glm-4.7",
		"retrying",
		2,
		true,
		false,
		".sdp/evidence/sdp_dev-2aq.20.2.json",
		"https://example.invalid/org/repo/pull/21",
		130*time.Millisecond,
	)

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	event, ok := records[0]["event"].(map[string]any)
	if !ok {
		t.Fatalf("missing event payload: %#v", records[0])
	}
	if errs := observability.ValidateUnifiedMetricsTraceEvent(event); len(errs) != 0 {
		t.Fatalf("event failed schema validation: %v", errs)
	}
	model, _ := event["model"].(map[string]any)
	if got, _ := model["name"].(string); got != "glm-4.7" {
		t.Fatalf("unexpected model tag: %q", got)
	}
}
