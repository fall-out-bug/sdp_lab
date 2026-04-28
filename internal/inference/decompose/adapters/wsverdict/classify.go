package wsverdict

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/inference/decompose"
)

const modelSonnet = "anthropic/claude-sonnet-4-6"

var allowedVerdicts = []string{"passed", "partial", "failed"}

// newClassifyStage returns Stage[ExtractOut, string] that calls Sonnet with an
// enum-constrained prompt and returns one of "passed"|"partial"|"failed".
func newClassifyStage(client LLMCaller) decompose.Stage[ExtractOut, string] {
	return decompose.NewStage[ExtractOut, string]("classify", func(ctx context.Context, e ExtractOut) (string, decompose.StageTrace, error) {
		start := time.Now()
		prompt := classifyPrompt(e)
		text, tokIn, tokOut, cost, err := client.Call(ctx, prompt, CallOptions{
			Model:     modelSonnet,
			MaxTokens: 16,
		})
		trace := decompose.StageTrace{
			LatencyMs:   time.Since(start).Milliseconds(),
			TokensIn:    tokIn,
			TokensOut:   tokOut,
			CostUSD:     cost,
			RawResponse: text,
		}
		if err != nil {
			return "", trace, fmt.Errorf("classify LLM call: %w", err)
		}
		verdict := strings.TrimSpace(strings.ToLower(text))
		if !isAllowed(verdict, allowedVerdicts) {
			return "", trace, fmt.Errorf("classify: unexpected verdict %q (allowed: %v)", verdict, allowedVerdicts)
		}
		return verdict, trace, nil
	})
}

func classifyPrompt(e ExtractOut) string {
	return fmt.Sprintf(`Based on this code change analysis, classify the workstream verdict.

Change type: %s
Summary: %s
Changed files: %s
Modules affected: %s

Reply with EXACTLY ONE of these words (nothing else): passed, partial, failed

- "passed": all acceptance criteria met, quality gates pass
- "partial": most criteria met, minor gaps remain
- "failed": critical criteria unmet or quality gates failing`,
		e.ChangeType, e.Summary,
		strings.Join(e.ChangedFiles, ", "),
		strings.Join(e.Modules, ", "))
}

func isAllowed(s string, allowed []string) bool {
	for _, v := range allowed {
		if s == v {
			return true
		}
	}
	return false
}
