package orchestrate

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/fall-out-bug/sdp_lab/internal/sdputil"
)

//go:embed schema/orchestrate-checkpoint.schema.json
var checkpointSchemaJSON []byte

var checkpointSchema *jsonschema.Schema

func init() {
	schema, err := jsonschema.CompileString("checkpoint.schema.json", string(checkpointSchemaJSON))
	if err != nil {
		panic(fmt.Sprintf("failed to compile checkpoint schema: %v", err))
	}
	checkpointSchema = schema
}

// Checkpoint is the .sdp/checkpoints/F{NNN}.json schema for the orchestrate state machine.
// Compatible with ciloop.Checkpoint for pr_number, feature_id, branch (used by sdp-ci-loop and stop gate).
type Checkpoint struct {
	Schema      string        `json:"schema"`
	FeatureID   string        `json:"feature_id"`
	Branch      string        `json:"branch"`
	PRNumber    *int          `json:"pr_number,omitempty"`
	PRURL       string        `json:"pr_url,omitempty"`
	Phase       string        `json:"phase"`
	CreatedAt   string        `json:"created_at,omitempty"`
	UpdatedAt   string        `json:"updated_at,omitempty"`
	Workstreams []WSStatus    `json:"workstreams,omitempty"`
	Review      *ReviewStatus `json:"review,omitempty"`
	QA          *QAStatus     `json:"qa,omitempty"`
	Integrity   string        `json:"integrity,omitempty"`
}

// ErrCheckpointCorrupted is returned when checkpoint integrity validation fails.
var ErrCheckpointCorrupted = errors.New("checkpoint corrupted")

// WSStatus tracks a single workstream's execution.
type WSStatus struct {
	ID          string          `json:"id"`
	Status      string          `json:"status"` // pending, in_progress, done
	VerdictFile string          `json:"verdict_file,omitempty"`
	Commit      string          `json:"commit,omitempty"`
	Attempts    int             `json:"attempts,omitempty"`
	Dispatch    *WSDispatchInfo `json:"dispatch,omitempty"`
}

// WSDispatchInfo is a lightweight record of which harness/model was dispatched
// for a workstream. Defined locally to avoid importing the dispatch package.
type WSDispatchInfo struct {
	Harness   string  `json:"harness"`
	Provider  string  `json:"provider"`
	Model     string  `json:"model"`
	Score     float64 `json:"score"`
	Reason    string  `json:"reason,omitempty"`
	Timestamp string  `json:"timestamp,omitempty"`
	ColdStart bool    `json:"cold_start,omitempty"`
}

// ReviewStatus tracks review phase state.
type ReviewStatus struct {
	Iteration   int    `json:"iteration"`
	VerdictFile string `json:"verdict_file,omitempty"`
	Status      string `json:"status"` // pending, approved
}

type QAStatus struct {
	Iteration   int    `json:"iteration"`
	VerdictFile string `json:"verdict_file,omitempty"`
	Status      string `json:"status"`
	EvidenceRef string `json:"evidence_ref,omitempty"`
}

// Phases in order.
const (
	PhaseInit   = "init"
	PhaseBuild  = "build"
	PhaseReview = "review"
	PhasePR     = "pr"
	PhaseCI     = "ci"
	PhaseQA     = "qa"
	PhaseDone   = "done"
)

// LoadCheckpoint reads the orchestrate checkpoint for a feature.
// Returns ErrCheckpointCorrupted if integrity hash is present but does not match.
// Also validates JSON schema and returns error if schema validation fails.
func LoadCheckpoint(dir, featureID string) (*Checkpoint, error) {
	if err := sdputil.ValidateFeatureID(featureID); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, featureID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read checkpoint %s: %w", path, err)
	}

	// Validate JSON schema before parsing (skip for legacy checkpoints without schema field)
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err == nil {
		if _, hasSchema := raw["schema"]; hasSchema {
			if err := validateCheckpointSchema(data); err != nil {
				return nil, fmt.Errorf("checkpoint schema validation failed: %w. Run --repair to recover", err)
			}
		}
	}

	var cp Checkpoint
	if err := sdputil.UnmarshalJSON(data, &cp); err != nil {
		return nil, fmt.Errorf("parse checkpoint %s: %w", path, err)
	}
	if cp.Integrity != "" {
		if err := validateCheckpointIntegrity(data, cp.Integrity); err != nil {
			return nil, fmt.Errorf("%w: %s. Run --repair to recover", ErrCheckpointCorrupted, err)
		}
	}
	return &cp, nil
}

// SaveCheckpoint writes the checkpoint to disk atomically with integrity hash.
func SaveCheckpoint(dir string, cp *Checkpoint) error {
	if err := sdputil.ValidateFeatureID(cp.FeatureID); err != nil {
		return fmt.Errorf("validate feature id: %w", err)
	}
	cp.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	// Compute integrity hash over all fields except integrity itself
	cp.Integrity = ""
	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal for integrity: %w", err)
	}
	cp.Integrity = computeHash(data)
	path := filepath.Join(dir, cp.FeatureID+".json")
	return sdputil.AtomicWriteJSON(path, cp)
}

// computeHash returns the SHA-256 hex digest of data.
func computeHash(data []byte) string {
	h := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(h[:])
}

// validateCheckpointIntegrity checks the integrity hash.
func validateCheckpointIntegrity(rawData []byte, expected string) error {
	// Re-parse, clear integrity, re-marshal, then compare hash
	var cp Checkpoint
	if err := json.Unmarshal(rawData, &cp); err != nil {
		return fmt.Errorf("re-parse for integrity check: %w", err)
	}
	cp.Integrity = ""
	data, err := json.MarshalIndent(&cp, "", "  ")
	if err != nil {
		return fmt.Errorf("re-marshal for integrity check: %w", err)
	}
	actual := computeHash(data)
	if actual != expected {
		return fmt.Errorf("integrity mismatch: expected %s, got %s", expected, actual)
	}
	return nil
}

// RepairCheckpoint attempts to recover a checkpoint from git history.
// It runs `git show HEAD:.sdp/checkpoints/<featureID>.json` and writes it.
func RepairCheckpoint(projectRoot, dir, featureID string) (*Checkpoint, error) {
	if err := sdputil.ValidateFeatureID(featureID); err != nil {
		return nil, err
	}
	relPath := ".sdp/checkpoints/" + featureID + ".json"

	// Try git show for last committed version
	data, err := gitShowFile(projectRoot, "HEAD", relPath)
	if err != nil {
		return nil, fmt.Errorf("repair: cannot recover from git: %w", err)
	}
	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, fmt.Errorf("repair: parse recovered checkpoint: %w", err)
	}
	// Re-save with fresh integrity hash
	if err := SaveCheckpoint(dir, &cp); err != nil {
		return nil, fmt.Errorf("repair: save recovered checkpoint: %w", err)
	}
	return &cp, nil
}

// gitShowFile runs `git show <ref>:<path>` in the given directory.
func gitShowFile(dir, ref, relPath string) ([]byte, error) {
	cmd := exec.Command("git", "show", ref+":"+relPath)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git show %s:%s: %w", ref, relPath, err)
	}
	return out, nil
}

// validateCheckpointSchema validates the checkpoint JSON against the schema.
func validateCheckpointSchema(data []byte) error {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("unmarshal for schema validation: %w", err)
	}
	if err := checkpointSchema.Validate(v); err != nil {
		return fmt.Errorf("schema validation: %w", err)
	}
	return nil
}
