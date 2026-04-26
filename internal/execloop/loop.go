package execloop

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"sdp_dev/internal/build"
)

// Config holds the self-testing loop configuration.
type Config struct {
	// MaxAttempts is the maximum number of iterations (default: 3).
	MaxAttempts int
	// Sandbox is the sandbox used for build/test execution.
	Sandbox build.Sandbox
	// RepoPath is the repository path.
	RepoPath string
	// RunID is the parent run ID.
	RunID string
	// OutputDir is the evidence output directory.
	OutputDir string
	// ScopeFiles are the files the agent declares it will touch.
	ScopeFiles []string
}

// AttemptResult holds the result of a single attempt.
type AttemptResult struct {
	Attempt    int           `json:"attempt"`
	Status     string        `json:"status"` // success|failed|scope_creep|error
	BuildOK    bool          `json:"build_ok"`
	TestsOK    bool          `json:"tests_ok"`
	TestOutput string        `json:"test_output,omitempty"`
	Diff       string        `json:"diff,omitempty"`
	Duration   time.Duration `json:"duration"`
	Error      string        `json:"error,omitempty"`
}

// LoopResult holds the final result of the self-testing loop.
type LoopResult struct {
	RunID     string          `json:"run_id"`
	Status    string          `json:"status"` // converged|max_attempts|scope_creep|error
	Attempts  []AttemptResult `json:"attempts"`
	Winner    int             `json:"winner,omitempty"` // 1-based attempt that passed (0 if none)
	TotalTime time.Duration   `json:"total_time"`
}

// Loop defines the self-testing loop interface.
type Loop interface {
	// Run executes the loop until tests pass or max attempts are reached.
	Run(ctx context.Context) (*LoopResult, error)
}

// DefaultLoop implements the self-testing loop.
type DefaultLoop struct {
	cfg Config
}

// NewDefaultLoop creates a new self-testing loop with the given configuration.
// Returns an error if MaxAttempts < 1 or Sandbox is nil.
func NewDefaultLoop(cfg Config) (*DefaultLoop, error) {
	if cfg.MaxAttempts < 1 {
		return nil, fmt.Errorf("execloop: MaxAttempts must be >= 1, got %d", cfg.MaxAttempts)
	}
	if cfg.Sandbox == nil {
		return nil, fmt.Errorf("execloop: Sandbox is required")
	}
	return &DefaultLoop{cfg: cfg}, nil
}

// Run executes the self-testing loop.
func (l *DefaultLoop) Run(ctx context.Context) (*LoopResult, error) {
	if l.cfg.Sandbox != nil {
		defer func() { _ = l.cfg.Sandbox.Cleanup() }()
	}
	start := time.Now()
	result := &LoopResult{
		RunID: l.cfg.RunID,
	}

	if err := l.ensureOutputDir(); err != nil {
		result.Status = "error"
		return result, fmt.Errorf("execloop: create output dirs: %w", err)
	}

	for attempt := 1; attempt <= l.cfg.MaxAttempts; attempt++ {
		// Check for context cancellation before each attempt.
		if ctx.Err() != nil {
			result.Status = "error"
			result.TotalTime = time.Since(start)
			if err := l.writeLoopResult(result); err != nil {
				slog.Error("execloop: failed to write loop result on context cancel", "error", err, "run_id", l.cfg.RunID)
			}
			return result, ctx.Err()
		}

		ar := l.runAttempt(ctx, attempt)
		result.Attempts = append(result.Attempts, ar)

		// Write individual attempt evidence.
		if err := l.writeAttemptResult(attempt, &ar); err != nil {
			slog.Error("execloop: failed to write attempt evidence", "error", err, "attempt", attempt, "run_id", l.cfg.RunID)
		}

		switch ar.Status {
		case "success":
			result.Status = "converged"
			result.Winner = attempt
			result.TotalTime = time.Since(start)
			if err := l.writeLoopResult(result); err != nil {
				slog.Error("execloop: failed to write loop result", "error", err, "run_id", l.cfg.RunID)
			}
			slog.Info("execloop: converged",
				"run_id", l.cfg.RunID,
				"attempt", attempt,
				"total_time", result.TotalTime,
			)
			return result, nil

		case "scope_creep":
			result.Status = "scope_creep"
			result.TotalTime = time.Since(start)
			if err := l.writeLoopResult(result); err != nil {
				slog.Error("execloop: failed to write loop result on scope creep", "error", err, "run_id", l.cfg.RunID)
			}
			slog.Warn("execloop: scope creep detected",
				"run_id", l.cfg.RunID,
				"attempt", attempt,
				"diff", ar.Diff,
			)
			return result, nil

		case "error":
			result.Status = "error"
			result.TotalTime = time.Since(start)
			if err := l.writeLoopResult(result); err != nil {
				slog.Error("execloop: failed to write loop result on error", "error", err, "run_id", l.cfg.RunID)
			}
			return result, fmt.Errorf("execloop: attempt %d error: %s", attempt, ar.Error)
		}

		// "failed" — continue to next attempt.
		slog.Info("execloop: attempt failed, retrying",
			"run_id", l.cfg.RunID,
			"attempt", attempt,
			"build_ok", ar.BuildOK,
			"tests_ok", ar.TestsOK,
		)
	}

	result.Status = "max_attempts"
	result.TotalTime = time.Since(start)
	if err := l.writeLoopResult(result); err != nil {
		slog.Error("execloop: failed to write loop result on max attempts", "error", err, "run_id", l.cfg.RunID)
	}
	slog.Info("execloop: max attempts reached",
		"run_id", l.cfg.RunID,
		"attempts", l.cfg.MaxAttempts,
		"total_time", result.TotalTime,
	)
	return result, nil
}

