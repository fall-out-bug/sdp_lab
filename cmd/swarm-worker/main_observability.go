package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sdp_dev/internal/observability"
)

func emitWorkerObservability(issueID string, phase string, status string, model string, startedAt time.Time, retryCount int, fallbackUsed bool, escalated bool, evidenceContextLink string, prURL string) {
	if strings.TrimSpace(evidenceContextLink) == "" && strings.TrimSpace(issueID) != "" {
		evidenceContextLink = filepath.Join(".sdp", "evidence", issueID+".json")
	}
	_ = observability.EmitIntakeRecords(os.Stderr, observability.IntakeEventInput{
		RunID:               strings.TrimSpace(os.Getenv("SDP_RUN_ID")),
		IssueID:             issueID,
		Phase:               phase,
		Status:              status,
		Component:           "swarm-worker",
		AgentRole:           "worker",
		ModelName:           model,
		Elapsed:             time.Since(startedAt),
		RetryCount:          retryCount,
		FallbackUsed:        fallbackUsed,
		Escalated:           escalated,
		EvidenceContextLink: evidenceContextLink,
		PRURL:               prURL,
	})
}

func buildWorkerObservabilityRecords(issueID string, phase string, status string, model string, retryCount int, fallbackUsed bool, escalated bool, evidenceContextLink string, prURL string, elapsed time.Duration) []map[string]any {
	return observability.BuildIntakeRecords(observability.IntakeEventInput{
		RunID:               "worker-run-test",
		IssueID:             issueID,
		Phase:               phase,
		Status:              status,
		Component:           "swarm-worker",
		AgentRole:           "worker",
		ModelName:           model,
		Elapsed:             elapsed,
		RetryCount:          retryCount,
		FallbackUsed:        fallbackUsed,
		Escalated:           escalated,
		EvidenceContextLink: evidenceContextLink,
		PRURL:               prURL,
	})
}

func extractLinkage(issueID string) (string, string) {
	path := filepath.Join(".sdp", "evidence", issueID+".json")
	b, err := os.ReadFile(path)
	if err != nil {
		return path, ""
	}
	var payload map[string]any
	if err := json.Unmarshal(b, &payload); err != nil {
		return path, ""
	}
	trace, _ := payload["trace"].(map[string]any)
	if trace == nil {
		return path, ""
	}
	evidence, _ := trace["evidence_context_link"].(string)
	if strings.TrimSpace(evidence) == "" {
		evidence = path
	}
	prURL, _ := trace["pr_url"].(string)
	return evidence, strings.TrimSpace(prURL)
}
