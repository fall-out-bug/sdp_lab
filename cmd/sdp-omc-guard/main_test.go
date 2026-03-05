package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestGuardHook_E2E tests the sdp-omc-guard CLI end-to-end.
func TestGuardHook_E2E(t *testing.T) {
	// Build the binary
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "sdp-omc-guard")

	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = "."
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}

	// Create a mock workstream file with scope
	wsDir := filepath.Join(tmpDir, "docs", "workstreams", "backlog")
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		t.Fatalf("create ws dir: %v", err)
	}

	wsContent := `# Test Workstream

## Scope Files
- ` + "`" + `internal/session/event.go` + "`" + `
- ` + "`" + `internal/session/writer.go` + "`" + `
`
	wsPath := filepath.Join(wsDir, "00-059-01.md")
	if err := os.WriteFile(wsPath, []byte(wsContent), 0o644); err != nil {
		t.Fatalf("write ws file: %v", err)
	}

	// Create .sdp dir for evidence
	sdpDir := filepath.Join(tmpDir, ".sdp", "log")
	if err := os.MkdirAll(sdpDir, 0o755); err != nil {
		t.Fatalf("create .sdp dir: %v", err)
	}

	tests := []struct {
		name           string
		toolName       string
		toolInput      map[string]interface{}
		expectDecision string
		expectExitCode int
	}{
		{
			name:     "in_scope_file_allowed",
			toolName: "edit",
			toolInput: map[string]interface{}{
				"file_path": "internal/session/event.go",
			},
			expectDecision: "allow",
			expectExitCode: 0,
		},
		{
			name:     "out_of_scope_file_denied",
			toolName: "edit",
			toolInput: map[string]interface{}{
				"file_path": "cmd/other/main.go",
			},
			expectDecision: "deny",
			expectExitCode: 2,
		},
		{
			name:     "read_operation_no_files",
			toolName: "read",
			toolInput: map[string]interface{}{
				"file_path": "any/file.txt",
			},
			expectDecision: "allow",
			expectExitCode: 0,
		},
		{
			name:     "write_out_of_scope",
			toolName: "write",
			toolInput: map[string]interface{}{
				"file_path": "external/path.txt",
			},
			expectDecision: "deny",
			expectExitCode: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := PreToolUseInput{
				SessionID:     "test-session-001",
				CWD:           tmpDir,
				HookEventName: "PreToolUse",
				ToolName:      tt.toolName,
				ToolInput:     tt.toolInput,
			}

			inputJSON, err := json.Marshal(input)
			if err != nil {
				t.Fatalf("marshal input: %v", err)
			}

			cmd := exec.Command(binaryPath, "--ws", "00-059-01", "--session-id", "test-session-001", "--emit-evidence=false")
			cmd.Dir = tmpDir
			cmd.Stdin = strings.NewReader(string(inputJSON))

			output, err := cmd.Output()
			exitCode := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				} else {
					t.Fatalf("run command: %v", err)
				}
			}

			if exitCode != tt.expectExitCode {
				t.Errorf("expected exit code %d, got %d\noutput: %s", tt.expectExitCode, exitCode, output)
			}

			// Parse result
			var result GuardResult
			if err := json.Unmarshal(output, &result); err != nil {
				t.Fatalf("parse result: %v\noutput: %s", err, output)
			}

			if result.Decision != tt.expectDecision {
				t.Errorf("expected decision %q, got %q", tt.expectDecision, result.Decision)
			}
		})
	}
}

// TestGuardHook_EmitEvidence tests that guard_check events are emitted.
func TestGuardHook_EmitEvidence(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping evidence test in short mode")
	}

	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "sdp-omc-guard")

	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = "."
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}

	// Create workstream
	wsDir := filepath.Join(tmpDir, "docs", "workstreams", "backlog")
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		t.Fatalf("create ws dir: %v", err)
	}

	wsContent := `# Test Workstream

## Scope Files
- ` + "`" + `internal/session/event.go` + "`" + `
`
	wsPath := filepath.Join(wsDir, "00-059-01.md")
	if err := os.WriteFile(wsPath, []byte(wsContent), 0o644); err != nil {
		t.Fatalf("write ws file: %v", err)
	}

	input := PreToolUseInput{
		SessionID:     "evidence-test-001",
		CWD:           tmpDir,
		HookEventName: "PreToolUse",
		ToolName:      "edit",
		ToolInput: map[string]interface{}{
			"file_path": "internal/session/event.go",
		},
	}

	inputJSON, _ := json.Marshal(input)

	cmd := exec.Command(binaryPath, "--ws", "00-059-01", "--session-id", "evidence-test-001", "--emit-evidence")
	cmd.Dir = tmpDir
	cmd.Stdin = strings.NewReader(string(inputJSON))
	_, _ = cmd.Output() // Ignore exit code for this test

	// Check evidence file was created
	evidencePath := filepath.Join(tmpDir, ".sdp", "log", "session-evidence-test-001.jsonl")
	content, err := os.ReadFile(evidencePath)
	if err != nil {
		t.Fatalf("read evidence file: %v", err)
	}

	// Verify it contains a guard_check event
	if !strings.Contains(string(content), `"type":"guard_check"`) {
		t.Errorf("evidence should contain guard_check event\ncontent: %s", content)
	}
}
