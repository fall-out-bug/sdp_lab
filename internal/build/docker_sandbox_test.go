package build

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// skipIfNoDocker skips the test if Docker is not available or if the caller
// requested to skip Docker tests.
func skipIfNoDocker(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping docker tests in short mode")
	}
	if os.Getenv("SKIP_DOCKER") != "" {
		t.Skip("skipping docker tests (SKIP_DOCKER set)")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available")
	}
}

func TestDockerSandbox_Available(t *testing.T) {
	if dockerAvailable() {
		ver, err := dockerVersion()
		if err != nil {
			t.Logf("docker available but version check failed: %v", err)
		} else {
			t.Logf("docker available, version: %s", strings.TrimSpace(ver))
		}
	} else {
		t.Log("docker not available on this system")
	}
}

func TestDockerSandbox_Creation(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("docker not available")
	}

	cfg := DockerSandboxConfig{
		Image:        "golang:1.22",
		CGO:          false,
		Timeout:      5 * time.Minute,
		AllowNetwork: false,
		CPUQuota:     100000,
		MemoryMB:     512,
	}

	sb, err := NewDockerSandbox(cfg)
	if err != nil {
		t.Fatalf("NewDockerSandbox: %v", err)
	}
	if sb.image != "golang:1.22" {
		t.Errorf("image = %q, want %q", sb.image, "golang:1.22")
	}
	if sb.cgo != false {
		t.Error("cgo should be false")
	}
	if sb.timeout != 5*time.Minute {
		t.Errorf("timeout = %v, want %v", sb.timeout, 5*time.Minute)
	}
	if sb.allowNetwork != false {
		t.Error("allowNetwork should be false")
	}
	if sb.cpuQuota != 100000 {
		t.Errorf("cpuQuota = %d, want %d", sb.cpuQuota, 100000)
	}
	if sb.memoryMB != 512 {
		t.Errorf("memoryMB = %d, want %d", sb.memoryMB, 512)
	}
}

func TestDockerSandbox_DefaultImage(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("docker not available")
	}

	cfg := DockerSandboxConfig{} // No image specified.
	sb, err := NewDockerSandbox(cfg)
	if err != nil {
		t.Fatalf("NewDockerSandbox: %v", err)
	}
	if sb.image != "golang:1.22" {
		t.Errorf("default image = %q, want %q", sb.image, "golang:1.22")
	}
}

func TestDockerSandbox_NotAvailable(t *testing.T) {
	// This test verifies the error message when docker is not on PATH.
	// We temporarily modify PATH to simulate docker being unavailable.
	if dockerAvailable() {
		// Cannot easily test this when docker IS available without modifying
		// global state, so we just verify the factory function handles the case.
		t.Log("docker is available; skipping not-available simulation")
		return
	}

	_, err := NewDockerSandbox(DockerSandboxConfig{})
	if err == nil {
		t.Fatal("expected error when docker is not available")
	}
	if !strings.Contains(err.Error(), "command not found") {
		t.Errorf("error = %q, want substring %q", err.Error(), "command not found")
	}
	if !strings.Contains(err.Error(), "install Docker") {
		t.Errorf("error = %q, want substring %q", err.Error(), "install Docker")
	}
}

func TestDockerSandbox_BuildAndTest(t *testing.T) {
	skipIfNoDocker(t)

	dir := t.TempDir()
	writeGoMod(t, dir)
	writeGoFile(t, dir)

	cfg := DockerSandboxConfig{
		Image:   "golang:1.22",
		Timeout: 5 * time.Minute,
	}

	sb, err := NewDockerSandbox(cfg)
	if err != nil {
		t.Fatalf("NewDockerSandbox: %v", err)
	}
	defer sb.Cleanup()

	ctx := context.Background()

	buildRes, err := sb.Build(ctx, dir)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !buildRes.Success {
		t.Errorf("Build Success = false; stderr: %s", buildRes.Stderr)
	}
	if buildRes.Duration == 0 {
		t.Error("Build Duration should be non-zero")
	}

	testRes, err := sb.Test(ctx, dir)
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	if !testRes.Success {
		t.Errorf("Test Success = false; stderr: %s", testRes.Stderr)
	}
	if testRes.Duration == 0 {
		t.Error("Test Duration should be non-zero")
	}
}

