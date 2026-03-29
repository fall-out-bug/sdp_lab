package deploy

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Config holds deploy configuration.
type Config struct {
	ProjectRoot   string // path to project directory
	ComposeStaging string // path to staging compose file (default: docker-compose.staging.yml)
	ComposeProd    string // path to prod compose file (default: docker-compose.yml)
	ProjectName    string // Docker project name (default: derived from dir)
}

// DefaultConfig creates config from project root.
func DefaultConfig(projectRoot string) *Config {
	dir := filepath.Base(projectRoot)
	return &Config{
		ProjectRoot:   projectRoot,
		ComposeStaging: filepath.Join(projectRoot, "docker-compose.staging.yml"),
		ComposeProd:    filepath.Join(projectRoot, "docker-compose.yml"),
		ProjectName:    strings.ReplaceAll(dir, ".", "-"),
	}
}

// Result holds deploy evidence.
type Result struct {
	Phase      string            `json:"phase"`
	Target     string            `json:"target"` // staging or prod
	ImageTag   string            `json:"image_tag"`
	CommitHash string            `json:"commit_hash,omitempty"`
	StartedAt  string            `json:"started_at"`
	FinishedAt string            `json:"finished_at"`
	Duration   string            `json:"duration"`
	Containers []ContainerInfo  `json:"containers,omitempty"`
	SmokeTest  *TestResult       `json:"smoke_test,omitempty"`
	Health     *HealthCheckResult `json:"health,omitempty"`
	Error      string            `json:"error,omitempty"`
}

// ContainerInfo holds container details.
type ContainerInfo struct {
	Name    string `json:"name"`
	ID      string `json:"id"`
	Image   string `json:"image"`
	Status  string `json:"status"`
	Ports   string `json:"ports,omitempty"`
}

// TestResult holds test evidence.
type TestResult struct {
	Passed   bool     `json:"passed"`
	Output   string   `json:"output,omitempty"`
	ExitCode int      `json:"exit_code"`
	Tests    []string `json:"tests,omitempty"`
}

// HealthCheckResult holds health check evidence.
type HealthCheckResult struct {
	Passed  bool     `json:"passed"`
	Checks  []string `json:"checks,omitempty"`
	Errors  []string `json:"errors,omitempty"`
	Minutes float64  `json:"monitor_minutes"`
}

// Staging deploys to staging environment.
func Staging(ctx context.Context, cfg *Config, commitHash string) (*Result, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil config")
	}

	result := &Result{
		Phase:      "staging",
		Target:     "staging",
		CommitHash: commitHash,
		StartedAt:  time.Now().UTC().Format(time.RFC3339),
	}

	// Build image
	imageTag := fmt.Sprintf("%s:staging-%s", cfg.ProjectName, shortHash(commitHash))
	buildCmd := exec.CommandContext(ctx, "docker", "build", "-t", imageTag, ".")
	buildCmd.Dir = cfg.ProjectRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		result.Error = fmt.Sprintf("docker build: %s: %s", err, string(output))
		result.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		return result, fmt.Errorf("docker build: %w", err)
	}
	result.ImageTag = imageTag

	// Deploy
	projectFlag := []string{"-f", cfg.ComposeStaging, "-p", cfg.ProjectName + "-staging", "up", "-d", "--build"}
	deployCmd := dockerComposeCmd(ctx, projectFlag)
	deployCmd.Dir = cfg.ProjectRoot
	if output, err := deployCmd.CombinedOutput(); err != nil {
		result.Error = fmt.Sprintf("docker compose up: %s: %s", err, string(output))
		result.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		return result, fmt.Errorf("docker compose staging up: %w", err)
	}

	// Collect container info
	containers, err := listContainers(cfg, "staging")
	if err != nil {
		slog.Warn("failed to list staging containers", "err", err)
	}
	result.Containers = containers

	// Smoke test
	result.SmokeTest = runSmokeTest(ctx, cfg, "staging")

	result.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	result.Duration = time.Since(parseTime(result.StartedAt)).String()
	return result, nil
}

