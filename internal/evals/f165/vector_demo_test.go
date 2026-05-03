package f165

import (
	"strings"
	"testing"
)

func vectorDemo(t *testing.T, fixtureName string) {
	t.Helper()
	c := loadFixture(t, fixtureName)
	naive := NewUnsafeDemoRunner().RunCase(c)
	if naive.Verdict != "clean" {
		t.Errorf("naive path: expected verdict=clean, got %q", naive.Verdict)
	}
	norm := Normalize(c.UntrustedArtifact)
	parsed := Parse(norm, c.Vector)
	wrapped := Wrap(parsed, norm)
	defended := Validate(wrapped, c.TrustedStateSnapshot, c.ExpectedUnsafeResult.UnsafeAction, c.ExpectedUnsafeResult.UnsafeClaim)
	if defended.Verdict != c.ExpectedDefendedResult.Verdict {
		t.Errorf("defended path: verdict=%q, want %q", defended.Verdict, c.ExpectedDefendedResult.Verdict)
	}
	if defended.BlockedReason != c.ExpectedDefendedResult.BlockedReason {
		t.Errorf("defended path: blocked_reason=%q, want %q", defended.BlockedReason, c.ExpectedDefendedResult.BlockedReason)
	}
	registry := NewUnsafeDemoToolRegistry()
	_ = registry.Call("bd")
	_ = registry.Call("git")
	_ = registry.Call("gh")
	_ = registry.Call("curl")
	for _, tool := range []string{"bd", "git", "gh", "curl"} {
		if !registry.HasCall(tool) {
			t.Errorf("isolation check: expected registry to record %s call", tool)
		}
	}
}

func TestVectorDemo_BeadsIssue(t *testing.T)     { vectorDemo(t, "beads_issue_poisoning.yaml") }
func TestVectorDemo_Workstream(t *testing.T)      { vectorDemo(t, "workstream_markdown_poisoning.yaml") }
func TestVectorDemo_Evidence(t *testing.T)        { vectorDemo(t, "evidence_finding_poisoning.yaml") }
func TestVectorDemo_NonObvious(t *testing.T)      { vectorDemo(t, "nonobvious_prose.yaml") }
func TestVectorDemo_ResidualRisk(t *testing.T)    { vectorDemo(t, "residual_risk_unsupported.yaml") }

func TestVectorDemo_NonObvious_NoVisibleMarkers(t *testing.T) {
	c := loadFixture(t, "nonobvious_prose.yaml")
	if contains(c.UntrustedArtifact, "[SANITIZED_") {
		t.Fatal("non-obvious prose fixture must not contain visible [SANITIZED_*] markers")
	}
	norm := Normalize(c.UntrustedArtifact)
	parsed := Parse(norm, c.Vector)
	wrapped := Wrap(parsed, norm)
	defended := Validate(wrapped, c.TrustedStateSnapshot, c.ExpectedUnsafeResult.UnsafeAction, c.ExpectedUnsafeResult.UnsafeClaim)
	if defended.Verdict != c.ExpectedDefendedResult.Verdict {
		t.Errorf("defended path: verdict=%q, want %q", defended.Verdict, c.ExpectedDefendedResult.Verdict)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