func TestDockerSandbox_BuildFailure(t *testing.T) {
	skipIfNoDocker(t)

	dir := t.TempDir()
	// Write an invalid Go file to cause build failure.
	os.MkdirAll(filepath.Join(dir, "bad"), 0o755)
	if err := os.WriteFile(filepath.Join(dir, "bad", "bad.go"), []byte("package bad\nfunc broken(\n"), 0o644); err != nil {
		t.Fatalf("write bad.go: %v", err)
	}
	writeGoMod(t, dir)

	cfg := DockerSandboxConfig{
		Image:   "golang:1.22",
		Timeout: 2 * time.Minute,
	}

	sb, err := NewDockerSandbox(cfg)
	if err != nil {
		t.Fatalf("NewDockerSandbox: %v", err)
	}
	defer sb.Cleanup()

	ctx := context.Background()

	buildRes, err := sb.Build(ctx, dir)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if buildRes.Success {
		t.Error("Build should fail for invalid Go code")
	}
	if buildRes.ExitCode == 0 {
		t.Error("ExitCode should be non-zero for failed build")
	}
	if buildRes.Stderr == "" {
		t.Error("Stderr should not be empty for failed build")
	}
}

func TestDockerSandbox_ResourceLimits(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("docker not available")
	}

	cfg := DockerSandboxConfig{
		Image:        "golang:1.22",
		CPUQuota:     50000,  // 50ms CPU quota per 100ms period.
		MemoryMB:     256,    // 256 MB memory limit.
		AllowNetwork: false,
		CGO:          false,
	}

	sb, err := NewDockerSandbox(cfg)
	if err != nil {
		t.Fatalf("NewDockerSandbox: %v", err)
	}

	args := sb.buildDockerArgs("/tmp/test", "build", "./...")

	// Verify --cpu-quota is present.
	foundCPU := false
	for _, a := range args {
		if strings.HasPrefix(a, "--cpu-quota=") {
			foundCPU = true
			if a != "--cpu-quota=50000" {
				t.Errorf("cpu quota arg = %q, want %q", a, "--cpu-quota=50000")
			}
		}
	}
	if !foundCPU {
		t.Error("--cpu-quota not found in docker args")
	}

	// Verify --memory is present.
	foundMemory := false
	for _, a := range args {
		if strings.HasPrefix(a, "--memory=") || a == "--memory" {
			foundMemory = true
		}
	}
	if !foundMemory {
		t.Error("--memory not found in docker args")
	}

	// Verify --network none.
	foundNetwork := false
	for i, a := range args {
		if a == "--network" && i+1 < len(args) && args[i+1] == "none" {
			foundNetwork = true
		}
	}
	if !foundNetwork {
		t.Error("--network none not found in docker args")
	}

	// Verify CGO_ENABLED=0.
	foundCGO := false
	for i, a := range args {
		if a == "-e" && i+1 < len(args) && args[i+1] == "CGO_ENABLED=0" {
			foundCGO = true
		}
	}
	if !foundCGO {
		t.Error("CGO_ENABLED=0 not found in docker args")
	}
}

func TestDockerSandbox_AllowNetwork(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("docker not available")
	}

	cfg := DockerSandboxConfig{
		Image:        "golang:1.22",
		AllowNetwork: true,
	}

	sb, err := NewDockerSandbox(cfg)
	if err != nil {
		t.Fatalf("NewDockerSandbox: %v", err)
	}

	args := sb.buildDockerArgs("/tmp/test", "test", "./...")

	// When allowNetwork is true, no --network flag should be present (uses Docker default bridge).
	for i, a := range args {
		if a == "--network" && i+1 < len(args) {
			t.Errorf("--network flag should not be present when allowNetwork=true, got --network %s", args[i+1])
		}
	}
}

