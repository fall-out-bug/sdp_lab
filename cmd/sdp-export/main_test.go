package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"sdp_dev/internal/evidence"
	"sdp_dev/internal/export"
)

func TestSanitizePII(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "redact email",
			input:    `{"email": "user@example.com", "name": "Test User"}`,
			expected: `{"email":"[REDACTED]","name":"Test User"}`,
		},
		{
			name:     "redact token",
			input:    `{"token": "abc123", "value": "data"}`,
			expected: `{"token":"[REDACTED]","value":"data"}`,
		},
		{
			name:     "redact password",
			input:    `{"password": "secret", "username": "user"}`,
			expected: `{"password":"[REDACTED]","username":"user"}`,
		},
		{
			name:     "nested PII",
			input:    `{"user": {"email": "user@example.com", "name": "Test"}}`,
			expected: `{"user":{"email":"[REDACTED]","name":"Test"}}`,
		},
		{
			name:     "no PII",
			input:    `{"name": "Test User", "value": 123}`,
			expected: `{"name":"Test User","value":123}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizePII([]byte(tt.input))
			// Parse and compare as JSON to ignore whitespace
			var resultJSON, expectedJSON interface{}
			if err := json.Unmarshal(result, &resultJSON); err != nil {
				t.Fatalf("unmarshal result: %v", err)
			}
			if err := json.Unmarshal([]byte(tt.expected), &expectedJSON); err != nil {
				t.Fatalf("unmarshal expected: %v", err)
			}

			resultBytes, _ := json.Marshal(resultJSON)
			expectedBytes, _ := json.Marshal(expectedJSON)

			if !bytes.Equal(resultBytes, expectedBytes) {
				t.Errorf("expected %s, got %s", tt.expected, string(result))
			}
		})
	}
}

func TestRecordFromAttestation(t *testing.T) {
	stmt := evidence.CodingWorkflowStatement{
		Predicate: evidence.CodingWorkflowPredicate{
			Intent: evidence.Intent{
				IssueID: "F070-01",
			},
			Provenance: evidence.Provenance{
				RunID:      "run-123",
				CapturedAt: "2026-04-26T12:00:00Z",
				Runtime:    "test-runtime",
			},
			Execution: evidence.Execution{
				ChangedFiles: []string{"file1.go", "file2.go"},
			},
			Boundary: evidence.Boundary{
				Compliance: evidence.BoundaryCompliance{
					OK: true,
				},
			},
		},
	}

	record := recordFromAttestation(stmt, ".sdp/evidence/test.json")

	if record.EventType != "attestation" {
		t.Errorf("expected event_type attestation, got %s", record.EventType)
	}
	if record.Source != "test-runtime" {
		t.Errorf("expected source test-runtime, got %s", record.Source)
	}
	if record.Severity != "info" {
		t.Errorf("expected severity info, got %s", record.Severity)
	}
	if record.RunID != "run-123" {
		t.Errorf("expected run_id run-123, got %s", record.RunID)
	}
	if record.FeatureID != "F070-01" {
		t.Errorf("expected feature_id F070-01, got %s", record.FeatureID)
	}

	changedFiles, ok := record.Details["changed_files"].(int)
	if !ok || changedFiles != 2 {
		t.Errorf("expected changed_files=2, got %v", record.Details["changed_files"])
	}
}

func TestRecordFromDiscrepancyReport(t *testing.T) {
	report := evidence.DiscrepancyReport{
		OK:    false,
		RunID: "run-456",
		Summary: "Discrepancies found: 1 critical",
		Discrepancies: []evidence.Discrepancy{
			{
				Type:        evidence.DiscrepancyBoundary,
				Severity:    "critical",
				Description: "Boundary violation",
			},
		},
	}

	record := recordFromDiscrepancyReport(report, ".sdp/evidence/report.json")

	if record.EventType != "discrepancy_report" {
		t.Errorf("expected event_type discrepancy_report, got %s", record.EventType)
	}
	if record.Source != "evidence" {
		t.Errorf("expected source evidence, got %s", record.Source)
	}
	if record.Severity != "error" {
		t.Errorf("expected severity error, got %s", record.Severity)
	}
	if record.RunID != "run-456" {
		t.Errorf("expected run_id run-456, got %s", record.RunID)
	}
}

func TestCreateZipBundle(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "sdp-export-test")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	outputPath := filepath.Join(tmpDir, "test-bundle.zip")

	// Create test data
	attestations := []evidence.CodingWorkflowStatement{
		{
			Predicate: evidence.CodingWorkflowPredicate{
				Provenance: evidence.Provenance{
					RunID: "run-1",
				},
			},
		},
	}

	reports := []evidence.DiscrepancyReport{
		{
			RunID: "run-1",
			OK:    true,
		},
	}

	bundle := export.NewExportBundle("test-tenant", "F070", []export.SIEMRecord{})

	// Create bundle
	err = createZipBundle(outputPath, attestations, reports, bundle)
	if err != nil {
		t.Fatalf("create bundle: %v", err)
	}

	// Verify bundle exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("bundle file not created")
	}

	// Read and verify zip contents
	zipReader, err := zip.OpenReader(outputPath)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer zipReader.Close()

	expectedFiles := []string{
		"manifest.json",
		"attestations/attestation-000.json",
		"discrepancies/report-000.json",
		"bundle.json",
	}

	fileMap := make(map[string]bool)
	for _, f := range zipReader.File {
		fileMap[f.Name] = true
	}

	for _, expected := range expectedFiles {
		if !fileMap[expected] {
			t.Errorf("bundle missing file: %s", expected)
		}
	}

	// Verify bundle.json contains valid data
	for _, f := range zipReader.File {
		if f.Name == "bundle.json" {
			reader, err := f.Open()
			if err != nil {
				t.Fatalf("open bundle.json: %v", err)
			}
			defer reader.Close()

			data, err := io.ReadAll(reader)
			if err != nil {
				t.Fatalf("read bundle.json: %v", err)
			}

			var bundleData export.ExportBundle
			if err := json.Unmarshal(data, &bundleData); err != nil {
				t.Fatalf("parse bundle.json: %v", err)
			}

			if bundleData.TenantID != "test-tenant" {
				t.Errorf("expected tenant_id test-tenant, got %s", bundleData.TenantID)
			}
			if bundleData.FeatureID != "F070" {
				t.Errorf("expected feature_id F070, got %s", bundleData.FeatureID)
			}
		}
	}
}

func TestRunVerify(t *testing.T) {
	// Create test bundle
	tmpDir, err := os.MkdirTemp("", "sdp-export-test")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	bundlePath := filepath.Join(tmpDir, "test-bundle.zip")
	attestations := []evidence.CodingWorkflowStatement{}
	reports := []evidence.DiscrepancyReport{}
	bundle := export.NewExportBundle("test", "F070", []export.SIEMRecord{})

	err = createZipBundle(bundlePath, attestations, reports, bundle)
	if err != nil {
		t.Fatalf("create bundle: %v", err)
	}

	// Test verification
	flagVerify = bundlePath
	err = runVerify()
	if err != nil {
		t.Errorf("verify failed: %v", err)
	}

	// Test invalid bundle
	invalidPath := filepath.Join(tmpDir, "invalid.zip")
	err = os.WriteFile(invalidPath, []byte("not a zip"), 0644)
	if err != nil {
		t.Fatalf("write invalid bundle: %v", err)
	}

	flagVerify = invalidPath
	err = runVerify()
	if err == nil {
		t.Error("expected error for invalid bundle")
	}
}

func TestIsPIIField(t *testing.T) {
	tests := []struct {
		field    string
		expected bool
	}{
		{"email", true},
		{"user_email", true},
		{"token", true},
		{"access_token", true},
		{"password", true},
		{"secret", true},
		{"api_key", true},
		{"apikey", true},
		{"name", false},
		{"id", false},
		{"value", false},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			result := isPIIField(tt.field)
			if result != tt.expected {
				t.Errorf("expected %v for %s, got %v", tt.expected, tt.field, result)
			}
		})
	}
}