// runAttempt executes a single build+test attempt.
func (l *DefaultLoop) runAttempt(ctx context.Context, attempt int) AttemptResult {
	start := time.Now()
	ar := AttemptResult{
		Attempt: attempt,
	}

	repo := l.cfg.RepoPath
	if repo == "" {
		repo, _ = os.Getwd()
	}

	// Step 1: Build.
	buildRes, err := l.cfg.Sandbox.Build(ctx, repo)
	if err != nil {
		ar.Status = "error"
		ar.Error = fmt.Sprintf("build: %v", err)
		ar.Duration = time.Since(start)
		return ar
	}

	if !buildRes.Success {
		ar.Status = "failed"
		ar.BuildOK = false
		ar.Error = buildRes.Stderr
		ar.Duration = time.Since(start)
		return ar
	}
	ar.BuildOK = true

	// Step 2: Test.
	testRes, err := l.cfg.Sandbox.Test(ctx, repo)
	if err != nil {
		ar.Status = "error"
		ar.Error = fmt.Sprintf("test: %v", err)
		ar.Duration = time.Since(start)
		return ar
	}

	if !testRes.Success {
		ar.Status = "failed"
		ar.TestsOK = false
		ar.TestOutput = testRes.Stderr
		ar.Duration = time.Since(start)
		return ar
	}

	ar.TestsOK = true

	// Step 3: Scope creep detection.
	if len(l.cfg.ScopeFiles) > 0 {
		creep, diff, err := l.detectScopeCreep(repo)
		if err != nil {
			slog.Warn("execloop: scope creep check failed", "error", err, "run_id", l.cfg.RunID)
		} else if creep {
			ar.Status = "scope_creep"
			ar.Diff = diff
			ar.Duration = time.Since(start)
			return ar
		}
	}

	ar.Status = "success"
	ar.Duration = time.Since(start)
	return ar
}

// detectScopeCreep checks if files outside ScopeFiles were modified,
// staged, or created (untracked). Returns (true, diffOutput, nil) if
// scope creep is detected.
func (l *DefaultLoop) detectScopeCreep(repo string) (bool, string, error) {
	// Use --porcelain to capture staged, unstaged, and untracked files.
	cmd := exec.Command("git", "status", "--porcelain", "--untracked-files=all")
	cmd.Dir = repo
	out, err := cmd.Output()
	if err != nil {
		return false, "", fmt.Errorf("git status --porcelain: %w", err)
	}

	// Build a set of allowed files for O(1) lookup.
	allowed := make(map[string]struct{}, len(l.cfg.ScopeFiles))
	for _, f := range l.cfg.ScopeFiles {
		allowed[f] = struct{}{}
	}

	var outside []string
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" {
			continue
		}
		// porcelain format: "XY filename" — strip the 2-char status + space.
		// Do NOT TrimSpace: XY may be " M" (space+M) for unstaged changes.
		if len(line) <= 3 {
			continue
		}
		f := line[3:]
		// Renames show as "XY old -> new" — extract the new name.
		if idx := strings.Index(line, "->"); idx >= 0 {
			f = strings.TrimSpace(line[idx+2:])
		}
		// Skip evidence artifacts created by the loop itself.
		if strings.HasPrefix(f, ".sdp/") {
			continue
		}
		if _, ok := allowed[f]; !ok {
			outside = append(outside, f)
		}
	}

	if len(outside) > 0 {
		return true, strings.Join(outside, ", "), nil
	}
	return false, "", nil
}

// ensureOutputDir creates the evidence output directories.
func (l *DefaultLoop) ensureOutputDir() error {
	if l.cfg.OutputDir == "" {
		return nil
	}
	attemptsDir := filepath.Join(l.cfg.OutputDir, "attempts")
	if err := os.MkdirAll(attemptsDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", attemptsDir, err)
	}
	return nil
}

// writeAttemptResult writes a single attempt result to the evidence directory.
func (l *DefaultLoop) writeAttemptResult(attempt int, ar *AttemptResult) error {
	if l.cfg.OutputDir == "" {
		return nil
	}
	path := filepath.Join(l.cfg.OutputDir, "attempts", fmt.Sprintf("%d.json", attempt))
	return writeJSON(path, ar)
}

// writeLoopResult writes the final loop result to the evidence directory.
func (l *DefaultLoop) writeLoopResult(result *LoopResult) error {
	if l.cfg.OutputDir == "" {
		return nil
	}
	path := filepath.Join(l.cfg.OutputDir, "loop-result.json")
	return writeJSON(path, result)
}

// writeJSON writes a value as indented JSON to the given path using atomic write.
func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}

	f, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpName := f.Name()

	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmpName)
		return fmt.Errorf("write temp: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("rename temp: %w", err)
	}
	return nil
}
