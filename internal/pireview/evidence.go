package pireview

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CollectTestEvidence runs the configured test command and captures deterministic evidence.
func CollectTestEvidence(ctx context.Context, cfg Config, runDir string) (*TestEvidence, error) {
	if runDir == "" {
		runDir = filepath.Join(cfg.ProjectRoot, ".sdp", "runs", "pi-review", "manual")
	}
	artifactPath := filepath.Join(runDir, "test-output.txt")

	cmd, err := resolveTestCommand(cfg)
	if err != nil {
		if err := ensurePrivateDir(runDir); err != nil {
			return nil, fmt.Errorf("evidence: mkdir: %w", err)
		}
		if err := writePrivateFile(artifactPath, []byte(err.Error()+"\n")); err != nil {
			return nil, fmt.Errorf("evidence: write skipped artifact: %w", err)
		}
		return &TestEvidence{
			Status:       "skipped",
			SkipReason:   err.Error(),
			ArtifactPath: artifactPath,
		}, nil
	}

	start := time.Now()
	out, exitCode := runTestCommand(ctx, cfg.ProjectRoot, cmd)
	duration := time.Since(start)

	if err := ensurePrivateDir(runDir); err != nil {
		return nil, fmt.Errorf("evidence: mkdir: %w", err)
	}

	if err := writePrivateFile(artifactPath, []byte(out)); err != nil {
		return nil, fmt.Errorf("evidence: write artifact: %w", err)
	}

	status := "passed"
	if exitCode != 0 {
		status = "failed"
	}

	return &TestEvidence{
		Status:       status,
		Command:      strings.Join(cmd, " "),
		ExitCode:     exitCode,
		DurationMs:   duration.Milliseconds(),
		ArtifactPath: artifactPath,
		Output:       out,
	}, nil
}

// resolveTestCommand determines which test command to run.
func resolveTestCommand(cfg Config) ([]string, error) {
	if cfg.TestCommand != "" {
		return strings.Fields(cfg.TestCommand), nil
	}

	// Detect based on project files
	root := cfg.ProjectRoot

	if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
		return []string{"go", "test", "./..."}, nil
	}

	if _, err := os.Stat(filepath.Join(root, "package.json")); err == nil {
		return []string{"npm", "test"}, nil
	}

	if _, err := os.Stat(filepath.Join(root, "pytest.ini")); err == nil {
		return []string{"pytest", "-q"}, nil
	}

	if _, err := os.Stat(filepath.Join(root, "pyproject.toml")); err == nil {
		return []string{"pytest", "-q"}, nil
	}

	return nil, fmt.Errorf("no test command configured and no project file detected")
}

// runTestCommand executes the test command and returns output + exit code.
func runTestCommand(ctx context.Context, dir string, cmd []string) (string, int) {
	if len(cmd) == 0 {
		return "", 1
	}

	c := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	c.Dir = dir
	out, err := c.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return string(out), exitErr.ExitCode()
		}
		return string(out), 1
	}
	return string(out), 0
}
