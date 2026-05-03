package f165

import (
	"strings"
	"testing"
)

func vectorDemo(t *testing.T, fixtureName string) {
	t.Helper()
	c := loadFixture(t, fixtureName)
	runner := NewUnsafeDemoRunner()
	naive := runner.RunCase(c)
	if naive.Verdict != "clean" {
		t.Errorf("naive path: expected verdict=clean, got %q", naive.Verdict)
	}
	if runner.UnsafeDemoAction(c) != c.ExpectedUnsafeResult.UnsafeAction {
		t.Errorf("naive action mismatch")
	}
	if runner.UnsafeDemoClaim(c) != c.ExpectedUnsafeResult.UnsafeClaim {
		t.Errorf("naive claim mismatch")
	}
	defended := DefendCase(c)
	if defended.Verdict != c.ExpectedDefendedResult.Verdict {
		t.Errorf("defended path: verdict=%q, want %q", defended.Verdict, c.ExpectedDefendedResult.Verdict)
	}
	if defended.BlockedReason != c.ExpectedDefendedResult.BlockedReason {
		t.Errorf("defended path: blocked_reason=%q, want %q", defended.BlockedReason, c.ExpectedDefendedResult.BlockedReason)
	}
}

func TestVectorDemo_BeadsIssue(t *testing.T)   { vectorDemo(t, "beads_issue_poisoning.yaml") }
func TestVectorDemo_Workstream(t *testing.T)   { vectorDemo(t, "workstream_markdown_poisoning.yaml") }
func TestVectorDemo_Evidence(t *testing.T)     { vectorDemo(t, "evidence_finding_poisoning.yaml") }
func TestVectorDemo_NonObvious(t *testing.T)   { vectorDemo(t, "nonobvious_prose.yaml") }
func TestVectorDemo_ResidualRisk(t *testing.T) { vectorDemo(t, "residual_risk_unsupported.yaml") }

func TestVectorDemo_NonObvious_NoVisibleMarkers(t *testing.T) {
	c := loadFixture(t, "nonobvious_prose.yaml")
	if contains(c.UntrustedArtifact, "[SANITIZED_") {
		t.Fatal("non-obvious prose fixture must not contain visible [SANITIZED_*] markers")
	}
	defended := DefendCase(c)
	if defended.Verdict != c.ExpectedDefendedResult.Verdict {
		t.Errorf("defended path: verdict=%q, want %q", defended.Verdict, c.ExpectedDefendedResult.Verdict)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
