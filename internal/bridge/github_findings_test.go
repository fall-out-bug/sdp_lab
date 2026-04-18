package bridge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// --- validateLabel tests ---

func TestValidateLabel_Valid(t *testing.T) {
	valid := []string{
		"bug",
		"enhancement",
		"P0-critical",
		"area/docs",
		"team_backend",
		"sdp.v2",
		"release/1.0",
	}
	for _, label := range valid {
		if err := validateLabel(label); err != nil {
			t.Errorf("validateLabel(%q) returned unexpected error: %v", label, err)
		}
	}
}

func TestValidateLabel_Empty(t *testing.T) {
	if err := validateLabel(""); err == nil {
		t.Error("validateLabel(\"\") should return error for empty label")
	}
}

func TestValidateLabel_Newlines(t *testing.T) {
	if err := validateLabel("bug\ninjection"); err == nil {
		t.Error("validateLabel with newline should return error")
	}
}

func TestValidateLabel_NullBytes(t *testing.T) {
	if err := validateLabel("bug\x00evil"); err == nil {
		t.Error("validateLabel with null byte should return error")
	}
}

func TestValidateLabel_InvalidChars(t *testing.T) {
	invalid := []string{
		"bug injection",     // space
		"label;rm -rf /",    // semicolon and spaces
		"$(whoami)",         // shell expansion
		"`cmd`",             // backtick
		"label's",           // single quote
		"label\"s",          // double quote
		"bug&flag",          // ampersand
	}
	for _, label := range invalid {
		if err := validateLabel(label); err == nil {
			t.Errorf("validateLabel(%q) should return error for invalid characters", label)
		}
	}
}

// --- ParseFindingsFile tests ---

const protocolFindingsJSON = `{
  "spec_version": "1.0",
  "findings_id": "pf-001",
  "timestamp": "2026-04-18T12:00:00Z",
  "source": {
    "check_name": "sdp-protocol-check",
    "workflow": "ci.yml",
    "run_id": 12345
  },
  "findings": [
    {
      "finding_key": "PROTO-001",
      "severity": "error",
      "category": "naming",
      "file": "test.go",
      "line": 10,
      "message": "bad name"
    }
  ],
  "summary": {
    "total": 1,
    "by_severity": {"error": 1},
    "by_category": {"naming": 1}
  }
}`

const docsFindingsJSON = `{
  "spec_version": "1.0",
  "findings_id": "df-001",
  "timestamp": "2026-04-18T12:00:00Z",
  "source": {
    "check_name": "sdp-doc-sync",
    "workflow": "docs.yml",
    "run_id": 67890
  },
  "findings": [
    {
      "finding_key": "DOCS-001",
      "severity": "warning",
      "category": "broken-link",
      "file": "README.md",
      "line": 5,
      "message": "link broken"
    }
  ],
  "summary": {
    "total": 1,
    "by_severity": {"warning": 1},
    "by_category": {"broken-link": 1}
  }
}`

func TestParseFindingsFile_Protocol(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "protocol.json")
	if err := os.WriteFile(path, []byte(protocolFindingsJSON), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	result, kind, err := ParseFindingsFile(path)
	if err != nil {
		t.Fatalf("ParseFindingsFile() error: %v", err)
	}
	if kind != "protocol" {
		t.Errorf("expected kind \"protocol\", got %q", kind)
	}
	pf, ok := result.(*ProtocolFindings)
	if !ok {
		t.Fatalf("expected *ProtocolFindings, got %T", result)
	}
	if pf.FindingsID != "pf-001" {
		t.Errorf("expected findings_id \"pf-001\", got %q", pf.FindingsID)
	}
	if len(pf.Findings) != 1 {
		t.Errorf("expected 1 finding, got %d", len(pf.Findings))
	}
	if pf.Findings[0].FindingKey != "PROTO-001" {
		t.Errorf("expected finding key \"PROTO-001\", got %q", pf.Findings[0].FindingKey)
	}
}

func TestParseFindingsFile_Docs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "docs.json")
	if err := os.WriteFile(path, []byte(docsFindingsJSON), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	result, kind, err := ParseFindingsFile(path)
	if err != nil {
		t.Fatalf("ParseFindingsFile() error: %v", err)
	}
	if kind != "docs" {
		t.Errorf("expected kind \"docs\", got %q", kind)
	}
	df, ok := result.(*DocsFindings)
	if !ok {
		t.Fatalf("expected *DocsFindings, got %T", result)
	}
	if df.FindingsID != "df-001" {
		t.Errorf("expected findings_id \"df-001\", got %q", df.FindingsID)
	}
	if len(df.Findings) != 1 {
		t.Errorf("expected 1 finding, got %d", len(df.Findings))
	}
}

