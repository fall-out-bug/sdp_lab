package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/pireview"
)

func TestWriteVerdictFile_UsesPrivatePermissions(t *testing.T) {
	dir := t.TempDir()
	verdict := &pireview.Verdict{
		Feature: "F168",
		Round:   1,
	}

	if err := writeVerdictFile(dir, verdict); err != nil {
		t.Fatalf("writeVerdictFile() error: %v", err)
	}

	filePath := filepath.Join(dir, ".sdp", "review_verdict.json")
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("read verdict file: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("review_verdict.json mode = %o, want 600", got)
	}

	dirInfo, err := os.Stat(filepath.Join(dir, ".sdp"))
	if err != nil {
		t.Fatalf("read .sdp dir: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o700 {
		t.Errorf(".sdp dir mode = %o, want 700", got)
	}
}

func TestWriteVerdictFile_NormalizesInsecurePaths(t *testing.T) {
	dir := t.TempDir()
	sdpDir := filepath.Join(dir, ".sdp")
	if err := os.MkdirAll(sdpDir, 0o755); err != nil {
		t.Fatalf("mkdir .sdp: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sdpDir, "review_verdict.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("write old verdict: %v", err)
	}

	verdict := &pireview.Verdict{
		Feature: "F168",
		Round:   2,
	}

	if err := writeVerdictFile(dir, verdict); err != nil {
		t.Fatalf("writeVerdictFile() error: %v", err)
	}

	info, err := os.Stat(filepath.Join(sdpDir, "review_verdict.json"))
	if err != nil {
		t.Fatalf("read verdict: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("review_verdict.json mode = %o, want 600", got)
	}

	dirInfo, err := os.Stat(sdpDir)
	if err != nil {
		t.Fatalf("read .sdp dir: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o700 {
		t.Errorf(".sdp dir mode = %o, want 700", got)
	}
}
