package harness

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"sdp_dev/internal/sdputil"
)

func LoadTaskContract(path string) (*TaskContract, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read contract %s: %w", path, err)
	}
	var contract TaskContract
	if err := json.NewDecoder(io.LimitReader(bytes.NewReader(data), sdputil.MaxJSONDecodeBytes)).Decode(&contract); err != nil {
		return nil, fmt.Errorf("parse contract %s: %w", path, err)
	}
	return &contract, nil
}

func SaveTaskContract(path string, contract *TaskContract) error {
	if contract == nil {
		return fmt.Errorf("contract is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create contract directory: %w", err)
	}
	data, err := json.MarshalIndent(contract, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal contract: %w", err)
	}
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write contract: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename contract: %w", err)
	}
	return nil
}

func LoadTaskSnapshot(path string) (*TaskSnapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read snapshot %s: %w", path, err)
	}
	var snapshot TaskSnapshot
	if err := json.NewDecoder(io.LimitReader(bytes.NewReader(data), sdputil.MaxJSONDecodeBytes)).Decode(&snapshot); err != nil {
		return nil, fmt.Errorf("parse snapshot %s: %w", path, err)
	}
	if snapshot.QualityResults == nil {
		snapshot.QualityResults = map[string]bool{}
	}
	return &snapshot, nil
}

func LoadClarificationChange(path string) (*ClarificationChange, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read clarification %s: %w", path, err)
	}
	var change ClarificationChange
	if err := json.NewDecoder(io.LimitReader(bytes.NewReader(data), sdputil.MaxJSONDecodeBytes)).Decode(&change); err != nil {
		return nil, fmt.Errorf("parse clarification %s: %w", path, err)
	}
	return &change, nil
}
