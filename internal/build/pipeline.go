package build

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// BuildConfig holds the configuration for a build run.
type BuildConfig struct {
	Idea      string `json:"idea"`
	Strict    bool   `json:"strict"`
	Local     bool   `json:"local"`
	Sandbox   string `json:"sandbox"`
	DryRun    bool   `json:"dry_run"`
	Format    string `json:"format"`
	OutputDir string `json:"output_dir"`
	RunID     string `json:"run_id"`
	RepoPath  string `json:"repo_path"`
	CGO       bool   `json:"cgo"`
}

// StageResult holds the result of a single pipeline stage.
type StageResult struct {
	Stage    string         `json:"stage"`
	Status   string         `json:"status"`
	Output   string         `json:"output,omitempty"`
	Error    string         `json:"error,omitempty"`
	Duration time.Duration  `json:"duration"`
	Evidence map[string]any `json:"evidence,omitempty"`
}

// BuildResult holds the complete build result.
type BuildResult struct {
	RunID     string         `json:"run_id"`
	Config    BuildConfig    `json:"config"`
	Stages    []StageResult  `json:"stages"`
	Status    string         `json:"status"`
	Summary   string         `json:"summary"`
	Evidence  map[string]any `json:"evidence,omitempty"`
	StartedAt time.Time      `json:"started_at"`
	EndedAt   time.Time      `json:"ended_at"`
}

// Pipeline defines the build pipeline interface.
type Pipeline interface {
	// Run executes the full build pipeline.
	Run(ctx context.Context) (*BuildResult, error)
	// DryRun returns the plan without executing.
	DryRun() (*BuildResult, error)
}

// DefaultPipeline implements the vibecode build pipeline.
type DefaultPipeline struct {
	config  BuildConfig
	sandbox Sandbox
}

// NewDefaultPipeline creates a pipeline with the given config.
func NewDefaultPipeline(cfg BuildConfig) (*DefaultPipeline, error) {
	if cfg.RunID == "" {
		cfg.RunID = uuid.New().String()
	}
	if cfg.OutputDir == "" {
		cfg.OutputDir = filepath.Join(".sdp", "evidence", cfg.RunID)
	}

	sandbox, err := NewSandbox(cfg.Sandbox, cfg.CGO)
	if err != nil {
		return nil, fmt.Errorf("build: create sandbox: %w", err)
	}

	return &DefaultPipeline{
		config:  cfg,
		sandbox: sandbox,
	}, nil
}

// DryRun returns a BuildResult showing what the pipeline would do, without executing.
func (p *DefaultPipeline) DryRun() (*BuildResult, error) {
	now := time.Now().UTC()
	return &BuildResult{
		RunID:  p.config.RunID,
		Config: p.config,
		Stages: []StageResult{
			{Stage: "dispatch", Status: "skipped", Output: "(dry-run) would classify idea and route to harness"},
			{Stage: "sandbox", Status: "skipped", Output: fmt.Sprintf("(dry-run) would run build+test with sandbox=%s", p.config.Sandbox)},
			{Stage: "test", Status: "skipped", Output: "(dry-run) would collect test results"},
			{Stage: "commit", Status: "skipped", Output: "(dry-run) would write evidence.json"},
		},
		Status:    "success",
		Summary:   "dry-run: no stages executed",
		StartedAt: now,
		EndedAt:   now,
	}, nil
}

// Run executes the full vibecode build pipeline.
func (p *DefaultPipeline) Run(ctx context.Context) (*BuildResult, error) {
	defer p.sandbox.Cleanup()

	if p.config.Strict {
		return nil, fmt.Errorf("build: strict mode (Phase FSM) not yet implemented (F134)")
	}

	result := &BuildResult{
		RunID:     p.config.RunID,
		Config:    p.config,
		Status:    "success",
		Evidence:  make(map[string]any),
		StartedAt: time.Now().UTC(),
	}
	defer func() {
		result.EndedAt = time.Now().UTC()
	}()

	if err := os.MkdirAll(p.config.OutputDir, 0o755); err != nil {
		return nil, fmt.Errorf("build: create output dir %s: %w", p.config.OutputDir, err)
	}

	// Stage 1: Dispatch — classify the idea and record routing decision.
	dispatchResult := p.runDispatchStage(ctx)
	result.Stages = append(result.Stages, dispatchResult)
	if dispatchResult.Status == "failed" {
		result.Status = "failed"
		result.Summary = fmt.Sprintf("dispatch stage failed: %s", dispatchResult.Error)
		if err := WriteBuildResult(result, p.config.OutputDir); err != nil {
			slog.Error("build: failed to write evidence on error path", "error", err, "run_id", p.config.RunID)
		}
		return result, nil
	}

	// Stage 2+3: Sandbox — build and test.
	sandboxResult := p.runSandboxStage(ctx)
	result.Stages = append(result.Stages, sandboxResult)
	if sandboxResult.Status == "failed" {
		result.Status = "partial"
		result.Summary = fmt.Sprintf("sandbox stage failed: %s", sandboxResult.Error)
		if err := WriteBuildResult(result, p.config.OutputDir); err != nil {
			slog.Error("build: failed to write evidence on error path", "error", err, "run_id", p.config.RunID)
		}
		return result, nil
	}

	// Stage 4: Commit — write evidence.
	commitResult := p.runCommitStage(ctx, result)
	result.Stages = append(result.Stages, commitResult)
	if commitResult.Status == "failed" {
		result.Status = "partial"
		result.Summary = fmt.Sprintf("commit stage failed: %s", commitResult.Error)
		return result, nil
	}

	result.EndedAt = time.Now().UTC()
	result.Summary = p.buildSummary(result)
	return result, nil
}

