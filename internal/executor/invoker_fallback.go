package executor

import (
	"context"
	"log"
	"os"
	"strings"

	"sdp_dev/internal/executor/omoclient"
	"sdp_dev/internal/orchestrate"
)

// InvokeWithFallback tries ServeInvoker first, falls back to exec if serve API unavailable.
func InvokeWithFallback(ctx context.Context, projectRoot, agent, prompt string) (string, int, error) {
	baseURL := strings.TrimSpace(os.Getenv("OMO_SERVE_URL"))
	if baseURL != "" {
		logger := log.New(log.Writer(), "[invoker] ", log.LstdFlags)
		serveInv := omoclient.NewServeInvoker(baseURL, logger)
		if running, _ := serveInv.Status(); running {
			output, exitCode, err := serveInv.Invoke(ctx, projectRoot, agent, prompt)
			if err == nil && output != "" {
				return output, exitCode, nil
			}
			logger.Printf("serve invoke failed (output=%d, err=%v), falling back to exec", len(output), err)
		}
	}
	return orchestrate.DefaultLLMInvoker.Invoke(ctx, projectRoot, agent, prompt)
}

// execFallback invokes opencode directly via exec (no serve mode).
