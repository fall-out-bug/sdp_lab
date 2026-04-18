//go:build smoke

// Package smoke contains end-to-end CLI smoke scenarios for SDP binaries.
// Run: go test -tags=smoke ./test/smoke/... -v
// Or via: scripts/run_smoke_tests.sh
package smoke

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

const (
	buildTimeout = 120 * time.Second
	runTimeout   = 60 * time.Second
)

func projectRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..")
}

func buildBinary(t *testing.T, root, pkg string) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), filepath.Base(pkg))
	ctx, cancel := context.WithTimeout(context.Background(), buildTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "build", "-o", out, "./cmd/"+pkg+"/")
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build %s failed: %v\n%s", pkg, err, output)
	}
	return out
}

func run(t *testing.T, bin string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	var outBuf, errBuf bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	cmd.Dir = projectRoot(t)
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Logf("run error (not ExitError): %v", err)
			exitCode = -1
		}
	}
	return outBuf.String(), errBuf.String(), exitCode
}

// Scenario 1: sdp-healthcheck --json exits 0 and emits valid JSON array.
func TestHealthcheck_JSONOutput(t *testing.T) {
	root := projectRoot(t)
	bin := buildBinary(t, root, "sdp-healthcheck")

	stdout, stderr, code := run(t, bin, "--json", "--check=git-clean")
	if code != 0 {
		t.Fatalf("want exit 0, got %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	var results []struct {
		Name   string `json:"name"`
		Status string `json:"status"`
		Detail string `json:"detail"`
	}
	if err := json.Unmarshal([]byte(stdout), &results); err != nil {
		t.Fatalf("invalid JSON output: %v\nraw: %s", err, stdout)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result in JSON output")
	}
	if results[0].Name == "" || results[0].Status == "" {
		t.Errorf("result missing name/status: %+v", results[0])
	}
}

// Scenario 2: sdp-healthcheck single-check mode returns exactly one result.
func TestHealthcheck_SingleCheck(t *testing.T) {
	root := projectRoot(t)
	bin := buildBinary(t, root, "sdp-healthcheck")

	stdout, _, code := run(t, bin, "--json", "--check=git-clean")
	if code != 0 && code != 1 {
		t.Fatalf("unexpected exit code %d (0 or 1 expected)", code)
	}
	var results []map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &results); err != nil {
		t.Fatalf("parse JSON: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected exactly 1 result for single check, got %d", len(results))
	}
	if _, ok := results[0]["name"]; !ok {
		t.Error("result missing 'name' field")
	}
}

// Scenario 3: sdp-healthcheck unknown check name exits non-zero.
func TestHealthcheck_UnknownCheck(t *testing.T) {
	root := projectRoot(t)
	bin := buildBinary(t, root, "sdp-healthcheck")

	_, _, code := run(t, bin, "--check=no-such-check")
	if code == 0 {
		t.Fatal("expected non-zero exit for unknown check, got 0")
	}
}

// Scenario 4: sdp-protocol-check --lint-skills emits valid JSON and known exit codes.
// Exit 0 = no issues, 1 = warnings only, 2 = errors present.
func TestProtocolCheck_LintSkillsJSON(t *testing.T) {
	root := projectRoot(t)
	bin := buildBinary(t, root, "sdp-protocol-check")

	stdout, stderr, code := run(t, bin, "--lint-skills", "--format=json")
	if code < 0 || code > 2 {
		t.Fatalf("unexpected exit code %d (0/1/2 allowed)\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if !json.Valid([]byte(stdout)) {
		t.Errorf("--lint-skills --format=json output is not valid JSON:\n%s", stdout)
	}
}

// Scenario 5: sdp-protocol-check --format=json emits parseable JSON with an 'issues' array.
func TestProtocolCheck_JSONHasIssuesField(t *testing.T) {
	root := projectRoot(t)
	bin := buildBinary(t, root, "sdp-protocol-check")

	stdout, _, code := run(t, bin, "--format=json")
	if code < 0 || code > 2 {
		t.Fatalf("unexpected exit code %d\nraw output: %s", code, stdout)
	}
	var result struct {
		Issues []map[string]interface{} `json:"issues"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nraw: %s", err, stdout)
	}
	// issues field may be empty but must be present (not nil)
	if result.Issues == nil {
		t.Error("'issues' field must be present in output (may be empty array)")
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