// runDispatchStage classifies the idea and records the dispatch decision.
// This is a stub: it classifies the idea text but does not invoke a live harness.
func (p *DefaultPipeline) runDispatchStage(ctx context.Context) StageResult {
	start := time.Now()
	sr := StageResult{Stage: "dispatch", Evidence: make(map[string]any)}

	classification := classifyIdea(p.config.Idea)
	sr.Evidence["classification"] = classification

	decision := defaultDispatchDecision(p.config, classification)
	sr.Evidence["decision"] = decision

	sr.Output = fmt.Sprintf("classified as %s/%s -> %s (%s/%s, score=%.2f)",
		classification.TaskType, classification.Complexity,
		decision.Harness, decision.Provider, decision.Model, decision.Score)
	sr.Status = "success"
	sr.Duration = time.Since(start)

	slog.Info("build: dispatch stage complete",
		"run_id", p.config.RunID,
		"task_type", classification.TaskType,
		"harness", decision.Harness,
		"duration", sr.Duration,
	)
	return sr
}

// runSandboxStage builds and tests the project.
func (p *DefaultPipeline) runSandboxStage(ctx context.Context) StageResult {
	start := time.Now()
	repo := p.config.RepoPath
	if repo == "" {
		var err error
		repo, err = os.Getwd()
		if err != nil {
			return StageResult{
				Stage:    "sandbox",
				Status:   "failed",
				Error:    fmt.Sprintf("get working directory: %v", err),
				Duration: time.Since(start),
			}
		}
	}

	buildRes, err := p.sandbox.Build(ctx, repo)
	if err != nil {
		return StageResult{
			Stage:    "sandbox",
			Status:   "failed",
			Error:    fmt.Sprintf("sandbox build: %v", err),
			Duration: time.Since(start),
		}
	}

	// If build failed, skip tests — they would always fail anyway.
	if !buildRes.Success {
		return StageResult{
			Stage:    "sandbox",
			Status:   "failed",
			Error:    fmt.Sprintf("build failed (exit %d): %s", buildRes.ExitCode, buildRes.Stderr),
			Output:   fmt.Sprintf("[%s] build:FAIL (exit %d)", p.config.Sandbox, buildRes.ExitCode),
			Duration: time.Since(start),
			Evidence: map[string]any{"build_ok": false, "tests_ok": false, "sandbox_type": p.config.Sandbox},
		}
	}

	testRes, err := p.sandbox.Test(ctx, repo)
	if err != nil {
		return StageResult{
			Stage:    "sandbox",
			Status:   "failed",
			Error:    fmt.Sprintf("sandbox test: %v", err),
			Duration: time.Since(start),
		}
	}

	sr := StageResult{
		Stage:    "sandbox",
		Status:   "success",
		Duration: time.Since(start),
		Evidence: make(map[string]any),
	}
	sr.Evidence["build_ok"] = buildRes.Success
	sr.Evidence["tests_ok"] = testRes.Success
	sr.Evidence["sandbox_type"] = p.config.Sandbox

	var parts []string
	if buildRes.Success {
		parts = append(parts, "build:ok")
	} else {
		parts = append(parts, fmt.Sprintf("build:FAIL (exit %d)", buildRes.ExitCode))
	}
	if testRes.Success {
		parts = append(parts, "tests:ok")
	} else {
		parts = append(parts, fmt.Sprintf("tests:FAIL (exit %d)", testRes.ExitCode))
		sr.Status = "failed"
		sr.Error = testRes.Stderr
	}
	sr.Output = fmt.Sprintf("[%s] %s", p.config.Sandbox, strings.Join(parts, " | "))

	slog.Info("build: sandbox stage complete",
		"run_id", p.config.RunID,
		"build_ok", buildRes.Success,
		"tests_ok", testRes.Success,
		"duration", sr.Duration,
	)
	return sr
}

