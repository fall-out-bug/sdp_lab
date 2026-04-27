package main

import (
	"archive/zip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/evidence"
	"github.com/fall-out-bug/sdp_lab/internal/export"
)

var (
	flagOutput    string
	flagRunID     string
	flagFeatureID string
	flagVerify    string
	flagNoPII     bool
)

func init() {
	flag.StringVar(&flagOutput, "output", "", "Output bundle path (default: sdp-audit-bundle-<timestamp>.zip)")
	flag.StringVar(&flagRunID, "run-id", "", "Filter by run ID")
	flag.StringVar(&flagFeatureID, "feature-id", "", "Filter by feature ID")
	flag.StringVar(&flagVerify, "verify", "", "Verify bundle integrity")
	flag.BoolVar(&flagNoPII, "no-pii", true, "Exclude PII and secrets (default: true)")
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if flagVerify != "" {
		if err := runVerify(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := runExport(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: sdp-export [flags]\n")
	fmt.Fprintf(os.Stderr, "\nFlags:\n")
	fmt.Fprintf(os.Stderr, "  -output string      Output bundle path (default: sdp-audit-bundle-<timestamp>.zip)\n")
	fmt.Fprintf(os.Stderr, "  -run-id string      Filter by run ID\n")
	fmt.Fprintf(os.Stderr, "  -feature-id string  Filter by feature ID\n")
	fmt.Fprintf(os.Stderr, "  -verify string      Verify bundle integrity\n")
	fmt.Fprintf(os.Stderr, "  -no-pii             Exclude PII and secrets (default: true)\n")
	fmt.Fprintf(os.Stderr, "  -h, -help           Show this help message\n")
	fmt.Fprintf(os.Stderr, "\nExamples:\n")
	fmt.Fprintf(os.Stderr, "  sdp-export                          # Export all evidence\n")
	fmt.Fprintf(os.Stderr, "  sdp-export -run-id run-abc123       # Export specific run\n")
	fmt.Fprintf(os.Stderr, "  sdp-export -verify bundle.zip       # Verify bundle\n")
	fmt.Fprintf(os.Stderr, "  sdp-export -feature-id F070         # Export feature evidence\n")
}

func runExport() error {
	// Determine output path
	outputPath := flagOutput
	if outputPath == "" {
		timestamp := time.Now().Format("20060102-150405")
		outputPath = fmt.Sprintf("sdp-audit-bundle-%s.zip", timestamp)
	}

	// Collect evidence
	evidenceDir := ".sdp/evidence"
	if _, err := os.Stat(evidenceDir); os.IsNotExist(err) {
		return fmt.Errorf("evidence directory not found: %s", evidenceDir)
	}

	files, err := os.ReadDir(evidenceDir)
	if err != nil {
		return fmt.Errorf("read evidence directory: %w", err)
	}

	var records []export.SIEMRecord
	var attestations []evidence.CodingWorkflowStatement
	var reports []evidence.DiscrepancyReport

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		path := filepath.Join(evidenceDir, file.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: read file %s: %v\n", path, err)
			continue
		}

		// Try to parse as attestation
		var stmt evidence.CodingWorkflowStatement
		if err := json.Unmarshal(data, &stmt); err == nil {
			// Filter by run ID if specified
			if flagRunID != "" && stmt.Predicate.Provenance.RunID != flagRunID {
				continue
			}
			// Filter by feature ID if specified
			if flagFeatureID != "" && !strings.Contains(stmt.Predicate.Intent.IssueID, flagFeatureID) {
				continue
			}

			attestations = append(attestations, stmt)
			records = append(records, recordFromAttestation(stmt, path))
			continue
		}

		// Try to parse as discrepancy report
		var report evidence.DiscrepancyReport
		if err := json.Unmarshal(data, &report); err == nil {
			// Filter by run ID if specified
			if flagRunID != "" && report.RunID != flagRunID {
				continue
			}

			reports = append(reports, report)
			records = append(records, recordFromDiscrepancyReport(report, path))
			continue
		}
	}

	// Create export bundle
	bundle := export.NewExportBundle("default", flagFeatureID, records)

	// Create ZIP archive
	if err := createZipBundle(outputPath, attestations, reports, bundle); err != nil {
		return fmt.Errorf("create bundle: %w", err)
	}

	fmt.Printf("Exported audit bundle: %s\n", outputPath)
	fmt.Printf("  Bundle ID: %s\n", bundle.BundleID)
	fmt.Printf("  Records: %d\n", bundle.RecordCount)
	fmt.Printf("  Checksum: %s\n", bundle.BundleChecksum)

	return nil
}

func createZipBundle(outputPath string, attestations []evidence.CodingWorkflowStatement, reports []evidence.DiscrepancyReport, bundle *export.ExportBundle) error {
	zipFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create zip file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Add manifest
	manifest := map[string]interface{}{
		"bundle_id":    bundle.BundleID,
		"created_at":   bundle.CreatedAt,
		"record_count": bundle.RecordCount,
		"checksum":     bundle.BundleChecksum,
		"filters": map[string]string{
			"run_id":     flagRunID,
			"feature_id": flagFeatureID,
		},
		"pii_excluded": flagNoPII,
	}

	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	if err := addFileToZip(zipWriter, "manifest.json", manifestData); err != nil {
		return err
	}

	// Add attestations
	for i, stmt := range attestations {
		data, err := json.MarshalIndent(stmt, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal attestation: %w", err)
		}

		// Sanitize PII if requested
		if flagNoPII {
			data = sanitizePII(data)
		}

		filename := fmt.Sprintf("attestations/attestation-%03d.json", i)
		if err := addFileToZip(zipWriter, filename, data); err != nil {
			return err
		}
	}

	// Add discrepancy reports
	for i, report := range reports {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal discrepancy report: %w", err)
		}

		filename := fmt.Sprintf("discrepancies/report-%03d.json", i)
		if err := addFileToZip(zipWriter, filename, data); err != nil {
			return err
		}
	}

	// Add bundle metadata
	metadataData, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal bundle: %w", err)
	}

	if err := addFileToZip(zipWriter, "bundle.json", metadataData); err != nil {
		return err
	}

	return nil
}

