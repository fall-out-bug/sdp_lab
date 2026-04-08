package discovery_test

import (
	"context"
	"os"
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

func TestValidate_ProducesVerdict(t *testing.T) {
	key := os.Getenv("OPENROUTER_API_KEY")
	if key == "" {
		t.Skip("OPENROUTER_API_KEY not set")
	}
	c := discovery.NewLLMClient(key, discovery.DefaultOpenRouterBase)
	frame := &discovery.FrameResult{
		ProblemStatement: "developers spend hours on manual product discovery before writing specs",
		Jobs:             []string{"validate ideas cheaply before investing in implementation"},
		Appetite:         "medium",
		RawIdea:          "automate product discovery using AI agents",
	}
	h := &discovery.HypothesisResult{
		Assumptions: []discovery.Assumption{
			{Statement: "solo founders avoid expensive discovery cycles because they lack time and budget", RATRank: 1, RATScore: 9, RiskLevel: "high", Uncertainty: "high"},
			{Statement: "LLM-generated validation is trusted enough to influence go/no-go decisions", RATRank: 2, RATScore: 6, RiskLevel: "high", Uncertainty: "medium"},
			{Statement: "a CLI tool fits the workflow of indie developers better than a web UI", RATRank: 3, RATScore: 4, RiskLevel: "medium", Uncertainty: "medium"},
		},
	}
	result, err := discovery.Validate(context.Background(), c, frame, h)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if len(result.Claims) == 0 {
		t.Error("no claims validated")
	}
	if len(result.Claims) > 3 {
		t.Errorf("expected at most 3 claims, got %d", len(result.Claims))
	}
	if result.FinalVerdict == "" {
		t.Error("empty final verdict")
	}
	validVerdicts := map[discovery.FinalVerdict]bool{
		discovery.VerdictGO:    true,
		discovery.VerdictPIVOT: true,
		discovery.VerdictKILL:  true,
	}
	if !validVerdicts[result.FinalVerdict] {
		t.Errorf("invalid final verdict: %q", result.FinalVerdict)
	}
	if result.VerdictReason == "" {
		t.Error("empty verdict reason")
	}
	for _, cv := range result.Claims {
		if len(cv.Evidence) == 0 {
			t.Errorf("claim rank %d has no evidence", cv.RATRank)
		}
		validClaimVerdicts := map[discovery.ClaimVerdict]bool{
			discovery.VerdictSupported:        true,
			discovery.VerdictContradicted:     true,
			discovery.VerdictInsufficientData: true,
		}
		if !validClaimVerdicts[cv.Verdict] {
			t.Errorf("claim rank %d has invalid verdict %q", cv.RATRank, cv.Verdict)
		}
	}
	t.Logf("final verdict: %s — %s", result.FinalVerdict, result.VerdictReason)
	t.Logf("cost: $%.5f", result.CostUSD)
}
