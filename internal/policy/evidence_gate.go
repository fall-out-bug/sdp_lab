package policy

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fall-out-bug/sdp_lab/internal/evidence"
)

type verifyAttestationFunc func(payload []byte) error

type discrepancyThresholds struct {
	Critical int
	High     int
	Medium   int
	Low      int
}

type evidenceGateConfig struct {
	RequireSignedAttestation bool
	Thresholds               discrepancyThresholds
}

type evidenceGateResult struct {
	Allowed       bool               `json:"allowed"`
	Reasons       []string           `json:"reasons,omitempty"`
	SeverityCount map[string]int     `json:"severity_count"`
	Config        evidenceGateConfig `json:"config"`
}

func defaultEvidenceGateConfig() evidenceGateConfig {
	return evidenceGateConfig{
		RequireSignedAttestation: true,
		Thresholds: discrepancyThresholds{
			Critical: 0,
			High:     0,
			Medium:   5,
			Low:      20,
		},
	}
}

func evaluateEvidenceGate(config evidenceGateConfig, signedAttestation []byte, report evidence.DiscrepancyReport, verify verifyAttestationFunc) evidenceGateResult {
	if verify == nil {
		verify = defaultVerifyAttestation
	}

	result := evidenceGateResult{
		Allowed:       true,
		Reasons:       []string{},
		SeverityCount: countDiscrepanciesBySeverity(report),
		Config:        config,
	}

	if len(signedAttestation) == 0 {
		result.Reasons = append(result.Reasons, "attestation payload is empty")
	}

	if config.RequireSignedAttestation && !hasEmbeddedSignatures(signedAttestation) {
		result.Reasons = append(result.Reasons, "attestation is not signed")
	}

	if err := verify(signedAttestation); err != nil {
		result.Reasons = append(result.Reasons, fmt.Sprintf("attestation verification failed: %v", err))
	}

	applyDiscrepancyThresholds(&result)
	result.Allowed = len(result.Reasons) == 0
	return result
}

func (r evidenceGateResult) AuditFields() map[string]interface{} {
	fields := map[string]interface{}{
		"allowed":        r.Allowed,
		"reasons":        r.Reasons,
		"severity_count": r.SeverityCount,
		"thresholds": map[string]int{
			"critical": r.Config.Thresholds.Critical,
			"high":     r.Config.Thresholds.High,
			"medium":   r.Config.Thresholds.Medium,
			"low":      r.Config.Thresholds.Low,
		},
	}
	return fields
}

func defaultVerifyAttestation(payload []byte) error {
	_, err := evidence.NewSigner().VerifyAttestation(payload)
	return fmt.Errorf("verify attestation: %w", err)
}

func countDiscrepanciesBySeverity(report evidence.DiscrepancyReport) map[string]int {
	counts := map[string]int{"critical": 0, "high": 0, "medium": 0, "low": 0}
	for _, item := range report.Discrepancies {
		severity := strings.ToLower(strings.TrimSpace(item.Severity))
		if _, ok := counts[severity]; ok {
			counts[severity]++
		}
	}
	return counts
}

func applyDiscrepancyThresholds(result *evidenceGateResult) {
	limits := map[string]int{
		"critical": result.Config.Thresholds.Critical,
		"high":     result.Config.Thresholds.High,
		"medium":   result.Config.Thresholds.Medium,
		"low":      result.Config.Thresholds.Low,
	}

	for severity, maxAllowed := range limits {
		actual := result.SeverityCount[severity]
		if actual > maxAllowed {
			result.Reasons = append(result.Reasons, fmt.Sprintf("%s discrepancies exceeded threshold: %d > %d", severity, actual, maxAllowed))
		}
	}
}

func hasEmbeddedSignatures(payload []byte) bool {
	if len(payload) == 0 {
		return false
	}

	var raw interface{}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return false
	}
	return containsNonEmptySignatureArray(raw)
}

func containsNonEmptySignatureArray(node interface{}) bool {
	switch typed := node.(type) {
	case map[string]interface{}:
		for key, value := range typed {
			if key == "signatures" {
				if signatures, ok := value.([]interface{}); ok && len(signatures) > 0 {
					return true
				}
			}
			if containsNonEmptySignatureArray(value) {
				return true
			}
		}
	case []interface{}:
		for _, value := range typed {
			if containsNonEmptySignatureArray(value) {
				return true
			}
		}
	}

	return false
}
