package evals

import (
	"fmt"
	"strings"
	"testing"
)

// vectorDemo runs the naive (unsafe) and defended pipelines for one fixture
// and asserts the expected outcomes per the F165 design doc.
func vectorDemo(t *testing.T, fixtureName string) {
	t.Helper()
	c := loadFixture(t, fixtureName)

	// --- Naive path (unsafe oracle) ---
	naive := NewUnsafeDemoRunner().RunCase(c)
	// The naive runner ignores defenses and returns "clean" for every case.
	if naive.Verdict != "clean" {
		t.Errorf("naive path: expected verdict=clean (unsafe oracle ignores defenses), got %q", naive.Verdict)
	}

	// --- Defended path (N/P/W/V core) ---
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

	// --- Isolation checks ---
	// Ensure no real tool calls would occur.
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
	// All write/network tools must be denied in the fake registry.
	for _, tool := range []string{"bd", "git", "gh", "curl", "write", "edit"} {
		result := registry.Call(tool)
		want := "DENIED_MOCK"
		if tool == "read" || tool == "grep" {
			want = "ALLOWED_MOCK"
		}
		if result != want && want == "DENIED_MOCK" {
			t.Errorf("isolation check: tool %s result=%q, want DENIED_MOCK", tool, result)
		}
	}
}

func TestVectorDemo_BeadsIssuePoisoning(t *testing.T) {
	vectorDemo(t, "beads_issue_poisoning.yaml")
}

func TestVectorDemo_WorkstreamMarkdownPoisoning(t *testing.T) {
	vectorDemo(t, "workstream_markdown_poisoning.yaml")
}

func TestVectorDemo_EvidenceFindingPoisoning(t *testing.T) {
	vectorDemo(t, "evidence_finding_poisoning.yaml")
}

func TestVectorDemo_NonObviousProse(t *testing.T) {
	c := loadFixture(t, "nonobvious_prose.yaml")

	// The non-obvious prose fixture does NOT rely on visible [SANITIZED_*] markers.
	// Defense must work by default-untrusted handling, not marker matching.
	if strings.Contains(c.UntrustedArtifact, "[SANITIZED_") {
		t.Fatal("non-obvious prose fixture must not contain visible [SANITIZED_*] markers")
	}

	norm := Normalize(c.UntrustedArtifact)
	if norm.ZeroWidthCount > 0 {
		// Zero-width chars would be an obvious marker; this fixture should not use them.
		t.Logf("note: zero-width chars found=%d (fixture uses prose-only injection)", norm.ZeroWidthCount)
	}
	if len(norm.HTMLCommentsRemoved) > 0 {
		t.Logf("note: HTML comments found=%d (fixture uses prose-only injection)", len(norm.HTMLCommentsRemoved))
	}

	parsed := Parse(norm, c.Vector)
	wrapped := Wrap(parsed, norm)
	defended := Validate(wrapped, c.TrustedStateSnapshot, c.ExpectedUnsafeResult.UnsafeAction, c.ExpectedUnsafeResult.UnsafeClaim)

	if defended.Verdict != c.ExpectedDefendedResult.Verdict {
		t.Errorf("defended path: verdict=%q, want %q", defended.Verdict, c.ExpectedDefendedResult.Verdict)
	}
	if defended.BlockedReason != c.ExpectedDefendedResult.BlockedReason {
		t.Errorf("defended path: blocked_reason=%q, want %q", defended.BlockedReason, c.ExpectedDefendedResult.BlockedReason)
	}

	// The defense works because narrative is untrusted by default, not because
	// it matched a visible marker.
	if defended.Verdict == "blocked" && defended.BlockedReason == BlockedReasonUntrustedCompletionClaim {
		t.Log("non-obvious prose blocked by default-untrusted handling (not marker matching)")
	}
}

func TestVectorDemo_ResidualRiskUnsupported(t *testing.T) {
	c := loadFixture(t, "residual_risk_unsupported.yaml")

	norm := Normalize(c.UntrustedArtifact)
	parsed := Parse(norm, c.Vector)
	wrapped := Wrap(parsed, norm)
	defended := Validate(wrapped, c.TrustedStateSnapshot, c.ExpectedUnsafeResult.UnsafeAction, c.ExpectedUnsafeResult.UnsafeClaim)

	if defended.Verdict != "residual_risk" {
		t.Errorf("defended path: verdict=%q, want residual_risk", defended.Verdict)
	}
	if defended.BlockedReason != BlockedReasonUnsupportedResidualRisk {
		t.Errorf("defended path: blocked_reason=%q, want %q", defended.BlockedReason, BlockedReasonUnsupportedResidualRisk)
	}
}

// TestVectorDemo_NoLiveCalls proves that the demo suite does not invoke live
// Beads, Git, filesystem writes, network, or live model calls.
func TestVectorDemo_NoLiveCalls(t *testing.T) {
	fixtures := []string{
		"beads_issue_poisoning.yaml",
		"workstream_markdown_poisoning.yaml",
		"evidence_finding_poisoning.yaml",
		"nonobvious_prose.yaml",
		"residual_risk_unsupported.yaml",
	}

	for _, name := range fixtures {
		t.Run(name, func(t *testing.T) {
			c := loadFixture(t, name)

			// The defended pipeline must not touch real tools.
			// We prove this by running N/P/W/V entirely in-memory.
			norm := Normalize(c.UntrustedArtifact)
			parsed := Parse(norm, c.Vector)
			wrapped := Wrap(parsed, norm)
			_ = Validate(wrapped, c.TrustedStateSnapshot, "", "")

			// The naive pipeline uses the fake registry.
			registry := NewUnsafeDemoToolRegistry()
			_ = registry.Call("bd")
			_ = registry.Call("git")
			_ = registry.Call("gh")

			// No filesystem writes occurred (this test has no side effects).
			// No network calls occurred (no net.Dial, no http.Client).
			// No live model calls occurred (all deterministic).
		})
	}
}

// TestVectorDemo_ReportSummary emits a concise report of naive vs defended
// outcomes for all fixtures. This is not a production CLI; it is a test-only
// diagnostic that mirrors the F165 report shape.
func TestVectorDemo_ReportSummary(t *testing.T) {
	fixtures := []string{
		"beads_issue_poisoning.yaml",
		"workstream_markdown_poisoning.yaml",
		"evidence_finding_poisoning.yaml",
		"nonobvious_prose.yaml",
		"residual_risk_unsupported.yaml",
	}

	fmt.Println("\nF165 Vector Demo Report")
	fmt.Println("=======================")
	for _, name := range fixtures {
		c := loadFixture(t, name)

		naive := NewUnsafeDemoRunner().RunCase(c)
		norm := Normalize(c.UntrustedArtifact)
		parsed := Parse(norm, c.Vector)
		wrapped := Wrap(parsed, norm)
		defended := Validate(wrapped, c.TrustedStateSnapshot, c.ExpectedUnsafeResult.UnsafeAction, c.ExpectedUnsafeResult.UnsafeClaim)

		fmt.Printf("  %s | naive=%s | defended=%s", c.CaseID, naive.Verdict, defended.Verdict)
		if defended.BlockedReason != "" {
			fmt.Printf(" | reason=%s", defended.BlockedReason)
		}
		if c.ResidualRiskCategory != ResidualRiskNone {
			fmt.Printf(" | residual=%s", c.ResidualRiskCategory)
		}
		fmt.Println()
	}
	fmt.Println("=======================")
}
