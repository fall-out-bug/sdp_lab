package build

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// DockerSandbox runs build and test commands inside disposable Docker containers.
// Each Build/Test call creates a fresh container with `docker run --rm`, so no
// persistent container state is managed.
type DockerSandbox struct {
	image        string        // Docker image (e.g., "golang:1.22").
	workdir      string        // Host directory mounted into container.
	cgo          bool          // Whether CGO is enabled.
	timeout      time.Duration // Per-command timeout (0 = no explicit timeout).
	allowNetwork bool          // Whether network access is allowed.
	cpuQuota     int64         // CPU quota in microseconds (0 = unlimited).
	memoryMB     int64         // Memory limit in MB (0 = unlimited).
}

// DockerSandboxConfig holds configuration for creating a DockerSandbox.
type DockerSandboxConfig struct {
	Image        string        // Docker image (required, defaults to "golang:1.22").
	CGO          bool          // Whether CGO is enabled.
	Timeout      time.Duration // Per-command timeout (0 = no explicit timeout).
	AllowNetwork bool          // Whether network access is allowed.
	CPUQuota     int64         // CPU quota in microseconds (0 = unlimited).
	MemoryMB     int64         // Memory limit in MB (0 = unlimited).
}

// NewDockerSandbox creates a DockerSandbox with the given configuration.
// Returns an error if the docker binary is not found on PATH.
func NewDockerSandbox(cfg DockerSandboxConfig) (*DockerSandbox, error) {
	if _, err := exec.LookPath("docker"); err != nil {
		return nil, fmt.Errorf("docker: command not found (install Docker or use --sandbox=none)")
	}

	image := cfg.Image
	if image == "" {
		image = "golang:1.22"
	}

	// Apply conservative resource defaults for safety.
	if cfg.CPUQuota == 0 {
		cfg.CPUQuota = 100000 // 1 CPU
	}
	if cfg.MemoryMB == 0 {
		cfg.MemoryMB = 2048 // 2 GB
	}

	return &DockerSandbox{
		image:        image,
		cgo:          cfg.CGO,
		timeout:      cfg.Timeout,
		allowNetwork: cfg.AllowNetwork,
		cpuQuota:     cfg.CPUQuota,
		memoryMB:     cfg.MemoryMB,
	}, nil
}

// Build runs `go build ./...` inside a fresh Docker container.
func (s *DockerSandbox) Build(ctx context.Context, dir string) (*SandboxResult, error) {
	return s.runInContainer(ctx, dir, "build", "./...")
}

// Test runs `go test ./...` inside a fresh Docker container.
func (s *DockerSandbox) Test(ctx context.Context, dir string) (*SandboxResult, error) {
	return s.runInContainer(ctx, dir, "test", "./...")
}

// Cleanup is a no-op. Docker run --rm auto-removes containers on exit.
// For abnormal termination (SIGKILL), the --rm flag still takes effect
// when the docker daemon cleans up.
func (s *DockerSandbox) Cleanup() error {
	return nil
}

// runInContainer executes a go command inside a fresh Docker container.
func (s *DockerSandbox) runInContainer(ctx context.Context, dir string, goCmd string, args ...string) (*SandboxResult, error) {
	start := time.Now()

	// Apply per-command timeout if configured.
	if s.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.timeout)
		defer cancel()
	}

	dockerArgs := s.buildDockerArgs(dir, goCmd, args...)
	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)

	// Capture stdout and stderr via pipes for streaming output.
	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("docker stdout pipe: %w", err)
	}
	errPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("docker stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("docker start: %w", err)
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
	duration := time.Since(start)

	if waitErr != nil {
		// Distinguish context cancellation/timeout from command failure.
		if ctx.Err() != nil {
			return nil, fmt.Errorf("docker %s: %w", goCmd, ctx.Err())
		}
		exitCode := 1
		if cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		}
		return &SandboxResult{
			Success:  false,
			ExitCode: exitCode,
			Stdout:   string(stdout),
			Stderr:   string(stderr),
			Duration: duration,
		}, nil
	}

	return &SandboxResult{
		Success:  true,
		ExitCode: 0,
		Stdout:   string(stdout),
		Stderr:   string(stderr),
		Duration: duration,
	}, nil
}

// buildDockerArgs constructs the docker run argument list.
func (s *DockerSandbox) buildDockerArgs(dir string, goCmd string, extraArgs ...string) []string {
	args := []string{"run", "--rm"}

	// Security hardening
	args = append(args, "--security-opt=no-new-privileges:true")
	args = append(args, "--cap-drop=ALL")

	// Run as current user to prevent root-owned files on host
	args = append(args, fmt.Sprintf("--user=%d:%d", os.Getuid(), os.Getgid()))

	// Mount the host directory and set the working directory.
	absDir := dir
	// Ensure absolute path for the mount.
	if abs, err := absPath(dir); err == nil {
		absDir = abs
	}
	// Volume is read-write by design: go build/test write cache, coverage, and test fixtures.
	// Container is hardened (--cap-drop=ALL, no-new-privileges, --user) to limit damage surface.
	args = append(args, "-v", absDir+":/work", "-w", "/work")

	// Use container-local paths for Go cache to avoid polluting host.
	args = append(args, "-e", "GOCACHE=/tmp/go-build-cache")
	args = append(args, "-e", "GOMODCACHE=/tmp/go-mod-cache")
	args = append(args, "-e", "GOTMPDIR=/tmp/go-tmp")

	// Network configuration.
	if !s.allowNetwork {
		args = append(args, "--network", "none")
	}
	// When allowNetwork is true, omit --network flag (Docker default bridge).
	// Do NOT use --network host as it exposes host services.

	// Resource limits.
	if s.cpuQuota > 0 {
		// Convert microseconds to the docker --cpus format (quota/100000).
		// Use --cpu-quota directly for precision.
		args = append(args, fmt.Sprintf("--cpu-quota=%d", s.cpuQuota))
	}
	if s.memoryMB > 0 {
		args = append(args, "--memory", fmt.Sprintf("%dm", s.memoryMB))
	}

	// CGO environment variable.
	if !s.cgo {
		args = append(args, "-e", "CGO_ENABLED=0")
	} else {
		args = append(args, "-e", "CGO_ENABLED=1")
	}

	// Docker image.
	args = append(args, s.image)

	// Go command and arguments.
	args = append(args, "go", goCmd)
	args = append(args, extraArgs...)

	return args
}

// absPath returns the absolute path for the given directory.
func absPath(dir string) (string, error) {
	if filepath.IsAbs(dir) {
		return dir, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	return filepath.Join(wd, dir), nil
}

// dockerAvailable checks whether the Docker daemon is accessible.
func dockerAvailable() bool {
	cmd := exec.Command("docker", "info", "--format", "{{.ServerVersion}}")
	return cmd.Run() == nil
}

// dockerVersion returns the docker version string for diagnostic purposes.
func dockerVersion() (string, error) {
	cmd := exec.Command("docker", "version", "--format", "{{.Client.Version}}")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("docker version: %w", err)
	}
	return buf.String(), nil
}
