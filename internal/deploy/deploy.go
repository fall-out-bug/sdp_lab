package deploy

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

	"sdp_dev/internal/secretscan"
)

// Config holds deploy configuration.
type Config struct {
	ProjectRoot    string // path to project directory
	ComposeStaging string // path to staging compose file (default: docker-compose.staging.yml)
	ComposeProd    string // path to prod compose file (default: docker-compose.yml)
	ProjectName    string // Docker project name (default: derived from dir)
}

// GatesConfig controls which hard gates are enforced before deploy.
type GatesConfig struct {
	SecretScan    bool          // Block if secretscan has findings (default: true)
	Evidence      bool          // Block if no evidence present (default: true)
	SmokeTest     bool          // Block if smoke tests fail (default: true)
	Staged        bool          // Enable staged (canary) rollout (default: false)
	CanaryReplicas int          // Number of canary replicas (default: 1)
	CanaryWait    time.Duration // How long to monitor canary before full rollout (default: 2m)
	CanaryService string        // Docker Compose service name for canary (default: "app")
}

// DefaultGatesConfig returns the default gate configuration (all hard gates on, no staged rollout).
func DefaultGatesConfig() *GatesConfig {
	return &GatesConfig{
		SecretScan:     true,
		Evidence:       true,
		SmokeTest:      true,
		Staged:         false,
		CanaryReplicas: 1,
		CanaryWait:     2 * time.Minute,
		CanaryService:  "app",
	}
}

// GateResult holds the result of a gate check.
type GateResult struct {
	Gate    string `json:"gate"`
	Passed  bool   `json:"passed"`
	Message string `json:"message,omitempty"`
}

// CheckGates runs all configured gates and returns results.
// Returns an error if any required gate fails.
func CheckGates(ctx context.Context, cfg *Config, gates *GatesConfig) ([]GateResult, error) {
	if gates == nil {
		gates = DefaultGatesConfig()
	}

	var results []GateResult
	var failed []string

	// Gate 1: Secret scan.
	if gates.SecretScan {
		gr := checkSecretScanGate(ctx, cfg)
		results = append(results, gr)
		if !gr.Passed {
			failed = append(failed, gr.Gate)
		}
	}

	// Gate 2: Evidence file.
	if gates.Evidence {
		gr := checkEvidenceGate(cfg)
		results = append(results, gr)
		if !gr.Passed {
			failed = append(failed, gr.Gate)
		}
	}

	// Gate 3: Smoke test.
	if gates.SmokeTest {
		gr := checkSmokeTestGate(ctx, cfg)
		results = append(results, gr)
		if !gr.Passed {
			failed = append(failed, gr.Gate)
		}
	}

	if len(failed) > 0 {
		return results, fmt.Errorf("deploy blocked by gates: %s", strings.Join(failed, ", "))
	}
	return results, nil
}

// checkSecretScanGate runs secretscan and returns pass/fail.
// Automatically excludes common test/fixture directories from scanning.
func checkSecretScanGate(ctx context.Context, cfg *Config) GateResult {
	// Exclude test fixture directories that contain intentional fake credentials.
	ignorePatterns := []string{
		"*_test.go",
		"*_test.py",
		"*_test.js",
		"testdata/**",
		"tests/**",
		"docs/**",
		".sdp/**",
		".worktrees/**",
	}
	scanner := secretscan.NewScanner(secretscan.WithIgnorePatterns(ignorePatterns))
	result, err := scanner.ScanDir(ctx, cfg.ProjectRoot)
	if err != nil {
		return GateResult{
			Gate:    "secret_scan",
			Passed:  false,
			Message: fmt.Sprintf("scan error: %v", err),
		}
	}
	// Fail closed: block on findings OR partial/incomplete scans.
	if result.Status != "clean" {
		if len(result.Findings) > 0 {
			critical, high, med := 0, 0, 0
			for _, f := range result.Findings {
				switch f.Severity {
				case "critical":
					critical++
				case "high":
					high++
				default:
					med++
				}
			}
			return GateResult{
				Gate:    "secret_scan",
				Passed:  false,
				Message: fmt.Sprintf("%d findings (%d critical, %d high, %d medium)", len(result.Findings), critical, high, med),
			}
		}
		// No findings but scan was incomplete (partial).
		return GateResult{
			Gate:    "secret_scan",
			Passed:  false,
			Message: fmt.Sprintf("scan incomplete: %d files scanned, %d skipped", result.FilesScanned, result.FilesSkipped),
		}
	}
	return GateResult{
		Gate:    "secret_scan",
		Passed:  true,
		Message: fmt.Sprintf("clean (%d files scanned)", result.FilesScanned),
	}
}