// runCommitStage writes evidence to the output directory.
// It does NOT perform a git commit — the caller/CLI decides that.
func (p *DefaultPipeline) runCommitStage(ctx context.Context, result *BuildResult) StageResult {
	start := time.Now()
	sr := StageResult{Stage: "commit", Evidence: make(map[string]any)}

	if err := WriteBuildResult(result, p.config.OutputDir); err != nil {
		sr.Status = "failed"
		sr.Error = fmt.Sprintf("write evidence: %v", err)
		sr.Duration = time.Since(start)
		return sr
	}

	sr.Status = "success"
	sr.Output = fmt.Sprintf("evidence written to %s", p.config.OutputDir)
	sr.Duration = time.Since(start)

	slog.Info("build: commit stage complete",
		"run_id", p.config.RunID,
		"output_dir", p.config.OutputDir,
		"duration", sr.Duration,
	)
	return sr
}

// buildSummary creates a human-readable summary of the pipeline result.
func (p *DefaultPipeline) buildSummary(result *BuildResult) string {
	var successes, failures int
	for _, s := range result.Stages {
		switch s.Status {
		case "success":
			successes++
		case "failed":
			failures++
		}
	}
	elapsed := result.EndedAt.Sub(result.StartedAt).Round(time.Millisecond)
	if failures == 0 {
		return fmt.Sprintf("all %d stages passed (%s)", successes, elapsed)
	}
	return fmt.Sprintf("%d/%d stages passed, %d failed (%s)", successes, len(result.Stages), failures, elapsed)
}

// --- Classification helpers (lightweight, no LLM) ---

type ideaClassification struct {
	TaskType    string `json:"task_type"`
	Complexity  string `json:"complexity"`
	RequiredCap string `json:"required_cap"`
}

// classifyIdea uses simple heuristics to classify the idea text.
func classifyIdea(idea string) ideaClassification {
	ic := ideaClassification{
		TaskType:    "feature",
		Complexity:  "medium",
		RequiredCap: "coding",
	}
	lower := strings.ToLower(idea)

	switch {
	case strings.Contains(lower, "refactor"):
		ic.TaskType = "refactor"
	case strings.Contains(lower, "fix") || strings.Contains(lower, "bug"):
		ic.TaskType = "bugfix"
	case strings.Contains(lower, "research") || strings.Contains(lower, "investigate") || strings.Contains(lower, "explore"):
		ic.TaskType = "research"
		ic.RequiredCap = "reasoning"
	case strings.Contains(lower, "design") || strings.Contains(lower, "architect"):
		ic.TaskType = "architecture"
		ic.RequiredCap = "reasoning"
	case strings.Contains(lower, "review") || strings.Contains(lower, "audit"):
		ic.TaskType = "analysis"
		ic.RequiredCap = "review"
	}

	switch {
	case strings.Contains(lower, "simple") || strings.Contains(lower, "trivial") || strings.Contains(lower, "minor"):
		ic.Complexity = "low"
	case strings.Contains(lower, "complex") || strings.Contains(lower, "multi-step") || strings.Contains(lower, "large"):
		ic.Complexity = "high"
	}

	return ic
}

// defaultDispatchDecision produces a plausible dispatch decision for the given config and classification.
func defaultDispatchDecision(cfg BuildConfig, ic ideaClassification) dispatchDecision {
	dec := dispatchDecision{
		Reason: fmt.Sprintf("auto-classified as %s/%s", ic.TaskType, ic.Complexity),
		Score:  0.8,
	}

	if cfg.Local {
		dec.Harness = "ollama"
		dec.Provider = "ollama"
		dec.Model = "devstral:24b"
	} else {
		dec.Harness = "claude-code"
		dec.Provider = "anthropic"
		dec.Model = "claude-sonnet-4-20250514"
	}

	return dec
}

type dispatchDecision struct {
	Harness  string  `json:"harness"`
	Provider string  `json:"provider"`
	Model    string  `json:"model"`
	Score    float64 `json:"score"`
	Reason   string  `json:"reason"`
}

// --- Command execution ---

// runGoCommand runs a go command in the given directory.
func runGoCommand(ctx context.Context, dir string, cgo bool, args ...string) (*SandboxResult, error) {
	start := time.Now()
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	if !cgo {
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	}

	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	errPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %v: %w", args, err)
	}

	// Read output concurrently.
	type readResult struct {
		data []byte
	}
	outCh := make(chan readResult, 1)
	errCh := make(chan readResult, 1)
	go func() {
		b, _ := io.ReadAll(outPipe)
		outCh <- readResult{b}
	}()
	go func() {
		b, _ := io.ReadAll(errPipe)
		errCh <- readResult{b}
	}()

	waitErr := cmd.Wait()
	stdout := (<-outCh).data
	stderr := (<-errCh).data

	if waitErr != nil {
		return &SandboxResult{
			Success:  false,
			ExitCode: cmd.ProcessState.ExitCode(),
			Stdout:   string(stdout),
			Stderr:   string(stderr),
			Duration: time.Since(start),
		}, nil
	}

	return &SandboxResult{
		Success:  true,
		ExitCode: 0,
		Stdout:   string(stdout),
		Stderr:   string(stderr),
		Duration: time.Since(start),
	}, nil
}
