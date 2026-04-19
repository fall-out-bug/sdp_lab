package gate

import (
	"encoding/json"
	"fmt"
	"os"
)

// RequiredEvidenceKeys per phase type.
var RequiredEvidenceKeys = map[GateType][]string{
	GateTypePlan:   {"test_coverage", "design_checklist"},
	GateTypeReview: {"spec_review_verdict", "code_review_verdict"},
	GateTypeEval:   {"go_test", "go_vet", "protocol_check", "smoke"},
}

// ValidateEvidenceSchema checks that the evidence JSON file contains
// all required keys for the given gate type.
func ValidateEvidenceSchema(gateType GateType, evidencePath string) error {
	if _, err := os.Stat(evidencePath); os.IsNotExist(err) {
		return &EvidenceNotFoundError{Path: evidencePath}
	}
	data, err := os.ReadFile(evidencePath)
	if err != nil {
		return fmt.Errorf("failed to read evidence file: %w", err)
	}
	if !json.Valid(data) {
		return &InvalidEvidenceError{Path: evidencePath}
	}
	var evidence map[string]interface{}
	if err := json.Unmarshal(data, &evidence); err != nil {
		return &InvalidEvidenceError{Path: evidencePath}
	}
	required, ok := RequiredEvidenceKeys[gateType]
	if !ok {
		return nil // no schema requirements for this gate type
	}
	var missing []string
	for _, key := range required {
		if _, found := evidence[key]; !found {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("evidence for %s gate missing required keys: %v", gateType, missing)
	}
	return nil
}