// checkEvidenceGate verifies evidence exists for the current run.
// Looks for evidence in two locations:
//   1. .sdp/evidence.json (legacy/top-level)
//   2. .sdp/evidence/<run_id>/evidence.json (per-run, written by sdp build)
//   3. .sdp/evidence/<feature>.json (per-feature)
func checkEvidenceGate(cfg *Config) GateResult {
	sdpDir := filepath.Join(cfg.ProjectRoot, ".sdp")

	// Check top-level evidence.json first.
	topLevel := filepath.Join(sdpDir, "evidence.json")
	if data, err := os.ReadFile(topLevel); err == nil {
		if validateEvidenceJSON(data) {
			return GateResult{Gate: "evidence", Passed: true, Message: "evidence.json present and valid"}
		}
		return GateResult{Gate: "evidence", Passed: false, Message: "evidence.json present but invalid JSON"}
	}

	// Check per-run evidence files under .sdp/evidence/.
	evidenceDir := filepath.Join(sdpDir, "evidence")
	matches, err := filepath.Glob(filepath.Join(evidenceDir, "*", "evidence.json"))
	if err == nil && len(matches) > 0 {
		latest := newestFile(matches)
		if data, err := os.ReadFile(latest); err == nil && validateEvidenceJSON(data) {
			return GateResult{Gate: "evidence", Passed: true, Message: fmt.Sprintf("evidence found: %s", latest)}
		}
	}

	// Check per-feature evidence files.
	featureMatches, err := filepath.Glob(filepath.Join(evidenceDir, "*.json"))
	if err == nil && len(featureMatches) > 0 {
		latest := newestFile(featureMatches)
		if data, err := os.ReadFile(latest); err == nil && validateEvidenceJSON(data) {
			return GateResult{Gate: "evidence", Passed: true, Message: fmt.Sprintf("evidence found: %s", latest)}
		}
	}

	return GateResult{
		Gate:    "evidence",
		Passed:  false,
		Message: "no evidence found — run sdp build first",
	}
}

// validateEvidenceJSON checks that data is valid JSON.
func validateEvidenceJSON(data []byte) bool {
	var ev map[string]any
	return json.Unmarshal(data, &ev) == nil
}

// newestFile returns the path with the most recent modification time.
func newestFile(paths []string) string {
	var best string
	var bestTime time.Time
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		if info.ModTime().After(bestTime) {
			bestTime = info.ModTime()
			best = p
		}
	}
	if best == "" && len(paths) > 0 {
		return paths[0]
	}
	return best
}

// checkSmokeTestGate runs the smoke test script.
func checkSmokeTestGate(ctx context.Context, cfg *Config) GateResult {
	smokeScript := filepath.Join(cfg.ProjectRoot, "smoke-test.sh")
	if _, err := os.Stat(smokeScript); os.IsNotExist(err) {
		return GateResult{
			Gate:   "smoke_test",
			Passed: true,
			Message: "no smoke-test.sh found, gate skipped",
		}
	}

	cmd := exec.CommandContext(ctx, "bash", smokeScript)
	cmd.Dir = cfg.ProjectRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		return GateResult{
			Gate:    "smoke_test",
			Passed:  false,
			Message: fmt.Sprintf("failed (exit %d): %s", cmd.ProcessState.ExitCode(), truncate(string(output), 200)),
		}
	}
	return GateResult{
		Gate:   "smoke_test",
		Passed: true,
		Message: "passed",
	}
}

