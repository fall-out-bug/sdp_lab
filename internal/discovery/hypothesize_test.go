package discovery_test

import (
	"context"
	"os"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/discovery"
)

func TestHypothesizeProducesTestCard(t *testing.T) {
	key := os.Getenv("OPENROUTER_API_KEY")
	if key == "" {
		t.Skip("OPENROUTER_API_KEY not set")
	}
	c := discovery.NewLLMClient(key, discovery.DefaultOpenRouterBase)
	frame := &discovery.FrameResult{
		ProblemStatement: "developers spend hours on manual product discovery before writing specs",
		Jobs:             []string{"developer wants to validate ideas quickly before investing in implementation"},
		Appetite:         "medium",
		RawIdea:          "automate product discovery using AI agents",
	}
	result, err := discovery.Hypothesize(context.Background(), c, frame)
	if err != nil {
		t.Fatalf("hypothesize: %v", err)
	}
	if result.WeBelieve == "" {
		t.Error("empty we_believe")
	}
	if result.ToVerify == "" {
		t.Error("empty to_verify")
	}
	if result.WeMeasure == "" {
		t.Error("empty we_measure")
	}
	if result.WeAreRightIf == "" {
		t.Error("empty we_are_right_if")
	}
	if len(result.Assumptions) == 0 {
		t.Error("no assumptions")
	}
	// verify RAT ranking: rank 1 has highest or equal RAT score
	if len(result.Assumptions) > 1 {
		if result.Assumptions[0].RATScore < result.Assumptions[len(result.Assumptions)-1].RATScore {
			t.Error("assumptions not sorted by RAT score descending")
		}
		if result.Assumptions[0].RATRank != 1 {
			t.Errorf("first assumption should have RATRank=1, got %d", result.Assumptions[0].RATRank)
		}
	}
	t.Logf("we_believe: %s", result.WeBelieve)
	t.Logf("riskiest: %s (RAT=%.0f)", result.Assumptions[0].Statement, result.Assumptions[0].RATScore)
}
