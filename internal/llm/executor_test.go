package llm

import (
	"context"
	"testing"
	"time"
)

func TestExecuteFailsWhenOpencodeMissing(t *testing.T) {
	dir := t.TempDir()
	boundary := BoundarySpec{
		AllowedPathPrefixes:   []string{"internal/"},
		ForbiddenPathPrefixes: []string{".git/"},
		ControlPathPrefixes:   []string{".beads/"},
	}
	req := ExecuteRequest{
		IssueID:            "test-1",
		Title:              "Test",
		Description:        "Desc",
		Model:              "glm-4.7",
		WorkDir:            dir,
		Boundary:           boundary,
		Timeout:            5 * time.Second,
		OpencodeBinary:     "/nonexistent/opencode-binary-xyz",
	}
	_, err := Execute(context.Background(), req)
	if err == nil {
		t.Fatal("expected error when opencode binary not found")
	}
}

