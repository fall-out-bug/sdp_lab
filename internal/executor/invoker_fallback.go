package executor

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/fall-out-bug/sdp_lab/internal/executor/omoclient"
	"github.com/fall-out-bug/sdp_lab/internal/kernel"
	"github.com/fall-out-bug/sdp_lab/internal/orchestrate"
)

// InvokeWithFallback tries ServeInvoker first, falls back to exec if serve API unavailable.
func InvokeWithFallback(ctx context.Context, req kernel.RuntimeInvocation) (kernel.RuntimeResult, error) {
	baseURL := strings.TrimSpace(os.Getenv("OMO_SERVE_URL"))
	if baseURL != "" {
		serveInv := omoclient.NewServeInvoker(baseURL)
		if running, _ := serveInv.Status(); running {
			result, err := serveInv.Invoke(ctx, req)
			if err == nil && result.Output != "" {
				return result, nil
			}
			slog.Debug("serve invoke failed, falling back to exec", "output_length", len(result.Output), "error", err)
		}
	}
	return orchestrate.GetDefaultInvoker().Invoke(ctx, req)
}

// execFallback invokes opencode directly via exec (no serve mode).
