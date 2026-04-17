package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"sdp_dev/internal/snapshot"
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

// runSnapCLI executes the sdp binary and returns combined stdout+stderr.
func runSnapCLI(t *testing.T, binPath string, args ...string) string {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	_ = cmd.Run()
	return out.String()
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
		name string
		args []string
	}{
		{"main-usage", nil},
		{"card-usage", []string{"card"}},
		{"board-usage", []string{"board"}},
		{"dispatch-usage", []string{"dispatch"}},
		{"doctor-usage", []string{"doctor"}},
		{"result-usage", []string{"result"}},
		{"orchestrate-usage", []string{"orchestrate"}},
		{"unknown-command", []string{"foobar"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			output := runSnapCLI(t, bin, tc.args...)
			if err := s.Compare(tc.name, output); err != nil {
				t.Fatal(err)
			}
		})
	}
}
