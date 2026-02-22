package llm

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ExecuteRequest holds parameters for LLM execution via opencode run.
type ExecuteRequest struct {
	IssueID             string
	Title               string
	Description         string
	AcceptanceCriteria  string
	SpecID              string
	Model               string
	WorkDir             string
	Boundary            BoundarySpec
	Timeout             time.Duration
	OpencodeBinary      string // default "opencode"
}

// ExecuteResult holds the outcome of LLM execution.
type ExecuteResult struct {
	ChangedFiles       []string
	Stdout             string
	Stderr             string
	ExitCode           int
	Duration           time.Duration
	ModelUsed          string
	SessionID          string
	Prompt             string // prompt sent to opencode, for evidence
	BoundaryViolation  error
}

// DefaultTimeout is the default execution timeout for builder tasks.
const DefaultTimeout = 10 * time.Minute

// Execute runs opencode with the given request and returns the result.
// Changed files are detected via git diff after execution.
func Execute(ctx context.Context, req ExecuteRequest) (ExecuteResult, error) {
	if req.Timeout == 0 {
		req.Timeout = DefaultTimeout
	}
	if req.OpencodeBinary == "" {
		req.OpencodeBinary = "opencode"
	}
	issue := IssueInput{
		ID:                 req.IssueID,
		Title:              req.Title,
		Description:        req.Description,
		AcceptanceCriteria: req.AcceptanceCriteria,
		SpecID:             req.SpecID,
	}
	prompt := BuildPrompt(issue, req.Boundary)

	runCtx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()

	args := []string{"run", prompt, "--dir", req.WorkDir, "--format", "default"}
	if req.Model != "" {
		args = append(args, "--model", req.Model)
	}

	cmd := exec.CommandContext(runCtx, req.OpencodeBinary, args...)
	cmd.Dir = req.WorkDir
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	res := ExecuteResult{
		Stdout:    stdout.String(),
		Stderr:    stderr.String(),
		Duration:  duration,
		ModelUsed: req.Model,
		Prompt:    prompt,
	}
	if cmd.ProcessState != nil {
		res.ExitCode = cmd.ProcessState.ExitCode()
	}

	if err != nil {
		if runCtx.Err() == context.DeadlineExceeded {
			return res, fmt.Errorf("opencode run timed out after %v", req.Timeout)
		}
		return res, fmt.Errorf("opencode run: %w", err)
	}

	changed, err := getChangedFiles(req.WorkDir)
	if err != nil {
		return res, fmt.Errorf("detect changed files: %w", err)
	}
	res.ChangedFiles = changed

	if err := ValidateChangedPaths(changed, req.Boundary); err != nil {
		res.BoundaryViolation = err
		return res, err
	}

	return res, nil
}

// getChangedFiles returns paths modified since HEAD (staged or unstaged).
func getChangedFiles(workDir string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", "HEAD")
	cmd.Dir = workDir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	names := strings.Fields(strings.TrimSpace(string(out)))
	result := make([]string, 0, len(names))
	for _, n := range names {
		result = append(result, filepath.ToSlash(n))
	}
	return result, nil
}
