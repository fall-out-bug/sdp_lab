package rules

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EvidenceEntry represents a single evidence.json entry.
type EvidenceEntry struct {
	RunID     string `json:"run_id"`
	Phase     string `json:"phase"`
	Verdict   string `json:"verdict"`
	Summary   string `json:"summary"`
	Timestamp string `json:"timestamp"`
	FilePath  string `json:"file_path,omitempty"`
}

// ReadEvidenceDir reads all .json files in dir and returns their parsed
// entries. Files that are missing or malformed are skipped silently by design:
// evidence files are append-only logs produced by CI runs; a corrupt or
// partial write should not block rule generation for the rest of the entries.
func ReadEvidenceDir(dir string) ([]EvidenceEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %q: %w", dir, err)
	}

	var result []EvidenceEntry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		parsed, err := readEvidenceFile(path)
		if err != nil {
			// Intentional: skip malformed files rather than failing the entire batch.
			continue
		}
		result = append(result, parsed...)
	}

	return result, nil
}

// readEvidenceFile reads and parses a single JSON evidence file.
// The file may contain either a single EvidenceEntry or an array of entries.
func readEvidenceFile(path string) ([]EvidenceEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Try array first.
	var arr []EvidenceEntry
	if err := json.Unmarshal(data, &arr); err == nil {
		for i := range arr {
			if arr[i].FilePath == "" {
				arr[i].FilePath = path
			}
		}
		return arr, nil
	}

	// Try single entry.
	var single EvidenceEntry
	if err := json.Unmarshal(data, &single); err != nil {
		return nil, err
	}
	if single.FilePath == "" {
		single.FilePath = path
	}
	return []EvidenceEntry{single}, nil
}
