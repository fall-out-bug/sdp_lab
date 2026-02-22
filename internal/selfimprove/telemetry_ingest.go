package selfimprove

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// RunEvent is a phase event from .sdp/runs/*.json
type RunEvent struct {
	At      string `json:"at"`
	Phase   string `json:"phase"`
	State   string `json:"state"`
	Message string `json:"message"`
	PRURL   string `json:"pr_url,omitempty"`
}

// RunDoc is the schema for .sdp/runs/*.json
type RunDoc struct {
	RunID     string     `json:"run_id"`
	IssueID   string     `json:"issue_id"`
	Events    []RunEvent `json:"events"`
	LastPhase string     `json:"last_phase"`
	LastState string     `json:"last_state"`
}

// TelemetryRecord is a parsed observability/intake record.
type TelemetryRecord struct {
	IssueID     string
	Phase       string
	Status      string
	Component   string
	RetryCount  int
	FallbackUsed bool
	Escalated   bool
}

// IngestRuns reads all run files from .sdp/runs and returns parsed docs.
func IngestRuns(workDir string) ([]RunDoc, error) {
	runsDir := filepath.Join(workDir, ".sdp", "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []RunDoc
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(runsDir, e.Name())
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var doc RunDoc
		if err := json.Unmarshal(b, &doc); err != nil {
			continue
		}
		out = append(out, doc)
	}
	return out, nil
}

// IngestIntakeJSONL reads JSONL from path and returns telemetry records.
func IngestIntakeJSONL(path string) ([]TelemetryRecord, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []TelemetryRecord
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var raw map[string]any
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}
		rec := parseTelemetryRecord(raw)
		if rec.IssueID != "" || rec.Phase != "" {
			out = append(out, rec)
		}
	}
	return out, nil
}

func parseTelemetryRecord(raw map[string]any) TelemetryRecord {
	var rec TelemetryRecord
	if p, ok := raw["protocol"].(map[string]any); ok {
		rec.IssueID, _ = p["issue_id"].(string)
		rec.Phase, _ = p["phase"].(string)
		rec.Status, _ = p["status"].(string)
	}
	if s, ok := raw["system"].(map[string]any); ok {
		rec.Component, _ = s["component"].(string)
	}
	if r, ok := raw["resilience"].(map[string]any); ok {
		if n, ok := r["retry_count"].(float64); ok {
			rec.RetryCount = int(n)
		}
		rec.FallbackUsed, _ = r["fallback_used"].(bool)
		rec.Escalated, _ = r["escalated"].(bool)
	}
	return rec
}
