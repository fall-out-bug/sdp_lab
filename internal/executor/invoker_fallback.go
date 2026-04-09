package executor

import (
	"context"
	"log"
	"os"
	"strings"

	"sdp_dev/internal/executor/omoclient"
	"sdp_dev/internal/kernel"
	"sdp_dev/internal/orchestrate"
)

// InvokeWithFallback tries ServeInvoker first, falls back to exec if serve API unavailable.
func InvokeWithFallback(ctx context.Context, req kernel.RuntimeInvocation) (kernel.RuntimeResult, error) {
	baseURL := strings.TrimSpace(os.Getenv("OMO_SERVE_URL"))
	if baseURL != "" {
		logger := log.New(log.Writer(), "[invoker] ", log.LstdFlags)
		serveInv := omoclient.NewServeInvoker(baseURL, logger)
		if running, _ := serveInv.Status(); running {
			result, err := serveInv.Invoke(ctx, req)
			if err == nil && result.Output != "" {
				return result, nil
			}
			logger.Printf("serve invoke failed (output=%d, err=%v), falling back to exec", len(result.Output), err)
		}
	}
	return orchestrate.GetDefaultInvoker().Invoke(ctx, req)
}

// execFallback invokes opencode directly via exec (no serve mode).
