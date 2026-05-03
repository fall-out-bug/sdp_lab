package f165

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const testdataDir = "../testdata/indirect_pi"

func loadFixture(t *testing.T, name string) Case {
	t.Helper()
	p := filepath.Join(testdataDir, name)
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	var c Case
	if err := yaml.Unmarshal(data, &c); err != nil {
		t.Fatalf("unmarshal fixture %s: %v", name, err)
	}
	return c
}

func TestFixtureSchema(t *testing.T) {
	matches, err := filepath.Glob(filepath.Join(testdataDir, "*.yaml"))
	if err != nil {
		t.Fatalf("glob fixtures: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("no fixtures found in %s", testdataDir)
	}

	var hasBeads, hasWorkstream, hasEvidence, hasNonObvious, hasResidual bool
	for _, p := range matches {
		name := filepath.Base(p)
		t.Run(name, func(t *testing.T) {
			c := loadFixture(t, name)
			if c.CaseID == "" {
				t.Errorf("case_id is required")
			}
			if !IsValidVector(c.Vector) {
				t.Errorf("invalid vector %q", c.Vector)
			}
			if c.UntrustedArtifact == "" {
				t.Errorf("untrusted_artifact is required")
			}
			if c.ExpectedUnsafeResult.UnsafeAction == "" || c.ExpectedUnsafeResult.UnsafeClaim == "" {
				t.Errorf("expected_unsafe_result fields are required")
			}
			if !IsValidVerdict(c.ExpectedDefendedResult.Verdict) {
				t.Errorf("invalid verdict %q", c.ExpectedDefendedResult.Verdict)
			}
			if c.ExpectedDefendedResult.Verdict == "blocked" && !IsValidBlockedReason(c.ExpectedDefendedResult.BlockedReason) {
				t.Errorf("blocked verdict requires valid blocked_reason, got %q", c.ExpectedDefendedResult.BlockedReason)
			}
			if !IsValidResidualRiskCategory(c.ResidualRiskCategory) {
				t.Errorf("invalid residual_risk_category %q", c.ResidualRiskCategory)
			}
			if strings.Contains(c.UntrustedArtifact, "http://") || strings.Contains(c.UntrustedArtifact, "https://") {
				t.Errorf("untrusted_artifact must not contain real external URLs")
			}
			if strings.Contains(c.UntrustedArtifact, "SECRET") || strings.Contains(c.UntrustedArtifact, "API_KEY") {
				t.Errorf("untrusted_artifact must not contain secret-like placeholders")
			}

			switch c.Vector {
			case "beads_issue":
				hasBeads = true
			case "workstream_markdown":
				hasWorkstream = true
			case "evidence_finding":
				hasEvidence = true
			}
			if c.CaseID == "F165-VEC-004" {
				hasNonObvious = true
			}
			if c.ResidualRiskCategory != ResidualRiskNone {
				hasResidual = true
			}
		})
	}

	if !hasBeads || !hasWorkstream || !hasEvidence {
		t.Errorf("corpus missing required vectors")
	}
	if !hasNonObvious {
		t.Errorf("corpus missing non-obvious prose case (F165-VEC-004)")
	}
	if !hasResidual {
		t.Errorf("corpus missing residual-risk case")
	}
}
