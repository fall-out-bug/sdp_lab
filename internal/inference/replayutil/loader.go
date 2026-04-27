// Package replayutil provides corpus-loading and evidence-writing helpers
// shared by sdp-confidence-replay and sdp-decompose-bench.
package replayutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Fixture is one corpus entry: raw JSON, its category, and a canonical ID
// derived from its path relative to the corpus root.
type Fixture struct {
	// ID is relative path without extension, e.g. "correct/clean-pass".
	ID string
	// Category is the top-level directory: "correct", "adversarial", "edge".
	Category string
	// GoldenStatus is the expected verdict from the fixture JSON.
	// Values: "passed", "partial", "failed" (normalised from PASS/FAIL/WARN).
	GoldenStatus string
	// Raw is the fixture file content.
	Raw []byte
	// Data is the parsed fixture as a generic map.
	Data map[string]any
}

// LoadCorpus walks the directory tree rooted at path, loading every *.json
// file as a Fixture. The path structure must be: <root>/<category>/<name>.json
func LoadCorpus(path string) ([]Fixture, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("replayutil: corpus path %q: %w", path, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("replayutil: corpus path %q is not a directory", path)
	}

	var fixtures []Fixture
	err = filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".json") {
			return nil
		}
		rel, err := filepath.Rel(path, p)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("read fixture %q: %w", p, err)
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			// Skip intentionally malformed fixtures (e.g. adversarial corpus entries).
			return nil
		}

		parts := strings.SplitN(rel, string(filepath.Separator), 2)
		category := ""
		if len(parts) > 0 {
			category = parts[0]
		}
		id := strings.TrimSuffix(rel, ".json")

		fixtures = append(fixtures, Fixture{
			ID:           id,
			Category:     category,
			GoldenStatus: normaliseVerdict(data),
			Raw:          raw,
			Data:         data,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("replayutil: walk corpus: %w", err)
	}
	return fixtures, nil
}

// normaliseVerdict extracts a canonical verdict from a fixture JSON map.
// F144 fixtures use "PASS"/"FAIL"; F146 uses "passed"/"failed"/"partial".
func normaliseVerdict(data map[string]any) string {
	v, _ := data["verdict"].(string)
	switch strings.ToUpper(v) {
	case "PASS", "PASSED":
		return "passed"
	case "FAIL", "FAILED":
		return "failed"
	case "PARTIAL", "WARN":
		return "partial"
	default:
		return strings.ToLower(v)
	}
}