// StagedRollout performs a canary deployment: deploy to a percentage of
// instances, wait for health confirmation, then promote to full.
func StagedRollout(ctx context.Context, cfg *Config, gates *GatesConfig, imageTag string) (*Result, error) {
	if gates == nil {
		gates = DefaultGatesConfig()
	}
	if gates.CanaryReplicas < 1 {
		return nil, fmt.Errorf("canary replicas must be >= 1, got %d", gates.CanaryReplicas)
	}
	if gates.CanaryService == "" {
		return nil, fmt.Errorf("canary service name is required for staged rollout")
	}
	if imageTag == "" {
		return nil, fmt.Errorf("image tag is required for staged rollout")
	}

	result := &Result{
		Phase:     "staged_rollout",
		Target:    "prod",
		ImageTag:  imageTag,
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// Phase 1: Deploy canary.
	canaryTag := fmt.Sprintf("%s:canary", cfg.ProjectName)
	if err := exec.CommandContext(ctx, "docker", "tag", imageTag, canaryTag).Run(); err != nil {
		result.Error = fmt.Sprintf("canary tag: %v", err)
		result.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		return result, fmt.Errorf("canary tag: %w", err)
	}

	// Deploy canary with limited scale using a compose override that
	// forces the canary service to use the candidate image.
	canaryProject := cfg.ProjectName + "-canary"
	overridePath, cleanup, err := writeCanaryOverride(cfg, gates.CanaryService, canaryTag)
	if err != nil {
		result.Error = fmt.Sprintf("canary override: %v", err)
		result.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		return result, fmt.Errorf("canary override: %w", err)
	}
	defer cleanup()

	deployArgs := []string{
		"-f", cfg.ComposeProd,
		"-f", overridePath,
		"-p", canaryProject,
		"up", "-d",
		"--scale", fmt.Sprintf("%s=%d", gates.CanaryService, gates.CanaryReplicas),
	}
	deployCmd := dockerComposeCmd(ctx, deployArgs)
	deployCmd.Dir = cfg.ProjectRoot
	if output, err := deployCmd.CombinedOutput(); err != nil {
		result.Error = fmt.Sprintf("canary deploy: %s: %s", err, string(output))
		result.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		// Attempt rollback on canary failure.
		rollbackCanary(ctx, cfg)
		return result, fmt.Errorf("canary deploy: %w", err)
	}

	// Phase 2: Monitor canary for the configured wait window.
	monitorResult := monitorCanary(ctx, cfg, gates.CanaryWait)
	if !monitorResult.Passed {
		slog.Warn("staged rollout: canary unhealthy, rolling back", "errors", monitorResult.Errors)
		rollbackCanary(ctx, cfg)
		result.Error = fmt.Sprintf("canary unhealthy: %s", strings.Join(monitorResult.Errors, "; "))
		result.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		writeRollbackEvidence(cfg, "canary unhealthy", canaryTag)
		return result, fmt.Errorf("canary unhealthy: %s", strings.Join(monitorResult.Errors, "; "))
	}

	// Phase 3: Promote to full production.
	prodTag := fmt.Sprintf("%s:latest", cfg.ProjectName)
	if err := exec.CommandContext(ctx, "docker", "tag", canaryTag, prodTag).Run(); err != nil {
		result.Error = fmt.Sprintf("promote tag: %v", err)
		result.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		return result, fmt.Errorf("promote: %w", err)
	}
	result.ImageTag = prodTag

	projectFlag := []string{"-f", cfg.ComposeProd, "-p", cfg.ProjectName + "-deploy", "up", "-d"}
	promoteCmd := dockerComposeCmd(ctx, projectFlag)
	promoteCmd.Dir = cfg.ProjectRoot
	if output, err := promoteCmd.CombinedOutput(); err != nil {
		result.Error = fmt.Sprintf("full deploy: %s: %s", err, string(output))
		result.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		return result, fmt.Errorf("full deploy: %w", err)
	}

	// Cleanup canary.
	rollbackCanary(ctx, cfg)

	// Run production health check on the promoted stack (not canary).
	result.Health = runHealthCheck(ctx, cfg, "prod", 30)
	result.SmokeTest = runSmokeTest(ctx, cfg, "prod")
	containers, _ := listContainers(cfg, "prod")
	result.Containers = containers
	result.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	result.Duration = time.Since(parseTime(result.StartedAt)).String()
	return result, nil
}

// writeCanaryOverride creates a temporary Docker Compose override file
// that forces the canary service to use the specified image.
// Returns the override file path, a cleanup function, and any error.
func writeCanaryOverride(cfg *Config, service, image string) (string, func(), error) {
	override := fmt.Sprintf(`services:
  %s:
    image: %s
`, service, image)

	f, err := os.CreateTemp("", "canary-override-*.yml")
	if err != nil {
		return "", func() {}, fmt.Errorf("create temp override: %w", err)
	}
	path := f.Name()
	if _, err := f.WriteString(override); err != nil {
		f.Close()
		os.Remove(path)
		return "", func() {}, fmt.Errorf("write override: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(path)
		return "", func() {}, fmt.Errorf("close override: %w", err)
	}
	return path, func() { os.Remove(path) }, nil
}

// monitorCanary polls container health for the given duration.
// Checks every 10s (or sooner if duration < 10s) and fails fast
// if containers go unhealthy.
func monitorCanary(ctx context.Context, cfg *Config, wait time.Duration) *HealthCheckResult {
	result := &HealthCheckResult{Minutes: wait.Minutes()}
	interval := 10 * time.Second
	if wait < interval {
		interval = wait
	}

	deadline := time.After(wait)
	checkNum := 0
	for {
		select {
		case <-ctx.Done():
			result.Passed = false
			result.Errors = append(result.Errors, "context cancelled")
			return result
		case <-deadline:
			// Final health check before promoting.
			final := runHealthCheck(ctx, cfg, "canary", int(wait.Seconds()))
			result.Checks = final.Checks
			result.Errors = final.Errors
			result.Passed = final.Passed
			return result
		case <-time.After(interval):
			checkNum++
			hc := runHealthCheck(ctx, cfg, "canary", int(wait.Seconds()))
			if !hc.Passed && len(hc.Errors) > 0 {
				// Fail fast: containers are already unhealthy.
				result.Checks = hc.Checks
				result.Errors = hc.Errors
				result.Passed = false
				return result
			}
			result.Checks = append(result.Checks, hc.Checks...)
		}
	}
}

// rollbackCanary tears down the canary deployment.
func rollbackCanary(ctx context.Context, cfg *Config) {
	canaryProject := cfg.ProjectName + "-canary"
	cmd := dockerComposeCmd(ctx, []string{"-f", cfg.ComposeProd, "-p", canaryProject, "down"})
	cmd.Dir = cfg.ProjectRoot
	if output, err := cmd.CombinedOutput(); err != nil {
		slog.Warn("canary cleanup failed", "err", err, "output", string(output))
	}
}

// writeRollbackEvidence saves rollback evidence to .sdp/.
func writeRollbackEvidence(cfg *Config, reason, tag string) {
	ev := map[string]any{
		"action":    "rollback",
		"reason":    reason,
		"tag":       tag,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	data, _ := json.MarshalIndent(ev, "", "  ")
	path := filepath.Join(cfg.ProjectRoot, ".sdp", "deploy.rollback.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		slog.Error("mkdir for rollback evidence", "err", err)
		return
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		slog.Error("write rollback evidence", "err", err)
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
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
	cmd := exec.CommandContext(context.Background(), "docker", "compose", "-p", projectName, "ps", "--format", "{{.Name}}\t{{.ID}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}")
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
