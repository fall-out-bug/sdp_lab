package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func writeJSON(path string, payload any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}

func loadEvidenceTemplate(root string) (map[string]any, error) {
	path := filepath.Join(root, "specs", "strict-evidence-template.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func appendNote(issueID string, note string) error {
	_, err := bdRunner("update", issueID, "--append-notes", note)
	return err
}
