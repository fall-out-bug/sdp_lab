package discovery_test

import (
	"strings"
	"testing"

	"sdp_dev/internal/discovery"
)

func TestNeedsExperiment_FalseWhenAllSupported(t *testing.T) {
	claims := []discovery.ClaimValidation{
		{Verdict: discovery.VerdictSupported},
		{Verdict: discovery.VerdictContradicted},
	}
	if discovery.NeedsExperimentFromClaims(claims) {
		t.Error("expected false: no insufficient_data verdict")
	}
}

func TestNeedsExperiment_TrueWhenAnyInsufficientData(t *testing.T) {
	claims := []discovery.ClaimValidation{
		{Verdict: discovery.VerdictSupported},
		{Verdict: discovery.VerdictInsufficientData},
	}
	if !discovery.NeedsExperimentFromClaims(claims) {
		t.Error("expected true: has insufficient_data verdict")
	}
}

func TestRenderClaimsForSynthesis_ContainsRankAndVerdict(t *testing.T) {
	claims := []discovery.ClaimValidation{
		{
			Claim:      "founders need validated ideas before coding",
			RATRank:    1,
			Verdict:    discovery.VerdictSupported,
			Confidence: 0.8,
			Notes:      "ample survey data",
			Evidence: []discovery.Evidence{
				{Direction: "for", Statement: "62% of indie hackers skip validation", IsEstimate: true},
			},
		},
	}
	out := discovery.RenderClaimsForSynthesis(claims)
	if out == "" {
		t.Fatal("empty render output")
	}
	for _, want := range []string{"Rank 1", "SUPPORTED", "founders need validated ideas"} {
		if !strings.Contains(out, want) {
			t.Errorf("render missing %q\nfull output:\n%s", want, out)
		}
	}
}
