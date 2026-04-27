// internal/discovery/clarify_test.go
package discovery_test

import (
	"context"
	"os"
	"testing"
	"github.com/fall-out-bug/sdp_lab/internal/discovery"
)

func TestGenerateClarifications_ProducesQuestions(t *testing.T) {
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
	reqs, err := discovery.GenerateClarifications(context.Background(), c, frame)
	if err != nil {
		t.Fatalf("clarify: %v", err)
	}
	if len(reqs) == 0 {
		t.Error("no clarification requests generated")
	}
	for i, r := range reqs {
		if r.Question == "" {
			t.Errorf("clarification[%d]: empty question", i)
		}
		validTypes := map[string]bool{
			"missing_info": true, "ambiguous_requirement": true,
			"approach_choice": true, "risk_confirmation": true,
		}
		if !validTypes[string(r.Type)] {
			t.Errorf("clarification[%d]: unknown type %q", i, r.Type)
		}
	}
	t.Logf("clarifications: %d requests", len(reqs))
	for _, r := range reqs {
		t.Logf("  [%s] %s", r.Type, r.Question)
	}
}