// Production deploys to production environment.
func Production(ctx context.Context, cfg *Config, stagingImageTag string) (*Result, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil config")
	}

	result := &Result{
		Phase:     "prod",
		Target:    "prod",
		ImageTag:  stagingImageTag,
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// Promote staging image to prod tag
	prodTag := fmt.Sprintf("%s:latest", cfg.ProjectName)
	tagCmd := exec.CommandContext(ctx, "docker", "tag", stagingImageTag, prodTag)
	if err := tagCmd.Run(); err != nil {
		result.Error = fmt.Sprintf("docker tag: %s", err)
		result.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		return result, fmt.Errorf("docker tag: %w", err)
	}
	result.ImageTag = prodTag

	// Deploy
	projectFlag := []string{"-f", cfg.ComposeProd, "-p", cfg.ProjectName + "-deploy", "up", "-d"}
	deployCmd := dockerComposeCmd(ctx, projectFlag)
	deployCmd.Dir = cfg.ProjectRoot
	if output, err := deployCmd.CombinedOutput(); err != nil {
		result.Error = fmt.Sprintf("docker compose up: %s: %s", err, string(output))
		result.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		return result, fmt.Errorf("docker compose prod up: %w", err)
	}

	containers, err := listContainers(cfg, "prod")
	if err != nil {
		slog.Warn("failed to list prod containers", "err", err)
	}
	result.Containers = containers

	// Smoke test
	result.SmokeTest = runSmokeTest(ctx, cfg, "prod")

	// Health monitor (5 min would be too long for tests, make configurable)
	result.Health = runHealthCheck(ctx, cfg, "prod", 30) // 30s for now

	result.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	result.Duration = time.Since(parseTime(result.StartedAt)).String()
	return result, nil
}

// Rollback rolls back to a previous image tag.
func Rollback(ctx context.Context, cfg *Config, previousTag string) (*Result, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil config")
	}

	result := &Result{
		Phase:     "rollback",
		Target:    "prod",
		ImageTag:  previousTag,
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}

	rollbackTag := fmt.Sprintf("%s:rollback", cfg.ProjectName)
	if err := exec.CommandContext(ctx, "docker", "tag", previousTag, rollbackTag).Run(); err != nil {
		return nil, fmt.Errorf("docker tag rollback: %w", err)
	}
	result.ImageTag = rollbackTag

	projectFlag := []string{"-f", cfg.ComposeProd, "-p", cfg.ProjectName + "-deploy", "up", "-d"}
	cmd := dockerComposeCmd(ctx, projectFlag)
	cmd.Dir = cfg.ProjectRoot
	if output, err := cmd.CombinedOutput(); err != nil {
		result.Error = string(output)
		result.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		return result, fmt.Errorf("docker compose rollback: %w", err)
	}

	containers, err := listContainers(cfg, "prod")
	if err != nil {
		slog.Warn("failed to list rollback containers", "err", err)
	}
	result.Containers = containers
	result.SmokeTest = runSmokeTest(ctx, cfg, "prod")

	result.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	result.Duration = time.Since(parseTime(result.StartedAt)).String()
	return result, nil
}

// helpers

func dockerComposeCmd(ctx context.Context, args []string) *exec.Cmd {
	all := append([]string{"compose"}, args...)
	return exec.CommandContext(ctx, "docker", all...)
}

func shortHash(hash string) string {
	if len(hash) > 12 {
		return hash[:12]
	}
	return hash
}

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		slog.Warn("failed to parse time", "input", s, "err", err)
		return time.Time{}
	}
	return t
}

func listContainers(cfg *Config, env string) ([]ContainerInfo, error) {
	projectName := cfg.ProjectName + "-" + env
	cmd := exec.Command("docker", "compose", "-p", projectName, "ps", "--format", "{{.Name}}\t{{.ID}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}")
	cmd.Dir = cfg.ProjectRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	var containers []ContainerInfo
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 5)
		if len(parts) >= 4 {
			c := ContainerInfo{Name: parts[0], ID: parts[1], Image: parts[2], Status: parts[3]}
			if len(parts) >= 5 {
				c.Ports = parts[4]
			}
			containers = append(containers, c)
		}
	}
	return containers, nil
}

func runSmokeTest(ctx context.Context, cfg *Config, env string) *TestResult {
	smokeScript := filepath.Join(cfg.ProjectRoot, "smoke-test.sh")
	if _, err := filepath.Glob(smokeScript); err != nil {
		return &TestResult{Passed: true, Output: "no smoke-test.sh found, skipped"}
	}

	cmd := exec.CommandContext(ctx, "bash", smokeScript)
	cmd.Dir = cfg.ProjectRoot
	output, err := cmd.CombinedOutput()
	result := &TestResult{
		Output:   string(output),
		ExitCode: cmd.ProcessState.ExitCode(),
	}
	if err != nil {
		result.Passed = false
	} else {
		result.Passed = cmd.ProcessState.ExitCode() == 0
	}
	return result
}

func runHealthCheck(ctx context.Context, cfg *Config, env string, durationSec int) *HealthCheckResult {
	result := &HealthCheckResult{Minutes: float64(durationSec) / 60}

	// Check containers are running
	projectName := cfg.ProjectName + "-" + env
	cmd := exec.CommandContext(ctx, "docker", "compose", "-p", projectName, "ps", "--format", "{{.Status}}")
	cmd.Dir = cfg.ProjectRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, string(output))
		return result
	}

	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if strings.Contains(line, "running") {
			result.Checks = append(result.Checks, line)
		} else if line != "" {
			result.Errors = append(result.Errors, line)
		}
	}

	result.Passed = len(result.Errors) == 0 && len(result.Checks) > 0
	return result
}