func addFileToZip(zipWriter *zip.Writer, filename string, data []byte) error {
	header := &zip.FileHeader{
		Name:   filename,
		Method: zip.Deflate,
	}
	header.SetModTime(time.Now())
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("create zip entry: %w", err)
	}
	_, err = writer.Write(data)
	return err
}

func sanitizePII(data []byte) []byte {
	// Redact common PII patterns
	// This is a basic implementation - a production version would be more sophisticated
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return data
	}

	// Recursively sanitize
	sanitizeMap(result)

	sanitized, _ := json.Marshal(result)
	return sanitized
}

func sanitizeMap(m map[string]interface{}) {
	for key, value := range m {
		switch v := value.(type) {
		case map[string]interface{}:
			sanitizeMap(v)
		case []interface{}:
			for _, item := range v {
				if itemMap, ok := item.(map[string]interface{}); ok {
					sanitizeMap(itemMap)
				}
			}
		case string:
			if isPIIField(key) {
				m[key] = "[REDACTED]"
			}
		}
	}
}

func isPIIField(key string) bool {
	piiFields := []string{"email", "token", "password", "secret", "api_key", "apikey"}
	for _, field := range piiFields {
		if strings.Contains(strings.ToLower(key), field) {
			return true
		}
	}
	return false
}

func recordFromAttestation(stmt evidence.CodingWorkflowStatement, path string) export.SIEMRecord {
	return export.SIEMRecord{
		Timestamp:    parseTime(stmt.Predicate.Provenance.CapturedAt),
		EventType:    "attestation",
		Source:       stmt.Predicate.Provenance.Runtime,
		Severity:     "info",
		RunID:        stmt.Predicate.Provenance.RunID,
		FeatureID:    stmt.Predicate.Intent.IssueID,
		WorkstreamID: stmt.Predicate.Trace.PRURL,
		Details: map[string]interface{}{
			"path":        path,
			"branch":      stmt.Predicate.Execution.Branch,
			"changed_files": len(stmt.Predicate.Execution.ChangedFiles),
			"boundary_ok": stmt.Predicate.Boundary.Compliance.OK,
		},
	}
}

func recordFromDiscrepancyReport(report evidence.DiscrepancyReport, path string) export.SIEMRecord {
	severity := "info"
	if !report.OK {
		severity = "error"
	}

	return export.SIEMRecord{
		Timestamp: time.Now(),
		EventType: "discrepancy_report",
		Source:    "evidence",
		Severity:  severity,
		RunID:     report.RunID,
		Details: map[string]interface{}{
			"path":          path,
			"summary":       report.Summary,
			"ok":            report.OK,
			"discrepancies": len(report.Discrepancies),
		},
	}
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Now()
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Now()
	}
	return t
}

func runVerify() error {
	// Open the zip file
	zipReader, err := zip.OpenReader(flagVerify)
	if err != nil {
		return fmt.Errorf("open zip file: %w", err)
	}
	defer zipReader.Close()

	// Find and verify bundle.json
	var bundleData []byte
	for _, file := range zipReader.File {
		if file.Name == "bundle.json" {
			reader, err := file.Open()
			if err != nil {
				return fmt.Errorf("open bundle.json: %w", err)
			}
			bundleData, err = io.ReadAll(reader)
			reader.Close()
			if err != nil {
				return fmt.Errorf("read bundle.json: %w", err)
			}
			break
		}
	}

	if bundleData == nil {
		return fmt.Errorf("bundle.json not found in archive")
	}

	var bundle export.ExportBundle
	if err := json.Unmarshal(bundleData, &bundle); err != nil {
		return fmt.Errorf("parse bundle: %w", err)
	}

	// Verify integrity
	if !bundle.Verify() {
		return fmt.Errorf("bundle verification failed: checksum mismatch")
	}

	fmt.Printf("Bundle verification: OK\n")
	fmt.Printf("  Bundle ID: %s\n", bundle.BundleID)
	fmt.Printf("  Records: %d\n", bundle.RecordCount)
	fmt.Printf("  Checksum: %s\n", bundle.BundleChecksum)
	fmt.Printf("  Created: %s\n", bundle.CreatedAt.Format(time.RFC3339))

	return nil
}
