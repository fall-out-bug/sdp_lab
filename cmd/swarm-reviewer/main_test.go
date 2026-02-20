package main

import (
	"testing"
	"time"

	"sdp_dev/internal/observability"
)

func TestBuildReviewerObservabilityRecordsValidatorCompatible(t *testing.T) {
	records := buildReviewerObservabilityRecords(
		"sdp_dev-2aq.20.2",
		"publish",
		"fallback",
		"glm-5",
		1,
		true,
		false,
		".sdp/evidence/sdp_dev-2aq.20.2.json",
		"https://example.invalid/org/repo/pull/88",
		90*time.Millisecond,
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
	linkage, _ := event["linkage"].(map[string]any)
	if got, _ := linkage["pr_url"].(string); got == "" {
		t.Fatal("expected non-empty linkage.pr_url")
	}
}
