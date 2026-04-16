package main

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func buildTestBinary(t *testing.T) string {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "sdp-test")
	cmd := exec.Command("go", "build", "-tags", "sqlite_fts5", "-o", binPath, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build test binary: %v\n%s", err, out)
	}
	return binPath
}

func TestMainUsage(t *testing.T) {
	binPath := buildTestBinary(t)
	cmd := exec.Command(binPath)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected error exit, got nil")
	}
	output := out.String()
	if !strings.Contains(output, "usage: sdp") {
		t.Fatalf("expected usage message in output, got: %s", output)
	}
	if !strings.Contains(output, "card|board|doctor|dispatch|result|orchestrate|attention") {
		t.Fatalf("expected subcommand list in output, got: %s", output)
	}
}

func TestCardUsage(t *testing.T) {
	binPath := buildTestBinary(t)
	cmd := exec.Command(binPath, "card")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected error exit, got nil")
	}
	output := out.String()
	if !strings.Contains(output, "usage: sdp card") {
		t.Fatalf("expected card usage in output, got: %s", output)
	}
	if !strings.Contains(output, "heartbeat") {
		t.Fatalf("expected heartbeat subcommand in card usage, got: %s", output)
	}
}

func TestBoardUsage(t *testing.T) {
	binPath := buildTestBinary(t)
	cmd := exec.Command(binPath, "board")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected error exit, got nil")
	}
	output := out.String()
	if !strings.Contains(output, "usage: sdp board") {
		t.Fatalf("expected board usage in output, got: %s", output)
	}
}

func TestDoctorControlUsage(t *testing.T) {
	binPath := buildTestBinary(t)
	cmd := exec.Command(binPath, "doctor")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected error exit, got nil")
	}
	output := out.String()
	if !strings.Contains(output, "usage: sdp doctor control") {
		t.Fatalf("expected doctor usage in output, got: %s", output)
	}
}

func TestDispatchUsage(t *testing.T) {
	binPath := buildTestBinary(t)
	cmd := exec.Command(binPath, "dispatch")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected error exit, got nil")
	}
	output := out.String()
	if !strings.Contains(output, "usage: sdp dispatch card") {
		t.Fatalf("expected dispatch usage in output, got: %s", output)
	}
}

func TestResultUsage(t *testing.T) {
	binPath := buildTestBinary(t)
	cmd := exec.Command(binPath, "result")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected error exit, got nil")
	}
	output := out.String()
	if !strings.Contains(output, "usage: sdp result ingest") {
		t.Fatalf("expected result usage in output, got: %s", output)
	}
}

func TestUnknownCommand(t *testing.T) {
	binPath := buildTestBinary(t)
	cmd := exec.Command(binPath, "unknown")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected error exit, got nil")
	}
}
