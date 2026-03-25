package beads

import (
	"encoding/json"
	"os/exec"
	"testing"
	"time"
)

func requireReadyCommand(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("bd"); err != nil {
		t.Skip("bd command not available")
	}
	if err := exec.Command("bd", "ready", "--json").Run(); err != nil {
		t.Skipf("bd ready not available in this environment: %v", err)
	}
}

func TestReadyCommand(t *testing.T) {
	requireReadyCommand(t)

	issues, err := ReadyCommand()
	if err != nil {
		t.Fatalf("ReadyCommand failed: %v", err)
	}

	if len(issues) == 0 {
		t.Log("No ready issues found (this is okay if backlog is empty)")
		return
	}

	// Check that issues have required fields
	for _, issue := range issues {
		if issue.ID == "" {
			t.Error("Issue ID is empty")
		}
		if issue.Title == "" {
			t.Error("Issue title is empty")
		}
		if issue.Status == "" {
			t.Error("Issue status is empty")
		}
		if issue.Priority < 1 {
			t.Error("Issue priority should be >= 1")
		}
	}

	t.Logf("Found %d ready issues", len(issues))
}

func TestReadyWithBlockersCommand(t *testing.T) {
	requireReadyCommand(t)

	issues, err := ReadyWithBlockersCommand()
	if err != nil {
		t.Fatalf("ReadyWithBlockersCommand failed: %v", err)
	}

	if len(issues) == 0 {
		t.Log("No ready issues found (this is okay if backlog is empty)")
		return
	}

	// Check that issues have required fields
	for _, issue := range issues {
		if issue.ID == "" {
			t.Error("Issue ID is empty")
		}
		if issue.Title == "" {
			t.Error("Issue title is empty")
		}
		// BlockedBy may be nil or empty for ready issues
	}

	t.Logf("Found %d ready issues with blocker info", len(issues))
}

func TestReadyIssueJSON(t *testing.T) {
	issue := ReadyIssue{
		Issue: Issue{
			ID:        "test-1",
			Title:     "Test Issue",
			Status:    "open",
			Priority:  1,
			Labels:    []string{"test"},
			BlockedBy: []string{"blocker-1"},
			CreatedAt: mustParseTime("2026-03-01T00:00:00Z"),
			UpdatedAt: mustParseTime("2026-03-01T00:00:00Z"),
		},
		WSID: "00-001-01",
	}

	data, err := json.Marshal(issue)
	if err != nil {
		t.Fatalf("Failed to marshal ReadyIssue: %v", err)
	}

	// Unmarshal and verify
	var parsed ReadyIssue
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal ReadyIssue: %v", err)
	}

	if parsed.ID != issue.ID {
		t.Errorf("ID mismatch: got %s, want %s", parsed.ID, issue.ID)
	}
	if parsed.Title != issue.Title {
		t.Errorf("Title mismatch: got %s, want %s", parsed.Title, issue.Title)
	}
	if parsed.WSID != issue.WSID {
		t.Errorf("WSID mismatch: got %s, want %s", parsed.WSID, issue.WSID)
	}
	if len(parsed.BlockedBy) != len(issue.BlockedBy) {
		t.Errorf("BlockedBy length mismatch: got %d, want %d", len(parsed.BlockedBy), len(issue.BlockedBy))
	}
}

func mustParseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}
