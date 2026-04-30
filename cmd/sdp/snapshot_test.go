package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/snapshot"
)

var (
	snapBinOnce sync.Once
	snapBinPath string
	snapBinErr  error
)

// getSnapBinary builds the sdp binary once and caches the path.
func getSnapBinary(t *testing.T) string {
	t.Helper()
	snapBinOnce.Do(func() {
		tmpDir, err := os.MkdirTemp("", "sdp-snap-*")
		if err != nil {
			snapBinErr = err
			return
		}
		binPath := filepath.Join(tmpDir, "sdp-snap")
		cmd := exec.Command("go", "build", "-tags", "sqlite_fts5", "-o", binPath, ".")
		if out, err := cmd.CombinedOutput(); err != nil {
			snapBinErr = fmt.Errorf("build failed: %v\n%s", err, out)
			return
		}
		snapBinPath = binPath
	})
	if snapBinErr != nil {
		t.Fatal(snapBinErr)
	}
	return snapBinPath
}

// CLIResult holds the combined output and exit error from a CLI invocation.
type CLIResult struct {
	Output string // combined stdout+stderr
	Err    error  // non-nil if the command exited with a non-zero status
}

// runSnapCLI executes the sdp binary and returns combined output and error.
func runSnapCLI(t *testing.T, binPath string, args ...string) CLIResult {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return CLIResult{Output: out.String(), Err: err}
}

// exitCode extracts the exit code from a CLIResult error, or returns 0 on success.
func exitCode(res CLIResult) int {
	if res.Err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(res.Err, &exitErr) {
		return exitErr.ExitCode()
	}
	// Non-exit error (e.g. exec not found) — treat as code 1.
	return 1
}

func newSnapSnapshotter(t *testing.T) *snapshot.Snapshotter {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return snapshot.New(filepath.Join(wd, ".snapshots"))
}

func TestSnapshot_CLIUsage(t *testing.T) {
	bin := getSnapBinary(t)
	s := newSnapSnapshotter(t)

	tests := []struct {
		name      string
		args      []string
		wantExit0 bool // true = expect exit 0; false = expect non-zero exit
	}{
		{"main-usage", nil, false},
		{"card-usage", []string{"card"}, false},
		{"board-usage", []string{"board"}, false},
		{"dispatch-usage", []string{"dispatch"}, false},
		{"doctor-usage", []string{"doctor"}, true},
		{"result-usage", []string{"result"}, false},
		{"orchestrate-usage", []string{"orchestrate"}, false},
		{"unknown-command", []string{"foobar"}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := runSnapCLI(t, bin, tc.args...)

			// Validate exit status.
			code := exitCode(res)
			if tc.wantExit0 && code != 0 {
				t.Fatalf("expected exit 0, got %d; output:\n%s", code, res.Output)
			}
			if !tc.wantExit0 && code == 0 {
				t.Fatalf("expected non-zero exit for %q, got 0; output:\n%s", tc.name, res.Output)
			}

			if err := s.Compare(tc.name, res.Output); err != nil {
				t.Fatal(err)
			}
		})
	}
}
