package orchestrate

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"sdp_dev/internal/harness"
	"sdp_dev/internal/sdputil"
)

var ErrContractGateBlocked = errors.New("contract gate blocked")

func ContractPaths(projectRoot, featureID string) (contractPath, snapshotPath, reportPath string, err error) {
	if err = sdputil.ValidateFeatureID(featureID); err != nil {
		return "", "", "", err
	}
	contractsDir := filepath.Join(projectRoot, ".sdp", "contracts")
	contractPath = filepath.Join(contractsDir, featureID+".json")
	snapshotPath = filepath.Join(contractsDir, featureID+".snapshot.json")
	reportPath = filepath.Join(contractsDir, featureID+".gate-report.json")
	return contractPath, snapshotPath, reportPath, nil
}

func EnforceContractGate(projectRoot, featureID string) (*harness.ComplianceReport, error) {
	contractPath, snapshotPath, reportPath, err := ContractPaths(projectRoot, featureID)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(contractPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat contract: %w", err)
	}

	contract, err := harness.LoadTaskContract(contractPath)
	if err != nil {
		return nil, err
	}
	snapshot, err := harness.LoadTaskSnapshot(snapshotPath)
	if err != nil {
		return nil, err
	}

	report := harness.EvaluateCompliance(contract, snapshot)
	if err := saveContractGateReport(reportPath, report); err != nil {
		return report, err
	}
	if report.Blocked {
		return report, fmt.Errorf("%w: %s", ErrContractGateBlocked, reportPath)
	}
	return report, nil
}

func saveContractGateReport(path string, report *harness.ComplianceReport) error {
	if report == nil {
		return fmt.Errorf("report is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create contracts dir: %w", err)
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename report: %w", err)
	}
	return nil
}
