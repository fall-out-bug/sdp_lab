package discovery_test

import (
	"context"
	"os"
	"testing"

	"sdp_dev/internal/discovery"
)

func TestScan_ProducesItemsWithCoverage(t *testing.T) {
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
	result, err := discovery.Scan(context.Background(), c, frame)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(result.Items) == 0 {
		t.Fatal("scan returned no items")
	}
	// verify depth evaluation ran
	for _, item := range result.Items {
		if item.CoverageScore == 0 && item.Disposition != discovery.DispositionIgnore {
			t.Errorf("item %s: coverage score not set", item.Name)
		}
	}
	settled := result.Settled()
	flagged := result.Flagged()
	t.Logf("settled=%d flagged=%d whitespace=%s", len(settled), len(flagged), result.Whitespace)
}
