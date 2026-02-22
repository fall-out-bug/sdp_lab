package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"sdp_dev/internal/observability"
)

func emitAutonomyObservability(issueID, phase, status, model string, startedAt time.Time) {
	evidenceLink := ""
	if strings.TrimSpace(issueID) != "" {
		evidenceLink = filepath.Join(".sdp", "evidence", issueID+".json")
	}
	_ = observability.EmitIntakeRecords(os.Stderr, observability.IntakeEventInput{
		RunID:               strings.TrimSpace(os.Getenv("SDP_RUN_ID")),
		IssueID:             issueID,
		Phase:               phase,
		Status:              status,
		Component:           "autonomy-worker",
		AgentRole:           "builder",
		ModelName:           model,
		Elapsed:             time.Since(startedAt),
		RetryCount:          0,
		FallbackUsed:        false,
		Escalated:           false,
		EvidenceContextLink: evidenceLink,
		PRURL:               "",
	})
}
