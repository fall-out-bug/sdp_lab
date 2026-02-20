package observability

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

type IntakeEventInput struct {
	RunID               string
	IssueID             string
	Phase               string
	Status              string
	Component           string
	AgentRole           string
	ModelName           string
	Elapsed             time.Duration
	RetryCount          int
	FallbackUsed        bool
	Escalated           bool
	EvidenceContextLink string
	PRURL               string
}

func BuildIntakeEvent(input IntakeEventInput) map[string]any {
	issueID := strings.TrimSpace(input.IssueID)
	if issueID == "" {
		issueID = "unknown"
	}
	phase := strings.TrimSpace(input.Phase)
	if phase == "" {
		phase = "execute"
	}
	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = "running"
	}
	runID := strings.TrimSpace(input.RunID)
	if runID == "" {
		runID = fmt.Sprintf("run-%d", time.Now().UTC().UnixNano())
	}
	component := strings.TrimSpace(input.Component)
	if component == "" {
		component = "unknown"
	}
	agentRole := strings.TrimSpace(input.AgentRole)
	if agentRole == "" {
		agentRole = "unknown"
	}
	model := strings.TrimSpace(input.ModelName)
	if model == "" {
		model = "unknown"
	}
	evidenceContextLink := strings.TrimSpace(input.EvidenceContextLink)
	if evidenceContextLink == "" {
		evidenceContextLink = "unknown"
	}
	prURL := strings.TrimSpace(input.PRURL)
	if prURL == "" {
		prURL = "unknown"
	}

	latencyBucket := latencyBucket(input.Elapsed)

	return map[string]any{
		"trace": map[string]any{
			"run_id": runID,
		},
		"protocol": map[string]any{
			"issue_id": issueID,
			"phase":    phase,
			"status":   status,
		},
		"system": map[string]any{
			"component":  component,
			"agent_role": agentRole,
		},
		"model": map[string]any{
			"name": model,
		},
		"metrics": map[string]any{
			"latency_bucket": latencyBucket,
		},
		"resilience": map[string]any{
			"retry_count":   input.RetryCount,
			"fallback_used": input.FallbackUsed,
			"escalated":     input.Escalated,
		},
		"linkage": map[string]any{
			"evidence_context_link": evidenceContextLink,
			"pr_url":                prURL,
		},
	}
}

func BuildIntakeRecords(input IntakeEventInput) []map[string]any {
	event := BuildIntakeEvent(input)
	emittedAt := time.Now().UTC().Format(time.RFC3339Nano)
	elapsedMs := input.Elapsed.Milliseconds()
	if elapsedMs < 0 {
		elapsedMs = 0
	}

	eventRecord := map[string]any{
		"record_type":      "event",
		"contract_version": UnifiedMetricsTraceSchemaVersion,
		"emitted_at":       emittedAt,
		"event":            event,
	}

	metricRecord := map[string]any{
		"record_type":      "metric",
		"contract_version": UnifiedMetricsTraceSchemaVersion,
		"emitted_at":       emittedAt,
		"event":            event,
		"metric": map[string]any{
			"name":     "protocol_phase_latency_ms",
			"value_ms": elapsedMs,
		},
	}

	return []map[string]any{eventRecord, metricRecord}
}

func EmitIntakeRecords(writer io.Writer, input IntakeEventInput) error {
	records := BuildIntakeRecords(input)
	enc := json.NewEncoder(writer)
	for _, record := range records {
		if err := enc.Encode(record); err != nil {
			return err
		}
	}
	return nil
}

func latencyBucket(elapsed time.Duration) string {
	ms := elapsed.Milliseconds()
	if ms <= 50 {
		return "le_50ms"
	}
	if ms <= 100 {
		return "le_100ms"
	}
	if ms <= 250 {
		return "le_250ms"
	}
	if ms <= 500 {
		return "le_500ms"
	}
	if ms <= 1000 {
		return "le_1000ms"
	}
	if ms <= 2500 {
		return "le_2500ms"
	}
	if ms <= 5000 {
		return "le_5000ms"
	}
	return "gt_5000ms"
}
