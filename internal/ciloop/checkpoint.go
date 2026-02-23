package ciloop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Checkpoint mirrors the .sdp/checkpoints/F{NNN}.json schema.
type Checkpoint struct {
	Schema    string `json:"schema"`
	FeatureID string `json:"feature_id"`
	Branch    string `json:"branch"`
	PRNumber  *int   `json:"pr_number"`
	PRURL     string `json:"pr_url"`
	Phase     string `json:"phase"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// validateFeatureID rejects featureID values that would allow path traversal.
func validateFeatureID(featureID string) error {
	if strings.ContainsAny(featureID, "/\\..") || featureID == "" {
		return fmt.Errorf("invalid feature_id %q: must not contain path separators or dots", featureID)
	}
	return nil
}

// LoadCheckpoint reads a checkpoint file for the given feature ID.
func LoadCheckpoint(dir, featureID string) (*Checkpoint, error) {
	if err := validateFeatureID(featureID); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, featureID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read checkpoint %s: %w", path, err)
	}
	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, fmt.Errorf("parse checkpoint %s: %w", path, err)
	}
	return &cp, nil
}

// SaveCheckpoint writes the checkpoint back to disk atomically.
func SaveCheckpoint(dir string, cp *Checkpoint) error {
	if err := validateFeatureID(cp.FeatureID); err != nil {
		return err
	}
	cp.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	cp.Phase = "ci"
	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal checkpoint: %w", err)
	}
	tmpPath := filepath.Join(dir, cp.FeatureID+".json.tmp")
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write checkpoint: %w", err)
	}
	path := filepath.Join(dir, cp.FeatureID+".json")
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename checkpoint: %w", err)
	}
	return nil
}
