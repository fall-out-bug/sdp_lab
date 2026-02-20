package artifact

import (
	"strings"
	"testing"
	"time"
)

func TestVerifyIssuePassesForValidChain(t *testing.T) {
	bus := NewBusService()
	issueID := "sdp_dev-2aq.16.4"

	_, err := bus.Ingest(IngestRequest{
		IssueID:       issueID,
		ArtifactID:    "intent-001",
		ArtifactClass: "intent-brief",
		Phase:         "intake",
		Role:          "planner",
		CapturedAt:    "2026-02-20T19:00:00Z",
		Payload:       map[string]any{"trigger": "agent"},
	})
	if err != nil {
		t.Fatalf("ingest first: %v", err)
	}
	_, err = bus.Ingest(IngestRequest{
		IssueID:       issueID,
		ArtifactID:    "verify-001",
		ArtifactClass: "verification-report",
		Phase:         "verify",
		Role:          "verifier",
		CapturedAt:    "2026-02-20T19:10:00Z",
		Payload:       map[string]any{"gate_name": "go-test", "gate_status": "pass"},
	})
	if err != nil {
		t.Fatalf("ingest second: %v", err)
	}

	report := bus.VerifyIssue(issueID, time.Date(2026, 2, 21, 0, 0, 0, 0, time.UTC))
	if !report.OK() {
		t.Fatalf("expected verification to pass, got %+v", report)
	}
	if report.RecordsChecked != 2 || report.IndexRowsChecked != 2 {
		t.Fatalf("unexpected report counts: %+v", report)
	}
}

func TestVerifyIssueDetectsTamperAndIndexMismatch(t *testing.T) {
	bus := NewBusService()
	issueID := "sdp_dev-2aq.16.4.tamper"

	_, err := bus.Ingest(IngestRequest{
		IssueID:       issueID,
		ArtifactID:    "intent-001",
		ArtifactClass: "intent-brief",
		Phase:         "intake",
		Role:          "planner",
		CapturedAt:    "2026-02-20T19:00:00Z",
		Payload:       map[string]any{"trigger": "agent"},
	})
	if err != nil {
		t.Fatalf("ingest first: %v", err)
	}
	_, err = bus.Ingest(IngestRequest{
		IssueID:       issueID,
		ArtifactID:    "plan-001",
		ArtifactClass: "execution-plan",
		Phase:         "plan",
		Role:          "planner",
		CapturedAt:    "2026-02-20T19:02:00Z",
		Payload:       map[string]any{"depends_on": []string{"sdp_dev-2aq.16.3"}},
	})
	if err != nil {
		t.Fatalf("ingest second: %v", err)
	}

	bus.byIssue[issueID][1].Payload = []byte(`{"depends_on":["tampered"]}`)
	bus.provenanceIdx[issueID][1].HashPrev = "deadbeef"

	report := bus.VerifyIssue(issueID, time.Date(2026, 2, 21, 0, 0, 0, 0, time.UTC))
	if report.IntegrityOK() {
		t.Fatalf("expected tamper findings, got %+v", report)
	}
	if !containsFinding(report.TamperFindings, "payload digest mismatch") {
		t.Fatalf("expected payload digest mismatch finding, got %+v", report.TamperFindings)
	}
	if !containsFinding(report.TamperFindings, "index hash linkage mismatch") {
		t.Fatalf("expected index hash linkage mismatch finding, got %+v", report.TamperFindings)
	}
}

func TestVerifyIssueDetectsRetentionViolation(t *testing.T) {
	bus := NewBusService()
	issueID := "sdp_dev-2aq.16.4.retention"

	_, err := bus.Ingest(IngestRequest{
		IssueID:       issueID,
		ArtifactID:    "intent-001",
		ArtifactClass: "intent-brief",
		Phase:         "intake",
		Role:          "planner",
		CapturedAt:    "2023-01-01T00:00:00Z",
		Payload:       map[string]any{"trigger": "agent"},
	})
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}

	report := bus.VerifyIssue(issueID, time.Date(2026, 2, 21, 0, 0, 0, 0, time.UTC))
	if report.RetentionOK() {
		t.Fatalf("expected retention finding, got %+v", report)
	}
	if !containsFinding(report.RetentionFindings, "exceeded retention") {
		t.Fatalf("expected retention exceeded finding, got %+v", report.RetentionFindings)
	}
}

func containsFinding(findings []string, want string) bool {
	for _, finding := range findings {
		if strings.Contains(finding, want) {
			return true
		}
	}
	return false
}