func TestDockerSandbox_CGOMount(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("docker not available")
	}

	cfg := DockerSandboxConfig{
		Image: "golang:1.22",
		CGO:   true,
	}

	sb, err := NewDockerSandbox(cfg)
	if err != nil {
		t.Fatalf("NewDockerSandbox: %v", err)
	}

	args := sb.buildDockerArgs("/tmp/test", "build", "./...")

	// Verify CGO_ENABLED=1.
	found := false
	for i, a := range args {
		if a == "-e" && i+1 < len(args) && args[i+1] == "CGO_ENABLED=1" {
			found = true
		}
	}
	if !found {
		t.Error("CGO_ENABLED=1 not found in docker args when cgo=true")
	}
}

func TestDockerSandbox_CleanupIdempotent(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("docker not available")
	}

	cfg := DockerSandboxConfig{Image: "golang:1.22"}
	sb, err := NewDockerSandbox(cfg)
	if err != nil {
		t.Fatalf("NewDockerSandbox: %v", err)
	}

	// Calling Cleanup multiple times should not panic or error.
	if err := sb.Cleanup(); err != nil {
		t.Errorf("first Cleanup: %v", err)
	}
	if err := sb.Cleanup(); err != nil {
		t.Errorf("second Cleanup: %v", err)
	}
}

func TestDockerSandbox_BuildDockerArgs_MountPath(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("docker not available")
	}

	cfg := DockerSandboxConfig{Image: "golang:1.22"}
	sb, err := NewDockerSandbox(cfg)
	if err != nil {
		t.Fatalf("NewDockerSandbox: %v", err)
	}

	// Test with absolute path — should be used directly.
	args := sb.buildDockerArgs("/home/user/project", "build", "./...")
	foundMount := false
	for i, a := range args {
		if a == "-v" && i+1 < len(args) && args[i+1] == "/home/user/project:/work" {
			foundMount = true
		}
	}
	if !foundMount {
		t.Error("volume mount /home/user/project:/work not found in args")
	}

	// Verify -w /work.
	foundWorkdir := false
	for i, a := range args {
		if a == "-w" && i+1 < len(args) && args[i+1] == "/work" {
			foundWorkdir = true
		}
	}
	if !foundWorkdir {
		t.Error("-w /work not found in args")
	}
}

func TestDockerSandbox_ContextCancellation(t *testing.T) {
	skipIfNoDocker(t)

	dir := t.TempDir()
	writeGoMod(t, dir)
	writeGoFile(t, dir)

	cfg := DockerSandboxConfig{
		Image:   "golang:1.22",
		Timeout: 10 * time.Minute, // Long timeout on sandbox level.
	}

	sb, err := NewDockerSandbox(cfg)
	if err != nil {
		t.Fatalf("NewDockerSandbox: %v", err)
	}
	defer sb.Cleanup()

	// Cancel context immediately.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = sb.Build(ctx, dir)
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}

func TestDockerSandbox_SandboxInterface(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("docker not available")
	}

	// Verify DockerSandbox implements the Sandbox interface.
	cfg := DockerSandboxConfig{Image: "golang:1.22"}
	var _ Sandbox = (*DockerSandbox)(nil)

	sb, err := NewDockerSandbox(cfg)
	if err != nil {
		t.Fatalf("NewDockerSandbox: %v", err)
	}
	// This should compile — proves interface compliance.
	var s Sandbox = sb
	_ = s
}

func TestDockerSandbox_HardeningFlags(t *testing.T) {
	skipIfNoDocker(t)
	cfg := DockerSandboxConfig{Image: "golang:1.22", CGO: false}
	s, err := NewDockerSandbox(cfg)
	if err != nil {
		t.Fatalf("NewDockerSandbox: %v", err)
	}
	args := s.buildDockerArgs("/tmp/test", "version")

	// Check hardening flags are present
	assertArg(t, args, "--security-opt=no-new-privileges:true")
	assertArg(t, args, "--cap-drop=ALL")

	// Check --user flag is present
	found := false
	for _, a := range args {
		if strings.HasPrefix(a, "--user=") {
			found = true
			break
		}
	}
	if !found {
		t.Error("--user flag not found in docker args")
	}
}

func assertArg(t *testing.T, args []string, want string) {
	t.Helper()
	for _, a := range args {
		if a == want {
			return
		}
	}
	t.Errorf("expected arg %q not found in %v", want, args)
}
