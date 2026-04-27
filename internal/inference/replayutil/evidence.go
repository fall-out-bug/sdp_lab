package replayutil

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// RunRecord is the per-fixture A/B benchmark record written to CSV.
type RunRecord struct {
	ID               string
	Category         string
	MonolithicStatus string
	DecomposedStatus string
	LatencyMonoMs    int64
	LatencyDecompMs  int64
	TokensMono       int
	TokensDecomp     int
	CostMonoUSD      float64
	CostDecompUSD    float64
}

// AggregateEvidence holds the full run output for JSON evidence.
type AggregateEvidence struct {
	RunID     string      `json:"run_id"`
	Timestamp string      `json:"timestamp"`
	Records   []RunRecord `json:"records"`
}

// WriteEvidence writes per-run JSON and aggregate CSV to dir/<runID>/.
// Creates the directory if it does not exist.
func WriteEvidence(dir, runID string, evidence AggregateEvidence) error {
	outDir := filepath.Join(dir, runID)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("evidence mkdir: %w", err)
	}

	// Per-run JSON.
	jsonPath := filepath.Join(outDir, "evidence.json")
	data, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return fmt.Errorf("evidence marshal: %w", err)
	}
	if err := os.WriteFile(jsonPath, data, 0o644); err != nil {
		return fmt.Errorf("evidence write json: %w", err)
	}

	// Aggregate CSV.
	csvPath := filepath.Join(outDir, "results.csv")
	f, err := os.Create(csvPath)
	if err != nil {
		return fmt.Errorf("evidence create csv: %w", err)
	}

	w := csv.NewWriter(f)
	if err := w.Write([]string{
		"id", "category",
		"monolithic_status", "decomposed_status",
		"latency_mono", "latency_decomp",
		"tokens_mono", "tokens_decomp",
		"cost_mono", "cost_decomp",
	}); err != nil {
		f.Close()
		return fmt.Errorf("evidence write csv header: %w", err)
	}

	for _, r := range evidence.Records {
		if err := w.Write([]string{
			r.ID, r.Category,
			r.MonolithicStatus, r.DecomposedStatus,
			strconv.FormatInt(r.LatencyMonoMs, 10),
			strconv.FormatInt(r.LatencyDecompMs, 10),
			strconv.Itoa(r.TokensMono),
			strconv.Itoa(r.TokensDecomp),
			strconv.FormatFloat(r.CostMonoUSD, 'f', 6, 64),
			strconv.FormatFloat(r.CostDecompUSD, 'f', 6, 64),
		}); err != nil {
			f.Close()
			return fmt.Errorf("evidence write csv row: %w", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		f.Close()
		return fmt.Errorf("evidence flush csv: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("evidence close csv: %w", err)
	}
	return nil
}

// NewRunID generates a time-based run ID suitable for directory names.
func NewRunID() string {
	return time.Now().UTC().Format("20060102-150405")
}