func TestParseFindingsFile_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("{not valid json}"), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	_, _, err := ParseFindingsFile(path)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestParseFindingsFile_MissingSource(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "no-source.json")
	// Valid JSON but no "source" field
	if err := os.WriteFile(path, []byte(`{"spec_version":"1.0","findings":[]}`), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	_, _, err := ParseFindingsFile(path)
	if err == nil {
		t.Error("expected error for missing source field, got nil")
	}
}

// --- LoadLocalFindings tests ---

func TestLoadLocalFindings_MixedFiles(t *testing.T) {
	dir := t.TempDir()

	// Write a valid protocol findings file
	if err := os.WriteFile(filepath.Join(dir, "good_protocol.json"), []byte(protocolFindingsJSON), 0644); err != nil {
		t.Fatalf("write protocol file: %v", err)
	}

	// Write a valid docs findings file
	if err := os.WriteFile(filepath.Join(dir, "good_docs.json"), []byte(docsFindingsJSON), 0644); err != nil {
		t.Fatalf("write docs file: %v", err)
	}

	// Write an invalid JSON file (should be skipped)
	if err := os.WriteFile(filepath.Join(dir, "bad.json"), []byte("not json"), 0644); err != nil {
		t.Fatalf("write bad file: %v", err)
	}

	// Write a non-JSON file (should be skipped)
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hello"), 0644); err != nil {
		t.Fatalf("write txt file: %v", err)
	}

	// Create a subdirectory (should be skipped)
	if err := os.Mkdir(filepath.Join(dir, "subdir"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	findings, types, err := LoadLocalFindings(dir)
	if err != nil {
		t.Fatalf("LoadLocalFindings() error: %v", err)
	}
	if len(findings) != 2 {
		t.Errorf("expected 2 findings, got %d", len(findings))
	}
	if len(types) != 2 {
		t.Errorf("expected 2 types, got %d", len(types))
	}

	// Verify we got one of each kind
	seen := map[string]int{}
	for _, k := range types {
		seen[k]++
	}
	if seen["protocol"] != 1 || seen["docs"] != 1 {
		t.Errorf("expected one protocol and one docs, got %v", seen)
	}
}

func TestLoadLocalFindings_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	findings, types, err := LoadLocalFindings(dir)
	if err != nil {
		t.Fatalf("LoadLocalFindings() error: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected 0 findings in empty dir, got %d", len(findings))
	}
	if len(types) != 0 {
		t.Errorf("expected 0 types in empty dir, got %d", len(types))
	}
}

func TestLoadLocalFindings_NonexistentDir(t *testing.T) {
	_, _, err := LoadLocalFindings("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("expected error for nonexistent directory, got nil")
	}
}

// --- WorkflowRun JSON unmarshaling test ---

func TestWorkflowRun_UnmarshalJSON(t *testing.T) {
	input := `[{
		"id": 9876543210,
		"name": "CI Pipeline",
		"headBranch": "main",
		"status": "completed",
		"conclusion": "success",
		"createdAt": "2026-04-18T10:00:00Z"
	}]`

	var runs []WorkflowRun
	if err := json.Unmarshal([]byte(input), &runs); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}

	r := runs[0]
	if r.ID != 9876543210 {
		t.Errorf("expected ID 9876543210, got %d", r.ID)
	}
	if r.Name != "CI Pipeline" {
		t.Errorf("expected name \"CI Pipeline\", got %q", r.Name)
	}
	if r.HeadBranch != "main" {
		t.Errorf("expected headBranch \"main\", got %q", r.HeadBranch)
	}
	if r.Status != "completed" {
		t.Errorf("expected status \"completed\", got %q", r.Status)
	}
	if r.Conclusion != "success" {
		t.Errorf("expected conclusion \"success\", got %q", r.Conclusion)
	}
	if r.CreatedAt != "2026-04-18T10:00:00Z" {
		t.Errorf("expected createdAt \"2026-04-18T10:00:00Z\", got %q", r.CreatedAt)
	}
}
