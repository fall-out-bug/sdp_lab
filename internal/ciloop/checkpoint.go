package ciloop

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"sdp_dev/internal/sdputil"
)

// Checkpoint mirrors the .sdp/checkpoints/F{NNN}.json schema.
// ciloop only updates phase/updated_at; SaveCheckpoint merges to preserve Workstreams, Review, CreatedAt.
type Checkpoint struct {
	Schema    string `json:"schema"`
	FeatureID string `json:"feature_id"`
	Branch    string `json:"branch"`
	PRNumber  *int   `json:"pr_number,omitempty"`
	PRURL     string `json:"pr_url,omitempty"`
	Phase     string `json:"phase"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// LoadCheckpoint reads a checkpoint file for the given feature ID.
func LoadCheckpoint(dir, featureID string) (*Checkpoint, error) {
	if err := sdputil.ValidateFeatureID(featureID); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, featureID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read checkpoint %s: %w", path, err)
	}
	var cp Checkpoint
	if err := sdputil.UnmarshalJSON(data, &cp); err != nil {
		return nil, fmt.Errorf("parse checkpoint %s: %w", path, err)
	}
	return &cp, nil
}

// SaveCheckpoint writes the checkpoint to disk atomically.
// Merges cp's fields into existing file to preserve Workstreams, Review, CreatedAt (orchestrate fields).
func SaveCheckpoint(dir string, cp *Checkpoint) error {
	if err := sdputil.ValidateFeatureID(cp.FeatureID); err != nil {
		return fmt.Errorf("validate feature id: %w", err)
	}
	path := filepath.Join(dir, cp.FeatureID+".json")
	cp.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	var raw map[string]any
	if data, err := os.ReadFile(path); err == nil {
		if err := sdputil.UnmarshalJSON(data, &raw); err == nil {
			// Merge: overlay cp's fields, preserve others (workstreams, review, created_at)
			mergeCheckpointInto(raw, cp)
			recomputeIntegrity(raw)
			data, err := json.MarshalIndent(raw, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal checkpoint: %w", err)
			}
			return writeCheckpointAtomically(path, data)
		}
	}
	// No existing file or unparseable: write cp only
	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal checkpoint: %w", err)
	}
	return writeCheckpointAtomically(path, data)
}

func mergeCheckpointInto(raw map[string]any, cp *Checkpoint) {
	raw["schema"] = cp.Schema
	raw["feature_id"] = cp.FeatureID
	raw["branch"] = cp.Branch
	raw["phase"] = cp.Phase
	raw["updated_at"] = cp.UpdatedAt
	if cp.PRNumber != nil {
		raw["pr_number"] = *cp.PRNumber
	} else {
		delete(raw, "pr_number")
	}
	raw["pr_url"] = cp.PRURL
}

// recomputeIntegrity recalculates the SHA-256 integrity hash on a raw checkpoint map.
// To match orchestrate's validation (which roundtrips through a struct), we marshal the
// map to JSON, re-parse into a stable struct, clear integrity, re-marshal, and hash that.
func recomputeIntegrity(raw map[string]any) {
	raw["integrity"] = ""
	// Marshal map → JSON → re-parse into ordered struct → re-marshal for stable key order
	tmpData, err := json.Marshal(raw)
	if err != nil {
		return
	}
	var stable struct {
		Schema      string `json:"schema"`
		FeatureID   string `json:"feature_id"`
		Branch      string `json:"branch"`
		PRNumber    *int   `json:"pr_number,omitempty"`
		PRURL       string `json:"pr_url,omitempty"`
		Phase       string `json:"phase"`
		CreatedAt   string `json:"created_at,omitempty"`
		UpdatedAt   string `json:"updated_at,omitempty"`
		Workstreams json.RawMessage `json:"workstreams,omitempty"`
		Review      json.RawMessage `json:"review,omitempty"`
		QA          json.RawMessage `json:"qa,omitempty"`
		Integrity   string          `json:"integrity,omitempty"`
	}
	if err := json.Unmarshal(tmpData, &stable); err != nil {
		return
	}
	stable.Integrity = ""
	hashData, err := json.MarshalIndent(&stable, "", "  ")
	if err != nil {
		return
	}
	h := sha256.Sum256(hashData)
	raw["integrity"] = "sha256:" + hex.EncodeToString(h[:])
}

func writeCheckpointAtomically(path string, data []byte) error {
	return sdputil.AtomicWriteFile(path, data, 0o644)
}
