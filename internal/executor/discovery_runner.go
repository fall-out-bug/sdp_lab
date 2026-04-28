package executor

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/fall-out-bug/sdp_lab/internal/control"
)

// isDiscoveryCard returns true if the card has IssueType "discovery".
func isDiscoveryCard(card *control.FeatureCard) bool {
	return card != nil && card.IssueType == "discovery"
}

// extractDiscoveryIdea strips the "Discovery: " prefix from a title.
func extractDiscoveryIdea(title string) string {
	return strings.TrimPrefix(title, "Discovery: ")
}

// findSdpBinary returns the path to the sdp binary using PATH lookup.
func findSdpBinary() (string, error) {
	path, err := exec.LookPath("sdp")
	if err != nil {
		return "", fmt.Errorf("sdp binary not found in PATH: %w", err)
	}
	return path, nil
}

// RunDiscoveryFromCard runs the SDP discovery pipeline for a discovery-typed card.
// It shells out to the sdp binary and, on completion, closes the card via bd.
func RunDiscoveryFromCard(ctx context.Context, store *control.Store, card *control.FeatureCard, projectRoot string) error {
	if card == nil {
		return fmt.Errorf("nil card")
	}

	logger := slog.Default().With("component", "executor.discovery-runner", "card_id", card.ID)

	idea := extractDiscoveryIdea(card.Title)
	logger.Info("running discovery pipeline", "idea", idea)

	sdpPath, err := findSdpBinary()
	if err != nil {
		return fmt.Errorf("find sdp binary: %w", err)
	}

	sdpCmd := exec.CommandContext(ctx, sdpPath, "discovery", idea)
	sdpCmd.Dir = projectRoot
	sdpCmd.Env = os.Environ()
	sdpCmd.Stdout = os.Stdout
	sdpCmd.Stderr = os.Stderr

	if runErr := sdpCmd.Run(); runErr != nil {
		logger.Error("discovery pipeline failed", "error", runErr)
		return fmt.Errorf("sdp discovery %q: %w", idea, runErr)
	}

	logger.Info("discovery pipeline completed, closing card", "card_id", card.ID)

	closeCmd := exec.CommandContext(ctx, "bd", "close", card.ID)
	closeCmd.Env = os.Environ()
	if out, err := closeCmd.CombinedOutput(); err != nil {
		logger.Warn("failed to close discovery card", "card_id", card.ID, "error", err, "output", string(out))
	}

	return nil
}
