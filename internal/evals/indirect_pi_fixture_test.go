package evals

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const indirectPITestdataDir = "testdata/indirect_pi"

// TestIndirectPIFixtureSchema validates every YAML fixture in testdata/indirect_pi.
func TestIndirectPIFixtureSchema(t *testing.T) {
	matches, err := filepath.Glob(filepath.Join(indirectPITestdataDir, "*.yaml"))
	if err != nil {
		t.Fatalf("glob fixtures: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("no fixtures found in %s", indirectPITestdataDir)
	}

	var (
		hasBeadsIssue        bool
		hasWorkstream        bool
		hasEvidenceFinding   bool
		hasNonObvious        bool
		hasResidualRisk      bool
	)

	for _, p := range matches {
		name := filepath.Base(p)
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(p)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			var c IndirectPICase
			if err := yaml.Unmarshal(data, &c); err != nil {
				t.Fatalf("unmarshal fixture: %v", err)
			}

			if c.CaseID == "" {
				t.Errorf("case_id is required")
			}
			if !IsValidVector(c.Vector) {
				t.Errorf("invalid vector %q, must be one of %v", c.Vector, ValidVectors)
			}
			if c.TrustedOperatorRequest == "" {
				t.Errorf("trusted_operator_request is required")
			}
			if c.UntrustedArtifact == "" {
				t.Errorf("untrusted_artifact is required")
			}
			if c.ExpectedUnsafeResult.UnsafeAction == "" {
				t.Errorf("expected_unsafe_result.unsafe_action is required")
			}
			if c.ExpectedUnsafeResult.UnsafeClaim == "" {
				t.Errorf("expected_unsafe_result.unsafe_claim is required")
			}
			if !IsValidVerdict(c.ExpectedDefendedResult.Verdict) {
				t.Errorf("invalid verdict %q, must be one of %v", c.ExpectedDefendedResult.Verdict, ValidVerdicts)
			}
			if c.ExpectedDefendedResult.Verdict == "blocked" && !IsValidBlockedReason(c.ExpectedDefendedResult.BlockedReason) {
				t.Errorf("blocked verdict requires valid blocked_reason, got %q", c.ExpectedDefendedResult.BlockedReason)
			}
			if c.EvidenceExpectation == "" {
				t.Errorf("evidence_expectation is required")
			}
			if !IsValidResidualRiskCategory(c.ResidualRiskCategory) {
				t.Errorf("invalid residual_risk_category %q", c.ResidualRiskCategory)
			}

			// No real external URLs, secrets, or live account actions.
			if strings.Contains(c.UntrustedArtifact, "http://") || strings.Contains(c.UntrustedArtifact, "https://") {
				t.Errorf("untrusted_artifact must not contain real external URLs")
			}
			if strings.Contains(c.UntrustedArtifact, "SECRET") || strings.Contains(c.UntrustedArtifact, "API_KEY") {
				t.Errorf("untrusted_artifact must not contain secret-like placeholders")
			}

			// Track coverage requirements.
			switch c.Vector {
			case "beads_issue":
				hasBeadsIssue = true
			case "workstream_markdown":
				hasWorkstream = true
			case "evidence_finding":
				hasEvidenceFinding = true
			}
			if c.CaseID == "F165-VEC-004" {
				hasNonObvious = true
			}
			if c.ResidualRiskCategory != ResidualRiskNone {
				hasResidualRisk = true
			}
		})
	}

	if !hasBeadsIssue {
		t.Errorf("corpus missing beads_issue vector")
	}
	if !hasWorkstream {
		t.Errorf("corpus missing workstream_markdown vector")
	}
	if !hasEvidenceFinding {
		t.Errorf("corpus missing evidence_finding vector")
	}
	if !hasNonObvious {
		t.Errorf("corpus missing non-obvious prose case (F165-VEC-004)")
	}
	if !hasResidualRisk {
		t.Errorf("corpus missing residual-risk case")
	}
}

// TestIndirectPIBlockedReasonClosedSet ensures no new blocked_reason values leak in.
func TestIndirectPIBlockedReasonClosedSet(t *testing.T) {
	valid := map[string]bool{
		BlockedReasonUntrustedCompletionClaim:        true,
		BlockedReasonScopePolicyConflict:             true,
		BlockedReasonEvidenceSourceMismatch:          true,
		BlockedReasonWriteWithoutTrustedAuthorization: true,
		BlockedReasonParseError:                       true,
		BlockedReasonPolicyConflict:                   true,
		BlockedReasonUnsupportedResidualRisk:          true,
	}
	invalid := []string{
		"other",
		"",
		"malicious_reason",
		"prompt_injection",
	}
	for _, r := range invalid {
		if IsValidBlockedReason(r) {
			t.Errorf("expected %q to be invalid blocked_reason", r)
		}
	}
	for r := range valid {
		if !IsValidBlockedReason(r) {
			t.Errorf("expected %q to be valid blocked_reason", r)
		}
	}
}

// TestIndirectPIResidualRiskClosedSet ensures no new residual-risk categories leak in.
func TestIndirectPIResidualRiskClosedSet(t *testing.T) {
	invalid := []string{"other", "", "some_risk"}
	for _, c := range invalid {
		if IsValidResidualRiskCategory(c) {
			t.Errorf("expected %q to be invalid residual_risk_category", c)
		}
	}
}
