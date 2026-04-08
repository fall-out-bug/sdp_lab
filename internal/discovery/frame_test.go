package discovery_test

import (
	"context"
	"os"
	"testing"
	"sdp_dev/internal/discovery"
)

func TestFrame_ProducesValidOutput(t *testing.T) {
	key := os.Getenv("OPENROUTER_API_KEY")
	if key == "" {
		t.Skip("OPENROUTER_API_KEY not set")
	}
	c := discovery.NewLLMClient(key, discovery.DefaultOpenRouterBase)
	result, err := discovery.Frame(context.Background(), c, "automate code review using AI agents")
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	if result.ProblemStatement == "" {
		t.Error("empty problem_statement")
	}
	if len(result.Jobs) == 0 {
		t.Error("no jobs identified")
	}
	if result.Appetite == "" {
		t.Error("empty appetite")
	}
	t.Logf("frame result: %+v", result)
}
