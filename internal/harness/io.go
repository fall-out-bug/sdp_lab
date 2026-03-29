package harness

import (
	"fmt"
	"os"

	"sdp_dev/internal/sdputil"
)

func LoadTaskContract(path string) (*TaskContract, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read contract %s: %w", path, err)
	}
	var contract TaskContract
	if err := sdputil.UnmarshalJSON(data, &contract); err != nil {
		return nil, fmt.Errorf("parse contract %s: %w", path, err)
	}
	return &contract, nil
}

func SaveTaskContract(path string, contract *TaskContract) error {
	if contract == nil {
		return fmt.Errorf("contract is required")
	}
	return sdputil.AtomicWriteJSON(path, contract)
}

func LoadTaskSnapshot(path string) (*TaskSnapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read snapshot %s: %w", path, err)
	}
	var snapshot TaskSnapshot
	if err := sdputil.UnmarshalJSON(data, &snapshot); err != nil {
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
	if err := sdputil.UnmarshalJSON(data, &change); err != nil {
		return nil, fmt.Errorf("parse clarification %s: %w", path, err)
	}
	return &change, nil
}
