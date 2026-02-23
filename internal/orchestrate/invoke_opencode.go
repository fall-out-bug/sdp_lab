package orchestrate

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// buildPromptWithContext injects the pre-hydrated context packet into the prompt.
func buildPromptWithContext(dir, basePrompt string) string {
	pkt, err := LoadContextPacket(dir)
	if err != nil || pkt == nil {
		return basePrompt
	}
	return basePrompt + pkt.FormatForPrompt()
}

// InvokeOpenCode runs `opencode run --agent orchestrator` with the given prompt.
// Returns the combined stdout+stderr and exit code.
func InvokeOpenCode(ctx context.Context, dir, agent, prompt string) (string, int, error) {
	if agent == "" {
		agent = "orchestrator"
	}
	cmd := exec.CommandContext(ctx, "opencode", "run", "--agent", agent)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(prompt)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return string(out), exitErr.ExitCode(), nil
		}
		return string(out), -1, fmt.Errorf("opencode run: %w", err)
	}
	return string(out), 0, nil
}

// RunBuildPhase invokes opencode to execute a single @build workstream.
func RunBuildPhase(ctx context.Context, dir, wsID string) (commit string, err error) {
	prompt := buildPromptWithContext(dir, fmt.Sprintf("Execute @build %s. Output only code and commit message. After commit, output the commit hash.", wsID))
	out, code, err := InvokeOpenCode(ctx, dir, "implementer", prompt)
	if err != nil {
		return "", err
	}
	if code != 0 {
		return "", fmt.Errorf("opencode build exited %d: %s", code, out)
	}
	// Extract last line as commit hash if it looks like a SHA
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		s := strings.TrimSpace(lines[i])
		if len(s) == 40 && isHex(s) {
			return s, nil
		}
	}
	return "", nil
}

// RunReviewPhase invokes opencode to execute @review for a feature.
func RunReviewPhase(ctx context.Context, dir, featureID string) (approved bool, err error) {
	prompt := buildPromptWithContext(dir, fmt.Sprintf("Execute @review %s. Fix P0/P1 findings. Output APPROVED when done.", featureID))
	out, code, err := InvokeOpenCode(ctx, dir, "reviewer", prompt)
	if err != nil {
		return false, err
	}
	approved = code == 0 && strings.Contains(strings.ToUpper(out), "APPROVED")
	return approved, nil
}

func isHex(s string) bool {
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}
